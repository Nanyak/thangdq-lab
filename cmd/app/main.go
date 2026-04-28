package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httphandler "github.com/Nanyak/thangdq-lab/internal/controller/http"
	"github.com/Nanyak/thangdq-lab/internal/infrastructure/auth/cognito"
	"github.com/Nanyak/thangdq-lab/internal/repository/mongodb"
	"github.com/Nanyak/thangdq-lab/internal/repository/redis"
	s3repo "github.com/Nanyak/thangdq-lab/internal/repository/s3"
	"github.com/Nanyak/thangdq-lab/internal/usecase"
	"github.com/Nanyak/thangdq-lab/internal/usecase/shortcode"
	"github.com/Nanyak/thangdq-lab/pkg/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	goredis "github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize MongoDB with retries
	mongoClient, err := connectMongoDB(cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	// Initialize Redis
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}()

	// Verify Redis connection with retries
	ctx := context.Background()
	for i := 0; i < cfg.MongoDB.MaxRetries; i++ {
		if err := redisClient.Ping(ctx).Err(); err == nil {
			log.Printf("Successfully connected to Redis at %s", cfg.Redis.Addr)
			break
		} else {
			log.Printf("Redis connection attempt %d/%d failed: %v", i+1, cfg.MongoDB.MaxRetries, err)
			if i == cfg.MongoDB.MaxRetries-1 {
				log.Fatalf("Failed to connect to Redis after %d retries: %v", cfg.MongoDB.MaxRetries, err)
			}
			time.Sleep(cfg.MongoDB.RetryInterval)
		}
	}

	// Initialize repositories
	db := mongoClient.Database(cfg.MongoDB.Database)
	linkRepo, err := mongodb.NewLinkRepository(db, cfg.MongoDB.Collection)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB repository: %v", err)
	}

	linkCache := redis.NewLinkCache(redisClient, cfg.CacheTTL)

	// S3 Storage initialization
	s3Storage, err := s3repo.NewS3Storage(&cfg.S3)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}
	storageUsecase := usecase.NewStorage(s3Storage)

	// Cognito auth initialization
	cognitoService, err := cognito.NewCognitoService(&cfg.Cognito)
	if err != nil {
		log.Fatalf("Failed to initialize Cognito service: %v", err)
	}
	authUsecase := usecase.NewAuth(cognitoService)

	// Initialize use case
	generator := shortcode.NewGenerator()
	urlShortener := usecase.NewURLShortener(linkRepo, linkCache, generator)

	// Initialize HTTP handlers
	handler := httphandler.NewHandler(urlShortener, cfg.Server.BaseURL)
	storageHandler := httphandler.NewStorageHandler(storageUsecase)
	authHandler := httphandler.NewAuthHandler(authUsecase)

	// Setup router
	router := gin.Default()
	router.Use(cors.New(cors.Config{
	AllowOrigins:     []string{"https://stuffsy.site"},
	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowHeaders:     []string{"Content-Type", "Authorization"},
	ExposeHeaders:   []string{"Content-Length"},
	AllowCredentials: true,
}))
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})
	router.GET("/r/:shortUrl", handler.RedirectURL)

	// API routes
	api := router.Group("/v1/api")
	{
		api.POST("/url", handler.CreateShortURL)
	}

	// Storage routes (protected)
	files := api.Group("/files")
	files.Use(httphandler.AuthMiddleware(authUsecase))
	{
		files.POST("", storageHandler.UploadFile)
		files.GET("", storageHandler.ListFiles)
		files.DELETE("", storageHandler.DeleteFile)    // ?key=path/to/file
		files.GET("/url", storageHandler.GetPresignedURL) // ?key=path/to/file
	}

	// Auth routes
	auth := api.Group("/auth")
	{
		auth.POST("/signup", authHandler.SignUp)
		auth.POST("/confirm-signup", authHandler.ConfirmSignUp)
		auth.POST("/resend-confirmation-code", authHandler.ResendConfirmationCode)
		auth.POST("/signin", authHandler.SignIn)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/confirm-forgot-password", authHandler.ConfirmForgotPassword)
		auth.POST("/refresh", authHandler.RefreshToken)

		// Protected auth routes
		authProtected := auth.Group("")
		authProtected.Use(httphandler.AuthMiddleware(authUsecase))
		{
			authProtected.GET("/me", authHandler.GetUser)
			authProtected.POST("/signout", authHandler.SignOut)
		}
	}

	// Start server with graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func connectMongoDB(cfg config.MongoDBConfig) (*mongo.Client, error) {
	ctx := context.Background()
	clientOptions := options.Client().ApplyURI(cfg.URI)

	var client *mongo.Client
	var err error

	for i := 0; i < cfg.MaxRetries; i++ {
		client, err = mongo.Connect(ctx, clientOptions)
		if err == nil {
			if err = client.Ping(ctx, nil); err == nil {
				log.Printf("Successfully connected to MongoDB at %s", cfg.URI)
				return client, nil
			}
		}

		log.Printf("MongoDB connection attempt %d/%d failed: %v", i+1, cfg.MaxRetries, err)
		time.Sleep(cfg.RetryInterval)
	}

	return nil, fmt.Errorf("failed to connect to MongoDB after %d retries: %w", cfg.MaxRetries, err)
}
