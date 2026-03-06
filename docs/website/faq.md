# FAQ

## What is the difference between the in-memory and Redis limiter?

The **in-memory limiter** maintains sliding windows in process memory. It is fast and requires no external dependencies, but the counters are not shared between application instances. The **Redis limiter** uses atomic Lua scripts on Redis sorted sets, providing distributed rate limiting across multiple instances. Both implement the same `Limiter` interface and can be swapped without changing application code.

## What happens when the limiter returns an error in the HTTP middleware?

The HTTP middleware is **fail-open**: if the limiter returns an error (e.g., Redis is unreachable), the request is allowed through. This prevents rate limiter failures from causing a total service outage. The middleware sets `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` headers on every response.

## How does the sliding window algorithm avoid boundary bursts?

The sliding window divides each time window into configurable sub-windows. The count at any point is the sum of the current sub-window plus a weighted fraction of the previous sub-window based on how far into the current window the request falls. This smooths the transition between windows, unlike a fixed-window counter that resets abruptly.

## How does the adaptive rate limiter decide when to adjust?

The adaptive limiter tracks consecutive successes and failures. After 10 consecutive successful operations, the rate increases by 1 (up to `maxRate`). After 3 consecutive failures, the rate decreases by 1 (down to `minRate`). Counters reset after each adjustment. This provides gradual adaptation without oscillating.

## Can I use different rate limits for different endpoints?

Yes. Create separate `Limiter` instances with different `Config` values and apply them to different route groups. Alternatively, use a custom `KeyFunc` that incorporates the request path into the key, so each endpoint has its own counter within a single limiter instance.
