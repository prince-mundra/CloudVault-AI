package models

import "time"

type FileTypeUsage struct {
	ContentType string `json:"content_type"`
	FileCount   int64  `json:"file_count"`
	SizeBytes   int64  `json:"size_bytes"`
}

type FileActivity struct {
	ID        int64     `json:"id"`
	ActorID   *string   `json:"actor_id,omitempty"`
	FileID    *string   `json:"file_id,omitempty"`
	FileName  string    `json:"file_name,omitempty"`
	Action    string    `json:"action"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type StorageAnalytics struct {
	TotalFiles     int64           `json:"total_files"`
	UsedBytes      int64           `json:"used_bytes"`
	ReservedBytes  int64           `json:"reserved_bytes"`
	QuotaBytes     int64           `json:"quota_bytes"`
	RemainingBytes int64           `json:"remaining_bytes"`
	FileTypes      []FileTypeUsage `json:"file_types"`
	RecentUploads  []*File         `json:"recent_uploads"`
	RecentActivity []FileActivity  `json:"recent_activity"`
}
