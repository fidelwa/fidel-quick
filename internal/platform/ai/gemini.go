package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const invoicePrompt = `Analiza esta imagen de factura/ticket de compra. Extrae TODOS los datos posibles.
Responde SOLO con un JSON object con esta estructura:
{
  "total": 0.00, "subtotal": 0.00, "tax": 0.00, "tip": 0.00,
  "currency": "MXN",
  "business_name": "", "business_rfc": "", "business_address": "",
  "invoice_number": "", "date": "", "payment_method": "",
  "items": [{"description": "", "quantity": 1, "unit_price": 0.00, "total": 0.00}],
  "confident": true
}
Si no puedes leer algun campo, dejalo vacio o en 0. El campo "confident" indica
si pudiste extraer al menos el total correctamente.`

// GeminiClient processes invoice photos using the Gemini API.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewGeminiClient creates a client for Gemini vision API.
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:     apiKey,
		model:      "gemini-2.5-flash",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// AnalyzeInvoice sends image bytes to Gemini and extracts invoice data.
func (c *GeminiClient) AnalyzeInvoice(ctx context.Context, imageData []byte, mimeType string) (*InvoiceResult, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	body := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{
						InlineData: &geminiInlineData{
							MimeType: mimeType,
							Data:     base64.StdEncoding.EncodeToString(imageData),
						},
					},
					{
						Text: invoicePrompt,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal gemini request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call gemini api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read gemini response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gemini API error %d: %s", resp.StatusCode, string(respBody))
	}

	return parseGeminiResponse(respBody)
}

func parseGeminiResponse(respBody []byte) (*InvoiceResult, error) {
	var apiResp geminiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal gemini response: %w", err)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return &InvoiceResult{Confident: false, RawResponse: string(respBody)}, nil
	}

	text := apiResp.Candidates[0].Content.Parts[0].Text

	// Find JSON in response text
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return &InvoiceResult{Confident: false, RawResponse: text}, nil
	}

	var result InvoiceResult
	if err := json.Unmarshal([]byte(text[start:end+1]), &result); err != nil {
		return &InvoiceResult{Confident: false, RawResponse: text}, nil
	}

	result.RawResponse = text
	return &result, nil
}

// --- Gemini API types ---

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}
