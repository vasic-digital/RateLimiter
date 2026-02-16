package limiter

import (
	"context"
	"time"
)

// Result represents the outcome of a rate limit check.
type Result struct {
	Allowed    bool
	Remaining  int
	Limit      int
	RetryAfter time.Duration
	ResetAt    time.Time
}

// Limiter defines the rate limiting interface.
type Limiter interface {
	Allow(ctx context.Context, key string) (*Result, error)
	Reset(ctx context.Context, key string) error
}

// Config holds rate limiter configuration.
type Config struct {
	Rate   int           // Number of requests allowed
	Window time.Duration // Time window for the rate
	Burst  int           // Maximum burst size (0 = same as Rate)
}

// EffectiveBurst returns the burst value, defaulting to Rate if Burst is 0.
func (c *Config) EffectiveBurst() int {
	if c.Burst <= 0 {
		return c.Rate
	}
	return c.Burst
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Rate:   100,
		Window: time.Minute,
		Burst:  0,
	}
}
