package models

import "time"

type ShareLink struct {
	ID                string     `json:"id"`
	FileID            string     `json:"file_id"`
	OwnerID           string     `json:"owner_id"`
	PasswordProtected bool       `json:"password_protected"`
	ExpiresAt         time.Time  `json:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	AccessCount       int64      `json:"access_count"`
	TokenHash         string     `json:"-"`
	PasswordHash      string     `json:"-"`
}
