package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Handler func(ctx context.Context, rec *kgo.Record) error

type Consumer struct {
	cl       *kgo.Client
	handlers map[string]Handler
	log      *slog.Logger
}

func NewConsumer(brokers []string, group string, handlers map[string]Handler, log *slog.Logger) (*Consumer, error) {
	topics := make([]string, 0, len(handlers))
	for t := range handlers {
		topics = append(topics, t)
	}
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()), // group baru baca dari awal
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return nil, err
	}
	return &Consumer{cl, handlers, log}, nil
}

func (c *Consumer) Run(ctx context.Context) {
	defer c.cl.Close()
	for {
		fetches := c.cl.PollRecords(ctx, 100)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			return
		}
		fetches.EachError(func(t string, p int32, err error) {
			c.log.Error("kafka fetch", "topic", t, "partition", p, "error", err)
		})

		var done []*kgo.Record
		it := fetches.RecordIter()
		for !it.Done() {
			rec := it.Next()
			if !c.handleWithRetry(ctx, rec) {
				return // ctx dibatalkan
			}
			done = append(done, rec)
		}
		if len(done) > 0 {
			if err := c.cl.CommitRecords(ctx, done...); err != nil {
				c.log.Error("kafka commit", "error", err)
			}
		}
		c.cl.AllowRebalance()
	}
}

func (c *Consumer) handleWithRetry(ctx context.Context, rec *kgo.Record) bool {
	backoff := time.Second
	for {
		err := c.handlers[rec.Topic](ctx, rec)
		if err == nil {
			return true
		}
		c.log.Error("handle record", "topic", rec.Topic, "key", string(rec.Key), "error", err)
		select {
		case <-ctx.Done():
			return false
		case <-time.After(backoff):
			backoff = min(backoff*2, 30*time.Second)
		}
	}
}
