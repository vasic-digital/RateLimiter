package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"digital.vasic.ratelimiter/pkg/limiter"
	"digital.vasic.ratelimiter/pkg/memory"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupRouter creates a Gin engine with the rate limit middleware and a
// single GET /ping endpoint that returns 200 "pong".
func setupRouter(rl limiter.Limiter, keyFunc KeyFunc) *gin.Engine {
	r := gin.New()
	r.Use(RateLimit(rl, keyFunc))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	return r
}

func TestRateLimitMiddleware_AllowsWithinLimit(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  10,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	router := setupRouter(rl, IPKeyFunc())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

func TestRateLimitMiddleware_BlocksOverLimit(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	router := setupRouter(rl, IPKeyFunc())

	// First request should be allowed.
	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request from same IP should be blocked.
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	assert.Contains(t, rec2.Body.String(), "rate limit exceeded")
}

func TestRateLimitMiddleware_DifferentKeysIndependent(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	router := setupRouter(rl, IPKeyFunc())

	// Request from IP A should be allowed.
	reqA := httptest.NewRequest(http.MethodGet, "/ping", nil)
	reqA.RemoteAddr = "10.0.0.1:1234"
	recA := httptest.NewRecorder()
	router.ServeHTTP(recA, reqA)
	assert.Equal(t, http.StatusOK, recA.Code)

	// Request from IP B should also be allowed (independent limit).
	reqB := httptest.NewRequest(http.MethodGet, "/ping", nil)
	reqB.RemoteAddr = "10.0.0.2:5678"
	recB := httptest.NewRecorder()
	router.ServeHTTP(recB, reqB)
	assert.Equal(t, http.StatusOK, recB.Code)
}

func TestHeaderKeyFunc(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   1,
		Window: time.Second,
		Burst:  1,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	router := setupRouter(rl, HeaderKeyFunc("X-API-Key"))

	// First request with api-key-1 should be allowed.
	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req1.Header.Set("X-API-Key", "api-key-1")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request with same api-key-1 should be blocked.
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.Header.Set("X-API-Key", "api-key-1")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)

	// Request with different api-key-2 should still be allowed.
	req3 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req3.Header.Set("X-API-Key", "api-key-2")
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)
}

func TestHeaderKeyFunc_FallsBackToClientIP(t *testing.T) {
	cfg := &limiter.Config{
		Rate:   10,
		Window: time.Second,
		Burst:  10,
	}
	rl := memory.New(cfg)
	defer rl.Stop()

	router := setupRouter(rl, HeaderKeyFunc("X-API-Key"))

	// Request without the header should still work (falls back to client IP).
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
