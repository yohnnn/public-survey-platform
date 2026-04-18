package cache

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("cache key not found")

type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Increment(ctx context.Context, key string) (int64, error)
	Close() error
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
