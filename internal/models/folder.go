package models

import "time"

// Folder represents one owner-scoped node in a folder tree.
type Folder struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
