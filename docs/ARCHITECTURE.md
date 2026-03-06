# RateLimiter -- Architecture

## Purpose

`digital.vasic.ratelimiter` provides pluggable rate limiting for Go services. It ships two implementations behind a common interface: an in-memory sliding window limiter for single-instance deployments and a Redis-backed distributed limiter for multi-instance environments. An HTTP middleware adapter makes it straightforward to add rate limiting to any `net/http` compatible router.

## Package Overview

| Package | Import Path | Responsibility |
|---------|-------------|----------------|
| `limiter` | `digital.vasic.ratelimiter/pkg/limiter` | Core `Limiter` interface, `Config` struct, and `Result` type. This is the contract all implementations satisfy. |
| `sliding` | `digital.vasic.ratelimiter/pkg/sliding` | Sliding window counter algorithm. Divides time into configurable sub-windows for smooth rate limiting without fixed-window boundary bursts. |
| `memory` | `digital.vasic.ratelimiter/pkg/memory` | In-memory `Limiter` implementation. Maintains a `sliding.Window` per key with automatic background cleanup of idle windows. |
| `redis` | `digital.vasic.ratelimiter/pkg/redis` | Redis-backed distributed `Limiter` implementation. Uses an atomic Lua script operating on sorted sets for race-free sliding window counting. |
| `middleware` | `digital.vasic.ratelimiter/pkg/middleware` | HTTP middleware adapter. Extracts a key from each request, calls `Limiter.Allow`, sets standard rate limit headers, and returns 429 when exceeded. |

## Design Patterns

### Strategy (`limiter` + `memory` + `redis`)

The `Limiter` interface defines a single `Allow(ctx, key)` method. The `memory` and `redis` packages provide interchangeable implementations. Consumers program against the interface and swap backends via configuration without changing application code.

### Template Method (`sliding`)

`sliding.Window` encapsulates the sliding window algorithm -- sub-window bucketing, cleanup, counting, and expiry calculation -- that both the in-memory limiter and (conceptually) the Redis Lua script follow. The in-memory limiter delegates directly to `Window`; the Redis limiter reimplements the same logic atomically in Lua.

### Middleware / Decorator (`middleware`)

`HTTPMiddleware` wraps an `http.Handler` with rate limiting behavior. It is a pure decorator: it checks the limiter, sets response headers, and either forwards to the next handler or short-circuits with 429. The middleware is fail-open -- if the limiter returns an error, the request is allowed through.

### Functional Options (`redis`)

The Redis limiter uses the functional options pattern (`WithPrefix`) for optional configuration, keeping the constructor signature clean while allowing extensibility.

## Dependency Diagram

```
+-------------------------------+
|        Consumer code          |
+-------------------------------+
        |               |
        v               v
+---------------+  +--------------------+
| pkg/middleware |  | (direct Limiter    |
| HTTPMiddleware |  |  usage)            |
+---------------+  +--------------------+
        |               |
        v               v
+-------------------------------+
|       pkg/limiter             |
|   Limiter interface           |
|   Config, Result              |
+-------------------------------+
      ^               ^
      |               |
+-------------+  +-------------+
| pkg/memory  |  | pkg/redis   |
| RateLimiter |  | RateLimiter |
+-------------+  +-------------+
      |                  |
      v                  v
+-------------+  +-------------------+
| pkg/sliding |  | go-redis/v9       |
| Window      |  | (Lua script on    |
|             |  |  sorted sets)     |
+-------------+  +-------------------+
```

## Key Interfaces

### Limiter (`limiter`)

```go
type Limiter interface {
    Allow(ctx context.Context, key string) (*Result, error)
    Reset(ctx context.Context, key string) error
}
```

The central abstraction. `Allow` checks whether a request identified by `key` is permitted and returns quota information. `Reset` clears the counter for a key.

### Result (`limiter`)

```go
type Result struct {
    Allowed    bool          // Whether the request is allowed
    Remaining  int           // Requests remaining in the current window
    Limit      int           // Maximum requests per window
    RetryAfter time.Duration // How long to wait before retrying (only set when denied)
    ResetAt    time.Time     // When the current window resets
}
```

### Config (`limiter`)

```go
type Config struct {
    Rate   int           // Requests allowed per window
    Window time.Duration // Window duration
    Burst  int           // Max burst size (0 defaults to Rate)
}
```

### KeyFunc (`middleware`)

```go
type KeyFunc func(*http.Request) string
```

Extracts the rate limiting key from an HTTP request. Built-in implementations: `IPKeyFunc()` (by remote address) and `HeaderKeyFunc(header)` (by header value with IP fallback).

### OnLimited (`middleware`)

```go
type OnLimited func(w http.ResponseWriter, r *http.Request, result *limiter.Result)
```

Callback invoked when a request is rate limited. The default returns a 429 JSON response with `Retry-After` header.

## Usage Example

```go
package main

import (
    "net/http"
    "time"

    "digital.vasic.ratelimiter/pkg/limiter"
    "digital.vasic.ratelimiter/pkg/memory"
    "digital.vasic.ratelimiter/pkg/middleware"
)

func main() {
    // Create an in-memory limiter: 100 requests per minute
    rl := memory.New(&limiter.Config{
        Rate:   100,
        Window: time.Minute,
    })
    defer rl.Stop()

    // Wrap your handler with rate limiting middleware
    mux := http.NewServeMux()
    mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"ok": true}`))
    })

    // Apply middleware keyed by client IP
    handler := middleware.HTTPMiddleware(rl, middleware.IPKeyFunc())(mux)

    http.ListenAndServe(":8080", handler)
}
```

For distributed deployments, swap `memory.New` with `redis.New`:

```go
import (
    goredis "github.com/redis/go-redis/v9"
    "digital.vasic.ratelimiter/pkg/redis"
)

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
rl := redis.New(client, &limiter.Config{
    Rate:   100,
    Window: time.Minute,
}, redis.WithPrefix("myapp:"))
```
