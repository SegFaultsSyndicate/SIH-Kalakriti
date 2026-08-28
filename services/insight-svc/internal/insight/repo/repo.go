// services/insight-svc/internal/insight/repo/repo.go
package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/<org>/kalakriti/pkg/domain"
	pkgdomain "github.com/<org>/kalakriti/pkg/domain"
	"github.com/<org>/kalakriti/services/insight-svc/internal/insight/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo implements StatementRepo.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a Repo.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// GetOrdersByArtisan returns completed orders for an artisan in the period.
func (r *Repo) GetOrdersByArtisan(ctx context.Context, artisanID uuid.UUID, from, to time.Time) ([]domain.OrderSummary, error) {
	query := `
		SELECT
			oa.created_at,
			oa.order_id,
			'Buyer' as buyer_name,
			ps.amount_paise
		FROM order_allocations oa
		JOIN payment_splits ps ON ps.artisan_id = oa.artisan_id AND ps.order_id = oa.order_id
		WHERE oa.artisan_id = $1
		  AND oa.status = 'completed'
		  AND oa.created_at >= $2
		  AND oa.created_at < $3
		ORDER BY oa.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, artisanID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var summaries []domain.OrderSummary
	for rows.Next() {
		var s domain.OrderSummary
		var buyerName string
		if err := rows.Scan(&s.Date, &s.OrderID, &buyerName, &s.AmountPaise); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		s.BuyerName = buyerName
		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}

// SaveStatement persists an income statement.
func (r *Repo) SaveStatement(ctx context.Context, stmt *domain.IncomeStatement) error {
	query := `
		INSERT INTO income_statements (
			id, artisan_id, year, month, order_count,
			gross_paise, fees_paise, net_paise, code, signature, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		stmt.ID, stmt.ArtisanID, stmt.Year, stmt.Month, stmt.OrderCount,
		stmt.GrossPaise, stmt.FeesPaise, stmt.NetPaise, stmt.Code, stmt.Signature, stmt.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert statement: %w", err)
	}

	return nil
}

// GetStatementByCode retrieves a statement by its verification code.
func (r *Repo) GetStatementByCode(ctx context.Context, code string) (*domain.IncomeStatement, error) {
	query := `
		SELECT
			id, artisan_id, year, month, order_count,
			gross_paise, fees_paise, net_paise, code, signature, created_at
		FROM income_statements
		WHERE code = $1
	`

	var stmt domain.IncomeStatement
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&stmt.ID, &stmt.ArtisanID, &stmt.Year, &stmt.Month, &stmt.OrderCount,
		&stmt.GrossPaise, &stmt.FeesPaise, &stmt.NetPaise, &stmt.Code, &stmt.Signature, &stmt.CreatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, pkgdomain.ErrNotFound
		}
		return nil, fmt.Errorf("query statement: %w", err)
	}

	return &stmt, nil
}
