from __future__ import annotations

import re
from contextlib import asynccontextmanager
from typing import Any

import httpx
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


class ChatHistoryItem(BaseModel):
    role: str = Field(pattern=r"^(user|assistant)$")
    content: str = Field(min_length=1, max_length=4000)


class ChatRequest(BaseModel):
    user_id: str = Field(min_length=1, max_length=128)
    message: str = Field(min_length=1, max_length=4000)
    history: list[ChatHistoryItem] = Field(default_factory=list, max_length=20)
    file_ids: list[str] = Field(default_factory=list, max_length=50)


class CitationItem(BaseModel):
    source_number: int
    file_id: str
    file_name: str
    chunk_index: int
    page_number: int | None
    chunk_text: str
    score: float


class ChatResponse(BaseModel):
    answer: str
    citations: list[CitationItem]


_CONTENT_TYPE_PREFIX = re.compile(r"[A-Za-z0-9!#$&^_.+-]+/(?:[A-Za-z0-9!#$&^_.+-]*)\Z")


class _RequestBodyTooLarge(Exception):
    pass


class InternalBodyLimitMiddleware:
    def __init__(self, app: Any) -> None:
        self.app = app

    async def __call__(self, scope: dict[str, Any], receive: Any, send: Any) -> None:
        if scope.get("type") != "http" or scope.get("method") != "POST":
            await self.app(scope, receive, send)
            return

        path = scope.get("path", "")
        if path == "/internal/v1/search":
            max_bytes = 16 * 1024
        elif path == "/internal/v1/chat":
            max_bytes = 64 * 1024
        else:
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
            if declared_length < 0 or declared_length > max_bytes:
                await JSONResponse(status_code=413, content={"detail": "Request body too large"})(scope, receive, send)
                return

        received_bytes = 0

        async def limited_receive() -> dict[str, Any]:
            nonlocal received_bytes
            message = await receive()
            if message.get("type") == "http.request":
                received_bytes += len(message.get("body", b""))
                if received_bytes > max_bytes:
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


def create_app(
    settings: Settings | None = None,
    pool: Any | None = None,
    embedder: Any | None = None,
    http_client: Any | None = None,
) -> FastAPI:
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
    app.add_middleware(InternalBodyLimitMiddleware)
    app.state.settings = cfg
    app.state.pool = db_pool
    app.state.embedder = local_embedder
    app.state.http_client = http_client

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

    @app.post("/internal/v1/chat", response_model=ChatResponse, dependencies=[Depends(require_service_signature)])
    async def rag_chat(body: ChatRequest) -> ChatResponse:
        try:
            query_vector = (await local_embedder.embed([body.message]))[0]
            vector_literal = "[" + ",".join(format(value, ".9g") for value in query_vector) + "]"
            has_file_filter = len(body.file_ids) > 0
            file_ids_list = body.file_ids if has_file_filter else [""]

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
                  AND (%s = FALSE OR c.file_id = ANY(%s))
                ORDER BY c.embedding <=> %s::vector
                LIMIT 5
            """
            async with db_pool.connection() as conn:
                cursor = await conn.execute(
                    sql,
                    (
                        vector_literal,
                        body.user_id,
                        body.user_id,
                        cfg.ollama_model,
                        has_file_filter,
                        file_ids_list,
                        vector_literal,
                    ),
                )
                rows = await cursor.fetchall()
        except EmbeddingError:
            raise HTTPException(status_code=503, detail="Local embedding model unavailable") from None
        except Exception:  # noqa: BLE001 - keep internal retrieval details out of HTTP responses
            raise HTTPException(status_code=503, detail="Semantic retrieval unavailable") from None

        if not rows:
            return ChatResponse(
                answer="I could not find enough relevant information in your accessible documents to answer this question.",
                citations=[],
            )

        top_score = float(rows[0]["score"])
        if top_score < cfg.rag_similarity_threshold:
            return ChatResponse(
                answer="I could not find enough relevant information in your accessible documents to answer this question.",
                citations=[],
            )

        context_blocks: list[str] = []
        all_citations: list[CitationItem] = []
        for idx, row in enumerate(rows, start=1):
            page_str = f"Page {row['page_number']}" if row["page_number"] is not None else "Page N/A"
            header = f"--- SOURCE [{idx}] (File: {row['file_name']}, {page_str}) ---"
            context_blocks.append(f"{header}\n{str(row['chunk_text']).strip()}")
            all_citations.append(
                CitationItem(
                    source_number=idx,
                    file_id=row["file_id"],
                    file_name=row["file_name"],
                    chunk_index=int(row["chunk_index"]),
                    page_number=row["page_number"],
                    chunk_text=str(row["chunk_text"]),
                    score=round(float(row["score"]), 4),
                )
            )

        sources_text = "\n\n".join(context_blocks)
        system_instruction = (
            "You are CloudVault Assistant, an AI assistant answering questions based solely on the provided document excerpts.\n"
            "Rules:\n"
            "1. Answer the question truthfully and concisely using ONLY the provided sources.\n"
            "2. Support every factual statement with its source citation number in brackets, e.g. [1] or [2].\n"
            "3. If the provided sources do not contain sufficient evidence to answer the question, state: "
            "'I could not find enough relevant information in your accessible documents to answer this question.'\n"
            "4. Never invent facts, filenames, page numbers, or citations not present in the sources."
        )

        messages: list[dict[str, str]] = [{"role": "system", "content": system_instruction}]
        for h in body.history[-10:]:
            messages.append({"role": h.role, "content": h.content})
        messages.append({"role": "user", "content": f"Sources:\n{sources_text}\n\nQuestion: {body.message}\n\nAnswer:"})

        payload = {
            "model": cfg.summary_model,
            "stream": False,
            "messages": messages,
            "options": {"temperature": 0.1, "num_ctx": 8192, "num_predict": 1024},
        }

        try:
            if http_client is not None:
                response = await http_client.post(f"{cfg.ollama_url}/api/chat", json=payload, timeout=120.0)
                response.raise_for_status()
                result = response.json()
            else:
                async with httpx.AsyncClient(timeout=120.0) as client:
                    response = await client.post(f"{cfg.ollama_url}/api/chat", json=payload)
                    response.raise_for_status()
                    result = response.json()
        except (httpx.HTTPError, ValueError, TimeoutError):
            raise HTTPException(status_code=503, detail="Local AI generation model unavailable") from None

        if not isinstance(result, dict):
            raise HTTPException(status_code=503, detail="Local AI generation model returned invalid response")

        message_data = result.get("message")
        content = message_data.get("content") if isinstance(message_data, dict) else None
        if not isinstance(content, str) or not content.strip():
            raise HTTPException(status_code=503, detail="Local AI generation model returned empty output")

        answer_text = content.strip()
        abstention_phrases = [
            "could not find enough relevant information",
            "not enough information",
            "does not contain enough information",
            "cannot answer based on the provided",
        ]
        if any(phrase in answer_text.lower() for phrase in abstention_phrases):
            return ChatResponse(answer=answer_text, citations=[])

        cited_nums = set(int(m) for m in re.findall(r"\[(\d+)\]", answer_text))
        if cited_nums:
            active_citations = [c for c in all_citations if c.source_number in cited_nums]
        else:
            active_citations = all_citations[:3]

        return ChatResponse(answer=answer_text, citations=active_citations)

    return app


app = create_app()
