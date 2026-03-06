# Course: Rate Limiting for Go Services

## Module Overview

This course covers the `digital.vasic.ratelimiter` module, providing pluggable rate limiting with in-memory and Redis backends, sliding window and token bucket algorithms, HTTP middleware with standard rate limit headers, Gin integration, and adaptive rate control. You will learn to protect services from overload in both single-instance and distributed environments.

## Prerequisites

- Intermediate Go knowledge (interfaces, goroutines, HTTP middleware)
- Basic understanding of rate limiting concepts (windows, buckets)
- Familiarity with Redis (optional, for distributed lessons)
- Go 1.24+ installed

## Lessons

| # | Title | Duration |
|---|-------|----------|
| 1 | Core Interface and Sliding Window Algorithm | 40 min |
| 2 | In-Memory and Redis Backends | 45 min |
| 3 | HTTP Middleware, Gin, and Advanced Patterns | 45 min |

## Source Files

- `pkg/limiter/` -- Core `Limiter` interface, `Config`, `Result`
- `pkg/sliding/` -- Sliding window counter algorithm
- `pkg/memory/` -- In-memory rate limiter with background cleanup
- `pkg/redis/` -- Redis-backed distributed limiter (Lua scripts)
- `pkg/middleware/` -- HTTP middleware adapter
- `pkg/gin/` -- Gin framework integration
- `pkg/adaptive/` -- Adaptive rate limiting
- `pkg/tokenbucket/` -- Token bucket algorithm
- `pkg/throttler/` -- Per-operation throttling
