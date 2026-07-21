package stream

import (
	"context"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"ukiran.com/minurl/internal/data"
)

type (
	FlushFunc func(
		ctx context.Context,
		batch []data.MinUrl,
		msgs []jetstream.Msg,
	) error

	BatchOpts struct {
		MaxBatchSize  int
		FlushInterval time.Duration
		FlushFunc     FlushFunc
	}
)

type Streamer interface {
	Publish(
		ctx context.Context,
		subject string,
		payload []byte,
	) error

	// Start begins pulling messages from the stream and executing the
	// batching/flushing process. It blocks until the context is canceled.
	Start(ctx context.Context) error
}

// NOTE: There is no Acking for Publish(), Do I need it ?

/* --- TODO: Refactoring Ideas
type StreamPubAck struct {
	ID        string
	Timestamp int64
	Metadata  map[string]string // Any engine-specific extra info
}
type Streams struct {
	PGStream PostgresStream
}
func NewStreams(nc *nats.Conn, ctx context.Context) Streams {
	return Streams{}
}
// --- NATS
type NatsConfig struct{}
type NatsStream struct {
	js   jetstream.JetStream
	opts NatsConfig
}
*/
