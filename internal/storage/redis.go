package storage

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	Ctx         = context.Background()
)

func InitRedis(addr string, password string, db int) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	err := RedisClient.Ping(Ctx).Err()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to Redis successfully")
}
