package limiter

import (
	"context"

	"github.com/go-redis/redismock/v9"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

var ctx = context.TODO()

func setupMockRedis() redismock.ClientMock {
	db, mock := redismock.NewClientMock()
	storage.RedisClient = db
	return mock
}

