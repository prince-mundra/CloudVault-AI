package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/souvik03-136/Go-Store/internal/models"
)

type FileActivityRepository struct{ db *sql.DB }

func NewFileActivityRepository(db *sql.DB) *FileActivityRepository {
	return &FileActivityRepository{db: db}
}

func (r *FileActivityRepository) Record(ownerID, actorID string, fileID *string, fileName, action string, sizeBytes int64) error {
	if action != "upload" && action != "download" && action != "delete" {
		return errors.New("invalid file activity action")
	}
	if sizeBytes < 0 {
		return errors.New("activity size cannot be negative")
	}
	var actor any
	if actorID != "" {
		actor = actorID
	}
	_, err := r.db.Exec(`INSERT INTO file_activity (owner_id,actor_id,file_id,file_name,action,size_bytes) VALUES ($1,$2,$3,$4,$5,$6)`, ownerID, actor, fileID, fileName, action, sizeBytes)
	if err != nil {
		return fmt.Errorf("recording file activity: %w", err)
	}
	return nil
}

func (r *FileActivityRepository) GetStorageAnalytics(ownerID string) (*models.StorageAnalytics, error) {
	stats := &models.StorageAnalytics{FileTypes: make([]models.FileTypeUsage, 0), RecentUploads: make([]*models.File, 0), RecentActivity: make([]models.FileActivity, 0)}
	if err := r.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(size),0) FROM files WHERE owner_id=$1 AND state='active'`, ownerID).Scan(&stats.TotalFiles, &stats.UsedBytes); err != nil {
		return nil, fmt.Errorf("reading storage totals: %w", err)
	}
	if err := r.db.QueryRow(`SELECT COALESCE(SUM(size_bytes),0) FROM upload_reservations WHERE owner_id=$1 AND expires_at>NOW()`, ownerID).Scan(&stats.ReservedBytes); err != nil {
		return nil, fmt.Errorf("reading reserved storage: %w", err)
	}
	rows, err := r.db.Query(`SELECT COALESCE(NULLIF(content_type,''),'unknown'),COUNT(*),COALESCE(SUM(size),0) FROM files WHERE owner_id=$1 AND state='active' GROUP BY COALESCE(NULLIF(content_type,''),'unknown') ORDER BY SUM(size) DESC`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("reading storage file types: %w", err)
	}
	for rows.Next() {
		var item models.FileTypeUsage
		if err := rows.Scan(&item.ContentType, &item.FileCount, &item.SizeBytes); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning file type totals: %w", err)
		}
		stats.FileTypes = append(stats.FileTypes, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterating file type totals: %w", err)
	}
	rows.Close()
	uploads, err := r.db.Query(`SELECT `+fileColumns+` FROM files WHERE owner_id=$1 AND state='active' ORDER BY created_at DESC,id DESC LIMIT 10`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("reading recent uploads: %w", err)
	}
	for uploads.Next() {
		f, err := scanFileRows(uploads)
		if err != nil {
			uploads.Close()
			return nil, err
		}
		stats.RecentUploads = append(stats.RecentUploads, f)
	}
	if err := uploads.Err(); err != nil {
		uploads.Close()
		return nil, fmt.Errorf("iterating recent uploads: %w", err)
	}
	uploads.Close()
	activity, err := r.db.Query(`SELECT id,actor_id,file_id,file_name,action,size_bytes,created_at FROM file_activity WHERE owner_id=$1 ORDER BY created_at DESC,id DESC LIMIT 20`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("reading recent file activity: %w", err)
	}
	defer activity.Close()
	for activity.Next() {
		var item models.FileActivity
		var actor sql.NullString
		if err := activity.Scan(&item.ID, &actor, &item.FileID, &item.FileName, &item.Action, &item.SizeBytes, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning file activity: %w", err)
		}
		if actor.Valid {
			item.ActorID = &actor.String
		}
		stats.RecentActivity = append(stats.RecentActivity, item)
	}
	if err := activity.Err(); err != nil {
		return nil, fmt.Errorf("iterating file activity: %w", err)
	}
	return stats, nil
}
