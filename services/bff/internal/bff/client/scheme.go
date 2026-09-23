// services/bff/internal/bff/client/scheme.go
package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	schemesv1 "github.com/ZoroNewbie00/kalakriti/pkg/pb/schemes/v1"
)

type Schemes struct {
	schemes schemesv1.SchemeServiceClient
}

func NewSchemes(conn grpc.ClientConnInterface) *Schemes {
	return &Schemes{schemes: schemesv1.NewSchemeServiceClient(conn)}
}

func (s *Schemes) ListSchemes(ctx context.Context) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	resp, err := s.schemes.ListSchemes(ctx, &schemesv1.ListSchemesRequest{})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := make([]map[string]any, len(resp.Schemes))
	for i, scheme := range resp.Schemes {
		out[i] = schemeToMap(scheme)
	}
	return out, nil
}

func (s *Schemes) MatchSchemes(ctx context.Context) ([]map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	resp, err := s.schemes.MatchSchemes(ctx, &schemesv1.MatchSchemesRequest{})
	if err != nil {
		return nil, grpcErr(err)
	}
	out := make([]map[string]any, len(resp.Matches))
	for i, m := range resp.Matches {
		checks := make([]map[string]any, len(m.ManualChecks))
		for j, c := range m.ManualChecks {
			checks[j] = map[string]any{"i18n_key": c.I18NKey, "check_text": c.CheckText}
		}
		status := "UNSPECIFIED"
		switch m.Status {
		case schemesv1.MatchStatus_MATCH_STATUS_MAY_QUALIFY:
			status = "MAY_QUALIFY"
		case schemesv1.MatchStatus_MATCH_STATUS_CHECK_REQUIRED:
			status = "CHECK_REQUIRED"
		case schemesv1.MatchStatus_MATCH_STATUS_UNLIKELY:
			status = "UNLIKELY"
		}
		out[i] = map[string]any{
			"scheme":                  schemeToMap(m.Scheme),
			"status":                  status,
			"matched_criteria_labels": m.MatchedCriteriaLabels,
			"unmet_criteria_labels":   m.UnmetCriteriaLabels,
			"manual_checks":           checks,
		}
	}
	return out, nil
}

func (s *Schemes) UpsertScheme(ctx context.Context, fields map[string]any) (map[string]any, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	code, _ := fields["code"].(string)
	if code == "" {
		return nil, domain.InvalidInput("code: is required")
	}
	ministry, _ := fields["ministry"].(string)
	officialURL, _ := fields["official_url"].(string)
	if officialURL == "" {
		return nil, domain.InvalidInput("official_url: is required")
	}
	req := &schemesv1.UpsertSchemeRequest{Code: code, Ministry: ministry, OfficialUrl: officialURL}
	if id, ok := fields["id"].(string); ok && id != "" {
		req.Id = &id
	}
	if authStr, ok := fields["authority"].(string); ok && authStr == "STATE" {
		req.Authority = schemesv1.SchemeAuthority_SCHEME_AUTHORITY_STATE
	} else {
		req.Authority = schemesv1.SchemeAuthority_SCHEME_AUTHORITY_CENTRAL
	}
	if stateCode, ok := fields["state_code"].(string); ok && stateCode != "" {
		req.StateCode = &stateCode
	}
	if nameKey, ok := fields["name_i18n_key"].(string); ok && nameKey != "" {
		req.NameI18NKey = &nameKey
	}
	if nameText, ok := fields["name_text"].(string); ok && nameText != "" {
		req.NameText = &nameText
	}
	if summaryKey, ok := fields["summary_i18n_key"].(string); ok && summaryKey != "" {
		req.SummaryI18NKey = &summaryKey
	}
	if summaryText, ok := fields["summary_text"].(string); ok && summaryText != "" {
		req.SummaryText = &summaryText
	}
	if sortOrder, ok := fields["sort_order"].(float64); ok {
		req.SortOrder = int32(sortOrder)
	} else if sortOrderInt, ok := fields["sort_order"].(int); ok {
		req.SortOrder = int32(sortOrderInt)
	}

	if rawCriteria, ok := fields["criteria"].([]any); ok {
		for _, raw := range rawCriteria {
			if cMap, ok := raw.(map[string]any); ok {
				cProto := &schemesv1.SchemeCriterionProto{}
				if t, ok := cMap["type"].(string); ok {
					cProto.Type = toProtoCriterionType(t)
				}
				if sv, ok := cMap["string_values"].([]any); ok {
					for _, val := range sv {
						if str, ok := val.(string); ok {
							cProto.StringValues = append(cProto.StringValues, str)
						}
					}
				}
				if iv, ok := cMap["int_value"].(float64); ok {
					v := int64(iv)
					cProto.IntValue = &v
				}
				if neg, ok := cMap["negate"].(bool); ok {
					cProto.Negate = neg
				}
				req.Criteria = append(req.Criteria, cProto)
			}
		}
	}

	if rawChecks, ok := fields["manual_checks"].([]any); ok {
		for _, raw := range rawChecks {
			if chkMap, ok := raw.(map[string]any); ok {
				chkProto := &schemesv1.SchemeManualCheckProto{}
				if k, ok := chkMap["i18n_key"].(string); ok && k != "" {
					chkProto.I18NKey = &k
				}
				if t, ok := chkMap["check_text"].(string); ok && t != "" {
					chkProto.CheckText = &t
				}
				req.ManualChecks = append(req.ManualChecks, chkProto)
			}
		}
	}

	resp, err := s.schemes.UpsertScheme(ctx, req)
	if err != nil {
		return nil, grpcErr(err)
	}
	return schemeToMap(resp.Scheme), nil
}

func (s *Schemes) DeleteScheme(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := s.schemes.DeleteScheme(ctx, &schemesv1.DeleteSchemeRequest{Id: id})
	return grpcErr(err)
}

func toProtoCriterionType(t string) schemesv1.SchemeCriterionType {
	switch t {
	case "SOCIAL_CATEGORY":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_SOCIAL_CATEGORY
	case "STATE_CODE":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_STATE_CODE
	case "CRAFT_ID":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_CRAFT_ID
	case "MIN_YEARS_EXPERIENCE":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_MIN_YEARS_EXPERIENCE
	case "HAS_PEHCHAN_ID":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_HAS_PEHCHAN_ID
	case "HAS_PM_VISHWAKARMA_ID":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_HAS_PM_VISHWAKARMA_ID
	case "CLUSTER_MEMBER":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_CLUSTER_MEMBER
	case "SHG_MEMBER":
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_SHG_MEMBER
	default:
		return schemesv1.SchemeCriterionType_SCHEME_CRITERION_TYPE_UNSPECIFIED
	}
}

func schemeToMap(s *schemesv1.GovernmentScheme) map[string]any {
	authority := "CENTRAL"
	if s.Authority == schemesv1.SchemeAuthority_SCHEME_AUTHORITY_STATE {
		authority = "STATE"
	}
	out := map[string]any{
		"id": s.Id, "code": s.Code, "authority": authority, "ministry": s.Ministry,
		"official_url": s.OfficialUrl, "sort_order": s.SortOrder,
	}
	if s.StateCode != nil {
		out["state_code"] = *s.StateCode
	}
	if s.NameI18NKey != nil {
		out["name_i18n_key"] = *s.NameI18NKey
	}
	if s.NameText != nil {
		out["name_text"] = *s.NameText
	}
	if s.SummaryI18NKey != nil {
		out["summary_i18n_key"] = *s.SummaryI18NKey
	}
	if s.SummaryText != nil {
		out["summary_text"] = *s.SummaryText
	}
	return out
}
