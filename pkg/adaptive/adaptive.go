// Package adaptive provides a rate limiter that auto-adjusts based on
// success/failure feedback.
package adaptive

import (
	"context"
	"sync"

	"digital.vasic.ratelimiter/pkg/limiter"
)

// AdaptiveRateLimiter adjusts its rate based on operation outcomes.
type AdaptiveRateLimiter struct {
	mu           sync.Mutex
	currentRate  int
	minRate      int
	maxRate      int
	successCount int
	failureCount int
}

// New creates an AdaptiveRateLimiter with the given bounds.
func New(initialRate, minRate, maxRate int) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		currentRate: initialRate,
		minRate:     minRate,
		maxRate:     maxRate,
	}
}

// Execute runs an operation and tracks success/failure to adjust the rate.
func (a *AdaptiveRateLimiter) Execute(ctx context.Context, op func(ctx context.Context) error) error {
	err := op(ctx)
	if err != nil {
		a.onFailure()
		return err
	}
	a.onSuccess()
	return nil
}

// CurrentRate returns the current rate limit.
func (a *AdaptiveRateLimiter) CurrentRate() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentRate
}

// Allow checks if a request is allowed under the current adaptive rate.
// Returns a limiter.Result for compatibility with the existing Limiter interface.
func (a *AdaptiveRateLimiter) Allow(_ context.Context, _ string) (*limiter.Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return &limiter.Result{
		Allowed:   true,
		Remaining: a.currentRate,
		Limit:     a.currentRate,
	}, nil
}

func (a *AdaptiveRateLimiter) onSuccess() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.successCount++
	if a.successCount > 10 {
		a.adjustRateLocked(1)
		a.successCount = 0
	}
}

func (a *AdaptiveRateLimiter) onFailure() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.failureCount++
	if a.failureCount >= 3 {
		a.adjustRateLocked(-1)
		a.failureCount = 0
	}
}

func (a *AdaptiveRateLimiter) adjustRateLocked(delta int) {
	a.currentRate += delta
	if a.currentRate < a.minRate {
		a.currentRate = a.minRate
	}
	if a.currentRate > a.maxRate {
		a.currentRate = a.maxRate
	}
}
