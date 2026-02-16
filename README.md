# digital.vasic.ratelimiter

A Go module providing rate limiting with in-memory and Redis-backed implementations, plus HTTP middleware.

## Features

- **Sliding window algorithm** for smooth, accurate rate limiting without burst problems at window boundaries.
- **In-memory limiter** for single-instance deployments with automatic cleanup of idle keys.
- **Redis-backed limiter** for distributed, multi-instance deployments using atomic Lua scripts.
- **HTTP middleware** adapter compatible with any `net/http` router (standard library, chi, gorilla/mux, etc.).
- **Fail-open design** in middleware: if the limiter encounters an error, requests are allowed through.

## Installation

```bash
go get digital.vasic.ratelimiter
```

## Usage

### In-Memory Rate Limiter

```go
package main

import (
    "context"
    "fmt"

    "digital.vasic.ratelimiter/pkg/limiter"
    "digital.vasic.ratelimiter/pkg/memory"
    "time"
)

func main() {
    cfg := &limiter.Config{
        Rate:   100,
        Window: time.Minute,
        Burst:  0, // defaults to Rate
    }

    rl := memory.New(cfg)
    defer rl.Stop()

    result, err := rl.Allow(context.Background(), "user:123")
    if err != nil {
        panic(err)
    }

    if result.Allowed {
        fmt.Printf("Allowed. %d requests remaining.\n", result.Remaining)
    } else {
        fmt.Printf("Rate limited. Retry after %s.\n", result.RetryAfter)
    }
}
```

### Redis-Backed Rate Limiter

```go
package main

import (
    "context"
    "fmt"
    "time"

    "digital.vasic.ratelimiter/pkg/limiter"
    rlredis "digital.vasic.ratelimiter/pkg/redis"
    goredis "github.com/redis/go-redis/v9"
)

func main() {
    client := goredis.NewClient(&goredis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    cfg := &limiter.Config{
        Rate:   1000,
        Window: time.Minute,
        Burst:  1200,
    }

    rl := rlredis.New(client, cfg, rlredis.WithPrefix("myapp:"))

    result, err := rl.Allow(context.Background(), "api-key:abc")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Allowed: %v, Remaining: %d\n", result.Allowed, result.Remaining)
}
```

### HTTP Middleware

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
    cfg := &limiter.Config{
        Rate:   60,
        Window: time.Minute,
    }

    rl := memory.New(cfg)
    defer rl.Stop()

    mux := http.NewServeMux()
    mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })

    // Apply rate limiting middleware
    handler := middleware.HTTPMiddleware(rl, middleware.IPKeyFunc())(mux)

    http.ListenAndServe(":8080", handler)
}
```

### Custom Key Functions and Error Handling

```go
// Rate limit by API key header
handler := middleware.HTTPMiddleware(rl, middleware.HeaderKeyFunc("X-API-Key"))(mux)

// Custom on-limited handler
opts := &middleware.Options{
    KeyFunc: middleware.IPKeyFunc(),
    OnLimited: func(w http.ResponseWriter, r *http.Request, result *limiter.Result) {
        w.Header().Set("Content-Type", "text/plain")
        w.WriteHeader(http.StatusTooManyRequests)
        fmt.Fprintf(w, "Slow down! Try again in %s", result.RetryAfter)
    },
}
handler := middleware.HTTPMiddlewareWithOptions(rl, opts)(mux)
```

## Package Structure

| Package | Description |
|---|---|
| `pkg/limiter` | Core interfaces (`Limiter`, `Config`, `Result`) |
| `pkg/sliding` | Sliding window counter algorithm |
| `pkg/memory` | In-memory rate limiter with background cleanup |
| `pkg/redis` | Redis-backed distributed rate limiter |
| `pkg/middleware` | HTTP middleware adapter |

## Testing

```bash
go test ./... -count=1
```

Redis tests use [miniredis](https://github.com/alicebob/miniredis) and require no running Redis instance.

## License

See LICENSE file.
