package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type Cache struct {
	Redis *redis.Client
}

func NewCache() *Cache {
	Redis := redis.NewClient(
		&redis.Options{
			Addr:     "redis-16388.c322.us-east-1-2.ec2.cloud.redislabs.com:16388",
			Password: "jtzsjGjxXiWgoyIKiPu7rmRPzT7PX0wP",
			Username: "default",
		})
	_, err := Redis.Ping(context.Background()).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis Cloud: %v", err))
	}
	return &Cache{Redis: Redis}
}

func (cache *Cache) CheckDataInCache(ctx context.Context, key string, targetObj interface{}) (found bool, strVal string) {
	res, err := cache.Redis.Get(ctx, key).Result()
	if err != nil {
		return false, res
	}
	if targetObj == nil {
		return true, res
	}
	err = json.Unmarshal([]byte(res), &targetObj)
	if err != nil {
		logrus.Errorf("error while marshalling redis output to %v : %s", targetObj, err)
	}
	return true, res
}

func (cache *Cache) SetDataInCache(ctx context.Context, key string, data interface{}, exp time.Duration) (ok bool, err error) {
	_, err = cache.Redis.Set(ctx, key, data, exp).Result()
	return err == nil, err
}
