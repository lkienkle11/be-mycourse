package cache_clients

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"

	"mycourse-io-be/pkg/setting"
)

var Redis *redis.Client

func SetupRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     setting.RedisSetting.Addr,
		Password: setting.RedisSetting.Password,
		DB:       setting.RedisSetting.DB,
	})

	if err := Redis.Ping(context.Background()).Err(); err != nil {
		log.Printf("redis not ready: %v", err)
	}
}
