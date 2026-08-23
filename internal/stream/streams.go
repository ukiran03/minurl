package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

	// NOTE: There is no Acking for Publish(), Do I need it ?
}

// RunBatchProcess is a single shared generic function for ANY Stream
func RunBatchProcess[T data.BatchItem](
	ctx context.Context,
	msgsCtx jetstream.MessagesContext,
	opts BatchOpts[T],
	logger *slog.Logger,
) error {
	if msgsCtx == nil || opts.FlushFunc == nil {
		return fmt.Errorf(
			"error RunBatchProcess requires both MsgsCtx and FlushFunc to be set",
		)
	}
	if opts.MaxBatchSize <= 0 {
		opts.MaxBatchSize = 100
	}
	if opts.FlushInterval <= 0 {
		opts.FlushInterval = 2 * time.Second
	}

	// ensure NATS subscription stops when we exit this loop to prevent
	// goroutine leaks
	defer msgsCtx.Stop()

	ticker := time.NewTicker(opts.FlushInterval)
	defer ticker.Stop()

	batch := make([]T, 0, opts.MaxBatchSize)
	natsMsgs := make([]jetstream.Msg, 0, opts.MaxBatchSize)

	// using an unbuffered channel to tightly couple reader & consumer,
	// preventing unacknowledged messages from getting stuck in an internal
	// channel buffer during shutdown.
	msgChan := make(chan jetstream.Msg)

	// start background reader to fetch messages from Jetstream, this prevents
	// Next() from blocking ticker/context select loop.
	go func() {
		defer close(msgChan)
		for {
			msg, err := msgsCtx.Next()
			if err != nil {
				// triggers when msgsCtx.Stop() is called or connection drops
				return
			}
			select {
			case msgChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// use parent context for shutdown safety, combined with a timeout
		writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := opts.FlushFunc(writeCtx, batch, natsMsgs); err != nil {
			logger.Error("Flush failed", "error", err, "batch", len(batch))
		}
		// safely clear slices for the next batch without reallocation
		batch = batch[:0]
		natsMsgs = natsMsgs[:0]
	}

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// msgChan is closed, indicating the JetStream iterator has
				// stopped.
				flush()
				return nil
			}

			var item T
			if err := json.Unmarshal(msg.Data(), &item); err != nil {
				metadata, _ := msg.Metadata()
				seq := uint64(0)
				if metadata != nil {
					// stream sequence number for a message.
					seq = metadata.Sequence.Stream
				}
				logger.Error(
					"Malformed JSON skipped",
					"Seq", seq, "error", err,
				)
				_ = msg.Term() // NACK + terminate bad payloads instantly
				continue
			}

			batch = append(batch, item)
			natsMsgs = append(natsMsgs, msg)

			if len(batch) >= opts.MaxBatchSize {
				flush()
				select {
				// drain the ticker channel if a tick occurred during the flush
				// to prevent an immediate double-flush.
				case <-ticker.C:
				default:
				}
				ticker.Reset(opts.FlushInterval)
			}

		case <-ticker.C:
			flush()

		case <-ctx.Done():
			flush()
			return nil
		}
	}
}
