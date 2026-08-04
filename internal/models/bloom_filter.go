package models

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	bloomFilterKey = "minurl:long:bloom"
	errorRate      = 0.01
	capacity       = 10000000
)

type BloomFilter struct {
	Rdb *redis.Client
}

func NewBloomFilter(rdb *redis.Client) *BloomFilter {
	return &BloomFilter{
		Rdb: rdb,
	}
}

// initFilter runs: BF.RESERVE minurl:long:bloom 0.01 10000000
func (bf *BloomFilter) InitFilter(ctx context.Context) error {
	_, err := bf.Rdb.BFReserve(ctx, bloomFilterKey, errorRate, capacity).
		Result()
	return err
}

// Exists checks if the long URL might alredy exist (BF.EXISTS)
func (bf *BloomFilter) Exists(
	ctx context.Context,
	longURL string,
) (bool, error) {
	return bf.Rdb.BFExists(ctx, bloomFilterKey, longURL).Result()
}

// Add registers a new long URL into the filter (BF.ADD)
func (bf *BloomFilter) Add(ctx context.Context, longURL string) error {
	_, err := bf.Rdb.BFAdd(ctx, bloomFilterKey, longURL).Result()
	return err
}
