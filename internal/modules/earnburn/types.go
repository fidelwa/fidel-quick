package earnburn

import "time"

type Transaction struct {
	ID                    string
	ClientID              string
	ProgramID             string
	CollaboratorID        string
	Type                  string // "earn", "burn", "adjustment"
	Amount                int
	BalanceAfter          int
	InvoiceURL            string
	Description           string
	ManualEntry           bool
	CorrectionReason      string
	CorrectionEvidenceURL string
	CorrectableUntil      *time.Time
	CreatedAt             time.Time
}

type Reward struct {
	ID          string
	CustomerID  string
	ProgramID   string
	Name        string
	Description string
	PointsCost  int
	Active      bool
}

type Redemption struct {
	ID          string
	ClientID    string
	RewardID    string
	ProgramID   string
	Code        string
	Status      string // "pending", "confirmed", "expired", "cancelled"
	PointsSpent int
	ConfirmedBy string
	ExpiresAt   time.Time
	ConfirmedAt *time.Time
	CreatedAt   time.Time
}

type Program struct {
	ID          string
	CustomerID  string
	Type        string
	Name        string
	PointsRatio int
	Active      bool
}

type AddPointsReq struct {
	ClientID       string
	ProgramID      string
	CollaboratorID string
	Amount         float64 // purchase amount in currency
	InvoiceURL     string
	ManualEntry    bool
}

type UpdatePointsReq struct {
	TransactionID         string
	CollaboratorID        string
	NewAmount             int // new points amount
	CorrectionReason      string
	CorrectionEvidenceURL string
}

type RedemptionReq struct {
	ClientID  string
	ProgramID string
	RewardID  string
}

type LoadPointsReq struct {
	ClientID       string
	ProgramID      string
	CollaboratorID string
	Amount         float64 // purchase amount in currency
	InvoiceURL     string
}

type Customer struct {
	ID             string
	Name           string
	Slug           string
	Phone          string
	Address        string
	LogoURL        string
	Description    string
	WelcomeMessage string
	Active         bool
}

type Collaborator struct {
	ID         string
	CustomerID string
	Name       string
	Phone      string
	HashID     string
	Active     bool
}

type FeedbackEntry struct {
	ID         string
	Message    string
	ClientName string
	CreatedAt  time.Time
}
