package stream

import (
	"context"
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
	consumer, err := initStream(ctx, jets, StreamConfig{
		StreamName:  PgStreamName,
		StreamCfg:   PgStreamCfg,
		ConsumerCfg: PgsConsumerCfg,
	})
	if err != nil {
		return nil, err
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
		return HandleFlush(ctx, batch, msgs, pgs.Logger, pgs.Store.Copy)
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
