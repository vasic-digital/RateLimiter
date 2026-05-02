package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"digital.vasic.ratelimiter/pkg/limiter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowWithinLimit(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  5,
	}

	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		result, err := rl.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, result.Allowed, "request %d should be allowed", i+1)
		assert.Equal(t, 5, result.Limit)
		assert.Equal(t, 5-i-1, result.Remaining, "remaining should decrease")
	}
}

func TestAllowExceedingLimit(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   3,
		Window: time.Second,
		Burst:  3,
	}

	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Exhaust the limit
	for i := 0; i < 3; i++ {
		result, err := rl.Allow(ctx, "test-key")
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Next request should be denied
	result, err := rl.Allow(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
	assert.True(t, result.RetryAfter > 0, "retry after should be positive")
}

func TestSeparateKeys(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   2,
		Window: time.Second,
		Burst:  2,
	}

	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Exhaust limit for key-a
	rl.Allow(ctx, "key-a")
	rl.Allow(ctx, "key-a")
	result, _ := rl.Allow(ctx, "key-a")
	assert.False(t, result.Allowed)

	// key-b should still have capacity
	result, err := rl.Allow(ctx, "key-b")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 1, result.Remaining)
}

func TestReset(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   2,
		Window: time.Second,
		Burst:  2,
	}

	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Exhaust limit
	rl.Allow(ctx, "key")
	rl.Allow(ctx, "key")
	result, _ := rl.Allow(ctx, "key")
	assert.False(t, result.Allowed)

	// Reset the key
	err := rl.Reset(ctx, "key")
	require.NoError(t, err)

	// Should be allowed again
	result, err = rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 1, result.Remaining)
}

func TestDefaultBurstEqualsRate(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  0, // should default to Rate
	}

	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, 10, result.Limit)
}

func TestConcurrentAccess(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1000,
		Window: time.Second,
		Burst:  1000,
	}

	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()
	var wg sync.WaitGroup

	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				result, err := rl.Allow(ctx, "concurrent-key")
				if err == nil && result.Allowed {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, 1000, allowedCount, "all 1000 requests should be allowed")
}

func TestStopIsIdempotent(t *testing.T) {
	rl := New(nil)

	// Should not panic when called multiple times
	rl.Stop()
	rl.Stop()
}

func TestResultResetAt(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}

	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	result, _ := rl.Allow(ctx, "key")
	assert.True(t, result.Allowed)
	assert.True(t, result.ResetAt.After(time.Now()), "resetAt should be in the future")
}

// Verify that the RateLimiter satisfies the Limiter interface
var _ limiter.Limiter = (*RateLimiter)(nil)
