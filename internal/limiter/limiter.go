package limiter

import "context"

// RateLimiter adalah interface untuk semua algoritma rate limiting
type RateLimiter interface {
	// Allow mengecek apakah request diizinkan
	Allow(ctx context.Context, key string) (bool, float64, error)

	// Reset menghapus semua state untuk key tertentu
	Reset(ctx context.Context, key string) error

	// GetStatus mendapatkan status rate limiter untuk key tertentu
	GetStatus(ctx context.Context, key string) (*Status, error)
}

// Status menyimpan informasi status rate limiter
type Status struct {
	Key       string  `json:"key"`
	Current   float64 `json:"current"`    
	Capacity  float64 `json:"capacity"`   
	Remaining float64 `json:"remaining"`  
	LeakRate  float64 `json:"leak_rate"`  
	IsLimited bool    `json:"is_limited"` 
	Algorithm string  `json:"algorithm"`  
}
