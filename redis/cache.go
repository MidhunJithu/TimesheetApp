package redis

import (
	"example/timesheet/models"

	"github.com/go-redis/redis/v8"
)

func NewCache() *models.Cache {
	return &models.Cache{
		Redis: redis.NewClient(
			&redis.Options{
				Addr:     "localhost:8080",
				Password: "",
				DB:       0,
			},
		),
	}
}