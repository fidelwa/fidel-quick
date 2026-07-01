package whatsapp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sign returns the "sha256=<hex>" header value Meta would send for body+secret.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	const secret = "super-secret"
	body := []byte(`{"object":"whatsapp_business_account"}`)
	valid := sign(body, secret)

	tests := []struct {
		name   string
		header string
		body   []byte
		want   bool
	}{
		{"valid signature", valid, body, true},
		{"wrong secret", sign(body, "other-secret"), body, false},
		{"tampered body", valid, []byte(`{"object":"tampered"}`), false},
		{"missing prefix", hex.EncodeToString([]byte("nope")), body, false},
		{"empty header", "", body, false},
		{"prefix only", "sha256=", body, false},
		{"non-hex payload", "sha256=zzzz", body, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, verifySignature(tt.header, tt.body, secret))
		})
	}
}

// newSignatureTestHandler builds a handler exercising only the signature gate.
// The dependencies below the gate are nil because a rejected request (401)
// never reaches them.
func newSignatureTestHandler(secret string) *WebhookHandler {
	return &WebhookHandler{
		appSecret: secret,
		log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func postWebhook(t *testing.T, h *WebhookHandler, body []byte, header string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set("X-Hub-Signature-256", header)
	}
	c.Request = req
	h.Receive(c)
	return w
}

func TestReceive_InvalidSignatureRejected(t *testing.T) {
	const secret = "app-secret"
	body := []byte(`{"object":"whatsapp_business_account","entry":[]}`)

	// Signed with the wrong secret => must be rejected before processing.
	w := postWebhook(t, newSignatureTestHandler(secret), body, sign(body, "attacker"))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReceive_ValidSignatureAccepted(t *testing.T) {
	const secret = "app-secret"
	body := []byte(`{"object":"whatsapp_business_account","entry":[]}`)

	// Empty entry list => passes the gate, binds JSON, responds 200,
	// and spawns no message goroutines (so nil deps are never touched).
	w := postWebhook(t, newSignatureTestHandler(secret), body, sign(body, secret))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "received")
}

func TestReceive_EmptySecretSkipsValidation(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account","entry":[]}`)

	// No secret configured (dev): validation is skipped even without a header.
	w := postWebhook(t, newSignatureTestHandler(""), body, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "received")
}
