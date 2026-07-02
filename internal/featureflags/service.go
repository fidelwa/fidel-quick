package featureflags

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
)

// cacheTTL is how long a flag's resolved definition lives in Redis. Toggling a
// flag through the admin API invalidates the cache immediately, so this only
// bounds staleness when the DB is changed out of band.
const cacheTTL = 60 * time.Second

// Cache abstracts the flag definition cache so the service can be tested
// without Redis.
type Cache interface {
	Get(ctx context.Context, key string) (*Flag, bool)
	Set(ctx context.Context, key string, flag *Flag)
	Delete(ctx context.Context, key string)
}

// Service resolves feature flags with a read-through Redis cache in front of
// the repository.
type Service struct {
	repo  Repository
	cache Cache
	log   *slog.Logger
}

func NewService(repo Repository, cache Cache, log *slog.Logger) *Service {
	return &Service{repo: repo, cache: cache, log: log}
}

// Enabled resolves whether `key` is enabled for `customerID`, applying
// override > global > default precedence. Unknown flags (not yet created in the
// DB) resolve to false — the safe default for gating new behaviour behind a
// flag that hasn't been provisioned. Errors are logged and treated as false so
// a flag lookup never breaks a request path.
func (s *Service) Enabled(ctx context.Context, key, customerID string) bool {
	flag, err := s.load(ctx, key)
	if err != nil {
		if s.log != nil {
			s.log.Warn("feature flag lookup failed; defaulting to disabled", "key", key, "error", err)
		}
		return false
	}
	if flag == nil {
		return false
	}
	return flag.Resolve(customerID)
}

// load fetches a flag definition, using the cache when available. A nil flag
// with nil error means "the flag does not exist".
func (s *Service) load(ctx context.Context, key string) (*Flag, error) {
	if s.cache != nil {
		if f, ok := s.cache.Get(ctx, key); ok {
			return f, nil
		}
	}
	flag, err := s.repo.Get(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if s.cache != nil {
		s.cache.Set(ctx, key, flag)
	}
	return flag, nil
}

// EnabledFor resolves every known flag for the given customer and returns a map
// of flag key → enabled. Used to expose the active flags to the frontend (e.g.
// via /auth/me) for UI gating. This reads all definitions from the repository
// (admin/session path, not a hot loop) rather than the per-key cache.
func (s *Service) EnabledFor(ctx context.Context, customerID string) (map[string]bool, error) {
	flags, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(flags))
	for i := range flags {
		out[flags[i].Key] = flags[i].Resolve(customerID)
	}
	return out, nil
}

// List returns every flag definition (uncached — admin read path).
func (s *Service) List(ctx context.Context) ([]Flag, error) {
	return s.repo.List(ctx)
}

// Update upserts a flag and invalidates its cache entry so the next resolution
// picks up the change immediately (no waiting for the TTL).
func (s *Service) Update(ctx context.Context, key string, in UpdateInput) (*Flag, error) {
	flag, err := s.repo.Upsert(ctx, key, in)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.Delete(ctx, key)
	}
	return flag, nil
}

// RedisCache implements Cache backed by Redis, storing the JSON-encoded flag
// definition under ff:{key} with a short TTL.
type RedisCache struct {
	client *redis.Client
	log    *slog.Logger
}

func NewRedisCache(client *redis.Client, log *slog.Logger) *RedisCache {
	return &RedisCache{client: client, log: log}
}

func (c *RedisCache) Get(ctx context.Context, key string) (*Flag, bool) {
	b, err := c.client.Get(ctx, cacheKey(key)).Bytes()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		if c.log != nil {
			c.log.Warn("feature flag cache get failed", "key", key, "error", err)
		}
		return nil, false
	}
	var f Flag
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, false
	}
	return &f, true
}

func (c *RedisCache) Set(ctx context.Context, key string, flag *Flag) {
	b, err := json.Marshal(flag)
	if err != nil {
		return
	}
	if err := c.client.Set(ctx, cacheKey(key), b, cacheTTL).Err(); err != nil && c.log != nil {
		c.log.Warn("feature flag cache set failed", "key", key, "error", err)
	}
}

func (c *RedisCache) Delete(ctx context.Context, key string) {
	if err := c.client.Del(ctx, cacheKey(key)).Err(); err != nil && c.log != nil {
		c.log.Warn("feature flag cache delete failed", "key", key, "error", err)
	}
}

func cacheKey(key string) string { return "ff:" + key }
