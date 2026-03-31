# Architecture -- RateLimiter

## Purpose

Go module providing rate limiting with in-memory and Redis-backed implementations plus HTTP middleware. Uses a sliding window algorithm for smooth, accurate rate limiting without burst problems at window boundaries.

## Structure

```
pkg/
  limiter/      Core interfaces: Limiter, Config, Result
  sliding/      Sliding window counter algorithm (divides time into sub-windows)
  memory/       In-memory rate limiter with background cleanup of idle keys
  redis/        Redis-backed distributed rate limiter using atomic Lua scripts
  middleware/   HTTP middleware adapter compatible with any net/http router
```

## Key Components

- **`limiter.Limiter`** -- Interface: Allow(ctx, key) returning Result{Allowed, Remaining, RetryAfter, ResetAt}
- **`limiter.Config`** -- Rate (requests), Window (duration), Burst (defaults to Rate)
- **`memory.Limiter`** -- In-memory implementation using sliding window counters with periodic background cleanup
- **`redis.Limiter`** -- Distributed implementation using atomic Lua scripts for race-free Redis operations
- **`middleware.HTTPMiddleware`** -- Wraps any Limiter as `func(http.Handler) http.Handler`; fail-open design (errors allow requests through)
- **`middleware.IPKeyFunc`** / **`middleware.HeaderKeyFunc`** -- Key extraction functions for identifying clients

## Data Flow

```
HTTP Request -> middleware.HTTPMiddleware(limiter, keyFunc)(handler)
    |
    keyFunc(request) -> "user:123" or IP address
    |
    limiter.Allow(ctx, key) -> sliding window check
        |
        Result.Allowed? -> next handler
        !Allowed? -> 429 Too Many Requests (with Retry-After header)
```

## Dependencies

- `github.com/redis/go-redis/v9` -- Redis client
- `github.com/alicebob/miniredis/v2` -- In-process Redis for tests
- `github.com/stretchr/testify` -- Test assertions

## Testing Strategy

Table-driven tests with `testify`. Redis tests use miniredis (no running Redis required). Tests cover rate limit enforcement, burst handling, window sliding, key cleanup, middleware integration with httptest, IP and header key extraction, and fail-open error behavior.
