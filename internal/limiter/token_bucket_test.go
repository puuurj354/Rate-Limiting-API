package limiter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

var tokenCtx = context.TODO()

// setupTokenMockRedis initializes mock Redis for testing
func setupTokenMockRedis() redismock.ClientMock {
	db, mock := redismock.NewClientMock()
	storage.RedisClient = db
	return mock
}

// TestNewTokenBucket verifies TokenBucket constructor
func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 2, time.Hour)

	assert.Equal(t, 10.0, tb.Capacity)  // Max tokens
	assert.Equal(t, 2.0, tb.RefillRate) // Tokens per second
	assert.Equal(t, time.Hour, tb.TTL)  // Redis TTL
}

// TestTokenBucket_ImplementsInterface ensures TokenBucket implements RateLimiter
func TestTokenBucket_ImplementsInterface(t *testing.T) {
	var _ RateLimiter = (*TokenBucket)(nil)
}

// TestTokenBucket_Allow_FirstRequest tests first request with full bucket
func TestTokenBucket_Allow_FirstRequest(t *testing.T) {
	mock := setupTokenMockRedis()
	tb := NewTokenBucket(5, 1, time.Hour)

	key := "test_token_key"
	tokensKey := fmt.Sprintf("token:%s:tokens", key)
	timeKey := fmt.Sprintf("token:%s:time", key)

	// First request - keys don't exist, bucket starts full
	mock.ExpectGet(tokensKey).RedisNil()
	mock.ExpectGet(timeKey).RedisNil()
	mock.ExpectSet(tokensKey, "4", time.Hour).SetVal("OK") // 5 - 1 = 4 tokens remaining
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err := tb.Allow(tokenCtx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 4.0, remaining) // 5 - 1 = 4 tokens

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTokenBucket_Allow_NoTokens tests denial when no tokens available
func TestTokenBucket_Allow_NoTokens(t *testing.T) {
	mock := setupTokenMockRedis()
	tb := NewTokenBucket(5, 1, time.Hour)

	key := "test_token_no_tokens"
	tokensKey := fmt.Sprintf("token:%s:tokens", key)
	timeKey := fmt.Sprintf("token:%s:time", key)
	now := fmt.Sprintf("%d", time.Now().Unix())

	// Bucket is empty (0 tokens)
	mock.ExpectGet(tokensKey).SetVal("0")
	mock.ExpectGet(timeKey).SetVal(now)

	allowed, remaining, err := tb.Allow(tokenCtx, key)
	assert.NoError(t, err)
	assert.False(t, allowed)        // Should be denied
	assert.Equal(t, 0.0, remaining) // No tokens

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTokenBucket_Allow_RefillOverTime tests token refill after time passes
func TestTokenBucket_Allow_RefillOverTime(t *testing.T) {
	mock := setupTokenMockRedis()
	tb := NewTokenBucket(5, 1, time.Hour)

	key := "test_token_refill"
	tokensKey := fmt.Sprintf("token:%s:tokens", key)
	timeKey := fmt.Sprintf("token:%s:time", key)

	// Bucket was empty (0 tokens) but 3 seconds have passed, refilled 3 tokens
	pastTime := fmt.Sprintf("%d", time.Now().Unix()-3)

	mock.ExpectGet(tokensKey).SetVal("0")
	mock.ExpectGet(timeKey).SetVal(pastTime)
	mock.ExpectSet(tokensKey, "2", time.Hour).SetVal("OK") // 0 + 3 - 1 = 2 tokens
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err := tb.Allow(tokenCtx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 2.0, remaining) // 0 + 3 (refilled) - 1 (consumed) = 2

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTokenBucket_Allow_CapAtMax tests that tokens don't exceed capacity
func TestTokenBucket_Allow_CapAtMax(t *testing.T) {
	mock := setupTokenMockRedis()
	tb := NewTokenBucket(5, 1, time.Hour)

	key := "test_token_cap"
	tokensKey := fmt.Sprintf("token:%s:tokens", key)
	timeKey := fmt.Sprintf("token:%s:time", key)

	// Bucket had 4 tokens, 10 seconds passed (would add 10, but caps at 5)
	pastTime := fmt.Sprintf("%d", time.Now().Unix()-10)

	mock.ExpectGet(tokensKey).SetVal("4")
	mock.ExpectGet(timeKey).SetVal(pastTime)
	mock.ExpectSet(tokensKey, "4", time.Hour).SetVal("OK") // capped at 5, then -1 = 4
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err := tb.Allow(tokenCtx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 4.0, remaining) // min(4+10, 5) - 1 = 4

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTokenBucket_Reset tests resetting token bucket state
func TestTokenBucket_Reset(t *testing.T) {
	mock := setupTokenMockRedis()
	tb := NewTokenBucket(5, 1, time.Hour)

	key := "reset_token_test"
	tokensKey := fmt.Sprintf("token:%s:tokens", key)
	timeKey := fmt.Sprintf("token:%s:time", key)

	mock.ExpectDel(tokensKey).SetVal(1)
	mock.ExpectDel(timeKey).SetVal(1)

	err := tb.Reset(tokenCtx, key)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTokenBucket_GetStatus tests retrieving current status
func TestTokenBucket_GetStatus(t *testing.T) {
	mock := setupTokenMockRedis()
	tb := NewTokenBucket(10, 2, time.Hour)

	key := "status_token_test"
	tokensKey := fmt.Sprintf("token:%s:tokens", key)
	timeKey := fmt.Sprintf("token:%s:time", key)
	now := fmt.Sprintf("%d", time.Now().Unix())

	mock.ExpectGet(tokensKey).SetVal("7")
	mock.ExpectGet(timeKey).SetVal(now)

	status, err := tb.GetStatus(tokenCtx, key)
	assert.NoError(t, err)
	assert.Equal(t, key, status.Key)
	assert.Equal(t, 3.0, status.Current) // 10 - 7 = 3 usage
	assert.Equal(t, 10.0, status.Capacity)
	assert.Equal(t, 7.0, status.Remaining) // 7 tokens available
	assert.Equal(t, 2.0, status.LeakRate)  // Refill rate
	assert.False(t, status.IsLimited)
	assert.Equal(t, "token_bucket", status.Algorithm)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestTokenBucket_GetStatus_Limited tests status when bucket is empty
func TestTokenBucket_GetStatus_Limited(t *testing.T) {
	mock := setupTokenMockRedis()
	tb := NewTokenBucket(5, 1, time.Hour)

	key := "limited_token_test"
	tokensKey := fmt.Sprintf("token:%s:tokens", key)
	timeKey := fmt.Sprintf("token:%s:time", key)
	now := fmt.Sprintf("%d", time.Now().Unix())

	mock.ExpectGet(tokensKey).SetVal("0")
	mock.ExpectGet(timeKey).SetVal(now)

	status, err := tb.GetStatus(tokenCtx, key)
	assert.NoError(t, err)
	assert.Equal(t, 5.0, status.Current)   // Full usage
	assert.Equal(t, 0.0, status.Remaining) // No tokens
	assert.True(t, status.IsLimited)

	assert.NoError(t, mock.ExpectationsWereMet())
}
