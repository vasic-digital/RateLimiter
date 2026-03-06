# Lesson 3: HTTP Middleware, Gin, and Advanced Patterns

## Objectives

- Add rate limiting to HTTP and Gin handlers
- Use the token bucket for burst-tolerant limiting
- Apply per-operation throttling and adaptive rate control

## Concepts

### HTTP Middleware

`middleware.HTTPMiddleware` wraps an `http.Handler` with rate limiting. It extracts a key from the request, calls `Allow`, sets standard rate limit headers, and returns 429 when the limit is exceeded.

Key features:
- **Fail-open** -- on limiter errors, requests pass through
- **Standard headers** -- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- **Customizable** -- `KeyFunc` for key extraction, `OnLimited` for custom rejection responses

### Gin Middleware

`gin.RateLimit` provides a native Gin handler function. On limiter error, it returns 500. On rate limit exceeded, it returns 429 with a JSON body.

### Token Bucket

`tokenbucket.TokenBucket` allows burst traffic while maintaining an average rate. Tokens are consumed on each request and refilled at a constant rate. When tokens are exhausted, requests are rejected (or blocked with `Acquire`).

### Throttler

`throttler.Throttler` limits operations by ID within time windows. Unlike the `Limiter` interface, it uses a simpler check-and-count approach without the sliding window complexity.

### Adaptive Rate Limiter

`adaptive.AdaptiveRateLimiter` adjusts its rate based on operation outcomes. After 10 consecutive successes, the rate increases by 1. After 3 consecutive failures, it decreases by 1. The rate stays within configured min/max bounds.

## Code Walkthrough

### HTTP middleware with custom key

```go
handler := middleware.HTTPMiddlewareWithOptions(rl, &middleware.Options{
    KeyFunc: middleware.HeaderKeyFunc("X-API-Key"),
    OnLimited: func(w http.ResponseWriter, r *http.Request, result *limiter.Result) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusTooManyRequests)
        fmt.Fprintf(w, `{"error":"slow down","retry_after_ms":%d}`,
            result.RetryAfter.Milliseconds())
    },
})
```

### Gin middleware

```go
r := gin.Default()
r.Use(ginrl.RateLimit(rl, ginrl.IPKeyFunc()))
r.GET("/api/data", dataHandler)
```

### Token bucket

```go
tb := tokenbucket.New(100, 20.0) // 100 capacity, 20 tokens/sec

// Non-blocking check
if tb.TryAcquire() {
    handleRequest()
}

// Blocking wait
tb.Acquire() // polls every 50ms until a token is available

fmt.Printf("Tokens left: %d\n", tb.AvailableTokens())
```

### Per-operation throttling

```go
t := throttler.New(10*time.Second, 5) // 5 ops per 10s window

if t.TryThrottle("send-notification") {
    sendNotification()
} else {
    log.Println("notification throttled")
}

t.Clear("send-notification") // reset the counter
```

### Adaptive rate limiting

```go
arl := adaptive.New(50, 10, 200) // initial=50, min=10, max=200

err := arl.Execute(ctx, func(ctx context.Context) error {
    return callExternalAPI(ctx)
})
// Rate increases after 10 successes, decreases after 3 failures

fmt.Printf("Current rate: %d\n", arl.CurrentRate())
```

## Summary

The middleware packages provide drop-in rate limiting for HTTP and Gin. The token bucket handles burst scenarios. The throttler provides simple per-operation limits. The adaptive limiter automatically adjusts to the health of external dependencies, backing off when they struggle and ramping up when they recover.
