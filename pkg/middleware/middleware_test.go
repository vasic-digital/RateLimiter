package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.ratelimiter/pkg/limiter"
	"digital.vasic.ratelimiter/pkg/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// okHandler is a simple handler that returns 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
})

func TestHTTPMiddlewareAllowed(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  10,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	handler := HTTPMiddleware(rl, IPKeyFunc())(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())

	// Check rate limit headers
	assert.Equal(t, "10", rec.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
}

func TestHTTPMiddlewareDenied(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   2,
		Window: time.Second,
		Burst:  2,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	handler := HTTPMiddleware(rl, IPKeyFunc())(okHandler)

	// Exhaust the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "rate_limit_exceeded")
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestHTTPMiddlewareWithHeaderKey(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   2,
		Window: time.Second,
		Burst:  2,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	handler := HTTPMiddleware(rl, HeaderKeyFunc("X-API-Key"))(okHandler)

	// Exhaust limit for api-key-1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "api-key-1")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// api-key-1 should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "api-key-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// api-key-2 should still work
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "api-key-2")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHTTPMiddlewareHeaderFallback(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  10,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	keyFunc := HeaderKeyFunc("X-API-Key")

	// Without the header, should fall back to RemoteAddr
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	key := keyFunc(req)
	assert.Equal(t, req.RemoteAddr, key)

	// With the header, should use the header value
	req.Header.Set("X-API-Key", "my-key")
	key = keyFunc(req)
	assert.Equal(t, "my-key", key)
}

func TestHTTPMiddlewareCustomOnLimited(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	customCalled := false
	opts := &Options{
		KeyFunc: IPKeyFunc(),
		OnLimited: func(w http.ResponseWriter, _ *http.Request, _ *limiter.Result) {
			customCalled = true
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("custom limit response"))
		},
	}

	handler := HTTPMiddlewareWithOptions(rl, opts)(okHandler)

	// First request allowed
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request triggers custom handler
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, customCalled)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "custom limit response", rec.Body.String())
}

func TestHTTPMiddlewareNilOptions(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  10,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	// Should not panic with nil options
	handler := HTTPMiddlewareWithOptions(rl, nil)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// errorLimiter always returns an error
type errorLimiter struct{}

func (e *errorLimiter) Allow(_ context.Context, _ string) (*limiter.Result, error) {
	return nil, assert.AnError
}

func (e *errorLimiter) Reset(_ context.Context, _ string) error {
	return assert.AnError
}

func TestHTTPMiddlewareOnError(t *testing.T) {
	handler := HTTPMiddleware(&errorLimiter{}, IPKeyFunc())(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// On error, request should be allowed through (fail-open)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIPKeyFunc(t *testing.T) {
	keyFunc := IPKeyFunc()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	key := keyFunc(req)
	assert.Equal(t, "192.168.1.1:12345", key)
}

func TestDefaultOnLimited(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	result := &limiter.Result{
		Allowed:    false,
		RetryAfter: 5 * time.Second,
	}

	DefaultOnLimited(rec, req, result)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, "6", rec.Header().Get("Retry-After"))
	assert.Contains(t, rec.Body.String(), "rate_limit_exceeded")
}

func TestMiddlewareChaining(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   100,
		Window: time.Second,
		Burst:  100,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	// Test that the middleware works correctly when chained
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := HTTPMiddleware(rl, IPKeyFunc())(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.True(t, called, "inner handler should have been called")
	assert.Equal(t, http.StatusOK, rec.Code)
}
