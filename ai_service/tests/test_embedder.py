from __future__ import annotations

import asyncio
from pathlib import Path
from typing import Any, Self

import httpx
import pytest

from ai_service.config import Settings
from ai_service.embedder import EmbeddingError, OllamaEmbedder


class FakeResponse:
    def __init__(self, payload: Any) -> None:
        self.payload = payload

    def raise_for_status(self) -> None:
        return None

    def json(self) -> Any:
        return self.payload


class FakeAsyncClient:
    def __init__(self, payload: Any = None, error: Exception | None = None) -> None:
        self.payload = payload
        self.error = error

    async def __aenter__(self) -> Self:
        return self

    async def __aexit__(self, *_args: object) -> None:
        return None

    async def post(self, *_args: Any, **_kwargs: Any) -> FakeResponse:
        if self.error is not None:
            raise self.error
        return FakeResponse(self.payload)


def make_settings(tmp_path: Path) -> Settings:
    return Settings("db", 5432, "vault", "u", "p", "disable", "s" * 32, tmp_path, "http://ollama:11434", "nomic-embed-text", 768, 2, 5, 1000, 20, 10000, 1200, 150)


def test_embedding_response_must_be_json_object(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    import ai_service.embedder as module

    monkeypatch.setattr(module.httpx, "AsyncClient", lambda **_kwargs: FakeAsyncClient([{"embeddings": []}]))
    with pytest.raises(EmbeddingError, match="invalid response"):
        asyncio.run(OllamaEmbedder(make_settings(tmp_path)).embed(["query"]))


def test_embedding_rejects_non_numeric_vector_values(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    import ai_service.embedder as module

    payload = {"embeddings": [["not-a-number"] + [0.1] * 767]}
    monkeypatch.setattr(module.httpx, "AsyncClient", lambda **_kwargs: FakeAsyncClient(payload))
    with pytest.raises(EmbeddingError, match="non-numeric"):
        asyncio.run(OllamaEmbedder(make_settings(tmp_path)).embed(["query"]))


def test_embedding_rejects_nonfinite_vectors(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    import ai_service.embedder as module

    payload = {"embeddings": [[float("nan")] + [0.1] * 767]}
    monkeypatch.setattr(module.httpx, "AsyncClient", lambda **_kwargs: FakeAsyncClient(payload))
    with pytest.raises(EmbeddingError, match="non-finite"):
        asyncio.run(OllamaEmbedder(make_settings(tmp_path)).embed(["query"]))


def test_embedding_maps_unavailable_ollama_to_embedding_error(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    import ai_service.embedder as module

    monkeypatch.setattr(module.httpx, "AsyncClient", lambda **_kwargs: FakeAsyncClient(error=httpx.ConnectError("offline")))
    with pytest.raises(EmbeddingError, match="unavailable"):
        asyncio.run(OllamaEmbedder(make_settings(tmp_path)).embed(["query"]))
