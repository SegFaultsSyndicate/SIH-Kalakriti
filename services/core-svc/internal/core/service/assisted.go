// services/core-svc/internal/core/service/assisted.go

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// AssistedStore is the persistence surface of staff accounts and assisted mode (F14).
type AssistedStore interface {
	GetStaffAccount(ctx context.Context, id uuid.UUID) (domain.StaffAccount, error)
	CreateStaffAccount(ctx context.Context, s domain.StaffAccount) (domain.StaffAccount, error)
	ListStaffAccounts(ctx context.Context, stateCode *string) ([]domain.StaffAccount, error)
	SetStaffActive(ctx context.Context, id uuid.UUID, active bool) (domain.StaffAccount, error)
	ListAgentProductivity(ctx context.Context, stateCode, district *string) ([]domain.AgentProductivity, error)

	UpsertAssistedLink(ctx context.Context, l domain.AssistedLink) (domain.AssistedLink, error)
	AttachVoiceConsent(ctx context.Context, agentID, artisanID, mediaID uuid.UUID) (bool, error)
	HasActiveAssistedLink(ctx context.Context, agentID, artisanID uuid.UUID) (bool, error)
	RevokeAssistedLink(ctx context.Context, linkID, artisanID uuid.UUID) (bool, error)
	ListAgentArtisans(ctx context.Context, agentID uuid.UUID) ([]domain.AgentArtisan, error)
	ListArtisanHelpers(ctx context.Context, artisanID uuid.UUID) ([]domain.ArtisanHelper, error)
	ListLinksNeedingReview(ctx context.Context, stateCode, district *string) ([]domain.LinkForReview, error)
	MarkLinkReviewed(ctx context.Context, linkID uuid.UUID, reviewer string) (bool, error)
	GetListingHelper(ctx context.Context, listingID, artisanID uuid.UUID) (name string, cscID *string, ok bool, err error)
	InsertAssistedAudit(ctx context.Context, a domain.AssistedAudit, metadata []byte) error

	ArtisanExistsByPhone(ctx context.Context, phone string) (uuid.UUID, bool, error)
	GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error)
	GetMedia(ctx context.Context, id uuid.UUID) (domain.Media, error)
}

// ArtisanRegistrar is the slice of the identity service assisted onboarding
// reuses, so an agent-registered artisan goes through exactly the same
// validation, craft edges and artisan.registered event as a self-registration.
type ArtisanRegistrar interface {
	RegisterArtisan(ctx context.Context, in domain.RegisterArtisanInput, idempotencyKey string) (domain.Artisan, auth.TokenPair, error)
}

// Assisted is staff account administration and assisted mode.
type Assisted struct {
	store     AssistedStore
	otp       Challenger
	registrar ArtisanRegistrar
	log       *slog.Logger
}

// NewAssisted builds the assisted-mode service.
func NewAssisted(store AssistedStore, otp Challenger, registrar ArtisanRegistrar, log *slog.Logger) *Assisted {
	if log == nil {
		log = slog.Default()
	}
	return &Assisted{store: store, otp: otp, registrar: registrar, log: log}
}

var staffRoles = map[string]bool{"FIELD_AGENT": true, "CLUSTER_OFFICER": true, "MINISTRY": true}

// --- staff accounts ----------------------------------------------------------

func (s *Assisted) CreateStaff(ctx context.Context, in domain.StaffAccount) (domain.StaffAccount, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry)
	if err != nil {
		return domain.StaffAccount{}, err
	}
	if !phoneE164Pattern.MatchString(in.PhoneE164) {
		return domain.StaffAccount{}, pkgdomain.InvalidInput("phone_e164 must be E.164, e.g. +919876543210")
	}
	if strings.TrimSpace(in.DisplayName) == "" {
		return domain.StaffAccount{}, pkgdomain.InvalidInput("display_name is required")
	}
	if !staffRoles[in.Role] {
		return domain.StaffAccount{}, pkgdomain.InvalidInput("role must be FIELD_AGENT, CLUSTER_OFFICER or MINISTRY")
	}
	if in.District != nil && in.StateCode == nil {
		return domain.StaffAccount{}, pkgdomain.InvalidInput("a district scope needs a state")
	}
	in.ID = ids.New()
	in.CreatedBy = p.Subject
	return s.store.CreateStaffAccount(ctx, in)
}

func (s *Assisted) ListStaff(ctx context.Context, stateCode *string) ([]domain.StaffAccount, error) {
	if _, err := auth.RequireRole(ctx, auth.RoleMinistry); err != nil {
		return nil, err
	}
	return s.store.ListStaffAccounts(ctx, stateCode)
}

func (s *Assisted) SetStaffActive(ctx context.Context, id uuid.UUID, active bool) (domain.StaffAccount, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry)
	if err != nil {
		return domain.StaffAccount{}, err
	}
	if !active && p.Subject == id.String() {
		return domain.StaffAccount{}, pkgdomain.InvalidInput("you cannot deactivate your own account")
	}
	return s.store.SetStaffActive(ctx, id, active)
}

// staffSelf returns the calling staff account, refusing dev_role tokens
// (whose subject is a phone, not a staff id) and deactivated accounts.
func (s *Assisted) staffSelf(ctx context.Context, roles ...auth.Role) (domain.StaffAccount, auth.Principal, error) {
	p, err := auth.RequireRole(ctx, roles...)
	if err != nil {
		return domain.StaffAccount{}, auth.Principal{}, err
	}
	id, err := uuid.Parse(p.Subject)
	if err != nil {
		return domain.StaffAccount{}, auth.Principal{}, pkgdomain.Forbidden("a staff account is required (dev-role tokens cannot use assisted mode)")
	}
	staff, err := s.store.GetStaffAccount(ctx, id)
	if err != nil {
		return domain.StaffAccount{}, auth.Principal{}, err
	}
	if !staff.Active {
		return domain.StaffAccount{}, auth.Principal{}, pkgdomain.Forbidden("this staff account is deactivated")
	}
	return staff, p, nil
}

func (s *Assisted) GetMyStaffAccount(ctx context.Context) (domain.StaffAccount, error) {
	staff, _, err := s.staffSelf(ctx, auth.RoleFieldAgent, auth.RoleClusterOfficer, auth.RoleMinistry)
	return staff, err
}

func (s *Assisted) ListAgentProductivity(ctx context.Context, stateCode, district *string) ([]domain.AgentProductivity, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry, auth.RoleClusterOfficer)
	if err != nil {
		return nil, err
	}
	if p.Role == auth.RoleClusterOfficer {
		stateCode, district = scopeFilter(p, stateCode, district)
	}
	return s.store.ListAgentProductivity(ctx, stateCode, district)
}

// --- linking an artisan ------------------------------------------------------

var agentRoles = []auth.Role{auth.RoleFieldAgent, auth.RoleClusterOfficer}

// StartArtisanConsent sends an OTP to the ARTISAN's phone; the artisan reads
// it out to the agent, which is their consent.
func (s *Assisted) StartArtisanConsent(ctx context.Context, phone, language string) (Challenge, error) {
	if _, _, err := s.staffSelf(ctx, agentRoles...); err != nil {
		return Challenge{}, err
	}
	if !phoneE164Pattern.MatchString(phone) {
		return Challenge{}, pkgdomain.InvalidInput("phone_e164 must be E.164, e.g. +919876543210")
	}
	return s.otp.Request(ctx, phone, language)
}

// LinkArtisanInput is one onboarding: consent plus, for a new artisan, their
// registration details.
type LinkArtisanInput struct {
	PhoneE164     string
	ConsentMethod string
	ChallengeID   string
	OTP           string
	Registration  *domain.RegisterArtisanInput
}

// LinkArtisanResult reports what LinkArtisan did.
type LinkArtisanResult struct {
	ArtisanID     uuid.UUID
	LinkID        uuid.UUID
	RegisteredNow bool
	NeedsReview   bool
}

// voiceConsentPending is a voice link's consent_ref until the recording is
// attached (AttachVoiceConsent); such links are always flagged for review.
const voiceConsentPending = "awaiting-recording"

func (s *Assisted) LinkArtisan(ctx context.Context, in LinkArtisanInput) (LinkArtisanResult, error) {
	staff, _, err := s.staffSelf(ctx, agentRoles...)
	if err != nil {
		return LinkArtisanResult{}, err
	}
	if !phoneE164Pattern.MatchString(in.PhoneE164) {
		return LinkArtisanResult{}, pkgdomain.InvalidInput("phone_e164 must be E.164, e.g. +919876543210")
	}

	link := domain.AssistedLink{ID: ids.New(), AgentID: staff.ID, ConsentMethod: in.ConsentMethod}
	switch in.ConsentMethod {
	case domain.ConsentArtisanOTP:
		if in.ChallengeID == "" || in.OTP == "" {
			return LinkArtisanResult{}, pkgdomain.InvalidInput("challenge_id and otp are required for ARTISAN_OTP consent")
		}
		if err := s.otp.Verify(ctx, in.ChallengeID, in.PhoneE164, in.OTP); err != nil {
			return LinkArtisanResult{}, err
		}
		link.ConsentRef = in.ChallengeID
	case domain.ConsentVoiceRecording:
		link.ConsentRef = voiceConsentPending
		link.NeedsReview = true
	default:
		return LinkArtisanResult{}, pkgdomain.InvalidInput("consent_method must be ARTISAN_OTP or VOICE_RECORDING")
	}

	result := LinkArtisanResult{NeedsReview: link.NeedsReview}
	artisanID, exists, err := s.store.ArtisanExistsByPhone(ctx, in.PhoneE164)
	if err != nil {
		return LinkArtisanResult{}, err
	}
	if !exists {
		if in.Registration == nil {
			return LinkArtisanResult{}, pkgdomain.InvalidInput("this phone has no Kalakriti profile yet: registration details are required")
		}
		reg := *in.Registration
		reg.PhoneE164 = in.PhoneE164
		reg.CreatedBy = "agent:" + staff.ID.String()
		// Register exactly as the artisan would have, bound to their own
		// phone. The tokens RegisterArtisan mints are discarded: the agent
		// never holds an artisan session.
		regCtx := auth.ContextWithPrincipal(ctx, auth.Principal{Role: auth.RoleArtisan, PhoneE164: in.PhoneE164})
		created, _, err := s.registrar.RegisterArtisan(regCtx, reg, "assisted:"+link.ID.String())
		if err != nil {
			return LinkArtisanResult{}, err
		}
		artisanID, result.RegisteredNow = created.ID, true
	}

	link.ArtisanID = artisanID
	saved, err := s.store.UpsertAssistedLink(ctx, link)
	if err != nil {
		return LinkArtisanResult{}, err
	}
	result.ArtisanID, result.LinkID = artisanID, saved.ID

	meta, _ := json.Marshal(map[string]any{"consent_method": in.ConsentMethod, "registered_now": result.RegisteredNow})
	if err := s.store.InsertAssistedAudit(ctx, domain.AssistedAudit{
		ActorID: staff.ID, SubjectID: artisanID, Action: "assisted.link",
		ResourceType: "assisted_link", ResourceID: saved.ID,
	}, meta); err != nil {
		s.log.WarnContext(ctx, "assisted link audit", "error", err)
	}
	return result, nil
}

// AttachVoiceConsent attaches the recorded consent to the calling agent's
// voice-consent link. The recording must be an audio clip belonging to that
// artisan (uploaded in assisted mode after linking).
func (s *Assisted) AttachVoiceConsent(ctx context.Context, artisanID, mediaID uuid.UUID) (bool, error) {
	staff, _, err := s.staffSelf(ctx, agentRoles...)
	if err != nil {
		return false, err
	}
	media, err := s.store.GetMedia(ctx, mediaID)
	if err != nil {
		return false, err
	}
	if media.ArtisanID != artisanID || media.Kind != domain.MediaKind("AUDIO") {
		return false, pkgdomain.InvalidInput("media_id must be an audio recording belonging to this artisan")
	}
	return s.store.AttachVoiceConsent(ctx, staff.ID, artisanID, mediaID)
}

func (s *Assisted) ListMyArtisans(ctx context.Context) ([]domain.AgentArtisan, error) {
	staff, _, err := s.staffSelf(ctx, agentRoles...)
	if err != nil {
		return nil, err
	}
	return s.store.ListAgentArtisans(ctx, staff.ID)
}

// ResolveOnBehalf reports whether the calling agent may act for artisanID
// right now. It is the single check the bff runs before honouring
// X-On-Behalf-Of; a revoked link or a deactivated agent answers false.
func (s *Assisted) ResolveOnBehalf(ctx context.Context, artisanID uuid.UUID) (bool, string, error) {
	staff, _, err := s.staffSelf(ctx, agentRoles...)
	if err != nil {
		return false, "", err
	}
	linked, err := s.store.HasActiveAssistedLink(ctx, staff.ID, artisanID)
	if err != nil || !linked {
		return false, "", err
	}
	a, err := s.store.GetArtisan(ctx, artisanID)
	if err != nil {
		return false, "", err
	}
	return true, a.DisplayName, nil
}

// --- the artisan's side -----------------------------------------------------

func (s *Assisted) ListMyHelpers(ctx context.Context) ([]domain.ArtisanHelper, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.ListArtisanHelpers(ctx, artisanID)
}

// RevokeHelper withdraws an agent's access immediately. The artisan's own
// session only: an agent can never revoke (or un-revoke) consent.
func (s *Assisted) RevokeHelper(ctx context.Context, linkID uuid.UUID) (bool, error) {
	artisanID, p, err := artisanSelf(ctx)
	if err != nil {
		return false, err
	}
	if p.Actor != "" {
		return false, pkgdomain.Forbidden("only the artisan can remove a helper, from their own phone")
	}
	return s.store.RevokeAssistedLink(ctx, linkID, artisanID)
}

func (s *Assisted) GetListingHelper(ctx context.Context, listingID uuid.UUID) (string, *string, bool, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return "", nil, false, err
	}
	return s.store.GetListingHelper(ctx, listingID, artisanID)
}

// --- officer review ------------------------------------------------------------

func (s *Assisted) ListLinksForReview(ctx context.Context, stateCode, district *string) ([]domain.LinkForReview, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry, auth.RoleClusterOfficer)
	if err != nil {
		return nil, err
	}
	if p.Role == auth.RoleClusterOfficer {
		stateCode, district = scopeFilter(p, stateCode, district)
	}
	return s.store.ListLinksNeedingReview(ctx, stateCode, district)
}

func (s *Assisted) MarkLinkReviewed(ctx context.Context, linkID uuid.UUID) (bool, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry, auth.RoleClusterOfficer)
	if err != nil {
		return false, err
	}
	return s.store.MarkLinkReviewed(ctx, linkID, p.Subject)
}

// --- audit -----------------------------------------------------------------

// AuditWriter records assisted-mode writes. The gRPC audit interceptor
// depends on this, not the repo, so "allowed action -> audit row" is a unit
// test rather than a database test.
type AuditWriter interface {
	InsertAssistedAudit(ctx context.Context, a domain.AssistedAudit, metadata []byte) error
}

// AuditOnBehalf writes one audit row for a write made by an agent acting for
// an artisan. resourceID is the created entity's id when known, else the
// artisan's own id.
func AuditOnBehalf(ctx context.Context, w AuditWriter, p auth.Principal, method string, resourceID *uuid.UUID) error {
	actor, err := uuid.Parse(p.Actor)
	if err != nil {
		return fmt.Errorf("assisted audit: actor %q is not a staff id: %w", p.Actor, err)
	}
	subject, err := uuid.Parse(p.Subject)
	if err != nil {
		return fmt.Errorf("assisted audit: subject %q is not an artisan id: %w", p.Subject, err)
	}
	res := subject
	if resourceID != nil {
		res = *resourceID
	}
	return w.InsertAssistedAudit(ctx, domain.AssistedAudit{
		ActorID: actor, SubjectID: subject, Action: method, ResourceType: "rpc", ResourceID: res,
	}, nil)
}
