package limiter

import "context"

type LeakyBucket struct {
	Capacity float64
	LeakRate float64
}

// Allow mengecek apakah request diizinkan dan mengupdate status di Redis
func (lb *LeakyBucket) Allow(ctx context.Context, key string) (bool, float64, error) {
	return false, 0, nil
}
