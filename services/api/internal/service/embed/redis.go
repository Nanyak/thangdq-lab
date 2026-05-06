package embed

import (
	"context"
	"encoding/json"

	goredis "github.com/go-redis/redis/v8"
)

type RedisPublisher struct {
	client   *goredis.Client
	queueKey string
}

func NewRedisPublisher(client *goredis.Client, queueKey string) *RedisPublisher {
	return &RedisPublisher{client: client, queueKey: queueKey}
}

func (p *RedisPublisher) Publish(ctx context.Context, job Job) error {
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return p.client.RPush(ctx, p.queueKey, string(body)).Err()
}
