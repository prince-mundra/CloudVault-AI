ALTER TABLE users ADD COLUMN auth_version BIGINT NOT NULL DEFAULT 1;

ALTER TABLE files
    ADD COLUMN sha256 CHAR(64),
    ADD COLUMN state VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD CONSTRAINT files_state_check CHECK (state IN ('active', 'deleting'));
CREATE INDEX idx_files_owner_sha256 ON files (owner_id, sha256) WHERE sha256 IS NOT NULL;
CREATE INDEX idx_files_state ON files (state);

ALTER TABLE file_activity ADD COLUMN actor_id VARCHAR(36) REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_file_activity_actor_created ON file_activity (actor_id, created_at DESC);

CREATE TABLE upload_reservations (
    id VARCHAR(36) PRIMARY KEY,
    owner_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sha256 CHAR(64) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    object_key VARCHAR(1024) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, sha256)
);
CREATE INDEX idx_upload_reservations_owner_expiry ON upload_reservations (owner_id, expires_at);

CREATE TABLE audit_events (
    id BIGSERIAL PRIMARY KEY,
    actor_id VARCHAR(36) REFERENCES users(id) ON DELETE SET NULL,
    actor_subject VARCHAR(128) NOT NULL,
    owner_id VARCHAR(36) REFERENCES users(id) ON DELETE SET NULL,
    resource_type VARCHAR(40) NOT NULL,
    resource_id VARCHAR(36),
    action VARCHAR(60) NOT NULL,
    outcome VARCHAR(20) NOT NULL CHECK (outcome IN ('success', 'failure')),
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_events_actor_created ON audit_events (actor_subject, created_at DESC);
CREATE INDEX idx_audit_events_owner_created ON audit_events (owner_id, created_at DESC);
CREATE INDEX idx_audit_events_resource ON audit_events (resource_type, resource_id, created_at DESC);

CREATE TABLE file_delete_jobs (
    id VARCHAR(36) PRIMARY KEY,
    file_id VARCHAR(36) NOT NULL UNIQUE REFERENCES files(id) ON DELETE CASCADE,
    owner_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_key VARCHAR(1024) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    actor_subject VARCHAR(128) NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_file_delete_jobs_available ON file_delete_jobs (available_at);

CREATE TABLE share_links (
    id VARCHAR(36) PRIMARY KEY,
    token_hash CHAR(64) NOT NULL UNIQUE,
    file_id VARCHAR(36) NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    owner_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ,
    access_count BIGINT NOT NULL DEFAULT 0 CHECK (access_count >= 0)
);
CREATE INDEX idx_share_links_owner_created ON share_links (owner_id, created_at DESC);
CREATE INDEX idx_share_links_file ON share_links (file_id);
