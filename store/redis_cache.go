package store

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(client *redis.Client, ttlHours int) *RedisCache {
	return &RedisCache{
		client: client,
		ttl:    time.Duration(ttlHours) * time.Hour,
	}
}

// Get retrieves a value from the cache. Returns empty string on cache miss.
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Cache miss
	}
	if err != nil {
		return "", err
	}
	return result, nil
}

// Set stores a value in the cache with the configured TTL.
func (c *RedisCache) Set(ctx context.Context, key, value string) error {
	return c.client.Set(ctx, key, value, c.ttl).Err()
}
