package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"digital.vasic.ratelimiter/pkg/limiter"
	"digital.vasic.ratelimiter/pkg/memory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_ZeroRate(t *testing.T) {
	t.Parallel()

	// Rate of 0 should be clamped to 1 by the sliding window
	cfg := &limiter.Config{
		Rate:   0,
		Window: time.Second,
		Burst:  0,
	}

	rl := memory.New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Effective rate should be 1 (clamped by sliding.NewWindow)
	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	// The first request should be allowed since effective burst defaults to rate
	assert.True(t, result.Allowed)

	// Second request should be denied (rate=1)
	result, err = rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestRateLimiter_NegativeBurst(t *testing.T) {
	t.Parallel()

	cfg := &limiter.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  -1, // EffectiveBurst() returns Rate when Burst <= 0
	}

	rl := memory.New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, 5, result.Limit, "effective burst should equal rate when burst is negative")
	assert.True(t, result.Allowed)
}

func TestRateLimiter_ConcurrentSameKey(t *testing.T) {
	t.Parallel()

	cfg := &limiter.Config{
		Rate:   50,
		Window: time.Second,
		Burst:  50,
	}

	rl := memory.New(cfg)
	defer rl.Stop()

	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0
	deniedCount := 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := rl.Allow(ctx, "shared-key")
			require.NoError(t, err)
			mu.Lock()
			if result.Allowed {
				allowedCount++
			} else {
				deniedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	assert.Equal(t, 50, allowedCount, "exactly 50 should be allowed")
	assert.Equal(t, 50, deniedCount, "exactly 50 should be denied")
}

func TestRateLimiter_AfterStop(t *testing.T) {
	t.Parallel()

	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  10,
	}

	rl := memory.New(cfg)
	rl.Stop()

	// After stop, Allow should still work (just no background cleanup)
	ctx := context.Background()
	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRateLimiter_VeryHighRate_NoLimiting(t *testing.T) {
	t.Parallel()

	cfg := &limiter.Config{
		Rate:   1000000,
		Window: time.Second,
		Burst:  1000000,
	}

	rl := memory.New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// With such a high rate, all requests should be allowed
	for i := 0; i < 1000; i++ {
		result, err := rl.Allow(ctx, "key")
		require.NoError(t, err)
		assert.True(t, result.Allowed, "request %d should be allowed with rate 1M", i)
	}
}

func TestRateLimiter_RateOnePerWindow(t *testing.T) {
	t.Parallel()

	cfg := &limiter.Config{
		Rate:   1,
		Window: 200 * time.Millisecond,
		Burst:  1,
	}

	rl := memory.New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// First request allowed
	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Second request denied
	result, err = rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Greater(t, result.RetryAfter, time.Duration(0))

	// Wait for window to pass
	time.Sleep(250 * time.Millisecond)

	// Now should be allowed again
	result, err = rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRateLimiter_ResetNonexistentKey(t *testing.T) {
	t.Parallel()

	rl := memory.New(nil)
	defer rl.Stop()

	// Resetting a key that never existed should not error
	err := rl.Reset(context.Background(), "nonexistent")
	assert.NoError(t, err)
}

func TestRateLimiter_ManyDistinctKeys(t *testing.T) {
	t.Parallel()

	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}

	rl := memory.New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Each key should have its own independent window
	for i := 0; i < 100; i++ {
		key := "key-" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		result, err := rl.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "first request for %s should be allowed", key)
	}
}

func TestRateLimiter_ConcurrentResetAndAllow(t *testing.T) {
	t.Parallel()

	cfg := &limiter.Config{
		Rate:   5,
		Window: time.Second,
		Burst:  5,
	}

	rl := memory.New(cfg)
	defer rl.Stop()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent Allow and Reset on the same key
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = rl.Allow(ctx, "contested-key")
		}()
		go func() {
			defer wg.Done()
			_ = rl.Reset(ctx, "contested-key")
		}()
	}

	wg.Wait()
	// No deadlock or panic means success
}

func TestEffectiveBurst_DefaultsToRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   limiter.Config
		expected int
	}{
		{"zero burst", limiter.Config{Rate: 10, Burst: 0}, 10},
		{"negative burst", limiter.Config{Rate: 7, Burst: -5}, 7},
		{"positive burst", limiter.Config{Rate: 10, Burst: 20}, 20},
		{"burst equals rate", limiter.Config{Rate: 5, Burst: 5}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.config.EffectiveBurst())
		})
	}
}
