// internal/observability/health.go
package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthChecker struct {
	pg    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthChecker(pg *pgxpool.Pool, redis *redis.Client) *HealthChecker {
	return &HealthChecker{pg: pg, redis: redis}
}

// Liveness check - always returns 200 if process is running
func (h *HealthChecker) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Readiness check - verifies dependencies
func (h *HealthChecker) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]bool{
		"postgres": h.checkPostgres(ctx),
		"redis":    h.checkRedis(ctx),
	}

	allHealthy := true
	for _, healthy := range checks {
		if !healthy {
			allHealthy = false
			break
		}
	}

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(checks)
}

func (h *HealthChecker) checkPostgres(ctx context.Context) bool {
	if h.pg == nil {
		return false
	}
	err := h.pg.Ping(ctx)
	return err == nil
}

func (h *HealthChecker) checkRedis(ctx context.Context) bool {
	if h.redis == nil {
		return false
	}
	err := h.redis.Ping(ctx).Err()
	return err == nil
}
