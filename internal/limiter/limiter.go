package limiter

import "context"

// RateLimiter adalah interface untuk semua algoritma rate limiting
type RateLimiter interface {
	// Allow mengecek apakah request diizinkan
	// Returns: (allowed, remaining capacity, error)
	Allow(ctx context.Context, key string) (bool, float64, error)

	// Reset menghapus semua state untuk key tertentu
	Reset(ctx context.Context, key string) error

	// GetStatus mendapatkan status rate limiter untuk key tertentu
	// Returns: (current usage, capacity, error)
	GetStatus(ctx context.Context, key string) (*Status, error)
}

// Status menyimpan informasi status rate limiter
type Status struct {
	Key       string  `json:"key"`
	Current   float64 `json:"current"`    // Current usage/water level
	Capacity  float64 `json:"capacity"`   // Maximum capacity
	Remaining float64 `json:"remaining"`  // Remaining capacity
	LeakRate  float64 `json:"leak_rate"`  // Leak rate per second (or refill rate for token bucket)
	IsLimited bool    `json:"is_limited"` // Apakah sedang di-limit
	Algorithm string  `json:"algorithm"`  // Algorithm name: "leaky_bucket" or "token_bucket"
}
