# Getting Started

## Installation

```bash
go get digital.vasic.ratelimiter
```

## In-Memory Rate Limiting

Create an in-memory limiter with a sliding window:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "digital.vasic.ratelimiter/pkg/limiter"
    "digital.vasic.ratelimiter/pkg/memory"
)

func main() {
    rl := memory.New(&limiter.Config{
        Rate:   100,          // 100 requests
        Window: time.Minute,  // per minute
    })
    defer rl.Stop()

    ctx := context.Background()
    result, err := rl.Allow(ctx, "user:123")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Allowed: %v, Remaining: %d/%d\n",
        result.Allowed, result.Remaining, result.Limit)
}
```

## HTTP Middleware

Add rate limiting to any `net/http` handler:

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
    rl := memory.New(&limiter.Config{
        Rate:   60,
        Window: time.Minute,
    })
    defer rl.Stop()

    mux := http.NewServeMux()
    mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"ok": true}`))
    })

    // Rate limit by client IP
    handler := middleware.HTTPMiddleware(rl, middleware.IPKeyFunc())(mux)
    http.ListenAndServe(":8080", handler)
}
```

## Redis-Backed Distributed Limiter

For multi-instance deployments, swap to the Redis backend:

```go
package main

import (
    "time"

    goredis "github.com/redis/go-redis/v9"
    "digital.vasic.ratelimiter/pkg/limiter"
    "digital.vasic.ratelimiter/pkg/redis"
)

func main() {
    client := goredis.NewClient(&goredis.Options{
        Addr: "localhost:6379",
    })

    rl := redis.New(client, &limiter.Config{
        Rate:   100,
        Window: time.Minute,
    }, redis.WithPrefix("myapp:"))

    // Same Limiter interface as the in-memory version
    result, _ := rl.Allow(ctx, "user:123")
    fmt.Printf("Allowed: %v\n", result.Allowed)
}
```

## Token Bucket

Use the token bucket for burst-tolerant rate limiting:

```go
import "digital.vasic.ratelimiter/pkg/tokenbucket"

// 50 token capacity, refills at 10 tokens/sec
tb := tokenbucket.New(50, 10.0)

if tb.TryAcquire() {
    fmt.Println("Request allowed")
}

fmt.Printf("Available tokens: %d\n", tb.AvailableTokens())

// Blocking acquire
tb.Acquire() // waits until a token is available
```

## Gin Middleware

Rate limit Gin routes:

```go
import (
    "github.com/gin-gonic/gin"
    ginrl "digital.vasic.ratelimiter/pkg/gin"
)

r := gin.Default()
r.Use(ginrl.RateLimit(rl, ginrl.IPKeyFunc()))
```
