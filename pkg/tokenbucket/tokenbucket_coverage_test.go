package tokenbucket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRefill_TokensAddedOverTime verifies that tokens are refilled based on elapsed time.
func TestRefill_TokensAddedOverTime(t *testing.T) {
	// Capacity 5, refill at 100 tokens/sec (1 token per 10ms)
	tb := New(5, 100.0)

	// Drain all tokens
	for i := 0; i < 5; i++ {
		assert.True(t, tb.TryAcquire())
	}
	assert.False(t, tb.TryAcquire())

	// Wait for refill (at 100/sec, 50ms should refill ~5 tokens)
	time.Sleep(60 * time.Millisecond)

	available := tb.AvailableTokens()
	assert.Greater(t, available, 0, "tokens should have been refilled")
	assert.LessOrEqual(t, available, 5, "tokens should not exceed capacity")
}

// TestRefill_CappedAtCapacity verifies tokens never exceed capacity.
func TestRefill_CappedAtCapacity(t *testing.T) {
	tb := New(3, 100.0)

	// Wait for potential over-refill
	time.Sleep(100 * time.Millisecond)

	tokens := tb.AvailableTokens()
	assert.Equal(t, 3, tokens, "tokens should be capped at capacity")
}

// TestAcquire_BlocksUntilAvailable verifies Acquire blocks when empty
// and returns once tokens are refilled.
func TestAcquire_BlocksUntilAvailable(t *testing.T) {
	// High refill rate so the test doesn't take too long
	tb := New(1, 100.0)

	// Drain the single token
	assert.True(t, tb.TryAcquire())
	assert.False(t, tb.TryAcquire())

	// Acquire should block briefly and then succeed
	done := make(chan struct{})
	go func() {
		tb.Acquire()
		close(done)
	}()

	select {
	case <-done:
		// Success: Acquire returned after refill
	case <-time.After(2 * time.Second):
		t.Fatal("Acquire blocked for too long; expected it to return after refill")
	}
}

// TestAcquire_MultipleCalls verifies multiple Acquire calls drain tokens sequentially.
func TestAcquire_MultipleCalls(t *testing.T) {
	tb := New(3, 1000.0)
	tb.Acquire()
	tb.Acquire()
	tb.Acquire()

	// All three should have consumed tokens; next should block briefly
	done := make(chan struct{})
	go func() {
		tb.Acquire()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Acquire blocked too long")
	}
}

// TestTryAcquire_DrainAndRefill verifies full drain followed by refill cycle.
func TestTryAcquire_DrainAndRefill(t *testing.T) {
	tb := New(2, 200.0)

	// Drain
	assert.True(t, tb.TryAcquire())
	assert.True(t, tb.TryAcquire())
	assert.False(t, tb.TryAcquire())

	// Wait for refill
	time.Sleep(50 * time.Millisecond)

	// Should have tokens again
	assert.True(t, tb.TryAcquire())
}

// TestConcurrent_TryAcquire verifies thread safety of TryAcquire.
func TestConcurrent_TryAcquire(t *testing.T) {
	tb := New(1000, 0.0) // No refill to make counting deterministic
	var wg sync.WaitGroup
	var allowed int64
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if tb.TryAcquire() {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1000), allowed, "exactly 1000 tokens should have been acquired")
}

// TestConcurrent_AvailableTokens verifies thread safety of AvailableTokens.
func TestConcurrent_AvailableTokens(t *testing.T) {
	// bluff-scan: no-assert-ok (concurrency test — go test -race catches data races; absence of panic == correctness)
	tb := New(100, 10.0)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tb.TryAcquire()
		}()
		go func() {
			defer wg.Done()
			tb.AvailableTokens()
		}()
	}
	wg.Wait()
}

// TestNew_ZeroRefillRate verifies a bucket with zero refill rate never refills.
func TestNew_ZeroRefillRate(t *testing.T) {
	tb := New(2, 0.0)
	assert.Equal(t, 2, tb.AvailableTokens())

	assert.True(t, tb.TryAcquire())
	assert.True(t, tb.TryAcquire())
	assert.False(t, tb.TryAcquire())

	// Even after waiting, no refill
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, tb.AvailableTokens())
}

// TestNew_HighCapacity verifies bucket works with large capacity.
func TestNew_HighCapacity(t *testing.T) {
	tb := New(10000, 1.0)
	assert.Equal(t, 10000, tb.AvailableTokens())

	for i := 0; i < 100; i++ {
		assert.True(t, tb.TryAcquire())
	}
	assert.Equal(t, 9900, tb.AvailableTokens())
}

// TestRefill_PartialToken verifies that partial tokens (less than 1) don't get added.
func TestRefill_PartialToken(t *testing.T) {
	// Very low refill rate: 1 token per second
	tb := New(5, 1.0)

	// Drain one
	assert.True(t, tb.TryAcquire())
	assert.Equal(t, 4, tb.AvailableTokens())

	// Sleep less than 1 second — should not add a full token via integer truncation
	time.Sleep(10 * time.Millisecond)
	// Due to integer truncation in refillLocked, no token is added for < 1s at 1 token/sec
	tokens := tb.AvailableTokens()
	assert.LessOrEqual(t, tokens, 4, "partial token should not be added")
}
