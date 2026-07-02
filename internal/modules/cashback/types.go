package cashback

import (
	"errors"
	"time"

	"github.com/theluisbolivar/fidel-quick/internal/platform/ai"
)

// CashbackProgram is wire-serialized with `id` (mapped from the customer_sisfi
// row id) so the frontend can use the same `id` field across all program types.
type CashbackProgram struct {
	CustomerSisfiID string  `json:"id"`
	CustomerID      string  `json:"customer_id"`
	Name            string  `json:"name"`
	CashbackRate    float64 `json:"cashback_rate"`
	Active          bool    `json:"active"`
	// ExpiryDays (FID-34): días tras los cuales el saldo de una acreditación vence.
	// nil = sin vencimiento (comportamiento por defecto).
	ExpiryDays *int `json:"expiry_days"`
	// MinTicketAmount (FID-36): monto mínimo de compra para acreditar cashback.
	// nil = sin mínimo (comportamiento por defecto).
	MinTicketAmount *float64 `json:"min_ticket_amount"`
	// MaxCashbackPerTx (FID-37): techo de cashback por transacción.
	// nil = sin cap (comportamiento por defecto).
	MaxCashbackPerTx *float64 `json:"max_cashback_per_tx"`
	// MaxCashbackPerPeriod (FID-37): techo de cashback acumulado en la ventana de
	// expiry_days (o el mes calendario si expiry_days es nil).
	// nil = sin cap (comportamiento por defecto).
	MaxCashbackPerPeriod *float64 `json:"max_cashback_per_period"`
}

type CashbackTransaction struct {
	ID                    string     `json:"id"`
	ClientID              string     `json:"client_id"`
	CustomerSisfiID       string     `json:"customer_sisfi_id"`
	CollaboratorID        string     `json:"collaborator_id"`
	Type                  string     `json:"type"` // "earn", "burn", "adjustment"
	Amount                float64    `json:"amount"`
	PurchaseAmount        float64    `json:"purchase_amount"`
	BalanceAfter          float64    `json:"balance_after"`
	InvoiceURL            string     `json:"invoice_url"`
	Description           string     `json:"description"`
	ManualEntry           bool       `json:"manual_entry"`
	CorrectionReason      string     `json:"correction_reason"`
	CorrectionEvidenceURL string     `json:"correction_evidence_url"`
	CorrectableUntil      *time.Time `json:"correctable_until"`
	CreatedAt             time.Time  `json:"created_at"`

	// Anti-fraude (FID-33): huella del ticket. ReceiptHash es "" cuando no se
	// pudo computar un hash confiable (folio ausente o extract no confiable).
	ReceiptData       []byte   `json:"-"`
	ReceiptHash       string   `json:"receipt_hash,omitempty"`
	ReceiptHashFields []string `json:"receipt_hash_fields,omitempty"`
	ReceiptConfident  bool     `json:"receipt_confident,omitempty"`

	// FID-37 (cap por periodo, LG-2): parámetros transitorios para que
	// AddCashbackTx aplique el techo por periodo DENTRO de la misma transacción
	// (lectura de la ventana + clamp + insert atómicos), evitando la carrera de
	// dos requests concurrentes que exceden el cap. No se persisten.
	// PeriodCap nil = sin cap por periodo; PeriodWindowDays es la ventana en días.
	PeriodCap        *float64 `json:"-"`
	PeriodWindowDays int      `json:"-"`
}

// ErrPeriodCapExhausted (FID-37) se devuelve cuando el techo por periodo ya se
// alcanzó al momento de acreditar dentro de la transacción (chequeo atómico).
var ErrPeriodCapExhausted = errors.New("se alcanzó el máximo de cashback del periodo")

type CashbackReward struct {
	ID              string  `json:"id"`
	CustomerID      string  `json:"customer_id"`
	CustomerSisfiID string  `json:"customer_sisfi_id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Cost            float64 `json:"cost"`
	Active          bool    `json:"active"`

	// FID-38 (stock): disponibilidad limitada del premio.
	// StockTotal nil = stock ilimitado (comportamiento por defecto).
	// RedeemedCount es cuántas unidades ya se canjearon; se incrementa de forma
	// atómica dentro de la transacción del burn.
	// LimitPerClient nil = sin límite por cliente (reservado para uso futuro).
	StockTotal     *int `json:"stock_total"`
	RedeemedCount  int  `json:"redeemed_count"`
	LimitPerClient *int `json:"limit_per_client"`
}

type CashbackRedemption struct {
	ID              string     `json:"id"`
	ClientID        string     `json:"client_id"`
	RewardID        string     `json:"reward_id"`
	CustomerSisfiID string     `json:"customer_sisfi_id"`
	Code            string     `json:"code"`
	Status          string     `json:"status"` // "pending", "confirmed", "expired", "cancelled"
	AmountSpent     float64    `json:"amount_spent"`
	ConfirmedBy     string     `json:"confirmed_by"`
	ExpiresAt       time.Time  `json:"expires_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type AddCashbackReq struct {
	ClientID        string
	CustomerSisfiID string
	CollaboratorID  string
	Amount          float64 // purchase amount in currency
	InvoiceURL      string
	ManualEntry     bool
	// Invoice is the full AI extract of the receipt (may be nil). When present it
	// is persisted and used to compute the dedup hash.
	Invoice *ai.InvoiceResult
}

type UpdateCashbackReq struct {
	TransactionID         string
	CollaboratorID        string
	NewPurchaseAmount     float64 // new invoice amount — cashback is recalculated
	CorrectionReason      string
	CorrectionEvidenceURL string
}

type CashbackRedemptionReq struct {
	ClientID        string
	CustomerSisfiID string
	RewardID        string
}
