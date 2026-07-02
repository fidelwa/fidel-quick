package admin

import (
	"sync"
	"time"
)

// RateLimiter tracks how many times a key was hit inside a rolling window and
// reports whether a new hit is allowed. Implementations must be safe for
// concurrent use.
type RateLimiter interface {
	// Allow records one hit for key and returns false when the caller has
	// exceeded the configured limit inside the window.
	Allow(key string) bool
}

// memoryRateLimiter is a fixed-window, in-memory limiter. It is intentionally
// simple: password reset is low-traffic, and a single instance is enough for
// the MVP. A Redis-backed limiter can implement the same interface later
// without touching the service.
// sweepEvery controls how often (in Allow calls) the limiter walks the whole
// map dropping keys whose hits have all expired. A small constant keeps the
// amortized cost of the sweep negligible.
const sweepEvery = 256

type memoryRateLimiter struct {
	mu         sync.Mutex
	limit      int
	window     time.Duration
	hits       map[string][]time.Time
	now        func() time.Time
	sweepCount int
}

// sweepExpiredLocked removes every key whose hits all fall on/before cutoff.
// The caller must hold l.mu.
func (l *memoryRateLimiter) sweepExpiredLocked(cutoff time.Time) {
	for k, ts := range l.hits {
		alive := false
		for _, t := range ts {
			if t.After(cutoff) {
				alive = true
				break
			}
		}
		if !alive {
			delete(l.hits, k)
		}
	}
}

// NewMemoryRateLimiter builds an in-memory limiter allowing `limit` hits per
// `window` per key.
func NewMemoryRateLimiter(limit int, window time.Duration) RateLimiter {
	return &memoryRateLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
		now:    time.Now,
	}
}

func (l *memoryRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)

	// Drop hits that fell out of the window.
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}

	l.hits[key] = append(kept, now)

	// Lazy sweep: periodically drop keys whose every hit has expired so the
	// map doesn't grow without bound for one-shot keys (e.g. an IP that hits
	// once and never returns). Cheap because it only runs every sweepEvery
	// calls, not on every Allow.
	l.sweepCount++
	if l.sweepCount >= sweepEvery {
		l.sweepCount = 0
		l.sweepExpiredLocked(cutoff)
	}

	return true
}

// noopRateLimiter allows everything. Used when a nil limiter is injected
// (e.g. some unit tests) so the service never has to nil-check.
type noopRateLimiter struct{}

func (noopRateLimiter) Allow(string) bool { return true }
