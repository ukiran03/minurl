package data

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Defining custom errors allows your cache-aside logic to easily
// know when to query Postgres instead.
var ErrCacheMiss = errors.New("cache miss")

type RedisStore struct {
	Rdb *redis.Client
	TTL time.Duration
}

func NewRedisStore(rdb *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{
		Rdb: rdb,
		TTL: ttl,
	}
}

// Put updates the cache.
func (s *RedisStore) Put(ctx context.Context, minurl *MinUrl) error {
	return s.Rdb.Set(ctx, minurl.Slug, minurl.URL, s.TTL).Err()
}

// Get checks the cache.
func (s *RedisStore) Get(ctx context.Context, minurl *MinUrl) (string, error) {
	val, err := s.Rdb.Get(ctx, minurl.Slug).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", err // Redis connection error
	}
	return val, nil
}

// Delete evicts the item from cache
// DOUBT: useful for Cache-Aside or Write-Through invalidation
func (s *RedisStore) Delete(ctx context.Context, minurl *MinUrl) error {
	return s.Rdb.Del(ctx, minurl.Slug).Err()
}
