// pkg/kafka/producer.go

// Package kafka wraps segmentio/kafka-go with the platform's own conventions:
// every message keyed by aggregate id for per-aggregate ordering, and consumer
// handling with bounded retry and a dead-letter topic.
package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer writes messages synchronously, one aggregate id at a time ordered
// within its partition.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer builds a Producer against brokers. RequiredAcks is "all": a write
// is not acknowledged until every in-sync replica has it, trading latency for
// never losing an event the caller believes was published.
func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			WriteTimeout: 10 * time.Second,
		},
	}
}

// Publish writes one message to topic, keyed by aggregateID so every event
// about the same aggregate lands in the same partition and is read in order.
func (p *Producer) Publish(ctx context.Context, topic, aggregateID string, payload []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(aggregateID),
		Value: payload,
		Time:  time.Now().UTC(),
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("publishing to %s: %w", topic, err)
	}
	return nil
}

// Close flushes and closes the underlying writer.
func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("closing kafka producer: %w", err)
	}
	return nil
}
