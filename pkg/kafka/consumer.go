// pkg/kafka/consumer.go
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/segfaultsyndicate/kalakriti/pkg/topics"
)

// HandlerFunc processes one message. An error means "retry"; after MaxRetries
// failures the runner routes the message to the dead-letter topic and moves on
// rather than blocking the partition forever on one poison message.
type HandlerFunc func(ctx context.Context, msg kafka.Message) error

// ConsumerConfig controls a ConsumerGroup runner.
type ConsumerConfig struct {
	Brokers []string
	Topic   string
	GroupID string
	// MaxRetries and RetryBackoff default to 3 and 1s (doubling each attempt)
	// when left zero.
	MaxRetries   int
	RetryBackoff time.Duration
	// OnDeadLetter is called with the final error when a message has exhausted
	// its retries, just before it is dead-lettered. It is how a consumer records
	// the failure against its own aggregate; it must not block for long, and an
	// error from it is logged rather than retried.
	OnDeadLetter func(ctx context.Context, msg kafka.Message, cause error)
}

// ConsumerGroup runs one handler over one topic with at-least-once delivery: the
// offset for a message commits only after its handler succeeds or the message
// has been dead-lettered, so a crash mid-handling redelivers it rather than
// losing it.
type ConsumerGroup struct {
	reader    *kafka.Reader
	dlqWriter *Producer
	cfg       ConsumerConfig
	log       *slog.Logger
}

// NewConsumerGroup builds a runner. kafka-go's Reader with GroupID set already
// speaks the consumer group protocol (join, sync, heartbeat, rebalance), so no
// separate group-membership code is needed here.
func NewConsumerGroup(cfg ConsumerConfig, log *slog.Logger) *ConsumerGroup {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = time.Second
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &ConsumerGroup{
		reader:    reader,
		dlqWriter: NewProducer(cfg.Brokers),
		cfg:       cfg,
		log:       log,
	}
}

// Run consumes until ctx is cancelled. A cancelled context is treated as a
// normal shutdown: the reader and dead-letter writer are closed and Run returns
// nil, not an error.
func (c *ConsumerGroup) Run(ctx context.Context, handle HandlerFunc) error {
	defer func() {
		_ = c.reader.Close()
		_ = c.dlqWriter.Close()
	}()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("fetching message from %s: %w", c.cfg.Topic, err)
		}

		if handleErr := c.handleWithRetry(ctx, msg, handle); handleErr != nil {
			c.log.Error("dead-lettering message after exhausting retries",
				"topic", c.cfg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", handleErr)
			if c.cfg.OnDeadLetter != nil {
				c.cfg.OnDeadLetter(ctx, msg, handleErr)
			}
			if dlqErr := c.dlqWriter.Publish(ctx, topics.DLQ(c.cfg.Topic), string(msg.Key), msg.Value); dlqErr != nil {
				return fmt.Errorf("dead-lettering message at offset %d: %w", msg.Offset, dlqErr)
			}
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("committing offset %d on %s: %w", msg.Offset, c.cfg.Topic, err)
		}
	}
}

// handleWithRetry calls handle, retrying with exponential backoff (starting at
// cfg.RetryBackoff, doubling each time) up to cfg.MaxRetries times.
func (c *ConsumerGroup) handleWithRetry(ctx context.Context, msg kafka.Message, handle HandlerFunc) error {
	backoff := c.cfg.RetryBackoff
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
		if err := handle(ctx, msg); err != nil {
			lastErr = err
			c.log.Warn("handler failed, retrying",
				"topic", c.cfg.Topic, "offset", msg.Offset, "attempt", attempt+1, "error", err)
			continue
		}
		return nil
	}
	return fmt.Errorf("handler failed after %d attempts: %w", c.cfg.MaxRetries+1, lastErr)
}
