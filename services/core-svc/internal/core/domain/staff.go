// services/core-svc/internal/core/domain/staff.go

package domain

import (
	"time"

	"github.com/google/uuid"
)

// StaffAccount is a ministry-issued login for field and oversight staff. A
// phone that matches an active row logs in as Role instead of as an artisan.
type StaffAccount struct {
	ID          uuid.UUID
	PhoneE164   string
	DisplayName string
	// Role is FIELD_AGENT, CLUSTER_OFFICER or MINISTRY.
	Role      string
	StateCode *string
	District  *string
	ClusterID *uuid.UUID
	CSCID     *string
	Active    bool
	CreatedBy string
	CreatedAt time.Time
}

// Assisted-mode consent methods (assisted_link.consent_method).
const (
	ConsentArtisanOTP     = "ARTISAN_OTP"
	ConsentVoiceRecording = "VOICE_RECORDING"
)

// AssistedLink is an artisan's consent for one agent to act for them.
type AssistedLink struct {
	ID            uuid.UUID
	AgentID       uuid.UUID
	ArtisanID     uuid.UUID
	ConsentMethod string
	ConsentRef    string
	ConsentAt     time.Time
	NeedsReview   bool
	RevokedAt     *time.Time
}

// AgentArtisan is one row of an agent's "My artisans" list.
type AgentArtisan struct {
	ArtisanID      uuid.UUID
	DisplayName    string
	Village        *string
	District       *string
	StateCode      string
	PhotoMediaID   *uuid.UUID
	ConsentMethod  string
	NeedsReview    bool
	LinkedAt       time.Time
	LastActivityAt *time.Time
	DraftCount     int64
}

// ArtisanHelper is one agent currently allowed to act for an artisan.
type ArtisanHelper struct {
	LinkID        uuid.UUID
	AgentID       uuid.UUID
	DisplayName   string
	Role          string
	CSCID         *string
	ConsentMethod string
	LinkedAt      time.Time
}

// LinkForReview is a voice-consent link awaiting an officer's review.
type LinkForReview struct {
	LinkID      uuid.UUID
	AgentID     uuid.UUID
	AgentName   string
	ArtisanID   uuid.UUID
	ArtisanName string
	District    *string
	StateCode   string
	ConsentRef  string
	CreatedAt   time.Time
}

// AgentProductivity is one row of the admin's agent table.
type AgentProductivity struct {
	AgentID           uuid.UUID
	DisplayName       string
	Role              string
	StateCode         *string
	District          *string
	CSCID             *string
	Active            bool
	ArtisansOnboarded int64
	ListingsCreated   int64
	LastActiveAt      *time.Time
}

// AssistedAudit is one audit_log row for a write made on an artisan's behalf.
type AssistedAudit struct {
	ActorID      uuid.UUID
	SubjectID    uuid.UUID
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
}
