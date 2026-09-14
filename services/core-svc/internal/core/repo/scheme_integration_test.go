// services/core-svc/internal/core/repo/scheme_integration_test.go
//go:build integration

package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepo_ListActiveSchemes_ReturnsSeeded8(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(ctx, t)
	r := New(pool)

	schemes, err := r.ListActiveSchemes(ctx)
	require.NoError(t, err)
	require.Len(t, schemes, 8)
}
