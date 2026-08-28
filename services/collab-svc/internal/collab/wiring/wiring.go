// services/collab-svc/internal/collab/wiring/wiring.go

// Package wiring binds collab-svc's concrete repository to the interfaces
// the service layer declares, so the service package itself never imports
// the repository.
package wiring

import (
	"context"

	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/repo"
	"github.com/segfaultsyndicate/kalakriti/services/collab-svc/internal/collab/service"
)

// Store binds a repository to service.Store. Every read method is promoted
// from the embedded *repo.Repo; only InTx needs adapting, because Go has no
// variance on function parameters: *repo.Repo.InTx hands its callback a
// *repo.Tx, while service.Store.InTx is declared over the service.Tx
// interface.
type Store struct {
	*repo.Repo
}

// NewStore wraps a repository as the fulfilment saga's persistence port.
func NewStore(r *repo.Repo) Store { return Store{Repo: r} }

// Store satisfies the fulfilment saga's persistence contract.
var _ service.Store = Store{}

// InTx adapts the repository's concrete transaction type to service.Tx.
func (s Store) InTx(ctx context.Context, fn func(ctx context.Context, tx service.Tx) error) error {
	return s.Repo.InTx(ctx, func(ctx context.Context, tx *repo.Tx) error {
		return fn(ctx, tx)
	})
}
