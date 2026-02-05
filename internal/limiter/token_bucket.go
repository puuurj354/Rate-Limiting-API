package limiter

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

// Pastikan TokenBucket implement RateLimiter interface
var _ RateLimiter = (*TokenBucket)(nil)

// TokenBucket implements the Token Bucket rate limiting algorithm.
// Unlike Leaky Bucket which drains at a constant rate, Token Bucket:
// - Refills tokens at a constant rate
// - Each request consumes 1 token
// - If no tokens available, request is denied
// - Allows bursts up to bucket capacity
type TokenBucket struct {
	Capacity   float64
	RefillRate float64
	TTL        time.Duration
}

// NewTokenBucket creates a new TokenBucket instance
// capacity: maximum tokens bucket can hold
// refillRate: tokens added per second
// ttl: time-to-live for Redis keys
func NewTokenBucket(capacity, refillRate float64, ttl time.Duration) *TokenBucket {
	return &TokenBucket{
		Capacity:   capacity,
		RefillRate: refillRate,
		TTL:        ttl,
	}
}

// tokensKey generates Redis key for token count
func (tb *TokenBucket) tokensKey(key string) string {
	return "token:" + key + ":tokens"
}

// timeKey generates Redis key for last refill timestamp
func (tb *TokenBucket) timeKey(key string) string {
	return "token:" + key + ":time"
}

func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, float64, error) {
	tokensKey := tb.tokensKey(key) // Redis key for token storage
	timeKey := tb.timeKey(key)     // Redis key for last refill time

	now := time.Now().Unix() // Current Unix timestamp

	// Retrieve current token count from Redis
	tokensVal, err := storage.RedisClient.Get(ctx, tokensKey).Result()
	var tokens float64
	if err == redis.Nil {
		// Key doesn't exist - start with full bucket
		tokens = tb.Capacity
	} else if err != nil {
		// Redis error - return failure
		return false, 0, err
	} else {
		// Parse existing token count
		tokens, _ = strconv.ParseFloat(tokensVal, 64)
	}

	// Retrieve last refill timestamp from Redis
	timeVal, err := storage.RedisClient.Get(ctx, timeKey).Result()
	var lastTime int64
	if err == redis.Nil {
		// Key doesn't exist - use current time
		lastTime = now
	} else if err != nil {
		// Redis error - return failure
		return false, 0, err
	} else {
		// Parse existing timestamp
		lastTime, _ = strconv.ParseInt(timeVal, 10, 64)
	}

	// Calculate tokens to add based on elapsed time
	elapsed := float64(now - lastTime)     // Seconds since last refill
	tokensToAdd := elapsed * tb.RefillRate // Tokens earned in that time
	tokens = tokens + tokensToAdd          // Add refilled tokens
	if tokens > tb.Capacity {              // Cap at maximum capacity
		tokens = tb.Capacity
	}

	// Check if we have tokens available
	if tokens < 1 {
		// No tokens available - deny request
		return false, 0, nil
	}

	// Consume 1 token for this request
	tokens = tokens - 1

	// Save updated token count to Redis with TTL
	err = storage.RedisClient.Set(ctx, tokensKey, strconv.FormatFloat(tokens, 'f', -1, 64), tb.TTL).Err()
	if err != nil {
		return false, 0, err
	}

	// Save current time as last refill time to Redis with TTL
	err = storage.RedisClient.Set(ctx, timeKey, strconv.FormatInt(now, 10), tb.TTL).Err()
	if err != nil {
		return false, 0, err
	}

	// Return success with remaining token count
	return true, tokens, nil
}

// Reset clears all state for a specific key
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	tokensKey := tb.tokensKey(key) // Redis key for token storage
	timeKey := tb.timeKey(key)     // Redis key for last refill time

	// Use pipeline to delete both keys atomically
	pipe := storage.RedisClient.Pipeline()
	pipe.Del(ctx, tokensKey) // Delete token count
	pipe.Del(ctx, timeKey)   // Delete timestamp
	_, err := pipe.Exec(ctx) // Execute pipeline
	return err
}

// GetStatus retrieves current rate limiter status for a key
func (tb *TokenBucket) GetStatus(ctx context.Context, key string) (*Status, error) {
	tokensKey := tb.tokensKey(key) // Redis key for token storage
	timeKey := tb.timeKey(key)     // Redis key for last refill time

	now := time.Now().Unix() // Current Unix timestamp

	// Retrieve current token count from Redis
	tokensVal, err := storage.RedisClient.Get(ctx, tokensKey).Result()
	var tokens float64
	var keyExists bool
	if err == redis.Nil {
		// Key doesn't exist - bucket is full
		tokens = tb.Capacity
		keyExists = false
	} else if err != nil {
		// Redis error - return failure
		return nil, err
	} else {
		// Parse existing token count
		tokens, _ = strconv.ParseFloat(tokensVal, 64)
		keyExists = true
	}

	// Retrieve last refill timestamp from Redis
	timeVal, err := storage.RedisClient.Get(ctx, timeKey).Result()
	var lastTime int64
	if err == redis.Nil {
		// Key doesn't exist - use current time
		lastTime = now
	} else if err != nil {
		// Redis error - return failure
		return nil, err
	} else {
		// Parse existing timestamp
		lastTime, _ = strconv.ParseInt(timeVal, 10, 64)
	}

	// Calculate tokens added since last refill
	elapsed := float64(now - lastTime)     // Seconds since last refill
	tokensToAdd := elapsed * tb.RefillRate // Tokens earned
	tokens = tokens + tokensToAdd          // Add refilled tokens
	if tokens > tb.Capacity {              // Cap at maximum capacity
		tokens = tb.Capacity
	}

	// IMPORTANT: Save the updated state to Redis to ensure consistency
	// Only save if the key already exists (has been used before)
	if keyExists && elapsed > 0 {
		// Save updated token count
		storage.RedisClient.Set(ctx, tokensKey, strconv.FormatFloat(tokens, 'f', -1, 64), tb.TTL)
		// Save current time as last refill time
		storage.RedisClient.Set(ctx, timeKey, strconv.FormatInt(now, 10), tb.TTL)
	}

	// For Token Bucket, "current usage" = capacity - tokens (inverse of Leaky Bucket)
	// This makes the UI consistent: higher usage = less remaining
	currentUsage := tb.Capacity - tokens
	if currentUsage < 0 {
		currentUsage = 0
	}

	// Build and return status
	return &Status{
		Key:       key,
		Current:   currentUsage,   // "Used" tokens (capacity - available)
		Capacity:  tb.Capacity,    // Maximum bucket size
		Remaining: tokens,         // Available tokens
		LeakRate:  tb.RefillRate,  // Refill rate (reusing LeakRate field)
		IsLimited: tokens < 1,     // Limited if no tokens available
		Algorithm: "token_bucket", // Algorithm identifier
	}, nil
}
