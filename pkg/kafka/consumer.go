// pkg/kafka/consumer.go
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/ZoroNewbie00/kalakriti/pkg/topics"
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
	c := &ConsumerGroup{cfg: cfg, log: log}
	c.reader, c.dlqWriter = c.newReaderAndDLQ()
	return c
}

// newReaderAndDLQ builds a fresh reader and dead-letter producer from cfg.
// Split out of NewConsumerGroup so Run's reconnect path can rebuild both
// after a fatal fetch/commit error without duplicating the construction
// logic.
func (c *ConsumerGroup) newReaderAndDLQ() (*kafka.Reader, *Producer) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.cfg.Brokers,
		Topic:    c.cfg.Topic,
		GroupID:  c.cfg.GroupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return reader, NewProducer(c.cfg.Brokers)
}

// initialReconnectBackoff and reconnectBackoffCap bound Run's reconnect
// pacing after a fetch/commit/dead-letter error. Deliberately separate from
// ConsumerConfig.RetryBackoff, which paces per-message handler retries on
// an already-connected reader -- a different concern with a different
// tuning knob, even though both happen to default to the same value.
const (
	initialReconnectBackoff = time.Second
	reconnectBackoffCap     = 30 * time.Second
)

// Run consumes until ctx is cancelled, transparently reconnecting with
// exponential backoff on any fetch/commit/dead-letter error instead of
// returning it to the caller. It previously returned that error, and every
// caller (see each service's main.go) only logged it and let the consumer's
// goroutine exit for good -- a single transient Kafka disconnect
// permanently killed that consumer for the remaining life of the process,
// while /healthz stayed green and pkg/outbox's relay (the equivalent
// long-lived job elsewhere in this codebase) kept retrying correctly the
// whole time. See WIRING_AUDIT_PLAN.md F-6.
//
// Run now only returns (nil) on a genuine ctx cancellation/shutdown.
func (c *ConsumerGroup) Run(ctx context.Context, handle HandlerFunc) error {
	backoff := initialReconnectBackoff

	for {
		err := c.runOnce(ctx, handle)
		if err == nil {
			return nil
		}

		c.log.Error("consumer disconnected, reconnecting",
			"topic", c.cfg.Topic, "error", err, "backoff", backoff)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > reconnectBackoffCap {
			backoff = reconnectBackoffCap
		}

		// runOnce's defer already closed the previous reader/dlqWriter --
		// whatever failed may have left them unusable, so this consumer
		// gets fresh ones rather than retrying on the same connection.
		c.reader, c.dlqWriter = c.newReaderAndDLQ()
	}
}

// runOnce is the fetch/handle/commit loop for one reader connection. It
// returns nil only on a genuine ctx cancellation (Run then stops entirely);
// any other error means the connection needs to be rebuilt, which Run does.
func (c *ConsumerGroup) runOnce(ctx context.Context, handle HandlerFunc) error {
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
