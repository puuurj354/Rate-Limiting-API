package limiter

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

type LeakyBucket struct {
	Capacity float64 // Kapasitas maksimum bucket
	LeakRate float64 // Jumlah request yang "bocor" per detik
}

// Allow mengecek apakah request diizinkan dan mengupdate status di Redis
// Returns: (allowed, remaining capacity, error)
func (lb *LeakyBucket) Allow(ctx context.Context, key string) (bool, float64, error) {
	waterKey := "bucket:" + key + ":water"
	timeKey := "bucket:" + key + ":time"

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

	// Simpan level air terbaru ke Redis
	err = storage.RedisClient.Set(ctx, waterKey, strconv.FormatFloat(waterLevel, 'f', -1, 64), 0).Err()
	if err != nil {
		return false, 0, err
	}

	// Simpan waktu update terbaru ke Redis
	err = storage.RedisClient.Set(ctx, timeKey, strconv.FormatInt(now, 10), 0).Err()
	if err != nil {
		return false, 0, err
	}

	remaining := lb.Capacity - waterLevel
	return true, remaining, nil
}
