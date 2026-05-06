package admin

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

type Repository interface {
	GetByEmail(email string) (*Admin, error)
	GetByID(id string) (*Admin, error)
	GetByGoogleSub(sub string) (*Admin, error)
	Create(admin *Admin) error
	CreateCustomer(name, slug, phone, description string) (customerID string, err error)
	SlugExists(slug string) (bool, error)
	LinkGoogle(adminID, sub, email string) error
	UnlinkGoogle(adminID string) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const adminColumns = `id, customer_id, email, password_hash, google_sub, google_email, active, created_at, updated_at`

func scanAdmin(row interface{ Scan(...any) error }) (*Admin, error) {
	var a Admin
	var sub, gemail sql.NullString
	if err := row.Scan(&a.ID, &a.CustomerID, &a.Email, &a.PasswordHash, &sub, &gemail, &a.Active, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	if sub.Valid {
		v := sub.String
		a.GoogleSub = &v
	}
	if gemail.Valid {
		v := gemail.String
		a.GoogleEmail = &v
	}
	return &a, nil
}

func (r *PostgresRepository) GetByEmail(email string) (*Admin, error) {
	row := r.db.QueryRow(
		`SELECT `+adminColumns+` FROM admins WHERE email = $1 AND active = true`,
		email,
	)
	a, err := scanAdmin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("admin not found", nil)
	}
	if err != nil {
		return nil, apperror.Internal("failed to query admin", err)
	}
	return a, nil
}

func (r *PostgresRepository) GetByID(id string) (*Admin, error) {
	row := r.db.QueryRow(
		`SELECT `+adminColumns+` FROM admins WHERE id = $1 AND active = true`,
		id,
	)
	a, err := scanAdmin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("admin not found", nil)
	}
	if err != nil {
		return nil, apperror.Internal("failed to query admin", err)
	}
	return a, nil
}

func (r *PostgresRepository) GetByGoogleSub(sub string) (*Admin, error) {
	row := r.db.QueryRow(
		`SELECT `+adminColumns+` FROM admins WHERE google_sub = $1 AND active = true`,
		sub,
	)
	a, err := scanAdmin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("admin not found", nil)
	}
	if err != nil {
		return nil, apperror.Internal("failed to query admin", err)
	}
	return a, nil
}

func (r *PostgresRepository) Create(admin *Admin) error {
	err := r.db.QueryRow(
		`INSERT INTO admins (customer_id, email, password_hash, google_sub, google_email)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		admin.CustomerID, admin.Email, admin.PasswordHash,
		nullableString(admin.GoogleSub), nullableString(admin.GoogleEmail),
	).Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return apperror.Conflict("email already registered", nil)
		}
		return apperror.Internal("failed to create admin", err)
	}
	return nil
}

func (r *PostgresRepository) CreateCustomer(name, slug, phone, description string) (string, error) {
	var id string
	err := r.db.QueryRow(
		`INSERT INTO customers (name, slug, phone, description)
		 VALUES ($1, $2, $3, NULLIF($4, '')) RETURNING id`,
		name, slug, phone, description,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create customer: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) SlugExists(slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM customers WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) LinkGoogle(adminID, sub, email string) error {
	res, err := r.db.Exec(
		`UPDATE admins SET google_sub = $1, google_email = $2, updated_at = NOW()
		 WHERE id = $3`,
		sub, email, adminID,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return apperror.Conflict("google account already linked to another admin", nil)
		}
		return apperror.Internal("failed to link google", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return apperror.NotFound("admin not found", nil)
	}
	return nil
}

func (r *PostgresRepository) UnlinkGoogle(adminID string) error {
	res, err := r.db.Exec(
		`UPDATE admins SET google_sub = NULL, google_email = NULL, updated_at = NOW()
		 WHERE id = $1`,
		adminID,
	)
	if err != nil {
		return apperror.Internal("failed to unlink google", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return apperror.NotFound("admin not found", nil)
	}
	return nil
}

func nullableString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
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
