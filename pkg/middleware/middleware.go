package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"digital.vasic.ratelimiter/pkg/limiter"
)

// KeyFunc extracts a rate limiting key from an HTTP request.
// Common implementations key by IP address, user ID, API key, etc.
type KeyFunc func(*http.Request) string

// IPKeyFunc returns a KeyFunc that uses the request's RemoteAddr as the key.
func IPKeyFunc() KeyFunc {
	return func(r *http.Request) string {
		return r.RemoteAddr
	}
}

// HeaderKeyFunc returns a KeyFunc that uses the value of the specified
// header as the key, falling back to RemoteAddr if the header is absent.
func HeaderKeyFunc(header string) KeyFunc {
	return func(r *http.Request) string {
		if val := r.Header.Get(header); val != "" {
			return val
		}
		return r.RemoteAddr
	}
}

// OnLimited is called when a request is rate limited.
// It receives the response writer, the request, and the rate limit result.
type OnLimited func(w http.ResponseWriter, r *http.Request, result *limiter.Result)

// DefaultOnLimited is the default handler for rate limited requests.
// It returns a 429 Too Many Requests response with a JSON body.
func DefaultOnLimited(w http.ResponseWriter, _ *http.Request, result *limiter.Result) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())+1))
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error":"rate_limit_exceeded","retry_after":"%s"}`, result.RetryAfter.Round(time.Millisecond))
}

// Options configures the HTTP middleware behavior.
type Options struct {
	// KeyFunc extracts the rate limiting key from the request.
	// Defaults to IPKeyFunc().
	KeyFunc KeyFunc

	// OnLimited is called when a request is rate limited.
	// Defaults to DefaultOnLimited.
	OnLimited OnLimited
}

// HTTPMiddleware creates rate limiting HTTP middleware.
// It wraps an http.Handler and applies rate limiting based on the provided
// limiter and key extraction function.
func HTTPMiddleware(l limiter.Limiter, keyFunc KeyFunc) func(http.Handler) http.Handler {
	return HTTPMiddlewareWithOptions(l, &Options{
		KeyFunc:   keyFunc,
		OnLimited: DefaultOnLimited,
	})
}

// HTTPMiddlewareWithOptions creates rate limiting HTTP middleware with full options.
func HTTPMiddlewareWithOptions(l limiter.Limiter, opts *Options) func(http.Handler) http.Handler {
	if opts == nil {
		opts = &Options{}
	}
	if opts.KeyFunc == nil {
		opts.KeyFunc = IPKeyFunc()
	}
	if opts.OnLimited == nil {
		opts.OnLimited = DefaultOnLimited
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := opts.KeyFunc(r)

			result, err := l.Allow(r.Context(), key)
			if err != nil {
				// On error, allow the request but do not set rate limit headers
				next.ServeHTTP(w, r)
				return
			}

			// Set standard rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed {
				opts.OnLimited(w, r, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
