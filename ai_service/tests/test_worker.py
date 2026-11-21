from __future__ import annotations

import asyncio
import hashlib
import json
import time
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any

from ai_service.config import Settings
from ai_service.worker import IngestionWorker


class FakeCursor:
    def __init__(self, row: dict[str, Any] | None = None, rows: list[dict[str, Any]] | None = None) -> None:
        self.row = row
        self.rows = rows or []

    async def fetchone(self) -> dict[str, Any] | None:
        return self.row

    async def fetchall(self) -> list[dict[str, Any]]:
        return self.rows


class FakeConnection:
    def __init__(self, file_row: dict[str, Any], job_attempts: int) -> None:
        self.file_row = file_row
        self.job_attempts = job_attempts
        self.job_state = "processing"
        self.statements: list[tuple[str, tuple[Any, ...]]] = []
        self.inserted: list[tuple[Any, ...]] = []

    @asynccontextmanager
    async def transaction(self):
        yield

    async def execute(self, query: str, params: tuple[Any, ...] = ()) -> FakeCursor:
        self.statements.append((query, params))
        normalized = " ".join(query.split()).lower()
        if "from document_chunks as c" in normalized:
            user_id, model, content_type = params[1], params[3], params[4]
            authorized = user_id == self.file_row.get("owner_id") or user_id in self.file_row.get("readers", set())
            results = []
            if authorized and self.file_row["state"] == "active":
                for chunk in self.inserted:
                    if chunk[0] != self.file_row["id"] or chunk[5] != model:
                        continue
                    if content_type and not self.file_row["content_type"].startswith(content_type):
                        continue
                    results.append({
                        "file_id": chunk[0],
                        "file_name": self.file_row["name"],
                        "chunk_index": chunk[1],
                        "page_number": chunk[2],
                        "chunk_text": chunk[3],
                        "score": 0.99,
                    })
            return FakeCursor(rows=results[: params[-1]])
        if normalized.startswith("select id, path, name, content_type, size, sha256, state from files"):
            return FakeCursor(self.file_row)
        if normalized.startswith("select state, attempts from document_ingestion_jobs"):
            return FakeCursor({"state": self.job_state, "attempts": self.job_attempts})
        if normalized.startswith("select state from files"):
            return FakeCursor({"state": self.file_row["state"]})
        if "set state='succeeded'" in normalized and params[-1] == self.job_attempts:
            self.job_state = "succeeded"
        if "set state='failed'" in normalized and len(params) >= 3 and params[-1] == self.job_attempts:
            self.job_state = "failed"
        if "set state=%s" in normalized and params and params[-1] == self.job_attempts:
            self.job_state = params[0]
        return FakeCursor()

    async def executemany(self, query: str, params: list[tuple[Any, ...]]) -> None:
        self.statements.append((query, ()))
        self.inserted = list(params)


class FakePool:
    def __init__(self, connection: FakeConnection) -> None:
        self.conn = connection

    @asynccontextmanager
    async def connection(self):
        yield self.conn


class FakeEmbedder:
    def __init__(self, error: Exception | None = None) -> None:
        self.calls: list[list[str]] = []
        self.error = error

    async def embed(self, texts: list[str]) -> list[list[float]]:
        self.calls.append(list(texts))
        if self.error is not None:
            raise self.error
        return [[0.01] * 768 for _ in texts]


def settings(root: Path) -> Settings:
    return Settings(
        db_host="db",
        db_port=5432,
        db_name="vault",
        db_user="user",
        db_password="password",
        db_sslmode="disable",
        internal_secret="s" * 32,
        storage_root=root,
        ollama_url="http://ollama:11434",
        ollama_model="nomic-embed-text",
        embedding_dimension=768,
        poll_seconds=1,
        max_attempts=3,
        max_document_bytes=100_000,
        max_pdf_pages=20,
        max_extracted_chars=10_000,
        chunk_size=64,
        chunk_overlap=8,
        max_ocr_pages=5,
    )


def make_file(root: Path) -> dict[str, Any]:
    object_path = root / "owner-1" / "file-1.txt"
    object_path.parent.mkdir(parents=True)
    content = b"A locally stored report with enough text to exercise chunking and embedding. " * 4
    object_path.write_bytes(content)
    return {
        "id": "file-1",
        "path": "owner-1/file-1.txt",
        "name": "report.txt",
        "content_type": "text/plain",
        "size": len(content),
        "sha256": hashlib.sha256(content).hexdigest(),
        "state": "active",
        "owner_id": "owner-1",
        "readers": {"reader-1"},
    }


def test_worker_processes_and_commits_chunks_for_current_attempt(tmp_path: Path) -> None:
    file_row = make_file(tmp_path)
    connection = FakeConnection(file_row, job_attempts=1)
    embedder = FakeEmbedder()
    worker = IngestionWorker(settings(tmp_path), FakePool(connection), embedder)

    asyncio.run(worker.process({"id": 7, "file_id": "file-1", "attempts": 1}))

    assert connection.job_state == "succeeded"
    assert connection.inserted
    assert embedder.calls
    assert all(row[0] == "file-1" for row in connection.inserted)
    assert all(row[5] == "nomic-embed-text" for row in connection.inserted)
    assert any("FOR UPDATE" in statement and "document_ingestion_jobs" in statement for statement, _ in connection.statements)
    assert any("attempts=%s" in statement for statement, _ in connection.statements if "UPDATE document_ingestion_jobs" in statement)


def test_stale_worker_attempt_cannot_replace_chunks(tmp_path: Path) -> None:
    file_row = make_file(tmp_path)
    connection = FakeConnection(file_row, job_attempts=2)
    embedder = FakeEmbedder()
    worker = IngestionWorker(settings(tmp_path), FakePool(connection), embedder)

    asyncio.run(worker.process({"id": 7, "file_id": "file-1", "attempts": 1}))

    assert embedder.calls
    assert connection.inserted == []
    assert not any("DELETE FROM document_chunks" in statement for statement, _ in connection.statements)
    assert connection.job_state == "processing"


def test_worker_fails_closed_when_stored_sha_is_missing(tmp_path: Path) -> None:
    file_row = make_file(tmp_path)
    file_row["sha256"] = ""
    connection = FakeConnection(file_row, job_attempts=1)
    worker = IngestionWorker(settings(tmp_path), FakePool(connection), FakeEmbedder())

    asyncio.run(worker.process({"id": 7, "file_id": "file-1", "attempts": 1}))

    assert connection.job_state == "failed"
    assert not connection.inserted


def test_worker_retries_transient_embedding_failure_until_attempt_limit(tmp_path: Path) -> None:
    file_row = make_file(tmp_path)
    connection = FakeConnection(file_row, job_attempts=1)
    worker = IngestionWorker(settings(tmp_path), FakePool(connection), FakeEmbedder(RuntimeError("offline")))

    asyncio.run(worker.process({"id": 7, "file_id": "file-1", "attempts": 1}))

    assert connection.job_state == "queued"
    retry_updates = [params for query, params in connection.statements if "set state=%s" in query.lower()]
    assert retry_updates and retry_updates[-1][-1] == 1


def test_worker_marks_transient_failure_at_attempt_limit_terminal(tmp_path: Path) -> None:
    file_row = make_file(tmp_path)
    connection = FakeConnection(file_row, job_attempts=3)
    worker = IngestionWorker(settings(tmp_path), FakePool(connection), FakeEmbedder(RuntimeError("offline")))

    asyncio.run(worker.process({"id": 7, "file_id": "file-1", "attempts": 3}))

    assert connection.job_state == "failed"
    retry_updates = [params for query, params in connection.statements if "set state=%s" in query.lower()]
    assert retry_updates and retry_updates[-1][0] == "failed"


def test_claim_one_recovery_sql_is_skip_locked_and_attempt_bounded(tmp_path: Path) -> None:
    connection = FakeConnection(make_file(tmp_path), job_attempts=1)
    worker = IngestionWorker(settings(tmp_path), FakePool(connection), FakeEmbedder())

    result = asyncio.run(worker.claim_one())

    assert result is None
    assert len(connection.statements) == 2
    claim_sql, claim_params = connection.statements[1]
    assert "FOR UPDATE SKIP LOCKED" in claim_sql
    assert "locked_at < NOW() - INTERVAL '10 minutes'" in claim_sql
    assert claim_sql.count("attempts < %s") == 2
    assert claim_params == (3, 3)


def test_local_ingestion_to_authorized_semantic_search_pipeline(tmp_path: Path, monkeypatch: Any) -> None:
    monkeypatch.setenv("AI_SHARED_SECRET", "s" * 32)
    import httpx

    from ai_service.api import create_app
    from ai_service.security import request_signature

    file_row = make_file(tmp_path)
    connection = FakeConnection(file_row, job_attempts=1)
    local_settings = settings(tmp_path)
    embedder = FakeEmbedder()
    worker = IngestionWorker(local_settings, FakePool(connection), embedder)

    async def run_pipeline() -> tuple[httpx.Response, httpx.Response, httpx.Response, httpx.Response, httpx.Response]:
        await worker.process({"id": 7, "file_id": "file-1", "attempts": 1})
        app = create_app(local_settings, pool=FakePool(connection), embedder=embedder)
        transport = httpx.ASGITransport(app=app)
        async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
            async def search(user_id: str, content_type: str = "text/") -> httpx.Response:
                path = "/internal/v1/search"
                timestamp = str(int(time.time()))
                body = json.dumps(
                    {"user_id": user_id, "query": "local report", "limit": 10, "content_type": content_type},
                    separators=(",", ":"),
                ).encode()
                signature = request_signature(local_settings.internal_secret, timestamp, "POST", path, body)
                headers = {
                    "Content-Type": "application/json",
                    "X-CloudVault-Timestamp": timestamp,
                    "X-CloudVault-Signature": signature,
                }
                return await client.post(path, content=body, headers=headers)

            owner = await search("owner-1")
            reader = await search("reader-1")
            outsider = await search("outsider")
            wrong_type = await search("owner-1", "image/")
            file_row["state"] = "deleting"
            deleted = await search("owner-1")
            return owner, reader, outsider, wrong_type, deleted

    owner, reader, outsider, wrong_type, deleted = asyncio.run(run_pipeline())

    assert connection.job_state == "succeeded"
    assert connection.inserted
    assert owner.status_code == 200 and owner.json()["results"]
    assert all(hit["file_id"] == "file-1" for hit in owner.json()["results"])
    assert reader.status_code == 200 and reader.json()["results"]
    assert outsider.status_code == 200 and outsider.json()["results"] == []
    assert wrong_type.status_code == 200 and wrong_type.json()["results"] == []
    assert deleted.status_code == 200 and deleted.json()["results"] == []
