package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

func TestRegisterArtisanInput_SocialCategoryValidation(t *testing.T) {
	validInput := func() domain.RegisterArtisanInput {
		return domain.RegisterArtisanInput{
			DisplayName: "Ramesh Sharma",
			PhoneE164:   "+919876543210",
			CraftIDs:    []uuid.UUID{uuid.New()},
			Languages:   []string{"HINDI"},
			Region: domain.Region{
				StateCode: "RJ",
			},
		}
	}

	t.Run("valid categories", func(t *testing.T) {
		validCats := []string{"GENERAL", "OBC", "SC", "ST", "EWS", "PREFER_NOT_TO_SAY"}
		for _, cat := range validCats {
			c := cat
			in := validInput()
			in.SocialCategory = &c
			require.NoError(t, in.Validate(), "category %s should be valid", cat)
		}
	})

	t.Run("nil category is valid", func(t *testing.T) {
		in := validInput()
		in.SocialCategory = nil
		require.NoError(t, in.Validate())
	})

	t.Run("invalid category fails", func(t *testing.T) {
		bad := "INVALID_CATEGORY"
		in := validInput()
		in.SocialCategory = &bad
		require.Error(t, in.Validate())
	})
}
