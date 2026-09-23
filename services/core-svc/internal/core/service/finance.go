// services/core-svc/internal/core/service/finance.go

package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ZoroNewbie00/kalakriti/pkg/auth"
	pkgdomain "github.com/ZoroNewbie00/kalakriti/pkg/domain"
	"github.com/ZoroNewbie00/kalakriti/pkg/ids"
	"github.com/ZoroNewbie00/kalakriti/pkg/money"

	"github.com/ZoroNewbie00/kalakriti/services/core-svc/internal/core/domain"
)

// FinanceStore is the persistence surface of the finance-link service (F12).
type FinanceStore interface {
	InsertFinanceLink(ctx context.Context, l domain.FinanceLink, refHash []byte, t domain.FinanceTerms) (domain.FinanceLink, error)
	GetFinanceLink(ctx context.Context, id uuid.UUID) (domain.FinanceLink, error)
	ListFinanceLinks(ctx context.Context, artisanID uuid.UUID) ([]domain.FinanceLink, error)
	UpdateFinanceLink(ctx context.Context, id, artisanID uuid.UUID, t domain.FinanceTerms) (domain.FinanceLink, error)
	DeleteFinanceLink(ctx context.Context, id, artisanID uuid.UUID) (bool, error)
	SetFinanceLinkStatus(ctx context.Context, id uuid.UUID, status, reviewer string, reason *string) (domain.FinanceLink, error)
	ListFinanceLinksForReview(ctx context.Context, status, stateCode, district *string) ([]domain.FinanceLinkForReview, error)
	GetEarnedInRange(ctx context.Context, artisanID uuid.UUID, from, to time.Time) (platform, offline, pending int64, err error)
	GetArtisan(ctx context.Context, id uuid.UUID) (domain.Artisan, error)
	HasActiveAssistedLink(ctx context.Context, agentID, artisanID uuid.UUID) (bool, error)
}

// Finance links an artisan's finance-corporation loan to their shop.
type Finance struct {
	store FinanceStore
	// salt keys the HMAC over loan references; see domain.HashReference.
	salt []byte
	log  *slog.Logger
	now  func() time.Time
}

// NewFinance builds the finance service. salt must be a deployment secret.
func NewFinance(store FinanceStore, salt []byte, log *slog.Logger) *Finance {
	if log == nil {
		log = slog.Default()
	}
	return &Finance{store: store, salt: salt, log: log, now: time.Now}
}

// LinkFinanceInput is a new link as the artisan enters it. Reference is the
// full plaintext reference and must never be logged.
type LinkFinanceInput struct {
	Corporation    string
	Reference      string
	Terms          domain.FinanceTerms
	ConsentGiven   bool
	ConsentVersion string
}

func (s *Finance) LinkFinance(ctx context.Context, in LinkFinanceInput) (domain.FinanceLink, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return domain.FinanceLink{}, err
	}
	// DPDP Act 2023: no consent, no record.
	if !in.ConsentGiven || strings.TrimSpace(in.ConsentVersion) == "" {
		return domain.FinanceLink{}, pkgdomain.InvalidInput("consent is required to link a loan record")
	}
	if !domain.FinanceCorporations[in.Corporation] {
		return domain.FinanceLink{}, pkgdomain.InvalidInput(fmt.Sprintf("corporation %q is not a known finance corporation", in.Corporation))
	}
	if err := in.Terms.Validate(); err != nil {
		return domain.FinanceLink{}, err
	}
	last4, hash, err := domain.HashReference(s.salt, in.Reference)
	if err != nil {
		return domain.FinanceLink{}, err
	}
	return s.store.InsertFinanceLink(ctx, domain.FinanceLink{
		ID: ids.New(), ArtisanID: artisanID, Corporation: in.Corporation, ReferenceLast4: last4,
		ConsentVersion: in.ConsentVersion,
	}, hash, in.Terms)
}

func (s *Finance) ListMyFinanceLinks(ctx context.Context) ([]domain.FinanceLink, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.ListFinanceLinks(ctx, artisanID)
}

func (s *Finance) UpdateFinanceLink(ctx context.Context, id uuid.UUID, t domain.FinanceTerms) (domain.FinanceLink, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return domain.FinanceLink{}, err
	}
	if err := t.Validate(); err != nil {
		return domain.FinanceLink{}, err
	}
	return s.store.UpdateFinanceLink(ctx, id, artisanID, t)
}

// DeleteFinanceLink is consent withdrawal: a hard delete, from the artisan's
// own session only -- never by an agent acting for them.
func (s *Finance) DeleteFinanceLink(ctx context.Context, id uuid.UUID) (bool, error) {
	artisanID, p, err := artisanSelf(ctx)
	if err != nil {
		return false, err
	}
	if p.Actor != "" {
		return false, pkgdomain.Forbidden("only the artisan can withdraw consent, from their own phone")
	}
	return s.store.DeleteFinanceLink(ctx, id, artisanID)
}

// ReviewFinanceLink marks a link verified (after checking the sanction
// letter) or rejected. MINISTRY may review any link; a CLUSTER_OFFICER only
// in their scope; a FIELD_AGENT only for artisans linked to them.
func (s *Finance) ReviewFinanceLink(ctx context.Context, id uuid.UUID, verified bool, reason *string) (domain.FinanceLink, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry, auth.RoleClusterOfficer, auth.RoleFieldAgent)
	if err != nil {
		return domain.FinanceLink{}, err
	}
	link, err := s.store.GetFinanceLink(ctx, id)
	if err != nil {
		return domain.FinanceLink{}, err
	}
	if err := s.authoriseStaffFor(ctx, p, link.ArtisanID); err != nil {
		return domain.FinanceLink{}, err
	}
	status := domain.FinanceVerified
	if !verified {
		status = domain.FinanceRejected
		if reason == nil || strings.TrimSpace(*reason) == "" {
			return domain.FinanceLink{}, pkgdomain.InvalidInput("a reason is required to reject a link")
		}
	}
	return s.store.SetFinanceLinkStatus(ctx, id, status, p.Subject, reason)
}

// authoriseStaffFor checks a staff principal may act on one artisan's record.
func (s *Finance) authoriseStaffFor(ctx context.Context, p auth.Principal, artisanID uuid.UUID) error {
	switch p.Role {
	case auth.RoleMinistry:
		return nil
	case auth.RoleFieldAgent:
		agentID, err := uuid.Parse(p.Subject)
		if err != nil {
			return pkgdomain.Forbidden("a staff account is required")
		}
		linked, err := s.store.HasActiveAssistedLink(ctx, agentID, artisanID)
		if err != nil {
			return err
		}
		if !linked {
			return pkgdomain.Forbidden("you are not linked to this artisan")
		}
		return nil
	default: // CLUSTER_OFFICER
		a, err := s.store.GetArtisan(ctx, artisanID)
		if err != nil {
			return err
		}
		if !inScope(p, a.Region.StateCode, a.Region.District) {
			return pkgdomain.Forbidden("this artisan is outside your area")
		}
		return nil
	}
}

// inScope reports whether a region falls inside a staff principal's scope.
func inScope(p auth.Principal, stateCode string, district *string) bool {
	if p.ScopeState != "" && p.ScopeState != stateCode {
		return false
	}
	if p.ScopeDistrict != "" && (district == nil || !strings.EqualFold(*district, p.ScopeDistrict)) {
		return false
	}
	return true
}

// scopeFilter clamps a requested region to the principal's own scope.
func scopeFilter(p auth.Principal, stateCode, district *string) (*string, *string) {
	if p.ScopeState != "" {
		stateCode = &p.ScopeState
	}
	if p.ScopeDistrict != "" {
		district = &p.ScopeDistrict
	}
	return stateCode, district
}

func (s *Finance) ListFinanceLinksForReview(ctx context.Context, status, stateCode, district *string) ([]domain.FinanceLinkForReview, error) {
	p, err := auth.RequireRole(ctx, auth.RoleMinistry, auth.RoleClusterOfficer)
	if err != nil {
		return nil, err
	}
	if p.Role == auth.RoleClusterOfficer {
		stateCode, district = scopeFilter(p, stateCode, district)
	}
	return s.store.ListFinanceLinksForReview(ctx, status, stateCode, district)
}

// GetRepaymentCoverage reports how far the month's income covers the
// artisan's EMIs. month is "YYYY-MM" or empty for the current month.
func (s *Finance) GetRepaymentCoverage(ctx context.Context, month string) (domain.RepaymentCoverage, error) {
	artisanID, _, err := artisanSelf(ctx)
	if err != nil {
		return domain.RepaymentCoverage{}, err
	}
	today := s.now().UTC()
	start, err := domain.ParseMonth(month, today)
	if err != nil {
		return domain.RepaymentCoverage{}, err
	}
	links, err := s.store.ListFinanceLinks(ctx, artisanID)
	if err != nil {
		return domain.RepaymentCoverage{}, err
	}
	platform, offline, pending, err := s.store.GetEarnedInRange(ctx, artisanID, start, start.AddDate(0, 1, 0))
	if err != nil {
		return domain.RepaymentCoverage{}, err
	}
	return domain.ComputeCoverage(links, start, platform, offline, pending, today), nil
}

// ReminderStore is what the EMI reminder job needs.
type ReminderStore interface {
	ListDueReminders(ctx context.Context, emiDay int32, dueMonth time.Time) ([]domain.EMIReminder, error)
	SendEMIReminder(ctx context.Context, r domain.EMIReminder, dueMonth time.Time, title, body string, payload map[string]any) (bool, error)
	GetEarnedInRange(ctx context.Context, artisanID uuid.UUID, from, to time.Time) (platform, offline, pending int64, err error)
}

// emiReminderLead is how far ahead of the EMI date the reminder goes out.
const emiReminderLead = 3 * 24 * time.Hour

// SendDueEMIReminders sends one reminder per link whose EMI falls three days
// from now, saying how much the artisan has earned this month so far. Safe
// to run on every replica: ClaimEmiReminder's (link, month) key de-duplicates.
func SendDueEMIReminders(ctx context.Context, store ReminderStore, now time.Time, log *slog.Logger) (int, error) {
	target := now.UTC().Add(emiReminderLead)
	if target.Day() > 28 { // emi_day_of_month is 1..28
		return 0, nil
	}
	dueMonth := time.Date(target.Year(), target.Month(), 1, 0, 0, 0, 0, time.UTC)
	due, err := store.ListDueReminders(ctx, int32(target.Day()), dueMonth)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, r := range due {
		platform, offline, _, err := store.GetEarnedInRange(ctx, r.ArtisanID, dueMonth, dueMonth.AddDate(0, 1, 0))
		if err != nil {
			log.Warn("emi reminder: reading earnings", "link_id", r.LinkID, "error", err)
			continue
		}
		earned := platform + offline
		// The app renders payload.i18n_key in the artisan's own language; the
		// English title/body are the fallback for any other channel.
		payload := map[string]any{
			"i18n_key": "finance.reminder.body", "link_id": r.LinkID.String(),
			"params": map[string]any{
				"earned": money.New(earned).Format(), "emi": money.New(r.EmiPaise).Format(), "days": 3,
			},
		}
		body := fmt.Sprintf("Your EMI of %s is due in 3 days. You have earned %s this month.",
			money.New(r.EmiPaise).Format(), money.New(earned).Format())
		ok, err := store.SendEMIReminder(ctx, r, dueMonth, "EMI due in 3 days", body, payload)
		if err != nil {
			log.Warn("emi reminder: sending", "link_id", r.LinkID, "error", err)
			continue
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

// RunEMIReminders runs SendDueEMIReminders now and then every interval until
// ctx is cancelled.
func RunEMIReminders(ctx context.Context, store ReminderStore, interval time.Duration, log *slog.Logger) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		if n, err := SendDueEMIReminders(ctx, store, time.Now(), log); err != nil {
			log.Error("emi reminders", "error", err)
		} else if n > 0 {
			log.Info("emi reminders sent", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
