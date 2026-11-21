package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrSummaryStatusNotFound = errors.New("summary status not found")

type SummaryStatus struct {
	State     string    `json:"state"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	Summary   string    `json:"summary,omitempty"`
	Model     string    `json:"model,omitempty"`
}

type SummaryRepository struct{ db *sql.DB }

func NewSummaryRepository(db *sql.DB) *SummaryRepository { return &SummaryRepository{db: db} }

// GetStatus allows only the file owner or a user with an explicit read grant.
// Inactive files and inaccessible files intentionally share the same not-found result.
func (r *SummaryRepository) GetStatus(ctx context.Context, fileID, userID string) (*SummaryStatus, error) {
	status := &SummaryStatus{}
	err := r.db.QueryRowContext(ctx, `
		SELECT j.state,j.attempts,j.last_error,j.updated_at,
		       COALESCE(s.summary_text,''),COALESCE(s.summary_model,'')
		FROM files AS f
		JOIN document_summary_jobs AS j ON j.file_id=f.id
		LEFT JOIN document_summaries AS s ON s.file_id=f.id
		WHERE f.id=$1 AND f.state='active'
		  AND (f.owner_id=$2 OR EXISTS (
			  SELECT 1 FROM permissions AS p
			  WHERE p.file_id=f.id AND p.user_id=$2 AND p.can_read=TRUE
		  ))`, fileID, userID).Scan(
		&status.State, &status.Attempts, &status.LastError, &status.UpdatedAt, &status.Summary, &status.Model,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSummaryStatusNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading document summary status: %w", err)
	}
	return status, nil
}
