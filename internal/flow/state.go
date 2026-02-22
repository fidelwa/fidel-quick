package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const flowTTL = 30 * time.Minute

// StateStore persists flow state in Redis.
type StateStore struct {
	client *redis.Client
}

func NewStateStore(client *redis.Client) *StateStore {
	return &StateStore{client: client}
}

// Get returns the active flow state, or nil if none.
func (s *StateStore) Get(ctx context.Context, phone, customerID string) (*State, error) {
	data, err := s.client.Get(ctx, flowKey(phone, customerID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get flow state: %w", err)
	}

	// Refresh TTL on access
	s.client.Expire(ctx, flowKey(phone, customerID), flowTTL)

	var fs State
	if err := json.Unmarshal(data, &fs); err != nil {
		return nil, fmt.Errorf("unmarshal flow state: %w", err)
	}
	return &fs, nil
}

// Set stores or updates the flow state.
func (s *StateStore) Set(ctx context.Context, phone, customerID string, fs *State) error {
	data, err := json.Marshal(fs)
	if err != nil {
		return fmt.Errorf("marshal flow state: %w", err)
	}
	return s.client.Set(ctx, flowKey(phone, customerID), data, flowTTL).Err()
}

// Delete removes the active flow.
func (s *StateStore) Delete(ctx context.Context, phone, customerID string) error {
	return s.client.Del(ctx, flowKey(phone, customerID)).Err()
}

func flowKey(phone, customerID string) string {
	return "flow:" + phone + ":" + customerID
}
