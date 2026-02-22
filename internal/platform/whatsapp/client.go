package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const metaAPIBase = "https://graph.facebook.com/v21.0"

type Client struct {
	apiToken   string
	phoneID    string
	httpClient *http.Client
}

func NewClient(apiToken, phoneID string) *Client {
	return &Client{
		apiToken:   apiToken,
		phoneID:    phoneID,
		httpClient: &http.Client{},
	}
}

// SendText sends a plain text message.
func (c *Client) SendText(ctx context.Context, to, text string) error {
	req := SendTextRequest{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text:             &TextBody{Body: text},
	}
	return c.send(ctx, req)
}

// SendInteractiveList sends a WhatsApp interactive list message.
func (c *Client) SendInteractiveList(ctx context.Context, to, header, body string, options []ListOption) error {
	rows := make([]OutRow, len(options))
	for i, opt := range options {
		rows[i] = OutRow{
			ID:          opt.ID,
			Title:       truncate(opt.Title, 24),
			Description: truncate(opt.Description, 72),
		}
	}

	req := SendTextRequest{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "interactive",
		Interactive: &OutInteractive{
			Type: "list",
			Header: &OutHeader{
				Type: "text",
				Text: truncate(header, 60),
			},
			Body: OutBody{Text: body},
			Action: OutAction{
				Button: "Ver opciones",
				Sections: []OutSection{
					{Title: "Opciones", Rows: rows},
				},
			},
		},
	}
	return c.send(ctx, req)
}

// GetMediaURL retrieves the download URL for a media file.
func (c *Client) GetMediaURL(ctx context.Context, mediaID string) (string, error) {
	url := fmt.Sprintf("%s/%s", metaAPIBase, mediaID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get media url: %w", err)
	}
	defer resp.Body.Close()

	var result MediaURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode media url: %w", err)
	}
	return result.URL, nil
}

// DownloadMedia downloads a media file from the given URL.
func (c *Client) DownloadMedia(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (c *Client) send(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/messages", metaAPIBase, c.phoneID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WhatsApp API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ListOption matches the flow engine's ListOption type.
type ListOption struct {
	ID          string
	Title       string
	Description string
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
