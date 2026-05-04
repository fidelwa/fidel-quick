package sisfi

import "database/sql"

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListSisfi returns all active loyalty system types.
func (r *Repository) ListSisfi() ([]Sisfi, error) {
	rows, err := r.db.Query(
		`SELECT id, name, COALESCE(description, ''), active, created_at FROM sisfi WHERE active = true ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Sisfi
	for rows.Next() {
		var s Sisfi
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Active, &s.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, nil
}

// ListByCustomer returns active loyalty systems for a customer.
func (r *Repository) ListByCustomer(customerID string) ([]CustomerSisfi, error) {
	rows, err := r.db.Query(
		`SELECT cs.id, cs.customer_id, cs.sisfi_id, cs.name, cs.active, cs.created_at, cs.updated_at
		 FROM customer_sisfi cs
		 JOIN sisfi s ON s.id = cs.sisfi_id AND s.active = true
		 WHERE cs.customer_id = $1 AND cs.active = true
		 ORDER BY cs.sisfi_id`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CustomerSisfi
	for rows.Next() {
		var cs CustomerSisfi
		if err := rows.Scan(&cs.ID, &cs.CustomerID, &cs.SisfiID, &cs.Name, &cs.Active, &cs.CreatedAt, &cs.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, cs)
	}
	return items, nil
}

// GetByID returns a customer_sisfi by its UUID.
func (r *Repository) GetByID(id string) (*CustomerSisfi, error) {
	var cs CustomerSisfi
	err := r.db.QueryRow(
		`SELECT id, customer_id, sisfi_id, name, active, created_at, updated_at
		 FROM customer_sisfi WHERE id = $1`, id,
	).Scan(&cs.ID, &cs.CustomerID, &cs.SisfiID, &cs.Name, &cs.Active, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// Create inserts a new customer_sisfi record.
func (r *Repository) Create(cs *CustomerSisfi) error {
	return r.db.QueryRow(
		`INSERT INTO customer_sisfi (customer_id, sisfi_id, name)
		 VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		cs.CustomerID, cs.SisfiID, cs.Name,
	).Scan(&cs.ID, &cs.CreatedAt, &cs.UpdatedAt)
}

// Update modifies a customer_sisfi record.
func (r *Repository) Update(cs *CustomerSisfi) error {
	_, err := r.db.Exec(
		`UPDATE customer_sisfi SET name = $2, active = $3, updated_at = now() WHERE id = $1`,
		cs.ID, cs.Name, cs.Active)
	return err
}

// GetActiveModules returns the list of active sisfi module names for a customer.
func (r *Repository) GetActiveModules(customerID string) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT cs.sisfi_id FROM customer_sisfi cs
		 JOIN sisfi s ON s.id = cs.sisfi_id AND s.active = true
		 WHERE cs.customer_id = $1 AND cs.active = true`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}
	return modules, nil
}
