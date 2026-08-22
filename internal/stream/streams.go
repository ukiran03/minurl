package stream

import (
	"context"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"ukiran.com/minurl/internal/data"
)

type FlushFunc[T data.BatchItem] func(
	ctx context.Context,
	batch []T,
	msgs []jetstream.Msg,
) error

type BatchOpts[T data.BatchItem] struct {
	MaxBatchSize  int
	FlushInterval time.Duration
	FlushFunc     FlushFunc[T]
}

type Streamer interface {
	Publish(ctx context.Context, subject string, payload []byte) error

	// Start begins pulling messages from the stream and executing the
	// batching/flushing process. It blocks until the context is canceled.
	Start(ctx context.Context) error
}

// NOTE: There is no Acking for Publish(), Do I need it ?
