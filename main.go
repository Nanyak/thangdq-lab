package main

import (
	"github.com/Nanyak/thangdq-lab/handler"
	"github.com/Nanyak/thangdq-lab/store"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	storeService := store.NewStorageService(redisClient)

	r.POST("/url", func(c *gin.Context) {
		handler.CreateShortUrlHandler(c, storeService)
	})

	r.GET("/:shortUrl", func(c *gin.Context) {
		handler.RedirectUrlHandler(c, storeService)
	})
	err := r.Run(":9808") // listen and serve on
	if err != nil {
		panic(err)
	}
}
