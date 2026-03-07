package throttler

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTryThrottle_WindowExpiry verifies that after the window expires,
// the operation counter resets and new operations are allowed.
func TestTryThrottle_WindowExpiry(t *testing.T) {
	th := New(50*time.Millisecond, 1)

	// First operation allowed
	assert.True(t, th.TryThrottle("op1"))

	// Second operation within window denied
	assert.False(t, th.TryThrottle("op1"))

	// Wait for window to expire
	time.Sleep(80 * time.Millisecond)

	// After window expiry, should be allowed again (exercises the else branch)
	assert.True(t, th.TryThrottle("op1"))
}

// TestTryThrottle_WindowExpiry_MultipleOps verifies multiple operations
// work correctly after window expiry.
func TestTryThrottle_WindowExpiry_MultipleOps(t *testing.T) {
	th := New(50*time.Millisecond, 3)

	// Use all operations in first window
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op1"))
	assert.False(t, th.TryThrottle("op1"))

	// Wait for window to expire
	time.Sleep(80 * time.Millisecond)

	// Should get a fresh set of operations
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op1"))
	assert.False(t, th.TryThrottle("op1"))
}

// TestClear_NonExistentKey verifies Clear on a non-existent key is a no-op.
func TestClear_NonExistentKey(t *testing.T) {
	th := New(time.Second, 5)
	// Should not panic
	th.Clear("nonexistent")
}

// TestConcurrent_TryThrottle verifies thread safety under concurrent access.
func TestConcurrent_TryThrottle(t *testing.T) {
	th := New(time.Minute, 1000)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				th.TryThrottle("concurrent-op")
			}
		}()
	}
	wg.Wait()
}

// TestConcurrent_TryThrottleAndClear tests thread safety of TryThrottle and Clear.
func TestConcurrent_TryThrottleAndClear(t *testing.T) {
	th := New(time.Minute, 100)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			th.TryThrottle("op")
		}()
		go func() {
			defer wg.Done()
			th.Clear("op")
		}()
	}
	wg.Wait()
}

// TestTryThrottle_MultipleOperationIDs verifies independent tracking per ID
// including window expiry for each.
func TestTryThrottle_MultipleOperationIDs(t *testing.T) {
	th := New(50*time.Millisecond, 1)

	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op2"))
	assert.False(t, th.TryThrottle("op1"))
	assert.False(t, th.TryThrottle("op2"))

	time.Sleep(80 * time.Millisecond)

	// Both should reset independently
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op2"))
}

// TestNew_ZeroWindow verifies behavior with a zero-duration window.
func TestNew_ZeroWindow(t *testing.T) {
	th := New(0, 5)
	assert.NotNil(t, th)
	// With zero window, now-lastOp will always be >= windowMs (0),
	// so it should always reset the counter.
	assert.True(t, th.TryThrottle("op"))
	assert.True(t, th.TryThrottle("op"))
}
