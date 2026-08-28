// services/core-svc/internal/core/handler/health.go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Health reports liveness and readiness over both gRPC (via grpc_health_v1,
// registered in main) and plain HTTP for container probes and load balancers.
type Health struct {
	pool  *pgxpool.Pool
	redis redis.Cmdable
	// ready flips to false the moment shutdown begins, so a load balancer stops
	// sending new work before the server actually stops accepting it.
	ready atomic.Bool
}

// NewHealth builds the health reporter. It starts not-ready; call SetReady once
// dependencies are up.
func NewHealth(pool *pgxpool.Pool, rdb redis.Cmdable) *Health {
	return &Health{pool: pool, redis: rdb}
}

// SetReady marks the service ready or draining.
func (h *Health) SetReady(ready bool) { h.ready.Store(ready) }

// Ready reports the current readiness flag.
func (h *Health) Ready() bool { return h.ready.Load() }

// checkTimeout bounds each dependency probe so a wedged dependency cannot hang
// the probe endpoint itself.
const checkTimeout = 2 * time.Second

// Check pings every dependency, returning the first failure.
func (h *Health) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		return err
	}
	return h.redis.Ping(ctx).Err()
}

// LiveHandler answers whether the process is running at all. It deliberately
// checks nothing: a liveness probe that fails on a dependency outage causes a
// restart loop that makes the outage worse.
func (h *Health) LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// ReadyHandler answers whether the service can serve traffic right now: it is
// past startup, not draining, and its dependencies respond.
func (h *Health) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.Ready() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "draining",
			})
			return
		}
		if err := h.Check(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not ready",
				"reason": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
