package admin

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"
)

type Service struct {
	repo           Repository
	jwtSecret      []byte
	googleClientID string
}

func NewService(repo Repository, jwtSecret, googleClientID string) *Service {
	return &Service{
		repo:           repo,
		jwtSecret:      []byte(jwtSecret),
		googleClientID: googleClientID,
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

	token, err := s.generateJWT(admin)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		Admin: AdminSummary{
			ID:         admin.ID,
			Email:      admin.Email,
			CustomerID: admin.CustomerID,
		},
	}, nil
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

	token, err := s.generateJWT(admin)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		Admin: AdminSummary{
			ID:         admin.ID,
			Email:      admin.Email,
			CustomerID: admin.CustomerID,
		},
	}, nil
}

func (s *Service) GoogleLogin(googleToken string) (*AuthResponse, error) {
	email, err := s.verifyGoogleToken(googleToken)
	if err != nil {
		return nil, err
	}

	admin, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	token, err := s.generateJWT(admin)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		Admin: AdminSummary{
			ID:         admin.ID,
			Email:      admin.Email,
			CustomerID: admin.CustomerID,
		},
	}, nil
}

func (s *Service) GoogleOnboard(req GoogleOnboardingRequest) (*AuthResponse, error) {
	email, err := s.verifyGoogleToken(req.GoogleToken)
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

	// Google users get a random password hash (they authenticate via Google, not password)
	randomPass := make([]byte, 32)
	rand.Read(randomPass)
	hash, err := bcrypt.GenerateFromPassword(randomPass, bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
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

	token, err := s.generateJWT(admin)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		Admin: AdminSummary{
			ID:         admin.ID,
			Email:      admin.Email,
			CustomerID: admin.CustomerID,
		},
	}, nil
}

func (s *Service) verifyGoogleToken(idToken string) (string, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken))
	if err != nil {
		return "", fmt.Errorf("verify google token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid google token")
	}

	var claims struct {
		Aud           string `json:"aud"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return "", fmt.Errorf("decode google token: %w", err)
	}

	if s.googleClientID != "" && claims.Aud != s.googleClientID {
		return "", fmt.Errorf("google token audience mismatch")
	}

	if claims.EmailVerified != "true" {
		return "", fmt.Errorf("google email not verified")
	}

	return claims.Email, nil
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
	rand.Read(buf)
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
