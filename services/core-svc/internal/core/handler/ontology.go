// services/core-svc/internal/core/handler/ontology.go
package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	pkgdomain "github.com/segfaultsyndicate/kalakriti/pkg/domain"
	catalogv1 "github.com/segfaultsyndicate/kalakriti/pkg/pb/catalog/v1"

	"github.com/segfaultsyndicate/kalakriti/services/core-svc/internal/core/service"
)

// Ontology implements catalog.v1.OntologyService.
type Ontology struct {
	catalogv1.UnimplementedOntologyServiceServer
	svc *service.Ontology
}

// NewOntology builds the ontology handler.
func NewOntology(svc *service.Ontology) *Ontology { return &Ontology{svc: svc} }

// ResolveCraftAlias links craft mentions in free text, spans included.
func (h *Ontology) ResolveCraftAlias(
	ctx context.Context,
	req *catalogv1.ResolveCraftAliasRequest,
) (*catalogv1.ResolveCraftAliasResponse, error) {
	matches, err := h.svc.ResolveAlias(ctx, req.GetText(), languageNameFromProto(req.GetLanguage()))
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	stats, err := h.svc.Stats(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}

	out := make([]*catalogv1.CraftMatch, 0, len(matches))
	for _, m := range matches {
		out = append(out, craftMatchToProto(m))
	}
	return &catalogv1.ResolveCraftAliasResponse{Matches: out, IndexVersion: stats.Version}, nil
}

// GetCraft fetches one craft by id or by slug.
func (h *Ontology) GetCraft(ctx context.Context, req *catalogv1.GetCraftRequest) (*catalogv1.GetCraftResponse, error) {
	if code := req.GetCode(); code != "" {
		craft, err := h.svc.GetCraftByCode(ctx, code)
		if err != nil {
			return nil, pkgdomain.GRPCError(err)
		}
		return &catalogv1.GetCraftResponse{Craft: craftToProto(craft)}, nil
	}

	id, err := parseUUID("craft_id", req.GetCraftId())
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	craft, err := h.svc.GetCraft(ctx, id)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.GetCraftResponse{Craft: craftToProto(craft)}, nil
}

// ListCrafts returns the whole ontology.
func (h *Ontology) ListCrafts(ctx context.Context, _ *catalogv1.ListCraftsRequest) (*catalogv1.ListCraftsResponse, error) {
	crafts, err := h.svc.ListCrafts(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	out := make([]*catalogv1.Craft, 0, len(crafts))
	for _, c := range crafts {
		out = append(out, craftToProto(c))
	}
	return &catalogv1.ListCraftsResponse{Crafts: out}, nil
}

// RefreshCraftIndex rebuilds the alias index across the fleet.
func (h *Ontology) RefreshCraftIndex(
	ctx context.Context,
	_ *catalogv1.RefreshCraftIndexRequest,
) (*catalogv1.RefreshCraftIndexResponse, error) {
	stats, err := h.svc.RefreshIndex(ctx)
	if err != nil {
		return nil, pkgdomain.GRPCError(err)
	}
	return &catalogv1.RefreshCraftIndexResponse{
		Stats: &catalogv1.CraftIndexStats{
			Version:    stats.Version,
			CraftCount: int32(stats.Crafts),
			AliasCount: int32(stats.Aliases),
			BuiltAt:    timestamppb.New(stats.BuiltAt),
		},
	}, nil
}
