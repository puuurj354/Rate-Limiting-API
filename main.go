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
	// Initialize Redis connection
	storage.InitRedis("localhost:6379", "", 0)

	// Create both rate limiting algorithm instances
	// Leaky Bucket: Capacity 10, LeakRate 2/sec, TTL 1 hour
	leakyBucket := limiter.NewLeakyBucket(10, 2, time.Hour)

	// Token Bucket: Capacity 10, RefillRate 2/sec, TTL 1 hour
	tokenBucket := limiter.NewTokenBucket(10, 2, time.Hour)

	// Create LimiterManager with both algorithms, default to leaky_bucket
	limiterManager := limiter.NewLimiterManager(leakyBucket, tokenBucket, "leaky_bucket")

	// Create dashboard handler with manager
	dashboardHandler := dashboard.NewHandler(limiterManager)

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
		// Algorithm switching endpoints
		dashboardGroup.GET("/algorithm", dashboardHandler.GetAlgorithm)
		dashboardGroup.POST("/algorithm", dashboardHandler.SetAlgorithm)
	}

	// API routes dengan rate limiting (uses the manager which delegates to active algorithm)
	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.RateLimit(limiterManager))
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
