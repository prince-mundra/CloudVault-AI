from __future__ import annotations

import asyncio
import time
from pathlib import Path
from types import SimpleNamespace

import httpx
import pytest
from fastapi import Depends, FastAPI

from ai_service.config import Settings, validate_settings
from ai_service.extract import (
    DocumentProcessingError,
    TextPage,
    chunk_pages,
    extract_document,
)
from ai_service.security import request_signature, require_service_signature
from ai_service.worker import IngestionWorker


def make_settings(root: Path, secret: str = "s" * 32) -> Settings:
    return Settings("db", 5432, "vault", "u", "p", "disable", secret, root, "http://ollama:11434", "nomic-embed-text", 768, 2, 5, 1000, 20, 10000, 1200, 150)


def test_chunking_overlaps_without_empty_chunks() -> None:
    chunks = chunk_pages([TextPage(1, "alpha beta gamma delta epsilon zeta")], chunk_size=18, overlap=4)
    assert len(chunks) >= 2
    assert all(text for _, _, text in chunks)
    assert chunks[0][1] == 1


def test_database_url_escapes_credentials(tmp_path: Path) -> None:
    settings = make_settings(tmp_path)
    settings = Settings(**{**settings.__dict__, "db_user": "user@local", "db_password": "p:a/ss"})
    assert "user%40local" in settings.database_url
    assert "p%3Aa%2Fss" in settings.database_url


def test_settings_reject_short_internal_secret(tmp_path: Path) -> None:
    with pytest.raises(ValueError):
        validate_settings(make_settings(tmp_path, "short"))


def test_settings_reject_nonpositive_ocr_page_limit(tmp_path: Path) -> None:
    settings = make_settings(tmp_path)
    settings = Settings(**{**settings.__dict__, "max_ocr_pages": 0})
    with pytest.raises(ValueError, match="OCR"):
        validate_settings(settings)


def test_settings_reject_external_embedding_service(tmp_path: Path) -> None:
    settings = make_settings(tmp_path)
    settings = Settings(**{**settings.__dict__, "ollama_url": "https://api.openai.com"})
    with pytest.raises(ValueError, match="local/private"):
        validate_settings(settings)

    local = Settings(**{**settings.__dict__, "ollama_url": "http://127.0.0.1:11434"})
    validate_settings(local)


def test_object_key_rejects_traversal(tmp_path: Path) -> None:
    worker = IngestionWorker(make_settings(tmp_path), pool=None, embedder=None)
    for key in ("../outside", "/absolute/path", "a/../../b", "a\\..\\b"):
        with pytest.raises(DocumentProcessingError):
            worker._resolve_object(key)


def test_text_extraction_and_hashable_chunking(tmp_path: Path) -> None:
    path = tmp_path / "notes.txt"
    path.write_text("Local models only. This text stays inside the deployment.", encoding="utf-8")
    pages = extract_document(path, "text/plain", "notes.txt", 10, 1000)
    chunks = chunk_pages(pages, 24, 4)
    assert len(chunks) > 1
    assert "Local models only" in " ".join(chunk[2] for chunk in chunks)


def test_text_extraction_stops_at_character_limit(tmp_path: Path) -> None:
    path = tmp_path / "large.txt"
    path.write_text("0123456789", encoding="utf-8")
    with pytest.raises(DocumentProcessingError, match="processing limit"):
        extract_document(path, "text/plain", "large.txt", 10, 9)


def test_docx_extraction_enforces_character_limit(tmp_path: Path) -> None:
    from docx import Document

    path = tmp_path / "notes.docx"
    document = Document()
    document.add_paragraph("first paragraph")
    document.add_paragraph("second paragraph")
    document.save(path)
    with pytest.raises(DocumentProcessingError, match="processing limit"):
        extract_document(path, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "notes.docx", 10, 20)


def test_pdf_extraction_enforces_cumulative_character_limit(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    from ai_service import extract as extraction

    class Page:
        def extract_text(self) -> str:
            return "123456"

    class Reader:
        def __init__(self) -> None:
            self.pages = [Page(), Page()]

    path = tmp_path / "synthetic.pdf"
    path.write_bytes(b"pdf")
    monkeypatch.setattr(extraction, "PdfReader", lambda *_args, **_kwargs: Reader())
    with pytest.raises(DocumentProcessingError, match="processing limit"):
        extract_document(path, "application/pdf", "synthetic.pdf", 10, 10)


def test_scanned_pdf_uses_bounded_local_ocr(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    import pytesseract
    from pypdf import PdfWriter

    path = tmp_path / "scan.pdf"
    writer = PdfWriter()
    writer.add_blank_page(width=72, height=72)
    with path.open("wb") as output:
        writer.write(output)
    monkeypatch.setattr(pytesseract, "image_to_string", lambda _image, timeout: "recognized local text")
    pages = extract_document(path, "application/pdf", "scan.pdf", 10, 1000)
    assert pages[0].page_number == 1
    assert pages[0].text == "recognized local text"


def test_scanned_pdf_respects_aggregate_ocr_page_limit(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    from pypdf import PdfWriter

    path = tmp_path / "scan.pdf"
    writer = PdfWriter()
    writer.add_blank_page(width=72, height=72)
    with path.open("wb") as output:
        writer.write(output)
    with pytest.raises(DocumentProcessingError, match="OCR limit"):
        extract_document(path, "application/pdf", "scan.pdf", 10, 1000, max_ocr_pages=0)


def test_pdf_page_limit_is_enforced(tmp_path: Path) -> None:
    from pypdf import PdfWriter

    path = tmp_path / "one-page.pdf"
    writer = PdfWriter()
    writer.add_blank_page(width=72, height=72)
    with path.open("wb") as output:
        writer.write(output)
    with pytest.raises(DocumentProcessingError):
        extract_document(path, "application/pdf", "one-page.pdf", 0, 1000)


def test_signature_binds_path_method_and_body() -> None:
    body = b'{"user_id":"u1","query":"hello"}'
    sig = request_signature("s" * 32, "123", "POST", "/internal/v1/search", body)
    assert sig != request_signature("s" * 32, "123", "GET", "/internal/v1/search", body)
    assert sig != request_signature("s" * 32, "123", "POST", "/internal/v1/other", body)


def test_fastapi_rejects_invalid_or_stale_service_signatures() -> None:
    secret = "test-secret-with-at-least-32-bytes"
    app = FastAPI()
    app.state.settings = SimpleNamespace(internal_secret=secret)

    @app.post("/internal/v1/search", dependencies=[Depends(require_service_signature)])
    async def signed_route():
        return {"ok": True}

    async def exercise() -> None:
        transport = httpx.ASGITransport(app=app)
        async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
            body = b'{}'
            timestamp = str(int(time.time()))
            valid = request_signature(secret, timestamp, "POST", "/internal/v1/search", body)
            headers = {"Content-Type": "application/json", "X-CloudVault-Timestamp": timestamp, "X-CloudVault-Signature": valid}
            assert (await client.post("/internal/v1/search", content=body, headers=headers)).status_code == 200
            headers["X-CloudVault-Signature"] = "0" * 64
            assert (await client.post("/internal/v1/search", content=body, headers=headers)).status_code == 401
            stale = str(int(time.time()) - 120)
            headers["X-CloudVault-Timestamp"] = stale
            headers["X-CloudVault-Signature"] = request_signature(secret, stale, "POST", "/internal/v1/search", body)
            assert (await client.post("/internal/v1/search", content=body, headers=headers)).status_code == 401

    asyncio.run(exercise())
