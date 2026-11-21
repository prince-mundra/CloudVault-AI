FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

RUN apt-get update \
	&& apt-get install -y --no-install-recommends tesseract-ocr libstdc++6 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system app && useradd --system --gid app --home-dir /app app

WORKDIR /app
COPY ai_service/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt
COPY ai_service /app/ai_service
RUN chown -R app:app /app
USER app

EXPOSE 8000
CMD ["uvicorn", "ai_service.api:app", "--host", "0.0.0.0", "--port", "8000"]
