package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Nanyak/thangdq-lab/handler"
	"github.com/Nanyak/thangdq-lab/store"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func main() {
	r := gin.Default()

	r.Use(cors.Default())
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// Ping Redis to check connection with retry
	ctx := context.Background()
	maxRetries := 10
	retryDelay := 2 * time.Second

	var err error
	for i := 0; i < maxRetries; i++ {
		_, err = redisClient.Ping(ctx).Result()
		if err == nil {
			fmt.Printf("Successfully connected to Redis at %s\n", redisAddr)
			break
		}
		fmt.Printf("Failed to connect to Redis (attempt %d/%d): %v. Retrying in %v...\n",
			i+1, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)
	}

	if err != nil {
		panic("Failed to connect to Redis after retries: " + err.Error())
	}
	storeService := store.NewStorageService(redisClient)

	r.POST("/url", func(c *gin.Context) {
		handler.CreateShortUrlHandler(c, storeService)
	})

	r.GET("/:shortUrl", func(c *gin.Context) {
		handler.RedirectUrlHandler(c, storeService)
	})
	err = r.Run(":9808") // listen and serve on
	if err != nil {
		panic(err)
	}
}
