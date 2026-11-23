from __future__ import annotations

import asyncio
import hashlib
import logging
from pathlib import Path, PurePosixPath
from typing import Any

from ai_service.config import Settings, from_env, validate_settings
from ai_service.db import create_pool
from ai_service.embedder import OllamaEmbedder
from ai_service.extract import DocumentProcessingError, chunk_pages, extract_document

logger = logging.getLogger("cloudvault.ai.worker")
BATCH_SIZE = 32


class IngestionWorker:
    def __init__(self, settings: Settings, pool: Any, embedder: Any) -> None:
        self.settings = settings
        self.pool = pool
        self.embedder = embedder

    async def claim_one(self) -> dict[str, Any] | None:
        sql = """
            WITH candidate AS (
              SELECT id FROM document_ingestion_jobs
              WHERE (state = 'queued' AND next_attempt_at <= NOW() AND attempts < %s)
                 OR (state = 'processing' AND locked_at < NOW() - INTERVAL '10 minutes' AND attempts < %s)
              ORDER BY next_attempt_at, created_at
              LIMIT 1
              FOR UPDATE SKIP LOCKED
            )
            UPDATE document_ingestion_jobs AS j
            SET state = 'processing', attempts = j.attempts + 1,
                locked_at = NOW(), started_at = COALESCE(j.started_at, NOW()),
                updated_at = NOW(), last_error = ''
            FROM candidate
            WHERE j.id = candidate.id
            RETURNING j.id, j.file_id, j.attempts
        """
        async with self.pool.connection() as conn, conn.transaction():
            await conn.execute(
                """UPDATE document_ingestion_jobs
                   SET state='failed', locked_at=NULL, completed_at=NOW(), updated_at=NOW(),
                       last_error='Worker lease expired after maximum attempts'
                   WHERE state='processing' AND locked_at < NOW() - INTERVAL '10 minutes' AND attempts >= %s""",
                (self.settings.max_attempts,),
            )
            cursor = await conn.execute(sql, (self.settings.max_attempts, self.settings.max_attempts))
            return await cursor.fetchone()

    def _resolve_object(self, object_key: str) -> Path:
        key = PurePosixPath(object_key)
        if not object_key or key.is_absolute() or any(part in {"", ".", ".."} for part in key.parts) or "\\" in object_key:
            raise DocumentProcessingError("Invalid stored object key")
        root = self.settings.storage_root.resolve(strict=True)
        path = (root / Path(*key.parts)).resolve(strict=True)
        if not path.is_relative_to(root) or not path.is_file():
            raise DocumentProcessingError("Stored document is unavailable")
        return path

    async def process(self, job: dict[str, Any]) -> None:
        job_id, file_id, attempt = job["id"], job["file_id"], job["attempts"]
        if attempt > self.settings.max_attempts:
            await self._mark_failed(job_id, attempt, "Worker lease expired after maximum attempts")
            return

        try:
            async with self.pool.connection() as conn:
                cursor = await conn.execute(
                    "SELECT id, path, name, content_type, size, sha256, state FROM files WHERE id = %s",
                    (file_id,),
                )
                file_row = await cursor.fetchone()
            if file_row is None:
                return  # The file's ON DELETE CASCADE removes its queue row as well.
            if file_row["state"] != "active":
                await self._mark_cancelled(job_id, attempt)
                return
            if file_row["size"] <= 0 or file_row["size"] > self.settings.max_document_bytes:
                raise DocumentProcessingError("Document size is outside the ingestion limit")

            path = self._resolve_object(file_row["path"])
            if path.stat().st_size != file_row["size"]:
                raise DocumentProcessingError("Stored document size does not match metadata")
            digest = hashlib.sha256()
            with path.open("rb") as source:
                for block in iter(lambda: source.read(1024 * 1024), b""):
                    digest.update(block)
            stored_hash = str(file_row["sha256"] or "").strip()
            if len(stored_hash) != 64 or any(char not in "0123456789abcdefABCDEF" for char in stored_hash):
                raise DocumentProcessingError("Stored document has no valid SHA-256 digest")
            if digest.hexdigest() != stored_hash.lower():
                raise DocumentProcessingError("Stored document integrity check failed")

            pages = await asyncio.to_thread(
                extract_document,
                path,
                file_row["content_type"],
                file_row["name"],
                self.settings.max_pdf_pages,
                self.settings.max_extracted_chars,
                self.settings.max_ocr_pages,
            )
            chunks = chunk_pages(pages, self.settings.chunk_size, self.settings.chunk_overlap)
            if not chunks or len(chunks) > 10000:
                raise DocumentProcessingError("Document produced no usable chunks or exceeded the chunk limit")

            vectors: list[list[float]] = []
            for start in range(0, len(chunks), BATCH_SIZE):
                texts = [chunk[2] for chunk in chunks[start : start + BATCH_SIZE]]
                vectors.extend(await self.embedder.embed(texts))
            if len(vectors) != len(chunks):
                raise RuntimeError("Embedding count does not match extracted chunk count")

            async with self.pool.connection() as conn, conn.transaction():
                cursor = await conn.execute(
                    "SELECT state, attempts FROM document_ingestion_jobs WHERE id = %s FOR UPDATE",
                    (job_id,),
                )
                lease = await cursor.fetchone()
                if lease is None or lease["state"] != "processing" or lease["attempts"] != attempt:
                    return
                cursor = await conn.execute("SELECT state FROM files WHERE id = %s FOR UPDATE", (file_id,))
                current = await cursor.fetchone()
                if current is None or current["state"] != "active":
                    await conn.execute(
                        "UPDATE document_ingestion_jobs SET state='cancelled', locked_at=NULL, updated_at=NOW() WHERE id=%s AND state='processing' AND attempts=%s",
                        (job_id, attempt),
                    )
                    return
                await conn.execute("DELETE FROM document_chunks WHERE file_id = %s", (file_id,))
                async with conn.cursor() as cur:
                    await cur.executemany(
                        """INSERT INTO document_chunks
                           (file_id, chunk_index, page_number, chunk_text, embedding, embedding_model)
                           VALUES (%s, %s, %s, %s, %s::vector, %s)""",
                        [
                            (file_id, chunk_index, page_number, text, self._vector_literal(vector), self.settings.ollama_model)
                            for (chunk_index, page_number, text), vector in zip(chunks, vectors, strict=True)
                        ],
                    )
                await conn.execute(
                    """UPDATE document_ingestion_jobs
                       SET state='succeeded', locked_at=NULL, completed_at=NOW(), updated_at=NOW(), last_error=''
                       WHERE id=%s AND state='processing' AND attempts=%s""",
                    (job_id, attempt),
                )
        except asyncio.CancelledError:
            raise
        except DocumentProcessingError as exc:
            logger.info("document ingestion rejected job=%s reason=%s", job_id, str(exc)[:120])
            await self._mark_failed(job_id, attempt, type(exc).__name__)
        except Exception as exc:  # noqa: BLE001 - retry unexpected DB, network, and model errors
            logger.warning("document ingestion retry job=%s error_type=%s", job_id, type(exc).__name__)
            await self._schedule_retry(job_id, attempt)

    @staticmethod
    def _vector_literal(vector: list[float]) -> str:
        return "[" + ",".join(format(value, ".9g") for value in vector) + "]"

    async def _mark_failed(self, job_id: int, attempt: int, reason: str) -> None:
        async with self.pool.connection() as conn:
            await conn.execute(
                """UPDATE document_ingestion_jobs SET state='failed', locked_at=NULL,
                   completed_at=NOW(), updated_at=NOW(), last_error=%s
                   WHERE id=%s AND state='processing' AND attempts=%s""",
                (reason[:160], job_id, attempt),
            )

    async def _mark_cancelled(self, job_id: int, attempt: int) -> None:
        async with self.pool.connection() as conn:
            await conn.execute(
                """UPDATE document_ingestion_jobs SET state='cancelled', locked_at=NULL,
                   completed_at=NOW(), updated_at=NOW() WHERE id=%s AND state='processing' AND attempts=%s""",
                (job_id, attempt),
            )

    async def _schedule_retry(self, job_id: int, attempt: int) -> None:
        state = "failed" if attempt >= self.settings.max_attempts else "queued"
        delay = min(3600, 5 * (2 ** min(attempt, 9)))
        async with self.pool.connection() as conn:
            await conn.execute(
                """UPDATE document_ingestion_jobs
                   SET state=%s, locked_at=NULL, completed_at=CASE WHEN %s='failed' THEN NOW() ELSE NULL END,
                       updated_at=NOW(), next_attempt_at=NOW() + make_interval(secs => %s),
                       last_error=%s
                       WHERE id=%s AND state='processing' AND attempts=%s""",
                (state, state, delay, "TransientProcessingError", job_id, attempt),
            )

    async def _keep_lease(self, job_id: int, attempt: int, task: asyncio.Task[None]) -> None:
        while not task.done():
            await asyncio.sleep(60)
            if task.done():
                return
            try:
                async with self.pool.connection() as conn:
                    await conn.execute(
                        "UPDATE document_ingestion_jobs SET locked_at=NOW(), updated_at=NOW() WHERE id=%s AND state='processing' AND attempts=%s",
                        (job_id, attempt),
                    )
            except Exception as exc:  # noqa: BLE001 - log heartbeat failures without killing an active job
                logger.warning("job heartbeat failed job=%s error_type=%s", job_id, type(exc).__name__)

    async def process_with_lease(self, job: dict[str, Any]) -> None:
        task = asyncio.create_task(self.process(job))
        heartbeat = asyncio.create_task(self._keep_lease(job["id"], job["attempts"], task))
        try:
            await task
        finally:
            heartbeat.cancel()
            try:
                await heartbeat
            except asyncio.CancelledError:
                pass

    async def run(self) -> None:
        while True:
            try:
                job = await self.claim_one()
            except asyncio.CancelledError:
                raise
            except Exception as exc:  # noqa: BLE001 - keep polling after transient PostgreSQL errors
                logger.warning("queue polling failed error_type=%s", type(exc).__name__)
                await asyncio.sleep(self.settings.poll_seconds)
                continue
            if job is None:
                await asyncio.sleep(self.settings.poll_seconds)
                continue
            try:
                await self.process_with_lease(job)
            except asyncio.CancelledError:
                raise
            except Exception as exc:  # noqa: BLE001 - recover the worker loop from unexpected job failures
                logger.warning("job loop recovered error_type=%s", type(exc).__name__)
                await asyncio.sleep(self.settings.poll_seconds)


async def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
    settings = from_env()
    validate_settings(settings)
    pool = create_pool(settings)
    await pool.open()
    await pool.wait()
    worker = IngestionWorker(settings, pool, OllamaEmbedder(settings))
    from ai_service.summarizer import OllamaSummarizer
    from ai_service.summary_worker import SummaryWorker

    summary_worker = SummaryWorker(settings, pool, OllamaSummarizer(settings))
    try:
        await asyncio.gather(worker.run(), summary_worker.run())
    finally:
        await pool.close()


if __name__ == "__main__":
    asyncio.run(main())
