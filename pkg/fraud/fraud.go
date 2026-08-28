// pkg/fraud/fraud.go

// Package fraud provides fraud detection and review workflows.
package fraud

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Detector manages fraud flags.
type Detector struct {
	db *sql.DB
}

// New creates a fraud detector.
func New(db *sql.DB) *Detector {
	return &Detector{db: db}
}

// Flag represents a fraud flag.
type Flag struct {
	ID              uuid.UUID
	ResourceType    string
	ResourceID      uuid.UUID
	FlagType        string
	Severity        string
	Description     string
	Metadata        map[string]any
	Status          string
	CreatedAt       time.Time
	ReviewedAt      *time.Time
	ReviewedBy      *uuid.UUID
	ResolutionNotes string
}

// CreateFlag manually flags a resource for review.
func (d *Detector) CreateFlag(ctx context.Context, flag Flag) error {
	var metadataJSON []byte
	var err error
	if flag.Metadata != nil {
		metadataJSON, err = json.Marshal(flag.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	_, err = d.db.ExecContext(ctx, `
		INSERT INTO fraud_flags (
			resource_type, resource_id, flag_type, severity, description, metadata
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, flag.ResourceType, flag.ResourceID, flag.FlagType, flag.Severity, flag.Description, metadataJSON)

	return err
}

// ListPending returns all pending fraud flags, ordered by severity.
func (d *Detector) ListPending(ctx context.Context, limit int) ([]Flag, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := d.db.QueryContext(ctx, `
		SELECT id, resource_type, resource_id, flag_type, severity, description,
		       metadata, status, created_at, reviewed_at, reviewed_by, resolution_notes
		FROM fraud_flags
		WHERE status IN ('pending', 'reviewing')
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
			END,
			created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		var f Flag
		var metadataJSON []byte

		err := rows.Scan(
			&f.ID, &f.ResourceType, &f.ResourceID, &f.FlagType, &f.Severity,
			&f.Description, &metadataJSON, &f.Status, &f.CreatedAt,
			&f.ReviewedAt, &f.ReviewedBy, &f.ResolutionNotes,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &f.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		flags = append(flags, f)
	}

	return flags, rows.Err()
}

// Resolve marks a flag as reviewed.
func (d *Detector) Resolve(ctx context.Context, flagID uuid.UUID, reviewerID uuid.UUID, isFraud bool, notes string) error {
	status := "resolved_ok"
	if isFraud {
		status = "resolved_fraud"
	}

	_, err := d.db.ExecContext(ctx, `
		UPDATE fraud_flags
		SET status = $1, reviewed_at = NOW(), reviewed_by = $2, resolution_notes = $3
		WHERE id = $4
	`, status, reviewerID, notes, flagID)

	return err
}

// GetFlagsForResource returns all flags for a specific resource.
func (d *Detector) GetFlagsForResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]Flag, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, resource_type, resource_id, flag_type, severity, description,
		       metadata, status, created_at, reviewed_at, reviewed_by, resolution_notes
		FROM fraud_flags
		WHERE resource_type = $1 AND resource_id = $2
		ORDER BY created_at DESC
	`, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		var f Flag
		var metadataJSON []byte

		err := rows.Scan(
			&f.ID, &f.ResourceType, &f.ResourceID, &f.FlagType, &f.Severity,
			&f.Description, &metadataJSON, &f.Status, &f.CreatedAt,
			&f.ReviewedAt, &f.ReviewedBy, &f.ResolutionNotes,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &f.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		flags = append(flags, f)
	}

	return flags, rows.Err()
}

// HasActiveFraudFlags checks if a resource has any unresolved fraud flags.
func (d *Detector) HasActiveFraudFlags(ctx context.Context, resourceType string, resourceID uuid.UUID) (bool, error) {
	var count int
	err := d.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM fraud_flags
		WHERE resource_type = $1 AND resource_id = $2 AND status IN ('pending', 'reviewing')
	`, resourceType, resourceID).Scan(&count)

	return count > 0, err
}

// Stats returns fraud detection statistics.
type Stats struct {
	PendingCount  int
	HighSeverity  int
	ResolvedOK    int
	ResolvedFraud int
}

// GetStats returns fraud flag statistics for dashboard.
func (d *Detector) GetStats(ctx context.Context) (Stats, error) {
	var stats Stats

	err := d.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status IN ('pending', 'reviewing')) AS pending,
			COUNT(*) FILTER (WHERE status IN ('pending', 'reviewing') AND severity IN ('high', 'critical')) AS high_severity,
			COUNT(*) FILTER (WHERE status = 'resolved_ok') AS resolved_ok,
			COUNT(*) FILTER (WHERE status = 'resolved_fraud') AS resolved_fraud
		FROM fraud_flags
	`).Scan(&stats.PendingCount, &stats.HighSeverity, &stats.ResolvedOK, &stats.ResolvedFraud)

	return stats, err
}
