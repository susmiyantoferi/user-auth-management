package datastore

import (
	"user-auth-exam/infrastructure/config"

	"github.com/redis/go-redis/v9"
)

func NewRedis(config *config.RedisConfig) *redis.Client {
	host := config.Host
	port := config.Port
	db := config.Db

	Redis := redis.NewClient(&redis.Options{
		Addr: host + ":" + port,
		DB:   db,
	})

	return Redis
}
