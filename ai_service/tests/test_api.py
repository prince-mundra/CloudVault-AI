from __future__ import annotations

import asyncio
import importlib
import json
import time
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any

import httpx

from ai_service.config import Settings
from ai_service.security import request_signature


class FakeCursor:
    async def fetchall(self) -> list[dict[str, Any]]:
        return [{"file_id": "f1", "file_name": "notes.txt", "chunk_index": 0, "page_number": 1, "chunk_text": "authorized text", "score": 0.91}]


class FakeConnection:
    def __init__(self) -> None:
        self.query = ""
        self.params: tuple[Any, ...] = ()

    async def execute(self, query: str, params: tuple[Any, ...] = ()) -> FakeCursor:
        self.query, self.params = query, params
        return FakeCursor()


class FakePool:
    def __init__(self) -> None:
        self.conn = FakeConnection()

    @asynccontextmanager
    async def connection(self):
        yield self.conn


class FakeEmbedder:
    async def embed(self, _texts: list[str]) -> list[list[float]]:
        return [([0.01] * 768)]


def test_internal_search_requires_hmac_and_uses_access_scope(monkeypatch, tmp_path: Path) -> None:
    secret = "phase-two-shared-secret-32-byte-min"
    monkeypatch.setenv("AI_SHARED_SECRET", secret)
    api_module = importlib.import_module("ai_service.api")
    settings = Settings("db", 5432, "vault", "u", "p", "disable", secret, tmp_path, "http://ollama:11434", "nomic-embed-text", 768, 2, 5, 1000, 20, 10000, 1200, 150)
    pool = FakePool()
    app = api_module.create_app(settings=settings, pool=pool, embedder=FakeEmbedder())

    async def exercise() -> None:
        transport = httpx.ASGITransport(app=app)
        async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
            payload = {"user_id": "user-1", "query": "find report", "limit": 5, "content_type": "application/pdf"}
            body = json.dumps(payload, separators=(",", ":")).encode()
            timestamp = str(int(time.time()))
            headers = {
                "Content-Type": "application/json",
                "X-CloudVault-Timestamp": timestamp,
                "X-CloudVault-Signature": request_signature(secret, timestamp, "POST", "/internal/v1/search", body),
            }
            response = await client.post("/internal/v1/search", content=body, headers=headers)
            assert response.status_code == 200
            assert response.json()["results"][0]["file_id"] == "f1"
            assert "p.can_read = TRUE" in pool.conn.query
            assert "f.owner_id = %s" in pool.conn.query
            assert pool.conn.params[1:3] == ("user-1", "user-1")
            assert "f.state = 'active'" in pool.conn.query
            assert "c.embedding_model = %s" in pool.conn.query
            assert "left(f.content_type" in pool.conn.query
            assert pool.conn.params[3:7] == ("nomic-embed-text", "application/pdf", "application/pdf", "application/pdf")
            headers["X-CloudVault-Signature"] = "0" * 64
            rejected = await client.post("/internal/v1/search", content=body, headers=headers)
            assert rejected.status_code == 401
            invalid_payload = {**payload, "content_type": "application/%"}
            invalid_body = json.dumps(invalid_payload, separators=(",", ":")).encode()
            headers["X-CloudVault-Signature"] = request_signature(secret, timestamp, "POST", "/internal/v1/search", invalid_body)
            invalid = await client.post("/internal/v1/search", content=invalid_body, headers=headers)
            assert invalid.status_code == 422
            oversized = await client.post("/internal/v1/search", content=b"x" * (16 * 1024 + 1), headers={"Content-Type": "application/json"})
            assert oversized.status_code == 413

    asyncio.run(exercise())


class FakeOllamaClient:
    def __init__(self, reply_content: str = "Based on the report [1], the revenue grew 24%.") -> None:
        self.reply_content = reply_content
        self.last_payload: dict[str, Any] = {}

    async def post(self, url: str, json: dict[str, Any], timeout: float = 120.0) -> Any:
        self.last_payload = json
        class _Resp:
            def raise_for_status(self): pass
            def json(self):
                return {"message": {"role": "assistant", "content": FakeOllamaClient.this.reply_content}}
        FakeOllamaClient.this = self
        return _Resp()


def test_internal_chat_rag_grounding_and_citations(monkeypatch, tmp_path: Path) -> None:
    secret = "phase-two-shared-secret-32-byte-min"
    monkeypatch.setenv("AI_SHARED_SECRET", secret)
    api_module = importlib.import_module("ai_service.api")
    settings = Settings("db", 5432, "vault", "u", "p", "disable", secret, tmp_path, "http://ollama:11434", "nomic-embed-text", 768, 2, 5, 1000, 20, 10000, 1200, 150)
    pool = FakePool()
    ollama_mock = FakeOllamaClient("Based on the data [1], growth was strong.")
    app = api_module.create_app(settings=settings, pool=pool, embedder=FakeEmbedder(), http_client=ollama_mock)

    async def exercise() -> None:
        transport = httpx.ASGITransport(app=app)
        async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
            payload = {
                "user_id": "user-42",
                "message": "What is the revenue growth?",
                "history": [{"role": "user", "content": "hello"}, {"role": "assistant", "content": "hi"}],
                "file_ids": ["f1"],
            }
            body = json.dumps(payload, separators=(",", ":")).encode()
            timestamp = str(int(time.time()))
            headers = {
                "Content-Type": "application/json",
                "X-CloudVault-Timestamp": timestamp,
                "X-CloudVault-Signature": request_signature(secret, timestamp, "POST", "/internal/v1/chat", body),
            }
            response = await client.post("/internal/v1/chat", content=body, headers=headers)
            assert response.status_code == 200
            data = response.json()
            assert "growth was strong" in data["answer"]
            assert len(data["citations"]) == 1
            cit = data["citations"][0]
            assert cit["file_id"] == "f1"
            assert cit["file_name"] == "notes.txt"
            assert cit["page_number"] == 1
            assert cit["chunk_index"] == 0
            assert cit["score"] == 0.91

            # Check isolation query was executed
            assert "f.state = 'active'" in pool.conn.query
            assert "p.can_read = TRUE" in pool.conn.query
            assert "f.owner_id = %s" in pool.conn.query
            assert pool.conn.params[1:3] == ("user-42", "user-42")
            assert pool.conn.params[4] is True  # has_file_filter
            assert pool.conn.params[5] == ["f1"]  # file_ids_list

            # Check Ollama payload structure
            assert len(ollama_mock.last_payload["messages"]) >= 3
            assert ollama_mock.last_payload["messages"][0]["role"] == "system"

            # Check threshold abstention
            class LowScoreConnection:
                async def execute(self, query: str, params: tuple[Any, ...] = ()) -> Any:
                    class _Cursor:
                        async def fetchall(self):
                            return [{"file_id": "f1", "file_name": "notes.txt", "chunk_index": 0, "page_number": 1, "chunk_text": "irrelevant", "score": 0.15}]
                    return _Cursor()

            pool.conn = LowScoreConnection()
            low_score_resp = await client.post("/internal/v1/chat", content=body, headers=headers)
            assert low_score_resp.status_code == 200
            assert "could not find enough relevant information" in low_score_resp.json()["answer"]
            assert low_score_resp.json()["citations"] == []

    asyncio.run(exercise())
