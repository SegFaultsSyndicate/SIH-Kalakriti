// internal/observability/kafka_propagation.go
package observability

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InjectTraceContext injects OpenTelemetry trace context into Kafka message headers
func InjectTraceContext(ctx context.Context, msg *kafka.Message) {
	propagator := otel.GetTextMapPropagator()
	carrier := NewKafkaMessageCarrier(msg)
	propagator.Inject(ctx, carrier)
}

// ExtractTraceContext extracts OpenTelemetry trace context from Kafka message headers
func ExtractTraceContext(ctx context.Context, msg kafka.Message) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := NewKafkaMessageCarrier(&msg)
	return propagator.Extract(ctx, carrier)
}

// KafkaMessageCarrier adapts kafka.Message to propagation.TextMapCarrier
type KafkaMessageCarrier struct {
	msg *kafka.Message
}

func NewKafkaMessageCarrier(msg *kafka.Message) *KafkaMessageCarrier {
	return &KafkaMessageCarrier{msg: msg}
}

func (c *KafkaMessageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *KafkaMessageCarrier) Set(key, value string) {
	// Remove existing header if present
	for i, h := range c.msg.Headers {
		if h.Key == key {
			c.msg.Headers = append(c.msg.Headers[:i], c.msg.Headers[i+1:]...)
			break
		}
	}
	// Add new header
	c.msg.Headers = append(c.msg.Headers, kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
}

func (c *KafkaMessageCarrier) Keys() []string {
	keys := make([]string, len(c.msg.Headers))
	for i, h := range c.msg.Headers {
		keys[i] = h.Key
	}
	return keys
}
