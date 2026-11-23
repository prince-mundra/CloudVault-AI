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

var (
	ErrQuotaExceeded       = errors.New("storage quota exceeded")
	ErrUploadInProgress    = errors.New("an upload with identical content is already in progress")
	ErrReservationNotFound = errors.New("upload reservation not found")
)

type UploadReservation struct {
	ID        string
	OwnerID   string
	SHA256    string
	SizeBytes int64
	ObjectKey string
	ExpiresAt time.Time
}

type UploadRepository struct{ db *sql.DB }

func NewUploadRepository(db *sql.DB) *UploadRepository { return &UploadRepository{db: db} }

// Reserve serializes all new reservations for one account by locking its user row.
// Used bytes are calculated from committed files plus outstanding reservations in
// the same transaction, so parallel uploads cannot spend the same quota twice.
func (r *UploadRepository) Reserve(ctx context.Context, ownerID, hash, objectKey string, size, quota int64) (*UploadReservation, *models.File, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("beginning quota reservation: %w", err)
	}
	defer tx.Rollback()
	var id string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, ownerID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errors.New("user not found")
		}
		return nil, nil, fmt.Errorf("locking upload owner: %w", err)
	}
	if f, err := scanUploadFile(tx.QueryRowContext(ctx, `SELECT id,name,path,url,size,content_type,owner_id,folder_id,created_at,updated_at,sha256,state FROM files WHERE owner_id=$1 AND sha256=$2 AND state='active' ORDER BY created_at LIMIT 1`, ownerID, hash)); err == nil {
		return nil, f, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, fmt.Errorf("checking duplicate content: %w", err)
	}
	var pending int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM upload_reservations WHERE owner_id=$1 AND sha256=$2`, ownerID, hash).Scan(&pending); err != nil {
		return nil, nil, fmt.Errorf("checking pending duplicate: %w", err)
	}
	if pending > 0 {
		return nil, nil, ErrUploadInProgress
	}
	var committed, reserved int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT SUM(size) FROM files WHERE owner_id=$1 AND state='active'),0), COALESCE((SELECT SUM(size_bytes) FROM upload_reservations WHERE owner_id=$1),0)`, ownerID).Scan(&committed, &reserved); err != nil {
		return nil, nil, fmt.Errorf("checking storage usage: %w", err)
	}
	if size < 0 || size > quota || committed > quota-size || reserved > quota-size-committed {
		return nil, nil, ErrQuotaExceeded
	}
	res := &UploadReservation{ID: uuid.NewString(), OwnerID: ownerID, SHA256: hash, SizeBytes: size, ObjectKey: objectKey, ExpiresAt: time.Now().Add(6 * time.Hour)}
	if _, err := tx.ExecContext(ctx, `INSERT INTO upload_reservations (id,owner_id,sha256,size_bytes,object_key,expires_at) VALUES ($1,$2,$3,$4,$5,$6)`, res.ID, res.OwnerID, res.SHA256, res.SizeBytes, res.ObjectKey, res.ExpiresAt); err != nil {
		return nil, nil, fmt.Errorf("creating upload reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("committing quota reservation: %w", err)
	}
	return res, nil, nil
}

// Finalize atomically converts the reservation into file metadata and records
// the successful upload and audit event. The object is already present in storage.
func (r *UploadRepository) Finalize(ctx context.Context, reservationID string, f *models.File, actorID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning upload finalization: %w", err)
	}
	defer tx.Rollback()
	var ownerID string
	if err := tx.QueryRowContext(ctx, `SELECT owner_id FROM upload_reservations WHERE id=$1 FOR UPDATE`, reservationID).Scan(&ownerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrReservationNotFound
		}
		return fmt.Errorf("reading upload reservation: %w", err)
	}
	if ownerID != f.OwnerID {
		return errors.New("reservation owner mismatch")
	}
	actor := actorID
	if actor == "" {
		actor = f.OwnerID
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO files (id,name,path,url,size,content_type,owner_id,folder_id,created_at,updated_at,sha256,state) VALUES ($1,$2,$3,'',$4,$5,$6,$7,$8,$9,$10,'active')`, f.ID, f.Name, f.Path, f.Size, f.ContentType, f.OwnerID, f.FolderID, f.CreatedAt, f.UpdatedAt, f.SHA256); err != nil {
		return fmt.Errorf("persisting file metadata: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO file_activity (owner_id,file_id,file_name,action,size_bytes,actor_id) VALUES ($1,$2,$3,'upload',$4,$5)`, f.OwnerID, f.ID, f.Name, f.Size, actor); err != nil {
		return fmt.Errorf("recording upload activity: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events (actor_id,actor_subject,owner_id,resource_type,resource_id,action,outcome,details) VALUES ($1,$1,$2,'file',$3,'upload','success',jsonb_build_object('size_bytes',$4::bigint,'sha256',$5::text,'content_type',$6::text))`, actor, f.OwnerID, f.ID, f.Size, f.SHA256, f.ContentType); err != nil {
		return fmt.Errorf("recording upload audit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO document_ingestion_jobs (file_id,state) VALUES ($1,'queued')`, f.ID); err != nil {
		return fmt.Errorf("queuing document ingestion: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO document_summary_jobs (file_id,state) VALUES ($1,'queued')`, f.ID); err != nil {
		return fmt.Errorf("queuing document summary: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM upload_reservations WHERE id=$1`, reservationID); err != nil {
		return fmt.Errorf("releasing committed reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing uploaded file: %w", err)
	}
	return nil
}

func (r *UploadRepository) Release(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM upload_reservations WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("releasing upload reservation: %w", err)
	}
	return nil
}

func (r *UploadRepository) Expired(ctx context.Context, ownerID string) ([]UploadReservation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,owner_id,sha256,size_bytes,object_key,expires_at FROM upload_reservations WHERE owner_id=$1 AND expires_at<=NOW() ORDER BY expires_at`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listing expired reservations: %w", err)
	}
	defer rows.Close()
	out := make([]UploadReservation, 0)
	for rows.Next() {
		var x UploadReservation
		if err := rows.Scan(&x.ID, &x.OwnerID, &x.SHA256, &x.SizeBytes, &x.ObjectKey, &x.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scanning expired reservation: %w", err)
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating expired reservations: %w", err)
	}
	return out, nil
}

func (r *UploadRepository) ReleaseExpired(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM upload_reservations WHERE id=$1 AND expires_at<=NOW()`, id)
	if err != nil {
		return fmt.Errorf("releasing expired reservation: %w", err)
	}
	return nil
}

func (r *UploadRepository) ExpiredAll(ctx context.Context) ([]UploadReservation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,owner_id,sha256,size_bytes,object_key,expires_at FROM upload_reservations WHERE expires_at<=NOW() ORDER BY expires_at LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("listing all expired reservations: %w", err)
	}
	defer rows.Close()
	out := make([]UploadReservation, 0)
	for rows.Next() {
		var x UploadReservation
		if err := rows.Scan(&x.ID, &x.OwnerID, &x.SHA256, &x.SizeBytes, &x.ObjectKey, &x.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scanning expired reservation: %w", err)
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating expired reservations: %w", err)
	}
	return out, nil
}

func scanUploadFile(row *sql.Row) (*models.File, error) {
	f := &models.File{}
	err := row.Scan(&f.ID, &f.Name, &f.Path, &f.Url, &f.Size, &f.ContentType, &f.OwnerID, &f.FolderID, &f.CreatedAt, &f.UpdatedAt, &f.SHA256, &f.State)
	if err != nil {
		return nil, err
	}
	return f, nil
}
