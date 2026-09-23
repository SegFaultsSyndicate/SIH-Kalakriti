package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/domain"
	"github.com/ZoroNewbie00/kalakriti/services/insight-svc/internal/insight/repo/db"
)

func (r *Repo) ListImpactArtisans(ctx context.Context, f domain.ImpactFilter) ([]domain.ImpactArtisan, error) {
	rows, err := r.q.ListImpactArtisans(ctx, db.ListImpactArtisansParams{
		StateCode: f.StateCode, District: f.District, SocialCategory: f.SocialCategory, Corporation: f.Corporation,
	})
	if err != nil {
		return nil, fmt.Errorf("impact artisans: %w", err)
	}
	out := make([]domain.ImpactArtisan, len(rows))
	for i, row := range rows {
		out[i] = domain.ImpactArtisan{
			ArtisanID: row.ArtisanID, StateCode: row.StateCode, District: row.District,
			SocialCategory: row.SocialCategory, Corporations: row.Corporations, FinanceVerified: row.FinanceVerified,
			RegisteredAt: row.RegisteredAt, BaselineBracket: row.BaselineBracket, BaselineMonthlyPaise: row.BaselineMonthlyPaise,
			Platform90d: row.PlatformPaise90d, Offline90d: row.OfflinePaise90d, Fair90d: row.FairPaise90d,
			LastSaleAt: epochToNil(row.LastSaleAt),
		}
	}
	return out, nil
}

func (r *Repo) GetSalesMix(ctx context.Context, f domain.ImpactFilter) ([]domain.SalesMixMonth, error) {
	rows, err := r.q.GetSalesMixByMonth(ctx, db.GetSalesMixByMonthParams{
		StateCode: f.StateCode, District: f.District, SocialCategory: f.SocialCategory, Corporation: f.Corporation,
		FromMonth: optDate(f.FromMonth), ToMonth: optDate(f.ToMonth),
	})
	if err != nil {
		return nil, fmt.Errorf("sales mix: %w", err)
	}
	out := make([]domain.SalesMixMonth, len(rows))
	for i, row := range rows {
		out[i] = domain.SalesMixMonth{
			Month: row.Month.Time, ArtisanCount: row.ArtisanCount, PlatformPaise: row.PlatformPaise,
			FairPaise: row.FairPaise, OtherOfflinePaise: row.OtherOfflinePaise,
		}
	}
	return out, nil
}

func (r *Repo) ListFinanceCoverageFacts(ctx context.Context, f domain.ImpactFilter) ([]domain.FinanceCoverageFact, error) {
	rows, err := r.q.ListFinanceCoverageFacts(ctx, db.ListFinanceCoverageFactsParams{
		StateCode: f.StateCode, District: f.District, SocialCategory: f.SocialCategory, Corporation: f.Corporation,
	})
	if err != nil {
		return nil, fmt.Errorf("finance coverage: %w", err)
	}
	out := make([]domain.FinanceCoverageFact, len(rows))
	for i, row := range rows {
		out[i] = domain.FinanceCoverageFact{
			ArtisanID: row.ArtisanID, Corporation: row.Corporation, Status: row.Status, EmiPaise: row.EmiPaise,
			Platform90d: row.PlatformPaise90d, Offline90d: row.OfflinePaise90d,
		}
	}
	return out, nil
}

func (r *Repo) GetLiteracyFunnel(ctx context.Context, f domain.ImpactFilter) ([]domain.LiteracyFunnelRow, error) {
	rows, err := r.q.GetLiteracyFunnel(ctx, db.GetLiteracyFunnelParams{
		StateCode: f.StateCode, District: f.District, SocialCategory: f.SocialCategory, Corporation: f.Corporation,
	})
	if err != nil {
		return nil, fmt.Errorf("literacy funnel: %w", err)
	}
	out := make([]domain.LiteracyFunnelRow, len(rows))
	for i, row := range rows {
		out[i] = domain.LiteracyFunnelRow{
			StateCode: row.StateCode, District: row.District, Artisans: row.Artisans,
			Started: row.Started, HalfWay: row.HalfWay, Certified: row.Certified,
		}
	}
	return out, nil
}

// GetImpactViewsRefreshedAt returns nil when the views were never refreshed.
func (r *Repo) GetImpactViewsRefreshedAt(ctx context.Context) (*time.Time, error) {
	t, err := r.q.GetImpactViewsRefreshedAt(ctx)
	if err != nil {
		return nil, fmt.Errorf("views refreshed at: %w", err)
	}
	return epochToNil(t), nil
}

func (r *Repo) CountCompletedLessons(ctx context.Context, artisanID uuid.UUID) (int64, error) {
	return r.q.CountCompletedLessons(ctx, artisanID)
}

// CreateLiteracyCertificate returns nil (no error) when the artisan already
// holds a certificate, i.e. a concurrent issue won.
func (r *Repo) CreateLiteracyCertificate(ctx context.Context, c domain.LiteracyCertificate) (*domain.LiteracyCertificate, error) {
	row, err := r.q.CreateLiteracyCertificate(ctx, db.CreateLiteracyCertificateParams{
		ID: c.ID, ArtisanID: c.ArtisanID, ShortCode: c.ShortCode, Signature: c.Signature,
		PublicKeyID: c.PublicKeyID, S3Key: c.S3Key,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("creating literacy certificate: %w", err)
	}
	return &domain.LiteracyCertificate{ID: row.ID, ArtisanID: row.ArtisanID, IssuedAt: row.IssuedAt, ShortCode: row.ShortCode, S3Key: row.S3Key}, nil
}

// GetLiteracyCertificateByArtisan returns nil when none has been issued.
func (r *Repo) GetLiteracyCertificateByArtisan(ctx context.Context, artisanID uuid.UUID) (*domain.LiteracyCertificate, error) {
	row, err := r.q.GetLiteracyCertificateByArtisan(ctx, artisanID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("literacy certificate: %w", err)
	}
	return &domain.LiteracyCertificate{ID: row.ID, ArtisanID: row.ArtisanID, IssuedAt: row.IssuedAt, ShortCode: row.ShortCode, S3Key: row.S3Key}, nil
}

// GetLiteracyCertificateByCode returns nil for an unknown code.
func (r *Repo) GetLiteracyCertificateByCode(ctx context.Context, code string) (*domain.LiteracyCertificate, error) {
	row, err := r.q.GetLiteracyCertificateByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("literacy certificate by code: %w", err)
	}
	return &domain.LiteracyCertificate{
		ID: row.ID, IssuedAt: row.IssuedAt, ShortCode: row.ShortCode, Signature: row.Signature,
		PublicKeyID: row.PublicKeyID, S3Key: row.S3Key, ArtisanName: row.ArtisanName,
		StateCode: row.StateCode, District: row.District, CraftName: row.CraftName,
	}, nil
}

func (r *Repo) LiteracyCertificateShortCodeExists(ctx context.Context, code string) (bool, error) {
	return r.q.LiteracyCertificateShortCodeExists(ctx, code)
}

func (r *Repo) GetArtisanName(ctx context.Context, artisanID uuid.UUID) (string, error) {
	name, err := r.q.GetArtisanDisplayName(ctx, artisanID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", pkgdomain.NotFound("artisan not found")
	}
	return name, err
}

// epochToNil undoes the 'epoch' sentinel the queries use for "never",
// because sqlc cannot type a nullable expression column.
func epochToNil(t time.Time) *time.Time {
	if t.Unix() == 0 {
		return nil
	}
	return &t
}

func optDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}
