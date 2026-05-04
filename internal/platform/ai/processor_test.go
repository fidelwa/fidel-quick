package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockDownloader struct {
	data []byte
	err  error
}

func (m *mockDownloader) DownloadMedia(_ context.Context, _ string) ([]byte, error) {
	return m.data, m.err
}

type mockStorage struct {
	returnURL string
	err       error
	uploaded  []uploadCall
}

type uploadCall struct {
	key         string
	contentType string
	dataLen     int
}

func (m *mockStorage) Upload(_ context.Context, key string, data []byte, contentType string) (string, error) {
	m.uploaded = append(m.uploaded, uploadCall{key: key, contentType: contentType, dataLen: len(data)})
	return m.returnURL, m.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- Tests ---

func TestInvoiceProcessor_ProcessPhoto_FullPipeline(t *testing.T) {
	// JPEG magic bytes
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	jpegData = append(jpegData, make([]byte, 508)...) // pad to 512 for DetectContentType

	invoice := InvoiceResult{Total: 250.00, Currency: "MXN", Confident: true}
	invoiceJSON, _ := json.Marshal(invoice)

	geminiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{Content: geminiContent{Parts: []geminiPart{{Text: string(invoiceJSON)}}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer geminiServer.Close()

	gemini := &GeminiClient{
		apiKey:     "test",
		model:      "gemini-2.0-flash",
		httpClient: geminiServer.Client(),
	}
	gemini.httpClient.Transport = &rewriteTransport{baseURL: geminiServer.URL}

	storage := &mockStorage{returnURL: "loyalty-invoices/invoices/2025-01-15/abc.jpg"}

	processor := NewInvoiceProcessor(
		&mockDownloader{data: jpegData},
		gemini,
		storage,
		testLogger(),
	)

	result, err := processor.ProcessPhoto(context.Background(), "https://wa.media/image123")

	require.NoError(t, err)
	assert.Equal(t, 250.00, result.Amount)
	assert.Equal(t, "MXN", result.Currency)
	assert.Equal(t, "loyalty-invoices/invoices/2025-01-15/abc.jpg", result.StorageURL)
	assert.True(t, result.Invoice.Confident)

	// Verify S3 upload was called
	require.Len(t, storage.uploaded, 1)
	assert.True(t, strings.HasPrefix(storage.uploaded[0].key, "invoices/"))
	assert.Equal(t, "image/jpeg", storage.uploaded[0].contentType)
}

func TestInvoiceProcessor_ProcessPhoto_DownloadFails(t *testing.T) {
	processor := NewInvoiceProcessor(
		&mockDownloader{err: fmt.Errorf("network timeout")},
		NewGeminiClient("key"),
		&mockStorage{},
		testLogger(),
	)

	result, err := processor.ProcessPhoto(context.Background(), "https://wa.media/img")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "download image")
}

func TestInvoiceProcessor_ProcessPhoto_GeminiFails_StillUploads(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	jpegData = append(jpegData, make([]byte, 508)...)

	geminiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer geminiServer.Close()

	gemini := &GeminiClient{
		apiKey:     "test",
		model:      "gemini-2.0-flash",
		httpClient: geminiServer.Client(),
	}
	gemini.httpClient.Transport = &rewriteTransport{baseURL: geminiServer.URL}

	storage := &mockStorage{returnURL: "bucket/invoices/fallback.jpg"}

	processor := NewInvoiceProcessor(
		&mockDownloader{data: jpegData},
		gemini,
		storage,
		testLogger(),
	)

	result, err := processor.ProcessPhoto(context.Background(), "https://wa.media/img")

	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Amount)
	assert.False(t, result.Invoice.Confident)
	assert.Equal(t, "bucket/invoices/fallback.jpg", result.StorageURL)
	require.Len(t, storage.uploaded, 1)
}

func TestInvoiceProcessor_ProcessPhoto_StorageFails(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	jpegData = append(jpegData, make([]byte, 508)...)

	geminiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{Content: geminiContent{Parts: []geminiPart{{Text: `{"total":100,"currency":"MXN","confident":true}`}}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer geminiServer.Close()

	gemini := &GeminiClient{
		apiKey:     "test",
		model:      "gemini-2.0-flash",
		httpClient: geminiServer.Client(),
	}
	gemini.httpClient.Transport = &rewriteTransport{baseURL: geminiServer.URL}

	processor := NewInvoiceProcessor(
		&mockDownloader{data: jpegData},
		gemini,
		&mockStorage{err: fmt.Errorf("s3 connection refused")},
		testLogger(),
	)

	result, err := processor.ProcessPhoto(context.Background(), "https://wa.media/img")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "upload image")
}

func TestExtensionFromMIME(t *testing.T) {
	tests := []struct {
		mime string
		ext  string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/webp", ".webp"},
		{"application/octet-stream", ".bin"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.ext, extensionFromMIME(tt.mime), "MIME: %s", tt.mime)
	}
}
