// pkg/audit/audit.go

// Package audit provides application-level audit logging for compliance.
// Database triggers handle most audits automatically, but this package lets
// services log audits explicitly when they have context triggers lack.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
)

// Logger writes audit events to the audit_log table.
type Logger struct {
	db *sql.DB
}

// New creates an audit logger.
func New(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// Event is an audit log entry.
type Event struct {
	ActorID      *uuid.UUID
	ActorType    string // user, service, system
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
	Changes      map[string]any
	IPAddress    *net.IP
	UserAgent    string
	Metadata     map[string]any
}

// Log writes an audit event.
func (l *Logger) Log(ctx context.Context, event Event) error {
	var actorID *uuid.UUID
	if event.ActorID != nil {
		actorID = event.ActorID
	}

	var changesJSON, metadataJSON []byte
	var err error

	if event.Changes != nil {
		changesJSON, err = json.Marshal(event.Changes)
		if err != nil {
			return fmt.Errorf("marshal changes: %w", err)
		}
	}

	if event.Metadata != nil {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	var ipAddr *string
	if event.IPAddress != nil {
		s := event.IPAddress.String()
		ipAddr = &s
	}

	_, err = l.db.ExecContext(ctx, `
		INSERT INTO audit_log (
			actor_id, actor_type, action, resource_type, resource_id,
			changes, ip_address, user_agent, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, actorID, event.ActorType, event.Action, event.ResourceType, event.ResourceID,
		changesJSON, ipAddr, event.UserAgent, metadataJSON)

	return err
}

// Query fetches audit events matching criteria.
type Query struct {
	ActorID      *uuid.UUID
	ResourceType string
	ResourceID   *uuid.UUID
	Action       string
	After        *time.Time
	Before       *time.Time
	Limit        int
}

// Entry is a row from the audit_log table.
type Entry struct {
	ID           uuid.UUID
	Timestamp    time.Time
	ActorID      *uuid.UUID
	ActorType    string
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
	Changes      map[string]any
	IPAddress    *net.IP
	UserAgent    string
	Metadata     map[string]any
}

// Fetch retrieves audit log entries matching the query.
func (l *Logger) Fetch(ctx context.Context, q Query) ([]Entry, error) {
	query := `
		SELECT id, timestamp, actor_id, actor_type, action, resource_type,
		       resource_id, changes, ip_address, user_agent, metadata
		FROM audit_log
		WHERE 1=1
	`
	args := []any{}
	argNum := 1

	if q.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id = $%d", argNum)
		args = append(args, q.ActorID)
		argNum++
	}

	if q.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argNum)
		args = append(args, q.ResourceType)
		argNum++
	}

	if q.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argNum)
		args = append(args, q.ResourceID)
		argNum++
	}

	if q.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argNum)
		args = append(args, q.Action)
		argNum++
	}

	if q.After != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argNum)
		args = append(args, q.After)
		argNum++
	}

	if q.Before != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argNum)
		args = append(args, q.Before)
		argNum++
	}

	query += " ORDER BY timestamp DESC"

	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, q.Limit)
	}

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var changesJSON, metadataJSON []byte
		var ipStr sql.NullString

		err := rows.Scan(
			&e.ID, &e.Timestamp, &e.ActorID, &e.ActorType, &e.Action,
			&e.ResourceType, &e.ResourceID, &changesJSON, &ipStr,
			&e.UserAgent, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		if len(changesJSON) > 0 {
			if err := json.Unmarshal(changesJSON, &e.Changes); err != nil {
				return nil, fmt.Errorf("unmarshal changes: %w", err)
			}
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		if ipStr.Valid {
			ip := net.ParseIP(ipStr.String)
			e.IPAddress = &ip
		}

		entries = append(entries, e)
	}

	return entries, rows.Err()
}
