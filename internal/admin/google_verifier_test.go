package admin

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIdP serves a JWKS document and signs ID tokens for verifier tests.
type fakeIdP struct {
	priv   *rsa.PrivateKey
	kid    string
	server *httptest.Server
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeIdP{priv: priv, kid: "test-kid-1"}
	mux := http.NewServeMux()
	mux.HandleFunc("/certs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kid": idp.kid,
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(idp.priv.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(idp.priv.PublicKey.E)).Bytes()),
			}},
		})
	})
	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	return idp
}

func (f *fakeIdP) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = f.kid
	s, err := tok.SignedString(f.priv)
	require.NoError(t, err)
	return s
}

func newTestVerifier(idp *fakeIdP, clientID string) *googleVerifier {
	return &googleVerifier{
		clientID: clientID,
		jwksURL:  idp.server.URL + "/certs",
		httpc:    &http.Client{Timeout: 2 * time.Second},
		keys:     map[string]*rsa.PublicKey{},
	}
}

func goodClaims(aud string) jwt.MapClaims {
	return jwt.MapClaims{
		"iss":            "https://accounts.google.com",
		"aud":            aud,
		"sub":            "google-sub-123",
		"email":          "user@example.com",
		"email_verified": true,
		"exp":            time.Now().Add(10 * time.Minute).Unix(),
		"iat":            time.Now().Unix(),
	}
}

func TestVerifier_ClientIDEmpty_Fails(t *testing.T) {
	v := NewGoogleVerifier("")
	_, _, err := v.Verify("any")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestVerifier_Success(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	tok := idp.sign(t, goodClaims("my-client-id"))
	email, sub, err := v.Verify(tok)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", email)
	assert.Equal(t, "google-sub-123", sub)
}

func TestVerifier_AlternateIssuer(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	claims := goodClaims("my-client-id")
	claims["iss"] = "accounts.google.com"
	tok := idp.sign(t, claims)
	_, _, err := v.Verify(tok)
	require.NoError(t, err)
}

func TestVerifier_AudienceMismatch(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	tok := idp.sign(t, goodClaims("other-client-id"))
	_, _, err := v.Verify(tok)
	require.Error(t, err)
}

func TestVerifier_EmailNotVerified(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	claims := goodClaims("my-client-id")
	claims["email_verified"] = false
	tok := idp.sign(t, claims)
	_, _, err := v.Verify(tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email not verified")
}

func TestVerifier_Expired(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	claims := goodClaims("my-client-id")
	claims["exp"] = time.Now().Add(-10 * time.Minute).Unix()
	tok := idp.sign(t, claims)
	_, _, err := v.Verify(tok)
	require.Error(t, err)
}

func TestVerifier_BadIssuer(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	claims := goodClaims("my-client-id")
	claims["iss"] = "https://evil.example.com"
	tok := idp.sign(t, claims)
	_, _, err := v.Verify(tok)
	require.Error(t, err)
}

func TestVerifier_UnknownKid(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, goodClaims("my-client-id"))
	tok.Header["kid"] = "unknown-kid"
	signed, err := tok.SignedString(idp.priv)
	require.NoError(t, err)

	_, _, err = v.Verify(signed)
	require.Error(t, err)
}

func TestVerifier_DifferentSigningKey(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	// Sign with a key that is NOT in the JWKS, but use the published kid
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, goodClaims("my-client-id"))
	tok.Header["kid"] = idp.kid
	signed, err := tok.SignedString(other)
	require.NoError(t, err)

	_, _, err = v.Verify(signed)
	require.Error(t, err)
}

func TestVerifier_HS256Rejected(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, goodClaims("my-client-id"))
	tok.Header["kid"] = idp.kid
	signed, err := tok.SignedString([]byte("shared-secret"))
	require.NoError(t, err)

	_, _, err = v.Verify(signed)
	require.Error(t, err)
}

func TestVerifier_MissingSub(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	claims := goodClaims("my-client-id")
	delete(claims, "sub")
	tok := idp.sign(t, claims)
	_, _, err := v.Verify(tok)
	require.Error(t, err)
}

func TestVerifier_KeyRotation(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")

	// Warm cache with current kid.
	tok := idp.sign(t, goodClaims("my-client-id"))
	_, _, err := v.Verify(tok)
	require.NoError(t, err)

	// Rotate: new key + new kid.
	newKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	idp.priv = newKey
	idp.kid = "test-kid-2"

	tok2 := idp.sign(t, goodClaims("my-client-id"))
	_, _, err = v.Verify(tok2)
	require.NoError(t, err, "verifier should refresh JWKS when kid is unknown")
}

func TestVerifier_JWKSUnreachable(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := &googleVerifier{
		clientID: "x",
		jwksURL:  "http://127.0.0.1:1/non-existent",
		httpc:    &http.Client{Timeout: 100 * time.Millisecond},
		keys:     map[string]*rsa.PublicKey{},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, goodClaims("x"))
	tok.Header["kid"] = "any"
	signed, _ := tok.SignedString(priv)

	_, _, err := v.Verify(signed)
	require.Error(t, err)
}

func TestVerifier_MalformedToken(t *testing.T) {
	idp := newFakeIdP(t)
	v := newTestVerifier(idp, "my-client-id")
	_, _, err := v.Verify("not-a-jwt")
	require.Error(t, err)
}

// ensure helper compiles with the package's expected types
func TestRsaPublicKey_Roundtrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	n := base64.RawURLEncoding.EncodeToString(priv.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.PublicKey.E)).Bytes())

	pk, err := rsaPublicKey(n, e)
	require.NoError(t, err)
	assert.Equal(t, priv.PublicKey.N, pk.N)
	assert.Equal(t, priv.PublicKey.E, pk.E)
	_ = fmt.Sprintf("%v", pk) // appease unused import in some toolchains
}
