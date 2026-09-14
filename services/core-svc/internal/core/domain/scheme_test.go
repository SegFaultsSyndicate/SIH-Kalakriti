// services/core-svc/internal/core/domain/scheme_test.go
package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMatchScheme_AllCriteriaMetNoManualChecks_MayQualify(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "test_scheme"}
	criteria := []SchemeCriterion{{Type: CriterionMinYearsExperience, IntValue: ptrInt64(2)}}
	facts := ArtisanMatchFacts{YearsOfExperience: ptrInt32(5)}

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, MayQualify, match.Status)
}

func TestMatchScheme_UnmetCriterion_Unlikely(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "test_scheme"}
	criteria := []SchemeCriterion{{Type: CriterionMinYearsExperience, IntValue: ptrInt64(10)}}
	facts := ArtisanMatchFacts{YearsOfExperience: ptrInt32(2)}

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, Unlikely, match.Status)
	require.Contains(t, match.UnmetCriteriaLabels, "scheme.criterion.min_years_experience")
}

func TestMatchScheme_AllCriteriaMetWithOutstandingManualChecks_CheckRequired(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "test_scheme"}
	manualChecks := []SchemeManualCheck{{I18nKey: ptrString("scheme.test.check.income")}}
	facts := ArtisanMatchFacts{}

	match := MatchScheme(scheme, nil, manualChecks, facts)
	require.Equal(t, CheckRequired, match.Status)
	require.Len(t, match.ManualChecks, 1)
}

func TestMatchScheme_NilSocialCategory_TreatsSocialCategoryCriterionAsUnmet_NoError(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "stand_up_india"}
	criteria := []SchemeCriterion{{Type: CriterionSocialCategory, StringValues: []string{"SC", "ST"}}}
	facts := ArtisanMatchFacts{SocialCategory: nil} // PREFER_NOT_TO_SAY or never answered

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, Unlikely, match.Status)
}

func TestMatchScheme_NegatedCriterion_MatchesWhenArtisanDoesNotHaveIt(t *testing.T) {
	scheme := Scheme{ID: uuid.New(), Code: "handicrafts_pehchan_id"}
	criteria := []SchemeCriterion{{Type: CriterionHasPehchanID, Negate: true}}
	facts := ArtisanMatchFacts{HasPehchanID: false}

	match := MatchScheme(scheme, criteria, nil, facts)
	require.Equal(t, MayQualify, match.Status)

	factsWithID := ArtisanMatchFacts{HasPehchanID: true}
	matchWithID := MatchScheme(scheme, criteria, nil, factsWithID)
	require.Equal(t, Unlikely, matchWithID.Status)
}

func ptrInt64(v int64) *int64   { return &v }
func ptrInt32(v int32) *int32   { return &v }
func ptrString(v string) *string { return &v }
