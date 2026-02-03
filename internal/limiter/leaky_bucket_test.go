package limiter

import (
	"context"
	"fmt"
	"testing"

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

func TestLeakyBucket_Allow_ExceedCapacity(t *testing.T) {
	mock := setupMockRedis()
	lb := &LeakyBucket{Capacity: 5, LeakRate: 1} // Kapasitas 5, leak rate 1 (tidak relevan di test ini karena tanpa delay)

	key := "test_key"
	redisKey := fmt.Sprintf("bucket:%s:water", key) // Asumsi key format

	mock.ExpectGet(redisKey).RedisNil()           // Pertama kali, key belum ada (nil)
	mock.ExpectSet(redisKey, "1", 0).SetVal("OK") // Set water level ke 1

	allowed, remaining, err := lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 4.0, remaining)

	mock.ExpectGet(redisKey).SetVal("1")
	mock.ExpectSet(redisKey, "2", 0).SetVal("OK")
	allowed, remaining, err = lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 3.0, remaining)

	for i := 3; i <= 5; i++ {
		mock.ExpectGet(redisKey).SetVal(fmt.Sprintf("%d", i-1))
		mock.ExpectSet(redisKey, fmt.Sprintf("%d", i), 0).SetVal("OK")
		allowed, remaining, err = lb.Allow(ctx, key)
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, float64(6-i), remaining)
	}

	mock.ExpectGet(redisKey).SetVal("5")
	// Tidak ada SET karena deny
	allowed, remaining, err = lb.Allow(ctx, key)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0.0, remaining)

	// Verifikasi semua expectation terpenuhi
	assert.NoError(t, mock.ExpectationsWereMet())
}
