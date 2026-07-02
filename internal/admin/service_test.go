package admin

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock Repository ---

type mockRepo struct {
	getByEmailFn          func(email string) (*Admin, error)
	getByIDFn             func(id string) (*Admin, error)
	getByGoogleSubFn      func(sub string) (*Admin, error)
	createFn              func(admin *Admin) error
	createCustomerFn      func(name, slug, phone, description string) (string, error)
	slugExistsFn          func(slug string) (bool, error)
	linkGoogleFn          func(adminID string, profile *GoogleProfile) error
	unlinkGoogleFn        func(adminID string) error
	updateGoogleProfileFn func(adminID string, profile *GoogleProfile) error
}

func (m *mockRepo) GetByEmail(email string) (*Admin, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(email)
	}
	return nil, apperror.NotFound("admin not found", nil)
}

func (m *mockRepo) GetByID(id string) (*Admin, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return &Admin{ID: id, Email: "stub@test.com", CustomerID: "cust"}, nil
}

func (m *mockRepo) GetByGoogleSub(sub string) (*Admin, error) {
	if m.getByGoogleSubFn != nil {
		return m.getByGoogleSubFn(sub)
	}
	return nil, apperror.NotFound("admin not found", nil)
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

func (m *mockRepo) CustomerPhoneExists(_ string) (bool, error) {
	return false, nil
}

func (m *mockRepo) LinkGoogle(adminID string, profile *GoogleProfile) error {
	if m.linkGoogleFn != nil {
		return m.linkGoogleFn(adminID, profile)
	}
	return nil
}

func (m *mockRepo) UnlinkGoogle(adminID string) error {
	if m.unlinkGoogleFn != nil {
		return m.unlinkGoogleFn(adminID)
	}
	return nil
}

func (m *mockRepo) UpdateGoogleProfile(adminID string, profile *GoogleProfile) error {
	if m.updateGoogleProfileFn != nil {
		return m.updateGoogleProfileFn(adminID, profile)
	}
	return nil
}

// --- Mock Google verifier ---

type mockVerifier struct {
	profile *GoogleProfile
	err     error
}

func (m *mockVerifier) Verify(_ string) (*GoogleProfile, error) {
	return m.profile, m.err
}

// helper to build a profile in tests.
func tprofile(email, sub string) *GoogleProfile {
	return &GoogleProfile{Email: email, Sub: sub, EmailVerified: true}
}

func strPtr(s string) *string { return &s }

// --- Tests: email/password ---

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
	svc := NewService(repo, "test-secret", nil)

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
	svc := NewService(repo, "test-secret", nil)

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
	svc := NewService(repo, "test-secret", nil)

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
	svc := NewService(repo, "test-secret", nil)

	resp, err := svc.Register("new@test.com", "password123", "cust-1")

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "new-admin", resp.Admin.ID)
	assert.Equal(t, "new@test.com", resp.Admin.Email)
	assert.Equal(t, "cust-1", resp.Admin.CustomerID)

	assert.NotEqual(t, "password123", createdAdmin.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(createdAdmin.PasswordHash), []byte("password123")))
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ *Admin) error {
			return &duplicateErr{}
		},
	}
	svc := NewService(repo, "test-secret", nil)

	_, err := svc.Register("existing@test.com", "password123", "cust-1")

	require.Error(t, err)
}

func TestGenerateJWT(t *testing.T) {
	svc := NewService(&mockRepo{}, "test-secret-key", nil)

	admin := &Admin{
		ID:         "admin-1",
		CustomerID: "cust-1",
		Email:      "admin@test.com",
	}

	token, err := svc.generateJWT(admin)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)
}

// --- Tests: Google login ---

func TestGoogleLogin_NotConfigured(t *testing.T) {
	svc := NewService(&mockRepo{}, "secret", nil)
	_, err := svc.GoogleLogin("any-token")
	require.Error(t, err)
}

func TestGoogleLogin_VerifyFails(t *testing.T) {
	svc := NewService(&mockRepo{}, "secret", &mockVerifier{err: errors.New("email not verified")})
	_, err := svc.GoogleLogin("bad-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email not verified")
}

func TestGoogleLogin_ExistingSub(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(sub string) (*Admin, error) {
			return &Admin{ID: "a-1", Email: "user@gmail.com", CustomerID: "c-1", GoogleSub: &sub}, nil
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("user@gmail.com", "g-123")})
	resp, err := svc.GoogleLogin("token")
	require.NoError(t, err)
	assert.Equal(t, "a-1", resp.Admin.ID)
}

func TestGoogleLogin_FallbackToEmailLinks(t *testing.T) {
	linked := false
	repo := &mockRepo{
		getByGoogleSubFn: func(_ string) (*Admin, error) {
			return nil, apperror.NotFound("admin not found", nil)
		},
		getByEmailFn: func(email string) (*Admin, error) {
			return &Admin{ID: "a-1", Email: email, CustomerID: "c-1"}, nil
		},
		linkGoogleFn: func(adminID string, profile *GoogleProfile) error {
			linked = true
			assert.Equal(t, "a-1", adminID)
			assert.Equal(t, "g-123", profile.Sub)
			assert.Equal(t, "user@gmail.com", profile.Email)
			return nil
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("user@gmail.com", "g-123")})
	resp, err := svc.GoogleLogin("token")
	require.NoError(t, err)
	assert.True(t, linked, "expected auto-link on first Google login")
	assert.Equal(t, "a-1", resp.Admin.ID)
}

func TestGoogleLogin_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(_ string) (*Admin, error) {
			return nil, apperror.NotFound("admin not found", nil)
		},
		getByEmailFn: func(_ string) (*Admin, error) {
			return nil, apperror.NotFound("admin not found", nil)
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("user@gmail.com", "g-1")})
	_, err := svc.GoogleLogin("token")
	require.Error(t, err)
}

// --- Tests: Google onboard ---

func TestGoogleOnboard_Success(t *testing.T) {
	var createdAdmin *Admin
	repo := &mockRepo{
		createFn: func(a *Admin) error {
			a.ID = "new-admin"
			createdAdmin = a
			return nil
		},
		createCustomerFn: func(_, _, _, _ string) (string, error) {
			return "cust-new", nil
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("owner@gmail.com", "g-7")})
	req := GoogleOnboardingRequest{GoogleToken: "tok", Name: "Mi Cafe", Phone: "+5215555"}
	resp, err := svc.GoogleOnboard(req)
	require.NoError(t, err)
	assert.Equal(t, "new-admin", resp.Admin.ID)
	require.NotNil(t, createdAdmin.GoogleSub)
	assert.Equal(t, "g-7", *createdAdmin.GoogleSub)
	require.NotNil(t, createdAdmin.GoogleEmail)
	assert.Equal(t, "owner@gmail.com", *createdAdmin.GoogleEmail)
}

func TestGoogleOnboard_NotConfigured(t *testing.T) {
	svc := NewService(&mockRepo{}, "secret", nil)
	_, err := svc.GoogleOnboard(GoogleOnboardingRequest{GoogleToken: "x", Name: "n", Phone: "p"})
	require.Error(t, err)
}

func TestGoogleOnboard_VerifyFails(t *testing.T) {
	svc := NewService(&mockRepo{}, "secret", &mockVerifier{err: errors.New("aud mismatch")})
	_, err := svc.GoogleOnboard(GoogleOnboardingRequest{GoogleToken: "x", Name: "n", Phone: "p"})
	require.Error(t, err)
}

func TestGoogleOnboard_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ *Admin) error {
			return apperror.Conflict("email already registered", nil)
		},
		createCustomerFn: func(_, _, _, _ string) (string, error) {
			return "cust-new", nil
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("dup@gmail.com", "g-1")})
	_, err := svc.GoogleOnboard(GoogleOnboardingRequest{GoogleToken: "tok", Name: "X", Phone: "p"})
	require.Error(t, err)
}

// --- Tests: Link/Unlink ---

func TestLinkGoogle_Success(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(_ string) (*Admin, error) {
			return nil, apperror.NotFound("not found", nil)
		},
		getByIDFn: func(id string) (*Admin, error) {
			return &Admin{ID: id, Email: "admin@x.com", CustomerID: "c", GoogleSub: strPtr("g-1"), GoogleEmail: strPtr("admin@gmail.com")}, nil
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("admin@gmail.com", "g-1")})
	a, err := svc.LinkGoogle("a-1", "tok")
	require.NoError(t, err)
	require.NotNil(t, a.GoogleEmail)
	assert.Equal(t, "admin@gmail.com", *a.GoogleEmail)
}

func TestLinkGoogle_AlreadyLinkedToOtherAdmin(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(_ string) (*Admin, error) {
			return &Admin{ID: "other-admin"}, nil
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("x@y.com", "g-1")})
	_, err := svc.LinkGoogle("a-1", "tok")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.HTTPStatus)
}

func TestLinkGoogle_SameAdminIdempotent(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(_ string) (*Admin, error) {
			return &Admin{ID: "a-1"}, nil
		},
		getByIDFn: func(id string) (*Admin, error) {
			return &Admin{ID: id, Email: "x@y.com", CustomerID: "c"}, nil
		},
	}
	svc := NewService(repo, "secret", &mockVerifier{profile: tprofile("x@y.com", "g-1")})
	_, err := svc.LinkGoogle("a-1", "tok")
	require.NoError(t, err)
}

func TestUnlinkGoogle_Success(t *testing.T) {
	called := false
	repo := &mockRepo{
		unlinkGoogleFn: func(_ string) error {
			called = true
			return nil
		},
		getByIDFn: func(id string) (*Admin, error) {
			return &Admin{ID: id, Email: "x@y.com"}, nil
		},
	}
	svc := NewService(repo, "secret", nil)
	a, err := svc.UnlinkGoogle("a-1")
	require.NoError(t, err)
	assert.True(t, called)
	assert.Nil(t, a.GoogleEmail)
}

// helper error types

type notFoundErr struct{}

func (e *notFoundErr) Error() string { return "not found" }

type duplicateErr struct{}

func (e *duplicateErr) Error() string { return "duplicate key" }
