// services/core-svc/internal/core/domain/trends.go

package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
)

// TrendSourceType classifies where a trend link points.
type TrendSourceType string

const (
	TrendSourceInstagram TrendSourceType = "INSTAGRAM"
	TrendSourcePinterest TrendSourceType = "PINTEREST"
	TrendSourceBlog      TrendSourceType = "BLOG"
	TrendSourceNews      TrendSourceType = "NEWS"
	TrendSourceYouTube   TrendSourceType = "YOUTUBE"
	TrendSourceOther     TrendSourceType = "OTHER"
)

// TrendLink is one curated reference to external market-trend content.
type TrendLink struct {
	ID                uuid.UUID
	Title             string
	Description       string
	URL               string
	SourceType        TrendSourceType
	CraftID           *uuid.UUID
	CraftName         *string
	ThumbnailURL      *string
	EmbedHTML         *string
	EmbedMetadataJSON *string
	Pinned            bool
	CreatedBy         string
	CuratorRole       string
	CuratorName       string
	ExpiresAt         *time.Time
	CreatedAt         time.Time
}

// CreateTrendLinkInput is what the service needs to create a trend link.
type CreateTrendLinkInput struct {
	Title          string
	Description    string
	URL            string
	SourceType     TrendSourceType
	CraftID        *uuid.UUID
	ThumbnailURL   *string
	AutoFetchEmbed bool
	CreatedBy      string
	CuratorRole    string
	CuratorName    string
}

// Validate checks trend link creation fields.
func (in CreateTrendLinkInput) Validate() error {
	if strings.TrimSpace(in.Title) == "" {
		return fmt.Errorf("title is required: %w", pkgdomain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.URL) == "" {
		return fmt.Errorf("url is required: %w", pkgdomain.ErrInvalidInput)
	}
	if !strings.HasPrefix(in.URL, "http://") && !strings.HasPrefix(in.URL, "https://") {
		return fmt.Errorf("url must start with http:// or https://: %w", pkgdomain.ErrInvalidInput)
	}
	switch in.SourceType {
	case TrendSourceInstagram, TrendSourcePinterest, TrendSourceBlog,
		TrendSourceNews, TrendSourceYouTube, TrendSourceOther:
		// valid
	default:
		return fmt.Errorf("invalid source_type %q: %w", in.SourceType, pkgdomain.ErrInvalidInput)
	}
	return nil
}

// TrendFilter holds query parameters for listing trend links.
type TrendFilter struct {
	CraftID        *uuid.UUID
	SourceType     *TrendSourceType
	ExcludeExpired bool
	Page           Page
}
