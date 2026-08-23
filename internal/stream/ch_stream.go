package stream

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"ukiran.com/minurl/internal/data"
)

const (
	ChStreamName         = "CH_CLICKS_STREAM"
	ChStreamConsumerName = "CH_CLICKS_CONSUMER"
	ChStreamSubjectName  = "ch.clicks.>"
)

var (
	ChStreamBatchOpts = BatchOpts[data.ClickEvent]{
		MaxBatchSize:  100, // should increase in real-world
		FlushInterval: 3 * time.Second,
	}

	ChStreamCfg = jetstream.StreamConfig{
		Name:        ChStreamName,
		Description: "TODO",
		Subjects:    []string{ChStreamSubjectName},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      24 * time.Hour,        // Keep data for 24h if consumer is down
		Storage:     jetstream.FileStorage, // Persist to disk so it survives restarts
	}

	ChStreamConsumerCfg = jetstream.ConsumerConfig{
		Durable:   ChStreamConsumerName,
		AckPolicy: jetstream.AckExplicitPolicy,
		AckWait:   30 * time.Second, // flushInterval + DB timeout
	}
)

type ClickHouseStream struct {
	Store    *data.ClickHouseStore
	Jets     jetstream.JetStream
	Consumer jetstream.Consumer
	Logger   *slog.Logger
}

// Ensure ClickHouseStream satisfies the Streamer interface at compile-time
var _ Streamer = (*ClickHouseStream)(nil)

func NewClickHouseStream(
	ctx context.Context,
	jets jetstream.JetStream,
	store *data.ClickHouseStore,
	logger *slog.Logger,
) (*ClickHouseStream, error) {
	consumer, err := initStream(ctx, jets, StreamConfig{
		StreamName:  ChStreamName,
		StreamCfg:   ChStreamCfg,
		ConsumerCfg: ChStreamConsumerCfg,
	})
	if err != nil {
		return nil, err
	}

	return &ClickHouseStream{
		Jets:     jets,
		Consumer: consumer,
		Store:    store,
		Logger:   logger,
	}, nil
}

// Publish writes messages to the designated stream subject
func (chs *ClickHouseStream) Publish(
	ctx context.Context, subject string, payload []byte,
) error {
	_, err := chs.Jets.Publish(ctx, subject, payload)
	return err
}

func (chs *ClickHouseStream) FlushHandler() FlushFunc[data.ClickEvent] {
	return func(
		ctx context.Context,
		batch []data.ClickEvent,
		msgs []jetstream.Msg,
	) error {
		return HandleFlush(ctx, batch, msgs, chs.Logger, chs.Store.Write)
	}
}

func (chs *ClickHouseStream) Start(ctx context.Context) error {
	msgsCtx, err := chs.Consumer.Messages()
	if err != nil {
		return err
	}

	opts := BatchOpts[data.ClickEvent]{
		MaxBatchSize:  ChStreamBatchOpts.MaxBatchSize,
		FlushInterval: ChStreamBatchOpts.FlushInterval,
		FlushFunc:     chs.FlushHandler(),
	}

	return RunBatchProcess(ctx, msgsCtx, opts, chs.Logger)
}
