package lock

import (
	"errors"
	"github.com/google/uuid"
	"github.com/zeriontech/sidecache/pkg/cache"
	"time"
)

type RedisLock struct {
	redis *cache.RedisRepository
	values map[string]string
}

const UnlockScript = `
	if redis.call("get", KEYS[1]) == ARGV[1] then
	    return redis.call("del", KEYS[1])
	else
	    return 0
	end
`

func NewRedisLock(redis *cache.RedisRepository) *RedisLock {
	return &RedisLock{redis: redis, values: map[string]string{}}
}

func (lock *RedisLock) Acquire(key string, ttl time.Duration) error {
	val := uuid.NewString()
	lock.values[key] = val
	return lock.redis.SetNX(key, val, ttl)
}

func (lock *RedisLock) Release(key string) error {
	val, ok := lock.values[key]
	if !ok {
		return errors.New("unknown key")
	}
	delete(lock.values, key)
	return lock.redis.Eval(UnlockScript, []string{key}, val)
}
