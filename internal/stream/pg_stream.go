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

const (
	PgStreamName   = "PG_WRITES_STREAM"
	PgConsumerName = "PG_WRITES_CONSUMER"
	PgsSubjectName = "pg.minurls.creates"
)

var (
	PgsBatchOpts = BatchOpts[data.MinUrl]{
		MaxBatchSize:  100,
		FlushInterval: 2 * time.Second,
	}

	PgStreamCfg = jetstream.StreamConfig{
		Name:        PgStreamName,
		Description: "Write-through cache stream for Postgres DB operations",
		Subjects:    []string{PgsSubjectName},
		Retention:   jetstream.WorkQueuePolicy,
		MaxMsgs:     100000,
		Discard:     jetstream.DiscardNew,
	}

	PgsConsumerCfg = jetstream.ConsumerConfig{
		Durable:   PgConsumerName,
		AckPolicy: jetstream.AckExplicitPolicy,
		AckWait:   30 * time.Second, // flushInterval + DB timeout
	}
)

type PostgresStream struct {
	Store    *data.PostgresStore
	Jets     jetstream.JetStream
	Consumer jetstream.Consumer
	Logger   *slog.Logger
}

// Ensure PostgresStream satisfies the Streamer interface at compile-time
var _ Streamer = (*PostgresStream)(nil)

func NewPostgresStream(
	ctx context.Context,
	jets jetstream.JetStream,
	store *data.PostgresStore,
	logger *slog.Logger,
) (*PostgresStream, error) {
	// var stream jetstream.Stream
	var err error

	// Retry loop to handle NATS JetStream initialization lag during startup
	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		_, err = jets.CreateOrUpdateStream(ctx, PgStreamCfg)
		if err == nil {
			break
		}

		if i == maxRetries {
			return nil, fmt.Errorf(
				"failed to create/update stream after %d attempts: %w",
				maxRetries,
				err,
			)
		}

		// Context-aware sleep to respect timeouts/cancellations
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(i) * time.Second):
		}
	}

	// Consider adding retries here as well if consumer creation is flaky on
	// startup
	consumer, err := jets.CreateOrUpdateConsumer(
		ctx, PgStreamName, PgsConsumerCfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update consumer: %w", err)
	}

	return &PostgresStream{
		Jets:     jets,
		Consumer: consumer,
		Store:    store,
		Logger:   logger,
	}, nil
}

// Publish writes messages to the designated stream subject
func (pgs *PostgresStream) Publish(
	ctx context.Context, subject string, payload []byte,
) error {
	_, err := pgs.Jets.Publish(ctx, subject, payload)
	return err
}

func (pgs *PostgresStream) RunBatchProcess(
	ctx context.Context,
	msgsCtx jetstream.MessagesContext,
	opts BatchOpts[data.MinUrl],
) error {
	// Guard against nil configuration panics
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

	// Ensure NATS subscription stops when we exit this loop to prevent
	// goroutine leaks
	defer msgsCtx.Stop()

	ticker := time.NewTicker(opts.FlushInterval)
	defer ticker.Stop()

	batch := make([]data.MinUrl, 0, opts.MaxBatchSize)
	natsMsgs := make([]jetstream.Msg, 0, opts.MaxBatchSize)

	// Using an unbuffered channel to tightly couple reader & consumer,
	// preventing unacknowledged messages from getting stuck in an internal
	// channel buffer during shutdown.
	msgChan := make(chan jetstream.Msg)

	// Start a background reader to fetch messages from JetStream. This
	// prevents Next() from blocking ticker/context select loop.
	go func() {
		defer close(msgChan)
		for {
			msg, err := msgsCtx.Next()
			if err != nil {
				// Triggers when msgsCtx.Stop() is called or connection drops
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

		// Use parent context for shutdown safety, combined with a timeout
		writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := opts.FlushFunc(writeCtx, batch, natsMsgs); err != nil {
			pgs.Logger.Error("Flush failed", "error", err, "batch", len(batch))
		}

		// Safely clear slices for the next batch without reallocation
		batch = batch[:0]
		natsMsgs = natsMsgs[:0]
	}

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// msgChan closed means the JetStream iterator stopped
				flush()
				return nil
			}

			var u data.MinUrl
			if err := json.Unmarshal(msg.Data(), &u); err != nil {
				metadata, _ := msg.Metadata()
				seq := uint64(0)
				if metadata != nil {
					seq = metadata.Sequence.Stream
				}
				pgs.Logger.Error(
					"Malformed JSON skipped", "Seq", seq, "error", err,
				)

				_ = msg.Term() // NACK + terminate bad payloads instantly
				continue
			}

			batch = append(batch, u)
			natsMsgs = append(natsMsgs, msg)

			if len(batch) >= opts.MaxBatchSize {
				flush()
				select {
				// Drain ticker channel if a tick happened exactly
				// during flush to avoid instant double-flushing
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

// FlushHandler bridges PostgresStore.Copy and NATS JetStream ACKs
func (pgs *PostgresStream) FlushHandler() FlushFunc[data.MinUrl] {
	return func(
		ctx context.Context,
		batch []data.MinUrl,
		msgs []jetstream.Msg,
	) error {
		// Write the batch to Postgres using Copy
		if err := pgs.Store.Copy(ctx, batch); err != nil {
			return err // so RunBatchProcess doesn't ACK and NATS redelivers
		}

		// If DB write succeeded, ACK all messages in the batch
		for _, msg := range msgs {
			if err := msg.Ack(); err != nil {
				pgs.Logger.Error("failed to ack message", "error", err)
			}
		}

		return nil
	}
}

func (pgs *PostgresStream) Start(ctx context.Context) error {
	msgsCtx, err := pgs.Consumer.Messages()
	if err != nil {
		return err
	}

	opts := BatchOpts[data.MinUrl]{
		MaxBatchSize:  PgsBatchOpts.MaxBatchSize,
		FlushInterval: PgsBatchOpts.FlushInterval,
		FlushFunc:     pgs.FlushHandler(), // Connects the store to the stream
	}

	return pgs.RunBatchProcess(ctx, msgsCtx, opts)
}
