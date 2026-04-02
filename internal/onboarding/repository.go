package onboarding

import (
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByCustomerID(customerID string) (*Onboarding, error) {
	var o Onboarding
	var completedAt sql.NullTime
	err := r.db.QueryRow(
		`SELECT id, customer_id, current_step, completed, completed_at, created_at, updated_at
		 FROM onboarding WHERE customer_id = $1`,
		customerID,
	).Scan(&o.ID, &o.CustomerID, &o.CurrentStep, &o.Completed, &completedAt, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		o.CompletedAt = &completedAt.Time
	}
	return &o, nil
}

func (r *Repository) Upsert(customerID string, step int) (*Onboarding, error) {
	var o Onboarding
	var completedAt sql.NullTime
	err := r.db.QueryRow(
		`INSERT INTO onboarding (customer_id, current_step)
		 VALUES ($1, $2)
		 ON CONFLICT (customer_id)
		 DO UPDATE SET current_step = GREATEST(onboarding.current_step, $2), updated_at = now()
		 RETURNING id, customer_id, current_step, completed, completed_at, created_at, updated_at`,
		customerID, step,
	).Scan(&o.ID, &o.CustomerID, &o.CurrentStep, &o.Completed, &completedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		o.CompletedAt = &completedAt.Time
	}
	return &o, nil
}

func (r *Repository) Complete(customerID string) (*Onboarding, error) {
	var o Onboarding
	now := time.Now()
	err := r.db.QueryRow(
		`UPDATE onboarding SET completed = true, completed_at = $2, current_step = 4, updated_at = $2
		 WHERE customer_id = $1
		 RETURNING id, customer_id, current_step, completed, completed_at, created_at, updated_at`,
		customerID, now,
	).Scan(&o.ID, &o.CustomerID, &o.CurrentStep, &o.Completed, &o.CompletedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
