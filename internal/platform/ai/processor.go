package ai

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// MediaDownloader downloads media from external sources (satisfied by whatsapp.Client).
type MediaDownloader interface {
	DownloadMedia(ctx context.Context, url string) ([]byte, error)
}

// ObjectStorage stores files (satisfied by storage.S3Client).
type ObjectStorage interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
}

// PhotoProcessResult is what the flow engine receives after processing.
type PhotoProcessResult struct {
	StorageURL string
	Amount     float64
	Currency   string
	Invoice    *InvoiceResult
}

// InvoiceProcessor orchestrates: download → analyze → upload.
type InvoiceProcessor struct {
	downloader MediaDownloader
	analyzer   *GeminiClient
	storage    ObjectStorage
	log        *slog.Logger
}

// NewInvoiceProcessor creates a processor that downloads images, analyzes with Gemini, and uploads to S3.
func NewInvoiceProcessor(dl MediaDownloader, ai *GeminiClient, st ObjectStorage, log *slog.Logger) *InvoiceProcessor {
	return &InvoiceProcessor{
		downloader: dl,
		analyzer:   ai,
		storage:    st,
		log:        log,
	}
}

// ProcessPhoto downloads the image, sends to Gemini for analysis, and uploads to S3.
func (p *InvoiceProcessor) ProcessPhoto(ctx context.Context, imageURL string) (*PhotoProcessResult, error) {
	// 1. Download image from WhatsApp
	data, err := p.downloader.DownloadMedia(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}

	// 2. Detect MIME type
	mimeType := http.DetectContentType(data)

	// 3. Analyze with Gemini
	invoice, err := p.analyzer.AnalyzeInvoice(ctx, data, mimeType)
	if err != nil {
		p.log.Warn("gemini analysis failed, uploading image without analysis", "error", err)
		invoice = &InvoiceResult{Confident: false}
	}

	// 4. Upload to S3
	ext := extensionFromMIME(mimeType)
	key := fmt.Sprintf("invoices/%s/%s%s", time.Now().Format("2006-01-02"), uuid.New().String(), ext)

	storageURL, err := p.storage.Upload(ctx, key, data, mimeType)
	if err != nil {
		return nil, fmt.Errorf("upload image: %w", err)
	}

	p.log.Info("photo processed",
		"storage_url", storageURL,
		"total", invoice.Total,
		"subtotal", invoice.Subtotal,
		"tax", invoice.Tax,
		"tip", invoice.Tip,
		"currency", invoice.Currency,
		"confident", invoice.Confident,
		"business_name", invoice.BusinessName,
		"business_rfc", invoice.BusinessRFC,
		"business_address", invoice.BusinessAddress,
		"invoice_number", invoice.InvoiceNumber,
		"date", invoice.Date,
		"payment_method", invoice.PaymentMethod,
		"items_count", len(invoice.Items),
	)

	for i, item := range invoice.Items {
		p.log.Info("invoice item",
			"index", i,
			"description", item.Description,
			"qty", item.Quantity,
			"unit_price", item.UnitPrice,
			"total", item.Total,
		)
	}

	return &PhotoProcessResult{
		StorageURL: storageURL,
		Amount:     invoice.Total,
		Currency:   invoice.Currency,
		Invoice:    invoice,
	}, nil
}

func extensionFromMIME(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
