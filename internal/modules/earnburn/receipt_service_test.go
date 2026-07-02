package earnburn

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"github.com/theluisbolivar/fidel-quick/internal/platform/ai"
)

func confidentInvoice() *ai.InvoiceResult {
	return &ai.InvoiceResult{
		Total:         200,
		Currency:      "MXN",
		BusinessName:  "Café Central",
		BusinessRFC:   "GODE561231GR8",
		InvoiceNumber: "A-000123",
		Date:          "2026-06-25",
		Confident:     true,
	}
}

// AddPoints must compute and forward the receipt fingerprint to the repository.
func TestAddPoints_PersistsReceiptFingerprint(t *testing.T) {
	var captured *Transaction
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10}, nil
		},
		addPointsTxFn: func(_ context.Context, tx *Transaction) (int, error) {
			captured = tx
			return tx.Amount, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.AddPoints(context.Background(), AddPointsReq{
		ClientID:        "client-1",
		CustomerSisfiID: "cs-1",
		Amount:          200,
		Invoice:         confidentInvoice(),
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.NotEmpty(t, captured.ReceiptHash, "expected a receipt hash for a confident invoice with folio")
	assert.NotEmpty(t, captured.ReceiptData, "expected the full extract to be persisted")
	assert.True(t, captured.ReceiptConfident)
	assert.Equal(t, []string{"business_rfc", "invoice_number", "date", "total"}, captured.ReceiptHashFields)
}

// A duplicate receipt (unique-index violation surfaced as ErrDuplicateReceipt)
// must be rejected with a typed Conflict error and must NOT credit points.
func TestAddPoints_RejectsDuplicateReceipt(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10}, nil
		},
		addPointsTxFn: func(_ context.Context, _ *Transaction) (int, error) {
			return 0, ErrDuplicateReceipt
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.AddPoints(context.Background(), AddPointsReq{
		ClientID:        "client-1",
		CustomerSisfiID: "cs-1",
		Amount:          200,
		Invoice:         confidentInvoice(),
	})
	require.Error(t, err)

	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr), "expected an apperror.AppError")
	assert.Equal(t, "conflict", appErr.Code)
	assert.Contains(t, appErr.Message, "ticket ya registrado")
}

// Without a reliable extract (no folio) the hash is left empty and points are
// still credited (provisional policy — pending Pablo).
func TestAddPoints_NoHashWhenMissingFolioButStillCredits(t *testing.T) {
	var captured *Transaction
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*EarnBurnProgram, error) {
			return &EarnBurnProgram{CustomerSisfiID: "cs-1", PointsRatio: 10}, nil
		},
		addPointsTxFn: func(_ context.Context, tx *Transaction) (int, error) {
			captured = tx
			return tx.Amount, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	inv := confidentInvoice()
	inv.InvoiceNumber = "" // missing folio → no hash

	tx, err := svc.AddPoints(context.Background(), AddPointsReq{
		ClientID:        "client-1",
		CustomerSisfiID: "cs-1",
		Amount:          200,
		Invoice:         inv,
	})
	require.NoError(t, err)
	assert.Equal(t, 20, tx.Amount) // still credited
	require.NotNil(t, captured)
	assert.Empty(t, captured.ReceiptHash, "no hash without a folio")
	assert.NotEmpty(t, captured.ReceiptData, "extract still persisted for auditing")
}
