// services/core-svc/internal/core/domain/badge_test.go
package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGrantBadgeInput_Validate(t *testing.T) {
	valid := GrantBadgeInput{ArtisanID: uuid.New(), BadgeID: uuid.New(), GrantedBy: "admin-1"}
	require.NoError(t, valid.Validate())

	missing := GrantBadgeInput{BadgeID: uuid.New(), GrantedBy: "admin-1"}
	require.Error(t, missing.Validate())
}

func TestRevokeBadgeInput_Validate(t *testing.T) {
	valid := RevokeBadgeInput{ArtisanID: uuid.New(), BadgeID: uuid.New(), RevokedBy: "admin-1", Reason: "rule violation"}
	require.NoError(t, valid.Validate())

	missing := RevokeBadgeInput{BadgeID: uuid.New(), RevokedBy: "admin-1", Reason: "rule violation"}
	require.Error(t, missing.Validate())

	missingReason := RevokeBadgeInput{ArtisanID: uuid.New(), BadgeID: uuid.New(), RevokedBy: "admin-1"}
	require.Error(t, missingReason.Validate())
}
