package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/souvik03-136/Go-Store/internal/models"
)

type FolderRepository struct{ db *sql.DB }

func NewFolderRepository(db *sql.DB) *FolderRepository { return &FolderRepository{db: db} }

func (r *FolderRepository) Create(f *models.Folder) error {
	_, err := r.db.Exec(`INSERT INTO folders (id, owner_id, parent_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, f.ID, f.OwnerID, f.ParentID, f.Name, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}
	return nil
}

func (r *FolderRepository) GetByID(id string) (*models.Folder, error) {
	return scanFolder(r.db.QueryRow(`SELECT id, owner_id, parent_id, name, created_at, updated_at FROM folders WHERE id=$1`, id))
}

func (r *FolderRepository) GetOwned(id, ownerID string) (*models.Folder, error) {
	return scanFolder(r.db.QueryRow(`SELECT id, owner_id, parent_id, name, created_at, updated_at FROM folders WHERE id=$1 AND owner_id=$2`, id, ownerID))
}

func (r *FolderRepository) List(ownerID string, parentID *string) ([]*models.Folder, error) {
	rows, err := r.db.Query(`SELECT id, owner_id, parent_id, name, created_at, updated_at
		FROM folders WHERE owner_id=$1 AND parent_id IS NOT DISTINCT FROM $2
		ORDER BY name, id`, ownerID, parentID)
	if err != nil {
		return nil, fmt.Errorf("listing folders: %w", err)
	}
	defer rows.Close()
	folders := make([]*models.Folder, 0)
	for rows.Next() {
		f, err := scanFolderRows(rows)
		if err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating folders: %w", err)
	}
	return folders, nil
}

func (r *FolderRepository) Update(f *models.Folder) error {
	res, err := r.db.Exec(`UPDATE folders SET parent_id=$1, name=$2, updated_at=$3 WHERE id=$4 AND owner_id=$5`, f.ParentID, f.Name, time.Now(), f.ID, f.OwnerID)
	if err != nil {
		return fmt.Errorf("updating folder: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("folder not found")
	}
	return nil
}

func (r *FolderRepository) Delete(id, ownerID string) error {
	res, err := r.db.Exec(`DELETE FROM folders WHERE id=$1 AND owner_id=$2`, id, ownerID)
	if err != nil {
		return fmt.Errorf("deleting folder (must be empty): %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("folder not found")
	}
	return nil
}

func (r *FolderRepository) IsEmpty(id string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM folders WHERE parent_id=$1) +
		(SELECT COUNT(*) FROM files WHERE folder_id=$1)`, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking folder contents: %w", err)
	}
	return count == 0, nil
}

func scanFolder(row *sql.Row) (*models.Folder, error) {
	f := &models.Folder{}
	var parent sql.NullString
	if err := row.Scan(&f.ID, &f.OwnerID, &parent, &f.Name, &f.CreatedAt, &f.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("folder not found")
		}
		return nil, fmt.Errorf("scanning folder: %w", err)
	}
	if parent.Valid {
		f.ParentID = &parent.String
	}
	return f, nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanFolderRows(row rowScanner) (*models.Folder, error) {
	f := &models.Folder{}
	var parent sql.NullString
	if err := row.Scan(&f.ID, &f.OwnerID, &parent, &f.Name, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scanning folder row: %w", err)
	}
	if parent.Valid {
		f.ParentID = &parent.String
	}
	return f, nil
}
