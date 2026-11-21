# CloudVault AI — Handover

**Snapshot date:** 2026-11-21  
**Source:** Current `/home/ubuntu/CloudVault-AI` working tree at backup time.  
**Verification scope:** Code/static checks only; no live database or Docker-based end-to-end run was possible in the verification environment.

## Architecture

- **Go backend:** Go 1.21 module, Gin HTTP API, PostgreSQL persistence, JWT authentication, quota-aware upload reservations, file/folder management, sharing, audit activity, and local/S3/GCS storage adapters.
- **Database:** PostgreSQL is the supported database. Migrations add the core users/files/permissions model, folders, token revocation, roles, activity/audit and upload hardening, then pgvector-based document ingestion and summaries.
- **AI service:** Python 3.12/FastAPI internal API plus a durable worker. The worker extracts PDF/DOCX/TXT content, chunks page-aware text, generates 768-dimensional embeddings with local Ollama `nomic-embed-text`, persists chunks/vectors, and runs a separate local-Ollama summary queue using `llama3.2:3b` by default.
- **Go-to-AI communication:** Internal semantic search is authenticated with a shared HMAC secret. The AI service filters retrieval to active files owned by the requesting user or explicitly shared with read permission.
- **Docker Compose:** Default mode is Go + pgvector PostgreSQL + local file storage. The optional `ai` profile adds Ollama, model initialization, the FastAPI API, and the worker.
- **Frontend:** No React frontend is present in this snapshot. `docs/rag-frontend-plan.md` is a design plan, not an implementation.

## Implemented features in this snapshot

- Registration/login, JWT validation and revocation, role-aware authorization, rate limiting, and CORS controls.
- File upload/download/deletion, local and cloud storage adapters, integrity/duplicate handling, concurrency-safe quota reservations, folder management, sharing permissions, share links, and audit/activity records.
- Authenticated ingestion status, document extraction/chunking, local Ollama embedding, durable ingestion jobs, and pgvector semantic search.
- Independent summary jobs and persisted summaries. Upload finalization queues ingestion and summary work transactionally. `GET /v1/files/:id/summary` requires authentication and permits the owner or a user with explicit read access.
- OpenAPI specification at `docs/openapi.yaml`, project documentation, tests, Compose configuration, and migration files through 011.

## Pending features and verification

- RAG/conversational document chat is not implemented.
- React frontend is not implemented.
- The real registration → authenticated PDF upload → storage → ingestion/chunks/vectors → summary retrieval → semantic-search workflow has **not** been run in the recorded verification environment.
- Runtime behavior against actual PostgreSQL/pgvector, Docker Compose, Ollama model downloads/inference, OCR, and live service connectivity remains to be verified on a machine with those dependencies.

## Database migration status

- This snapshot contains up migrations **001–011**. Migration 010 enables the pgvector extension, creates the ingestion queue, and stores embeddings as `vector(768)` with a cosine HNSW index. Migration 011 creates `document_summary_jobs` and `document_summaries` and schedules summaries for eligible existing ingestion jobs.
- Migration order and source-level references were inspected. **No live database was available, and no migration was applied in this verification.**
- Down migrations exist for 004–011. Down migrations for 001–003 are absent. Existing down migrations intentionally drop schema/data (including vectors and summaries); do not roll back production data without a separately reviewed backup and recovery plan.

## Docker configuration

- `.env.example` is included; real `.env` files/secrets are not.
- Default mode keeps `AI_SERVICE_URL` blank. Semantic search then returns a controlled `503` rather than contacting an unavailable AI service.
- AI mode uses the same configured `AI_SHARED_SECRET` for Go and the AI service and requires a value of at least 32 bytes. It uses local Ollama models, not hosted inference APIs.
- Database migrations are **not** run automatically by Compose. Apply them before starting the application. The host-side migration script requires the `migrate` CLI.

### Run with Docker Compose

1. Copy `.env.example` to `.env`, then set `DB_PASSWORD` and `JWT_SECRET_KEY`. Use random values (for example, generate locally with `openssl rand -hex 32`) and do not commit `.env`. Set `AI_SHARED_SECRET` to another random value of at least 32 bytes only when enabling AI.
2. Start PostgreSQL and apply migrations from the host:

   ```bash
   docker compose up -d postgres
   set -a; . ./.env; set +a
   DB_HOST=localhost ./scripts/migrate.sh up
   ```

   The host must have the `migrate` CLI installed and the Compose-published PostgreSQL port available.
3. Start the default application stack and check the API health endpoint:

   ```bash
   docker compose up --build -d
   curl -fsS http://localhost:8080/healthz
   ```
4. To enable local AI services, run:

   ```bash
   AI_SERVICE_URL=http://ai-api:8000 docker compose --profile ai up --build -d
   docker compose --profile ai ps
   docker compose exec ollama ollama list
   curl -fsS http://127.0.0.1:8000/healthz
   ```

   The model-init service pulls the configured embedding model (`nomic-embed-text` by default) and summary model (`llama3.2:3b` by default). The AI `/healthz` check reports database connectivity; it does not by itself prove model inference works.

### Run available checks

```bash
go test -race -count=1 ./...
go vet ./...
go build ./...
python3 -m pip install -r ai_service/requirements-dev.txt
python3 -m pytest -q ai_service/tests
```

`ruff check ai_service` was also run in the verification environment where Ruff was available.

## Tests actually executed and recorded

- `go test -race -count=1 ./...` — **PASS**.
- `go vet ./...` — **PASS**.
- `go build ./...` — **PASS**.
- `go test -v -count=1 ./internal/repository` — **PASS**, with three PostgreSQL integration tests explicitly skipped because `POSTGRES_TEST_DSN` was unset.
- `./.venv/bin/python -m pytest -q ai_service/tests` — **36 passed**.
- `./.venv/bin/ruff check ai_service` — **PASS**.
- Python AST syntax parse — **16 files parsed successfully**.
- PyYAML parse of `docker-compose.yml` and `docs/openapi.yaml` — **PASS**; static assertions confirmed profile membership, shared-secret interpolation, model defaults, and migration versions.
- `git diff --check` — **PASS**.

These checks do not constitute runtime or end-to-end verification. Docker, PostgreSQL, Ollama, `.env`, and `POSTGRES_TEST_DSN` were unavailable; no service was started and no migration or PDF workflow was executed.

## Known unresolved issues / limitations

- Migrations 001–003 have no down migration files.
- No live Compose validation/build/start or live database migration has been recorded for this snapshot.
- PostgreSQL integration tests were skipped, and the PDF ingestion, vector search, summary generation, and authenticated summary/search APIs remain runtime-unverified.
- No RAG chat or frontend is included.
- No additional code defect was confirmed by the cited checks; this is not evidence that untested runtime paths are bug-free.
