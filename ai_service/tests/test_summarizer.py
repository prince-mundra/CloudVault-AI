from __future__ import annotations

import asyncio
from pathlib import Path
from typing import Any

import httpx
import pytest

from ai_service.config import Settings
from ai_service.summarizer import OllamaSummarizer, SummaryError


def settings(root: Path) -> Settings:
    return Settings(
        db_host="db", db_port=5432, db_name="vault", db_user="user", db_password="password",
        db_sslmode="disable", internal_secret="s" * 32, storage_root=root,
        ollama_url="http://ollama:11434", ollama_model="nomic-embed-text",
        embedding_dimension=768, poll_seconds=1, max_attempts=3, max_document_bytes=100_000,
        max_pdf_pages=20, max_extracted_chars=50_000, chunk_size=1200, chunk_overlap=150,
        max_ocr_pages=5, summary_model="llama3.2:3b",
    )


def test_short_summary_uses_separate_local_generation_model(tmp_path: Path) -> None:
    calls: list[httpx.Request] = []

    def handler(request: httpx.Request) -> httpx.Response:
        calls.append(request)
        return httpx.Response(200, json={"message": {"role": "assistant", "content": "A factual summary."}})

    async def exercise() -> str:
        async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
            summarizer = OllamaSummarizer(settings(tmp_path), client=client)
            return await summarizer.summarize([(0, 1, "The report records a quarterly revenue of 42.")])

    result = asyncio.run(exercise())
    assert result == "A factual summary."
    assert len(calls) == 1
    assert calls[0].url == "http://ollama:11434/api/chat"
    payload = calls[0].read().decode()
    assert '"model":"llama3.2:3b"' in payload
    assert "nomic-embed-text" not in payload


def test_long_summary_splits_and_reduces_intermediate_summaries(tmp_path: Path) -> None:
    calls: list[dict[str, Any]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        import json

        payload = json.loads(request.content)
        calls.append(payload)
        return httpx.Response(200, json={"message": {"content": f"Summary {len(calls)}"}})

    async def exercise() -> str:
        async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
            summarizer = OllamaSummarizer(settings(tmp_path), client=client)
            # One unusually large persisted chunk proves the summary layer applies
            # its own input bound rather than relying on the extraction chunk size.
            return await summarizer.summarize([(0, 1, "x" * (OllamaSummarizer.INPUT_CHAR_BUDGET + 1))])

    result = asyncio.run(exercise())
    assert result == "Summary 3"
    assert len(calls) == 3  # two bounded map calls plus one reduce call
    source_inputs = [call["messages"][1]["content"].split("--- SOURCE ---\n", 1)[1] for call in calls]
    assert all(len(text) <= OllamaSummarizer.INPUT_CHAR_BUDGET for text in source_inputs)
    assert calls[-1]["messages"][1]["content"].startswith("Synthesize these intermediate summaries")


def test_empty_input_is_rejected_without_model_call(tmp_path: Path) -> None:
    async def exercise() -> None:
        summarizer = OllamaSummarizer(settings(tmp_path))
        with pytest.raises(SummaryError, match="no text"):
            await summarizer.summarize([])

    asyncio.run(exercise())


def test_malformed_or_empty_ollama_output_is_typed_error(tmp_path: Path) -> None:
    def handler(_request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"message": {"content": "  "}})

    async def exercise() -> None:
        async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
            summarizer = OllamaSummarizer(settings(tmp_path), client=client)
            with pytest.raises(SummaryError, match="empty or oversized"):
                await summarizer.summarize([(0, None, "source text")])

    asyncio.run(exercise())


def test_http_failure_is_typed_error(tmp_path: Path) -> None:
    def handler(_request: httpx.Request) -> httpx.Response:
        return httpx.Response(503)

    async def exercise() -> None:
        async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
            summarizer = OllamaSummarizer(settings(tmp_path), client=client)
            with pytest.raises(SummaryError, match="unavailable"):
                await summarizer.summarize([(0, None, "source text")])

    asyncio.run(exercise())
