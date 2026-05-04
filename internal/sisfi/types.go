package sisfi

import "time"

// Sisfi represents a loyalty system type in the platform catalog.
type Sisfi struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

// CustomerSisfi represents an active loyalty system for a specific customer.
type CustomerSisfi struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	SisfiID    string    `json:"sisfi_id"`
	Name       string    `json:"name"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
