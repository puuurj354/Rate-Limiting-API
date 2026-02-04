package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/user/Rate-Limiting-API/internal/limiter"
)

// MockRateLimiter untuk testing
type MockRateLimiter struct {
	AllowFunc     func(ctx context.Context, key string) (bool, float64, error)
	ResetFunc     func(ctx context.Context, key string) error
	GetStatusFunc func(ctx context.Context, key string) (*limiter.Status, error)
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string) (bool, float64, error) {
	if m.AllowFunc != nil {
		return m.AllowFunc(ctx, key)
	}
	return true, 10, nil
}

func (m *MockRateLimiter) Reset(ctx context.Context, key string) error {
	if m.ResetFunc != nil {
		return m.ResetFunc(ctx, key)
	}
	return nil
}

func (m *MockRateLimiter) GetStatus(ctx context.Context, key string) (*limiter.Status, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(ctx, key)
	}
	return &limiter.Status{}, nil
}

func setupRouter(rl limiter.RateLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(rl))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	return r
}

func TestRateLimit_Allowed(t *testing.T) {
	mock := &MockRateLimiter{
		AllowFunc: func(ctx context.Context, key string) (bool, float64, error) {
			return true, 9, nil
		},
	}

	router := setupRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "9", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimit_Denied(t *testing.T) {
	mock := &MockRateLimiter{
		AllowFunc: func(ctx context.Context, key string) (bool, float64, error) {
			return false, 0, nil
		},
	}

	router := setupRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimit_Error(t *testing.T) {
	mock := &MockRateLimiter{
		AllowFunc: func(ctx context.Context, key string) (bool, float64, error) {
			return false, 0, assert.AnError
		},
	}

	router := setupRouter(mock)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDefaultKeyFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"

	key := DefaultKeyFunc(c)
	assert.Equal(t, "192.168.1.1", key)
}

func TestRateLimitByAPIKey(t *testing.T) {
	mock := &MockRateLimiter{
		AllowFunc: func(ctx context.Context, key string) (bool, float64, error) {
			// Verify the key is prefixed with "apikey:"
			assert.Equal(t, "apikey:test-api-key", key)
			return true, 5, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitByAPIKey(mock, "X-API-Key"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitByAPIKey_Fallback(t *testing.T) {
	mock := &MockRateLimiter{
		AllowFunc: func(ctx context.Context, key string) (bool, float64, error) {
			// Should fallback to IP when no API key
			assert.NotContains(t, key, "apikey:")
			return true, 5, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitByAPIKey(mock, "X-API-Key"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// No API key header
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitWithConfig_PanicOnNilLimiter(t *testing.T) {
	assert.Panics(t, func() {
		RateLimitWithConfig(RateLimitConfig{
			Limiter: nil,
		})
	})
}

func TestRateLimitWithConfig_CustomErrorHandler(t *testing.T) {
	mock := &MockRateLimiter{
		AllowFunc: func(ctx context.Context, key string) (bool, float64, error) {
			return false, 0, nil
		},
	}

	customErrHandler := func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"custom": "error"})
		c.Abort()
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitWithConfig(RateLimitConfig{
		Limiter:    mock,
		ErrHandler: customErrHandler,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
