// internal/observability/metrics.go
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RED metrics
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "status"},
	)

	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total requests",
		},
		[]string{"service", "method", "status"},
	)

	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total errors",
		},
		[]string{"service", "method", "error_type"},
	)

	// Kafka consumer lag
	KafkaConsumerLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Kafka consumer lag by topic and partition",
		},
		[]string{"consumer_group", "topic", "partition"},
	)

	// Pipeline step duration
	PipelineStepDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_step_duration_seconds",
			Help:    "ML pipeline step duration",
			Buckets: []float64{.1, .5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"step"},
	)

	// Saga state counts
	SagaStateCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "saga_state_count",
			Help: "Count of sagas in each state",
		},
		[]string{"saga_type", "state"},
	)
)
