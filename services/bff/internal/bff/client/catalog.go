// services/bff/internal/bff/client/catalog.go
package client

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"

	"github.com/ZoroNewbie00/kalakriti/services/bff/internal/bff/handler"
)

// Catalog is bff's view of core-svc's catalog, ontology, identity and
// curation services, satisfying handler.CatalogService.
type Catalog struct {
	catalog  catalogv1.CatalogServiceClient
	ontology catalogv1.OntologyServiceClient
	identity identityv1.IdentityServiceClient
	media    catalogv1.MediaServiceClient
	curation catalogv1.CurationServiceClient
}

// NewCatalog builds the catalog client, sharing conn with core-svc's other services.
func NewCatalog(conn grpc.ClientConnInterface) *Catalog {
	return &Catalog{
		catalog:  catalogv1.NewCatalogServiceClient(conn),
		ontology: catalogv1.NewOntologyServiceClient(conn),
		identity: identityv1.NewIdentityServiceClient(conn),
		media:    catalogv1.NewMediaServiceClient(conn),
		curation: catalogv1.NewCurationServiceClient(conn),
	}
}

func clusterMemberRoleFromString(s string) identityv1.ClusterMemberRole {
	switch s {
	case "COORDINATOR":
		return identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_COORDINATOR
	case "MASTER":
		return identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_MASTER
	default:
		return identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_MEMBER
	}
}

func clusterToMap(c *catalogv1.Cluster) map[string]any {
	return map[string]any{
		"id":                     c.GetId(),
		"name":                   c.GetName(),
		"craft_ids":              c.GetCraftIds(),
		"state_code":             c.GetRegion().GetStateCode(),
		"district":               c.GetRegion().District,
		"artisan_count":          c.GetArtisanCount(),
		"coordinator_phone_e164": c.CoordinatorPhoneE164,
	}
}

func clusterMemberToMap(m *identityv1.ClusterMember) map[string]any {
	return map[string]any{
		"cluster_id":   m.GetClusterId(),
		"artisan_id":   m.GetArtisanId(),
		"display_name": m.GetDisplayName(),
		"role":         m.GetRole().String(),
		"joined_at":    m.GetJoinedAt().AsTime().Format(time.RFC3339),
	}
}

// CreateCluster registers a cluster in a district.
func (c *Catalog) CreateCluster(ctx context.Context, name, stateCode string, district, coordinatorPhone *string, idempotencyKey string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.CreateCluster(ctx, &identityv1.CreateClusterRequest{
		Name:                 name,
		Region:               &commonv1.GeoRegion{StateCode: stateCode, District: district},
		CoordinatorPhoneE164: coordinatorPhone,
		IdempotencyKey:       idempotencyKey,
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return clusterToMap(resp.GetCluster()), nil
}

// GetCluster fetches one cluster by id.
func (c *Catalog) GetCluster(ctx context.Context, clusterID string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.GetCluster(ctx, &identityv1.GetClusterRequest{ClusterId: clusterID})
	if err != nil {
		return nil, grpcErr(err)
	}
	return clusterToMap(resp.GetCluster()), nil
}

// ListClusterMembers returns a cluster's full roster. Callers on this
// dashboard don't page a single cluster's membership, so this drains every
// page rather than surfacing a cursor the UI would have nowhere to put.
func (c *Catalog) ListClusterMembers(ctx context.Context, clusterID string) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	out := make([]map[string]any, 0, 32)
	var cursor string
	for {
		resp, err := c.identity.ListClusterMembers(ctx, &identityv1.ListClusterMembersRequest{
			ClusterId: clusterID,
			Page:      &commonv1.PageRequest{PageToken: cursor, PageSize: 200},
		})
		if err != nil {
			return nil, grpcErr(err)
		}
		for _, m := range resp.GetMembers() {
			out = append(out, clusterMemberToMap(m))
		}
		cursor = resp.GetPage().GetNextPageToken()
		if cursor == "" || len(resp.GetMembers()) == 0 {
			break
		}
	}
	return out, nil
}

// AddClusterMember adds an artisan to a cluster, or changes their role.
func (c *Catalog) AddClusterMember(ctx context.Context, clusterID, artisanID, role, idempotencyKey string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.AddClusterMember(ctx, &identityv1.AddClusterMemberRequest{
		ClusterId: clusterID, ArtisanId: artisanID,
		Role: clusterMemberRoleFromString(role), IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return clusterMemberToMap(resp.GetMember()), nil
}

// RemoveClusterMember removes an artisan from a cluster.
func (c *Catalog) RemoveClusterMember(ctx context.Context, clusterID, artisanID, idempotencyKey string) (bool, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.RemoveClusterMember(ctx, &identityv1.RemoveClusterMemberRequest{
		ClusterId: clusterID, ArtisanId: artisanID, IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return false, grpcErr(err)
	}
	return resp.GetRemoved(), nil
}

func shgMemberInputsFromMaps(members []map[string]any) []*identityv1.SelfHelpGroupMemberInput {
	out := make([]*identityv1.SelfHelpGroupMemberInput, 0, len(members))
	for _, m := range members {
		artisanID, _ := m["artisan_id"].(string)
		sharePct, _ := m["share_pct"].(float64)
		out = append(out, &identityv1.SelfHelpGroupMemberInput{ArtisanId: artisanID, SharePct: int32(sharePct)})
	}
	return out
}

func shgToMap(g *catalogv1.SelfHelpGroup) map[string]any {
	return map[string]any{
		"id":                   g.GetId(),
		"name":                 g.GetName(),
		"registration_no":      g.GetRegistrationNo(),
		"cluster_id":           g.ClusterId,
		"member_artisan_ids":   g.GetMemberArtisanIds(),
		"signatory_artisan_id": g.SignatoryArtisanId,
	}
}

func shgMembersToMaps(members []*identityv1.SelfHelpGroupMember) []map[string]any {
	out := make([]map[string]any, 0, len(members))
	for _, m := range members {
		out = append(out, map[string]any{
			"self_help_group_id": m.GetSelfHelpGroupId(),
			"artisan_id":         m.GetArtisanId(),
			"display_name":       m.GetDisplayName(),
			"share_pct":          m.GetSharePct(),
			"joined_at":          m.GetJoinedAt().AsTime().Format(time.RFC3339),
		})
	}
	return out
}

// CreateSelfHelpGroup registers an SHG with its founding share split.
// share_pct across members must sum to exactly 100 -- enforced server-side
// in core-svc, not just in the UI's live validator.
func (c *Catalog) CreateSelfHelpGroup(ctx context.Context, name, registrationNo string, clusterID *string, members []map[string]any, idempotencyKey string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.CreateSelfHelpGroup(ctx, &identityv1.CreateSelfHelpGroupRequest{
		Name: name, RegistrationNo: registrationNo, ClusterId: clusterID,
		Members: shgMemberInputsFromMaps(members), IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := shgToMap(resp.GetSelfHelpGroup())
	out["members"] = shgMembersToMaps(resp.GetMembers())
	return out, nil
}

// GetSelfHelpGroup fetches one SHG with its current roster.
func (c *Catalog) GetSelfHelpGroup(ctx context.Context, shgID string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.GetSelfHelpGroup(ctx, &identityv1.GetSelfHelpGroupRequest{SelfHelpGroupId: shgID})
	if err != nil {
		return nil, grpcErr(err)
	}
	group := resp.GetSelfHelpGroup()
	out := shgToMap(group)
	members, err := c.identity.ListSelfHelpGroupMembers(ctx, &identityv1.ListSelfHelpGroupMembersRequest{SelfHelpGroupId: shgID})
	if err == nil {
		out["members"] = shgMembersToMaps(members.GetMembers())
	}
	return out, nil
}

// SetSelfHelpGroupMembers replaces an SHG's roster and share split wholesale.
func (c *Catalog) SetSelfHelpGroupMembers(ctx context.Context, shgID string, members []map[string]any, idempotencyKey string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.SetSelfHelpGroupMembers(ctx, &identityv1.SetSelfHelpGroupMembersRequest{
		SelfHelpGroupId: shgID, Members: shgMemberInputsFromMaps(members), IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	return map[string]any{"members": shgMembersToMaps(resp.GetMembers())}, nil
}

// SuspendListing withdraws a published listing -- a human decision, taken
// through the review queue, never an automated one. See handler.CatalogService.
func (c *Catalog) SuspendListing(ctx context.Context, listingID, reason, idempotencyKey string) (handler.Listing, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.curation.SuspendListing(ctx, &catalogv1.SuspendListingRequest{
		ListingId: listingID, Reason: reason, IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return handler.Listing{}, grpcErr(err)
	}
	l := resp.GetListing()
	title, _ := listingCopy(l)
	return handler.Listing{ID: l.GetId(), ProductID: l.GetProductId(), ArtisanID: l.GetArtisanId(), Title: title,
		Price: l.GetPrice().GetAmountPaise(), Currency: l.GetPrice().GetCurrencyCode()}, nil
}

// ReinstateListing returns a suspended listing to PUBLISHED.
func (c *Catalog) ReinstateListing(ctx context.Context, listingID, idempotencyKey string) (handler.Listing, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.curation.ReinstateListing(ctx, &catalogv1.ReinstateListingRequest{
		ListingId: listingID, IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return handler.Listing{}, grpcErr(err)
	}
	l := resp.GetListing()
	title, _ := listingCopy(l)
	return handler.Listing{ID: l.GetId(), ProductID: l.GetProductId(), ArtisanID: l.GetArtisanId(), Title: title,
		Price: l.GetPrice().GetAmountPaise(), Currency: l.GetPrice().GetCurrencyCode()}, nil
}

// RefreshCraftIndex rebuilds the ontology alias index from Postgres.
func (c *Catalog) RefreshCraftIndex(ctx context.Context, idempotencyKey string) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.ontology.RefreshCraftIndex(ctx, &catalogv1.RefreshCraftIndexRequest{IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, grpcErr(err)
	}
	stats := resp.GetStats()
	return map[string]any{
		"version":     stats.GetVersion(),
		"craft_count": stats.GetCraftCount(),
		"alias_count": stats.GetAliasCount(),
		"built_at":    stats.GetBuiltAt().AsTime().Format(time.RFC3339),
	}, nil
}

// resolveMediaURL returns a signed GET URL for one asset, or "" if the ref is
// nil or the signing call fails -- a missing image degrades a card, it
// shouldn't fail the page.
func (c *Catalog) resolveMediaURL(ctx context.Context, ref *commonv1.MediaRef) string {
	if ref == nil || ref.GetId() == "" {
		return ""
	}
	resp, err := c.media.GetMediaURL(ctx, &catalogv1.GetMediaURLRequest{MediaId: ref.GetId()})
	if err != nil {
		return ""
	}
	return resp.GetUrl()
}

// firstImage picks the first IMAGE-kind ref out of a product's media, for use
// as a listing's card thumbnail.
func firstImage(media []*commonv1.MediaRef) *commonv1.MediaRef {
	for _, ref := range media {
		if ref.GetKind() == commonv1.MediaKind_MEDIA_KIND_IMAGE {
			return ref
		}
	}
	return nil
}

func craftToHandler(craft *catalogv1.Craft) handler.Craft {
	regions := make([]string, 0, len(craft.GetRegions()))
	for _, r := range craft.GetRegions() {
		regions = append(regions, regionLocation(r))
	}
	return handler.Craft{
		ID:               craft.GetId(),
		Slug:             craft.GetCode(),
		DisplayName:      craft.GetDisplayName(),
		GIRegistrationNo: craft.GiRegistrationNo,
		Regions:          regions,
		Techniques:       craft.GetTechniques(),
		Materials:        craft.GetMaterials(),
	}
}

// ListCrafts returns the whole ontology; it is a few hundred rows and is
// served from memory on the ontology side.
func (c *Catalog) ListCrafts(ctx context.Context) ([]handler.Craft, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.ontology.ListCrafts(ctx, &catalogv1.ListCraftsRequest{})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := make([]handler.Craft, 0, len(resp.GetCrafts()))
	for _, craft := range resp.GetCrafts() {
		out = append(out, craftToHandler(craft))
	}
	return out, nil
}

// GetCraftBySlug resolves a craft landing page by its ontology slug.
func (c *Catalog) GetCraftBySlug(ctx context.Context, slug string) (*handler.Craft, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{Code: slug})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := craftToHandler(resp.GetCraft())
	return &out, nil
}

// ListArtisansByCraft finds makers with a published listing under this
// craft. There is no dedicated "artisans by craft" read, so this scans
// published listings filtered by craft and dedupes their artisan_id --
// bounded by limit, acceptable at catalogue sizes a hackathon demo runs at.
//
// ponytail: an artisan-craft junction read would make this one query instead
// of a listings scan plus N identity lookups; add one if the craft page ever
// needs to page past the first screen of makers.
func (c *Catalog) ListArtisansByCraft(ctx context.Context, craftID string, limit int32) ([]handler.Artisan, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.ListListings(ctx, &catalogv1.ListListingsRequest{
		CraftId: &craftID,
		State:   catalogv1.ListingState_LISTING_STATE_PUBLISHED,
		Page:    &commonv1.PageRequest{PageSize: limit * 4},
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	seen := make(map[string]struct{}, limit)
	out := make([]handler.Artisan, 0, limit)
	for _, listing := range resp.GetListings() {
		id := listing.GetArtisanId()
		if _, dup := seen[id]; dup || id == "" {
			continue
		}
		seen[id] = struct{}{}
		if int32(len(out)) >= limit {
			break
		}
		if art, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: id}); err == nil {
			out = append(out, handler.Artisan{
				ID:          art.GetArtisan().GetId(),
				DisplayName: art.GetArtisan().GetDisplayName(),
				ClusterID:   art.GetArtisan().ClusterId,
			})
		}
	}
	return out, nil
}

// ListProcessClips derives the process-provenance feed from recent published
// listings' own product video, newest first. There is no dedicated feed
// store: search-svc and catalog-svc both index by listing, not by media
// asset, so this scans listings and keeps the first video each carries.
//
// ponytail: a scan over recent listings, not an index -- fine at hackathon
// scale. Add a real feed table (or a media_kind=VIDEO search facet) if the
// catalogue grows past a few thousand published listings.
func (c *Catalog) ListProcessClips(ctx context.Context, limit int32) ([]handler.ProcessClip, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	scanLimit := limit * 3
	resp, err := c.catalog.ListListings(ctx, &catalogv1.ListListingsRequest{
		State: catalogv1.ListingState_LISTING_STATE_PUBLISHED,
		Page:  &commonv1.PageRequest{PageSize: scanLimit},
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]handler.ProcessClip, 0, limit)
	for _, listing := range resp.GetListings() {
		if int32(len(out)) >= limit {
			break
		}
		full, err := c.catalog.GetListing(ctx, &catalogv1.GetListingRequest{
			ListingId: listing.GetId(), IncludeProduct: true,
		})
		if err != nil || full.GetProduct() == nil {
			continue
		}
		var video *commonv1.MediaRef
		for _, ref := range full.GetProduct().GetMedia() {
			if ref.GetKind() == commonv1.MediaKind_MEDIA_KIND_VIDEO {
				video = ref
				break
			}
		}
		if video == nil {
			continue
		}
		title, _ := listingCopy(listing)
		clip := handler.ProcessClip{
			ListingID:   listing.GetId(),
			ListingSlug: buildSlug(title, listing.GetId()),
			Title:       title,
			ArtisanID:   listing.GetArtisanId(),
			VideoURL:    c.resolveMediaURL(ctx, video),
			DurationMs:  video.DurationMs,
		}
		if art, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: listing.GetArtisanId()}); err == nil {
			clip.ArtisanName = art.GetArtisan().GetDisplayName()
		}
		out = append(out, clip)
	}
	return out, nil
}

func (c *Catalog) GetProvenanceByShortCode(ctx context.Context, code string) (handler.ProvenanceRecord, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.GetProvenanceByShortCode(ctx, &catalogv1.GetProvenanceByShortCodeRequest{ShortCode: code})
	if err != nil {
		return handler.ProvenanceRecord{}, grpcErr(err)
	}
	rec := resp.GetRecord()
	return handler.ProvenanceRecord{
		ID:               rec.GetId(),
		ListingID:        rec.GetListingId(),
		ArtisanID:        rec.GetArtisanId(),
		CraftID:          rec.GetCraftId(),
		ContentHash:      rec.GetContentHash(),
		PreviousHash:     rec.PreviousHash,
		Signature:        rec.GetSignature(),
		SignatureAlgo:    rec.GetSignatureAlgorithm(),
		PublicKeyID:      rec.GetPublicKeyId(),
		ShortCode:        rec.GetShortCode(),
		TechniqueMatched: rec.GetTechniqueMatched(),
		MediaHashes:      rec.GetMediaHashes(),
		SealedAt:         rec.GetSealedAt().AsTime(),
	}, nil
}

// GetListingBySlug resolves a slug built by buildSlug back to its id
// (parseSlugID) and fetches the listing. The slug is derived, not stored —
// no proto change, no migration — so it stays valid even if the title later
// changes; the id is the only part that must round-trip.
func (c *Catalog) GetListingBySlug(ctx context.Context, slug string) (*handler.ListingDetail, error) {
	id, ok := parseSlugID(slug)
	if !ok {
		return nil, domain.NotFound("listing not found")
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.GetListing(ctx, &catalogv1.GetListingRequest{ListingId: id, IncludeProduct: true})
	if err != nil {
		return nil, grpcErr(err)
	}
	listing := resp.GetListing()
	title, description := listingCopy(listing)

	var artisanName string
	if art, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: listing.GetArtisanId()}); err == nil {
		artisanName = art.GetArtisan().GetDisplayName()
	}
	var craftName, imageURL string
	if product := resp.GetProduct(); product != nil {
		if craft, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: product.GetCraftId()}); err == nil {
			craftName = craft.GetCraft().GetDisplayName()
		}
		imageURL = c.resolveMediaURL(ctx, firstImage(product.GetMedia()))
	}

	return &handler.ListingDetail{
		ID:          listing.GetId(),
		Slug:        buildSlug(title, listing.GetId()),
		Title:       title,
		Description: description,
		PricePaise:  listing.GetPrice().GetAmountPaise(),
		Currency:    listing.GetPrice().GetCurrencyCode(),
		ImageURL:    imageURL,
		ArtisanName: artisanName,
		CraftName:   craftName,
		Available:   listing.GetState() == catalogv1.ListingState_LISTING_STATE_PUBLISHED,
	}, nil
}

// GetArtisanBySlug mirrors GetListingBySlug for artisan profile pages.
func (c *Catalog) GetArtisanBySlug(ctx context.Context, slug string) (*handler.ArtisanProfile, error) {
	id, ok := parseSlugID(slug)
	if !ok {
		return nil, domain.NotFound("artisan not found")
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: id})
	if err != nil {
		return nil, grpcErr(err)
	}
	art := resp.GetArtisan()

	var craftName, craftSlug string
	if craftIDs := art.GetCraftIds(); len(craftIDs) > 0 {
		if craft, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: craftIDs[0]}); err == nil {
			craftName = craft.GetCraft().GetDisplayName()
			craftSlug = craft.GetCraft().GetCode()
		}
	}

	var bio string
	if art.Bio != nil {
		bio = *art.Bio
	}

	return &handler.ArtisanProfile{
		ID:              art.GetId(),
		Slug:            buildSlug(art.GetDisplayName(), art.GetId()),
		DisplayName:     art.GetDisplayName(),
		Bio:             bio,
		Location:        regionLocation(art.GetRegion()),
		District:        art.GetRegion().District,
		StateCode:       art.GetRegion().GetStateCode(),
		ClusterID:       art.ClusterId,
		ImageURL:        c.resolveMediaURL(ctx, art.GetPhoto()),
		CraftName:       craftName,
		CraftSlug:       craftSlug,
		Verified:        art.GetVerified(),
		YearsExperience: art.YearsOfExperience,
	}, nil
}

// regionLocation renders a GeoRegion as a short human-readable location.
func regionLocation(r *commonv1.GeoRegion) string {
	if r == nil {
		return ""
	}
	var parts []string
	if r.District != nil && *r.District != "" {
		parts = append(parts, *r.District)
	}
	if r.GetStateCode() != "" {
		parts = append(parts, r.GetStateCode())
	}
	return strings.Join(parts, ", ")
}

// ListPublishedListings feeds the sitemap. ListListingsRequest pages by
// opaque cursor, not numeric offset, so only the first page (offset 0) is
// servable without persisting a cursor between calls; anything else comes
// back empty rather than silently returning the first page again under a
// different offset's URL.
func (c *Catalog) ListPublishedListings(ctx context.Context, limit, offset int32) ([]handler.ListingDetail, error) {
	if offset != 0 {
		return nil, nil
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.ListListings(ctx, &catalogv1.ListListingsRequest{
		State: catalogv1.ListingState_LISTING_STATE_PUBLISHED,
		Page:  &commonv1.PageRequest{PageSize: limit},
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	out := make([]handler.ListingDetail, 0, len(resp.GetListings()))
	for _, l := range resp.GetListings() {
		title, _ := listingCopy(l)
		out = append(out, handler.ListingDetail{
			ID:         l.GetId(),
			Slug:       buildSlug(title, l.GetId()),
			Title:      title,
			PricePaise: l.GetPrice().GetAmountPaise(),
			Currency:   l.GetPrice().GetCurrencyCode(),
			Available:  true,
		})
	}
	return out, nil
}

func (c *Catalog) GetListing(ctx context.Context, listingID string) (handler.Listing, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.catalog.GetListing(ctx, &catalogv1.GetListingRequest{ListingId: listingID})
	if err != nil {
		return handler.Listing{}, grpcErr(err)
	}

	l := resp.GetListing()
	title, _ := listingCopy(l)
	return handler.Listing{
		ID:        l.GetId(),
		ProductID: l.GetProductId(),
		ArtisanID: l.GetArtisanId(),
		Title:     title,
		Price:     l.GetPrice().GetAmountPaise(),
		Currency:  l.GetPrice().GetCurrencyCode(),
	}, nil
}

func (c *Catalog) GetArtisan(ctx context.Context, artisanID string) (handler.Artisan, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.identity.GetArtisan(ctx, &identityv1.GetArtisanRequest{ArtisanId: artisanID})
	if err != nil {
		return handler.Artisan{}, grpcErr(err)
	}

	art := resp.GetArtisan()
	return handler.Artisan{
		ID:          art.GetId(),
		DisplayName: art.GetDisplayName(),
		District:    art.GetRegion().GetDistrict(),
		ClusterID:   art.ClusterId,
	}, nil
}

func (c *Catalog) GetCraft(ctx context.Context, craftID string) (handler.Craft, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	resp, err := c.ontology.GetCraft(ctx, &catalogv1.GetCraftRequest{CraftId: craftID})
	if err != nil {
		return handler.Craft{}, grpcErr(err)
	}

	craft := resp.GetCraft()
	return handler.Craft{
		ID:               craft.GetId(),
		DisplayName:      craft.GetDisplayName(),
		GIRegistrationNo: craft.GiRegistrationNo,
	}, nil
}

// listingCopy picks a listing's English translation, falling back to
// whichever translation came back first if English isn't among them.
func listingCopy(listing *catalogv1.Listing) (title, description string) {
	translations := listing.GetTranslations()
	for _, t := range translations {
		if t.GetLanguage() == commonv1.Language_LANGUAGE_ENGLISH {
			return t.GetTitle(), t.GetDescription()
		}
	}
	if len(translations) > 0 {
		return translations[0].GetTitle(), translations[0].GetDescription()
	}
	return "", ""
}
