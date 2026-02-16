package sliding

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWindow(t *testing.T) {
	w := NewWindow(10, time.Minute, 6)
	require.NotNil(t, w)
	assert.Equal(t, 10, w.Rate())
	assert.Equal(t, time.Minute, w.WindowDuration())
}

func TestNewWindowDefaults(t *testing.T) {
	// Zero/negative values should get safe defaults
	w := NewWindow(0, 0, 0)
	require.NotNil(t, w)
	assert.Equal(t, 1, w.Rate())
	assert.Equal(t, time.Minute, w.WindowDuration())
}

func TestAllowBasic(t *testing.T) {
	w := NewWindow(3, time.Second, 10)
	now := time.Now()

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, count, _ := w.Allow(now)
		assert.True(t, allowed, "request %d should be allowed", i+1)
		assert.Equal(t, i+1, count)
	}

	// 4th request should be denied
	allowed, count, _ := w.Allow(now)
	assert.False(t, allowed)
	assert.Equal(t, 3, count)
}

func TestAllowAfterWindowExpires(t *testing.T) {
	w := NewWindow(2, 100*time.Millisecond, 10)
	now := time.Now()

	// Use up the limit
	w.Allow(now)
	w.Allow(now)

	allowed, _, _ := w.Allow(now)
	assert.False(t, allowed, "should be denied when limit reached")

	// After the window passes, requests should be allowed again
	future := now.Add(150 * time.Millisecond)
	allowed, count, _ := w.Allow(future)
	assert.True(t, allowed, "should be allowed after window expires")
	assert.Equal(t, 1, count)
}

func TestCount(t *testing.T) {
	w := NewWindow(10, time.Second, 10)
	now := time.Now()

	assert.Equal(t, 0, w.Count(now))

	w.Allow(now)
	w.Allow(now)
	w.Allow(now)

	assert.Equal(t, 3, w.Count(now))
}

func TestReset(t *testing.T) {
	w := NewWindow(5, time.Second, 10)
	now := time.Now()

	w.Allow(now)
	w.Allow(now)
	assert.Equal(t, 2, w.Count(now))

	w.Reset()
	assert.Equal(t, 0, w.Count(now))
}

func TestSlidingBehavior(t *testing.T) {
	// With a 1-second window and 4 requests allowed,
	// requests made at different sub-windows should
	// slide out independently.
	w := NewWindow(4, time.Second, 10)
	base := time.Now()

	// Make 2 requests at t=0
	w.Allow(base)
	w.Allow(base)

	// Make 2 requests at t=500ms
	mid := base.Add(500 * time.Millisecond)
	w.Allow(mid)
	w.Allow(mid)

	// At t=500ms we should have 4 (all within 1s window)
	assert.Equal(t, 4, w.Count(mid))

	// At t=500ms, 5th request should be denied
	allowed, _, _ := w.Allow(mid)
	assert.False(t, allowed)

	// At t=1100ms, the first 2 requests have expired but the mid ones remain
	later := base.Add(1100 * time.Millisecond)
	assert.Equal(t, 2, w.Count(later))

	// Should be able to make more requests now
	allowed, _, _ = w.Allow(later)
	assert.True(t, allowed)
}

func TestResetAtReturnValue(t *testing.T) {
	w := NewWindow(1, time.Second, 10)
	now := time.Now()

	// First request allowed
	allowed, _, resetAt := w.Allow(now)
	assert.True(t, allowed)
	assert.True(t, resetAt.After(now), "resetAt should be in the future")

	// Second request denied, resetAt should indicate when to retry
	allowed, _, resetAt = w.Allow(now)
	assert.False(t, allowed)
	assert.True(t, resetAt.After(now), "resetAt should be in the future")
	assert.True(t, resetAt.Before(now.Add(2*time.Second)), "resetAt should be within 2 windows")
}

func TestConcurrentAccess(t *testing.T) {
	w := NewWindow(1000, time.Second, 10)
	now := time.Now()

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				w.Allow(now)
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have counted all 1000 requests
	assert.Equal(t, 1000, w.Count(now))
}
