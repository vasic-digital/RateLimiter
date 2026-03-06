package gin

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"digital.vasic.ratelimiter/pkg/limiter"
)

// KeyFunc extracts a rate limiting key from a Gin context.
type KeyFunc func(c *gin.Context) string

// IPKeyFunc returns a KeyFunc that uses the client's IP address as the key.
func IPKeyFunc() KeyFunc {
	return func(c *gin.Context) string { return c.ClientIP() }
}

// HeaderKeyFunc returns a KeyFunc that uses the value of the specified
// header as the key, falling back to the client IP if the header is absent.
func HeaderKeyFunc(header string) KeyFunc {
	return func(c *gin.Context) string {
		if val := c.GetHeader(header); val != "" {
			return val
		}
		return c.ClientIP()
	}
}

// RateLimit returns Gin middleware that applies rate limiting using the
// provided Limiter and KeyFunc. When the limiter returns an error, the
// request is rejected with 500. When the rate limit is exceeded, the
// request is rejected with 429.
func RateLimit(rl limiter.Limiter, keyFunc KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)
		result, err := rl.Allow(context.Background(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rate limiter error"})
			return
		}
		if !result.Allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
