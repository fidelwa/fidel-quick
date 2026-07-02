package cashback

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

// AddCashback must compute and forward the receipt fingerprint to the repository.
func TestAddCashback_PersistsReceiptFingerprint(t *testing.T) {
	var captured *CashbackTransaction
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*CashbackProgram, error) {
			return &CashbackProgram{CustomerSisfiID: "cs-1", CashbackRate: 0.05}, nil
		},
		addCashbackTxFn: func(_ context.Context, tx *CashbackTransaction) (float64, error) {
			captured = tx
			return tx.Amount, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.AddCashback(context.Background(), AddCashbackReq{
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
// must be rejected with a typed Conflict error and must NOT credit cashback.
func TestAddCashback_RejectsDuplicateReceipt(t *testing.T) {
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*CashbackProgram, error) {
			return &CashbackProgram{CustomerSisfiID: "cs-1", CashbackRate: 0.05}, nil
		},
		addCashbackTxFn: func(_ context.Context, _ *CashbackTransaction) (float64, error) {
			return 0, ErrDuplicateReceipt
		},
	}
	svc := newTestService(repo, newMockCache())

	_, err := svc.AddCashback(context.Background(), AddCashbackReq{
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

// Without a reliable extract (no folio) the hash is left empty and cashback is
// still credited (provisional policy — pending Pablo).
func TestAddCashback_NoHashWhenMissingFolioButStillCredits(t *testing.T) {
	var captured *CashbackTransaction
	repo := &mockRepo{
		getProgramFn: func(_ context.Context, _ string) (*CashbackProgram, error) {
			return &CashbackProgram{CustomerSisfiID: "cs-1", CashbackRate: 0.05}, nil
		},
		addCashbackTxFn: func(_ context.Context, tx *CashbackTransaction) (float64, error) {
			captured = tx
			return tx.Amount, nil
		},
	}
	svc := newTestService(repo, newMockCache())

	inv := confidentInvoice()
	inv.InvoiceNumber = "" // missing folio → no hash

	tx, err := svc.AddCashback(context.Background(), AddCashbackReq{
		ClientID:        "client-1",
		CustomerSisfiID: "cs-1",
		Amount:          200,
		Invoice:         inv,
	})
	require.NoError(t, err)
	assert.Equal(t, 10.0, tx.Amount) // still credited: 200 * 0.05
	require.NotNil(t, captured)
	assert.Empty(t, captured.ReceiptHash, "no hash without a folio")
	assert.NotEmpty(t, captured.ReceiptData, "extract still persisted for auditing")
}
