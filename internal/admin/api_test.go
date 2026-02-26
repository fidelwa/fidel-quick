package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(repo Repository) *gin.Engine {
	svc := NewService(repo, "test-jwt-secret")
	handler := NewAPIHandler(svc)

	r := gin.New()
	auth := r.Group("/api/v1/auth")
	handler.RegisterRoutes(auth)
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
	r := setupRouter(repo)

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
	r := setupRouter(repo)

	body := `{"email":"admin@test.com","password":"wrongpassword"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestLogin_API_MissingFields(t *testing.T) {
	r := setupRouter(&mockRepo{})

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
	r := setupRouter(repo)

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
	r := setupRouter(&mockRepo{})

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
	r := setupRouter(repo)

	body := `{"email":"existing@test.com","password":"password123","customer_id":"550e8400-e29b-41d4-a716-446655440000"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 409, w.Code)
}
