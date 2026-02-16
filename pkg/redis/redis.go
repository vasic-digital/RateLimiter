package redis

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"digital.vasic.ratelimiter/pkg/limiter"
	goredis "github.com/redis/go-redis/v9"
)

// luaSlidingWindowScript is an atomic Lua script that implements
// a sliding window rate limiter in Redis using a sorted set.
// Members are unique request IDs (timestamp + counter), scores are timestamps.
const luaSlidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

-- Remove entries outside the current window
local window_start = now - window
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Count current entries
local count = redis.call('ZCARD', key)

if count < limit then
    -- Add the new request
    redis.call('ZADD', key, now, member)
    -- Set expiry on the key to auto-cleanup
    redis.call('PEXPIRE', key, window + 1000)
    return {1, limit - count - 1, 0}
else
    -- Get the oldest entry to calculate retry-after
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local retry_after = 0
    if #oldest >= 2 then
        retry_after = tonumber(oldest[2]) + window - now
        if retry_after < 0 then
            retry_after = 0
        end
    end
    return {0, 0, retry_after}
end
`

// RateLimiter is a Redis-backed distributed rate limiter.
// It uses a sliding window algorithm implemented via a Lua script
// for atomic, race-condition-free operation across multiple instances.
type RateLimiter struct {
	client  goredis.Cmdable
	config  *limiter.Config
	prefix  string
	script  *goredis.Script
	counter uint64
}

// Option is a functional option for configuring the Redis rate limiter.
type Option func(*RateLimiter)

// WithPrefix sets a key prefix for all Redis keys used by this limiter.
func WithPrefix(prefix string) Option {
	return func(rl *RateLimiter) {
		rl.prefix = prefix
	}
}

// New creates a new Redis-backed rate limiter.
// The client parameter must be a connected Redis client (or compatible, like miniredis).
// If config is nil, DefaultConfig() is used.
func New(client goredis.Cmdable, config *limiter.Config, opts ...Option) *RateLimiter {
	if config == nil {
		config = limiter.DefaultConfig()
	}

	rl := &RateLimiter{
		client: client,
		config: config,
		prefix: "ratelimit:",
		script: goredis.NewScript(luaSlidingWindowScript),
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}

// Allow checks whether a request identified by key should be allowed.
// The check and update are performed atomically in Redis using a Lua script.
func (rl *RateLimiter) Allow(ctx context.Context, key string) (*limiter.Result, error) {
	redisKey := rl.prefix + key
	now := time.Now()
	nowMs := now.UnixMilli()
	windowMs := rl.config.Window.Milliseconds()
	effectiveLimit := rl.config.EffectiveBurst()

	// Generate a unique member ID for this request (atomic for concurrency)
	cnt := atomic.AddUint64(&rl.counter, 1)
	member := fmt.Sprintf("%d:%d", nowMs, cnt)

	raw, err := rl.script.Run(ctx, rl.client, []string{redisKey},
		nowMs,
		windowMs,
		effectiveLimit,
		member,
	).Int64Slice()

	if err != nil {
		return nil, fmt.Errorf("rate limiter lua script failed: %w", err)
	}

	if len(raw) < 3 {
		return nil, fmt.Errorf("unexpected lua script result length: %d", len(raw))
	}

	allowed := raw[0] == 1
	remaining := int(raw[1])
	retryAfterMs := raw[2]

	result := &limiter.Result{
		Allowed:   allowed,
		Remaining: remaining,
		Limit:     effectiveLimit,
		ResetAt:   now.Add(rl.config.Window),
	}

	if !allowed {
		result.RetryAfter = time.Duration(retryAfterMs) * time.Millisecond
	}

	return result, nil
}

// Reset removes the rate limit data for the given key from Redis.
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := rl.prefix + key
	return rl.client.Del(ctx, redisKey).Err()
}
