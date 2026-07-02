package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestNoopRateLimiter_AlwaysAllows(t *testing.T) {
	var l RateLimiter = noopRateLimiter{}
	for i := 0; i < 100; i++ {
		assert.True(t, l.Allow("k"))
	}
}
