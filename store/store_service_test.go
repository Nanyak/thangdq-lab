package store

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func setupTestStorageService(t *testing.T) *StorageService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := redisClient.Ping(ctx).Err()
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = redisClient.FlushDB(context.Background()).Err()
		_ = redisClient.Close()
	})

	return NewStorageService(redisClient)
}

func TestSaveAndRetrieveURL(t *testing.T) {
	service := setupTestStorageService(t)
	ctx := context.Background()

	err := service.SaveUrlMapping(ctx, "abc", "https://google.com")
	assert.NoError(t, err)

	url, err := service.RetrieveInitialUrl(ctx, "abc")
	assert.NoError(t, err)
	assert.Equal(t, "https://google.com", url)
}
