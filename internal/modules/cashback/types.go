package cashback

import "time"

type CashbackProgram struct {
	ID           string  `json:"id"`
	CustomerID   string  `json:"customer_id"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	CashbackRate float64 `json:"cashback_rate"`
	Active       bool    `json:"active"`
}

type CashbackTransaction struct {
	ID                    string     `json:"id"`
	ClientID              string     `json:"client_id"`
	ProgramID             string     `json:"program_id"`
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
}

type CashbackReward struct {
	ID          string  `json:"id"`
	CustomerID  string  `json:"customer_id"`
	ProgramID   string  `json:"program_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
	Active      bool    `json:"active"`
}

type CashbackRedemption struct {
	ID          string     `json:"id"`
	ClientID    string     `json:"client_id"`
	RewardID    string     `json:"reward_id"`
	ProgramID   string     `json:"program_id"`
	Code        string     `json:"code"`
	Status      string     `json:"status"` // "pending", "confirmed", "expired", "cancelled"
	AmountSpent float64    `json:"amount_spent"`
	ConfirmedBy string     `json:"confirmed_by"`
	ExpiresAt   time.Time  `json:"expires_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AddCashbackReq struct {
	ClientID       string
	ProgramID      string
	CollaboratorID string
	Amount         float64 // purchase amount in currency
	InvoiceURL     string
	ManualEntry    bool
}

type UpdateCashbackReq struct {
	TransactionID         string
	CollaboratorID        string
	NewPurchaseAmount     float64 // new invoice amount — cashback is recalculated
	CorrectionReason      string
	CorrectionEvidenceURL string
}

type CashbackRedemptionReq struct {
	ClientID  string
	ProgramID string
	RewardID  string
}
