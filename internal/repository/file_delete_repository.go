package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/souvik03-136/Go-Store/internal/models"
)

type FileDeleteJob struct {
	ID, FileID, OwnerID, ObjectKey, ActorSubject, FileName string
	SizeBytes                                              int64
}
type FileDeleteRepository struct{ db *sql.DB }

func NewFileDeleteRepository(db *sql.DB) *FileDeleteRepository { return &FileDeleteRepository{db: db} }

func (r *FileDeleteRepository) Request(ctx context.Context, file *models.File, actorID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE files SET state='deleting',updated_at=NOW() WHERE id=$1 AND owner_id=$2 AND state='active'`, file.ID, file.OwnerID)
	if err != nil {
		return fmt.Errorf("marking file for deletion: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("file not found or deletion already pending")
	}
	jobID := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO file_delete_jobs(id,file_id,owner_id,object_key,size_bytes,actor_subject,file_name) VALUES($1,$2,$3,$4,$5,$6,$7)`, jobID, file.ID, file.OwnerID, file.Path, file.Size, actorID, file.Name); err != nil {
		return fmt.Errorf("queuing file deletion: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events(actor_id,actor_subject,owner_id,resource_type,resource_id,action,outcome,details) VALUES($1,$1,$2,'file',$3,'delete_requested','success',jsonb_build_object('size_bytes',$4))`, actorID, file.OwnerID, file.ID, file.Size); err != nil {
		return fmt.Errorf("recording delete request: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing delete request: %w", err)
	}
	return nil
}

func (r *FileDeleteRepository) Claim(ctx context.Context) (*FileDeleteJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	job := &FileDeleteJob{}
	err = tx.QueryRowContext(ctx, `SELECT id,file_id,owner_id,object_key,size_bytes,actor_subject,file_name FROM file_delete_jobs WHERE available_at<=NOW() ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&job.ID, &job.FileID, &job.OwnerID, &job.ObjectKey, &job.SizeBytes, &job.ActorSubject, &job.FileName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claiming delete job: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE file_delete_jobs SET available_at=NOW()+INTERVAL '1 minute' WHERE id=$1`, job.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return job, nil
}

func (r *FileDeleteRepository) Complete(ctx context.Context, job *FileDeleteJob) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM files WHERE id=$1 AND state='deleting'`, job.FileID)
	if err != nil {
		return fmt.Errorf("removing deleted file metadata: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("file no longer exists or is not deleting")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO file_activity(owner_id,actor_id,file_id,file_name,action,size_bytes) VALUES($1,(SELECT id FROM users WHERE id=$2),NULL,$3,'delete',$4)`, job.OwnerID, job.ActorSubject, job.FileName, job.SizeBytes); err != nil {
		return fmt.Errorf("recording delete activity: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events(actor_id,actor_subject,owner_id,resource_type,resource_id,action,outcome,details) VALUES((SELECT id FROM users WHERE id=$1),$1,$2,'file',$3,'delete','success',jsonb_build_object('size_bytes',$4))`, job.ActorSubject, job.OwnerID, job.FileID, job.SizeBytes); err != nil {
		return fmt.Errorf("recording deletion audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing deletion: %w", err)
	}
	return nil
}
func (r *FileDeleteRepository) Fail(ctx context.Context, job *FileDeleteJob, cause error) error {
	_, err := r.db.ExecContext(ctx, `UPDATE file_delete_jobs SET attempts=attempts+1,last_error=$1,available_at=NOW()+make_interval(secs => LEAST(3600,(5*power(2,LEAST(attempts,9)))::int)) WHERE id=$2`, cause.Error(), job.ID)
	if err != nil {
		return fmt.Errorf("rescheduling failed deletion: %w", err)
	}
	return nil
}
func (r *FileDeleteRepository) Wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
