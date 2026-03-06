# RateLimiter Module

`digital.vasic.ratelimiter` provides pluggable rate limiting for Go services. It ships in-memory and Redis-backed implementations behind a common `Limiter` interface, an HTTP middleware adapter, a Gin middleware adapter, a token bucket, a per-operation throttler, and an adaptive rate limiter.

## Key Features

- **Limiter interface** -- `Allow(ctx, key)` returns quota information; swap backends without changing application code
- **In-memory limiter** -- Sliding window counter per key with automatic background cleanup
- **Redis limiter** -- Distributed rate limiting via atomic Lua scripts on sorted sets
- **HTTP middleware** -- Standard `net/http` middleware with `KeyFunc`, rate limit headers, and 429 responses
- **Gin middleware** -- Gin-native rate limiting adapter
- **Token bucket** -- Classic token bucket algorithm with configurable capacity and refill rate
- **Throttler** -- Per-operation-ID throttling within time windows
- **Adaptive limiter** -- Auto-adjusts rate based on success/failure feedback

## Package Overview

| Package | Purpose |
|---------|---------|
| `pkg/limiter` | Core `Limiter` interface, `Config`, and `Result` types |
| `pkg/sliding` | Sliding window counter algorithm |
| `pkg/memory` | In-memory `Limiter` with per-key sliding windows |
| `pkg/redis` | Redis-backed distributed `Limiter` (Lua scripts) |
| `pkg/middleware` | HTTP middleware with key extraction and 429 handling |
| `pkg/gin` | Gin framework middleware adapter |
| `pkg/tokenbucket` | Token bucket rate limiter |
| `pkg/throttler` | Per-operation throttling within time windows |
| `pkg/adaptive` | Adaptive rate limiter that adjusts based on outcomes |

## Installation

```bash
go get digital.vasic.ratelimiter
```

Requires Go 1.24 or later.

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| `github.com/redis/go-redis/v9` | Redis client (for `pkg/redis` only) |
| `github.com/gin-gonic/gin` | Gin framework (for `pkg/gin` only) |
