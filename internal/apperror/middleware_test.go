package apperror

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestErrorHandler_AppError_NotFound(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler(testLogger()))
	r.GET("/test", func(c *gin.Context) {
		c.Error(NotFound("recurso no encontrado", nil))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "recurso no encontrado", resp["error"])
	assert.Equal(t, "not_found", resp["code"])
}

func TestErrorHandler_AppError_BadRequest(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler(testLogger()))
	r.GET("/test", func(c *gin.Context) {
		c.Error(BadRequest("datos invalidos", nil))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestErrorHandler_AppError_Internal(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler(testLogger()))
	r.GET("/test", func(c *gin.Context) {
		c.Error(Internal("db error", fmt.Errorf("connection refused")))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Internal errors should not expose details
	assert.Equal(t, "error interno", resp["error"])
	assert.Equal(t, "internal_error", resp["code"])
}

func TestErrorHandler_UnknownError(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler(testLogger()))
	r.GET("/test", func(c *gin.Context) {
		c.Error(fmt.Errorf("random panic"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "error interno", resp["error"])
}

func TestErrorHandler_NoError(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler(testLogger()))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestErrorHandler_Conflict(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler(testLogger()))
	r.GET("/test", func(c *gin.Context) {
		c.Error(Conflict("duplicate", nil))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 409, w.Code)
}
