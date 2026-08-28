// pkg/breaker/breaker.go

// Package breaker implements a simple circuit breaker.
package breaker

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker open")

// Breaker is a simple circuit breaker.
type Breaker struct {
	mu           sync.Mutex
	failures     int
	threshold    int
	timeout      time.Duration
	openedAt     time.Time
	halfOpenSuccesses int
}

// New creates a circuit breaker.
func New(threshold int, timeout time.Duration) *Breaker {
	return &Breaker{
		threshold: threshold,
		timeout:   timeout,
	}
}

// Call executes fn if the circuit is closed.
func (b *Breaker) Call(fn func() error) error {
	b.mu.Lock()
	if b.failures >= b.threshold && time.Since(b.openedAt) < b.timeout {
		b.mu.Unlock()
		return ErrCircuitOpen
	}

	// Half-open: try one request.
	if b.failures >= b.threshold && time.Since(b.openedAt) >= b.timeout {
		b.mu.Unlock()
		err := fn()
		b.mu.Lock()
		if err == nil {
			b.halfOpenSuccesses++
			if b.halfOpenSuccesses >= 2 {
				b.failures = 0
				b.halfOpenSuccesses = 0
			}
		} else {
			b.failures++
			b.openedAt = time.Now()
			b.halfOpenSuccesses = 0
		}
		b.mu.Unlock()
		return err
	}

	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	if err != nil {
		b.failures++
		if b.failures == b.threshold {
			b.openedAt = time.Now()
		}
	} else {
		b.failures = 0
	}
	b.mu.Unlock()

	return err
}
