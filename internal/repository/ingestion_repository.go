package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrIngestionStatusNotFound = errors.New("ingestion status not found")

type IngestionStatus struct {
	State     string    `json:"state"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IngestionRepository struct{ db *sql.DB }

func NewIngestionRepository(db *sql.DB) *IngestionRepository {
	return &IngestionRepository{db: db}
}

func (r *IngestionRepository) GetStatus(ctx context.Context, fileID, ownerID string) (*IngestionStatus, error) {
	status := &IngestionStatus{}
	err := r.db.QueryRowContext(ctx, `
		SELECT j.state, j.attempts, j.last_error, j.updated_at
		FROM files AS f
		JOIN document_ingestion_jobs AS j ON j.file_id=f.id
		WHERE f.id=$1 AND f.owner_id=$2`, fileID, ownerID).Scan(&status.State, &status.Attempts, &status.LastError, &status.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrIngestionStatusNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading ingestion status: %w", err)
	}
	return status, nil
}
