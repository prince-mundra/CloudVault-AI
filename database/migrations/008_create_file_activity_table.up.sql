CREATE TABLE IF NOT EXISTS file_activity (
    id BIGSERIAL PRIMARY KEY,
    owner_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_id VARCHAR(36) REFERENCES files(id) ON DELETE SET NULL,
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    action VARCHAR(20) NOT NULL CHECK (action IN ('upload', 'download', 'delete')),
    size_bytes BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_file_activity_owner_created ON file_activity (owner_id, created_at DESC);
