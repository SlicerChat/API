package db

import (
	"github.com/go-redis/redis"
	"slicerapi/internal/util"
)

// Redis is the Redis client.
var Redis *redis.Client

// Nil is Redis' Nil value. Used to avoid reimporting.
const Nil = redis.Nil

// Connect connects to the Redis server.
func ConnectRedis() error {
	Redis = redis.NewClient(&redis.Options{
		Addr:     util.Config.DB.Redis.Address,
		Password: util.Config.DB.Redis.Password,
		DB:       util.Config.DB.Redis.ID,
	})

	_, err := Redis.Ping().Result()
	return err
}
