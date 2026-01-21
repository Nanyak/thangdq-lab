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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	r := gin.Default()

	r.Use(cors.Default())
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	ctx := context.Background()
	maxRetries := 10
	retryDelay := 2 * time.Second

	// Initialize MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoDB := os.Getenv("MONGO_DB")
	if mongoDB == "" {
		mongoDB = "url_shortener"
	}

	mongoCollection := os.Getenv("MONGO_COLLECTION")
	if mongoCollection == "" {
		mongoCollection = "links"
	}

	var mongoClient *mongo.Client
	var err error

	for i := 0; i < maxRetries; i++ {
		clientOpts := options.Client().ApplyURI(mongoURI)
		mongoClient, err = mongo.Connect(ctx, clientOpts)
		if err == nil {
			err = mongoClient.Ping(ctx, nil)
			if err == nil {
				fmt.Printf("Successfully connected to MongoDB at %s\n", mongoURI)
				break
			}
		}
		fmt.Printf("Failed to connect to MongoDB (attempt %d/%d): %v. Retrying in %v...\n",
			i+1, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)
	}

	if err != nil {
		panic("Failed to connect to MongoDB after retries: " + err.Error())
	}

	collection := mongoClient.Database(mongoDB).Collection(mongoCollection)
	mongoRepo := store.NewMongoRepository(collection)

	// Ensure indexes on startup
	if err := mongoRepo.EnsureIndexes(ctx); err != nil {
		fmt.Printf("Warning: failed to ensure MongoDB indexes: %v\n", err)
	}

	// Initialize Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

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

	redisCache := store.NewRedisCache(redisClient, 6) // 6 hour TTL

	storeService := store.NewStorageService(mongoRepo, redisCache)

	r.POST("/url", func(c *gin.Context) {
		handler.CreateShortUrlHandler(c, storeService)
	})

	r.GET("/:shortUrl", func(c *gin.Context) {
		handler.RedirectUrlHandler(c, storeService)
	})
	err = r.Run(":9808")
	if err != nil {
		panic(err)
	}
}
