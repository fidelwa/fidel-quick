package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGeminiResponse_ValidInvoice(t *testing.T) {
	invoice := InvoiceResult{
		Total:        325.50,
		Subtotal:     280.60,
		Tax:          44.90,
		Currency:     "MXN",
		BusinessName: "Tacos El Paisa",
		BusinessRFC:  "TEP123456ABC",
		Date:         "2025-01-15",
		Confident:    true,
		Items: []InvoiceItem{
			{Description: "Tacos al pastor x5", Quantity: 5, UnitPrice: 25.00, Total: 125.00},
			{Description: "Refrescos", Quantity: 3, UnitPrice: 35.00, Total: 105.00},
		},
	}

	invoiceJSON, _ := json.Marshal(invoice)
	resp := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: "Here is the invoice data:\n" + string(invoiceJSON)},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(resp)
	result, err := parseGeminiResponse(body)

	require.NoError(t, err)
	assert.True(t, result.Confident)
	assert.Equal(t, 325.50, result.Total)
	assert.Equal(t, 280.60, result.Subtotal)
	assert.Equal(t, 44.90, result.Tax)
	assert.Equal(t, "MXN", result.Currency)
	assert.Equal(t, "Tacos El Paisa", result.BusinessName)
	assert.Equal(t, "TEP123456ABC", result.BusinessRFC)
	assert.Len(t, result.Items, 2)
}

func TestParseGeminiResponse_NoCandidates(t *testing.T) {
	resp := geminiResponse{Candidates: nil}
	body, _ := json.Marshal(resp)

	result, err := parseGeminiResponse(body)

	require.NoError(t, err)
	assert.False(t, result.Confident)
}

func TestParseGeminiResponse_NoJSON(t *testing.T) {
	resp := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: "I cannot read this image clearly"},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(resp)
	result, err := parseGeminiResponse(body)

	require.NoError(t, err)
	assert.False(t, result.Confident)
}

func TestParseGeminiResponse_InvalidJSON(t *testing.T) {
	resp := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: "Here is the data: {invalid json}"},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(resp)
	result, err := parseGeminiResponse(body)

	require.NoError(t, err)
	assert.False(t, result.Confident)
}

func TestGeminiClient_AnalyzeInvoice_Success(t *testing.T) {
	invoice := InvoiceResult{
		Total:    150.00,
		Currency: "MXN",
		Confident: true,
	}
	invoiceJSON, _ := json.Marshal(invoice)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "gemini-2.0-flash")

		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{Content: geminiContent{Parts: []geminiPart{{Text: string(invoiceJSON)}}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "test-key",
		model:      "gemini-2.0-flash",
		httpClient: server.Client(),
	}

	// Override the URL by using a custom transport
	originalURL := server.URL
	client.httpClient.Transport = &rewriteTransport{
		base:    server.Client().Transport,
		baseURL: originalURL,
	}

	result, err := client.AnalyzeInvoice(context.Background(), []byte("fake-image-data"), "image/jpeg")

	require.NoError(t, err)
	assert.True(t, result.Confident)
	assert.Equal(t, 150.00, result.Total)
}

func TestGeminiClient_AnalyzeInvoice_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	client := &GeminiClient{
		apiKey:     "bad-key",
		model:      "gemini-2.0-flash",
		httpClient: server.Client(),
	}
	client.httpClient.Transport = &rewriteTransport{
		base:    server.Client().Transport,
		baseURL: server.URL,
	}

	result, err := client.AnalyzeInvoice(context.Background(), []byte("fake"), "image/jpeg")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "gemini API error 401")
}

// rewriteTransport redirects all requests to the test server.
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.baseURL[len("http://"):]
	if t.base != nil {
		return t.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
