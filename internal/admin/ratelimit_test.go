package admin

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRateLimiter_AllowsUpToLimit(t *testing.T) {
	l := NewMemoryRateLimiter(3, time.Hour)
	assert.True(t, l.Allow("k"))
	assert.True(t, l.Allow("k"))
	assert.True(t, l.Allow("k"))
	assert.False(t, l.Allow("k"), "4th hit over limit of 3 must be denied")
}

func TestMemoryRateLimiter_PerKey(t *testing.T) {
	l := NewMemoryRateLimiter(1, time.Hour)
	assert.True(t, l.Allow("a"))
	assert.False(t, l.Allow("a"))
	assert.True(t, l.Allow("b"), "different key has its own budget")
}

func TestMemoryRateLimiter_WindowExpiry(t *testing.T) {
	l := NewMemoryRateLimiter(1, time.Hour).(*memoryRateLimiter)
	base := time.Now()
	l.now = func() time.Time { return base }
	assert.True(t, l.Allow("k"))
	assert.False(t, l.Allow("k"))
	// Advance past the window — the old hit should age out.
	l.now = func() time.Time { return base.Add(2 * time.Hour) }
	assert.True(t, l.Allow("k"))
}

// TestMemoryRateLimiter_SweepsExpiredKeys verifies that keys whose hits have
// all expired are eventually dropped from the map, so it doesn't grow without
// bound for one-shot keys (LG-2).
func TestMemoryRateLimiter_SweepsExpiredKeys(t *testing.T) {
	l := NewMemoryRateLimiter(5, time.Hour).(*memoryRateLimiter)
	base := time.Now()
	l.now = func() time.Time { return base }

	// Hit many distinct one-shot keys at t=base.
	const n = 300 // > sweepEvery so a sweep runs
	for i := 0; i < n; i++ {
		require.True(t, l.Allow("ip-"+strconv.Itoa(i)))
	}
	assert.Equal(t, n, len(l.hits), "each distinct key is tracked while fresh")

	// Advance past the window so every stored hit is now expired, then make a
	// single call to trigger the lazy sweep.
	l.now = func() time.Time { return base.Add(2 * time.Hour) }
	// Force a sweep on the next Allow regardless of prior counter state.
	l.sweepCount = sweepEvery - 1
	require.True(t, l.Allow("trigger"))

	// Only the just-added "trigger" key should remain; all expired keys gone.
	assert.Equal(t, 1, len(l.hits), "expired one-shot keys must be swept from the map")
	_, ok := l.hits["trigger"]
	assert.True(t, ok)
}

func TestNoopRateLimiter_AlwaysAllows(t *testing.T) {
	var l RateLimiter = noopRateLimiter{}
	for i := 0; i < 100; i++ {
		assert.True(t, l.Allow("k"))
	}
}
