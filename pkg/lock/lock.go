package lock

import (
	"time"
)

type Lock interface {
	Acquire(key string, ttl time.Duration) error
	Release(key string) error
}
