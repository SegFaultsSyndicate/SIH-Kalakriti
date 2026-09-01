// services/core-svc/internal/core/handler/convert.go
package handler

import (
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	catalogv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/catalog/v1"
	commonv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/common/v1"
	identityv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/identity/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// parseUUID converts a request field to a UUID, reporting a field-named
// ErrInvalidInput rather than a bare parse error.
func parseUUID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s %q is not a valid uuid: %w", field, value, pkgdomain.ErrInvalidInput)
	}
	return id, nil
}

// parseOptionalUUID converts an optional request field to a UUID pointer.
func parseOptionalUUID(field string, value *string) (*uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	id, err := parseUUID(field, *value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// parseUUIDs converts a repeated request field to UUIDs.
func parseUUIDs(field string, values []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		id, err := parseUUID(field, v)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// regionFromProto converts a common.v1.GeoRegion into the domain type. A nil
// region yields a zero Region, which domain validation then rejects where
// state_code is required.
func regionFromProto(r *commonv1.GeoRegion) domain.Region {
	if r == nil {
		return domain.Region{}
	}
	return domain.Region{
		StateCode: r.GetStateCode(),
		District:  r.District,
		Block:     r.Block,
		Village:   r.Village,
		Pincode:   r.Pincode,
	}
}

// regionToProto converts the domain type back to the wire type.
func regionToProto(r domain.Region) *commonv1.GeoRegion {
	return &commonv1.GeoRegion{
		StateCode: r.StateCode,
		District:  r.District,
		Block:     r.Block,
		Village:   r.Village,
		Pincode:   r.Pincode,
	}
}

// languagesFromProto renders proto language enums as the plain names the
// language_code Postgres enum stores, dropping LANGUAGE_UNSPECIFIED.
func languagesFromProto(langs []commonv1.Language) []string {
	out := make([]string, 0, len(langs))
	for _, l := range langs {
		if l == commonv1.Language_LANGUAGE_UNSPECIFIED {
			continue
		}
		out = append(out, trimEnumPrefix(l.String(), "LANGUAGE_"))
	}
	return out
}

// languagesToProto converts stored language names back to proto enums. An
// unrecognised stored value maps to UNSPECIFIED rather than failing the read.
func languagesToProto(langs []string) []commonv1.Language {
	out := make([]commonv1.Language, 0, len(langs))
	for _, l := range langs {
		if v, ok := commonv1.Language_value["LANGUAGE_"+l]; ok {
			out = append(out, commonv1.Language(v))
			continue
		}
		out = append(out, commonv1.Language_LANGUAGE_UNSPECIFIED)
	}
	return out
}

// trimEnumPrefix strips a proto enum's value prefix, e.g. LANGUAGE_HINDI -> HINDI.
func trimEnumPrefix(value, prefix string) string {
	if len(value) > len(prefix) && value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return value
}

// clusterRoleFromProto maps the wire role onto the domain role, defaulting an
// unspecified role to MEMBER.
func clusterRoleFromProto(r identityv1.ClusterMemberRole) domain.ClusterMemberRole {
	switch r {
	case identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_COORDINATOR:
		return domain.ClusterRoleCoordinator
	case identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_MASTER:
		return domain.ClusterRoleMaster
	default:
		return domain.ClusterRoleMember
	}
}

// clusterRoleToProto maps the domain role onto the wire role.
func clusterRoleToProto(r domain.ClusterMemberRole) identityv1.ClusterMemberRole {
	switch r {
	case domain.ClusterRoleCoordinator:
		return identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_COORDINATOR
	case domain.ClusterRoleMaster:
		return identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_MASTER
	case domain.ClusterRoleMember:
		return identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_MEMBER
	default:
		return identityv1.ClusterMemberRole_CLUSTER_MEMBER_ROLE_UNSPECIFIED
	}
}

// artisanToProto converts a domain artisan to the catalog.v1 wire type.
func artisanToProto(a domain.Artisan) *catalogv1.Artisan {
	out := &catalogv1.Artisan{
		Id:                a.ID.String(),
		UserId:            a.UserID,
		DisplayName:       a.DisplayName,
		PhoneE164:         a.PhoneE164,
		CraftIds:          uuidsToStrings(a.CraftIDs),
		Languages:         languagesToProto(a.Languages),
		Region:            regionToProto(a.Region),
		YearsOfExperience: a.YearsOfExperience,
		Bio:               a.Bio,
		Verified:          a.Verified,
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(a.CreatedAt),
			UpdatedAt: timestamppb.New(a.UpdatedAt),
			CreatedBy: a.CreatedBy,
		},
	}
	if a.PrimaryClusterID != nil {
		id := a.PrimaryClusterID.String()
		out.ClusterId = &id
	}
	return out
}

// clusterToProto converts a domain cluster to the catalog.v1 wire type.
func clusterToProto(c domain.Cluster) *catalogv1.Cluster {
	return &catalogv1.Cluster{
		Id:                   c.ID.String(),
		Name:                 c.Name,
		Region:               regionToProto(c.Region),
		ArtisanCount:         c.ArtisanCount,
		CoordinatorPhoneE164: c.CoordinatorPhoneE164,
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(c.CreatedAt),
			UpdatedAt: timestamppb.New(c.UpdatedAt),
			CreatedBy: "system",
		},
	}
}

// shgToProto converts a domain SHG to the catalog.v1 wire type.
func shgToProto(g domain.SelfHelpGroup, members []domain.SHGMember) *catalogv1.SelfHelpGroup {
	out := &catalogv1.SelfHelpGroup{
		Id:             g.ID.String(),
		Name:           g.Name,
		RegistrationNo: g.RegistrationNo,
		Audit: &commonv1.AuditMeta{
			CreatedAt: timestamppb.New(g.CreatedAt),
			UpdatedAt: timestamppb.New(g.UpdatedAt),
			CreatedBy: "system",
		},
	}
	if g.ClusterID != nil {
		id := g.ClusterID.String()
		out.ClusterId = &id
	}
	if g.SignatoryArtisanID != nil {
		id := g.SignatoryArtisanID.String()
		out.SignatoryArtisanId = &id
	}
	out.MemberArtisanIds = make([]string, 0, len(members))
	for _, m := range members {
		out.MemberArtisanIds = append(out.MemberArtisanIds, m.ArtisanID.String())
	}
	return out
}

// clusterMemberToProto converts a domain membership to the wire type.
func clusterMemberToProto(m domain.ClusterMember) *identityv1.ClusterMember {
	return &identityv1.ClusterMember{
		ClusterId:   m.ClusterID.String(),
		ArtisanId:   m.ArtisanID.String(),
		DisplayName: m.DisplayName,
		Role:        clusterRoleToProto(m.Role),
		JoinedAt:    timestamppb.New(m.JoinedAt),
	}
}

// shgMemberToProto converts a domain SHG membership to the wire type.
func shgMemberToProto(m domain.SHGMember) *identityv1.SelfHelpGroupMember {
	return &identityv1.SelfHelpGroupMember{
		SelfHelpGroupId: m.SHGID.String(),
		ArtisanId:       m.ArtisanID.String(),
		DisplayName:     m.DisplayName,
		SharePct:        m.SharePct,
		JoinedAt:        timestamppb.New(m.JoinedAt),
	}
}

// shgMembersToProto converts a roster.
func shgMembersToProto(members []domain.SHGMember) []*identityv1.SelfHelpGroupMember {
	out := make([]*identityv1.SelfHelpGroupMember, 0, len(members))
	for _, m := range members {
		out = append(out, shgMemberToProto(m))
	}
	return out
}

// shgSharesFromProto converts wire share inputs to the domain type.
func shgSharesFromProto(in []*identityv1.SelfHelpGroupMemberInput) ([]domain.SHGMemberShare, error) {
	out := make([]domain.SHGMemberShare, 0, len(in))
	for _, m := range in {
		id, err := parseUUID("members.artisan_id", m.GetArtisanId())
		if err != nil {
			return nil, err
		}
		out = append(out, domain.SHGMemberShare{ArtisanID: id, SharePct: m.GetSharePct()})
	}
	return out, nil
}

// pageFromProto converts a common.v1.PageRequest into the domain page.
func pageFromProto(p *commonv1.PageRequest) (domain.Page, error) {
	if p == nil {
		return domain.Page{}, nil
	}
	page := domain.Page{Size: p.GetPageSize()}
	if c := p.GetPageToken(); c != "" {
		id, err := parseUUID("page.cursor", c)
		if err != nil {
			return domain.Page{}, err
		}
		page.Cursor = &id
	}
	return page, nil
}

// nextPage builds the cursor for the following page. An empty cursor means the
// caller has reached the end.
func nextPage(lastID string, returned int, requested int32) *commonv1.PageResponse {
	if returned == 0 || int32(returned) < requested {
		return &commonv1.PageResponse{NextPageToken: ""}
	}
	return &commonv1.PageResponse{NextPageToken: lastID}
}

// uuidsToStrings renders ids for a wire message.
func uuidsToStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

// tokensToProto converts an issued token pair to the wire type.
func tokensToProto(t auth.TokenPair) *identityv1.TokenPair {
	return &identityv1.TokenPair{
		AccessToken:      t.AccessToken,
		RefreshToken:     t.RefreshToken,
		AccessExpiresAt:  timestamppb.New(t.AccessExpiresAt),
		RefreshExpiresAt: timestamppb.New(t.RefreshExpiresAt),
		TokenType:        "Bearer",
	}
}
