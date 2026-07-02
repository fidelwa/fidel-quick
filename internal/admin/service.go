package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"github.com/theluisbolivar/fidel-quick/internal/platform/email"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"
)

// resetTokenTTL is how long a password reset link stays valid.
const resetTokenTTL = time.Hour

type Service struct {
	repo      Repository
	jwtSecret []byte
	verifier  GoogleVerifier

	// Password reset (FID-16). Configured via WithPasswordReset; when unset
	// the forgot/reset endpoints return a "not configured" error.
	emailSender email.Sender
	appURL      string
	forgotLimit RateLimiter // keyed by email
	resetLimit  RateLimiter // keyed by client IP
	log         *slog.Logger
}

// NewService wires the repository, JWT signing secret, and Google verifier.
// Pass a nil verifier to disable Google login/signup/linking — the corresponding
// service methods will then return a "google login not configured" error.
func NewService(repo Repository, jwtSecret string, verifier GoogleVerifier) *Service {
	return &Service{
		repo:        repo,
		jwtSecret:   []byte(jwtSecret),
		verifier:    verifier,
		forgotLimit: noopRateLimiter{},
		resetLimit:  noopRateLimiter{},
		log:         slog.Default(),
	}
}

// WithPasswordReset enables the forgot/reset-password flow. appURL is the
// public admin base URL used to build the reset link
// (<appURL>/reset-password?token=...). A nil logger falls back to slog.Default.
func (s *Service) WithPasswordReset(sender email.Sender, appURL string, log *slog.Logger) *Service {
	s.emailSender = sender
	s.appURL = strings.TrimRight(appURL, "/")
	s.forgotLimit = NewMemoryRateLimiter(5, time.Hour)
	s.resetLimit = NewMemoryRateLimiter(10, time.Hour)
	if log != nil {
		s.log = log
	}
	return s
}

func (s *Service) Login(email, password string) (*AuthResponse, error) {
	admin, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return s.makeAuthResponse(admin)
}

// ForgotPassword issues a password reset token for the given email and emails
// the reset link. To avoid leaking which emails are registered, it NEVER
// reports whether the email exists — callers should always return 200. A
// non-nil error means an infrastructure failure (rate limit, config), not
// "email unknown".
//
// The plaintext token (32 random bytes, base64url) travels only in the link;
// only its SHA-256 hash is persisted.
func (s *Service) ForgotPassword(ctx context.Context, emailAddr string) error {
	if s.emailSender == nil {
		return apperror.Internal("password reset not configured", nil)
	}

	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))
	if !s.forgotLimit.Allow(emailAddr) {
		return apperror.TooManyRequests("demasiadas solicitudes, intenta más tarde", nil)
	}

	admin, err := s.repo.GetByEmail(emailAddr)
	if err != nil {
		// Unknown email (or lookup miss): swallow so we don't enumerate.
		// Only surface unexpected internal errors.
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.HTTPStatus == 404 {
			s.log.Info("forgot-password for unknown email", "email", emailAddr)
			return nil
		}
		return err
	}

	plain, hash, err := generateResetToken()
	if err != nil {
		return apperror.Internal("failed to generate reset token", err)
	}

	if err := s.repo.CreatePasswordResetToken(admin.ID, hash, time.Now().Add(resetTokenTTL)); err != nil {
		return err
	}

	link := s.appURL + "/reset-password?token=" + plain
	body := fmt.Sprintf(
		"Recibimos una solicitud para restablecer tu contraseña.\n\n"+
			"Abre este enlace para elegir una nueva (válido por 1 hora):\n%s\n\n"+
			"Si no fuiste tú, ignora este correo.",
		link,
	)
	if err := s.emailSender.Send(ctx, email.Message{
		To:      admin.Email,
		Subject: "Restablece tu contraseña de Fidel",
		Body:    body,
	}); err != nil {
		return apperror.Internal("failed to send reset email", err)
	}

	return nil
}

// ResetPassword validates a reset token and, if valid (exists, not expired,
// not used), sets the admin's new password and invalidates outstanding tokens
// — all atomically. clientIP is used only for rate limiting.
func (s *Service) ResetPassword(clientIP, token, newPassword string) error {
	if s.emailSender == nil {
		return apperror.Internal("password reset not configured", nil)
	}
	if !s.resetLimit.Allow(clientIP) {
		return apperror.TooManyRequests("demasiadas solicitudes, intenta más tarde", nil)
	}
	if len(newPassword) < 8 {
		return apperror.BadRequest("la contraseña debe tener al menos 8 caracteres", nil)
	}

	hash := hashResetToken(token)
	row, err := s.repo.GetPasswordResetToken(hash)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.HTTPStatus == 404 {
			return apperror.BadRequest("token inválido o expirado", nil)
		}
		return err
	}
	if row.UsedAt != nil {
		return apperror.BadRequest("token inválido o expirado", nil)
	}
	if time.Now().After(row.ExpiresAt) {
		return apperror.BadRequest("token inválido o expirado", nil)
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.Internal("failed to hash password", err)
	}

	return s.repo.ConsumePasswordReset(row.ID, row.AdminID, string(pwHash))
}

// generateResetToken returns a URL-safe plaintext token and its SHA-256 hash
// (hex). 32 random bytes → 256 bits of entropy.
func generateResetToken() (plain, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(buf)
	return plain, hashResetToken(plain), nil
}

func hashResetToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (s *Service) Register(email, password, customerID string) (*AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	admin := &Admin{
		CustomerID:   customerID,
		Email:        email,
		PasswordHash: string(hash),
		Active:       true,
	}

	if err := s.repo.Create(admin); err != nil {
		return nil, err
	}

	return s.makeAuthResponse(admin)
}

func (s *Service) GoogleLogin(googleToken string) (*AuthResponse, error) {
	if s.verifier == nil {
		return nil, apperror.BadRequest("google login not configured", nil)
	}
	profile, err := s.verifier.Verify(googleToken)
	if err != nil {
		return nil, err
	}

	// Resolve by sub first; fall back to email and auto-link only when the sub
	// is genuinely not linked yet. Any other error (DB down, etc.) must
	// propagate — treating it as "not found" would silently re-link accounts.
	admin, err := s.repo.GetByGoogleSub(profile.Sub)
	if err != nil {
		if !isNotFound(err) {
			return nil, err
		}
		admin, err = s.repo.GetByEmail(profile.Email)
		if err != nil {
			return nil, err
		}
		if err := s.repo.LinkGoogle(admin.ID, profile); err != nil {
			return nil, err
		}
		// Re-fetch to get the canonical state with profile fields populated.
		admin, err = s.repo.GetByID(admin.ID)
		if err != nil {
			return nil, err
		}
	} else {
		// Sub already linked — refresh the cached profile (avatar/name may
		// have changed in Google).
		if err := s.repo.UpdateGoogleProfile(admin.ID, profile); err != nil {
			return nil, err
		}
		admin, err = s.repo.GetByID(admin.ID)
		if err != nil {
			return nil, err
		}
	}

	return s.makeAuthResponse(admin)
}

func (s *Service) GoogleOnboard(req GoogleOnboardingRequest) (*AuthResponse, error) {
	if s.verifier == nil {
		return nil, apperror.BadRequest("google login not configured", nil)
	}
	profile, err := s.verifier.Verify(req.GoogleToken)
	if err != nil {
		return nil, err
	}

	slug, err := s.generateUniqueSlug(req.Name)
	if err != nil {
		return nil, fmt.Errorf("generate slug: %w", err)
	}

	customerID, err := s.repo.CreateCustomer(req.Name, slug, req.Phone, req.Description)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	// Google users get a random password hash (they authenticate via Google, not password).
	randomPass := make([]byte, 32)
	if _, err := rand.Read(randomPass); err != nil {
		return nil, fmt.Errorf("read random: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword(randomPass, bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	admin := &Admin{
		CustomerID:   customerID,
		Email:        profile.Email,
		PasswordHash: string(hash),
		GoogleSub:    strPtrIf(profile.Sub),
		GoogleEmail:  strPtrIf(profile.Email),
		FullName:     strPtrIf(profile.FullName),
		AvatarURL:    strPtrIf(profile.Picture),
		Locale:       strPtrIf(profile.Locale),
		HostedDomain: strPtrIf(profile.HostedDomain),
		Active:       true,
	}

	if err := s.repo.Create(admin); err != nil {
		return nil, err
	}

	return s.makeAuthResponse(admin)
}

// isNotFound reports whether err represents a "resource not found" condition,
// so callers can distinguish a missing row from a real failure (DB errors, etc.).
func isNotFound(err error) bool {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "not_found"
	}
	return false
}

// strPtrIf returns nil for empty strings, &s otherwise. Avoids storing
// empty strings as non-NULL values for optional Google profile fields.
func strPtrIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// LinkGoogle associates the Google identity in googleToken to the admin
// identified by adminID. Fails with 409 if the Google sub is already linked
// to a different admin.
func (s *Service) LinkGoogle(adminID, googleToken string) (*Admin, error) {
	if s.verifier == nil {
		return nil, apperror.BadRequest("google login not configured", nil)
	}
	profile, err := s.verifier.Verify(googleToken)
	if err != nil {
		return nil, err
	}

	// Reject if the sub is already linked to another admin.
	if existing, err := s.repo.GetByGoogleSub(profile.Sub); err == nil && existing.ID != adminID {
		return nil, apperror.Conflict("google account already linked to another admin", nil)
	}

	if err := s.repo.LinkGoogle(adminID, profile); err != nil {
		return nil, err
	}
	return s.repo.GetByID(adminID)
}

func (s *Service) UnlinkGoogle(adminID string) (*Admin, error) {
	if err := s.repo.UnlinkGoogle(adminID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(adminID)
}

// CheckPhoneExists indica si ya hay algún customer activo con ese phone.
// No expone qué negocio es — solo true/false — para que el frontend pueda
// pedir confirmación antes de registrar otro negocio con el mismo número.
func (s *Service) CheckPhoneExists(phone string) (bool, error) {
	return s.repo.CustomerPhoneExists(phone)
}

func (s *Service) Onboard(req OnboardingRequest) (*AuthResponse, error) {
	slug, err := s.generateUniqueSlug(req.Name)
	if err != nil {
		return nil, fmt.Errorf("generate slug: %w", err)
	}

	customerID, err := s.repo.CreateCustomer(req.Name, slug, req.Phone, req.Description)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return s.Register(req.AdminEmail, req.AdminPassword, customerID)
}

func (s *Service) makeAuthResponse(admin *Admin) (*AuthResponse, error) {
	token, err := s.generateJWT(admin)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		Token: token,
		Admin: AdminSummary{
			ID:           admin.ID,
			Email:        admin.Email,
			CustomerID:   admin.CustomerID,
			GoogleEmail:  admin.GoogleEmail,
			FullName:     admin.FullName,
			AvatarURL:    admin.AvatarURL,
			Locale:       admin.Locale,
			HostedDomain: admin.HostedDomain,
		},
	}, nil
}

// generateUniqueSlug creates a slug from the first word of the name + 3 random chars.
func (s *Service) generateUniqueSlug(name string) (string, error) {
	base := slugBase(name)
	for i := 0; i < 10; i++ {
		slug := base + "-" + randomChars(3)
		exists, err := s.repo.SlugExists(slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
	}
	return "", fmt.Errorf("could not generate unique slug after 10 attempts")
}

// slugBase extracts the first word, lowercased, ASCII-only.
func slugBase(name string) string {
	// Normalize and strip accents
	s := norm.NFD.String(name)
	var b strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue // skip combining marks (accents)
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32) // lowercase
		} else if r == ' ' || r == '-' {
			if b.Len() > 0 {
				break // stop at first space/dash — we only want the first word
			}
		}
	}
	result := b.String()
	if result == "" {
		result = "negocio"
	}
	return result
}

func randomChars(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("a", n)
	}
	for i := range buf {
		buf[i] = charset[buf[i]%byte(len(charset))]
	}
	return string(buf)
}

func (s *Service) generateJWT(admin *Admin) (string, error) {
	claims := jwt.MapClaims{
		"admin_id":    admin.ID,
		"customer_id": admin.CustomerID,
		"email":       admin.Email,
		"exp":         time.Now().Add(72 * time.Hour).Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
