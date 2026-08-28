// pkg/kafka/reader.go

package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// Reader wraps kafka-go Reader for simple consumer use cases.
type Reader struct {
	*kafka.Reader
}

// ReaderConfig holds Reader configuration.
type ReaderConfig struct {
	Brokers  []string
	Topic    string
	GroupID  string
	MinBytes int
	MaxBytes int
}

// NewReader creates a kafka Reader.
func NewReader(cfg ReaderConfig) *Reader {
	return &Reader{
		Reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  cfg.Brokers,
			Topic:    cfg.Topic,
			GroupID:  cfg.GroupID,
			MinBytes: cfg.MinBytes,
			MaxBytes: cfg.MaxBytes,
		}),
	}
}

// Message is an alias for kafka-go Message.
type Message = kafka.Message

// ReadMessage reads the next message.
func (r *Reader) ReadMessage(ctx context.Context) (Message, error) {
	return r.Reader.ReadMessage(ctx)
}
