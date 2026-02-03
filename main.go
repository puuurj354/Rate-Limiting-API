package main

import (
	"github.com/gin-gonic/gin"
	"github.com/user/Rate-Limiting-API/internal/storage"
)

func main() {
	storage.InitRedis("localhost:6379", "", 0)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run()
}
