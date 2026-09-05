// services/bff/internal/bff/handler/admin.go
//
// Batch 13's write surfaces: cluster and self-help-group administration
// (/clusters), listing suspension/reinstatement (/moderation) and the craft
// index refresh (/crafts). All gated to CLUSTER_OFFICER or MINISTRY -- see
// each handler for which.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	"github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/httpx"
)

// idempotencyKey reads the caller's dedup key to forward as the gRPC
// request's own idempotency_key field. This is separate from
// middleware.Idempotency's X-Idempotency-Key replay cache (which wraps these
// routes via withIdempotency): the client sends "Idempotency-Key" (see
// packages/api/src/transport.ts), so that is what is read here too.
func idempotencyKey(r *http.Request) string {
	return r.Header.Get("Idempotency-Key")
}

// CreateCluster registers a cluster.
func (h *APIHandler) CreateCluster(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}

	var req struct {
		Name                 string  `json:"name"`
		StateCode            string  `json:"state_code"`
		District             *string `json:"district"`
		CoordinatorPhoneE164 *string `json:"coordinator_phone_e164"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, err)
		return
	}

	cluster, err := h.catalogSvc.CreateCluster(r.Context(), req.Name, req.StateCode, req.District, req.CoordinatorPhoneE164, idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, cluster)
}

// GetCluster fetches one cluster.
func (h *APIHandler) GetCluster(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	cluster, err := h.catalogSvc.GetCluster(r.Context(), httpx.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cluster)
}

// ListClusterMembers returns a cluster's full roster.
func (h *APIHandler) ListClusterMembers(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	members, err := h.catalogSvc.ListClusterMembers(r.Context(), httpx.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"members": members})
}

// AddClusterMember adds (or re-roles) an artisan in a cluster -- this is also
// how bulk onboarding assigns a freshly registered artisan to their cluster.
func (h *APIHandler) AddClusterMember(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	var req struct {
		ArtisanID string `json:"artisan_id"`
		Role      string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, err)
		return
	}
	member, err := h.catalogSvc.AddClusterMember(r.Context(), httpx.URLParam(r, "id"), req.ArtisanID, req.Role, idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, member)
}

// RemoveClusterMember removes an artisan from a cluster.
func (h *APIHandler) RemoveClusterMember(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	removed, err := h.catalogSvc.RemoveClusterMember(r.Context(), httpx.URLParam(r, "id"), httpx.URLParam(r, "artisanId"), idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": removed})
}

// CreateSelfHelpGroup registers an SHG. share_pct across members must sum to
// exactly 100 -- rejected server-side in core-svc if it doesn't.
func (h *APIHandler) CreateSelfHelpGroup(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	var req struct {
		Name           string           `json:"name"`
		RegistrationNo string           `json:"registration_no"`
		ClusterID      *string          `json:"cluster_id"`
		Members        []map[string]any `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, err)
		return
	}
	group, err := h.catalogSvc.CreateSelfHelpGroup(r.Context(), req.Name, req.RegistrationNo, req.ClusterID, req.Members, idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, group)
}

// GetSelfHelpGroup fetches one SHG with its current roster.
func (h *APIHandler) GetSelfHelpGroup(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	group, err := h.catalogSvc.GetSelfHelpGroup(r.Context(), httpx.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, group)
}

// SetSelfHelpGroupMembers replaces an SHG's roster and share split wholesale.
func (h *APIHandler) SetSelfHelpGroupMembers(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	var req struct {
		Members []map[string]any `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, err)
		return
	}
	roster, err := h.catalogSvc.SetSelfHelpGroupMembers(r.Context(), httpx.URLParam(r, "id"), req.Members, idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, roster)
}

// SuspendListing withdraws a listing -- always a human decision made from the
// moderation review queue, never triggered automatically by a detector.
func (h *APIHandler) SuspendListing(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, err)
		return
	}
	if req.Reason == "" {
		httpx.Error(w, domain.InvalidInput("reason is required"))
		return
	}
	listing, err := h.catalogSvc.SuspendListing(r.Context(), httpx.URLParam(r, "id"), req.Reason, idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, listing)
}

// ReinstateListing returns a suspended listing to PUBLISHED.
func (h *APIHandler) ReinstateListing(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	listing, err := h.catalogSvc.ReinstateListing(r.Context(), httpx.URLParam(r, "id"), idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, listing)
}

// OnboardClusterArtisan registers an artisan into a specific cluster on a
// cluster officer's behalf -- the bulk-onboarding CSV commit calls this once
// per validated row.
//
// POST /artisans (RegisterArtisan above) is self-registration only: it takes
// the phone number from the caller's OWN authenticated principal, ignoring
// anything in the body, because that is what the artisan app's onboarding
// flow needs. identity.v1's RegisterArtisan RPC itself takes an arbitrary
// phone_e164 field -- there was just no REST path exposing that to a caller
// registering someone else. This route is that path, scoped to
// CLUSTER_OFFICER/MINISTRY and to a single artisan per call (bulk CSV commit
// on the frontend still calls it once per row; there is no batch RPC).
func (h *APIHandler) OnboardClusterArtisan(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleClusterOfficer, auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}

	var fields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.Error(w, err)
		return
	}
	phone, _ := fields["phone_e164"].(string)
	if phone == "" {
		httpx.Error(w, domain.InvalidInput("phone_e164 is required"))
		return
	}
	fields["cluster_id"] = httpx.URLParam(r, "id")

	artisanID, regErr := h.artisanSvc.Register(r.Context(), phone, idempotencyKey(r), fields)
	if regErr != nil {
		httpx.Error(w, regErr)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"artisan_id": artisanID})
}

// RefreshCraftIndex rebuilds the ontology alias index from Postgres.
func (h *APIHandler) RefreshCraftIndex(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireRole(r.Context(), auth.RoleMinistry); err != nil {
		httpx.Error(w, err)
		return
	}
	stats, err := h.catalogSvc.RefreshCraftIndex(r.Context(), idempotencyKey(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, stats)
}
