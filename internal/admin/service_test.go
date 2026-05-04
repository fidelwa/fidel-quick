package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock Repository ---

type mockRepo struct {
	getByEmailFn     func(email string) (*Admin, error)
	createFn         func(admin *Admin) error
	createCustomerFn func(name, slug, phone, description string) (string, error)
	slugExistsFn     func(slug string) (bool, error)
}

func (m *mockRepo) GetByEmail(email string) (*Admin, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(email)
	}
	return nil, nil
}

func (m *mockRepo) Create(admin *Admin) error {
	if m.createFn != nil {
		return m.createFn(admin)
	}
	admin.ID = "new-admin"
	return nil
}

func (m *mockRepo) CreateCustomer(name, slug, phone, description string) (string, error) {
	if m.createCustomerFn != nil {
		return m.createCustomerFn(name, slug, phone, description)
	}
	return "new-cust", nil
}

func (m *mockRepo) SlugExists(slug string) (bool, error) {
	if m.slugExistsFn != nil {
		return m.slugExistsFn(slug)
	}
	return false, nil
}

// --- Tests ---

func TestLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo := &mockRepo{
		getByEmailFn: func(email string) (*Admin, error) {
			return &Admin{
				ID:           "admin-1",
				CustomerID:   "cust-1",
				Email:        email,
				PasswordHash: string(hash),
				Active:       true,
			}, nil
		},
	}
	svc := NewService(repo, "test-secret", "")

	resp, err := svc.Login("admin@test.com", "password123")

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "admin-1", resp.Admin.ID)
	assert.Equal(t, "admin@test.com", resp.Admin.Email)
	assert.Equal(t, "cust-1", resp.Admin.CustomerID)
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	repo := &mockRepo{
		getByEmailFn: func(_ string) (*Admin, error) {
			return &Admin{PasswordHash: string(hash)}, nil
		},
	}
	svc := NewService(repo, "test-secret", "")

	_, err := svc.Login("admin@test.com", "wrong")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		getByEmailFn: func(_ string) (*Admin, error) {
			return nil, &notFoundErr{}
		},
	}
	svc := NewService(repo, "test-secret", "")

	_, err := svc.Login("nonexistent@test.com", "password")

	require.Error(t, err)
}

func TestRegister_Success(t *testing.T) {
	var createdAdmin *Admin
	repo := &mockRepo{
		createFn: func(admin *Admin) error {
			createdAdmin = admin
			admin.ID = "new-admin"
			return nil
		},
	}
	svc := NewService(repo, "test-secret", "")

	resp, err := svc.Register("new@test.com", "password123", "cust-1")

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "new-admin", resp.Admin.ID)
	assert.Equal(t, "new@test.com", resp.Admin.Email)
	assert.Equal(t, "cust-1", resp.Admin.CustomerID)

	// Verify password was hashed
	assert.NotEqual(t, "password123", createdAdmin.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(createdAdmin.PasswordHash), []byte("password123")))
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ *Admin) error {
			return &duplicateErr{}
		},
	}
	svc := NewService(repo, "test-secret", "")

	_, err := svc.Register("existing@test.com", "password123", "cust-1")

	require.Error(t, err)
}

func TestGenerateJWT(t *testing.T) {
	svc := NewService(&mockRepo{}, "test-secret-key", "")

	admin := &Admin{
		ID:         "admin-1",
		CustomerID: "cust-1",
		Email:      "admin@test.com",
	}

	token, err := svc.generateJWT(admin)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// Token should be a valid JWT (3 parts separated by dots)
	assert.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)
}

// helper error types

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }

type duplicateErr struct{}

func (e *duplicateErr) Error() string { return "duplicate key" }
