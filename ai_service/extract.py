from __future__ import annotations

import re
import zipfile
from dataclasses import dataclass
from pathlib import Path

import pypdfium2 as pdfium
import pytesseract
from docx import Document
from PIL import Image
from pypdf import PdfReader


class DocumentProcessingError(Exception):
    """A document is invalid, unsupported, or exceeds configured processing bounds."""


@dataclass(frozen=True)
class TextPage:
    page_number: int | None
    text: str


_TEXT_EXTENSIONS = {".txt", ".md", ".markdown", ".csv", ".tsv", ".json", ".xml", ".html", ".htm", ".log", ".rst"}


def extract_document(
    path: Path,
    content_type: str,
    filename: str,
    max_pdf_pages: int,
    max_chars: int,
    max_ocr_pages: int = 40,
) -> list[TextPage]:
    suffix = Path(filename).suffix.lower()
    mime = (content_type or "").split(";", 1)[0].strip().lower()
    try:
        size = path.stat().st_size
    except OSError as exc:
        raise DocumentProcessingError("Stored document is unavailable") from exc
    if size <= 0:
        raise DocumentProcessingError("Document is empty")
    if max_chars <= 0 or max_pdf_pages <= 0 or max_ocr_pages < 0:
        raise DocumentProcessingError("Document extraction limits must be positive")

    pages: list[TextPage]
    if mime == "application/pdf" or suffix == ".pdf":
        pages = _extract_pdf(path, max_pdf_pages, max_chars, max_ocr_pages)
    elif suffix == ".docx" or mime == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
        pages = _extract_docx(path, max_chars)
    elif mime.startswith("image/") or suffix in {".png", ".jpg", ".jpeg", ".tif", ".tiff", ".bmp", ".webp"}:
        pages = [_extract_image(path)]
    elif mime.startswith("text/") or suffix in _TEXT_EXTENSIONS or mime in {"application/json", "application/xml"}:
        pages = [_extract_text(path, max_chars)]
    else:
        raise DocumentProcessingError("Unsupported document type")

    total = sum(len(p.text) for p in pages)
    if total > max_chars:
        raise DocumentProcessingError("Extracted text exceeds the configured processing limit")
    if not any(p.text.strip() for p in pages):
        raise DocumentProcessingError("No extractable text found")
    return pages


def _extract_text(path: Path, max_chars: int) -> TextPage:
    try:
        with path.open("r", encoding="utf-8", errors="replace") as source:
            text = source.read(max_chars + 1)
        if len(text) > max_chars:
            raise DocumentProcessingError("Extracted text exceeds the configured processing limit")
        return TextPage(None, text)
    except DocumentProcessingError:
        raise
    except OSError as exc:
        raise DocumentProcessingError("Unable to read text document") from exc


def _extract_docx(path: Path, max_chars: int) -> list[TextPage]:
    try:
        with zipfile.ZipFile(path) as archive:
            members = archive.infolist()
            if len(members) > 10000 or sum(item.file_size for item in members) > 100_000_000:
                raise DocumentProcessingError("DOCX archive exceeds the expanded-size limit")
        doc = Document(str(path))
        parts: list[str] = []
        total_chars = 0
        for paragraph in doc.paragraphs:
            text = paragraph.text.strip()
            if text:
                total_chars += len(text) + bool(parts)
                if total_chars > max_chars:
                    raise DocumentProcessingError("Extracted text exceeds the configured processing limit")
                parts.append(text)
        for table in doc.tables:
            for row in table.rows:
                cells = [cell.text.strip() for cell in row.cells]
                if any(cells):
                    text = " | ".join(cells)
                    total_chars += len(text) + bool(parts)
                    if total_chars > max_chars:
                        raise DocumentProcessingError("Extracted text exceeds the configured processing limit")
                    parts.append(text)
        return [TextPage(None, "\n".join(parts))]
    except DocumentProcessingError:
        raise
    except Exception as exc:
        raise DocumentProcessingError("Invalid or unreadable DOCX document") from exc


def _extract_pdf(path: Path, max_pages: int, max_chars: int, max_ocr_pages: int) -> list[TextPage]:
    try:
        reader = PdfReader(str(path), strict=False)
    except Exception as exc:
        raise DocumentProcessingError("Invalid or unreadable PDF document") from exc
    pages: list[TextPage] = []
    total_chars = 0
    ocr_pages = 0
    pdfium_doc = None
    try:
        if len(reader.pages) > max_pages:
            raise DocumentProcessingError("PDF exceeds the configured page limit")
        for index, page in enumerate(reader.pages, start=1):
            try:
                text = (page.extract_text() or "").strip()
            except Exception:  # noqa: BLE001 - try bounded local OCR when PDF text parsing fails
                text = ""
            if not text:
                if ocr_pages >= max_ocr_pages:
                    raise DocumentProcessingError("PDF exceeds the configured scanned-page OCR limit")
                ocr_pages += 1
                if pdfium_doc is None:
                    pdfium_doc = pdfium.PdfDocument(str(path))
                render_page = pdfium_doc[index - 1]
                width, height = render_page.get_size()
                largest_side = max(width, height)
                scale = min(2.0, 4000.0 / max(largest_side, 1.0))
                bitmap = render_page.render(scale=scale)
                image = bitmap.to_pil()
                try:
                    text = pytesseract.image_to_string(image, timeout=30).strip()
                except Exception as exc:
                    raise DocumentProcessingError("Local OCR failed for a scanned PDF page") from exc
                finally:
                    bitmap.close()
                    render_page.close()
            if text:
                total_chars += len(text)
                if total_chars > max_chars:
                    raise DocumentProcessingError("Extracted text exceeds the configured processing limit")
                pages.append(TextPage(index, text))
    except DocumentProcessingError:
        raise
    except Exception as exc:
        raise DocumentProcessingError("Unable to extract PDF content") from exc
    finally:
        if pdfium_doc is not None:
            pdfium_doc.close()
    return pages


def _extract_image(path: Path) -> TextPage:
    try:
        with Image.open(path) as image:
            if image.width * image.height > 40_000_000:
                raise DocumentProcessingError("Image exceeds the OCR pixel limit")
            text = pytesseract.image_to_string(image, timeout=30).strip()
        return TextPage(1, text)
    except Exception as exc:
        raise DocumentProcessingError("Invalid image or local OCR failed") from exc


def chunk_pages(pages: list[TextPage], chunk_size: int, overlap: int) -> list[tuple[int, int | None, str]]:
    chunks: list[tuple[int, int | None, str]] = []
    index = 0
    for page in pages:
        normalized = re.sub(r"[ \t]+", " ", page.text).strip()
        start = 0
        while start < len(normalized):
            end = min(start + chunk_size, len(normalized))
            if end < len(normalized):
                boundary = max(normalized.rfind("\n", start, end), normalized.rfind(". ", start, end), normalized.rfind(" ", start, end))
                if boundary > start + chunk_size // 2:
                    end = boundary + (1 if normalized[boundary:boundary + 1] == "." else 0)
            text = normalized[start:end].strip()
            if text:
                chunks.append((index, page.page_number, text))
                index += 1
            if end >= len(normalized):
                break
            start = max(start + 1, end - overlap)
    return chunks
