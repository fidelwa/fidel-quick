package pushcard

import "time"

// Config is the per-customer_sisfi configuration of the pushcard module.
type Config struct {
	CustomerSisfiID  string `json:"customer_sisfi_id"`
	CustomerID       string `json:"customer_id"`
	Name             string `json:"name"`
	CardSlots        int    `json:"card_slots"`
	RewardOnComplete string `json:"reward_on_complete"`
	Active           bool   `json:"active"`
	// CardExpiryDays is the number of days an 'open' card lives from its
	// created_at before it is auto-cancelled. nil = never expires (default).
	CardExpiryDays *int `json:"card_expiry_days"`
}

// Card represents a single client pushcard.
// Lifecycle: open → completed → redeemed (or cancelled).
type Card struct {
	ID              string     `json:"id"`
	CustomerSisfiID string     `json:"customer_sisfi_id"`
	ClientID        string     `json:"client_id"`
	Status          string     `json:"status"` // open, completed, redeemed, cancelled
	StampsCount     int        `json:"stamps_count"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Stamp represents a single stamp added by a collaborator.
type Stamp struct {
	ID             string    `json:"id"`
	CardID         string    `json:"card_id"`
	CollaboratorID string    `json:"collaborator_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// AddStampReq is the input for adding a stamp.
type AddStampReq struct {
	CustomerSisfiID string `json:"customer_sisfi_id"`
	ClientID        string `json:"client_id"`
	CollaboratorID  string `json:"collaborator_id"`
}

// AddStampResult is returned after adding a stamp.
type AddStampResult struct {
	Card        *Card  `json:"card"`
	StampsCount int    `json:"stamps_count"`
	CardSlots   int    `json:"card_slots"`
	Completed   bool   `json:"completed"`
	StampID     string `json:"stamp_id"`
}

// CardProgress is a read-only view of the client's current card.
type CardProgress struct {
	HasOpenCard bool   `json:"has_open_card"`
	Card        *Card  `json:"card,omitempty"`
	StampsCount int    `json:"stamps_count"`
	CardSlots   int    `json:"card_slots"`
	Visual      string `json:"visual"` // ●●●○○ style
}

const (
	StatusOpen      = "open"
	StatusCompleted = "completed"
	StatusRedeemed  = "redeemed"
	StatusCancelled = "cancelled"
)
