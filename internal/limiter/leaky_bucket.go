package limiter

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

// Pastikan LeakyBucket implement RateLimiter interface
var _ RateLimiter = (*LeakyBucket)(nil)

type LeakyBucket struct {
	Capacity float64       // Kapasitas maksimum bucket
	LeakRate float64       // Jumlah request yang "bocor" per detik
	TTL      time.Duration // TTL untuk Redis keys (0 = no expiry)
}

// NewLeakyBucket membuat instance baru LeakyBucket
func NewLeakyBucket(capacity, leakRate float64, ttl time.Duration) *LeakyBucket {
	return &LeakyBucket{
		Capacity: capacity,
		LeakRate: leakRate,
		TTL:      ttl,
	}
}

// waterKey dan timeKey helper untuk generate Redis keys
func (lb *LeakyBucket) waterKey(key string) string {
	return "bucket:" + key + ":water"
}

func (lb *LeakyBucket) timeKey(key string) string {
	return "bucket:" + key + ":time"
}

// Allow mengecek apakah request diizinkan dan mengupdate status di Redis
// Returns: (allowed, remaining capacity, error)
func (lb *LeakyBucket) Allow(ctx context.Context, key string) (bool, float64, error) {
	waterKey := lb.waterKey(key)
	timeKey := lb.timeKey(key)

	now := time.Now().Unix()

	// Ambil nilai water level dari Redis
	waterVal, err := storage.RedisClient.Get(ctx, waterKey).Result()
	var waterLevel float64
	if err == redis.Nil {
		waterLevel = 0
	} else if err != nil {
		return false, 0, err
	} else {
		waterLevel, _ = strconv.ParseFloat(waterVal, 64)
	}

	// Ambil last update time dari Redis
	timeVal, err := storage.RedisClient.Get(ctx, timeKey).Result()
	var lastTime int64
	if err == redis.Nil {
		lastTime = now
	} else if err != nil {
		return false, 0, err
	} else {
		lastTime, _ = strconv.ParseInt(timeVal, 10, 64)
	}

	// Hitung jumlah air yang sudah "bocor" berdasarkan waktu berlalu
	elapsed := float64(now - lastTime)
	leaked := elapsed * lb.LeakRate
	waterLevel = waterLevel - leaked
	if waterLevel < 0 {
		waterLevel = 0
	}

	// Cek apakah masih ada ruang di bucket
	if waterLevel >= lb.Capacity {
		// Bucket penuh, tolak request
		return false, 0, nil
	}

	// Tambahkan 1 "air" (request) ke bucket
	waterLevel = waterLevel + 1

	// Simpan level air terbaru ke Redis dengan TTL
	err = storage.RedisClient.Set(ctx, waterKey, strconv.FormatFloat(waterLevel, 'f', -1, 64), lb.TTL).Err()
	if err != nil {
		return false, 0, err
	}

	// Simpan waktu update terbaru ke Redis dengan TTL
	err = storage.RedisClient.Set(ctx, timeKey, strconv.FormatInt(now, 10), lb.TTL).Err()
	if err != nil {
		return false, 0, err
	}

	remaining := lb.Capacity - waterLevel
	return true, remaining, nil
}

// Reset menghapus semua state untuk key tertentu
func (lb *LeakyBucket) Reset(ctx context.Context, key string) error {
	waterKey := lb.waterKey(key)
	timeKey := lb.timeKey(key)

	pipe := storage.RedisClient.Pipeline()
	pipe.Del(ctx, waterKey)
	pipe.Del(ctx, timeKey)
	_, err := pipe.Exec(ctx)
	return err
}

// GetStatus mendapatkan status rate limiter untuk key tertentu
func (lb *LeakyBucket) GetStatus(ctx context.Context, key string) (*Status, error) {
	waterKey := lb.waterKey(key)
	timeKey := lb.timeKey(key)

	now := time.Now().Unix()

	// Ambil nilai water level dari Redis
	waterVal, err := storage.RedisClient.Get(ctx, waterKey).Result()
	var waterLevel float64
	if err == redis.Nil {
		waterLevel = 0
	} else if err != nil {
		return nil, err
	} else {
		waterLevel, _ = strconv.ParseFloat(waterVal, 64)
	}

	// Ambil last update time dari Redis
	timeVal, err := storage.RedisClient.Get(ctx, timeKey).Result()
	var lastTime int64
	if err == redis.Nil {
		lastTime = now
	} else if err != nil {
		return nil, err
	} else {
		lastTime, _ = strconv.ParseInt(timeVal, 10, 64)
	}

	// Hitung leakage
	elapsed := float64(now - lastTime)
	leaked := elapsed * lb.LeakRate
	waterLevel = waterLevel - leaked
	if waterLevel < 0 {
		waterLevel = 0
	}

	remaining := lb.Capacity - waterLevel
	if remaining < 0 {
		remaining = 0
	}

	return &Status{
		Key:       key,
		Current:   waterLevel,
		Capacity:  lb.Capacity,
		Remaining: remaining,
		LeakRate:  lb.LeakRate,
		IsLimited: waterLevel >= lb.Capacity,
		Algorithm: "leaky_bucket",
	}, nil
}
