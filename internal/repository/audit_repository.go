package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type AuditEvent struct {
	ActorID      string
	ActorSubject string
	OwnerID      string
	ResourceType string
	ResourceID   string
	Action       string
	Outcome      string
	Details      map[string]any
}
type AuditRepository struct{ db *sql.DB }

func NewAuditRepository(db *sql.DB) *AuditRepository { return &AuditRepository{db: db} }

type auditExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertAuditEvent(ctx context.Context, exec auditExecer, event AuditEvent) error {
	if event.ActorSubject == "" || event.Action == "" || event.ResourceType == "" {
		return fmt.Errorf("audit actor, action, and resource type are required")
	}
	if event.Outcome == "" {
		event.Outcome = "success"
	}
	if event.Outcome != "success" && event.Outcome != "failure" {
		return fmt.Errorf("invalid audit outcome")
	}
	if event.Details == nil {
		event.Details = map[string]any{}
	}
	payload, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("encoding audit details: %w", err)
	}
	var actor, owner, resource any
	if event.ActorID != "" {
		actor = event.ActorID
	}
	if event.OwnerID != "" {
		owner = event.OwnerID
	}
	if event.ResourceID != "" {
		resource = event.ResourceID
	}
	if _, err := exec.ExecContext(ctx, `INSERT INTO audit_events(actor_id,actor_subject,owner_id,resource_type,resource_id,action,outcome,details) VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb)`, actor, event.ActorSubject, owner, event.ResourceType, resource, event.Action, event.Outcome, string(payload)); err != nil {
		return fmt.Errorf("persisting audit event: %w", err)
	}
	return nil
}
func (r *AuditRepository) Record(ctx context.Context, event AuditEvent) error {
	return insertAuditEvent(ctx, r.db, event)
}
