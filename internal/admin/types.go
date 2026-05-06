package admin

import "time"

type Admin struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customer_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	GoogleSub    *string   `json:"-"`
	GoogleEmail  *string   `json:"google_email,omitempty"`
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

type OnboardingRequest struct {
	Name          string `json:"name" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	Description   string `json:"description"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
	AdminPassword string `json:"admin_password" binding:"required,min=8"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	Admin AdminSummary `json:"admin"`
}

type AdminSummary struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	CustomerID  string  `json:"customer_id"`
	GoogleEmail *string `json:"google_email,omitempty"`
}

type GoogleOnboardingRequest struct {
	GoogleToken string `json:"google_token" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	Description string `json:"description"`
}

type GoogleLoginRequest struct {
	GoogleToken string `json:"google_token" binding:"required"`
}

// LinkGoogleRequest binds the body of POST /api/v1/auth/link/google.
type LinkGoogleRequest struct {
	GoogleToken string `json:"google_token" binding:"required"`
}
