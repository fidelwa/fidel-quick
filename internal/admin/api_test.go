package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(repo Repository, verifier GoogleVerifier) *gin.Engine {
	svc := NewService(repo, "test-jwt-secret", verifier)
	handler := NewAPIHandler(svc)

	r := gin.New()
	auth := r.Group("/api/v1/auth")
	handler.RegisterRoutes(auth)
	return r
}

func setupAuthenticatedRouter(repo Repository, verifier GoogleVerifier, adminID string) *gin.Engine {
	svc := NewService(repo, "test-jwt-secret", verifier)
	handler := NewAPIHandler(svc)
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		if adminID != "" {
			c.Set("admin_id", adminID)
		}
		c.Next()
	})
	v1.Use(apperror.ErrorHandler(log))
	handler.RegisterAuthenticatedRoutes(v1)
	return r
}

func TestLogin_API_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo := &mockRepo{
		getByEmailFn: func(email string) (*Admin, error) {
			return &Admin{
				ID: "admin-1", CustomerID: "cust-1",
				Email: email, PasswordHash: string(hash),
			}, nil
		},
	}
	r := setupRouter(repo, nil)

	body := `{"email":"admin@test.com","password":"password123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp AuthResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "admin-1", resp.Admin.ID)
	assert.Equal(t, "admin@test.com", resp.Admin.Email)
}

func TestLogin_API_InvalidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	repo := &mockRepo{
		getByEmailFn: func(_ string) (*Admin, error) {
			return &Admin{PasswordHash: string(hash)}, nil
		},
	}
	r := setupRouter(repo, nil)

	body := `{"email":"admin@test.com","password":"wrongpassword"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestLogin_API_MissingFields(t *testing.T) {
	r := setupRouter(&mockRepo{}, nil)

	body := `{"email":"admin@test.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestRegister_API_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(admin *Admin) error {
			admin.ID = "new-admin"
			return nil
		},
	}
	r := setupRouter(repo, nil)

	body := `{"email":"new@test.com","password":"password123","customer_id":"550e8400-e29b-41d4-a716-446655440000"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)

	var resp AuthResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "new-admin", resp.Admin.ID)
}

func TestRegister_API_MissingFields(t *testing.T) {
	r := setupRouter(&mockRepo{}, nil)

	body := `{"email":"new@test.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestRegister_API_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ *Admin) error {
			return &duplicateErr{}
		},
	}
	r := setupRouter(repo, nil)

	body := `{"email":"existing@test.com","password":"password123","customer_id":"550e8400-e29b-41d4-a716-446655440000"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 409, w.Code)
}

// --- Google login ---

func TestGoogleLogin_API_Success(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(sub string) (*Admin, error) {
			return &Admin{ID: "a-1", Email: "u@gmail.com", CustomerID: "c-1", GoogleSub: &sub}, nil
		},
	}
	r := setupRouter(repo, &mockVerifier{profile: tprofile("u@gmail.com", "g-1")})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login/google",
		strings.NewReader(`{"google_token":"valid-token"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp AuthResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "a-1", resp.Admin.ID)
}

func TestGoogleLogin_API_MissingToken(t *testing.T) {
	r := setupRouter(&mockRepo{}, &mockVerifier{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login/google",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestGoogleLogin_API_InvalidToken(t *testing.T) {
	r := setupRouter(&mockRepo{}, &mockVerifier{err: errors.New("aud mismatch")})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login/google",
		strings.NewReader(`{"google_token":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)
}

// --- Link/Unlink ---

func TestLinkGoogle_API_Success(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(_ string) (*Admin, error) {
			return nil, apperror.NotFound("not found", nil)
		},
		getByIDFn: func(id string) (*Admin, error) {
			gmail := "u@gmail.com"
			return &Admin{ID: id, Email: "admin@x.com", CustomerID: "c-1", GoogleEmail: &gmail}, nil
		},
	}
	r := setupAuthenticatedRouter(repo, &mockVerifier{profile: tprofile("u@gmail.com", "g-1")}, "a-1")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/link/google",
		strings.NewReader(`{"google_token":"tok"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var summary AdminSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))
	require.NotNil(t, summary.GoogleEmail)
	assert.Equal(t, "u@gmail.com", *summary.GoogleEmail)
}

func TestLinkGoogle_API_NoAdminID(t *testing.T) {
	r := setupAuthenticatedRouter(&mockRepo{}, &mockVerifier{}, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/link/google",
		strings.NewReader(`{"google_token":"tok"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestLinkGoogle_API_Conflict(t *testing.T) {
	repo := &mockRepo{
		getByGoogleSubFn: func(_ string) (*Admin, error) {
			return &Admin{ID: "other-admin"}, nil
		},
	}
	r := setupAuthenticatedRouter(repo, &mockVerifier{profile: tprofile("u@gmail.com", "g-1")}, "a-1")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/link/google",
		strings.NewReader(`{"google_token":"tok"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 409, w.Code)
}

func TestUnlinkGoogle_API_Success(t *testing.T) {
	repo := &mockRepo{
		unlinkGoogleFn: func(_ string) error { return nil },
		getByIDFn: func(id string) (*Admin, error) {
			return &Admin{ID: id, Email: "admin@x.com", CustomerID: "c-1"}, nil
		},
	}
	r := setupAuthenticatedRouter(repo, nil, "a-1")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/auth/link/google", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	var summary AdminSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))
	assert.Nil(t, summary.GoogleEmail)
}

func TestMe_API_Success(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id string) (*Admin, error) {
			gmail := "u@gmail.com"
			return &Admin{ID: id, Email: "admin@x.com", CustomerID: "c-1", GoogleEmail: &gmail}, nil
		},
	}
	r := setupAuthenticatedRouter(repo, nil, "a-1")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	var summary AdminSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))
	require.NotNil(t, summary.GoogleEmail)
	assert.Equal(t, "u@gmail.com", *summary.GoogleEmail)
}
