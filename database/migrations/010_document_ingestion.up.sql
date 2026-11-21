CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE document_ingestion_jobs (
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
CREATE INDEX idx_document_ingestion_claim
    ON document_ingestion_jobs (state, next_attempt_at, created_at);
CREATE INDEX idx_document_ingestion_lease
    ON document_ingestion_jobs (locked_at) WHERE state = 'processing';

INSERT INTO document_ingestion_jobs (file_id, state)
SELECT id, 'queued'
FROM files
WHERE state = 'active' AND sha256 IS NOT NULL AND btrim(sha256) <> ''
ON CONFLICT (file_id) DO NOTHING;

CREATE TABLE document_chunks (
    id BIGSERIAL PRIMARY KEY,
    file_id VARCHAR(36) NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
    page_number INTEGER CHECK (page_number IS NULL OR page_number > 0),
    chunk_text TEXT NOT NULL,
    embedding vector(768) NOT NULL,
    embedding_model VARCHAR(120) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (file_id, chunk_index)
);
CREATE INDEX idx_document_chunks_file ON document_chunks (file_id, chunk_index);
CREATE INDEX idx_document_chunks_embedding
    ON document_chunks USING hnsw (embedding vector_cosine_ops);
