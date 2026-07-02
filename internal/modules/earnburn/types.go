package earnburn

import (
	"time"

	"github.com/theluisbolivar/fidel-quick/internal/platform/ai"
)

type Transaction struct {
	ID                    string     `json:"id"`
	ClientID              string     `json:"client_id"`
	CustomerSisfiID       string     `json:"customer_sisfi_id"`
	CollaboratorID        string     `json:"collaborator_id"`
	Type                  string     `json:"type"` // "earn", "burn", "adjustment"
	Amount                int        `json:"amount"`
	BalanceAfter          int        `json:"balance_after"`
	InvoiceURL            string     `json:"invoice_url"`
	Description           string     `json:"description"`
	ManualEntry           bool       `json:"manual_entry"`
	CorrectionReason      string     `json:"correction_reason"`
	CorrectionEvidenceURL string     `json:"correction_evidence_url"`
	CorrectableUntil      *time.Time `json:"correctable_until"`
	CreatedAt             time.Time  `json:"created_at"`

	// Anti-fraude (FID-33): huella del ticket. ReceiptData es el JSON del
	// extract completo; ReceiptHash es "" cuando no se pudo computar un hash
	// confiable (folio ausente o extract no confiable).
	ReceiptData       []byte   `json:"-"`
	ReceiptHash       string   `json:"receipt_hash,omitempty"`
	ReceiptHashFields []string `json:"receipt_hash_fields,omitempty"`
	ReceiptConfident  bool     `json:"receipt_confident,omitempty"`
}

type Reward struct {
	ID              string `json:"id"`
	CustomerID      string `json:"customer_id"`
	CustomerSisfiID string `json:"customer_sisfi_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	PointsCost      int    `json:"points_cost"`
	Active          bool   `json:"active"`
}

type Redemption struct {
	ID              string     `json:"id"`
	ClientID        string     `json:"client_id"`
	RewardID        string     `json:"reward_id"`
	CustomerSisfiID string     `json:"customer_sisfi_id"`
	Code            string     `json:"code"`
	Status          string     `json:"status"` // "pending", "confirmed", "expired", "cancelled"
	PointsSpent     int        `json:"points_spent"`
	ConfirmedBy     string     `json:"confirmed_by"`
	ExpiresAt       time.Time  `json:"expires_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// EarnBurnProgram is wire-serialized with `id` (mapped from the customer_sisfi
// row id) so the frontend can use the same `id` field across all program types.
type EarnBurnProgram struct {
	CustomerSisfiID string `json:"id"`
	CustomerID      string `json:"customer_id"`
	Name            string `json:"name"`
	PointsRatio     int    `json:"points_ratio"`
	Active          bool   `json:"active"`
	// ExpiryDays (FID-34): días tras los cuales los puntos de una carga vencen.
	// nil = sin vencimiento (comportamiento por defecto).
	ExpiryDays *int `json:"expiry_days"`
	// MinTicketAmount (FID-36): monto mínimo de compra para acreditar puntos.
	// nil = sin mínimo (comportamiento por defecto).
	MinTicketAmount *float64 `json:"min_ticket_amount"`
}

type AddPointsReq struct {
	ClientID        string
	CustomerSisfiID string
	CollaboratorID  string
	Amount          float64 // purchase amount in currency
	InvoiceURL      string
	ManualEntry     bool
	// Invoice is the full AI extract of the receipt (may be nil for manual/legacy
	// entries). When present it is persisted and used to compute the dedup hash.
	Invoice *ai.InvoiceResult
}

type UpdatePointsReq struct {
	TransactionID         string
	CollaboratorID        string
	NewAmount             int // new points amount
	CorrectionReason      string
	CorrectionEvidenceURL string
}

type RedemptionReq struct {
	ClientID        string
	CustomerSisfiID string
	RewardID        string
}

type LoadPointsReq struct {
	ClientID        string
	CustomerSisfiID string
	CollaboratorID  string
	Amount          float64 // purchase amount in currency
	InvoiceURL      string
}

type Customer struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	LogoURL        string `json:"logo_url"`
	Description    string `json:"description"`
	WelcomeMessage string `json:"welcome_message"`
	Active         bool   `json:"active"`
}

type Collaborator struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	HashID     string `json:"hash_id"`
	Active     bool   `json:"active"`
}

type Client struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Name       string    `json:"name"`
	Phone      string    `json:"phone"`
	CreatedAt  time.Time `json:"created_at"`
}

type FeedbackEntry struct {
	ID         string    `json:"id"`
	Message    string    `json:"message"`
	ClientName string    `json:"client_name"`
	CreatedAt  time.Time `json:"created_at"`
}
