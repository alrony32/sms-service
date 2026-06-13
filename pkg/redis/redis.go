package redis

import (
	"context"
	"sync"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sms-service/internal/config"
)

var (
	client *goredis.Client
	once   sync.Once
	Ctx    = context.Background()
)

func Client(cfg *config.Config) *goredis.Client {
	once.Do(func() {
		addr := cfg.Redis.Host + ":" + cfg.Redis.Port

		client = goredis.NewClient(&goredis.Options{
			Addr:     addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	})

	return client
}
