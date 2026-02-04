package main

import (
	"html/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user/Rate-Limiting-API/internal/dashboard"
	"github.com/user/Rate-Limiting-API/internal/limiter"
	"github.com/user/Rate-Limiting-API/internal/middleware"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

// Custom template functions
var funcMap = template.FuncMap{
	"div": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"mul": func(a, b float64) float64 {
		return a * b
	},
}

func main() {
	// Initialize Redis
	storage.InitRedis("localhost:6379", "", 0)

	// Create rate limiter
	// Capacity: 10 requests, LeakRate: 2 per second, TTL: 1 hour
	rateLimiter := limiter.NewLeakyBucket(10, 2, time.Hour)

	// Create dashboard handler
	dashboardHandler := dashboard.NewHandler(rateLimiter)

	r := gin.Default()

	// Set custom template functions before loading templates
	r.SetFuncMap(funcMap)

	// Load HTML templates
	r.LoadHTMLFiles(
		"templates/index.html",
		"templates/partials/status.html",
		"templates/partials/keys.html",
		"templates/partials/test_result.html",
	)

	// Dashboard routes (tanpa rate limiting)
	dashboardGroup := r.Group("/dashboard")
	{
		dashboardGroup.GET("/", dashboardHandler.Index)
		dashboardGroup.GET("/status", dashboardHandler.GetStatus)
		dashboardGroup.GET("/status/json", dashboardHandler.GetStatusJSON)
		dashboardGroup.GET("/keys", dashboardHandler.ListKeys)
		dashboardGroup.GET("/keys/json", dashboardHandler.ListKeysJSON)
		dashboardGroup.POST("/reset", dashboardHandler.ResetKey)
		dashboardGroup.POST("/test", dashboardHandler.TestRequest)
	}

	// API routes dengan rate limiting
	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.RateLimit(rateLimiter))
	{
		apiGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		// Contoh endpoint lainnya
		apiGroup.GET("/data", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"data":    "This is rate-limited data",
				"time":    time.Now().Format(time.RFC3339),
				"message": "You are within rate limit!",
			})
		})
	}

	// Public endpoint (tanpa rate limiting)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	r.Run(":8080")
}
