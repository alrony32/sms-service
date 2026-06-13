package ratelimit

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Limiter struct {
	client *goredis.Client
}

func New(client *goredis.Client) *Limiter {
	return &Limiter{client: client}
}

func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 {
		return true, nil
	}

	count, err := l.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {

		if err := l.client.Expire(ctx, key, window).Err(); err != nil {
			return false, err
		}
	}

	return count <= int64(limit), nil
}
