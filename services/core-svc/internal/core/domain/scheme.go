// services/core-svc/internal/core/domain/scheme.go

package domain

import (
	"github.com/google/uuid"
)

// MatchStatus is the outcome of matching one artisan against one scheme.
// This is the entire vocabulary this feature is allowed to show an artisan
// -- there is deliberately no fourth "Eligible" status. See the spec's
// "one rule everything else follows".
type MatchStatus string

const (
	MayQualify    MatchStatus = "MAY_QUALIFY"
	CheckRequired MatchStatus = "CHECK_REQUIRED"
	Unlikely      MatchStatus = "UNLIKELY"
)

// SchemeCriterionType is one machine-checkable fact about an artisan.
type SchemeCriterionType string

const (
	CriterionSocialCategory     SchemeCriterionType = "SOCIAL_CATEGORY"
	CriterionStateCode          SchemeCriterionType = "STATE_CODE"
	CriterionCraftID            SchemeCriterionType = "CRAFT_ID"
	CriterionMinYearsExperience SchemeCriterionType = "MIN_YEARS_EXPERIENCE"
	CriterionHasPehchanID       SchemeCriterionType = "HAS_PEHCHAN_ID"
	CriterionHasPMVishwakarmaID SchemeCriterionType = "HAS_PM_VISHWAKARMA_ID"
	CriterionClusterMember      SchemeCriterionType = "CLUSTER_MEMBER"
	CriterionSHGMember          SchemeCriterionType = "SHG_MEMBER"
)

// Scheme is one catalog entry.
type Scheme struct {
	ID             uuid.UUID
	Code           string
	Authority      string
	Ministry       string
	OfficialURL    string
	StateCode      *string
	NameI18nKey    *string
	NameText       *string
	SummaryI18nKey *string
	SummaryText    *string
	SortOrder      int32
}

// SchemeCriterion is one machine-checkable rule attached to a scheme. All of
// a scheme's criteria must pass (AND) for the scheme to be MAY_QUALIFY or
// CHECK_REQUIRED rather than UNLIKELY.
type SchemeCriterion struct {
	Type         SchemeCriterionType
	StringValues []string
	IntValue     *int64
	Negate       bool
}

// SchemeManualCheck is one not-machine-checkable item the artisan must read
// and self-confirm. Never evaluated automatically.
type SchemeManualCheck struct {
	I18nKey   *string
	CheckText *string
}

// ArtisanMatchFacts is everything MatchScheme needs to know about one
// artisan. A nil pointer field means "unknown" and any criterion depending
// on it is treated as unmet, never as an error and never as a pass.
type ArtisanMatchFacts struct {
	StateCode          string
	YearsOfExperience  *int32
	SocialCategory     *string // one of the social_category enum values, or nil
	HasPehchanID       bool
	HasPMVishwakarmaID bool
	IsClusterMember    bool
	IsSHGMember        bool
	CraftIDs           []uuid.UUID
}

// SchemeMatch is one scheme's result for one artisan.
type SchemeMatch struct {
	Scheme                Scheme
	Status                MatchStatus
	MatchedCriteriaLabels []string
	UnmetCriteriaLabels   []string
	ManualChecks          []SchemeManualCheck
}

// MatchScheme is a pure function: no I/O, no randomness, no clock. Every
// criterion must pass for the scheme to be anything but UNLIKELY; if every
// criterion passes and there is at least one manual check, the status is
// CHECK_REQUIRED; if every criterion passes and there are no manual checks,
// it is MAY_QUALIFY. A criterion whose underlying fact is unknown (a nil
// pointer in facts) is treated as unmet, matching the spec's requirement
// that PREFER_NOT_TO_SAY / never-answered social_category must not error
// and must not silently pass.
func MatchScheme(scheme Scheme, criteria []SchemeCriterion, manualChecks []SchemeManualCheck, facts ArtisanMatchFacts) SchemeMatch {
	match := SchemeMatch{Scheme: scheme, ManualChecks: manualChecks}

	allMet := true
	for _, c := range criteria {
		met := criterionMet(c, facts)
		label := "scheme.criterion." + criterionLabelSuffix(c.Type)
		if met {
			match.MatchedCriteriaLabels = append(match.MatchedCriteriaLabels, label)
		} else {
			match.UnmetCriteriaLabels = append(match.UnmetCriteriaLabels, label)
			allMet = false
		}
	}

	switch {
	case !allMet:
		match.Status = Unlikely
	case len(manualChecks) > 0:
		match.Status = CheckRequired
	default:
		match.Status = MayQualify
	}
	return match
}

func criterionMet(c SchemeCriterion, facts ArtisanMatchFacts) bool {
	var raw bool
	switch c.Type {
	case CriterionSocialCategory:
		raw = facts.SocialCategory != nil && containsString(c.StringValues, *facts.SocialCategory)
	case CriterionStateCode:
		raw = containsString(c.StringValues, facts.StateCode)
	case CriterionCraftID:
		raw = anyUUIDInStrings(facts.CraftIDs, c.StringValues)
	case CriterionMinYearsExperience:
		raw = c.IntValue != nil && facts.YearsOfExperience != nil && int64(*facts.YearsOfExperience) >= *c.IntValue
	case CriterionHasPehchanID:
		raw = facts.HasPehchanID
	case CriterionHasPMVishwakarmaID:
		raw = facts.HasPMVishwakarmaID
	case CriterionClusterMember:
		raw = facts.IsClusterMember
	case CriterionSHGMember:
		raw = facts.IsSHGMember
	default:
		raw = false
	}
	if c.Negate {
		return !raw
	}
	return raw
}

func criterionLabelSuffix(t SchemeCriterionType) string {
	switch t {
	case CriterionSocialCategory:
		return "social_category"
	case CriterionStateCode:
		return "state_code"
	case CriterionCraftID:
		return "craft"
	case CriterionMinYearsExperience:
		return "min_years_experience"
	case CriterionHasPehchanID:
		return "has_pehchan_id"
	case CriterionHasPMVishwakarmaID:
		return "has_pm_vishwakarma_id"
	case CriterionClusterMember:
		return "cluster_member"
	case CriterionSHGMember:
		return "shg_member"
	default:
		return "unknown"
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func anyUUIDInStrings(ids []uuid.UUID, targets []string) bool {
	for _, id := range ids {
		if containsString(targets, id.String()) {
			return true
		}
	}
	return false
}
