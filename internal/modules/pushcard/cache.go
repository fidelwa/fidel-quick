package pushcard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	redemptionTTL = 1 * time.Hour
)

// RedemptionCode is stored in Redis under pc:redeem:{code}. The collaborator
// confirms a card by entering this code; it maps back to the card_id and to
// the customer_sisfi to make sure the canje is in the right business.
type RedemptionCode struct {
	CardID          string `json:"card_id"`
	ClientID        string `json:"client_id"`
	CustomerID      string `json:"customer_id"`
	CustomerSisfiID string `json:"customer_sisfi_id"`
	RewardName      string `json:"reward_name"`
}

// Cache abstracts the redemption code store so the service can be tested.
type Cache interface {
	SetRedemption(ctx context.Context, code string, data *RedemptionCode) error
	GetRedemption(ctx context.Context, code string) (*RedemptionCode, error)
	ConsumeRedemption(ctx context.Context, code string) (*RedemptionCode, error)
	DeleteRedemption(ctx context.Context, code string) error
}

// RedisCache implements Cache backed by Redis.
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) SetRedemption(ctx context.Context, code string, data *RedemptionCode) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal redemption: %w", err)
	}
	return c.client.Set(ctx, redemptionKey(code), b, redemptionTTL).Err()
}

func (c *RedisCache) GetRedemption(ctx context.Context, code string) (*RedemptionCode, error) {
	b, err := c.client.Get(ctx, redemptionKey(code)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get redemption: %w", err)
	}
	var data RedemptionCode
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal redemption: %w", err)
	}
	return &data, nil
}

func (c *RedisCache) ConsumeRedemption(ctx context.Context, code string) (*RedemptionCode, error) {
	b, err := c.client.GetDel(ctx, redemptionKey(code)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("consume redemption: %w", err)
	}
	var data RedemptionCode
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal redemption: %w", err)
	}
	return &data, nil
}

func (c *RedisCache) DeleteRedemption(ctx context.Context, code string) error {
	return c.client.Del(ctx, redemptionKey(code)).Err()
}

func redemptionKey(code string) string { return "pc:redeem:" + code }
