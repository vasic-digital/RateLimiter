# Lesson 1: Core Interface and Sliding Window

## Objectives

- Understand the `Limiter` interface and `Result` type
- Configure rate, window, and burst parameters
- Learn how the sliding window counter algorithm works

## Concepts

### The Limiter Interface

```go
type Limiter interface {
    Allow(ctx context.Context, key string) (*Result, error)
    Reset(ctx context.Context, key string) error
}
```

### The Result Type

```go
type Result struct {
    Allowed    bool
    Remaining  int
    Limit      int
    RetryAfter time.Duration
    ResetAt    time.Time
}
```

- `Allowed` -- whether the request should proceed
- `Remaining` -- how many requests are left in the current window
- `RetryAfter` -- how long to wait before retrying (only set when rejected)
- `ResetAt` -- when the current window resets

### Configuration

```go
type Config struct {
    Rate   int           // requests allowed per window
    Window time.Duration // time window
    Burst  int           // max burst size (0 = same as Rate)
}
```

`EffectiveBurst()` returns `Burst` if set, otherwise `Rate`. This centralizes the burst-defaults-to-rate logic.

### Sliding Window Algorithm

The `sliding.Window` divides time into sub-windows (configurable granularity). Each sub-window tracks its own count. The total count is the sum across all sub-windows within the current window. This provides smoother rate limiting than a fixed window, avoiding the burst problem at boundaries.

## Code Walkthrough

### Creating a window

```go
// 100 requests per minute, divided into 10 sub-windows
w := sliding.NewWindow(100, time.Minute, 10)
```

### Checking and recording

```go
allowed, count, resetAt := w.Allow(time.Now())
// allowed=true, count=1, resetAt=now+1min
```

Each `Allow` call:
1. Cleans up expired sub-windows
2. Counts total requests in the current window
3. If under the rate limit, records the request in the current sub-window
4. Returns the result

### Querying without recording

```go
count := w.Count(time.Now()) // reads without incrementing
```

### Sub-window cleanup

Expired sub-windows (older than the full window duration) are removed on every `Allow` and `Count` call, preventing memory growth.

## Practice Exercise

1. Create a `sliding.Window` with rate=10 and a 1-second window. Call `Allow` 10 times rapidly and verify all succeed. Call an 11th time and verify it is rejected with `Allowed=false` and `RetryAfter > 0`.
2. Test the sliding behavior: allow 5 requests, wait half a window period, then verify the count has dropped (old sub-windows expired). Allow 5 more and verify they succeed.
3. Verify `EffectiveBurst()` logic: create a `Config` with `Rate=100, Burst=0` and verify effective burst is 100. Set `Burst=200` and verify it returns 200.
