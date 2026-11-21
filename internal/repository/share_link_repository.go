package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/souvik03-136/Go-Store/internal/models"
)

type ShareLinkRepository struct{ db *sql.DB }

func NewShareLinkRepository(db *sql.DB) *ShareLinkRepository { return &ShareLinkRepository{db: db} }

func (r *ShareLinkRepository) Create(ctx context.Context, link *models.ShareLink) error {
	var password any
	if link.PasswordHash != "" {
		password = link.PasswordHash
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO share_links(id,token_hash,file_id,owner_id,password_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, link.ID, link.TokenHash, link.FileID, link.OwnerID, password, link.ExpiresAt)
	if err != nil {
		return fmt.Errorf("creating share link: %w", err)
	}
	return nil
}
func (r *ShareLinkRepository) FindActive(ctx context.Context, tokenHash string) (*models.ShareLink, error) {
	link := &models.ShareLink{}
	var pass sql.NullString
	var revoked sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id,file_id,owner_id,token_hash,password_hash,expires_at,revoked_at,created_at,access_count FROM share_links WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>NOW()`, tokenHash).Scan(&link.ID, &link.FileID, &link.OwnerID, &link.TokenHash, &pass, &link.ExpiresAt, &revoked, &link.CreatedAt, &link.AccessCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("share link not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("looking up share link: %w", err)
	}
	if pass.Valid {
		link.PasswordHash = pass.String
	}
	if revoked.Valid {
		t := revoked.Time
		link.RevokedAt = &t
	}
	link.PasswordProtected = link.PasswordHash != ""
	return link, nil
}
func (r *ShareLinkRepository) Touch(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE share_links SET access_count=access_count+1,last_accessed_at=NOW() WHERE id=$1 AND revoked_at IS NULL AND expires_at>NOW()`, id)
	if err != nil {
		return fmt.Errorf("updating share link access: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("share link not found or expired")
	}
	return nil
}
func (r *ShareLinkRepository) Revoke(ctx context.Context, id, ownerID string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE share_links SET revoked_at=NOW() WHERE id=$1 AND owner_id=$2 AND revoked_at IS NULL`, id, ownerID)
	if err != nil {
		return fmt.Errorf("revoking share link: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("share link not found")
	}
	return nil
}
func (r *ShareLinkRepository) ListByOwner(ctx context.Context, ownerID, fileID string) ([]*models.ShareLink, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,file_id,owner_id,expires_at,revoked_at,created_at,access_count,(password_hash IS NOT NULL) FROM share_links WHERE owner_id=$1 AND file_id=$2 ORDER BY created_at DESC LIMIT 100`, ownerID, fileID)
	if err != nil {
		return nil, fmt.Errorf("listing share links: %w", err)
	}
	defer rows.Close()
	out := make([]*models.ShareLink, 0)
	for rows.Next() {
		x := &models.ShareLink{}
		var revoked sql.NullTime
		if err := rows.Scan(&x.ID, &x.FileID, &x.OwnerID, &x.ExpiresAt, &revoked, &x.CreatedAt, &x.AccessCount, &x.PasswordProtected); err != nil {
			return nil, fmt.Errorf("scanning share link: %w", err)
		}
		if revoked.Valid {
			t := revoked.Time
			x.RevokedAt = &t
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating share links: %w", err)
	}
	return out, nil
}
