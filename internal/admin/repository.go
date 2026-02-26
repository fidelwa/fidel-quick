package admin

import (
	"database/sql"
	"errors"

	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

type Repository interface {
	GetByEmail(email string) (*Admin, error)
	Create(admin *Admin) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetByEmail(email string) (*Admin, error) {
	var a Admin
	err := r.db.QueryRow(
		`SELECT id, customer_id, email, password_hash, active, created_at, updated_at
		 FROM admins WHERE email = $1 AND active = true`,
		email,
	).Scan(&a.ID, &a.CustomerID, &a.Email, &a.PasswordHash, &a.Active, &a.CreatedAt, &a.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("admin not found", nil)
	}
	if err != nil {
		return nil, apperror.Internal("failed to query admin", err)
	}
	return &a, nil
}

func (r *PostgresRepository) Create(admin *Admin) error {
	err := r.db.QueryRow(
		`INSERT INTO admins (customer_id, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		admin.CustomerID, admin.Email, admin.PasswordHash,
	).Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return apperror.Conflict("email already registered", nil)
		}
		return apperror.Internal("failed to create admin", err)
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) == false) &&
		(len(err.Error()) > 0 && contains(err.Error(), "duplicate key"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
