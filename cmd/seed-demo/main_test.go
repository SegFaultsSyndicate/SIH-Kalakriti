package main

import "testing"

// TestMosjePhonesUnique guards against a regression that would make
// seedMosjeTier4 register two artisans on the same phone number (the BFF
// treats phone as the OTP login identity, so a collision would silently
// merge two "different" demo artisans into one).
func TestMosjePhonesUnique(t *testing.T) {
	const numPerGroup = 6
	seen := map[string]bool{}
	for gi := range mosjeGroups {
		for i := 0; i < numPerGroup; i++ {
			phone := formatMosjePhone(gi, i)
			if seen[phone] {
				t.Fatalf("duplicate phone %s for group %d index %d", phone, gi, i)
			}
			seen[phone] = true
		}
	}
}

// TestMosjeGroupsMeetMinCohort guards against a config edit that would drop
// a group below impact.MinCohort (5): such a group would always read "<5"
// no matter how the seeder runs.
func TestMosjeGroupsMeetMinCohort(t *testing.T) {
	numPerGroup, _ := parseMosjePerGroupEnv("")
	if numPerGroup < 5 {
		t.Fatalf("numPerGroup = %d, want >= 5 (impact.MinCohort)", numPerGroup)
	}
	if len(mosjeGroups) == 0 {
		t.Fatal("mosjeGroups is empty")
	}
}
