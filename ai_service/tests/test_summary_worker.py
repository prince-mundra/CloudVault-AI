from __future__ import annotations

import asyncio
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any

from ai_service.config import Settings
from ai_service.summary_worker import SummaryWorker


class Cursor:
    def __init__(self, row: dict[str, Any] | None = None, rows: list[dict[str, Any]] | None = None) -> None:
        self.row, self.rows = row, rows or []

    async def fetchone(self) -> dict[str, Any] | None:
        return self.row

    async def fetchall(self) -> list[dict[str, Any]]:
        return self.rows


class Connection:
    def __init__(self, attempts: int = 1) -> None:
        self.attempts = attempts
        self.job_state = "processing"
        self.ingestion_state = "succeeded"
        self.file_row = {"state": "active", "sha256": "a" * 64}
        self.statements: list[tuple[str, tuple[Any, ...]]] = []
        self.summary_insert: tuple[Any, ...] | None = None
        self.chunks = [{"chunk_index": 0, "page_number": 1, "chunk_text": "approved text"}]

    @asynccontextmanager
    async def transaction(self):
        yield

    async def execute(self, query: str, params: tuple[Any, ...] = ()) -> Cursor:
        self.statements.append((query, params))
        normalized = " ".join(query.split()).lower()
        if normalized.startswith("select state,sha256 from files"):
            return Cursor(self.file_row)
        if normalized.startswith("select chunk_index,page_number,chunk_text from document_chunks"):
            return Cursor(rows=self.chunks)
        if normalized.startswith("select state,attempts from document_summary_jobs"):
            return Cursor({"state": self.job_state, "attempts": self.attempts})
        if normalized.startswith("insert into document_summaries"):
            self.summary_insert = params
        if normalized.startswith("update document_summary_jobs") and "set state='succeeded'" in normalized:
            self.job_state = "succeeded"
        if normalized.startswith("update document_summary_jobs") and "set state='cancelled'" in normalized:
            self.job_state = "cancelled"
        if normalized.startswith("update document_summary_jobs") and "set state=%s" in normalized:
            self.job_state = params[0]
        return Cursor()


class Pool:
    def __init__(self, conn: Connection) -> None:
        self.conn = conn

    @asynccontextmanager
    async def connection(self):
        yield self.conn


class FakeSummarizer:
    def __init__(self, error: Exception | None = None) -> None:
        self.error = error
        self.calls: list[list[tuple[int, int | None, str]]] = []

    async def summarize(self, chunks: list[tuple[int, int | None, str]]) -> str:
        self.calls.append(chunks)
        if self.error:
            raise self.error
        return "Factual document summary."


def settings(root: Path) -> Settings:
    return Settings(
        db_host="db", db_port=5432, db_name="vault", db_user="user", db_password="password",
        db_sslmode="disable", internal_secret="s" * 32, storage_root=root,
        ollama_url="http://ollama:11434", ollama_model="nomic-embed-text",
        embedding_dimension=768, poll_seconds=1, max_attempts=3, max_document_bytes=100_000,
        max_pdf_pages=20, max_extracted_chars=10_000, chunk_size=1200, chunk_overlap=150,
        max_ocr_pages=5, summary_model="llama3.2:3b",
    )


def test_success_persists_summary_and_model_under_current_attempt(tmp_path: Path) -> None:
    conn, summarizer = Connection(), FakeSummarizer()
    worker = SummaryWorker(settings(tmp_path), Pool(conn), summarizer)

    asyncio.run(worker.process({"id": 9, "file_id": "file-1", "attempts": 1}))

    assert conn.job_state == "succeeded"
    assert conn.summary_insert == ("file-1", "Factual document summary.", "llama3.2:3b", "a" * 64)
    assert summarizer.calls == [[(0, 1, "approved text")]]
    assert not any("UPDATE document_ingestion_jobs" in sql for sql, _ in conn.statements)


def test_stale_summary_attempt_cannot_persist(tmp_path: Path) -> None:
    conn, summarizer = Connection(attempts=2), FakeSummarizer()
    worker = SummaryWorker(settings(tmp_path), Pool(conn), summarizer)

    asyncio.run(worker.process({"id": 9, "file_id": "file-1", "attempts": 1}))

    assert conn.summary_insert is None
    assert conn.job_state == "processing"
    assert not any("UPDATE document_ingestion_jobs" in sql for sql, _ in conn.statements)


def test_model_failure_retries_only_summary_job(tmp_path: Path) -> None:
    conn = Connection()
    worker = SummaryWorker(settings(tmp_path), Pool(conn), FakeSummarizer(RuntimeError("model offline")))

    asyncio.run(worker.process({"id": 9, "file_id": "file-1", "attempts": 1}))

    assert conn.job_state == "queued"
    assert conn.ingestion_state == "succeeded"
    assert conn.summary_insert is None
    assert any("SummaryProcessingError" in str(params) for _, params in conn.statements)
    assert not any("UPDATE document_ingestion_jobs" in sql for sql, _ in conn.statements)


def test_summary_claim_waits_for_successful_ingestion_and_uses_skip_locked(tmp_path: Path) -> None:
    conn = Connection()
    worker = SummaryWorker(settings(tmp_path), Pool(conn), FakeSummarizer())

    result = asyncio.run(worker.claim_one())

    assert result is None
    sql = conn.statements[-1][0]
    assert "JOIN document_ingestion_jobs AS ij" in sql
    assert "ij.state='succeeded'" in sql
    assert "FOR UPDATE OF sj SKIP LOCKED" in sql
    assert conn.statements[-1][1] == (3, 3)
