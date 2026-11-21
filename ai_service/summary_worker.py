from __future__ import annotations

import asyncio
import logging
import re
from typing import Any

from ai_service.config import Settings, from_env, validate_settings
from ai_service.db import create_pool
from ai_service.summarizer import OllamaSummarizer, SummaryError

logger = logging.getLogger("cloudvault.ai.summary_worker")
_SHA256 = re.compile(r"^[0-9a-fA-F]{64}$")


class SummaryWorker:
    """Independent durable worker; summary failures never alter ingestion jobs."""

    def __init__(self, settings: Settings, pool: Any, summarizer: Any) -> None:
        self.settings = settings
        self.pool = pool
        self.summarizer = summarizer

    async def claim_one(self) -> dict[str, Any] | None:
        sql = """
            WITH candidate AS (
              SELECT sj.id
              FROM document_summary_jobs AS sj
              JOIN document_ingestion_jobs AS ij ON ij.file_id=sj.file_id AND ij.state='succeeded'
              JOIN files AS f ON f.id=sj.file_id AND f.state='active'
              WHERE (sj.state='queued' AND sj.next_attempt_at <= NOW() AND sj.attempts < %s)
                 OR (sj.state='processing' AND sj.locked_at < NOW() - INTERVAL '10 minutes' AND sj.attempts < %s)
              ORDER BY sj.next_attempt_at, sj.created_at
              LIMIT 1
              FOR UPDATE OF sj SKIP LOCKED
            )
            UPDATE document_summary_jobs AS sj
            SET state='processing', attempts=sj.attempts+1,
                locked_at=NOW(), started_at=COALESCE(sj.started_at,NOW()),
                updated_at=NOW(), last_error=''
            FROM candidate
            WHERE sj.id=candidate.id
            RETURNING sj.id,sj.file_id,sj.attempts
        """
        async with self.pool.connection() as conn, conn.transaction():
            await conn.execute(
                """UPDATE document_summary_jobs AS sj
                   SET state=CASE WHEN ij.state='cancelled' OR f.state<>'active' THEN 'cancelled' ELSE 'failed' END,
                       locked_at=NULL, completed_at=NOW(), updated_at=NOW(),
                       last_error=CASE WHEN ij.state IN ('failed','cancelled') THEN 'IngestionNotSuccessful' ELSE 'FileUnavailable' END
                   FROM document_ingestion_jobs AS ij JOIN files AS f ON f.id=ij.file_id
                   WHERE sj.file_id=ij.file_id AND sj.state='queued'
                     AND (ij.state IN ('failed','cancelled') OR f.state<>'active')"""
            )
            await conn.execute(
                """UPDATE document_summary_jobs
                   SET state='failed', locked_at=NULL, completed_at=NOW(), updated_at=NOW(),
                       last_error='Worker lease expired after maximum attempts'
                   WHERE state='processing' AND locked_at < NOW() - INTERVAL '10 minutes' AND attempts >= %s""",
                (self.settings.max_attempts,),
            )
            cursor = await conn.execute(sql, (self.settings.max_attempts, self.settings.max_attempts))
            return await cursor.fetchone()

    async def process(self, job: dict[str, Any]) -> None:
        job_id, file_id, attempt = job["id"], job["file_id"], job["attempts"]
        try:
            async with self.pool.connection() as conn:
                cursor = await conn.execute("SELECT state,sha256 FROM files WHERE id=%s", (file_id,))
                file_row = await cursor.fetchone()
                if file_row is None or file_row["state"] != "active":
                    await self._mark_cancelled(job_id, attempt)
                    return
                digest = str(file_row["sha256"] or "").strip()
                if not _SHA256.fullmatch(digest):
                    raise SummaryError("Document source digest is invalid")
                cursor = await conn.execute(
                    "SELECT chunk_index,page_number,chunk_text FROM document_chunks WHERE file_id=%s ORDER BY chunk_index",
                    (file_id,),
                )
                rows = await cursor.fetchall()
            chunks = [(int(row["chunk_index"]), row["page_number"], str(row["chunk_text"])) for row in rows]
            if not chunks:
                raise SummaryError("Ingested document chunks are unavailable")
            summary = await self.summarizer.summarize(chunks)
            if not isinstance(summary, str) or not summary.strip():
                raise SummaryError("Summarizer returned no usable text")

            async with self.pool.connection() as conn, conn.transaction():
                cursor = await conn.execute(
                    "SELECT state,attempts FROM document_summary_jobs WHERE id=%s FOR UPDATE",
                    (job_id,),
                )
                lease = await cursor.fetchone()
                if lease is None or lease["state"] != "processing" or lease["attempts"] != attempt:
                    return
                cursor = await conn.execute("SELECT state,sha256 FROM files WHERE id=%s FOR UPDATE", (file_id,))
                current = await cursor.fetchone()
                if current is None or current["state"] != "active" or str(current["sha256"] or "").strip().lower() != digest.lower():
                    await conn.execute(
                        "UPDATE document_summary_jobs SET state='cancelled',locked_at=NULL,completed_at=NOW(),updated_at=NOW() WHERE id=%s AND state='processing' AND attempts=%s",
                        (job_id, attempt),
                    )
                    return
                await conn.execute(
                    """INSERT INTO document_summaries(file_id,summary_text,summary_model,source_sha256)
                       VALUES(%s,%s,%s,%s)
                       ON CONFLICT(file_id) DO UPDATE SET summary_text=EXCLUDED.summary_text,
                         summary_model=EXCLUDED.summary_model,source_sha256=EXCLUDED.source_sha256,updated_at=NOW()""",
                    (file_id, summary.strip(), self.settings.summary_model, digest.lower()),
                )
                await conn.execute(
                    """UPDATE document_summary_jobs SET state='succeeded',locked_at=NULL,
                       completed_at=NOW(),updated_at=NOW(),last_error=''
                       WHERE id=%s AND state='processing' AND attempts=%s""",
                    (job_id, attempt),
                )
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # noqa: BLE001 - retry model/network/database failures independently
            logger.warning("document summary retry job=%s error_type=%s", job_id, type(exc).__name__)
            await self._schedule_retry(job_id, attempt)

    async def _mark_cancelled(self, job_id: int, attempt: int) -> None:
        async with self.pool.connection() as conn:
            await conn.execute(
                "UPDATE document_summary_jobs SET state='cancelled',locked_at=NULL,completed_at=NOW(),updated_at=NOW() WHERE id=%s AND state='processing' AND attempts=%s",
                (job_id, attempt),
            )

    async def _schedule_retry(self, job_id: int, attempt: int) -> None:
        state = "failed" if attempt >= self.settings.max_attempts else "queued"
        delay = min(3600, 5 * (2 ** min(attempt, 9)))
        async with self.pool.connection() as conn:
            await conn.execute(
                """UPDATE document_summary_jobs SET state=%s,locked_at=NULL,
                   completed_at=CASE WHEN %s='failed' THEN NOW() ELSE NULL END,
                   updated_at=NOW(),next_attempt_at=NOW()+make_interval(secs => %s),
                   last_error=%s WHERE id=%s AND state='processing' AND attempts=%s""",
                (state, state, delay, "SummaryProcessingError", job_id, attempt),
            )

    async def _keep_lease(self, job_id: int, attempt: int, task: asyncio.Task[None]) -> None:
        while not task.done():
            await asyncio.sleep(60)
            if task.done():
                return
            try:
                async with self.pool.connection() as conn:
                    await conn.execute(
                        "UPDATE document_summary_jobs SET locked_at=NOW(),updated_at=NOW() WHERE id=%s AND state='processing' AND attempts=%s",
                        (job_id, attempt),
                    )
            except Exception as exc:  # noqa: BLE001
                logger.warning("summary heartbeat failed job=%s error_type=%s", job_id, type(exc).__name__)

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
            except Exception as exc:  # noqa: BLE001
                logger.warning("summary queue polling failed error_type=%s", type(exc).__name__)
                await asyncio.sleep(self.settings.poll_seconds)
                continue
            if job is None:
                await asyncio.sleep(self.settings.poll_seconds)
                continue
            try:
                await self.process_with_lease(job)
            except asyncio.CancelledError:
                raise
            except Exception as exc:  # noqa: BLE001 - recover the summary loop after an unexpected job failure
                logger.warning("summary job loop recovered error_type=%s", type(exc).__name__)
                await asyncio.sleep(self.settings.poll_seconds)


async def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
    settings = from_env()
    validate_settings(settings)
    pool = create_pool(settings)
    await pool.open()
    await pool.wait()
    worker = SummaryWorker(settings, pool, OllamaSummarizer(settings))
    try:
        await worker.run()
    finally:
        await pool.close()


if __name__ == "__main__":
    asyncio.run(main())
