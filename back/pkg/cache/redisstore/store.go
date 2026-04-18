package redisstore

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yohnnn/public-survey-platform/back/pkg/cache"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type Store struct {
	client *redis.Client
}

func New(cfg Config) *Store {
	return &Store{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
	}
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Ping(ctx).Err()
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	if s == nil || s.client == nil {
		return nil, cache.ErrNotFound
	}

	value, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, cache.ErrNotFound
		}
		return nil, err
	}

	return value, nil
}

func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return nil
	}

	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *Store) Delete(ctx context.Context, keys ...string) error {
	if s == nil || s.client == nil || len(keys) == 0 {
		return nil
	}

	return s.client.Del(ctx, keys...).Err()
}

func (s *Store) Increment(ctx context.Context, key string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}

	return s.client.Incr(ctx, key).Result()
}

func (s *Store) Close() error {
	if s == nil || s.client == nil {
		return nil
	}

	return s.client.Close()
}
