package cashback

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repository interface {
	GetProgram(ctx context.Context, customerID string) (*CashbackProgram, error)
	GetProgramByID(ctx context.Context, customerSisfiID string) (*CashbackProgram, error)
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

func (r *PostgresRepository) GetProgram(ctx context.Context, customerID string) (*CashbackProgram, error) {
	var p CashbackProgram
	err := r.db.QueryRowContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, cc.cashback_rate, cs.active
		 FROM customer_sisfi cs
		 JOIN config_cashback cc ON cc.customer_sisfi_id = cs.id
		 WHERE cs.customer_id = $1 AND cs.sisfi_id = 'cashback' AND cs.active = true`, customerID,
	).Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &p.CashbackRate, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get cashback program: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) GetProgramByID(ctx context.Context, customerSisfiID string) (*CashbackProgram, error) {
	var p CashbackProgram
	err := r.db.QueryRowContext(ctx,
		`SELECT cs.id, cs.customer_id, cs.name, cc.cashback_rate, cs.active
		 FROM customer_sisfi cs
		 JOIN config_cashback cc ON cc.customer_sisfi_id = cs.id
		 WHERE cs.id = $1`, customerSisfiID,
	).Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &p.CashbackRate, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get cashback program by id: %w", err)
	}
	return &p, nil
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
func (r *PostgresRepository) AddCashbackTx(ctx context.Context, t *CashbackTransaction) (float64, error) {
	var newBalance float64
	err := r.WithTx(ctx, func(tx *sql.Tx) error {
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

		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions_cashback
			(id, client_id, customer_sisfi_id, collaborator_id, type, amount, purchase_amount, balance_after, invoice_url, description, manual_entry, correctable_until)
			VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6::NUMERIC, $7::NUMERIC, $8::NUMERIC, NULLIF($9, ''), NULLIF($10, ''), $11, $12)
		`, t.ID, t.ClientID, t.CustomerSisfiID, t.CollaboratorID, t.Type, t.Amount, t.PurchaseAmount, t.BalanceAfter,
			t.InvoiceURL, t.Description, t.ManualEntry, t.CorrectableUntil)
		return err
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
		`SELECT cs.id, cs.customer_id, cs.name, cc.cashback_rate, cs.active
		 FROM customer_sisfi cs
		 JOIN config_cashback cc ON cc.customer_sisfi_id = cs.id
		 WHERE cs.customer_id = $1 AND cs.sisfi_id = 'cashback'
		 ORDER BY cs.created_at`,
		customerID)
	if err != nil {
		return nil, fmt.Errorf("list cashback programs: %w", err)
	}
	defer rows.Close()

	var programs []CashbackProgram
	for rows.Next() {
		var p CashbackProgram
		if err := rows.Scan(&p.CustomerSisfiID, &p.CustomerID, &p.Name, &p.CashbackRate, &p.Active); err != nil {
			return nil, fmt.Errorf("scan cashback program: %w", err)
		}
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
