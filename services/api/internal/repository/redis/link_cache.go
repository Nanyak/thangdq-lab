package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type LinkCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewLinkCache(client *redis.Client, ttl time.Duration) *LinkCache {
	return &LinkCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *LinkCache) Set(ctx context.Context, shortCode string, originalURL string) error {
	key := fmt.Sprintf("link:%s", shortCode)

	err := c.client.Set(ctx, key, originalURL, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

func (c *LinkCache) Get(ctx context.Context, shortCode string) (string, error) {
	key := fmt.Sprintf("link:%s", shortCode)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("cache miss")
		}
		return "", fmt.Errorf("redis get failed: %w", err)
	}

	return val, nil
}
