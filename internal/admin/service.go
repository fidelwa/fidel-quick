package admin

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      Repository
	jwtSecret []byte
}

func NewService(repo Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
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
