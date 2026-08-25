package data

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type RedisStore struct {
	Rdb    *redis.Client
	TTL    time.Duration
	Logger *slog.Logger
}

func NewRedisStore(
	rdb *redis.Client, ttl time.Duration, logger *slog.Logger,
) *RedisStore {
	return &RedisStore{
		Rdb:    rdb,
		TTL:    ttl,
		Logger: logger,
	}
}

// Put writes the core object as a Hash and links an MD5 index pointer to it
// atomically
func (s *RedisStore) Put(ctx context.Context, minurl *MinUrl) error {
	pipe := s.Rdb.TxPipeline()
	expiry := minurl.Life.Expiry

	redisKey := "minurl:" + minurl.Slug
	data := map[string]any{
		"flake":    minurl.Flake,
		"slug":     minurl.Slug,
		"url":      minurl.URL,
		"url_hash": minurl.URLHash,
	}

	// Store the structural hash object
	pipe.HSet(ctx, redisKey, data)
	pipe.ExpireAt(ctx, redisKey, expiry)

	// Create the write-path secondary inde (MD5 Hash -> Redis Key Pointer)
	indexKey := "index:hash:" + minurl.URLHash
	pipe.Set(ctx, indexKey, redisKey, time.Until(expiry))

	_, err := pipe.Exec(ctx)
	return err
}

// GetByHash maps the incoming MD5 hash back to the core object (POST /v1/shorten path)
func (s *RedisStore) GetByHash(ctx context.Context, urlHash string) (
	*MinUrl, error,
) {
	// Resolve the secondary index pointer
	redisKey, err := s.Rdb.Get(ctx, "index:hash:"+urlHash).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	// Fetch all fields from the primary Hash
	return s.fetchMinUrl(ctx, redisKey)
}

// GetBySlug directly accesses the object via its short slug (GET /:slug path)
func (s *RedisStore) GetBySlug(ctx context.Context, slug string) (
	*MinUrl, error,
) {
	redisKey := "minurl:" + slug
	return s.fetchMinUrl(ctx, redisKey)
}

// Helper to scan a Redis Hash directly into our struct
func (s *RedisStore) fetchMinUrl(ctx context.Context, redisKey string) (
	*MinUrl, error,
) {
	var minurl MinUrl
	err := s.Rdb.HGetAll(ctx, redisKey).Scan(&minurl)
	if err != nil {
		return nil, err
	}

	// Redis HGetAll returns an empty map/struct if the key doesn't exist
	if minurl.Slug == "" {
		return nil, ErrCacheMiss
	}
	return &minurl, nil
}

// ---

// Delete completely removes the primary hash and its corresponding secondary index.
// You can pass a partial MinUrl struct containing at least the Slug. If URLHash is
// missing, it will look it up dynamically before purging.
func (s *RedisStore) Delete(ctx context.Context, minurl *MinUrl) error {
	redisKey := "minurl:" + minurl.Slug

	// If URLHash wasn't provided, fetch it from the Hash before we destroy it
	if minurl.URLHash == "" {
		hash, err := s.Rdb.HGet(ctx, redisKey, "url_hash").Result()
		if errors.Is(err, redis.Nil) {
			return nil // Key already gone, nothing to do
		}
		if err != nil {
			return err
		}
		minurl.URLHash = hash
	}

	// Use an atomic pipeline to delete both keys
	pipe := s.Rdb.TxPipeline()

	pipe.Del(ctx, redisKey) // Delete primary data
	pipe.Del(
		ctx,
		"index:hash:"+minurl.URLHash,
	) // Delete secondary lookup index

	_, err := pipe.Exec(ctx)
	return err
}
