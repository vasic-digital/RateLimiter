# Lesson 2: In-Memory and Redis Backends

## Objectives

- Use the in-memory rate limiter for single-instance deployments
- Use the Redis-backed limiter for distributed rate limiting
- Understand the Lua script approach for atomicity

## Concepts

### In-Memory Rate Limiter

`memory.RateLimiter` maintains a separate `sliding.Window` per key. A background goroutine cleans up idle windows every minute to prevent memory leaks.

### Redis Rate Limiter

`redis.RateLimiter` uses a Lua script executed atomically in Redis. Each key maps to a sorted set where members are unique request IDs and scores are timestamps. The script:

1. Removes entries outside the current window (`ZREMRANGEBYSCORE`)
2. Counts remaining entries (`ZCARD`)
3. If under the limit, adds the new request (`ZADD`) and sets expiry (`PEXPIRE`)
4. Returns allowed/remaining/retry-after in a single atomic operation

### Functional Options

The Redis limiter supports functional options:

```go
rl := redis.New(client, config, redis.WithPrefix("myapp:ratelimit:"))
```

## Code Walkthrough

### In-memory limiter

```go
rl := memory.New(&limiter.Config{
    Rate:   100,
    Window: time.Minute,
    Burst:  150, // allow bursts up to 150
})
defer rl.Stop() // stop background cleanup

result, err := rl.Allow(ctx, "client:10.0.0.1")
if !result.Allowed {
    fmt.Printf("Retry after: %s\n", result.RetryAfter)
}
```

### Reset a key

```go
rl.Reset(ctx, "client:10.0.0.1") // removes the sliding window
```

### Redis limiter

```go
import goredis "github.com/redis/go-redis/v9"

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})

rl := redis.New(client, &limiter.Config{
    Rate:   1000,
    Window: time.Hour,
}, redis.WithPrefix("api:"))

result, err := rl.Allow(ctx, "apikey:abc123")
```

The Lua script ensures that the check-and-increment is atomic even under high concurrency across multiple application instances.

### Swapping backends

Because both implement `limiter.Limiter`, you can swap between them:

```go
var rl limiter.Limiter
if useRedis {
    rl = redis.New(redisClient, config)
} else {
    rl = memory.New(config)
}
```

## Summary

The in-memory limiter is fast and dependency-free for single instances. The Redis limiter provides distributed rate limiting with atomic Lua scripts. Both implement the same `Limiter` interface, making backend selection a deployment decision rather than a code change.
