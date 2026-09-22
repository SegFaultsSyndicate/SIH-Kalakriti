// services/core-svc/internal/core/handler/scheme.go

package handler

import (
	"context"

	"github.com/google/uuid"

	schemesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/schemes/v1"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/service"
)

// Schemes implements schemes.v1.SchemeService.
type Schemes struct {
	schemesv1.UnimplementedSchemeServiceServer
	svc *service.Schemes
}

func NewSchemes(svc *service.Schemes) *Schemes { return &Schemes{svc: svc} }

func (h *Schemes) ListSchemes(ctx context.Context, req *schemesv1.ListSchemesRequest) (*schemesv1.ListSchemesResponse, error) {
	schemes, err := h.svc.ListSchemes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*schemesv1.GovernmentScheme, len(schemes))
	for i, s := range schemes {
		out[i] = toProtoScheme(s)
	}
	return &schemesv1.ListSchemesResponse{Schemes: out}, nil
}

func (h *Schemes) MatchSchemes(ctx context.Context, req *schemesv1.MatchSchemesRequest) (*schemesv1.MatchSchemesResponse, error) {
	matches, err := h.svc.MatchSchemes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*schemesv1.SchemeMatch, len(matches))
	for i, m := range matches {
		checks := make([]*schemesv1.SchemeManualCheckProto, len(m.ManualChecks))
		for j, c := range m.ManualChecks {
			checks[j] = &schemesv1.SchemeManualCheckProto{I18NKey: c.I18nKey, CheckText: c.CheckText}
		}
		out[i] = &schemesv1.SchemeMatch{
			Scheme: toProtoScheme(m.Scheme), Status: toProtoStatus(m.Status),
			MatchedCriteriaLabels: m.MatchedCriteriaLabels, UnmetCriteriaLabels: m.UnmetCriteriaLabels,
			ManualChecks: checks,
		}
	}
	return &schemesv1.MatchSchemesResponse{Matches: out}, nil
}

func (h *Schemes) UpsertScheme(ctx context.Context, req *schemesv1.UpsertSchemeRequest) (*schemesv1.UpsertSchemeResponse, error) {
	var id uuid.UUID
	if req.Id != nil && *req.Id != "" {
		parsed, err := uuid.Parse(*req.Id)
		if err != nil {
			return nil, err
		}
		id = parsed
	}
	criteria := make([]domain.SchemeCriterion, len(req.Criteria))
	for i, c := range req.Criteria {
		criteria[i] = domain.SchemeCriterion{
			Type: fromProtoCriterionType(c.Type), StringValues: c.StringValues, IntValue: c.IntValue, Negate: c.Negate,
		}
	}
	manualChecks := make([]domain.SchemeManualCheck, len(req.ManualChecks))
	for i, m := range req.ManualChecks {
		manualChecks[i] = domain.SchemeManualCheck{I18nKey: m.I18NKey, CheckText: m.CheckText}
	}
	scheme, err := h.svc.UpsertScheme(ctx, domain.Scheme{
		ID: id, Code: req.Code, Authority: fromProtoAuthority(req.Authority), Ministry: req.Ministry,
		OfficialURL: req.OfficialUrl, StateCode: req.StateCode, NameI18nKey: req.NameI18NKey,
		NameText: req.NameText, SummaryI18nKey: req.SummaryI18NKey, SummaryText: req.SummaryText,
		SortOrder: req.SortOrder,
	}, criteria, manualChecks)
	if err != nil {
		return nil, err
	}
	return &schemesv1.UpsertSchemeResponse{Scheme: toProtoScheme(scheme)}, nil
}

func (h *Schemes) DeleteScheme(ctx context.Context, req *schemesv1.DeleteSchemeRequest) (*schemesv1.DeleteSchemeResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}
	deleted, err := h.svc.DeleteScheme(ctx, id)
	if err != nil {
		return nil, err
	}
	return &schemesv1.DeleteSchemeResponse{Deleted: deleted}, nil
}

func toProtoScheme(s domain.Scheme) *schemesv1.GovernmentScheme {
	authority := schemesv1.SchemeAuthority_SCHEME_AUTHORITY_CENTRAL
	if s.Authority == "STATE" {
		authority = schemesv1.SchemeAuthority_SCHEME_AUTHORITY_STATE
	}
	return &schemesv1.GovernmentScheme{
		Id: s.ID.String(), Code: s.Code, Authority: authority, Ministry: s.Ministry, OfficialUrl: s.OfficialURL,
		StateCode: s.StateCode, NameI18NKey: s.NameI18nKey, NameText: s.NameText,
		SummaryI18NKey: s.SummaryI18nKey, SummaryText: s.SummaryText, SortOrder: s.SortOrder,
	}
}

func toProtoStatus(s domain.MatchStatus) schemesv1.MatchStatus {
	switch s {
	case domain.MayQualify:
		return schemesv1.MatchStatus_MATCH_STATUS_MAY_QUALIFY
	case domain.CheckRequired:
		return schemesv1.MatchStatus_MATCH_STATUS_CHECK_REQUIRED
	case domain.Unlikely:
		return schemesv1.MatchStatus_MATCH_STATUS_UNLIKELY
	default:
		return schemesv1.MatchStatus_MATCH_STATUS_UNSPECIFIED
	}
}

func fromProtoAuthority(a schemesv1.SchemeAuthority) string {
	if a == schemesv1.SchemeAuthority_SCHEME_AUTHORITY_STATE {
		return "STATE"
	}
	return "CENTRAL"
}

func fromProtoCriterionType(t schemesv1.SchemeCriterionType) domain.SchemeCriterionType {
	switch t {
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_SOCIAL_CATEGORY:
		return domain.CriterionSocialCategory
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_STATE_CODE:
		return domain.CriterionStateCode
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_CRAFT_ID:
		return domain.CriterionCraftID
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_MIN_YEARS_EXPERIENCE:
		return domain.CriterionMinYearsExperience
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_HAS_PEHCHAN_ID:
		return domain.CriterionHasPehchanID
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_HAS_PM_VISHWAKARMA_ID:
		return domain.CriterionHasPMVishwakarmaID
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_CLUSTER_MEMBER:
		return domain.CriterionClusterMember
	case schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_SHG_MEMBER:
		return domain.CriterionSHGMember
	default:
		return ""
	}
}
