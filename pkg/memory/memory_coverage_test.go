package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"digital.vasic.ratelimiter/pkg/limiter"
	"digital.vasic.ratelimiter/pkg/sliding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllow_RemainingNeverNegative exercises the remaining < 0 guard in Allow.
// We create a scenario where currentCount can exceed effectiveLimit by
// pre-populating a window with a rate higher than the config's EffectiveBurst.
func TestAllow_RemainingNeverNegative(t *testing.T) {
	// Create a limiter with burst=2 but inject a window with rate=10
	// so the window allows more requests than the limiter's effective burst.
	cfg := &limiter.Config{
		Rate:   2,
		Window: time.Second,
		Burst:  2,
	}
	rl := &RateLimiter{
		windows: make(map[string]*sliding.Window),
		config:  cfg,
		stopCh:  make(chan struct{}),
	}
	defer func() { close(rl.stopCh) }()

	// Inject a sliding window with a higher rate than the config's burst
	w := sliding.NewWindow(10, time.Second, 10)
	now := time.Now()
	// Fill the window beyond the limiter's effective burst of 2
	for i := 0; i < 5; i++ {
		w.Allow(now)
	}
	rl.windows["key"] = w

	ctx := context.Background()
	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	// currentCount (5) > effectiveLimit (2), so remaining should be clamped to 0
	assert.Equal(t, 0, result.Remaining, "remaining should be clamped to 0, never negative")
}

// TestAllow_RetryAfterClamped exercises the RetryAfter < 0 clamp in Allow.
// We inject a window whose resetAt is in the past so time.Until returns negative.
func TestAllow_RetryAfterClamped(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: 10 * time.Millisecond,
		Burst:  1,
	}
	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Exhaust the limit
	rl.Allow(ctx, "key")

	// Wait past the window so the resetAt from the window is in the past
	time.Sleep(50 * time.Millisecond)

	// Now Allow will compute resetAt from the old sub-window, which will be
	// in the past, making time.Until(resetAt) negative. The guard should clamp to 0.
	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	// After the window expired, the old entries are cleaned, so it should be allowed.
	// But if the window cleanup happened, it's allowed again. Let's verify RetryAfter is >= 0.
	assert.GreaterOrEqual(t, int64(result.RetryAfter), int64(0), "RetryAfter should never be negative")
}

// TestAllow_RetryAfterNonNegative checks that RetryAfter is non-negative on immediate denial.
func TestAllow_RetryAfterNonNegative(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: 50 * time.Millisecond,
		Burst:  1,
	}
	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Exhaust
	rl.Allow(ctx, "key")

	// Denied immediately
	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.GreaterOrEqual(t, int64(result.RetryAfter), int64(0), "RetryAfter should be non-negative")
}

// TestGetOrCreateWindow_DoubleCheck exercises the double-check locking path
// in getOrCreateWindow by having concurrent goroutines create the same key.
func TestGetOrCreateWindow_DoubleCheck(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   100,
		Window: time.Second,
		Burst:  100,
	}
	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()
	var wg sync.WaitGroup
	const goroutines = 50

	// All goroutines try to access the same key simultaneously,
	// forcing the double-check path in getOrCreateWindow.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := rl.Allow(ctx, "same-key")
			assert.NoError(t, err)
			assert.True(t, result.Allowed)
		}()
	}
	wg.Wait()

	// All goroutines should share the same window
	rl.mu.RLock()
	windowCount := len(rl.windows)
	rl.mu.RUnlock()
	assert.Equal(t, 1, windowCount, "should have exactly one window for the key")
}

// TestCleanup_RemovesExpiredWindows tests the cleanup goroutine path that
// removes idle windows by simulating the same logic cleanup performs.
func TestCleanup_RemovesExpiredWindows(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: 10 * time.Millisecond,
		Burst:  10,
	}
	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Create a few windows
	rl.Allow(ctx, "key-a")
	rl.Allow(ctx, "key-b")

	// Verify windows exist
	rl.mu.RLock()
	assert.Equal(t, 2, len(rl.windows))
	rl.mu.RUnlock()

	// Wait for the window to expire
	time.Sleep(50 * time.Millisecond)

	// Manually trigger cleanup logic since we can't wait for the 1-minute ticker.
	// We simulate what the cleanup goroutine does.
	rl.mu.Lock()
	now := time.Now()
	for key, w := range rl.windows {
		if w.Count(now) == 0 {
			delete(rl.windows, key)
		}
	}
	rl.mu.Unlock()

	rl.mu.RLock()
	remaining := len(rl.windows)
	rl.mu.RUnlock()
	assert.Equal(t, 0, remaining, "expired windows should be cleaned up")
}

// TestCleanup_KeepsActiveWindows verifies cleanup does not remove windows
// with active requests.
func TestCleanup_KeepsActiveWindows(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Minute, // long window so it won't expire
		Burst:  10,
	}
	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()
	rl.Allow(ctx, "active-key")

	// Simulate cleanup
	rl.mu.Lock()
	now := time.Now()
	for key, w := range rl.windows {
		if w.Count(now) == 0 {
			delete(rl.windows, key)
		}
	}
	rl.mu.Unlock()

	rl.mu.RLock()
	count := len(rl.windows)
	rl.mu.RUnlock()
	assert.Equal(t, 1, count, "active window should not be cleaned up")
}

// TestStop_PreventsDoubleClose verifies Stop is safe to call multiple times.
func TestStop_PreventsDoubleClose(t *testing.T) {
	rl := New(nil)
	rl.Stop()
	rl.Stop() // should not panic
}

// TestReset_NonExistentKey verifies that resetting a non-existent key is a no-op.
func TestReset_NonExistentKey(t *testing.T) {
	rl := New(nil)
	defer rl.Stop()

	err := rl.Reset(context.Background(), "does-not-exist")
	assert.NoError(t, err)
}

// TestAllow_WindowRecovery verifies requests are allowed again after the window passes.
func TestAllow_WindowRecovery(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: 50 * time.Millisecond,
		Burst:  1,
	}
	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()

	// Exhaust
	result, _ := rl.Allow(ctx, "key")
	assert.True(t, result.Allowed)
	result, _ = rl.Allow(ctx, "key")
	assert.False(t, result.Allowed)

	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)

	// Should be allowed again
	result, err := rl.Allow(ctx, "key")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

// TestConcurrentAllowAndReset verifies thread safety of Allow and Reset together.
func TestConcurrentAllowAndReset(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   100,
		Window: time.Second,
		Burst:  100,
	}
	rl := New(cfg)
	defer rl.Stop()

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			rl.Allow(ctx, "key")
		}()
		go func() {
			defer wg.Done()
			rl.Reset(ctx, "key")
		}()
	}
	wg.Wait()
}

// TestGetOrCreateWindow_DirectDoubleCheck directly tests the double-check
// path by pre-populating the window before calling getOrCreateWindow.
func TestGetOrCreateWindow_DirectDoubleCheck(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  10,
	}
	rl := &RateLimiter{
		windows: make(map[string]*sliding.Window),
		config:  cfg,
		stopCh:  make(chan struct{}),
	}
	defer func() { close(rl.stopCh) }()

	// Pre-populate a window
	existing := sliding.NewWindow(10, time.Second, 10)
	rl.windows["preloaded"] = existing

	// getOrCreateWindow should find it via the read lock path
	w := rl.getOrCreateWindow("preloaded")
	assert.Equal(t, existing, w, "should return the pre-existing window")

	// A new key should create a new window (different pointer)
	w2 := rl.getOrCreateWindow("new-key")
	assert.NotNil(t, w2)
	assert.False(t, w == w2, "new key should create a different window instance")
}
