package models

import "github.com/go-redis/redis/v8"

type Cache struct {
	Redis *redis.Client
}
