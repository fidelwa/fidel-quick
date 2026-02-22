package earnburn

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	otpIdentityTTL   = 15 * time.Minute
	otpRedemptionTTL = 1 * time.Hour
	otpLoadPointsTTL = 15 * time.Minute
)

// OTPData is stored in Redis under otp:{code}.
type OTPData struct {
	ClientID   string            `json:"client_id"`
	CustomerID string            `json:"customer_id"`
	Type       string            `json:"type"` // "identity", "redemption", "load_points"
	Metadata   map[string]string `json:"metadata"`
}

type Cache interface {
	SetOTP(ctx context.Context, code string, data *OTPData) error
	GetOTP(ctx context.Context, code string) (*OTPData, error)
	ConsumeOTP(ctx context.Context, code string) (*OTPData, error) // GETDEL
	DeleteOTP(ctx context.Context, code string) error

	// Active identity tracker: ensures only one identity OTP per client
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

	ttl := otpTTL(data.Type)
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
	return c.client.Set(ctx, activeIdentityKey(clientID), code, otpIdentityTTL).Err()
}

func (c *RedisCache) GetActiveIdentity(ctx context.Context, clientID string) (string, error) {
	code, err := c.client.Get(ctx, activeIdentityKey(clientID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return code, err
}

func (c *RedisCache) DeleteActiveIdentity(ctx context.Context, clientID string) error {
	return c.client.Del(ctx, activeIdentityKey(clientID)).Err()
}

func otpKey(code string) string             { return "otp:" + code }
func activeIdentityKey(clientID string) string { return "otp:active:" + clientID }

func otpTTL(otpType string) time.Duration {
	switch otpType {
	case "redemption":
		return otpRedemptionTTL
	case "load_points":
		return otpLoadPointsTTL
	default:
		return otpIdentityTTL
	}
}
