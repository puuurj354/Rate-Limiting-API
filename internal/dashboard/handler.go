package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user/Rate-Limiting-API/internal/limiter"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

// Handler adalah struct untuk dashboard handlers
type Handler struct {
	Limiter limiter.RateLimiter
}

// NewHandler membuat instance baru dashboard handler
func NewHandler(rl limiter.RateLimiter) *Handler {
	return &Handler{Limiter: rl}
}

// Index menampilkan halaman dashboard utama
func (h *Handler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Rate Limiting Dashboard",
	})
}

// GetStatus mendapatkan status rate limiter untuk key tertentu
func (h *Handler) GetStatus(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		key = c.ClientIP()
	}

	status, err := h.Limiter.GetStatus(c.Request.Context(), key)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "partials/status.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "partials/status.html", gin.H{
		"status": status,
	})
}

// GetStatusJSON mendapatkan status dalam format JSON
func (h *Handler) GetStatusJSON(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		key = c.ClientIP()
	}

	status, err := h.Limiter.GetStatus(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ResetKey reset rate limiter untuk key tertentu
func (h *Handler) ResetKey(c *gin.Context) {
	key := c.PostForm("key")
	if key == "" {
		// Check if request is from keys table or reset form
		hxTarget := c.GetHeader("HX-Target")
		if hxTarget == "keys-container" {
			c.HTML(http.StatusBadRequest, "partials/keys.html", gin.H{
				"error": "Key is required",
			})
		} else {
			c.HTML(http.StatusBadRequest, "partials/status.html", gin.H{
				"error": "Key is required",
			})
		}
		return
	}

	err := h.Limiter.Reset(c.Request.Context(), key)
	if err != nil {
		hxTarget := c.GetHeader("HX-Target")
		if hxTarget == "keys-container" {
			c.HTML(http.StatusInternalServerError, "partials/keys.html", gin.H{
				"error": err.Error(),
			})
		} else {
			c.HTML(http.StatusInternalServerError, "partials/status.html", gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	// Check HX-Target header to determine response type
	hxTarget := c.GetHeader("HX-Target")
	if hxTarget == "keys-container" {
		// Request from keys table - return refreshed keys list
		ctx := c.Request.Context()
		keys, _ := storage.RedisClient.Keys(ctx, "bucket:*:water").Result()
		var statuses []*limiter.Status
		for _, fullKey := range keys {
			k := fullKey[7 : len(fullKey)-6]
			status, err := h.Limiter.GetStatus(ctx, k)
			if err == nil {
				statuses = append(statuses, status)
			}
		}
		c.HTML(http.StatusOK, "partials/keys.html", gin.H{
			"keys": statuses,
		})
	} else {
		// Request from reset form - return status with message
		status, _ := h.Limiter.GetStatus(c.Request.Context(), key)
		c.HTML(http.StatusOK, "partials/status.html", gin.H{
			"status":  status,
			"message": "Rate limit for '" + key + "' reset successfully",
		})
	}
}

// ListKeys mendapatkan daftar semua keys yang sedang di-track
func (h *Handler) ListKeys(c *gin.Context) {
	ctx := c.Request.Context()

	// Scan untuk keys dengan pattern bucket:*:water
	keys, err := storage.RedisClient.Keys(ctx, "bucket:*:water").Result()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "partials/keys.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	// Extract key names dan dapatkan status masing-masing
	var statuses []*limiter.Status
	for _, fullKey := range keys {
		// Extract key dari "bucket:{key}:water"
		key := fullKey[7 : len(fullKey)-6] // Remove "bucket:" prefix and ":water" suffix
		status, err := h.Limiter.GetStatus(ctx, key)
		if err == nil {
			statuses = append(statuses, status)
		}
	}

	c.HTML(http.StatusOK, "partials/keys.html", gin.H{
		"keys": statuses,
	})
}

// ListKeysJSON mendapatkan daftar keys dalam format JSON
func (h *Handler) ListKeysJSON(c *gin.Context) {
	ctx := c.Request.Context()

	keys, err := storage.RedisClient.Keys(ctx, "bucket:*:water").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var statuses []*limiter.Status
	for _, fullKey := range keys {
		key := fullKey[7 : len(fullKey)-6]
		status, err := h.Limiter.GetStatus(ctx, key)
		if err == nil {
			statuses = append(statuses, status)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"keys": statuses,
	})
}

// TestRequest melakukan request test untuk demo rate limiting
func (h *Handler) TestRequest(c *gin.Context) {
	key := c.ClientIP()

	allowed, remaining, err := h.Limiter.Allow(c.Request.Context(), key)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "partials/test_result.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "partials/test_result.html", gin.H{
		"allowed":   allowed,
		"remaining": remaining,
		"key":       key,
	})
}
