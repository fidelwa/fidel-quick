package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

// Client processes ticket photos using the Claude API.
// This is the ONLY use of AI in the system — no conversational AI.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// ExtractAmountFromPhoto sends a ticket image URL to Claude and extracts the total amount.
func (c *Client) ExtractAmountFromPhoto(ctx context.Context, imageURL string) (*PhotoResult, error) {
	body := map[string]interface{}{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": 256,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type": "url",
							"url":  imageURL,
						},
					},
					{
						"type": "text",
						"text": "Extract the total amount from this receipt/ticket. Respond ONLY with a JSON object: {\"amount\": 1234.56, \"currency\": \"MXN\", \"confident\": true}. If you cannot read the total, respond with {\"amount\": 0, \"currency\": \"\", \"confident\": false}.",
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPI, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call anthropic api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	return parseResponse(respBody)
}

func parseResponse(respBody []byte) (*PhotoResult, error) {
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal api response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return &PhotoResult{Confident: false}, nil
	}

	text := apiResp.Content[0].Text

	// Try to parse the JSON from the response
	var result struct {
		Amount    json.Number `json:"amount"`
		Currency  string      `json:"currency"`
		Confident bool        `json:"confident"`
	}

	// Find JSON in the response text
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		jsonStr := text[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			amount, _ := strconv.ParseFloat(result.Amount.String(), 64)
			return &PhotoResult{
				Amount:    amount,
				Currency:  result.Currency,
				Confident: result.Confident,
				RawText:   text,
			}, nil
		}
	}

	return &PhotoResult{Confident: false, RawText: text}, nil
}
