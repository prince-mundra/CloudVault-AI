from __future__ import annotations

import math

import httpx

from ai_service.config import Settings


class EmbeddingError(Exception):
    pass


class OllamaEmbedder:
    def __init__(self, settings: Settings, timeout_seconds: float = 60.0) -> None:
        self.settings = settings
        self.timeout_seconds = timeout_seconds

    async def embed(self, texts: list[str]) -> list[list[float]]:
        if not texts:
            return []
        try:
            async with httpx.AsyncClient(timeout=self.timeout_seconds) as client:
                response = await client.post(
                    f"{self.settings.ollama_url}/api/embed",
                    json={"model": self.settings.ollama_model, "input": texts, "truncate": False},
                )
                response.raise_for_status()
                payload = response.json()
        except (httpx.HTTPError, ValueError) as exc:
            raise EmbeddingError("Local embedding model is unavailable") from exc

        if not isinstance(payload, dict):
            raise EmbeddingError("Local embedding model returned an invalid response")
        embeddings = payload.get("embeddings")
        if not isinstance(embeddings, list) or len(embeddings) != len(texts):
            raise EmbeddingError("Local embedding model returned an invalid response")
        result: list[list[float]] = []
        for vector in embeddings:
            if not isinstance(vector, list) or len(vector) != self.settings.embedding_dimension:
                raise EmbeddingError("Local embedding dimension does not match the database schema")
            try:
                values = [float(value) for value in vector]
            except (TypeError, ValueError, OverflowError) as exc:
                raise EmbeddingError("Local embedding model returned a non-numeric vector") from exc
            if not all(math.isfinite(value) for value in values):
                raise EmbeddingError("Local embedding model returned non-finite values")
            result.append(values)
        return result
