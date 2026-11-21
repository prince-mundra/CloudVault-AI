from __future__ import annotations

from typing import Any

import httpx

from ai_service.config import Settings


class SummaryError(Exception):
    """A model request or summary output was unusable; safe to retry separately."""


class OllamaSummarizer:
    """Bounded local Ollama map/reduce summarizer for already-extracted chunks."""

    INPUT_CHAR_BUDGET = 6_000
    MAX_MAP_GROUPS = 512
    MAX_REDUCTION_ROUNDS = 8
    MAX_OUTPUT_CHARS = 4_000

    def __init__(self, settings: Settings, timeout_seconds: float = 180.0, client: Any | None = None) -> None:
        self.settings = settings
        self.timeout_seconds = timeout_seconds
        self.client = client

    @staticmethod
    def _pack(chunks: list[tuple[int, int | None, str]]) -> list[str]:
        groups: list[str] = []
        current: list[str] = []
        current_size = 0
        for index, page, text in chunks:
            if not text.strip():
                continue
            label = f"[chunk {index + 1}" + (f", page {page}" if page is not None else "") + "]\n"
            piece = label + text.strip()
            # Existing ingestion chunks are bounded, but split defensively if a
            # future extractor or migration supplies a larger stored chunk.
            while len(piece) > OllamaSummarizer.INPUT_CHAR_BUDGET:
                if current:
                    groups.append("\n\n".join(current))
                    current, current_size = [], 0
                groups.append(piece[: OllamaSummarizer.INPUT_CHAR_BUDGET])
                piece = piece[OllamaSummarizer.INPUT_CHAR_BUDGET :]
            if current and current_size + len(piece) + 2 > OllamaSummarizer.INPUT_CHAR_BUDGET:
                groups.append("\n\n".join(current))
                current, current_size = [], 0
            current.append(piece)
            current_size += len(piece) + (2 if len(current) > 1 else 0)
        if current:
            groups.append("\n\n".join(current))
        return groups

    async def _generate(self, text: str, reduce: bool) -> str:
        system = (
            "Write a concise, factual summary using only the supplied document text. "
            "Do not add facts, assumptions, or outside knowledge. Preserve important names, dates, figures, "
            "decisions, and qualifications. If the supplied material is fragmentary, state that plainly."
        )
        instruction = (
            "Synthesize these intermediate summaries into one coherent document summary. Remove repetition "
            "without losing material facts."
            if reduce
            else "Summarize the supplied document excerpt. Identify its main topic and material points."
        )
        payload = {
            "model": self.settings.summary_model,
            "stream": False,
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": f"{instruction}\n\n--- SOURCE ---\n{text}"},
            ],
            "options": {"temperature": 0.1, "num_ctx": 8192, "num_predict": 768},
        }
        try:
            if self.client is not None:
                response = await self.client.post(f"{self.settings.ollama_url}/api/chat", json=payload, timeout=self.timeout_seconds)
                response.raise_for_status()
                result = response.json()
            else:
                async with httpx.AsyncClient(timeout=self.timeout_seconds) as client:
                    response = await client.post(f"{self.settings.ollama_url}/api/chat", json=payload)
                    response.raise_for_status()
                    result = response.json()
        except (httpx.HTTPError, ValueError, TimeoutError) as exc:
            raise SummaryError("Local summarization model is unavailable") from exc
        if not isinstance(result, dict):
            raise SummaryError("Local summarization model returned an invalid response")
        message = result.get("message")
        content = message.get("content") if isinstance(message, dict) else None
        if not isinstance(content, str) or not content.strip() or len(content) > self.MAX_OUTPUT_CHARS:
            raise SummaryError("Local summarization model returned empty or oversized output")
        return content.strip()

    async def summarize(self, chunks: list[tuple[int, int | None, str]]) -> str:
        groups = self._pack(chunks)
        if not groups:
            raise SummaryError("Document has no text available to summarize")
        if len(groups) > self.MAX_MAP_GROUPS:
            raise SummaryError("Document exceeds the bounded summary segment limit")
        summaries = [await self._generate(group, reduce=False) for group in groups]
        rounds = 0
        while len(summaries) > 1:
            rounds += 1
            if rounds > self.MAX_REDUCTION_ROUNDS:
                raise SummaryError("Document summary exceeded the bounded reduction depth")
            packed = self._pack([(i, None, text) for i, text in enumerate(summaries)])
            if len(packed) >= len(summaries):
                raise SummaryError("Intermediate summaries did not reduce to a bounded final summary")
            summaries = [await self._generate(group, reduce=True) for group in packed]
        return summaries[0]
