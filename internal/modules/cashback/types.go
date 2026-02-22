package cashback

import "time"

type CashbackProgram struct {
	ID           string
	CustomerID   string
	Type         string
	Name         string
	CashbackRate float64
	Active       bool
}

type CashbackTransaction struct {
	ID                    string
	ClientID              string
	ProgramID             string
	CollaboratorID        string
	Type                  string // "earn", "burn", "adjustment"
	Amount                float64
	PurchaseAmount        float64
	BalanceAfter          float64
	InvoiceURL            string
	Description           string
	ManualEntry           bool
	CorrectionReason      string
	CorrectionEvidenceURL string
	CorrectableUntil      *time.Time
	CreatedAt             time.Time
}

type CashbackReward struct {
	ID          string
	CustomerID  string
	ProgramID   string
	Name        string
	Description string
	Cost        float64
	Active      bool
}

type CashbackRedemption struct {
	ID          string
	ClientID    string
	RewardID    string
	ProgramID   string
	Code        string
	Status      string // "pending", "confirmed", "expired", "cancelled"
	AmountSpent float64
	ConfirmedBy string
	ExpiresAt   time.Time
	ConfirmedAt *time.Time
	CreatedAt   time.Time
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
