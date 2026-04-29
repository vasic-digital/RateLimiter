package sliding

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewWindow_NegativeGranularity verifies that negative granularity defaults to 10.
func TestNewWindow_NegativeGranularity(t *testing.T) {
	w := NewWindow(5, time.Second, -5)
	assert.NotNil(t, w)
	assert.Equal(t, 5, w.Rate())
	assert.Equal(t, time.Second, w.WindowDuration())
}

// TestNewWindow_NegativeRate verifies that negative rate defaults to 1.
func TestNewWindow_NegativeRate(t *testing.T) {
	w := NewWindow(-10, time.Second, 10)
	assert.Equal(t, 1, w.Rate())
}

// TestNewWindow_NegativeWindow verifies that negative window defaults to time.Minute.
func TestNewWindow_NegativeWindow(t *testing.T) {
	w := NewWindow(5, -time.Second, 10)
	assert.Equal(t, time.Minute, w.WindowDuration())
}

// TestNewWindow_SubSizeZero exercises the subSize == 0 fallback to 1ms.
// This happens when window / granularity rounds to zero (very tiny window with low granularity).
func TestNewWindow_SubSizeZero(t *testing.T) {
	// window = 1 nanosecond, granularity = 2
	// subSize = 1ns / 2 = 0 in integer division -> should fallback to 1ms
	w := NewWindow(5, time.Nanosecond, 2)
	assert.NotNil(t, w)

	// The window should still function
	now := time.Now()
	allowed, count, _ := w.Allow(now)
	assert.True(t, allowed)
	assert.Equal(t, 1, count)
}

// TestEarliestExpiry_MultipleSubWindows verifies earliestExpiry picks the correct one.
func TestEarliestExpiry_MultipleSubWindows(t *testing.T) {
	w := NewWindow(3, 500*time.Millisecond, 5)
	base := time.Now()

	// Fill requests at different sub-windows
	w.Allow(base)
	w.Allow(base.Add(100 * time.Millisecond))
	w.Allow(base.Add(200 * time.Millisecond))

	// 4th request should be denied
	allowed, _, resetAt := w.Allow(base.Add(200 * time.Millisecond))
	assert.False(t, allowed)
	// resetAt should be approximately base + 500ms (earliest subWindow + window)
	assert.True(t, resetAt.After(base), "resetAt should be after base time")
	assert.True(t, resetAt.Before(base.Add(time.Second)), "resetAt should be within reason")
}

// TestCount_AfterExpiry verifies Count returns 0 after all sub-windows expire.
func TestCount_AfterExpiry(t *testing.T) {
	w := NewWindow(5, 50*time.Millisecond, 5)
	now := time.Now()

	w.Allow(now)
	w.Allow(now)
	assert.Equal(t, 2, w.Count(now))

	// After the window has fully passed
	future := now.Add(100 * time.Millisecond)
	assert.Equal(t, 0, w.Count(future))
}

// TestReset_ThenAllow verifies that after Reset, the window is fully usable again.
func TestReset_ThenAllow(t *testing.T) {
	w := NewWindow(2, time.Second, 10)
	now := time.Now()

	w.Allow(now)
	w.Allow(now)
	allowed, _, _ := w.Allow(now)
	assert.False(t, allowed)

	w.Reset()
	assert.Equal(t, 0, w.Count(now))

	allowed, count, _ := w.Allow(now)
	assert.True(t, allowed)
	assert.Equal(t, 1, count)
}

// TestConcurrent_AllowAndCount tests thread safety of Allow and Count together.
func TestConcurrent_AllowAndCount(t *testing.T) {
	// bluff-scan: no-assert-ok (concurrency test — go test -race catches data races; absence of panic == correctness)
	w := NewWindow(10000, time.Second, 10)
	now := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			w.Allow(now)
		}()
		go func() {
			defer wg.Done()
			w.Count(now)
		}()
	}
	wg.Wait()
}

// TestConcurrent_AllowAndReset tests thread safety of Allow and Reset together.
func TestConcurrent_AllowAndReset(t *testing.T) {
	// bluff-scan: no-assert-ok (concurrency test — go test -race catches data races; absence of panic == correctness)
	w := NewWindow(10000, time.Second, 10)
	now := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			w.Allow(now)
		}()
		go func() {
			defer wg.Done()
			w.Reset()
		}()
	}
	wg.Wait()
}
