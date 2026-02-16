package memory

import (
	"context"
	"sync"
	"time"

	"digital.vasic.ratelimiter/pkg/limiter"
	"digital.vasic.ratelimiter/pkg/sliding"
)

// RateLimiter is an in-memory sliding window rate limiter.
// It maintains a separate sliding window for each key, making it
// suitable for per-client rate limiting in single-instance deployments.
type RateLimiter struct {
	mu      sync.RWMutex
	windows map[string]*sliding.Window
	config  *limiter.Config
	stopCh  chan struct{}
	stopped bool
}

// New creates a new in-memory rate limiter with the given configuration.
// If config is nil, DefaultConfig() is used.
// The limiter starts a background goroutine to clean up expired windows.
// Call Stop() when the limiter is no longer needed.
func New(config *limiter.Config) *RateLimiter {
	if config == nil {
		config = limiter.DefaultConfig()
	}

	rl := &RateLimiter{
		windows: make(map[string]*sliding.Window),
		config:  config,
		stopCh:  make(chan struct{}),
	}

	go rl.cleanup()

	return rl
}

// Allow checks whether a request identified by key should be allowed.
// It returns a Result indicating whether the request is allowed,
// the remaining quota, and when the window resets.
func (rl *RateLimiter) Allow(_ context.Context, key string) (*limiter.Result, error) {
	w := rl.getOrCreateWindow(key)
	now := time.Now()

	allowed, currentCount, resetAt := w.Allow(now)

	effectiveLimit := rl.config.EffectiveBurst()

	remaining := effectiveLimit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	result := &limiter.Result{
		Allowed:   allowed,
		Remaining: remaining,
		Limit:     effectiveLimit,
		ResetAt:   resetAt,
	}

	if !allowed {
		result.RetryAfter = time.Until(resetAt)
		if result.RetryAfter < 0 {
			result.RetryAfter = 0
		}
	}

	return result, nil
}

// Reset removes the sliding window for the given key, effectively
// resetting the rate limit counter for that key.
func (rl *RateLimiter) Reset(_ context.Context, key string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.windows, key)
	return nil
}

// Stop stops the background cleanup goroutine.
// The limiter should not be used after Stop is called.
func (rl *RateLimiter) Stop() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.stopped {
		close(rl.stopCh)
		rl.stopped = true
	}
}

// getOrCreateWindow returns the sliding window for the given key,
// creating one if it does not exist.
func (rl *RateLimiter) getOrCreateWindow(key string) *sliding.Window {
	rl.mu.RLock()
	w, ok := rl.windows[key]
	rl.mu.RUnlock()

	if ok {
		return w
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if w, ok := rl.windows[key]; ok {
		return w
	}

	w = sliding.NewWindow(rl.config.EffectiveBurst(), rl.config.Window, 10)
	rl.windows[key] = w
	return w
}

// cleanup periodically removes idle windows to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, w := range rl.windows {
				// If the window has no requests in the current period, remove it
				if w.Count(now) == 0 {
					delete(rl.windows, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}
