package pushcard

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/theluisbolivar/fidel-quick/internal/phone"
)

// Repository abstracts pushcard persistence so the service can be unit-tested with mocks.
type Repository interface {
	GetConfig(ctx context.Context, customerID string) (*Config, error)
	GetConfigByID(ctx context.Context, customerSisfiID string) (*Config, error)
	UpsertConfig(ctx context.Context, cfg *Config) error

	GetOpenCard(ctx context.Context, customerSisfiID, clientID string) (*Card, error)
	GetCard(ctx context.Context, cardID string) (*Card, error)
	OpenCard(ctx context.Context, customerSisfiID, clientID string) (*Card, error)
	CountStamps(ctx context.Context, cardID string) (int, error)
	AddStamp(ctx context.Context, stamp *Stamp) error
	CompleteCard(ctx context.Context, cardID string) error
	MarkRedeemed(ctx context.Context, cardID string) error

	// LastStampByCollaborator returns the most recent stamp by this collaborator
	// across any card, used for the 2h correction window.
	LastStampByCollaborator(ctx context.Context, collaboratorID string, since time.Duration) (*Stamp, error)
	DeleteStamp(ctx context.Context, stampID string) error

	ListCardsByCustomer(ctx context.Context, customerSisfiID, status string, limit int) ([]Card, error)

	// FindClientIDByPhone resolves a phone number (in any common variant)
	// to a client_id within the given customer scope.
	FindClientIDByPhone(ctx context.Context, customerID, phoneNumber string) (string, error)
}

// PostgresRepository is the concrete Repository backed by Postgres.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetConfig(ctx context.Context, customerID string) (*Config, error) {
	var c Config
	var rewardID sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT cs.id, cs.customer_id, cs.name, pc.card_slots, pc.reward_on_complete, cs.active
		FROM customer_sisfi cs
		JOIN pushcard_config pc ON pc.customer_sisfi_id = cs.id
		WHERE cs.customer_id = $1 AND cs.sisfi_id = 'pushcard' AND cs.active = true
	`, customerID).Scan(&c.CustomerSisfiID, &c.CustomerID, &c.Name, &c.CardSlots, &rewardID, &c.Active)
	if err != nil {
		return nil, fmt.Errorf("get pushcard config: %w", err)
	}
	c.RewardOnComplete = rewardID.String
	return &c, nil
}

func (r *PostgresRepository) GetConfigByID(ctx context.Context, customerSisfiID string) (*Config, error) {
	var c Config
	var rewardID sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT cs.id, cs.customer_id, cs.name, pc.card_slots, pc.reward_on_complete, cs.active
		FROM customer_sisfi cs
		JOIN pushcard_config pc ON pc.customer_sisfi_id = cs.id
		WHERE cs.id = $1
	`, customerSisfiID).Scan(&c.CustomerSisfiID, &c.CustomerID, &c.Name, &c.CardSlots, &rewardID, &c.Active)
	if err != nil {
		return nil, fmt.Errorf("get pushcard config by id: %w", err)
	}
	c.RewardOnComplete = rewardID.String
	return &c, nil
}

func (r *PostgresRepository) UpsertConfig(ctx context.Context, cfg *Config) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO pushcard_config (customer_sisfi_id, card_slots, reward_on_complete)
		VALUES ($1, $2, NULLIF($3, ''))
		ON CONFLICT (customer_sisfi_id) DO UPDATE
		SET card_slots = EXCLUDED.card_slots,
		    reward_on_complete = EXCLUDED.reward_on_complete,
		    updated_at = NOW()
	`, cfg.CustomerSisfiID, cfg.CardSlots, cfg.RewardOnComplete)
	if err != nil {
		return fmt.Errorf("upsert pushcard config: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetOpenCard(ctx context.Context, customerSisfiID, clientID string) (*Card, error) {
	c, err := r.queryCard(ctx, `
		SELECT id, customer_sisfi_id, client_id, status, completed_at, created_at, updated_at
		FROM pushcard_cards
		WHERE customer_sisfi_id = $1 AND client_id = $2 AND status = 'open'
		LIMIT 1
	`, customerSisfiID, clientID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *PostgresRepository) GetCard(ctx context.Context, cardID string) (*Card, error) {
	return r.queryCard(ctx, `
		SELECT id, customer_sisfi_id, client_id, status, completed_at, created_at, updated_at
		FROM pushcard_cards WHERE id = $1
	`, cardID)
}

func (r *PostgresRepository) queryCard(ctx context.Context, query string, args ...interface{}) (*Card, error) {
	var c Card
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&c.ID, &c.CustomerSisfiID, &c.ClientID, &c.Status,
		&completedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		c.CompletedAt = &completedAt.Time
	}
	return &c, nil
}

func (r *PostgresRepository) OpenCard(ctx context.Context, customerSisfiID, clientID string) (*Card, error) {
	var c Card
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO pushcard_cards (customer_sisfi_id, client_id, status)
		VALUES ($1, $2, 'open')
		RETURNING id, customer_sisfi_id, client_id, status, created_at, updated_at
	`, customerSisfiID, clientID).Scan(
		&c.ID, &c.CustomerSisfiID, &c.ClientID, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("open pushcard: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) CountStamps(ctx context.Context, cardID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pushcard_stamps WHERE card_id = $1`, cardID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count stamps: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) AddStamp(ctx context.Context, stamp *Stamp) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO pushcard_stamps (id, card_id, collaborator_id)
		VALUES ($1, $2, $3)
	`, stamp.ID, stamp.CardID, stamp.CollaboratorID)
	if err != nil {
		return fmt.Errorf("add stamp: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CompleteCard(ctx context.Context, cardID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE pushcard_cards
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'open'
	`, cardID)
	if err != nil {
		return fmt.Errorf("complete card: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkRedeemed(ctx context.Context, cardID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE pushcard_cards
		SET status = 'redeemed', updated_at = NOW()
		WHERE id = $1 AND status = 'completed'
	`, cardID)
	if err != nil {
		return fmt.Errorf("mark redeemed: %w", err)
	}
	return nil
}

func (r *PostgresRepository) LastStampByCollaborator(ctx context.Context, collaboratorID string, within time.Duration) (*Stamp, error) {
	cutoff := time.Now().Add(-within)
	var s Stamp
	err := r.db.QueryRowContext(ctx, `
		SELECT id, card_id, collaborator_id, created_at
		FROM pushcard_stamps
		WHERE collaborator_id = $1 AND created_at >= $2
		ORDER BY created_at DESC
		LIMIT 1
	`, collaboratorID, cutoff).Scan(&s.ID, &s.CardID, &s.CollaboratorID, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("last stamp: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) DeleteStamp(ctx context.Context, stampID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM pushcard_stamps WHERE id = $1`, stampID)
	if err != nil {
		return fmt.Errorf("delete stamp: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindClientIDByPhone(ctx context.Context, customerID, phoneNumber string) (string, error) {
	variants := phone.Variants(phoneNumber)
	if len(variants) == 0 {
		return "", fmt.Errorf("teléfono inválido")
	}
	var id string
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM clients WHERE phone = ANY($1) AND customer_id = $2`,
		pq.Array(variants), customerID,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find client by phone: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) ListCardsByCustomer(ctx context.Context, customerSisfiID, status string, limit int) ([]Card, error) {
	args := []interface{}{customerSisfiID, limit}
	query := `
		SELECT id, customer_sisfi_id, client_id, status, completed_at, created_at, updated_at
		FROM pushcard_cards
		WHERE customer_sisfi_id = $1
	`
	if status != "" {
		query += " AND status = $3"
		args = append(args, status)
	}
	query += " ORDER BY updated_at DESC LIMIT $2"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list cards: %w", err)
	}
	defer rows.Close()

	var cards []Card
	for rows.Next() {
		var c Card
		var completedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.CustomerSisfiID, &c.ClientID, &c.Status,
			&completedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan card: %w", err)
		}
		if completedAt.Valid {
			c.CompletedAt = &completedAt.Time
		}
		cards = append(cards, c)
	}
	return cards, nil
}
