// services/core-svc/internal/core/service/trends.go

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// TrendStore is the persistence surface for trend links.
type TrendStore interface {
	CreateTrendLink(ctx context.Context, link domain.TrendLink) (domain.TrendLink, error)
	GetTrendLink(ctx context.Context, id uuid.UUID) (domain.TrendLink, error)
	ListTrendLinks(ctx context.Context, filter domain.TrendFilter) ([]domain.TrendLink, error)
	DeleteTrendLink(ctx context.Context, id uuid.UUID) (bool, error)
	PinTrendLink(ctx context.Context, id uuid.UUID, pinned bool) (domain.TrendLink, error)
}

// Trends is the service for curated market trend links.
type Trends struct {
	store  TrendStore
	log    *slog.Logger
	client *http.Client
	now    func() time.Time
}

// NewTrends builds the trends service.
func NewTrends(store TrendStore, log *slog.Logger) *Trends {
	return &Trends{
		store:  store,
		log:    log,
		client: &http.Client{Timeout: 10 * time.Second},
		now:    time.Now,
	}
}

// CreateTrendLink creates a new trend link, optionally fetching oEmbed metadata.
func (s *Trends) CreateTrendLink(ctx context.Context, in domain.CreateTrendLinkInput) (domain.TrendLink, error) {
	// Both Admins (MINISTRY, CLUSTER_OFFICER) and Artisans can curate market trend links.
	p, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return domain.TrendLink{}, fmt.Errorf("not authenticated: %w", pkgdomain.ErrForbidden)
	}
	role := "ARTISAN"
	if p.HasRole(auth.RoleMinistry) {
		role = "MINISTRY"
	} else if p.HasRole(auth.RoleClusterOfficer) {
		role = "CLUSTER_OFFICER"
	} else if !p.HasRole(auth.RoleArtisan) {
		return domain.TrendLink{}, fmt.Errorf("insufficient permissions: %w", pkgdomain.ErrForbidden)
	}

	if err := in.Validate(); err != nil {
		return domain.TrendLink{}, err
	}

	curatorName := in.CuratorName
	if strings.TrimSpace(curatorName) == "" {
		curatorName = p.Subject
	}

	link := domain.TrendLink{
		ID:           ids.New(),
		Title:        in.Title,
		Description:  in.Description,
		URL:          in.URL,
		SourceType:   in.SourceType,
		CraftID:      in.CraftID,
		ThumbnailURL: in.ThumbnailURL,
		Pinned:       false,
		CreatedBy:    p.Subject,
		CuratorRole:  role,
		CuratorName:  curatorName,
		CreatedAt:    s.now().UTC(),
	}

	// Auto-fetch oEmbed metadata if requested.
	if in.AutoFetchEmbed {
		embed, embedErr := s.fetchOEmbed(ctx, in.URL, in.SourceType)
		if embedErr != nil {
			s.log.Warn("oembed_fetch_failed", "url", in.URL, "error", embedErr)
			// Non-fatal: create the link without embed data.
		} else {
			link.EmbedHTML = embed.HTML
			link.ThumbnailURL = firstNonNil(link.ThumbnailURL, embed.ThumbnailURL)
			if embed.RawJSON != nil {
				raw := string(embed.RawJSON)
				link.EmbedMetadataJSON = &raw
			}
		}
	}

	created, err := s.store.CreateTrendLink(ctx, link)
	if err != nil {
		return domain.TrendLink{}, err
	}

	s.log.Info("trend_link_created", "id", created.ID, "source", created.SourceType)
	return created, nil
}

// ListTrendLinks lists trend links with optional filters.
func (s *Trends) ListTrendLinks(ctx context.Context, filter domain.TrendFilter) ([]domain.TrendLink, error) {
	filter.Page = filter.Page.Normalise()
	return s.store.ListTrendLinks(ctx, filter)
}

// DeleteTrendLink removes a trend link.
func (s *Trends) DeleteTrendLink(ctx context.Context, id uuid.UUID) error {
	p, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return fmt.Errorf("not authenticated: %w", pkgdomain.ErrForbidden)
	}
	if !p.HasRole(auth.RoleMinistry) && !p.HasRole(auth.RoleClusterOfficer) {
		return fmt.Errorf("insufficient permissions: %w", pkgdomain.ErrForbidden)
	}

	deleted, err := s.store.DeleteTrendLink(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("trend_link not found: %w", pkgdomain.ErrNotFound)
	}

	s.log.Info("trend_link_deleted", "id", id)
	return nil
}

// PinTrendLink pins or unpins a trend link.
func (s *Trends) PinTrendLink(ctx context.Context, id uuid.UUID, pinned bool) (domain.TrendLink, error) {
	p, err := auth.RequirePrincipal(ctx)
	if err != nil {
		return domain.TrendLink{}, fmt.Errorf("not authenticated: %w", pkgdomain.ErrForbidden)
	}
	if !p.HasRole(auth.RoleMinistry) {
		return domain.TrendLink{}, fmt.Errorf("only MINISTRY can pin trends: %w", pkgdomain.ErrForbidden)
	}

	link, err := s.store.PinTrendLink(ctx, id, pinned)
	if err != nil {
		return domain.TrendLink{}, err
	}

	s.log.Info("trend_link_pinned", "id", id, "pinned", pinned)
	return link, nil
}

// oEmbedResult holds the parsed oEmbed response.
type oEmbedResult struct {
	HTML         *string
	ThumbnailURL *string
	RawJSON      json.RawMessage
}

// fetchOEmbed fetches oEmbed metadata for a URL based on its source type.
func (s *Trends) fetchOEmbed(ctx context.Context, rawURL string, source domain.TrendSourceType) (oEmbedResult, error) {
	var endpoint string

	switch source {
	case domain.TrendSourceInstagram:
		endpoint = "https://graph.facebook.com/v18.0/instagram_oembed?url=" + url.QueryEscape(rawURL)
	case domain.TrendSourceYouTube:
		endpoint = "https://www.youtube.com/oembed?format=json&url=" + url.QueryEscape(rawURL)
	case domain.TrendSourcePinterest:
		endpoint = "https://www.pinterest.com/oembed.json?url=" + url.QueryEscape(rawURL)
	default:
		// Try noembed.com as a generic fallback for other sources.
		endpoint = "https://noembed.com/embed?url=" + url.QueryEscape(rawURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return oEmbedResult{}, fmt.Errorf("building oembed request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return oEmbedResult{}, fmt.Errorf("oembed fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oEmbedResult{}, fmt.Errorf("oembed returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return oEmbedResult{}, fmt.Errorf("reading oembed response: %w", err)
	}

	var parsed struct {
		HTML         string `json:"html"`
		ThumbnailURL string `json:"thumbnail_url"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return oEmbedResult{}, fmt.Errorf("parsing oembed response: %w", err)
	}

	result := oEmbedResult{RawJSON: body}
	if strings.TrimSpace(parsed.HTML) != "" {
		result.HTML = &parsed.HTML
	}
	if strings.TrimSpace(parsed.ThumbnailURL) != "" {
		result.ThumbnailURL = &parsed.ThumbnailURL
	}

	return result, nil
}

// firstNonNil returns the first non-nil string pointer, or nil.
func firstNonNil(ptrs ...*string) *string {
	for _, p := range ptrs {
		if p != nil {
			return p
		}
	}
	return nil
}
