package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"github.com/theluisbolivar/fidel-quick/internal/platform/email"
	"golang.org/x/crypto/bcrypt"
)

// --- Stub email sender ---

type stubSender struct {
	sent []email.Message
	err  error
}

func (s *stubSender) Send(_ context.Context, msg email.Message) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, msg)
	return nil
}

// resetSvc builds a Service with password reset enabled against the given
// repo and email sender.
func resetSvc(repo Repository, sender email.Sender) *Service {
	svc := NewService(repo, "test-secret", nil)
	svc.WithPasswordReset(sender, "http://localhost:5173", nil)
	return svc
}

// --- ForgotPassword ---

func TestForgotPassword_HappyPath_SendsLink(t *testing.T) {
	var storedHash string
	repo := &mockRepo{
		getByEmailFn: func(e string) (*Admin, error) {
			return &Admin{ID: "a-1", Email: e, CustomerID: "c-1"}, nil
		},
		createResetTokenFn: func(_, tokenHash string, expiresAt time.Time) error {
			storedHash = tokenHash
			assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 5*time.Second)
			return nil
		},
	}
	sender := &stubSender{}
	svc := resetSvc(repo, sender)

	err := svc.ForgotPassword(context.Background(), "Owner@Test.com")
	require.NoError(t, err)

	// Persisted a hash (never the plaintext).
	require.NotEmpty(t, storedHash)
	require.Len(t, sender.sent, 1)
	msg := sender.sent[0]
	assert.Equal(t, "owner@test.com", msg.To)
	assert.Contains(t, msg.Body, "http://localhost:5173/reset-password?token=")

	// The token in the link must hash to what we stored.
	token := extractToken(t, msg.Body)
	assert.Equal(t, storedHash, hashResetToken(token))
	// And must NOT be stored verbatim.
	assert.NotEqual(t, token, storedHash)
}

func TestForgotPassword_UnknownEmail_NoErrorNoEmail(t *testing.T) {
	repo := &mockRepo{
		getByEmailFn: func(_ string) (*Admin, error) {
			return nil, apperror.NotFound("admin not found", nil)
		},
	}
	sender := &stubSender{}
	svc := resetSvc(repo, sender)

	err := svc.ForgotPassword(context.Background(), "ghost@test.com")
	require.NoError(t, err) // must NOT leak that the email is unknown
	assert.Empty(t, sender.sent)
}

func TestForgotPassword_RateLimited(t *testing.T) {
	repo := &mockRepo{
		getByEmailFn: func(e string) (*Admin, error) {
			return &Admin{ID: "a-1", Email: e}, nil
		},
	}
	svc := resetSvc(repo, &stubSender{})

	// 5 allowed per hour; the 6th must be rejected.
	for i := 0; i < 5; i++ {
		require.NoError(t, svc.ForgotPassword(context.Background(), "spammer@test.com"))
	}
	err := svc.ForgotPassword(context.Background(), "spammer@test.com")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 429, appErr.HTTPStatus)
}

// --- ResetPassword ---

func TestResetPassword_HappyPath(t *testing.T) {
	var consumed struct {
		tokenID, adminID, hash string
	}
	token := "plain-token-abc"
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			assert.Equal(t, hashResetToken(token), tokenHash)
			return &PasswordResetToken{
				ID:        "t-1",
				AdminID:   "a-1",
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(30 * time.Minute),
			}, nil
		},
		consumeResetFn: func(tokenID, adminID, newPasswordHash string) error {
			consumed.tokenID = tokenID
			consumed.adminID = adminID
			consumed.hash = newPasswordHash
			return nil
		},
	}
	svc := resetSvc(repo, &stubSender{})

	err := svc.ResetPassword("1.2.3.4", token, "newpassword123")
	require.NoError(t, err)
	assert.Equal(t, "t-1", consumed.tokenID)
	assert.Equal(t, "a-1", consumed.adminID)
	// The stored hash must verify against the new password.
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(consumed.hash), []byte("newpassword123")))
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	consumeCalled := false
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			return &PasswordResetToken{
				ID:        "t-1",
				AdminID:   "a-1",
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(-time.Minute), // expired
			}, nil
		},
		consumeResetFn: func(_, _, _ string) error { consumeCalled = true; return nil },
	}
	svc := resetSvc(repo, &stubSender{})

	err := svc.ResetPassword("1.2.3.4", "tok", "newpassword123")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.HTTPStatus)
	assert.False(t, consumeCalled, "must not update password for expired token")
}

func TestResetPassword_ReusedToken(t *testing.T) {
	used := time.Now().Add(-time.Minute)
	consumeCalled := false
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			return &PasswordResetToken{
				ID:        "t-1",
				AdminID:   "a-1",
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(30 * time.Minute),
				UsedAt:    &used, // already consumed
			}, nil
		},
		consumeResetFn: func(_, _, _ string) error { consumeCalled = true; return nil },
	}
	svc := resetSvc(repo, &stubSender{})

	err := svc.ResetPassword("1.2.3.4", "tok", "newpassword123")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.HTTPStatus)
	assert.False(t, consumeCalled, "must not reuse an already-used token")
}

func TestResetPassword_UnknownToken(t *testing.T) {
	repo := &mockRepo{
		getResetTokenFn: func(_ string) (*PasswordResetToken, error) {
			return nil, apperror.NotFound("reset token not found", nil)
		},
	}
	svc := resetSvc(repo, &stubSender{})

	err := svc.ResetPassword("1.2.3.4", "nope", "newpassword123")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

func TestResetPassword_ShortPassword(t *testing.T) {
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			return &PasswordResetToken{ID: "t-1", AdminID: "a-1", TokenHash: tokenHash, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}
	svc := resetSvc(repo, &stubSender{})

	err := svc.ResetPassword("1.2.3.4", "tok", "short")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

// TestResetPassword_RateLimited verifies the reset-password endpoint is rate
// limited per client IP (10/h). The 11th attempt from the same IP is rejected
// with 429, while a different IP still has its own budget (SV-1).
func TestResetPassword_RateLimited(t *testing.T) {
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			return &PasswordResetToken{
				ID: "t-1", AdminID: "a-1", TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}
	svc := resetSvc(repo, &stubSender{})

	// 10 allowed per hour per IP; the 11th from the same IP must be rejected.
	for i := 0; i < 10; i++ {
		require.NoError(t, svc.ResetPassword("9.9.9.9", "tok", "newpassword123"))
	}
	err := svc.ResetPassword("9.9.9.9", "tok", "newpassword123")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 429, appErr.HTTPStatus)

	// A different IP keeps its own independent budget.
	require.NoError(t, svc.ResetPassword("8.8.8.8", "tok", "newpassword123"))
}

// TestResetPassword_AdminGoneNoConsume verifies that if the admin row is gone
// (deleted/nonexistent) the repository reports an error and does NOT mark the
// token used — the whole transaction rolls back (LG-1). Exercised at the
// service level: ConsumePasswordReset surfaces the NotFound.
func TestResetPassword_AdminGoneNoConsume(t *testing.T) {
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			return &PasswordResetToken{
				ID: "t-1", AdminID: "ghost", TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
		consumeResetFn: func(_, adminID, _ string) error {
			// Emulates the repo detecting 0 rows affected on the admins UPDATE.
			assert.Equal(t, "ghost", adminID)
			return apperror.NotFound("admin not found", nil)
		},
	}
	svc := resetSvc(repo, &stubSender{})

	err := svc.ResetPassword("1.2.3.4", "tok", "newpassword123")
	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 404, appErr.HTTPStatus)
}

// --- API handlers ---

func setupResetRouter(repo Repository, sender email.Sender) *gin.Engine {
	svc := resetSvc(repo, sender)
	handler := NewAPIHandler(svc)
	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.Use(apperror.ErrorHandler(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))))
	handler.RegisterRoutes(auth)
	return r
}

func TestForgotPassword_API_Always200(t *testing.T) {
	// Even for an unknown email the API returns 200 with a neutral message.
	repo := &mockRepo{
		getByEmailFn: func(_ string) (*Admin, error) {
			return nil, apperror.NotFound("admin not found", nil)
		},
	}
	r := setupResetRouter(repo, &stubSender{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password",
		strings.NewReader(`{"email":"ghost@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body, "message")
}

func TestForgotPassword_API_MissingEmail(t *testing.T) {
	r := setupResetRouter(&mockRepo{}, &stubSender{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestResetPassword_API_Success(t *testing.T) {
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			return &PasswordResetToken{ID: "t-1", AdminID: "a-1", TokenHash: tokenHash, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}
	r := setupResetRouter(repo, &stubSender{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password",
		strings.NewReader(`{"token":"tok","new_password":"newpassword123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestResetPassword_API_ExpiredToken400(t *testing.T) {
	repo := &mockRepo{
		getResetTokenFn: func(tokenHash string) (*PasswordResetToken, error) {
			return &PasswordResetToken{ID: "t-1", AdminID: "a-1", TokenHash: tokenHash, ExpiresAt: time.Now().Add(-time.Hour)}, nil
		},
	}
	r := setupResetRouter(repo, &stubSender{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password",
		strings.NewReader(`{"token":"tok","new_password":"newpassword123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestResetPassword_API_MissingFields(t *testing.T) {
	r := setupResetRouter(&mockRepo{}, &stubSender{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password",
		strings.NewReader(`{"token":"tok"}`)) // no new_password
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

// extractToken pulls the ?token= value out of the emailed reset link.
func extractToken(t *testing.T, body string) string {
	t.Helper()
	const marker = "token="
	i := strings.Index(body, marker)
	require.GreaterOrEqual(t, i, 0, "no token in body")
	rest := body[i+len(marker):]
	// token ends at the first whitespace/newline.
	if j := strings.IndexAny(rest, " \n\t"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}
