CREATE TABLE IF NOT EXISTS folders (
    id         VARCHAR(36) PRIMARY KEY,
    owner_id   VARCHAR(36) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    parent_id  VARCHAR(36) REFERENCES folders (id) ON DELETE RESTRICT,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT folders_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX IF NOT EXISTS idx_folders_owner_parent ON folders (owner_id, parent_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_owner_root_name
    ON folders (owner_id, name) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_owner_parent_name
    ON folders (owner_id, parent_id, name) WHERE parent_id IS NOT NULL;
