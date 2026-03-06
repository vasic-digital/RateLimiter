// Package tokenbucket provides a token bucket rate limiter.
//
// It allows burst traffic while maintaining an average rate.
// Thread-safe via sync.Mutex.
package tokenbucket

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm.
type TokenBucket struct {
	mu           sync.Mutex
	capacity     int
	tokens       int
	refillRate   float64 // tokens per second
	lastRefillAt time.Time
}

// New creates a TokenBucket with given capacity and refill rate (tokens/sec).
func New(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:     capacity,
		tokens:       capacity,
		refillRate:   refillRate,
		lastRefillAt: time.Now(),
	}
}

// TryAcquire attempts to acquire a token. Returns true if successful.
func (tb *TokenBucket) TryAcquire() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refillLocked()

	if tb.tokens <= 0 {
		return false
	}
	tb.tokens--
	return true
}

// Acquire blocks until a token is available.
func (tb *TokenBucket) Acquire() {
	for !tb.TryAcquire() {
		time.Sleep(50 * time.Millisecond)
	}
}

// AvailableTokens returns the current number of available tokens.
func (tb *TokenBucket) AvailableTokens() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refillLocked()
	return tb.tokens
}

func (tb *TokenBucket) refillLocked() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillAt)

	if elapsed > 0 {
		tokensToAdd := int(elapsed.Seconds() * tb.refillRate)
		if tokensToAdd > 0 {
			tb.tokens += tokensToAdd
			if tb.tokens > tb.capacity {
				tb.tokens = tb.capacity
			}
			tb.lastRefillAt = now
		}
	}
}
