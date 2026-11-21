package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type TokenRevocationRepository struct{ db *sql.DB }

func NewTokenRevocationRepository(db *sql.DB) *TokenRevocationRepository {
	return &TokenRevocationRepository{db: db}
}

func (r *TokenRevocationRepository) Revoke(tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(`INSERT INTO revoked_tokens (token_hash, expires_at) VALUES ($1,$2)
		ON CONFLICT (token_hash) DO UPDATE SET expires_at=EXCLUDED.expires_at`, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}
	// Opportunistically prune expired fingerprints so the table remains bounded.
	_, _ = r.db.Exec(`DELETE FROM revoked_tokens WHERE expires_at < NOW()`)
	return nil
}

func (r *TokenRevocationRepository) IsRevoked(tokenHash string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM revoked_tokens WHERE token_hash=$1 AND expires_at > NOW())`, tokenHash).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking token revocation: %w", err)
	}
	return exists, nil
}
