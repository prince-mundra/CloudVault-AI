// internal/repository/file_repository.go

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/souvik03-136/Go-Store/internal/models"
)

// FileRepository handles all database operations for file metadata.
type FileRepository struct{ db *sql.DB }

func NewFileRepository(db *sql.DB) *FileRepository { return &FileRepository{db: db} }

func (r *FileRepository) CreateFile(file *models.File) error {
	_, err := r.db.Exec(`INSERT INTO files (id,name,path,url,size,content_type,owner_id,folder_id,created_at,updated_at,sha256,state)
		VALUES ($1,$2,$3,'',$4,$5,$6,$7,$8,$9,$10,'active')`, file.ID, file.Name, file.Path,
		file.Size, file.ContentType, file.OwnerID, file.FolderID, file.CreatedAt, file.UpdatedAt, file.SHA256)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	return nil
}

const fileColumns = `id,name,path,url,size,content_type,owner_id,folder_id,created_at,updated_at,sha256,state`

func (r *FileRepository) GetFileByID(id string) (*models.File, error) {
	return r.scanFile(r.db.QueryRow(`SELECT `+fileColumns+` FROM files WHERE id=$1 AND state='active'`, id))
}

func (r *FileRepository) ListFilesByOwner(ownerID string) ([]*models.File, error) {
	rows, err := r.db.Query(`SELECT `+fileColumns+` FROM files WHERE owner_id=$1 AND state='active' ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listing files for owner %s: %w", ownerID, err)
	}
	defer rows.Close()
	files := make([]*models.File, 0)
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating file rows: %w", err)
	}
	return files, nil
}

func (r *FileRepository) ListSharedWithUser(userID string) ([]*models.File, error) {
	rows, err := r.db.Query(`SELECT f.`+fileColumnsWithAlias("f")+` FROM files f JOIN permissions p ON p.file_id=f.id WHERE p.user_id=$1 AND p.can_read=TRUE AND f.state='active' ORDER BY f.updated_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing shared files: %w", err)
	}
	defer rows.Close()
	files := make([]*models.File, 0)
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating shared files: %w", err)
	}
	return files, nil
}

func fileColumnsWithAlias(alias string) string {
	return alias + `.id,` + alias + `.name,` + alias + `.path,` + alias + `.url,` + alias + `.size,` + alias + `.content_type,` + alias + `.owner_id,` + alias + `.folder_id,` + alias + `.created_at,` + alias + `.updated_at,` + alias + `.sha256,` + alias + `.state`
}

func (r *FileRepository) UpdateFile(file *models.File) error {
	res, err := r.db.Exec(`UPDATE files SET name=$1,folder_id=$2,updated_at=$3 WHERE id=$4 AND owner_id=$5 AND state='active'`, file.Name, file.FolderID, time.Now(), file.ID, file.OwnerID)
	if err != nil {
		return fmt.Errorf("updating file %s: %w", file.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("file not found")
	}
	return nil
}

// DeleteFile remains available for owner-account cascades and controlled maintenance;
// HTTP deletion uses the durable deletion-job workflow instead.
func (r *FileRepository) DeleteFile(id string) error {
	res, err := r.db.Exec(`DELETE FROM files WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("deleting file %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("file not found")
	}
	return nil
}

func (r *FileRepository) scanFile(row *sql.Row) (*models.File, error) {
	f := &models.File{}
	err := row.Scan(&f.ID, &f.Name, &f.Path, &f.Url, &f.Size, &f.ContentType, &f.OwnerID, &f.FolderID, &f.CreatedAt, &f.UpdatedAt, &f.SHA256, &f.State)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("file not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scanning file: %w", err)
	}
	return f, nil
}

func scanFileRows(row interface{ Scan(...any) error }) (*models.File, error) {
	f := &models.File{}
	if err := row.Scan(&f.ID, &f.Name, &f.Path, &f.Url, &f.Size, &f.ContentType, &f.OwnerID, &f.FolderID, &f.CreatedAt, &f.UpdatedAt, &f.SHA256, &f.State); err != nil {
		return nil, fmt.Errorf("scanning file row: %w", err)
	}
	return f, nil
}

// UpdateFileAndAudit commits metadata changes and the corresponding audit entry together.
func (r *FileRepository) UpdateFileAndAudit(ctx context.Context, file *models.File, event AuditEvent) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning file metadata update: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE files SET name=$1,folder_id=$2,updated_at=$3 WHERE id=$4 AND owner_id=$5 AND state='active'`, file.Name, file.FolderID, time.Now(), file.ID, file.OwnerID)
	if err != nil {
		return fmt.Errorf("updating file metadata: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("file not found")
	}
	if err := insertAuditEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing file metadata update: %w", err)
	}
	return nil
}
