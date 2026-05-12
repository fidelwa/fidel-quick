package resolver

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/lib/pq"
	"github.com/theluisbolivar/fidel-quick/internal/phone"
	"github.com/theluisbolivar/fidel-quick/internal/session"
)

// Repository abstracts database access for business/role resolution, landing, and auto-registration.
type Repository interface {
	GetActiveCustomerByID(ctx context.Context, id string) (name string, err error)
	// FindActiveCustomersByName busca customers activos cuyo nombre coincide
	// (case-insensitive, exact match con LOWER trim). Se usa para resolver
	// deeplinks que usan el nombre en lugar de UUID, ej. "Quiero unirme a
	// Santas Conchas". Devuelve [] si ninguno coincide; el caller maneja
	// 1=auto-resolve, >1=multi-business prompt.
	FindActiveCustomersByName(ctx context.Context, name string) ([]session.SelectionOption, error)
	UserExistsInBusiness(ctx context.Context, phone, customerID string) (bool, error)
	FindBusinessesByPhone(ctx context.Context, phone string) ([]session.SelectionOption, error)
	FindCollaborator(ctx context.Context, phone, customerID string) (id string, err error)
	FindClient(ctx context.Context, phone, customerID string) (id string, err error)
	RegisterClient(ctx context.Context, customerID, phone string) error
	GetCustomerBySlug(ctx context.Context, slug string) (id, name, slugOut, logoURL, description, welcomeMessage string, err error)
	GetActiveProgramTypes(ctx context.Context, customerID string) ([]string, error)
}

// PostgresRepository implements Repository.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetActiveCustomerByID(ctx context.Context, id string) (string, error) {
	var name string
	err := r.db.QueryRowContext(ctx,
		`SELECT name FROM customers WHERE id = $1 AND active = true`, id,
	).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("query customer by id: %w", err)
	}
	return name, nil
}

// FindActiveCustomersByName: case-insensitive exact match con trim. Sin LIKE
// para evitar matches accidentales — la idea es que el deeplink ponga el
// nombre exacto que el wizard registró.
func (r *PostgresRepository) FindActiveCustomersByName(ctx context.Context, name string) ([]session.SelectionOption, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name FROM customers
		WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) AND active = true
		ORDER BY created_at ASC
	`, name)
	if err != nil {
		return nil, fmt.Errorf("query customers by name: %w", err)
	}
	defer rows.Close()

	var out []session.SelectionOption
	for rows.Next() {
		var opt session.SelectionOption
		if err := rows.Scan(&opt.CustomerID, &opt.Name); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		out = append(out, opt)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UserExistsInBusiness(ctx context.Context, ph, customerID string) (bool, error) {
	variants := phone.Variants(ph)
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM clients WHERE phone = ANY($1) AND customer_id = $2
			UNION
			SELECT 1 FROM collaborators WHERE phone = ANY($1) AND customer_id = $2
		)`, pq.Array(variants), customerID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check existing user: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) FindBusinessesByPhone(ctx context.Context, ph string) ([]session.SelectionOption, error) {
	variants := phone.Variants(ph)
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT c.id, c.name FROM customers c
		INNER JOIN collaborators co ON co.customer_id = c.id
		WHERE co.phone = ANY($1) AND co.active = true AND c.active = true
		UNION
		SELECT DISTINCT c.id, c.name FROM customers c
		INNER JOIN clients cl ON cl.customer_id = c.id
		WHERE cl.phone = ANY($1) AND c.active = true
	`, pq.Array(variants))
	if err != nil {
		return nil, fmt.Errorf("query businesses by phone: %w", err)
	}
	defer rows.Close()

	var options []session.SelectionOption
	for rows.Next() {
		var opt session.SelectionOption
		if err := rows.Scan(&opt.CustomerID, &opt.Name); err != nil {
			return nil, fmt.Errorf("scan business: %w", err)
		}
		options = append(options, opt)
	}
	return options, nil
}

func (r *PostgresRepository) FindCollaborator(ctx context.Context, ph, customerID string) (string, error) {
	variants := phone.Variants(ph)
	var id string
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM collaborators WHERE phone = ANY($1) AND customer_id = $2 AND active = true`,
		pq.Array(variants), customerID,
	).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("query collaborator: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) FindClient(ctx context.Context, ph, customerID string) (string, error) {
	variants := phone.Variants(ph)
	var id string
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM clients WHERE phone = ANY($1) AND customer_id = $2`,
		pq.Array(variants), customerID,
	).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("query client: %w", err)
	}
	return id, nil
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

func (r *PostgresRepository) GetCustomerBySlug(ctx context.Context, slug string) (string, string, string, string, string, string, error) {
	var id, name, slugOut string
	var logoURL, description, welcomeMessage sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, COALESCE(logo_url, ''), COALESCE(description, ''), COALESCE(welcome_message, '')
		 FROM customers WHERE slug = $1 AND active = true`, slug,
	).Scan(&id, &name, &slugOut, &logoURL, &description, &welcomeMessage)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	return id, name, slugOut, logoURL.String, description.String, welcomeMessage.String, nil
}

func (r *PostgresRepository) GetActiveProgramTypes(ctx context.Context, customerID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT cs.sisfi_id FROM customer_sisfi cs
		JOIN sisfi s ON s.id = cs.sisfi_id AND s.active = true
		WHERE cs.customer_id = $1 AND cs.active = true
	`, customerID)
	if err != nil {
		return nil, fmt.Errorf("query active program types: %w", err)
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("scan program type: %w", err)
		}
		modules = append(modules, m)
	}
	return modules, nil
}
