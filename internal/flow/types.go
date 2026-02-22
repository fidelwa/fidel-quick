package flow

import "time"

// State represents the current progress of a multi-step flow.
type State struct {
	CurrentFlow   string            `json:"current_flow"`
	CurrentStep   int               `json:"current_step"`
	CollectedData map[string]string `json:"collected_data"`
	StartedAt     time.Time         `json:"started_at"`
}
