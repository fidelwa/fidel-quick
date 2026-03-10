package cashback

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	cbIdentityTTL   = 15 * time.Minute
	cbRedemptionTTL = 1 * time.Hour
	cbLoadPointsTTL = 15 * time.Minute
)

// OTPData is stored in Redis under otp:{code}.
type OTPData struct {
	ClientID   string            `json:"client_id"`
	CustomerID string            `json:"customer_id"`
	Type       string            `json:"type"` // "cb_redemption", "cb_load_points"
	Metadata   map[string]string `json:"metadata"`
}

type Cache interface {
	SetOTP(ctx context.Context, code string, data *OTPData) error
	GetOTP(ctx context.Context, code string) (*OTPData, error)
	ConsumeOTP(ctx context.Context, code string) (*OTPData, error)
	DeleteOTP(ctx context.Context, code string) error
	SetActiveIdentity(ctx context.Context, clientID, code string) error
	GetActiveIdentity(ctx context.Context, clientID string) (string, error)
	DeleteActiveIdentity(ctx context.Context, clientID string) error
}

// RedisCache implements Cache.
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) SetOTP(ctx context.Context, code string, data *OTPData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal otp: %w", err)
	}

	ttl := cbOtpTTL(data.Type)
	return c.client.Set(ctx, otpKey(code), b, ttl).Err()
}

func (c *RedisCache) GetOTP(ctx context.Context, code string) (*OTPData, error) {
	b, err := c.client.Get(ctx, otpKey(code)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get otp: %w", err)
	}

	var data OTPData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal otp: %w", err)
	}
	return &data, nil
}

func (c *RedisCache) ConsumeOTP(ctx context.Context, code string) (*OTPData, error) {
	b, err := c.client.GetDel(ctx, otpKey(code)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("consume otp: %w", err)
	}

	var data OTPData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal otp: %w", err)
	}
	return &data, nil
}

func (c *RedisCache) DeleteOTP(ctx context.Context, code string) error {
	return c.client.Del(ctx, otpKey(code)).Err()
}

func (c *RedisCache) SetActiveIdentity(ctx context.Context, clientID, code string) error {
	return c.client.Set(ctx, cbActiveIdentityKey(clientID), code, cbIdentityTTL).Err()
}

func (c *RedisCache) GetActiveIdentity(ctx context.Context, clientID string) (string, error) {
	code, err := c.client.Get(ctx, cbActiveIdentityKey(clientID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return code, err
}

func (c *RedisCache) DeleteActiveIdentity(ctx context.Context, clientID string) error {
	return c.client.Del(ctx, cbActiveIdentityKey(clientID)).Err()
}

func otpKey(code string) string                    { return "otp:" + code }
func cbActiveIdentityKey(clientID string) string    { return "cb:identity:" + clientID }

func cbOtpTTL(otpType string) time.Duration {
	switch otpType {
	case "cb_identity":
		return cbIdentityTTL
	case "cb_redemption":
		return cbRedemptionTTL
	case "cb_load_points":
		return cbLoadPointsTTL
	default:
		return cbLoadPointsTTL
	}
}
