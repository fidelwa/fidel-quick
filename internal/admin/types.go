package admin

import "time"

type Admin struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customer_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RegisterRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	CustomerID string `json:"customer_id" binding:"required,uuid"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	Admin AdminSummary `json:"admin"`
}

type AdminSummary struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	CustomerID string `json:"customer_id"`
}
