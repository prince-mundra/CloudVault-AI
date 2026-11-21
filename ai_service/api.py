from __future__ import annotations

import re
from contextlib import asynccontextmanager
from typing import Any

from fastapi import Depends, FastAPI, HTTPException
from pydantic import BaseModel, Field
from starlette.responses import JSONResponse

from ai_service.config import Settings, from_env, validate_settings
from ai_service.db import create_pool
from ai_service.embedder import EmbeddingError, OllamaEmbedder
from ai_service.security import require_service_signature


class SearchRequest(BaseModel):
    user_id: str = Field(min_length=1, max_length=128)
    query: str = Field(min_length=2, max_length=1000)
    limit: int = Field(default=10, ge=1, le=20)
    content_type: str = Field(default="", max_length=128)


_CONTENT_TYPE_PREFIX = re.compile(r"[A-Za-z0-9!#$&^_.+-]+/(?:[A-Za-z0-9!#$&^_.+-]*)\Z")


class _RequestBodyTooLarge(Exception):
    pass


class SearchBodyLimitMiddleware:
    def __init__(self, app: Any, max_bytes: int = 16 * 1024) -> None:
        self.app = app
        self.max_bytes = max_bytes

    async def __call__(self, scope: dict[str, Any], receive: Any, send: Any) -> None:
        if scope.get("type") != "http" or scope.get("path") != "/internal/v1/search" or scope.get("method") != "POST":
            await self.app(scope, receive, send)
            return
        headers = dict(scope.get("headers", []))
        content_length = headers.get(b"content-length")
        if content_length is not None:
            try:
                declared_length = int(content_length)
            except ValueError:
                await JSONResponse(status_code=400, content={"detail": "Invalid content length"})(scope, receive, send)
                return
            if declared_length < 0 or declared_length > self.max_bytes:
                await JSONResponse(status_code=413, content={"detail": "Request body too large"})(scope, receive, send)
                return

        received_bytes = 0

        async def limited_receive() -> dict[str, Any]:
            nonlocal received_bytes
            message = await receive()
            if message.get("type") == "http.request":
                received_bytes += len(message.get("body", b""))
                if received_bytes > self.max_bytes:
                    raise _RequestBodyTooLarge
            return message

        try:
            await self.app(scope, limited_receive, send)
        except _RequestBodyTooLarge:
            await JSONResponse(status_code=413, content={"detail": "Request body too large"})(scope, receive, send)


class SearchHit(BaseModel):
    file_id: str
    file_name: str
    chunk_index: int
    page_number: int | None
    text: str
    score: float


class SearchResponse(BaseModel):
    results: list[SearchHit]


def create_app(settings: Settings | None = None, pool: Any | None = None, embedder: Any | None = None) -> FastAPI:
    cfg = settings or from_env()
    validate_settings(cfg)
    owns_pool = pool is None
    db_pool = pool or create_pool(cfg)
    local_embedder = embedder or OllamaEmbedder(cfg)

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        if owns_pool:
            await db_pool.open()
            await db_pool.wait()
        yield
        if owns_pool:
            await db_pool.close()

    app = FastAPI(title="CloudVault AI internal service", version="0.1.0", docs_url=None, redoc_url=None, lifespan=lifespan)
    app.add_middleware(SearchBodyLimitMiddleware, max_bytes=16 * 1024)
    app.state.settings = cfg
    app.state.pool = db_pool
    app.state.embedder = local_embedder

    @app.get("/healthz")
    async def health() -> dict[str, str]:
        try:
            async with db_pool.connection() as conn:
                await conn.execute("SELECT 1")
        except Exception:  # noqa: BLE001 - keep database details out of the health response
            raise HTTPException(status_code=503, detail="AI service unhealthy") from None
        return {"status": "ok"}

    @app.post("/internal/v1/search", response_model=SearchResponse, dependencies=[Depends(require_service_signature)])
    async def semantic_search(body: SearchRequest) -> SearchResponse:
        if body.content_type and not _CONTENT_TYPE_PREFIX.fullmatch(body.content_type):
            raise HTTPException(status_code=422, detail="Invalid content_type prefix")
        try:
            query_vector = (await local_embedder.embed([body.query]))[0]
            vector_literal = "[" + ",".join(format(value, ".9g") for value in query_vector) + "]"
            sql = """
                SELECT c.file_id, f.name AS file_name, c.chunk_index, c.page_number,
                       c.chunk_text, 1 - (c.embedding <=> %s::vector) AS score
                FROM document_chunks AS c
                JOIN files AS f ON f.id = c.file_id
                WHERE f.state = 'active'
                  AND (
                    f.owner_id = %s
                    OR EXISTS (
                      SELECT 1 FROM permissions AS p
                      WHERE p.file_id = f.id AND p.user_id = %s AND p.can_read = TRUE
                    )
                  )
                  AND c.embedding_model = %s
                  AND (%s = '' OR left(f.content_type, char_length(%s)) = %s)
                ORDER BY c.embedding <=> %s::vector
                LIMIT %s
            """
            async with db_pool.connection() as conn:
                cursor = await conn.execute(
                    sql,
                    (
                        vector_literal,
                        body.user_id,
                        body.user_id,
                        cfg.ollama_model,
                        body.content_type,
                        body.content_type,
                        body.content_type,
                        vector_literal,
                        body.limit,
                    ),
                )
                rows = await cursor.fetchall()
        except EmbeddingError:
            raise HTTPException(status_code=503, detail="Local embedding model unavailable") from None
        except Exception:  # noqa: BLE001 - keep internal retrieval details out of HTTP responses
            raise HTTPException(status_code=503, detail="Semantic retrieval unavailable") from None

        hits = [
            SearchHit(
                file_id=row["file_id"],
                file_name=row["file_name"],
                chunk_index=row["chunk_index"],
                page_number=row["page_number"],
                text=row["chunk_text"],
                score=float(row["score"]),
            )
            for row in rows
        ]
        return SearchResponse(results=hits)

    return app


app = create_app()
