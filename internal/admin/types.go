package admin

import "time"

type Admin struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customer_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	GoogleSub    *string   `json:"-"`
	GoogleEmail  *string   `json:"google_email,omitempty"`
	FullName     *string   `json:"full_name,omitempty"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	Locale       *string   `json:"locale,omitempty"`
	HostedDomain *string   `json:"hosted_domain,omitempty"`
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
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	CustomerID   string  `json:"customer_id"`
	GoogleEmail  *string `json:"google_email,omitempty"`
	FullName     *string `json:"full_name,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	Locale       *string `json:"locale,omitempty"`
	HostedDomain *string `json:"hosted_domain,omitempty"`
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

// ForgotPasswordRequest binds the body of POST /api/v1/auth/forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest binds the body of POST /api/v1/auth/reset-password.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// PasswordResetToken mirrors a row in password_reset_tokens. Only token_hash
// is persisted — the plaintext token lives only in the emailed link.
type PasswordResetToken struct {
	ID        string
	AdminID   string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
