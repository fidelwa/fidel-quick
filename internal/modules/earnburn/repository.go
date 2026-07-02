package earnburn

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// ErrDuplicateReceipt is returned when a receipt with the same canonical hash was
// already registered for the same customer_sisfi (business + program). Detected
// via the partial unique index idx_transactions_earnburn_receipt_hash.
var ErrDuplicateReceipt = errors.New("ticket ya registrado")

// isUniqueViolation reports whether err is a Postgres unique_violation (23505).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

type Repository interface {
	GetProgram(ctx context.Context, customerID string) (*EarnBurnProgram, error)
	GetProgramByID(ctx context.Context, customerSisfiID string) (*EarnBurnProgram, error)
	ListPrograms(ctx context.Context, customerID string) ([]EarnBurnProgram, error)
	CreateProgram(ctx context.Context, p *EarnBurnProgram) error
	UpdateProgram(ctx context.Context, p *EarnBurnProgram, setActive *bool) error
	ExpirePoints(ctx context.Context, clientID, customerSisfiID string, expiryDays int) (int, error)
	GetBalance(ctx context.Context, clientID, customerSisfiID string) (int, error)
	UpsertBalance(ctx context.Context, clientID, customerSisfiID string, delta int) (int, error)
	CreateTransaction(ctx context.Context, tx *Transaction) error
	GetTransaction(ctx context.Context, id string) (*Transaction, error)
	ListTransactions(ctx context.Context, clientID, customerSisfiID string, limit int) ([]Transaction, error)
	ListCorrectableTransactions(ctx context.Context, clientID string) ([]Transaction, error)
	GetClientName(ctx context.Context, clientID string) (string, error)

	ListRewards(ctx context.Context, customerID, customerSisfiID string, maxPoints int) ([]Reward, error)
	GetReward(ctx context.Context, id string) (*Reward, error)
	CreateReward(ctx context.Context, r *Reward) error
	UpdateReward(ctx context.Context, r *Reward) error

	CreateRedemption(ctx context.Context, r *Redemption) error
	GetRedemptionByCode(ctx context.Context, code string) (*Redemption, error)
	ConfirmRedemption(ctx context.Context, id, collaboratorID string) error
	ExpirePendingRedemptions(ctx context.Context) (int, error)

	CreateFeedback(ctx context.Context, clientID, customerID, message string) error

	// Admin CRUD
	GetCustomer(ctx context.Context, id string) (*Customer, error)
	CreateCustomer(ctx context.Context, c *Customer) error
	UpdateCustomer(ctx context.Context, c *Customer) error
	CreateCollaborator(ctx context.Context, c *Collaborator) error
	ListCollaborators(ctx context.Context, customerID string) ([]Collaborator, error)
	ListAllRewards(ctx context.Context, customerSisfiID string) ([]Reward, error)
	CreateRewardAdmin(ctx context.Context, customerSisfiID string, r *Reward) error
	UpdateRewardAdmin(ctx context.Context, r *Reward) error
	ListFeedback(ctx context.Context, customerID string) ([]FeedbackEntry, error)
	ListClients(ctx context.Context, customerID string) ([]Client, error)
	RegisterClient(ctx context.Context, customerID, phone string) error

	GetClientPhone(ctx context.Context, clientID string) (string, error)

	// Transactional
	AddPointsTx(ctx context.Context, t *Transaction) (int, error)
	BurnPointsTx(ctx context.Context, t *Transaction, rd *Redemption) error
	AdjustPointsTx(ctx context.Context, t *Transaction) (int, error)
	EnsureBalance(ctx context.Context, clientID, customerSisfiID string) error
}

// PostgresRepository implements Repository.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetProgram(ctx context.Context, customerID string) (*EarnBurnProgram, error) {
	var p EarnBurnProgram
	var expiryDays sql.NullInt64
	var minTicket sql.NullFloat64
	err := r.db.QueryRowContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, ec.points_ratio, cs.active, ec.expiry_days, ec.min_ticket_amount
		 FROM customer_sisfi cs
		 JOIN config_earnburn ec ON ec.customer_sisfi_id = cs.id
		 WHERE cs.customer_id = $1 AND cs.sisfi_id = 'earn_burn' AND cs.active = true`, customerID,
	).Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &p.PointsRatio, &p.Active, &expiryDays, &minTicket)
	if err != nil {
		return nil, fmt.Errorf("get program: %w", err)
	}
	applyEarnConfig(&p, expiryDays, minTicket)
	return &p, nil
}

// applyEarnConfig maps nullable config columns onto the program struct.
func applyEarnConfig(p *EarnBurnProgram, expiryDays sql.NullInt64, minTicket sql.NullFloat64) {
	if expiryDays.Valid {
		d := int(expiryDays.Int64)
		p.ExpiryDays = &d
	}
	if minTicket.Valid {
		m := minTicket.Float64
		p.MinTicketAmount = &m
	}
}

// CreateProgram activates earn_burn for a customer: inserts customer_sisfi link
// and config_earnburn row in a single transaction. Sets p.CustomerSisfiID on success.
func (r *PostgresRepository) CreateProgram(ctx context.Context, p *EarnBurnProgram) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var customerSisfiID string
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO customer_sisfi (customer_id, sisfi_id, name)
		VALUES ($1, 'earn_burn', $2)
		RETURNING id
	`, p.CustomerID, p.Name).Scan(&customerSisfiID); err != nil {
		return fmt.Errorf("insert customer_sisfi: %w", err)
	}

	var expiryDays interface{}
	if p.ExpiryDays != nil {
		expiryDays = *p.ExpiryDays
	}
	var minTicket interface{}
	if p.MinTicketAmount != nil {
		minTicket = *p.MinTicketAmount
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO config_earnburn (customer_sisfi_id, points_ratio, expiry_days, min_ticket_amount)
		VALUES ($1, $2, $3, $4)
	`, customerSisfiID, p.PointsRatio, expiryDays, minTicket); err != nil {
		return fmt.Errorf("insert config_earnburn: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	p.CustomerSisfiID = customerSisfiID
	p.Active = true
	return nil
}

func (r *PostgresRepository) GetProgramByID(ctx context.Context, customerSisfiID string) (*EarnBurnProgram, error) {
	var p EarnBurnProgram
	var expiryDays sql.NullInt64
	var minTicket sql.NullFloat64
	err := r.db.QueryRowContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, ec.points_ratio, cs.active, ec.expiry_days, ec.min_ticket_amount
		 FROM customer_sisfi cs
		 JOIN config_earnburn ec ON ec.customer_sisfi_id = cs.id
		 WHERE cs.id = $1`, customerSisfiID,
	).Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &p.PointsRatio, &p.Active, &expiryDays, &minTicket)
	if err != nil {
		return nil, fmt.Errorf("get program by id: %w", err)
	}
	applyEarnConfig(&p, expiryDays, minTicket)
	return &p, nil
}

// UpdateProgram updates the customer_sisfi name/active and the config_earnburn
// row for a program.
//
// SEMÁNTICA (full-replace en config, LG-1): expiry_days y min_ticket_amount se
// asignan SIEMPRE con el valor recibido. Un puntero nil escribe NULL (limpia el
// límite: "sin vencimiento" / "sin mínimo"); NO deja el valor previo. Por eso el
// frontend debe enviar SIEMPRE todos los campos de config actuales en cada PUT
// (incluido al alternar `active`), o de lo contrario los borraría.
// Excepción: points_ratio usa COALESCE, así que un ratio <= 0 (mapeado a nil) sí
// preserva el valor previo, ya que un programa no puede quedar sin ratio.
func (r *PostgresRepository) UpdateProgram(ctx context.Context, p *EarnBurnProgram, setActive *bool) error {
	return r.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE customer_sisfi
			SET name = COALESCE(NULLIF($2, ''), name),
			    active = COALESCE($3, active)
			WHERE id = $1
		`, p.CustomerSisfiID, p.Name, setActive); err != nil {
			return fmt.Errorf("update customer_sisfi: %w", err)
		}

		var pointsRatio interface{}
		if p.PointsRatio > 0 {
			pointsRatio = p.PointsRatio
		}
		var expiryDays interface{}
		if p.ExpiryDays != nil {
			expiryDays = *p.ExpiryDays
		}
		var minTicket interface{}
		if p.MinTicketAmount != nil {
			minTicket = *p.MinTicketAmount
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE config_earnburn
			SET points_ratio      = COALESCE($2, points_ratio),
			    expiry_days       = $3,
			    min_ticket_amount = $4,
			    updated_at        = NOW()
			WHERE customer_sisfi_id = $1
		`, p.CustomerSisfiID, pointsRatio, expiryDays, minTicket); err != nil {
			return fmt.Errorf("update config_earnburn: %w", err)
		}
		return nil
	})
}

// ExpirePoints (FID-34) posts a compensating "expiration" transaction for each
// earn transaction older than expiryDays that has not yet been expired, then
// returns the resulting balance. Idempotent: an earn tx is only expired once
// (tracked by an expiration tx whose description references the earn tx id).
// The balance floor at 0 means over-expiring already-burned points is safe.
func (r *PostgresRepository) ExpirePoints(ctx context.Context, clientID, customerSisfiID string, expiryDays int) (int, error) {
	var newBalance int
	err := r.WithTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT e.id, e.amount
			FROM transactions_earnburn e
			WHERE e.client_id = $1 AND e.customer_sisfi_id = $2
			  AND e.type = 'earn' AND e.amount > 0
			  AND e.created_at < NOW() - ($3 * INTERVAL '1 day')
			  AND NOT EXISTS (
			      SELECT 1 FROM transactions_earnburn x
			      WHERE x.type = 'expiration' AND x.description = e.id::text
			  )
		`, clientID, customerSisfiID, expiryDays)
		if err != nil {
			return fmt.Errorf("select expiring earns: %w", err)
		}
		type expired struct {
			id     string
			amount int
		}
		var toExpire []expired
		for rows.Next() {
			var e expired
			if err := rows.Scan(&e.id, &e.amount); err != nil {
				rows.Close()
				return fmt.Errorf("scan expiring earn: %w", err)
			}
			toExpire = append(toExpire, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate expiring earns: %w", err)
		}

		// Read current balance (default 0 if no row).
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE((SELECT balance FROM balances_earnburn WHERE client_id = $1 AND customer_sisfi_id = $2), 0)`,
			clientID, customerSisfiID,
		).Scan(&newBalance); err != nil {
			return fmt.Errorf("read balance: %w", err)
		}

		for _, e := range toExpire {
			if err := tx.QueryRowContext(ctx, `
				UPDATE balances_earnburn
				SET balance = GREATEST(0, balance - $3), updated_at = NOW()
				WHERE client_id = $1 AND customer_sisfi_id = $2
				RETURNING balance
			`, clientID, customerSisfiID, e.amount).Scan(&newBalance); err != nil {
				if err == sql.ErrNoRows {
					// No balance row => nothing to expire against; still record the marker.
					newBalance = 0
				} else {
					return fmt.Errorf("deduct expired points: %w", err)
				}
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO transactions_earnburn (id, client_id, customer_sisfi_id, type, amount, balance_after, description)
				VALUES ($1, $2, $3, 'expiration', $4, $5, $6)
			`, generateUUID(), clientID, customerSisfiID, -e.amount, newBalance, e.id); err != nil {
				return fmt.Errorf("insert expiration tx: %w", err)
			}
		}
		return nil
	})
	return newBalance, err
}

func (r *PostgresRepository) GetBalance(ctx context.Context, clientID, customerSisfiID string) (int, error) {
	var balance int
	err := r.db.QueryRowContext(ctx,
		`SELECT balance FROM balances_earnburn WHERE client_id = $1 AND customer_sisfi_id = $2`,
		clientID, customerSisfiID,
	).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

func (r *PostgresRepository) UpsertBalance(ctx context.Context, clientID, customerSisfiID string, delta int) (int, error) {
	var newBalance int
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO balances_earnburn (client_id, customer_sisfi_id, balance)
		VALUES ($1, $2, GREATEST(0, $3))
		ON CONFLICT (client_id, customer_sisfi_id) DO UPDATE
		SET balance = GREATEST(0, balances_earnburn.balance + $3), updated_at = NOW()
		RETURNING balance
	`, clientID, customerSisfiID, delta).Scan(&newBalance)
	if err != nil {
		return 0, fmt.Errorf("upsert balance: %w", err)
	}
	return newBalance, nil
}

// CreateTransaction is a legacy non-atomic insert kept for interface parity.
//
// It is NOT part of any credit path: earn goes through AddPointsTx (atomic
// balance + insert with the receipt anti-fraud columns), burn through
// BurnPointsTx and adjustments through AdjustPointsTx. Nothing calls this method
// (verified: only the interface decl and the test mock reference it). Because it
// never accredits, it deliberately does NOT carry the receipt_* columns — adding
// them here would imply a credit path that does not exist. Do not wire this into
// crediting; use AddPointsTx. Left in place (rather than deleted) only to satisfy
// the Repository interface used elsewhere.
func (r *PostgresRepository) CreateTransaction(ctx context.Context, tx *Transaction) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO transactions_earnburn
		(id, client_id, customer_sisfi_id, collaborator_id, type, amount, balance_after, invoice_url, description, manual_entry, correction_reason, correction_evidence_url, correctable_until)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''), $10, NULLIF($11, ''), NULLIF($12, ''), $13)
	`, tx.ID, tx.ClientID, tx.CustomerSisfiID, tx.CollaboratorID, tx.Type, tx.Amount, tx.BalanceAfter,
		tx.InvoiceURL, tx.Description, tx.ManualEntry, tx.CorrectionReason, tx.CorrectionEvidenceURL, tx.CorrectableUntil)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	var tx Transaction
	var collabID, invoiceURL, desc, corrReason, corrEvidence sql.NullString
	var correctableUntil sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, client_id, customer_sisfi_id, collaborator_id, type, amount, balance_after,
		       invoice_url, description, manual_entry, correction_reason, correction_evidence_url,
		       correctable_until, created_at
		FROM transactions_earnburn WHERE id = $1
	`, id).Scan(&tx.ID, &tx.ClientID, &tx.CustomerSisfiID, &collabID, &tx.Type, &tx.Amount, &tx.BalanceAfter,
		&invoiceURL, &desc, &tx.ManualEntry, &corrReason, &corrEvidence, &correctableUntil, &tx.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	tx.CollaboratorID = collabID.String
	tx.InvoiceURL = invoiceURL.String
	tx.Description = desc.String
	tx.CorrectionReason = corrReason.String
	tx.CorrectionEvidenceURL = corrEvidence.String
	if correctableUntil.Valid {
		tx.CorrectableUntil = &correctableUntil.Time
	}
	return &tx, nil
}

func (r *PostgresRepository) ListTransactions(ctx context.Context, clientID, customerSisfiID string, limit int) ([]Transaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, amount, balance_after, description, created_at
		FROM transactions_earnburn
		WHERE client_id = $1 AND customer_sisfi_id = $2
		ORDER BY created_at DESC LIMIT $3
	`, clientID, customerSisfiID, limit)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		var desc sql.NullString
		if err := rows.Scan(&tx.ID, &tx.Type, &tx.Amount, &tx.BalanceAfter, &desc, &tx.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		tx.Description = desc.String
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *PostgresRepository) ListCorrectableTransactions(ctx context.Context, clientID string) ([]Transaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, amount, balance_after, created_at, correctable_until
		FROM transactions_earnburn
		WHERE client_id = $1 AND correctable_until > NOW() AND type = 'earn'
		ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list correctable: %w", err)
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		var cu sql.NullTime
		if err := rows.Scan(&tx.ID, &tx.Type, &tx.Amount, &tx.BalanceAfter, &tx.CreatedAt, &cu); err != nil {
			return nil, fmt.Errorf("scan correctable: %w", err)
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

func (r *PostgresRepository) ListRewards(ctx context.Context, customerID, customerSisfiID string, maxPoints int) ([]Reward, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, customer_id, customer_sisfi_id, name, COALESCE(description, ''), points_cost, active
		FROM rewards_earnburn
		WHERE customer_id = $1 AND customer_sisfi_id = $2 AND active = true AND points_cost <= $3
		ORDER BY points_cost ASC
	`, customerID, customerSisfiID, maxPoints)
	if err != nil {
		return nil, fmt.Errorf("list rewards: %w", err)
	}
	defer rows.Close()

	var rewards []Reward
	for rows.Next() {
		var rw Reward
		if err := rows.Scan(&rw.ID, &rw.CustomerID, &rw.CustomerSisfiID, &rw.Name, &rw.Description, &rw.PointsCost, &rw.Active); err != nil {
			return nil, fmt.Errorf("scan reward: %w", err)
		}
		rewards = append(rewards, rw)
	}
	return rewards, nil
}

func (r *PostgresRepository) GetReward(ctx context.Context, id string) (*Reward, error) {
	var rw Reward
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, customer_sisfi_id, name, description, points_cost, active FROM rewards_earnburn WHERE id = $1`, id,
	).Scan(&rw.ID, &rw.CustomerID, &rw.CustomerSisfiID, &rw.Name, &desc, &rw.PointsCost, &rw.Active)
	if err != nil {
		return nil, fmt.Errorf("get reward: %w", err)
	}
	rw.Description = desc.String
	return &rw, nil
}

func (r *PostgresRepository) CreateReward(ctx context.Context, rw *Reward) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO rewards_earnburn (id, customer_id, customer_sisfi_id, name, description, points_cost, active)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
	`, rw.ID, rw.CustomerID, rw.CustomerSisfiID, rw.Name, rw.Description, rw.PointsCost, rw.Active)
	return err
}

func (r *PostgresRepository) UpdateReward(ctx context.Context, rw *Reward) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE rewards_earnburn SET name = $2, description = NULLIF($3, ''), points_cost = $4, active = $5, updated_at = NOW()
		WHERE id = $1
	`, rw.ID, rw.Name, rw.Description, rw.PointsCost, rw.Active)
	return err
}

func (r *PostgresRepository) CreateRedemption(ctx context.Context, rd *Redemption) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO redemptions_earnburn (id, client_id, reward_id, customer_sisfi_id, code, status, points_spent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, rd.ID, rd.ClientID, rd.RewardID, rd.CustomerSisfiID, rd.Code, rd.Status, rd.PointsSpent, rd.ExpiresAt)
	return err
}

func (r *PostgresRepository) GetRedemptionByCode(ctx context.Context, code string) (*Redemption, error) {
	var rd Redemption
	var confirmedBy sql.NullString
	var confirmedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, client_id, reward_id, customer_sisfi_id, code, status, points_spent, confirmed_by, expires_at, confirmed_at, created_at
		FROM redemptions_earnburn WHERE code = $1
	`, code).Scan(&rd.ID, &rd.ClientID, &rd.RewardID, &rd.CustomerSisfiID, &rd.Code, &rd.Status,
		&rd.PointsSpent, &confirmedBy, &rd.ExpiresAt, &confirmedAt, &rd.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get redemption by code: %w", err)
	}
	rd.ConfirmedBy = confirmedBy.String
	if confirmedAt.Valid {
		rd.ConfirmedAt = &confirmedAt.Time
	}
	return &rd, nil
}

func (r *PostgresRepository) ConfirmRedemption(ctx context.Context, id, collaboratorID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE redemptions_earnburn SET status = 'confirmed', confirmed_by = $2, confirmed_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, id, collaboratorID)
	return err
}

func (r *PostgresRepository) ExpirePendingRedemptions(ctx context.Context) (int, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE redemptions_earnburn SET status = 'expired'
		WHERE status = 'pending' AND expires_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("expire redemptions: %w", err)
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

// EnsureBalance creates a zero-balance record if one doesn't exist.
func (r *PostgresRepository) EnsureBalance(ctx context.Context, clientID, customerSisfiID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO balances_earnburn (client_id, customer_sisfi_id, balance)
		VALUES ($1, $2, 0)
		ON CONFLICT (client_id, customer_sisfi_id) DO NOTHING
	`, clientID, customerSisfiID)
	return err
}

// WithTx wraps operations in a transaction.
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

// AddPointsTx atomically creates a transaction and updates balance.
func (r *PostgresRepository) AddPointsTx(ctx context.Context, t *Transaction) (int, error) {
	var newBalance int
	err := r.WithTx(ctx, func(tx *sql.Tx) error {
		// Upsert balance
		err := tx.QueryRowContext(ctx, `
			INSERT INTO balances_earnburn (client_id, customer_sisfi_id, balance)
			VALUES ($1, $2, GREATEST(0, $3))
			ON CONFLICT (client_id, customer_sisfi_id) DO UPDATE
			SET balance = GREATEST(0, balances_earnburn.balance + $3), updated_at = NOW()
			RETURNING balance
		`, t.ClientID, t.CustomerSisfiID, t.Amount).Scan(&newBalance)
		if err != nil {
			return fmt.Errorf("upsert balance: %w", err)
		}

		t.BalanceAfter = newBalance

		// Create transaction record. receipt_hash is inserted as NULL when empty
		// so the partial unique index only guards confidently-hashed tickets.
		// A unique violation here rolls back the whole tx (balance included), so a
		// duplicate ticket never credits points.
		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions_earnburn
			(id, client_id, customer_sisfi_id, collaborator_id, type, amount, balance_after, invoice_url, description, manual_entry, correctable_until, receipt_data, receipt_hash, receipt_hash_fields, receipt_confident)
			VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12, NULLIF($13, ''), $14, $15)
		`, t.ID, t.ClientID, t.CustomerSisfiID, t.CollaboratorID, t.Type, t.Amount, t.BalanceAfter,
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

// nullableJSON returns nil (SQL NULL) for empty payloads so JSONB columns stay
// NULL rather than storing an empty string / "null" literal.
func nullableJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

// BurnPointsTx atomically deducts points and creates a redemption record.
func (r *PostgresRepository) BurnPointsTx(ctx context.Context, t *Transaction, rd *Redemption) error {
	return r.WithTx(ctx, func(tx *sql.Tx) error {
		// Deduct balance
		var newBalance int
		err := tx.QueryRowContext(ctx, `
			UPDATE balances_earnburn
			SET balance = balance + $3, updated_at = NOW()
			WHERE client_id = $1 AND customer_sisfi_id = $2 AND balance >= $4
			RETURNING balance
		`, t.ClientID, t.CustomerSisfiID, t.Amount, -t.Amount).Scan(&newBalance)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("insufficient balance")
			}
			return fmt.Errorf("deduct balance: %w", err)
		}

		t.BalanceAfter = newBalance

		// Transaction record
		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions_earnburn (id, client_id, customer_sisfi_id, type, amount, balance_after, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, t.ID, t.ClientID, t.CustomerSisfiID, t.Type, t.Amount, t.BalanceAfter, t.Description)
		if err != nil {
			return fmt.Errorf("create burn tx: %w", err)
		}

		// Redemption record
		_, err = tx.ExecContext(ctx, `
			INSERT INTO redemptions_earnburn (id, client_id, reward_id, customer_sisfi_id, code, status, points_spent, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, rd.ID, rd.ClientID, rd.RewardID, rd.CustomerSisfiID, rd.Code, rd.Status, rd.PointsSpent, rd.ExpiresAt)
		return err
	})
}

// --- Admin CRUD ---

func (r *PostgresRepository) ListPrograms(ctx context.Context, customerID string) ([]EarnBurnProgram, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, ec.points_ratio, cs.active, ec.expiry_days, ec.min_ticket_amount
		 FROM customer_sisfi cs
		 JOIN config_earnburn ec ON ec.customer_sisfi_id = cs.id
		 WHERE cs.customer_id = $1 AND cs.sisfi_id = 'earn_burn' AND cs.active = true
		 ORDER BY cs.created_at`,
		customerID)
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	defer rows.Close()

	var programs []EarnBurnProgram
	for rows.Next() {
		var p EarnBurnProgram
		var expiryDays sql.NullInt64
		var minTicket sql.NullFloat64
		if err := rows.Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &p.PointsRatio, &p.Active, &expiryDays, &minTicket); err != nil {
			return nil, fmt.Errorf("scan program: %w", err)
		}
		applyEarnConfig(&p, expiryDays, minTicket)
		programs = append(programs, p)
	}
	return programs, nil
}

func (r *PostgresRepository) GetCustomer(ctx context.Context, id string) (*Customer, error) {
	var c Customer
	var address, logoURL, desc, welcome sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, phone, address, logo_url, description, welcome_message, active
		 FROM customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Slug, &c.Phone, &address, &logoURL, &desc, &welcome, &c.Active)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	c.Address = address.String
	c.LogoURL = logoURL.String
	c.Description = desc.String
	c.WelcomeMessage = welcome.String
	return &c, nil
}

func (r *PostgresRepository) CreateCustomer(ctx context.Context, c *Customer) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO customers (name, slug, phone, address, logo_url, description, welcome_message)
		 VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, '')) RETURNING id`,
		c.Name, c.Slug, c.Phone, c.Address, c.LogoURL, c.Description, c.WelcomeMessage,
	).Scan(&c.ID)
}

func (r *PostgresRepository) UpdateCustomer(ctx context.Context, c *Customer) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE customers SET
		 name = COALESCE(NULLIF($2, ''), name),
		 phone = COALESCE(NULLIF($3, ''), phone),
		 address = COALESCE(NULLIF($4, ''), address),
		 logo_url = COALESCE(NULLIF($5, ''), logo_url),
		 description = COALESCE(NULLIF($6, ''), description),
		 welcome_message = COALESCE(NULLIF($7, ''), welcome_message),
		 active = COALESCE($8, active),
		 updated_at = NOW() WHERE id = $1`,
		c.ID, c.Name, c.Phone, c.Address, c.LogoURL, c.Description, c.WelcomeMessage, c.Active,
	)
	return err
}

func (r *PostgresRepository) CreateCollaborator(ctx context.Context, c *Collaborator) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO collaborators (customer_id, name, phone, hash_id) VALUES ($1, $2, $3, $4) RETURNING id`,
		c.CustomerID, c.Name, c.Phone, c.HashID,
	).Scan(&c.ID)
}

func (r *PostgresRepository) ListCollaborators(ctx context.Context, customerID string) ([]Collaborator, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, name, phone, hash_id, active FROM collaborators WHERE customer_id = $1 ORDER BY name`,
		customerID)
	if err != nil {
		return nil, fmt.Errorf("list collaborators: %w", err)
	}
	defer rows.Close()

	var collabs []Collaborator
	for rows.Next() {
		var c Collaborator
		if err := rows.Scan(&c.ID, &c.CustomerID, &c.Name, &c.Phone, &c.HashID, &c.Active); err != nil {
			return nil, fmt.Errorf("scan collaborator: %w", err)
		}
		collabs = append(collabs, c)
	}
	return collabs, nil
}

func (r *PostgresRepository) ListAllRewards(ctx context.Context, customerSisfiID string) ([]Reward, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, customer_sisfi_id, name, COALESCE(description, ''), points_cost, active FROM rewards_earnburn
		 WHERE customer_sisfi_id = $1 ORDER BY points_cost`, customerSisfiID)
	if err != nil {
		return nil, fmt.Errorf("list all rewards: %w", err)
	}
	defer rows.Close()

	var rewards []Reward
	for rows.Next() {
		var rw Reward
		if err := rows.Scan(&rw.ID, &rw.CustomerID, &rw.CustomerSisfiID, &rw.Name, &rw.Description, &rw.PointsCost, &rw.Active); err != nil {
			return nil, fmt.Errorf("scan reward: %w", err)
		}
		rewards = append(rewards, rw)
	}
	return rewards, nil
}

func (r *PostgresRepository) CreateRewardAdmin(ctx context.Context, customerSisfiID string, rw *Reward) error {
	var customerID string
	err := r.db.QueryRowContext(ctx,
		`SELECT customer_id FROM customer_sisfi WHERE id = $1`, customerSisfiID,
	).Scan(&customerID)
	if err != nil {
		return fmt.Errorf("get customer_sisfi for reward: %w", err)
	}

	return r.db.QueryRowContext(ctx,
		`INSERT INTO rewards_earnburn (customer_id, customer_sisfi_id, name, description, points_cost)
		 VALUES ($1, $2, $3, NULLIF($4, ''), $5) RETURNING id`,
		customerID, customerSisfiID, rw.Name, rw.Description, rw.PointsCost,
	).Scan(&rw.ID)
}

func (r *PostgresRepository) UpdateRewardAdmin(ctx context.Context, rw *Reward) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE rewards_earnburn SET name = COALESCE(NULLIF($2, ''), name),
		 description = COALESCE(NULLIF($3, ''), description),
		 points_cost = CASE WHEN $4 > 0 THEN $4 ELSE points_cost END,
		 active = COALESCE($5, active), updated_at = NOW() WHERE id = $1`,
		rw.ID, rw.Name, rw.Description, rw.PointsCost, rw.Active,
	)
	return err
}

func (r *PostgresRepository) ListFeedback(ctx context.Context, customerID string) ([]FeedbackEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT f.id, f.message, COALESCE(c.name, ''), f.created_at FROM feedback f
		 LEFT JOIN clients c ON c.id = f.client_id
		 WHERE f.customer_id = $1 ORDER BY f.created_at DESC LIMIT 50`,
		customerID)
	if err != nil {
		return nil, fmt.Errorf("list feedback: %w", err)
	}
	defer rows.Close()

	var entries []FeedbackEntry
	for rows.Next() {
		var e FeedbackEntry
		if err := rows.Scan(&e.ID, &e.Message, &e.ClientName, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan feedback: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *PostgresRepository) ListClients(ctx context.Context, customerID string) ([]Client, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, name, phone, created_at FROM clients WHERE customer_id = $1 ORDER BY created_at DESC`,
		customerID)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		var name sql.NullString
		if err := rows.Scan(&c.ID, &c.CustomerID, &name, &c.Phone, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}
		c.Name = name.String
		clients = append(clients, c)
	}
	return clients, nil
}

func (r *PostgresRepository) RegisterClient(ctx context.Context, customerID, phone string) error {
	hash := generateClientHash()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO clients (customer_id, phone, hash) VALUES ($1, $2, $3) ON CONFLICT (customer_id, phone) DO NOTHING`,
		customerID, phone, hash,
	)
	return err
}

func generateClientHash() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// AdjustPointsTx atomically adjusts points for a correction.
func (r *PostgresRepository) AdjustPointsTx(ctx context.Context, t *Transaction) (int, error) {
	var newBalance int
	err := r.WithTx(ctx, func(tx *sql.Tx) error {
		// Verify original transaction is still correctable
		var correctableUntil time.Time
		err := tx.QueryRowContext(ctx,
			`SELECT correctable_until FROM transactions_earnburn WHERE id = $1 AND correctable_until > NOW()`,
			t.Description, // description stores the original tx ID for adjustments
		).Scan(&correctableUntil)
		if err != nil {
			return fmt.Errorf("transaction not correctable: %w", err)
		}

		// Update balance
		err = tx.QueryRowContext(ctx, `
			UPDATE balances_earnburn
			SET balance = GREATEST(0, balance + $3), updated_at = NOW()
			WHERE client_id = $1 AND customer_sisfi_id = $2
			RETURNING balance
		`, t.ClientID, t.CustomerSisfiID, t.Amount).Scan(&newBalance)
		if err != nil {
			return fmt.Errorf("adjust balance: %w", err)
		}

		t.BalanceAfter = newBalance

		// Create adjustment transaction
		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions_earnburn
			(id, client_id, customer_sisfi_id, collaborator_id, type, amount, balance_after, correction_reason, correction_evidence_url)
			VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''))
		`, t.ID, t.ClientID, t.CustomerSisfiID, t.CollaboratorID, t.Type, t.Amount, t.BalanceAfter,
			t.CorrectionReason, t.CorrectionEvidenceURL)
		return err
	})
	return newBalance, err
}
