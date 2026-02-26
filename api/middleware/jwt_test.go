package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const (
	testJWTSecret  = "test-secret-key"
	testBearerToken = "test-bearer-token"
)

func generateTestJWT(secret string, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func setupTestRouter() *gin.Engine {
	r := gin.New()
	r.Use(JWTOrBearer(testJWTSecret, testBearerToken))
	r.GET("/protected", func(c *gin.Context) {
		customerID, _ := c.Get("customer_id")
		adminID, _ := c.Get("admin_id")
		c.JSON(200, gin.H{
			"customer_id": customerID,
			"admin_id":    adminID,
		})
	})
	return r
}

func TestJWTOrBearer_ValidBearerToken(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+testBearerToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestJWTOrBearer_ValidJWT(t *testing.T) {
	token := generateTestJWT(testJWTSecret, jwt.MapClaims{
		"customer_id": "cust-123",
		"admin_id":    "admin-456",
		"exp":         time.Now().Add(1 * time.Hour).Unix(),
	})

	r := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "cust-123", resp["customer_id"])
	assert.Equal(t, "admin-456", resp["admin_id"])
}

func TestJWTOrBearer_ExpiredJWT(t *testing.T) {
	token := generateTestJWT(testJWTSecret, jwt.MapClaims{
		"customer_id": "cust-123",
		"exp":         time.Now().Add(-1 * time.Hour).Unix(),
	})

	r := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestJWTOrBearer_InvalidSecret(t *testing.T) {
	token := generateTestJWT("wrong-secret", jwt.MapClaims{
		"customer_id": "cust-123",
		"exp":         time.Now().Add(1 * time.Hour).Unix(),
	})

	r := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestJWTOrBearer_NoAuthHeader(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "unauthorized", resp["error"])
}

func TestJWTOrBearer_InvalidFormat(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestJWTOrBearer_WrongBearerToken(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	r.ServeHTTP(w, req)

	// wrong-token is not the bearer token and not a valid JWT, so 401
	assert.Equal(t, 401, w.Code)
}

func TestBearerAuth_Valid(t *testing.T) {
	r := gin.New()
	r.Use(BearerAuth("my-token"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestBearerAuth_Invalid(t *testing.T) {
	r := gin.New()
	r.Use(BearerAuth("my-token"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestBearerAuth_MissingHeader(t *testing.T) {
	r := gin.New()
	r.Use(BearerAuth("my-token"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}
