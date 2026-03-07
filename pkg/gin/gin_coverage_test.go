package gin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"digital.vasic.ratelimiter/pkg/limiter"
	"digital.vasic.ratelimiter/pkg/memory"
)

// errorLimiter always returns an error from Allow.
type errorLimiter struct{}

func (e *errorLimiter) Allow(_ context.Context, _ string) (*limiter.Result, error) {
	return nil, assert.AnError
}

func (e *errorLimiter) Reset(_ context.Context, _ string) error {
	return assert.AnError
}

// TestRateLimit_LimiterError verifies that a limiter error results in 500.
func TestRateLimit_LimiterError(t *testing.T) {
	r := gin.New()
	r.Use(RateLimit(&errorLimiter{}, IPKeyFunc()))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "rate limiter error")
}

// TestRateLimit_HeaderKeyFuncFallbackToIP verifies that HeaderKeyFunc falls
// back to client IP when the header is missing.
func TestRateLimit_HeaderKeyFuncFallbackToIP(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	r := gin.New()
	r.Use(RateLimit(rl, HeaderKeyFunc("X-Custom")))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Request without the header: uses client IP
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:9999"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request from same IP without header: should be blocked
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "1.2.3.4:9999"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
}

// TestRateLimit_IPKeyFunc_DifferentIPs verifies that IPKeyFunc correctly
// distinguishes requests from different IPs.
func TestRateLimit_IPKeyFunc_DifferentIPs(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	r := gin.New()
	r.Use(RateLimit(rl, IPKeyFunc()))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// IP A exhausted
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "10.0.0.1:1111"
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	req1b := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1b.RemoteAddr = "10.0.0.1:1111"
	rec1b := httptest.NewRecorder()
	r.ServeHTTP(rec1b, req1b)
	assert.Equal(t, http.StatusTooManyRequests, rec1b.Code)

	// IP B should still work
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.2:2222"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}
