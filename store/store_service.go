package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrShortURLNotFound = errors.New("short URL not found in storage")

type StorageService struct {
	redisClient *redis.Client
}

func NewStorageService(redisClient *redis.Client) *StorageService {
	return &StorageService{redisClient: redisClient}
}

// Note that in a real world usage, the cache duration shouldn't have
// an expiration time, an LRU policy config should be set where the
// values that are retrieved less often are purged automatically from
// the cache and stored back in RDBMS whenever the cache is full

const cacheDuration = 6 * time.Hour

// func InitStorageService() *StorageService {
// 	redisClient := redis.NewClient(&redis.Options{
// 		Addr:     "localhost:6379",
// 		Password: "",
// 		DB:       0,
// 	})

// 	pong, err := redisClient.Ping(ctx).Result()
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Printf("\nRedis started successfully: pong message = {%s}", pong)

// 	storeService.redisClient = redisClient
// 	return storeService
// }

func (s *StorageService) SaveUrlMapping(
	ctx context.Context,
	shortUrl string,
	originalUrl string,
) error {
	return s.redisClient.Set(ctx, shortUrl, originalUrl, cacheDuration).Err()
}

func (s *StorageService) RetrieveInitialUrl(
	ctx context.Context,
	shortUrl string,
) (string, error) {
	result, err := s.redisClient.Get(ctx, shortUrl).Result()
	if err == redis.Nil {
		return "", ErrShortURLNotFound
	}
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return result, err
}
