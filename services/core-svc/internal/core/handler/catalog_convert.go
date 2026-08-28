// services/core-svc/internal/core/handler/catalog_convert.go
package handler

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// defaultCurrency is what a listing is priced in when the wire message leaves
// the currency code unset; the schema allows nothing else today.
const defaultCurrency = "INR"

// languageNameFromProto renders one proto language enum as the plain name the
// language_code Postgres enum stores, or "" for UNSPECIFIED.
func languageNameFromProto(l commonv1.Language) string {
	if l == commonv1.Language_LANGUAGE_UNSPECIFIED {
		return ""
	}
	return trimEnumPrefix(l.String(), "LANGUAGE_")
}

// languageNameToProto converts one stored language name to its proto enum.
func languageNameToProto(name string) commonv1.Language {
	if v, ok := commonv1.Language_value["LANGUAGE_"+name]; ok {
		return commonv1.Language(v)
	}
	return commonv1.Language_LANGUAGE_UNSPECIFIED
}

// listingTypeFromProto maps the wire type onto the domain type. UNSPECIFIED is
// passed through as the empty type so domain validation names the field.
func listingTypeFromProto(t catalogv1.ListingType) domain.ListingType {
	switch t {
	case catalogv1.ListingType_LISTING_TYPE_MADE_TO_ORDER:
		return domain.ListingMadeToOrder
	case catalogv1.ListingType_LISTING_TYPE_READY_STOCK:
		return domain.ListingReadyStock
	default:
		return ""
	}
}

// listingTypeToProto maps the domain type onto the wire type.
func listingTypeToProto(t domain.ListingType) catalogv1.ListingType {
	switch t {
	case domain.ListingMadeToOrder:
		return catalogv1.ListingType_LISTING_TYPE_MADE_TO_ORDER
	case domain.ListingReadyStock:
		return catalogv1.ListingType_LISTING_TYPE_READY_STOCK
	default:
		return catalogv1.ListingType_LISTING_TYPE_UNSPECIFIED
	}
}

// listingStateFromProto maps the wire state onto the domain state.
func listingStateFromProto(s catalogv1.ListingState) domain.ListingState {
	switch s {
	case catalogv1.ListingState_LISTING_STATE_DRAFT:
		return domain.StateDraft
	case catalogv1.ListingState_LISTING_STATE_PENDING_ARTISAN_APPROVAL:
		return domain.StatePendingArtisanApproval
	case catalogv1.ListingState_LISTING_STATE_PUBLISHED:
		return domain.StatePublished
	case catalogv1.ListingState_LISTING_STATE_SUSPENDED:
		return domain.StateSuspended
	default:
		return ""
	}
}

// listingStateToProto maps the domain state onto the wire state.
func listingStateToProto(s domain.ListingState) catalogv1.ListingState {
	switch s {
	case domain.StateDraft:
		return catalogv1.ListingState_LISTING_STATE_DRAFT
	case domain.StatePendingArtisanApproval:
		return catalogv1.ListingState_LISTING_STATE_PENDING_ARTISAN_APPROVAL
	case domain.StatePublished:
		return catalogv1.ListingState_LISTING_STATE_PUBLISHED
	case domain.StateSuspended:
		return catalogv1.ListingState_LISTING_STATE_SUSPENDED
	default:
		return catalogv1.ListingState_LISTING_STATE_UNSPECIFIED
	}
}

// attributeSourceFromProto maps the wire source onto the domain source.
func attributeSourceFromProto(s catalogv1.AttributeSource) domain.AttributeSource {
	switch s {
	case catalogv1.AttributeSource_ATTRIBUTE_SOURCE_MODEL:
		return domain.SourceModel
	case catalogv1.AttributeSource_ATTRIBUTE_SOURCE_ARTISAN:
		return domain.SourceArtisan
	case catalogv1.AttributeSource_ATTRIBUTE_SOURCE_CURATOR:
		return domain.SourceCurator
	default:
		return ""
	}
}

// attributeSourceToProto maps the domain source onto the wire source.
func attributeSourceToProto(s domain.AttributeSource) catalogv1.AttributeSource {
	switch s {
	case domain.SourceModel:
		return catalogv1.AttributeSource_ATTRIBUTE_SOURCE_MODEL
	case domain.SourceArtisan:
		return catalogv1.AttributeSource_ATTRIBUTE_SOURCE_ARTISAN
	case domain.SourceCurator:
		return catalogv1.AttributeSource_ATTRIBUTE_SOURCE_CURATOR
	default:
		return catalogv1.AttributeSource_ATTRIBUTE_SOURCE_UNSPECIFIED
	}
}

// mediaRoleFromProto maps the wire role onto the domain role, defaulting an
// unspecified role to an ordinary gallery asset.
func mediaRoleFromProto(r catalogv1.ListingMediaRole) domain.ListingMediaRole {
	switch r {
	case catalogv1.ListingMediaRole_LISTING_MEDIA_ROLE_PRIMARY_IMAGE:
		return domain.MediaRolePrimaryImage
	case catalogv1.ListingMediaRole_LISTING_MEDIA_ROLE_PROCESS_VIDEO:
		return domain.MediaRoleProcessVideo
	default:
		return domain.MediaRoleGallery
	}
}

// mediaRoleToProto maps the domain role onto the wire role.
func mediaRoleToProto(r domain.ListingMediaRole) catalogv1.ListingMediaRole {
	switch r {
	case domain.MediaRolePrimaryImage:
		return catalogv1.ListingMediaRole_LISTING_MEDIA_ROLE_PRIMARY_IMAGE
	case domain.MediaRoleProcessVideo:
		return catalogv1.ListingMediaRole_LISTING_MEDIA_ROLE_PROCESS_VIDEO
	default:
		return catalogv1.ListingMediaRole_LISTING_MEDIA_ROLE_GALLERY
	}
}

// mediaKindToProto maps a stored media kind onto the wire enum.
func mediaKindToProto(k domain.MediaKind) commonv1.MediaKind {
	if v, ok := commonv1.MediaKind_value["MEDIA_KIND_"+string(k)]; ok {
		return commonv1.MediaKind(v)
	}
	return commonv1.MediaKind_MEDIA_KIND_UNSPECIFIED
}

// matchSourceToProto maps how a craft was matched onto the wire enum.
func matchSourceToProto(s domain.MatchSource) catalogv1.MatchSource {
	switch s {
	case domain.MatchExact:
		return catalogv1.MatchSource_MATCH_SOURCE_EXACT
	case domain.MatchTransliterated:
		return catalogv1.MatchSource_MATCH_SOURCE_TRANSLITERATED
	default:
		return catalogv1.MatchSource_MATCH_SOURCE_UNSPECIFIED
	}
}

// dimensionsFromProto converts measured size; a nil message is no measurement.
func dimensionsFromProto(d *commonv1.Dimensions) domain.Dimensions {
	if d == nil {
		return domain.Dimensions{}
	}
	return domain.Dimensions{
		LengthMM: d.LengthMm,
		WidthMM:  d.WidthMm,
		HeightMM: d.HeightMm,
		WeightG:  d.WeightG,
	}
}

// dimensionsToProto converts measured size back to the wire type, returning nil
// when nothing was measured so the field stays absent.
func dimensionsToProto(d domain.Dimensions) *commonv1.Dimensions {
	if d.LengthMM == nil && d.WidthMM == nil && d.HeightMM == nil && d.WeightG == nil {
		return nil
	}
	return &commonv1.Dimensions{
		LengthMm: d.LengthMM,
		WidthMm:  d.WidthMM,
		HeightMm: d.HeightMM,
		WeightG:  d.WeightG,
	}
}

// packagingFromProto converts the freight flags.
func packagingFromProto(p *catalogv1.PackagingMeta) domain.Packaging {
	if p == nil {
		return domain.Packaging{}
	}
	return domain.Packaging{
		Fragile:               p.GetFragile(),
		Oversized:             p.GetOversized(),
		RequiresCustomCrating: p.GetRequiresCustomCrating(),
		Packed:                dimensionsFromProto(p.GetPackedDimensions()),
	}
}

// packagingToProto converts the freight flags back to the wire type.
func packagingToProto(p domain.Packaging) *catalogv1.PackagingMeta {
	return &catalogv1.PackagingMeta{
		Fragile:               p.Fragile,
		Oversized:             p.Oversized,
		RequiresCustomCrating: p.RequiresCustomCrating,
		PackedDimensions:      dimensionsToProto(p.Packed),
	}
}

// translationFromProto converts one piece of copy.
func translationFromProto(t *catalogv1.ListingTranslation) domain.ListingTranslation {
	if t == nil {
		return domain.ListingTranslation{}
	}
	return domain.ListingTranslation{
		Language:         languageNameFromProto(t.GetLanguage()),
		Title:            t.GetTitle(),
		Description:      t.GetDescription(),
		Highlights:       t.GetHighlights(),
		MachineGenerated: t.GetMachineGenerated(),
		EditedBy:         t.EditedBy,
	}
}

// translationsFromProto converts a set of copy.
func translationsFromProto(in []*catalogv1.ListingTranslation) []domain.ListingTranslation {
	out := make([]domain.ListingTranslation, 0, len(in))
	for _, t := range in {
		out = append(out, translationFromProto(t))
	}
	return out
}

// translationToProto converts one piece of copy back to the wire type.
func translationToProto(t domain.ListingTranslation) *catalogv1.ListingTranslation {
	return &catalogv1.ListingTranslation{
		Language:         languageNameToProto(t.Language),
		Title:            t.Title,
		Description:      t.Description,
		Highlights:       t.Highlights,
		MachineGenerated: t.MachineGenerated,
		EditedBy:         t.EditedBy,
	}
}

// translationsToProto converts a set of copy, optionally narrowed to one
// language for a buyer surface that only renders theirs.
func translationsToProto(in []domain.ListingTranslation, only string) []*catalogv1.ListingTranslation {
	out := make([]*catalogv1.ListingTranslation, 0, len(in))
	for _, t := range in {
		if only != "" && t.Language != only {
			continue
		}
		out = append(out, translationToProto(t))
	}
	return out
}

// productToProto converts a domain product to the wire type. Media refs are left
// to the caller that has them: a product read does not hydrate the bucket.
func productToProto(p domain.Product) *catalogv1.Product {
	out := &catalogv1.Product{
		Id:           p.ID.String(),
		ArtisanId:    p.ArtisanID.String(),
		CraftId:      p.CraftID.String(),
		WorkingTitle: p.WorkingTitle,
		Dimensions:   dimensionsToProto(p.Dimensions),
		Materials:    p.Materials,
		Techniques:   p.Techniques,
		Colours:      p.Colours,
		Motifs:       p.Motifs,
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(p.CreatedAt),
			UpdatedAt: timestamppb.New(p.UpdatedAt),
			CreatedBy: p.CreatedBy,
		},
	}
	return out
}

// listingToProto converts a domain listing to the wire type. language narrows
// the copy when the caller asked for one; "" returns every translation.
func listingToProto(l domain.Listing, language string) *catalogv1.Listing {
	out := &catalogv1.Listing{
		Id:        l.ID.String(),
		ProductId: l.ProductID.String(),
		ArtisanId: l.ArtisanID.String(),
		Type:      listingTypeToProto(l.Type),
		State:     listingStateToProto(l.State),
		Price: &commonv1.Money{
			AmountPaise:  l.PricePaise,
			CurrencyCode: currencyOr(l.CurrencyCode),
		},
		StockQuantity:    l.StockQuantity,
		MinOrderQuantity: l.MinOrderQuantity,
		Packaging:        packagingToProto(l.Packaging),
		Translations:     translationsToProto(l.Translations, language),
		GiCertified:      l.GICertified,
		SuspensionReason: l.SuspensionReason,
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(l.CreatedAt),
			UpdatedAt: timestamppb.New(l.UpdatedAt),
			CreatedBy: l.CreatedBy,
		},
	}
	if l.Type == domain.ListingMadeToOrder {
		out.MadeToOrderTerms = &catalogv1.MadeToOrderTerms{
			LeadTimeDays:     int32OrZero(l.LeadTimeDays),
			CapacityPerMonth: int32OrZero(l.CapacityPerMonth),
			AcceptingOrders:  l.AcceptingOrders,
			AdvancePct:       int32OrZero(l.AdvancePct),
		}
	}
	if l.ProvenanceID != nil {
		id := l.ProvenanceID.String()
		out.ProvenanceId = &id
	}
	if l.PublishedAt != nil {
		out.PublishedAt = timestamppb.New(*l.PublishedAt)
	}
	return out
}

// craftToProto converts one ontology node to the wire type.
func craftToProto(c domain.Craft) *catalogv1.Craft {
	out := &catalogv1.Craft{
		Id:               c.ID.String(),
		Code:             c.Code,
		DisplayName:      c.DisplayName,
		GiRegistrationNo: c.GIRegistrationNo,
		Techniques:       c.Techniques,
		Materials:        c.Materials,
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(c.CreatedAt),
			UpdatedAt: timestamppb.New(c.UpdatedAt),
			CreatedBy: "ontology-seed",
		},
	}
	if c.ParentCraftID != nil {
		id := c.ParentCraftID.String()
		out.ParentCraftId = &id
	}
	return out
}

// craftMatchToProto converts one resolved mention, span included.
func craftMatchToProto(m domain.CraftMatch) *catalogv1.CraftMatch {
	return &catalogv1.CraftMatch{
		CraftId:     m.CraftID.String(),
		CraftCode:   m.Code,
		DisplayName: m.DisplayName,
		Alias:       m.Alias,
		MatchedText: m.MatchedText,
		StartOffset: int32(m.Start),
		EndOffset:   int32(m.End),
		Script:      m.Script,
		Language:    languageNameToProto(m.Language),
		Score:       m.Score,
		Source:      matchSourceToProto(m.Source),
	}
}

// attributeToProto converts one stored attribute.
func attributeToProto(a domain.ListingAttribute) *catalogv1.ListingAttribute {
	return &catalogv1.ListingAttribute{
		Name:       a.Name,
		Value:      a.Value,
		Confidence: a.Confidence,
		Source:     attributeSourceToProto(a.Source),
	}
}

// attributesToProto converts a stored set.
func attributesToProto(in []domain.ListingAttribute) []*catalogv1.ListingAttribute {
	out := make([]*catalogv1.ListingAttribute, 0, len(in))
	for _, a := range in {
		out = append(out, attributeToProto(a))
	}
	return out
}

// listingMediaToProto converts one attached asset.
func listingMediaToProto(m domain.ListingMedia) *catalogv1.ListingMediaItem {
	return &catalogv1.ListingMediaItem{
		MediaId: m.MediaID.String(),
		Ordinal: m.Ordinal,
		Role:    mediaRoleToProto(m.Role),
		Kind:    mediaKindToProto(m.Kind),
	}
}

// listingMediaSetToProto converts an attached set.
func listingMediaSetToProto(in []domain.ListingMedia) []*catalogv1.ListingMediaItem {
	out := make([]*catalogv1.ListingMediaItem, 0, len(in))
	for _, m := range in {
		out = append(out, listingMediaToProto(m))
	}
	return out
}

// mediaRefIDs pulls the ids out of a set of media refs, which is all the
// catalog needs: the rows themselves are already in the database.
func mediaRefIDs(refs []*commonv1.MediaRef) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(refs))
	for _, ref := range refs {
		id, err := parseUUID("media.id", ref.GetId())
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// currencyOr defaults a blank stored currency to INR.
func currencyOr(code string) string {
	if code == "" {
		return defaultCurrency
	}
	return code
}

// int32OrZero dereferences an optional count for a non-optional wire field.
func int32OrZero(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}
