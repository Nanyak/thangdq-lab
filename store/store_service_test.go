package store

import (
  "context"
  "time"
  "github.com/stretchr/testify/assert"
  "testing"

  "github.com/go-redis/redis/v8"
)

func setupTestStorageService(t *Testing.T) *StorageService(
  redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
    DB: 1,
    })

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.second)
    defer cancel()

    err := redisClient.Ping(ctx).Err()
    assert.NoError(t, err)

    t.Cleanup(func() {
      redisClient.FlushDB(context.Background())
      redisClient.Close()
    })
    return NewStorageService(redisClient)
}

func TestSaveAndRetrieveURL(t *testing T){
  service := setupTestStorageService(t)
  ctx := context.Background()

  err := service.SaveUrlMapping(ctx, "abc", "https://google.com")
  assert.NoError(t, err)

  url, err := service.RetrieveInitialUrl(ctx, "abc")
  assert.NoError(t, err)
  assert.Equal(t, "https://google.com", url)
}
