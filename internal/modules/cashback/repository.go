package cashback

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/lib/pq"
)

// ErrDuplicateReceipt is returned when a receipt with the same canonical hash was
// already registered for the same customer_sisfi (business + program). Detected
// via the partial unique index idx_transactions_cashback_receipt_hash.
var ErrDuplicateReceipt = errors.New("ticket ya registrado")

// isUniqueViolation reports whether err is a Postgres unique_violation (23505).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// nullableJSON returns nil (SQL NULL) for empty payloads so JSONB columns stay NULL.
func nullableJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

type Repository interface {
	GetProgram(ctx context.Context, customerID string) (*CashbackProgram, error)
	GetProgramByID(ctx context.Context, customerSisfiID string) (*CashbackProgram, error)
	CreateProgram(ctx context.Context, p *CashbackProgram) error
	UpdateProgram(ctx context.Context, p *CashbackProgram, setActive *bool) error
	ExpireBalance(ctx context.Context, clientID, customerSisfiID string, expiryDays int) (float64, error)
	SumCashbackInWindow(ctx context.Context, clientID, customerSisfiID string, windowDays int) (float64, error)
	GetBalance(ctx context.Context, clientID, customerSisfiID string) (float64, error)
	UpsertBalance(ctx context.Context, clientID, customerSisfiID string, delta float64) (float64, error)
	CreateTransaction(ctx context.Context, tx *CashbackTransaction) error
	GetTransaction(ctx context.Context, id string) (*CashbackTransaction, error)
	ListTransactions(ctx context.Context, clientID, customerSisfiID string, limit int) ([]CashbackTransaction, error)
	ListCorrectableTransactions(ctx context.Context, clientID string) ([]CashbackTransaction, error)
	GetClientName(ctx context.Context, clientID string) (string, error)

	ListRewards(ctx context.Context, customerID, customerSisfiID string, maxCost float64) ([]CashbackReward, error)
	GetReward(ctx context.Context, id string) (*CashbackReward, error)
	CreateReward(ctx context.Context, r *CashbackReward) error
	UpdateReward(ctx context.Context, r *CashbackReward) error

	CreateRedemption(ctx context.Context, r *CashbackRedemption) error
	GetRedemptionByCode(ctx context.Context, code string) (*CashbackRedemption, error)
	ConfirmRedemption(ctx context.Context, id, collaboratorID string) error
	ExpirePendingRedemptions(ctx context.Context) (int, error)

	CreateFeedback(ctx context.Context, clientID, customerID, message string) error

	// Admin CRUD
	ListPrograms(ctx context.Context, customerID string) ([]CashbackProgram, error)
	ListAllRewards(ctx context.Context, customerSisfiID string) ([]CashbackReward, error)
	CreateRewardAdmin(ctx context.Context, customerSisfiID string, r *CashbackReward) error
	UpdateRewardAdmin(ctx context.Context, r *CashbackReward) error

	GetClientPhone(ctx context.Context, clientID string) (string, error)

	// Transactional
	AddCashbackTx(ctx context.Context, t *CashbackTransaction) (float64, error)
	BurnCashbackTx(ctx context.Context, t *CashbackTransaction, rd *CashbackRedemption) error
	AdjustCashbackTx(ctx context.Context, t *CashbackTransaction) (float64, error)
	EnsureBalance(ctx context.Context, clientID, customerSisfiID string) error
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error
}

// PostgresRepository implements Repository.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// cashbackConfigCols are the config_cashback columns loaded onto every program.
const cashbackConfigCols = "cc.cashback_rate, cc.expiry_days, cc.min_ticket_amount, cc.max_cashback_per_tx, cc.max_cashback_per_period"

// cashbackConfigScan holds the nullable config columns for scanning.
type cashbackConfigScan struct {
	rate      float64
	expiry    sql.NullInt64
	minTicket sql.NullFloat64
	maxTx     sql.NullFloat64
	maxPeriod sql.NullFloat64
}

func (s *cashbackConfigScan) apply(p *CashbackProgram) {
	p.CashbackRate = s.rate
	if s.expiry.Valid {
		d := int(s.expiry.Int64)
		p.ExpiryDays = &d
	}
	if s.minTicket.Valid {
		v := s.minTicket.Float64
		p.MinTicketAmount = &v
	}
	if s.maxTx.Valid {
		v := s.maxTx.Float64
		p.MaxCashbackPerTx = &v
	}
	if s.maxPeriod.Valid {
		v := s.maxPeriod.Float64
		p.MaxCashbackPerPeriod = &v
	}
}

func (r *PostgresRepository) GetProgram(ctx context.Context, customerID string) (*CashbackProgram, error) {
	var p CashbackProgram
	var cfg cashbackConfigScan
	err := r.db.QueryRowContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, `+cashbackConfigCols+`, cs.active
		 FROM customer_sisfi cs
		 JOIN config_cashback cc ON cc.customer_sisfi_id = cs.id
		 WHERE cs.customer_id = $1 AND cs.sisfi_id = 'cashback' AND cs.active = true`, customerID,
	).Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &cfg.rate, &cfg.expiry, &cfg.minTicket, &cfg.maxTx, &cfg.maxPeriod, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get cashback program: %w", err)
	}
	cfg.apply(&p)
	return &p, nil
}

// CreateProgram activates cashback for a customer: inserts customer_sisfi link
// and config_cashback row in a single transaction. Sets p.CustomerSisfiID on success.
// CashbackRate is stored as a fraction (0 < rate <= 1).
func (r *PostgresRepository) CreateProgram(ctx context.Context, p *CashbackProgram) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var customerSisfiID string
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO customer_sisfi (customer_id, sisfi_id, name)
		VALUES ($1, 'cashback', $2)
		RETURNING id
	`, p.CustomerID, p.Name).Scan(&customerSisfiID); err != nil {
		return fmt.Errorf("insert customer_sisfi: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO config_cashback (customer_sisfi_id, cashback_rate, expiry_days, min_ticket_amount, max_cashback_per_tx, max_cashback_per_period)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, customerSisfiID, p.CashbackRate, nullableInt(p.ExpiryDays), nullableFloat(p.MinTicketAmount),
		nullableFloat(p.MaxCashbackPerTx), nullableFloat(p.MaxCashbackPerPeriod)); err != nil {
		return fmt.Errorf("insert config_cashback: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	p.CustomerSisfiID = customerSisfiID
	p.Active = true
	return nil
}

func (r *PostgresRepository) GetProgramByID(ctx context.Context, customerSisfiID string) (*CashbackProgram, error) {
	var p CashbackProgram
	var cfg cashbackConfigScan
	err := r.db.QueryRowContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, `+cashbackConfigCols+`, cs.active
		 FROM customer_sisfi cs
		 JOIN config_cashback cc ON cc.customer_sisfi_id = cs.id
		 WHERE cs.id = $1`, customerSisfiID,
	).Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &cfg.rate, &cfg.expiry, &cfg.minTicket, &cfg.maxTx, &cfg.maxPeriod, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get cashback program by id: %w", err)
	}
	cfg.apply(&p)
	return &p, nil
}

// UpdateProgram updates the customer_sisfi name/active and the config_cashback
// row.
//
// SEMÁNTICA (full-replace en config, LG-1/CF-2): expiry_days, min_ticket_amount,
// max_cashback_per_tx y max_cashback_per_period se asignan SIEMPRE con el valor
// recibido. Un puntero nil escribe NULL (limpia el límite: "sin vencimiento" /
// "sin mínimo" / "sin cap"); NO deja el valor previo. Por eso el frontend debe
// enviar SIEMPRE todos los campos de config actuales en cada PUT (incluido al
// alternar `active`), o de lo contrario los borraría.
// Excepción: cashback_rate usa COALESCE, así que un rate <= 0 (mapeado a nil) sí
// preserva el valor previo, ya que un programa no puede quedar sin tasa.
func (r *PostgresRepository) UpdateProgram(ctx context.Context, p *CashbackProgram, setActive *bool) error {
	return r.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE customer_sisfi
			SET name = COALESCE(NULLIF($2, ''), name),
			    active = COALESCE($3, active)
			WHERE id = $1
		`, p.CustomerSisfiID, p.Name, setActive); err != nil {
			return fmt.Errorf("update customer_sisfi: %w", err)
		}

		var rate interface{}
		if p.CashbackRate > 0 {
			rate = p.CashbackRate
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE config_cashback
			SET cashback_rate           = COALESCE($2::NUMERIC, cashback_rate),
			    expiry_days             = $3,
			    min_ticket_amount       = $4,
			    max_cashback_per_tx     = $5,
			    max_cashback_per_period = $6,
			    updated_at              = NOW()
			WHERE customer_sisfi_id = $1
		`, p.CustomerSisfiID, rate, nullableInt(p.ExpiryDays), nullableFloat(p.MinTicketAmount),
			nullableFloat(p.MaxCashbackPerTx), nullableFloat(p.MaxCashbackPerPeriod)); err != nil {
			return fmt.Errorf("update config_cashback: %w", err)
		}
		return nil
	})
}

// SumCashbackInWindow (FID-37) returns the total positive cashback ("earn")
// credited to a client within the last windowDays.
//
// NOTE (LG-2): el techo por periodo ya NO se enforcea con esta función fuera de
// la transacción (tenía una carrera check-then-insert). AddCashbackTx lee la
// misma suma DENTRO de la tx y recorta el monto de forma atómica. Esta función
// queda para consultas/reportes de solo lectura.
func (r *PostgresRepository) SumCashbackInWindow(ctx context.Context, clientID, customerSisfiID string, windowDays int) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions_cashback
		WHERE client_id = $1 AND customer_sisfi_id = $2
		  AND type = 'earn' AND amount > 0
		  AND created_at >= NOW() - ($3 * INTERVAL '1 day')
	`, clientID, customerSisfiID, windowDays).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum cashback in window: %w", err)
	}
	return total, nil
}

// ExpireBalance (FID-34) posts a compensating "expiration" transaction for each
// earn transaction older than expiryDays not yet expired, then returns the
// resulting balance. Idempotent (tracked by an expiration tx referencing the
// earn tx id in description). Balance floors at 0.
func (r *PostgresRepository) ExpireBalance(ctx context.Context, clientID, customerSisfiID string, expiryDays int) (float64, error) {
	var newBalance float64
	err := r.WithTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT e.id, e.amount
			FROM transactions_cashback e
			WHERE e.client_id = $1 AND e.customer_sisfi_id = $2
			  AND e.type = 'earn' AND e.amount > 0
			  AND e.created_at < NOW() - ($3 * INTERVAL '1 day')
			  AND NOT EXISTS (
			      SELECT 1 FROM transactions_cashback x
			      WHERE x.type = 'expiration' AND x.description = e.id::text
			  )
		`, clientID, customerSisfiID, expiryDays)
		if err != nil {
			return fmt.Errorf("select expiring cashback: %w", err)
		}
		type expired struct {
			id     string
			amount float64
		}
		var toExpire []expired
		for rows.Next() {
			var e expired
			if err := rows.Scan(&e.id, &e.amount); err != nil {
				rows.Close()
				return fmt.Errorf("scan expiring cashback: %w", err)
			}
			toExpire = append(toExpire, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate expiring cashback: %w", err)
		}

		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE((SELECT balance FROM balances_cashback WHERE client_id = $1 AND customer_sisfi_id = $2), 0)`,
			clientID, customerSisfiID,
		).Scan(&newBalance); err != nil {
			return fmt.Errorf("read cashback balance: %w", err)
		}

		for _, e := range toExpire {
			if err := tx.QueryRowContext(ctx, `
				UPDATE balances_cashback
				SET balance = GREATEST(0, balance - $3::NUMERIC), updated_at = NOW()
				WHERE client_id = $1 AND customer_sisfi_id = $2
				RETURNING balance
			`, clientID, customerSisfiID, e.amount).Scan(&newBalance); err != nil {
				if err == sql.ErrNoRows {
					newBalance = 0
				} else {
					return fmt.Errorf("deduct expired cashback: %w", err)
				}
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO transactions_cashback (id, client_id, customer_sisfi_id, type, amount, balance_after, description)
				VALUES ($1, $2, $3, 'expiration', $4::NUMERIC, $5::NUMERIC, $6)
			`, generateUUID(), clientID, customerSisfiID, -e.amount, newBalance, e.id); err != nil {
				return fmt.Errorf("insert cashback expiration tx: %w", err)
			}
		}
		return nil
	})
	return newBalance, err
}

func nullableInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullableFloat(v *float64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func (r *PostgresRepository) GetBalance(ctx context.Context, clientID, customerSisfiID string) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx,
		`SELECT balance FROM balances_cashback WHERE client_id = $1 AND customer_sisfi_id = $2`,
		clientID, customerSisfiID,
	).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get cashback balance: %w", err)
	}
	return balance, nil
}

func (r *PostgresRepository) UpsertBalance(ctx context.Context, clientID, customerSisfiID string, delta float64) (float64, error) {
	var newBalance float64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO balances_cashback (client_id, customer_sisfi_id, balance)
		VALUES ($1, $2, GREATEST(0, $3::NUMERIC))
		ON CONFLICT (client_id, customer_sisfi_id) DO UPDATE
		SET balance = GREATEST(0, balances_cashback.balance + $3::NUMERIC), updated_at = NOW()
		RETURNING balance
	`, clientID, customerSisfiID, delta).Scan(&newBalance)
	if err != nil {
		return 0, fmt.Errorf("upsert cashback balance: %w", err)
	}
	return newBalance, nil
}

func (r *PostgresRepository) CreateTransaction(ctx context.Context, tx *CashbackTransaction) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO transactions_cashback
		(id, client_id, customer_sisfi_id, collaborator_id, type, amount, purchase_amount, balance_after, invoice_url, description, manual_entry, correction_reason, correction_evidence_url, correctable_until)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6::NUMERIC, $7::NUMERIC, $8::NUMERIC, NULLIF($9, ''), NULLIF($10, ''), $11, NULLIF($12, ''), NULLIF($13, ''), $14)
	`, tx.ID, tx.ClientID, tx.CustomerSisfiID, tx.CollaboratorID, tx.Type, tx.Amount, tx.PurchaseAmount, tx.BalanceAfter,
		tx.InvoiceURL, tx.Description, tx.ManualEntry, tx.CorrectionReason, tx.CorrectionEvidenceURL, tx.CorrectableUntil)
	if err != nil {
		return fmt.Errorf("create cashback transaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetTransaction(ctx context.Context, id string) (*CashbackTransaction, error) {
	var tx CashbackTransaction
	var collabID, invoiceURL, desc, corrReason, corrEvidence sql.NullString
	var purchaseAmount sql.NullFloat64
	var correctableUntil sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, client_id, customer_sisfi_id, collaborator_id, type, amount, purchase_amount, balance_after,
		       invoice_url, description, manual_entry, correction_reason, correction_evidence_url,
		       correctable_until, created_at
		FROM transactions_cashback WHERE id = $1
	`, id).Scan(&tx.ID, &tx.ClientID, &tx.CustomerSisfiID, &collabID, &tx.Type, &tx.Amount, &purchaseAmount, &tx.BalanceAfter,
		&invoiceURL, &desc, &tx.ManualEntry, &corrReason, &corrEvidence, &correctableUntil, &tx.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get cashback transaction: %w", err)
	}
	tx.CollaboratorID = collabID.String
	tx.InvoiceURL = invoiceURL.String
	tx.Description = desc.String
	tx.CorrectionReason = corrReason.String
	tx.CorrectionEvidenceURL = corrEvidence.String
	if purchaseAmount.Valid {
		tx.PurchaseAmount = purchaseAmount.Float64
	}
	if correctableUntil.Valid {
		tx.CorrectableUntil = &correctableUntil.Time
	}
	return &tx, nil
}

func (r *PostgresRepository) ListTransactions(ctx context.Context, clientID, customerSisfiID string, limit int) ([]CashbackTransaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, amount, balance_after, description, created_at
		FROM transactions_cashback
		WHERE client_id = $1 AND customer_sisfi_id = $2
		ORDER BY created_at DESC LIMIT $3
	`, clientID, customerSisfiID, limit)
	if err != nil {
		return nil, fmt.Errorf("list cashback transactions: %w", err)
	}
	defer rows.Close()

	var txs []CashbackTransaction
	for rows.Next() {
		var tx CashbackTransaction
		var desc sql.NullString
		if err := rows.Scan(&tx.ID, &tx.Type, &tx.Amount, &tx.BalanceAfter, &desc, &tx.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan cashback transaction: %w", err)
		}
		tx.Description = desc.String
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *PostgresRepository) ListCorrectableTransactions(ctx context.Context, clientID string) ([]CashbackTransaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, amount, purchase_amount, balance_after, created_at, correctable_until
		FROM transactions_cashback
		WHERE client_id = $1 AND correctable_until > NOW() AND type = 'earn'
		ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list correctable cashback: %w", err)
	}
	defer rows.Close()

	var txs []CashbackTransaction
	for rows.Next() {
		var tx CashbackTransaction
		var purchaseAmount sql.NullFloat64
		var cu sql.NullTime
		if err := rows.Scan(&tx.ID, &tx.Type, &tx.Amount, &purchaseAmount, &tx.BalanceAfter, &tx.CreatedAt, &cu); err != nil {
			return nil, fmt.Errorf("scan correctable cashback: %w", err)
		}
		if purchaseAmount.Valid {
			tx.PurchaseAmount = purchaseAmount.Float64
		}
		if cu.Valid {
			tx.CorrectableUntil = &cu.Time
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *PostgresRepository) GetClientName(ctx context.Context, clientID string) (string, error) {
	var name sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT name FROM clients WHERE id = $1`, clientID,
	).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("get client name: %w", err)
	}
	return name.String, nil
}

func (r *PostgresRepository) GetClientPhone(ctx context.Context, clientID string) (string, error) {
	var phone string
	err := r.db.QueryRowContext(ctx,
		`SELECT phone FROM clients WHERE id = $1`, clientID,
	).Scan(&phone)
	if err != nil {
		return "", fmt.Errorf("get client phone: %w", err)
	}
	return phone, nil
}

func (r *PostgresRepository) ListRewards(ctx context.Context, customerID, customerSisfiID string, maxCost float64) ([]CashbackReward, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, customer_id, customer_sisfi_id, name, COALESCE(description, ''), cost, active
		FROM rewards_cashback
		WHERE customer_id = $1 AND customer_sisfi_id = $2 AND active = true AND cost <= $3::NUMERIC
		ORDER BY cost ASC
	`, customerID, customerSisfiID, maxCost)
	if err != nil {
		return nil, fmt.Errorf("list cashback rewards: %w", err)
	}
	defer rows.Close()

	var rewards []CashbackReward
	for rows.Next() {
		var rw CashbackReward
		if err := rows.Scan(&rw.ID, &rw.CustomerID, &rw.CustomerSisfiID, &rw.Name, &rw.Description, &rw.Cost, &rw.Active); err != nil {
			return nil, fmt.Errorf("scan cashback reward: %w", err)
		}
		rewards = append(rewards, rw)
	}
	return rewards, nil
}

func (r *PostgresRepository) GetReward(ctx context.Context, id string) (*CashbackReward, error) {
	var rw CashbackReward
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, customer_sisfi_id, name, description, cost, active FROM rewards_cashback WHERE id = $1`, id,
	).Scan(&rw.ID, &rw.CustomerID, &rw.CustomerSisfiID, &rw.Name, &desc, &rw.Cost, &rw.Active)
	if err != nil {
		return nil, fmt.Errorf("get cashback reward: %w", err)
	}
	rw.Description = desc.String
	return &rw, nil
}

func (r *PostgresRepository) CreateReward(ctx context.Context, rw *CashbackReward) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO rewards_cashback (id, customer_id, customer_sisfi_id, name, description, cost, active)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6::NUMERIC, $7)
	`, rw.ID, rw.CustomerID, rw.CustomerSisfiID, rw.Name, rw.Description, rw.Cost, rw.Active)
	return err
}

func (r *PostgresRepository) UpdateReward(ctx context.Context, rw *CashbackReward) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE rewards_cashback SET name = $2, description = NULLIF($3, ''), cost = $4::NUMERIC, active = $5, updated_at = NOW()
		WHERE id = $1
	`, rw.ID, rw.Name, rw.Description, rw.Cost, rw.Active)
	return err
}

func (r *PostgresRepository) CreateRedemption(ctx context.Context, rd *CashbackRedemption) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO redemptions_cashback (id, client_id, reward_id, customer_sisfi_id, code, status, amount_spent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::NUMERIC, $8)
	`, rd.ID, rd.ClientID, rd.RewardID, rd.CustomerSisfiID, rd.Code, rd.Status, rd.AmountSpent, rd.ExpiresAt)
	return err
}

func (r *PostgresRepository) GetRedemptionByCode(ctx context.Context, code string) (*CashbackRedemption, error) {
	var rd CashbackRedemption
	var confirmedBy sql.NullString
	var confirmedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, client_id, reward_id, customer_sisfi_id, code, status, amount_spent, confirmed_by, expires_at, confirmed_at, created_at
		FROM redemptions_cashback WHERE code = $1
	`, code).Scan(&rd.ID, &rd.ClientID, &rd.RewardID, &rd.CustomerSisfiID, &rd.Code, &rd.Status,
		&rd.AmountSpent, &confirmedBy, &rd.ExpiresAt, &confirmedAt, &rd.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get cashback redemption by code: %w", err)
	}
	rd.ConfirmedBy = confirmedBy.String
	if confirmedAt.Valid {
		rd.ConfirmedAt = &confirmedAt.Time
	}
	return &rd, nil
}

func (r *PostgresRepository) ConfirmRedemption(ctx context.Context, id, collaboratorID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE redemptions_cashback SET status = 'confirmed', confirmed_by = $2, confirmed_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, id, collaboratorID)
	return err
}

func (r *PostgresRepository) ExpirePendingRedemptions(ctx context.Context) (int, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE redemptions_cashback SET status = 'expired'
		WHERE status = 'pending' AND expires_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("expire cashback redemptions: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *PostgresRepository) CreateFeedback(ctx context.Context, clientID, customerID, message string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO feedback (id, client_id, customer_id, message) VALUES (gen_random_uuid(), $1, $2, $3)`,
		clientID, customerID, message)
	return err
}

func (r *PostgresRepository) EnsureBalance(ctx context.Context, clientID, customerSisfiID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO balances_cashback (client_id, customer_sisfi_id, balance)
		VALUES ($1, $2, 0)
		ON CONFLICT (client_id, customer_sisfi_id) DO NOTHING
	`, clientID, customerSisfiID)
	return err
}

func (r *PostgresRepository) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// AddCashbackTx atomically creates a transaction and updates balance.
//
// FID-37 (LG-2, cap por periodo atómico): cuando t.PeriodCap es no-nil, la suma
// de la ventana se lee DENTRO de esta misma transacción y el monto a acreditar
// (t.Amount) se recorta al remanente antes de tocar el balance. Así dos requests
// concurrentes no pueden exceder el techo (la lectura+clamp+insert son atómicos
// bajo la misma tx). Si el periodo ya está agotado devuelve ErrPeriodCapExhausted.
func (r *PostgresRepository) AddCashbackTx(ctx context.Context, t *CashbackTransaction) (float64, error) {
	var newBalance float64
	err := r.WithTx(ctx, func(tx *sql.Tx) error {
		// Cap por periodo: leer la ventana DENTRO de la tx y recortar el monto.
		if t.PeriodCap != nil {
			var accumulated float64
			if err := tx.QueryRowContext(ctx, `
				SELECT COALESCE(SUM(amount), 0)
				FROM transactions_cashback
				WHERE client_id = $1 AND customer_sisfi_id = $2
				  AND type = 'earn' AND amount > 0
				  AND created_at >= NOW() - ($3 * INTERVAL '1 day')
			`, t.ClientID, t.CustomerSisfiID, t.PeriodWindowDays).Scan(&accumulated); err != nil {
				return fmt.Errorf("sum cashback in window: %w", err)
			}
			remaining := *t.PeriodCap - accumulated
			if remaining <= 0 {
				return ErrPeriodCapExhausted
			}
			if t.Amount > remaining {
				t.Amount = math.Floor(remaining*100) / 100
			}
			if t.Amount <= 0 {
				return ErrPeriodCapExhausted
			}
		}

		err := tx.QueryRowContext(ctx, `
			INSERT INTO balances_cashback (client_id, customer_sisfi_id, balance)
			VALUES ($1, $2, GREATEST(0, $3::NUMERIC))
			ON CONFLICT (client_id, customer_sisfi_id) DO UPDATE
			SET balance = GREATEST(0, balances_cashback.balance + $3::NUMERIC), updated_at = NOW()
			RETURNING balance
		`, t.ClientID, t.CustomerSisfiID, t.Amount).Scan(&newBalance)
		if err != nil {
			return fmt.Errorf("upsert cashback balance: %w", err)
		}

		t.BalanceAfter = newBalance

		// receipt_hash is inserted as NULL when empty so the partial unique index
		// only guards confidently-hashed tickets. A unique violation rolls back the
		// whole tx (balance included), so a duplicate ticket never credits cashback.
		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions_cashback
			(id, client_id, customer_sisfi_id, collaborator_id, type, amount, purchase_amount, balance_after, invoice_url, description, manual_entry, correctable_until, receipt_data, receipt_hash, receipt_hash_fields, receipt_confident)
			VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6::NUMERIC, $7::NUMERIC, $8::NUMERIC, NULLIF($9, ''), NULLIF($10, ''), $11, $12, $13, NULLIF($14, ''), $15, $16)
		`, t.ID, t.ClientID, t.CustomerSisfiID, t.CollaboratorID, t.Type, t.Amount, t.PurchaseAmount, t.BalanceAfter,
			t.InvoiceURL, t.Description, t.ManualEntry, t.CorrectableUntil,
			nullableJSON(t.ReceiptData), t.ReceiptHash, pq.Array(t.ReceiptHashFields), t.ReceiptConfident)
		if err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicateReceipt
			}
			return err
		}
		return nil
	})
	return newBalance, err
}

// BurnCashbackTx atomically deducts cashback and creates a redemption record.
func (r *PostgresRepository) BurnCashbackTx(ctx context.Context, t *CashbackTransaction, rd *CashbackRedemption) error {
	return r.WithTx(ctx, func(tx *sql.Tx) error {
		var newBalance float64
		err := tx.QueryRowContext(ctx, `
			UPDATE balances_cashback
			SET balance = balance + $3::NUMERIC, updated_at = NOW()
			WHERE client_id = $1 AND customer_sisfi_id = $2 AND balance >= $4::NUMERIC
			RETURNING balance
		`, t.ClientID, t.CustomerSisfiID, t.Amount, -t.Amount).Scan(&newBalance)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("saldo insuficiente")
			}
			return fmt.Errorf("deduct cashback balance: %w", err)
		}

		t.BalanceAfter = newBalance

		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions_cashback (id, client_id, customer_sisfi_id, type, amount, balance_after, description)
			VALUES ($1, $2, $3, $4, $5::NUMERIC, $6::NUMERIC, $7)
		`, t.ID, t.ClientID, t.CustomerSisfiID, t.Type, t.Amount, t.BalanceAfter, t.Description)
		if err != nil {
			return fmt.Errorf("create burn cashback tx: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO redemptions_cashback (id, client_id, reward_id, customer_sisfi_id, code, status, amount_spent, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7::NUMERIC, $8)
		`, rd.ID, rd.ClientID, rd.RewardID, rd.CustomerSisfiID, rd.Code, rd.Status, rd.AmountSpent, rd.ExpiresAt)
		return err
	})
}

// AdjustCashbackTx atomically adjusts cashback for a correction.
func (r *PostgresRepository) AdjustCashbackTx(ctx context.Context, t *CashbackTransaction) (float64, error) {
	var newBalance float64
	err := r.WithTx(ctx, func(tx *sql.Tx) error {
		var correctableUntil time.Time
		err := tx.QueryRowContext(ctx,
			`SELECT correctable_until FROM transactions_cashback WHERE id = $1 AND correctable_until > NOW()`,
			t.Description, // description stores the original tx ID for adjustments
		).Scan(&correctableUntil)
		if err != nil {
			return fmt.Errorf("transaction not correctable: %w", err)
		}

		err = tx.QueryRowContext(ctx, `
			UPDATE balances_cashback
			SET balance = GREATEST(0, balance + $3::NUMERIC), updated_at = NOW()
			WHERE client_id = $1 AND customer_sisfi_id = $2
			RETURNING balance
		`, t.ClientID, t.CustomerSisfiID, t.Amount).Scan(&newBalance)
		if err != nil {
			return fmt.Errorf("adjust cashback balance: %w", err)
		}

		t.BalanceAfter = newBalance

		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions_cashback
			(id, client_id, customer_sisfi_id, collaborator_id, type, amount, balance_after, correction_reason, correction_evidence_url)
			VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6::NUMERIC, $7::NUMERIC, NULLIF($8, ''), NULLIF($9, ''))
		`, t.ID, t.ClientID, t.CustomerSisfiID, t.CollaboratorID, t.Type, t.Amount, t.BalanceAfter,
			t.CorrectionReason, t.CorrectionEvidenceURL)
		return err
	})
	return newBalance, err
}

// --- Admin CRUD ---

func (r *PostgresRepository) ListPrograms(ctx context.Context, customerID string) ([]CashbackProgram, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, `+cashbackConfigCols+`, cs.active
		 FROM customer_sisfi cs
		 JOIN config_cashback cc ON cc.customer_sisfi_id = cs.id
		 WHERE cs.customer_id = $1 AND cs.sisfi_id = 'cashback' AND cs.active = true
		 ORDER BY cs.created_at`,
		customerID)
	if err != nil {
		return nil, fmt.Errorf("list cashback programs: %w", err)
	}
	defer rows.Close()

	var programs []CashbackProgram
	for rows.Next() {
		var p CashbackProgram
		var cfg cashbackConfigScan
		if err := rows.Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &cfg.rate, &cfg.expiry, &cfg.minTicket, &cfg.maxTx, &cfg.maxPeriod, &p.Active); err != nil {
			return nil, fmt.Errorf("scan cashback program: %w", err)
		}
		cfg.apply(&p)
		programs = append(programs, p)
	}
	return programs, nil
}

func (r *PostgresRepository) ListAllRewards(ctx context.Context, customerSisfiID string) ([]CashbackReward, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, customer_sisfi_id, name, COALESCE(description, ''), cost, active FROM rewards_cashback
		 WHERE customer_sisfi_id = $1 ORDER BY cost`, customerSisfiID)
	if err != nil {
		return nil, fmt.Errorf("list all cashback rewards: %w", err)
	}
	defer rows.Close()

	var rewards []CashbackReward
	for rows.Next() {
		var rw CashbackReward
		if err := rows.Scan(&rw.ID, &rw.CustomerID, &rw.CustomerSisfiID, &rw.Name, &rw.Description, &rw.Cost, &rw.Active); err != nil {
			return nil, fmt.Errorf("scan cashback reward: %w", err)
		}
		rewards = append(rewards, rw)
	}
	return rewards, nil
}

func (r *PostgresRepository) CreateRewardAdmin(ctx context.Context, customerSisfiID string, rw *CashbackReward) error {
	var customerID string
	err := r.db.QueryRowContext(ctx,
		`SELECT customer_id FROM customer_sisfi WHERE id = $1`, customerSisfiID,
	).Scan(&customerID)
	if err != nil {
		return fmt.Errorf("get customer_sisfi for reward: %w", err)
	}

	return r.db.QueryRowContext(ctx,
		`INSERT INTO rewards_cashback (customer_id, customer_sisfi_id, name, description, cost)
		 VALUES ($1, $2, $3, NULLIF($4, ''), $5::NUMERIC) RETURNING id`,
		customerID, customerSisfiID, rw.Name, rw.Description, rw.Cost,
	).Scan(&rw.ID)
}

func (r *PostgresRepository) UpdateRewardAdmin(ctx context.Context, rw *CashbackReward) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE rewards_cashback SET name = COALESCE(NULLIF($2, ''), name),
		 description = COALESCE(NULLIF($3, ''), description),
		 cost = CASE WHEN $4::NUMERIC > 0 THEN $4::NUMERIC ELSE cost END,
		 active = COALESCE($5, active), updated_at = NOW() WHERE id = $1`,
		rw.ID, rw.Name, rw.Description, rw.Cost, rw.Active,
	)
	return err
}
