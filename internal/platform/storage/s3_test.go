package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient devuelve un S3Client apuntando a un servidor httptest que
// hace de endpoint S3-compatible (como GCS o MinIO local). No usa SSL para
// que el firmado y el PUT vayan al host del test server.
func newTestClient(t *testing.T, srv *httptest.Server) *S3Client {
	t.Helper()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	c, err := NewS3Client(u.Host, "AKIATESTKEY", "testsecret", "invoices-bucket", "us-east-1", false)
	require.NoError(t, err)
	return c
}

// assertPresigned valida que la URL sea una presigned URL AWS SigV4 con TTL 1h.
func assertPresigned(t *testing.T, raw string) {
	t.Helper()

	u, err := url.Parse(raw)
	require.NoError(t, err, "la URL devuelta debe ser válida")

	q := u.Query()

	// La signature (criterio 2): sin ella la URL no autoriza nada.
	assert.NotEmpty(t, q.Get("X-Amz-Signature"), "la URL debe incluir X-Amz-Signature")
	assert.NotEmpty(t, q.Get("X-Amz-Credential"), "la URL debe incluir X-Amz-Credential")
	assert.NotEmpty(t, q.Get("X-Amz-Date"), "la URL debe incluir X-Amz-Date")

	// El TTL de 1h (criterio 1): minio-go serializa presignedURLTTL en segundos.
	assert.Equal(t, "3600", q.Get("X-Amz-Expires"),
		"la URL solo debe ser válida durante el TTL de 1h")
}

func TestUpload_ReturnsPresignedURLWithTTL(t *testing.T) {
	var gotPut bool
	var gotKey string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// El upload es un PUT del objeto; el firmado no golpea la red.
		if r.Method == http.MethodPut {
			gotPut = true
			gotKey = r.URL.Path
			w.Header().Set("ETag", `"deadbeef"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	signed, err := c.Upload(context.Background(), "invoices/2026-01-01/abc.jpg", []byte("fake-image-bytes"), "image/jpeg")
	require.NoError(t, err)

	assert.True(t, gotPut, "Upload debe subir el objeto vía PutObject")
	assert.Contains(t, gotKey, "invoices/2026-01-01/abc.jpg")

	// Criterio 1 y 2: presigned URL vía PresignedGetObject, con signature y TTL 1h.
	assertPresigned(t, signed)
	assert.True(t, strings.Contains(signed, "invoices/2026-01-01/abc.jpg"),
		"la URL firmada debe apuntar a la key subida")
}

func TestPresignKey_ReturnsPresignedURLWithTTL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	// PresignKey no debe golpear la red (el firmado es local): re-firma una key existente.
	signed, err := c.PresignKey(context.Background(), "invoices/2026-01-01/old.png")
	require.NoError(t, err)

	assertPresigned(t, signed)
	assert.Contains(t, signed, "invoices/2026-01-01/old.png")
}
