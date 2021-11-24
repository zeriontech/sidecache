package cache

import (
	"errors"
	"go.uber.org/zap"
	"os"
	"time"

	"github.com/go-redis/redis"
)

type RedisRepository struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisRepository(logger *zap.Logger) (*RedisRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	if err := client.Ping().Err(); err != nil {
		return nil, err
	}

	return &RedisRepository{client: client, logger: logger}, nil
}

func (repository *RedisRepository) SetKey(key string, value []byte, ttl time.Duration) {
	status := repository.client.Set(key, string(value), ttl)
	_, err := status.Result()
	if err != nil {
		repository.logger.Error(err.Error())
	}
}

func (repository *RedisRepository) SetNX(key string, value string, ttl time.Duration) error {
	result := repository.client.SetNX(key, value, ttl)
	if result.Err() != nil {
		return result.Err()
	}
	if !result.Val() {
		return errors.New("value not set")
	}
	return nil
}

func (repository *RedisRepository) Get(key string) []byte {
	status := repository.client.Get(key)
	stringResult, err := status.Result()

	if err != nil {
		if err != redis.Nil {
			repository.logger.Error(err.Error())
		}
		return nil
	}

	return []byte(stringResult)
}

func (repository *RedisRepository) Eval(script string, keys []string, args ...interface{}) error {
	result := repository.client.Eval(script, keys, args...)
	return result.Err()
}
