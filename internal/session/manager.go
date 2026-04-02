package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	sessionTTL   = 30 * time.Minute
	selectionTTL = 5 * time.Minute
	flowTTL      = 30 * time.Minute
)

// UserContext holds the resolved session data for a user.
type UserContext struct {
	CustomerID     string   `json:"customer_id"`
	Role           string   `json:"role"`                        // "client" or "collaborator"
	UserID         string   `json:"user_id"`                     // client_id or collaborator_id
	BusinessName   string   `json:"business_name"`
	ActiveModules  []string `json:"active_modules"`              // e.g. ["earn_burn", "cashback"]
	CollaboratorID string   `json:"collaborator_id,omitempty"`   // set when user is also a collaborator
	ClientID       string   `json:"client_id,omitempty"`         // set when user is also a client
}

// FlowState holds the current step-by-step flow state.
type FlowState struct {
	CurrentFlow   string            `json:"current_flow"`
	CurrentStep   int               `json:"current_step"`
	CollectedData map[string]string `json:"collected_data"`
	StartedAt     time.Time         `json:"started_at"`
}

// SelectionOption represents a business the user can choose from.
type SelectionOption struct {
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
}

type Manager struct {
	client *redis.Client
}

func NewManager(client *redis.Client) *Manager {
	return &Manager{client: client}
}

// GetSession returns the cached user context, or nil if no session exists.
func (m *Manager) GetSession(ctx context.Context, phone string) (*UserContext, error) {
	data, err := m.client.Get(ctx, sessionKey(phone)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Refresh TTL on access
	m.client.Expire(ctx, sessionKey(phone), sessionTTL)

	var uc UserContext
	if err := json.Unmarshal(data, &uc); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &uc, nil
}

// SetSession stores the user context in Redis.
func (m *Manager) SetSession(ctx context.Context, phone string, uc *UserContext) error {
	data, err := json.Marshal(uc)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return m.client.Set(ctx, sessionKey(phone), data, sessionTTL).Err()
}

// DeleteSession removes the user session.
func (m *Manager) DeleteSession(ctx context.Context, phone string) error {
	return m.client.Del(ctx, sessionKey(phone)).Err()
}

// GetPendingSelection returns the pending business selection options.
func (m *Manager) GetPendingSelection(ctx context.Context, phone string) ([]SelectionOption, error) {
	data, err := m.client.Get(ctx, selectionKey(phone)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get selection: %w", err)
	}

	var opts []SelectionOption
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("unmarshal selection: %w", err)
	}
	return opts, nil
}

// SetPendingSelection stores business options for user to choose from.
func (m *Manager) SetPendingSelection(ctx context.Context, phone string, opts []SelectionOption) error {
	data, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("marshal selection: %w", err)
	}
	return m.client.Set(ctx, selectionKey(phone), data, selectionTTL).Err()
}

// DeletePendingSelection removes the pending selection.
func (m *Manager) DeletePendingSelection(ctx context.Context, phone string) error {
	return m.client.Del(ctx, selectionKey(phone)).Err()
}

// GetFlowState returns the active flow state, or nil if none.
func (m *Manager) GetFlowState(ctx context.Context, phone, customerID string) (*FlowState, error) {
	data, err := m.client.Get(ctx, flowKey(phone, customerID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get flow: %w", err)
	}

	// Refresh TTL on access
	m.client.Expire(ctx, flowKey(phone, customerID), flowTTL)

	var fs FlowState
	if err := json.Unmarshal(data, &fs); err != nil {
		return nil, fmt.Errorf("unmarshal flow: %w", err)
	}
	return &fs, nil
}

// SetFlowState stores or updates the flow state.
func (m *Manager) SetFlowState(ctx context.Context, phone, customerID string, fs *FlowState) error {
	data, err := json.Marshal(fs)
	if err != nil {
		return fmt.Errorf("marshal flow: %w", err)
	}
	return m.client.Set(ctx, flowKey(phone, customerID), data, flowTTL).Err()
}

// DeleteFlowState removes the active flow.
func (m *Manager) DeleteFlowState(ctx context.Context, phone, customerID string) error {
	return m.client.Del(ctx, flowKey(phone, customerID)).Err()
}

func sessionKey(phone string) string   { return "session:" + phone }
func selectionKey(phone string) string { return "session:select:" + phone }
func flowKey(phone, biz string) string { return "flow:" + phone + ":" + biz }
