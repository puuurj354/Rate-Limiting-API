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

var ctx = context.TODO()

func setupMockRedis() redismock.ClientMock {
	db, mock := redismock.NewClientMock()
	storage.RedisClient = db
	return mock
}

func TestNewLeakyBucket(t *testing.T) {
	lb := NewLeakyBucket(10, 2, time.Hour)

	assert.Equal(t, 10.0, lb.Capacity)
	assert.Equal(t, 2.0, lb.LeakRate)
	assert.Equal(t, time.Hour, lb.TTL)
}

func TestLeakyBucket_ImplementsInterface(t *testing.T) {
	var _ RateLimiter = (*LeakyBucket)(nil)
}

func TestLeakyBucket_Allow_FirstRequest(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(5, 1, time.Hour)

	key := "test_key"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)

	// First request - keys don't exist yet
	mock.ExpectGet(waterKey).RedisNil()
	mock.ExpectGet(timeKey).RedisNil()
	mock.ExpectSet(waterKey, "1", time.Hour).SetVal("OK")
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err := lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 4.0, remaining)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_Allow_ExceedCapacity(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(5, 1, time.Hour)

	key := "test_key"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)
	now := fmt.Sprintf("%d", time.Now().Unix())

	// Bucket sudah penuh (water level = 5)
	mock.ExpectGet(waterKey).SetVal("5")
	mock.ExpectGet(timeKey).SetVal(now)

	allowed, remaining, err := lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.False(t, allowed) // Should be denied
	assert.Equal(t, 0.0, remaining)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_Allow_LeakOverTime(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(5, 1, time.Hour)

	key := "test_key"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)

	// Bucket was full (5) but 3 seconds have passed, so leaked 3
	// Effective water level = 5 - 3 = 2
	pastTime := fmt.Sprintf("%d", time.Now().Unix()-3)

	mock.ExpectGet(waterKey).SetVal("5")
	mock.ExpectGet(timeKey).SetVal(pastTime)
	mock.ExpectSet(waterKey, "3", time.Hour).SetVal("OK") // 2 + 1 new request = 3
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err := lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 2.0, remaining) // 5 - 3 = 2

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_Allow_MultipleRequests(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(3, 0, time.Hour)

	key := "multi_test"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)
	now := fmt.Sprintf("%d", time.Now().Unix())

	// Request 1 - empty bucket
	mock.ExpectGet(waterKey).RedisNil()
	mock.ExpectGet(timeKey).RedisNil()
	mock.ExpectSet(waterKey, "1", time.Hour).SetVal("OK")
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err := lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 2.0, remaining)

	// Request 2
	mock.ExpectGet(waterKey).SetVal("1")
	mock.ExpectGet(timeKey).SetVal(now)
	mock.ExpectSet(waterKey, "2", time.Hour).SetVal("OK")
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err = lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 1.0, remaining)

	// Request 3
	mock.ExpectGet(waterKey).SetVal("2")
	mock.ExpectGet(timeKey).SetVal(now)
	mock.ExpectSet(waterKey, "3", time.Hour).SetVal("OK")
	mock.Regexp().ExpectSet(timeKey, `\d+`, time.Hour).SetVal("OK")

	allowed, remaining, err = lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 0.0, remaining)

	// Request 4 - should be denied (bucket full)
	mock.ExpectGet(waterKey).SetVal("3")
	mock.ExpectGet(timeKey).SetVal(now)

	allowed, remaining, err = lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0.0, remaining)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_Allow_RedisError(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(5, 1, time.Hour)

	key := "error_test"
	waterKey := fmt.Sprintf("bucket:%s:water", key)

	// Simulate Redis error
	mock.ExpectGet(waterKey).SetErr(fmt.Errorf("connection refused"))

	allowed, remaining, err := lb.Allow(ctx, key)
	assert.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0.0, remaining)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_Reset(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(5, 1, time.Hour)

	key := "reset_test"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)

	mock.ExpectDel(waterKey).SetVal(1)
	mock.ExpectDel(timeKey).SetVal(1)

	err := lb.Reset(ctx, key)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_GetStatus(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(10, 2, time.Hour)

	key := "status_test"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)
	now := fmt.Sprintf("%d", time.Now().Unix())

	mock.ExpectGet(waterKey).SetVal("3")
	mock.ExpectGet(timeKey).SetVal(now)

	status, err := lb.GetStatus(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, key, status.Key)
	assert.Equal(t, 3.0, status.Current)
	assert.Equal(t, 10.0, status.Capacity)
	assert.Equal(t, 7.0, status.Remaining)
	assert.Equal(t, 2.0, status.LeakRate)
	assert.False(t, status.IsLimited)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_GetStatus_Limited(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(5, 1, time.Hour)

	key := "limited_test"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)
	now := fmt.Sprintf("%d", time.Now().Unix())

	mock.ExpectGet(waterKey).SetVal("5")
	mock.ExpectGet(timeKey).SetVal(now)

	status, err := lb.GetStatus(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, 5.0, status.Current)
	assert.Equal(t, 0.0, status.Remaining)
	assert.True(t, status.IsLimited)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_GetStatus_Empty(t *testing.T) {
	mock := setupMockRedis()
	lb := NewLeakyBucket(10, 1, time.Hour)

	key := "empty_test"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)

	mock.ExpectGet(waterKey).RedisNil()
	mock.ExpectGet(timeKey).RedisNil()

	status, err := lb.GetStatus(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, status.Current)
	assert.Equal(t, 10.0, status.Remaining)
	assert.False(t, status.IsLimited)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeakyBucket_Allow_AfterLeak(t *testing.T) {
	mock := setupMockRedis()
	lb := &LeakyBucket{Capacity: 5, LeakRate: 1} // Leak 1 per second

	key := "leak_test"
	waterKey := fmt.Sprintf("bucket:%s:water", key)
	timeKey := fmt.Sprintf("bucket:%s:time", key)

	// Bucket was full (5) but 6 seconds have passed, so leaked 6
	// Effective water level = 5 - 6 = 0
	pastTime := fmt.Sprintf("%d", time.Now().Unix()-6)

	mock.ExpectGet(waterKey).SetVal("5")
	mock.ExpectGet(timeKey).SetVal(pastTime)
	mock.ExpectSet(waterKey, "1", 0).SetVal("OK") // 0 + 1 new request = 1
	mock.Regexp().ExpectSet(timeKey, `\d+`, 0).SetVal("OK")

	allowed, remaining, err := lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 4.0, remaining) // 5 - 1 = 4

	assert.NoError(t, mock.ExpectationsWereMet())
}
