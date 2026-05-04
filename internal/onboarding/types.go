package onboarding

import "time"

type Onboarding struct {
	ID          string     `json:"id"`
	CustomerID  string     `json:"customer_id"`
	CurrentStep int        `json:"current_step"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
