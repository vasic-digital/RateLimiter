# Examples

## API Rate Limiting by Header

Rate limit API requests by an API key header, falling back to IP:

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
        Rate:   1000,
        Window: time.Hour,
    })
    defer rl.Stop()

    mux := http.NewServeMux()
    mux.HandleFunc("/api/", apiHandler)

    // Key by X-API-Key header, fallback to IP
    handler := middleware.HTTPMiddleware(rl,
        middleware.HeaderKeyFunc("X-API-Key"),
    )(mux)

    http.ListenAndServe(":8080", handler)
}
```

## Adaptive Rate Limiting

Automatically adjust the rate based on operation success/failure:

```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "digital.vasic.ratelimiter/pkg/adaptive"
)

func main() {
    // Start at 50 req/s, adapt between 10 and 200
    arl := adaptive.New(50, 10, 200)

    ctx := context.Background()

    // Wrap operations -- rate adjusts automatically
    for i := 0; i < 100; i++ {
        err := arl.Execute(ctx, func(ctx context.Context) error {
            resp, err := http.Get("https://api.example.com/data")
            if err != nil {
                return err
            }
            resp.Body.Close()
            return nil
        })
        if err != nil {
            fmt.Printf("Request %d failed: %v\n", i, err)
        }
    }

    fmt.Printf("Current rate: %d\n", arl.CurrentRate())
    // Rate increases after 10 consecutive successes
    // Rate decreases after 3 consecutive failures
}
```

## Per-Operation Throttling

Throttle specific operations independently within time windows:

```go
package main

import (
    "fmt"
    "time"

    "digital.vasic.ratelimiter/pkg/throttler"
)

func main() {
    // Max 5 operations per 10-second window, per operation ID
    t := throttler.New(10*time.Second, 5)

    // Different operations have independent limits
    for i := 0; i < 10; i++ {
        if t.TryThrottle("send-email") {
            fmt.Printf("Email %d: allowed\n", i)
        } else {
            fmt.Printf("Email %d: throttled\n", i)
        }

        if t.TryThrottle("api-call") {
            fmt.Printf("API call %d: allowed\n", i)
        } else {
            fmt.Printf("API call %d: throttled\n", i)
        }
    }

    // Reset throttle for a specific operation
    t.Clear("send-email")
}
```
