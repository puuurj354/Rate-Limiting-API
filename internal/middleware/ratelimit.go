package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user/Rate-Limiting-API/internal/limiter"
)

// KeyFunc adalah fungsi untuk mendapatkan key identifier dari request
// Biasanya berdasarkan IP address, API key, atau user ID
type KeyFunc func(c *gin.Context) string

// DefaultKeyFunc menggunakan client IP sebagai key
func DefaultKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// RateLimitConfig adalah konfigurasi untuk middleware rate limiting
type RateLimitConfig struct {
	Limiter    limiter.RateLimiter
	KeyFunc    KeyFunc
	ErrHandler gin.HandlerFunc
}

// DefaultErrHandler adalah default error handler ketika rate limit tercapai
func DefaultErrHandler(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":   "Too Many Requests",
		"message": "Rate limit exceeded. Please try again later.",
	})
	c.Abort()
}

// RateLimit membuat middleware rate limiting dengan konfigurasi default
func RateLimit(rl limiter.RateLimiter) gin.HandlerFunc {
	return RateLimitWithConfig(RateLimitConfig{
		Limiter:    rl,
		KeyFunc:    DefaultKeyFunc,
		ErrHandler: DefaultErrHandler,
	})
}

// RateLimitWithConfig membuat middleware rate limiting dengan konfigurasi custom
func RateLimitWithConfig(config RateLimitConfig) gin.HandlerFunc {
	if config.Limiter == nil {
		panic("RateLimiter is required")
	}
	if config.KeyFunc == nil {
		config.KeyFunc = DefaultKeyFunc
	}
	if config.ErrHandler == nil {
		config.ErrHandler = DefaultErrHandler
	}

	return func(c *gin.Context) {
		key := config.KeyFunc(c)

		allowed, remaining, err := config.Limiter.Allow(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": "Failed to check rate limit",
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Remaining", strconv.FormatFloat(remaining, 'f', 0, 64))

		if !allowed {
			config.ErrHandler(c)
			return
		}

		c.Next()
	}
}

// RateLimitByAPIKey membuat middleware yang menggunakan API key sebagai identifier
func RateLimitByAPIKey(rl limiter.RateLimiter, headerName string) gin.HandlerFunc {
	return RateLimitWithConfig(RateLimitConfig{
		Limiter: rl,
		KeyFunc: func(c *gin.Context) string {
			apiKey := c.GetHeader(headerName)
			if apiKey == "" {
				return c.ClientIP() // Fallback ke IP jika tidak ada API key
			}
			return "apikey:" + apiKey
		},
		ErrHandler: DefaultErrHandler,
	})
}

// RateLimitByUserID membuat middleware yang menggunakan user ID dari context
func RateLimitByUserID(rl limiter.RateLimiter, contextKey string) gin.HandlerFunc {
	return RateLimitWithConfig(RateLimitConfig{
		Limiter: rl,
		KeyFunc: func(c *gin.Context) string {
			userID, exists := c.Get(contextKey)
			if !exists {
				return c.ClientIP() // Fallback ke IP jika tidak ada user ID
			}
			return "user:" + userID.(string)
		},
		ErrHandler: DefaultErrHandler,
	})
}
