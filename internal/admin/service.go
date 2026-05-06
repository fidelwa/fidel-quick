package admin

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"
)

type Service struct {
	repo      Repository
	jwtSecret []byte
	verifier  GoogleVerifier
}

// NewService wires the repository, JWT signing secret, and Google verifier.
// Pass a nil verifier to disable Google login/signup/linking — the corresponding
// service methods will then return a "google login not configured" error.
func NewService(repo Repository, jwtSecret string, verifier GoogleVerifier) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		verifier:  verifier,
	}
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
	email, sub, err := s.verifier.Verify(googleToken)
	if err != nil {
		return nil, err
	}

	// Resolve by sub first; fall back to email and auto-link if found.
	admin, err := s.repo.GetByGoogleSub(sub)
	if err != nil {
		admin, err = s.repo.GetByEmail(email)
		if err != nil {
			return nil, err
		}
		if err := s.repo.LinkGoogle(admin.ID, sub, email); err != nil {
			return nil, err
		}
		v := sub
		admin.GoogleSub = &v
		ge := email
		admin.GoogleEmail = &ge
	}

	return s.makeAuthResponse(admin)
}

func (s *Service) GoogleOnboard(req GoogleOnboardingRequest) (*AuthResponse, error) {
	if s.verifier == nil {
		return nil, apperror.BadRequest("google login not configured", nil)
	}
	email, sub, err := s.verifier.Verify(req.GoogleToken)
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

	subCopy := sub
	emailCopy := email
	admin := &Admin{
		CustomerID:   customerID,
		Email:        email,
		PasswordHash: string(hash),
		GoogleSub:    &subCopy,
		GoogleEmail:  &emailCopy,
		Active:       true,
	}

	if err := s.repo.Create(admin); err != nil {
		return nil, err
	}

	return s.makeAuthResponse(admin)
}

// LinkGoogle associates the Google identity in googleToken to the admin
// identified by adminID. Fails with 409 if the Google sub is already linked
// to a different admin.
func (s *Service) LinkGoogle(adminID, googleToken string) (*Admin, error) {
	if s.verifier == nil {
		return nil, apperror.BadRequest("google login not configured", nil)
	}
	email, sub, err := s.verifier.Verify(googleToken)
	if err != nil {
		return nil, err
	}

	// Reject if the sub is already linked to another admin.
	if existing, err := s.repo.GetByGoogleSub(sub); err == nil && existing.ID != adminID {
		return nil, apperror.Conflict("google account already linked to another admin", nil)
	}

	if err := s.repo.LinkGoogle(adminID, sub, email); err != nil {
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
			ID:          admin.ID,
			Email:       admin.Email,
			CustomerID:  admin.CustomerID,
			GoogleEmail: admin.GoogleEmail,
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
