CREATE TABLE document_summary_jobs (
    id BIGSERIAL PRIMARY KEY,
    file_id VARCHAR(36) NOT NULL UNIQUE REFERENCES files(id) ON DELETE CASCADE,
    state VARCHAR(20) NOT NULL DEFAULT 'queued'
        CHECK (state IN ('queued', 'processing', 'succeeded', 'failed', 'cancelled')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error VARCHAR(160) NOT NULL DEFAULT '',
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_document_summary_claim
    ON document_summary_jobs (state, next_attempt_at, created_at);
CREATE INDEX idx_document_summary_lease
    ON document_summary_jobs (locked_at) WHERE state = 'processing';

CREATE TABLE document_summaries (
    file_id VARCHAR(36) PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    summary_text TEXT NOT NULL,
    summary_model VARCHAR(120) NOT NULL,
    source_sha256 CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (length(btrim(summary_text)) > 0),
    CHECK (length(btrim(source_sha256)) = 64)
);

-- Schedule existing active documents as well. The summary worker will claim a
-- job only after its corresponding ingestion job has succeeded.
INSERT INTO document_summary_jobs (file_id, state)
SELECT f.id, 'queued'
FROM files AS f
JOIN document_ingestion_jobs AS ij ON ij.file_id = f.id
WHERE f.state = 'active'
ON CONFLICT (file_id) DO NOTHING;
