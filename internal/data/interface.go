package data

import "context"

// Store follows repository pattern
type Store interface {
	Put(ctx context.Context, minurl *MinUrl) error
	Get(ctx context.Context, minurl *MinUrl) (string, error)
	Delete(ctx context.Context, minurl *MinUrl) error
}
