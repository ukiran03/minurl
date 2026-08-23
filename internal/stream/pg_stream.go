package stream

import (
	"context"
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

	// Adding retries here as well if consumer creation is flaky on startup
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

	return RunBatchProcess(ctx, msgsCtx, opts, pgs.Logger)
}
