package limiter

import (
	"context"
	"sync"
)

// LimiterManager manages multiple rate limiting algorithms and allows switching between them.
// It provides a thread-safe way to change the active algorithm at runtime.
type LimiterManager struct {
	leakyBucket *LeakyBucket // Leaky Bucket algorithm instance
	tokenBucket *TokenBucket // Token Bucket algorithm instance
	current     string       // Current active algorithm: "leaky_bucket" or "token_bucket"
	mu          sync.RWMutex // Mutex for thread-safe access
}

// NewLimiterManager creates a new LimiterManager with both algorithms initialized.
// defaultAlgorithm: "leaky_bucket" or "token_bucket"
func NewLimiterManager(leaky *LeakyBucket, token *TokenBucket, defaultAlgorithm string) *LimiterManager {
	return &LimiterManager{
		leakyBucket: leaky,
		tokenBucket: token,
		current:     defaultAlgorithm,
	}
}

// GetCurrentAlgorithm returns the name of the currently active algorithm.
func (m *LimiterManager) GetCurrentAlgorithm() string {
	m.mu.RLock()         // Acquire read lock
	defer m.mu.RUnlock() // Release on function exit
	return m.current
}

// SetAlgorithm switches to a different rate limiting algorithm.
// algorithm: "leaky_bucket" or "token_bucket"
// Returns true if switch was successful, false if algorithm name is invalid.
func (m *LimiterManager) SetAlgorithm(algorithm string) bool {
	if algorithm != "leaky_bucket" && algorithm != "token_bucket" {
		return false // Invalid algorithm name
	}
	m.mu.Lock()         // Acquire write lock
	defer m.mu.Unlock() // Release on function exit
	m.current = algorithm
	return true
}

// GetActiveLimiter returns the currently active RateLimiter instance.
func (m *LimiterManager) GetActiveLimiter() RateLimiter {
	m.mu.RLock()         // Acquire read lock
	defer m.mu.RUnlock() // Release on function exit
	if m.current == "token_bucket" {
		return m.tokenBucket
	}
	return m.leakyBucket
}

// Allow delegates to the active algorithm's Allow method.
func (m *LimiterManager) Allow(ctx context.Context, key string) (bool, float64, error) {
	return m.GetActiveLimiter().Allow(ctx, key)
}

// Reset delegates to the active algorithm's Reset method.
func (m *LimiterManager) Reset(ctx context.Context, key string) error {
	return m.GetActiveLimiter().Reset(ctx, key)
}

// GetStatus delegates to the active algorithm's GetStatus method.
func (m *LimiterManager) GetStatus(ctx context.Context, key string) (*Status, error) {
	return m.GetActiveLimiter().GetStatus(ctx, key)
}

// GetAlgorithmInfo returns information about the current algorithm configuration.
func (m *LimiterManager) GetAlgorithmInfo() map[string]interface{} {
	m.mu.RLock()         // Acquire read lock
	defer m.mu.RUnlock() // Release on function exit

	info := map[string]interface{}{
		"current": m.current,
	}

	if m.current == "leaky_bucket" {
		info["capacity"] = m.leakyBucket.Capacity
		info["rate"] = m.leakyBucket.LeakRate
		info["rate_name"] = "leak_rate"
		info["description"] = "Requests add water; water leaks at constant rate. Full bucket = blocked."
	} else {
		info["capacity"] = m.tokenBucket.Capacity
		info["rate"] = m.tokenBucket.RefillRate
		info["rate_name"] = "refill_rate"
		info["description"] = "Tokens refill at constant rate; requests consume tokens. No tokens = blocked."
	}

	return info
}

// Ensure LimiterManager implements RateLimiter interface
var _ RateLimiter = (*LimiterManager)(nil)
