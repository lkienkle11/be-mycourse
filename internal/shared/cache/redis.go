package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"mycourse-io-be/internal/shared/setting"
)

var Redis *redis.Client

// RedisAvailable reports whether SetupRedis constructed a client (Redis is non-nil).
// Commands may still fail if the server is down or unreachable.
func RedisAvailable() bool {
	return Redis != nil
}

func SetupRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     setting.RedisSetting.Addr,
		Password: setting.RedisSetting.Password,
		DB:       setting.RedisSetting.DB,
	})

	if err := Redis.Ping(context.Background()).Err(); err != nil {
		zap.L().Warn("redis not ready", zap.Error(err))
	}
}
