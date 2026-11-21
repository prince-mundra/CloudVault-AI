from __future__ import annotations

import ipaddress
import os
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import quote, urlsplit


@dataclass(frozen=True)
class Settings:
    db_host: str
    db_port: int
    db_name: str
    db_user: str
    db_password: str
    db_sslmode: str
    internal_secret: str
    storage_root: Path
    ollama_url: str
    ollama_model: str
    embedding_dimension: int
    poll_seconds: float
    max_attempts: int
    max_document_bytes: int
    max_pdf_pages: int
    max_extracted_chars: int
    chunk_size: int
    chunk_overlap: int
    max_ocr_pages: int = 40
    summary_model: str = "llama3.2:3b"

    @property
    def database_url(self) -> str:
        user = quote(self.db_user, safe="")
        password = quote(self.db_password, safe="")
        name = quote(self.db_name, safe="")
        return (
            f"postgresql://{user}:{password}@{self.db_host}:{self.db_port}/{name}"
            f"?sslmode={quote(self.db_sslmode, safe='')}"
        )


def from_env() -> Settings:
    return Settings(
        db_host=os.getenv("DB_HOST", "postgres"),
        db_port=int(os.getenv("DB_PORT", "5432")),
        db_name=os.getenv("DB_DATABASE", "gostore"),
        db_user=os.getenv("DB_USERNAME", "postgres"),
        db_password=os.getenv("DB_PASSWORD", ""),
        db_sslmode=os.getenv("DB_SSL_MODE", "disable"),
        internal_secret=os.getenv("AI_SHARED_SECRET", ""),
        storage_root=Path(os.getenv("AI_STORAGE_ROOT", "/app/data/files")),
        ollama_url=os.getenv("OLLAMA_BASE_URL", "http://ollama:11434").rstrip("/"),
        ollama_model=os.getenv("OLLAMA_EMBED_MODEL", "nomic-embed-text"),
        embedding_dimension=int(os.getenv("AI_EMBEDDING_DIMENSION", "768")),
        poll_seconds=float(os.getenv("AI_POLL_SECONDS", "2")),
        max_attempts=int(os.getenv("AI_MAX_ATTEMPTS", "5")),
        max_document_bytes=int(os.getenv("AI_MAX_DOCUMENT_BYTES", str(50 * 1024 * 1024))),
        max_pdf_pages=int(os.getenv("AI_MAX_PDF_PAGES", "250")),
        max_extracted_chars=int(os.getenv("AI_MAX_EXTRACTED_CHARS", "2500000")),
        chunk_size=int(os.getenv("AI_CHUNK_SIZE", "1200")),
        chunk_overlap=int(os.getenv("AI_CHUNK_OVERLAP", "150")),
        max_ocr_pages=int(os.getenv("AI_MAX_OCR_PAGES", "40")),
        summary_model=os.getenv("OLLAMA_SUMMARY_MODEL", "llama3.2:3b"),
    )


def validate_settings(settings: Settings, require_secret: bool = True) -> None:
    if require_secret and len(settings.internal_secret.encode("utf-8")) < 32:
        raise ValueError("AI_SHARED_SECRET must contain at least 32 bytes")
    if settings.embedding_dimension != 768:
        raise ValueError("AI_EMBEDDING_DIMENSION must be 768 for the configured Ollama model")
    ollama = urlsplit(settings.ollama_url)
    host = (ollama.hostname or "").lower()
    if ollama.scheme not in {"http", "https"} or not host or ollama.username or ollama.password or ollama.query or ollama.fragment or ollama.path not in {"", "/"}:
        raise ValueError("OLLAMA_BASE_URL must be a local HTTP(S) origin without credentials or path")
    local_host = host in {"localhost", "ollama", "host.docker.internal"} or host.endswith(".local")
    if not local_host:
        try:
            address = ipaddress.ip_address(host)
            local_host = address.is_loopback or address.is_private or address.is_link_local
        except ValueError:
            local_host = False
    if not local_host:
        raise ValueError("OLLAMA_BASE_URL must resolve to a local/private Ollama host; external model APIs are not supported")
    if settings.max_document_bytes <= 0 or settings.max_pdf_pages <= 0 or settings.max_extracted_chars <= 0 or settings.max_ocr_pages <= 0:
        raise ValueError("document size, page, OCR, and extracted-text limits must be positive")
    if settings.max_attempts <= 0 or settings.poll_seconds <= 0:
        raise ValueError("worker retry and polling settings must be positive")
    if settings.chunk_size <= settings.chunk_overlap or settings.chunk_overlap < 0:
        raise ValueError("AI_CHUNK_SIZE must exceed a non-negative AI_CHUNK_OVERLAP")
    if not settings.summary_model.strip() or len(settings.summary_model) > 120:
        raise ValueError("OLLAMA_SUMMARY_MODEL must be a non-empty local Ollama model name of at most 120 characters")
