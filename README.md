# CloudVault AI — Go Backend Foundation

An in-progress secure document-management system extending the original Go-Store repository. The
Go API provides hardened authentication, owner-scoped file/folder operations, audited sharing,
local/cloud storage adapters, and quota-safe uploads. Phase 2 adds a local FastAPI ingestion worker
for extraction/OCR, Ollama embeddings, pgvector storage, permission-scoped semantic retrieval, and an
independent retryable local-Ollama document-summary queue. The React frontend and generated-answer
RAG flow are not implemented yet.

Built with **Go 1.21**, **Gin**, **PostgreSQL + pgvector**, **FastAPI**, **Ollama**, and **JWT authentication**.

---

## Architecture

```
cmd/api/
└── main.go                   Entry point — loads config, starts server

internal/
├── config/config.go          Reads .env + env vars; validates required fields
├── database/database.go      Opens + pings the *sql.DB connection pool
├── auth/
│   ├── jwt.go                GenerateToken / ValidateToken (golang-jwt/jwt v5)
│   ├── anonymous.go          Anonymous session ID generator
│   └── middleware.go         CORS, request logger, JWTAuthMiddleware
├── models/                   Pure data structs — NO database calls
│   ├── user.go
│   ├── file.go
│   ├── permission.go
│   ├── folder.go
│   └── analytics.go
├── repository/               database/sql queries — one struct per table
│   ├── user_repository.go
│   ├── file_repository.go
│   ├── file_search_repository.go
│   ├── folder_repository.go
│   ├── permission_repository.go
│   ├── token_revocation_repository.go
│   └── file_activity_repository.go
├── storage/                  Local/cloud storage abstraction
│   ├── storage.go            Storage interface
│   ├── s3_storage.go         AWS S3 implementation
│   ├── gcs_storage.go        Google Cloud Storage implementation
│   └── local_storage.go     Local filesystem implementation
├── services/                 Business logic — orchestrates repo + storage
│   ├── auth_service.go
│   ├── user_service.go
│   ├── file_service.go
│   ├── folder_service.go
│   ├── sharing_service.go
│   └── analytics_service.go
├── controllers/              Thin HTTP handlers — parse input, call service, write response
│   ├── auth_controller.go
│   ├── user_controller.go
│   ├── file_controller.go
│   ├── folder_controller.go
│   ├── sharing_controller.go
│   └── analytics_controller.go
├── merrors/merrors.go        Consistent JSON error responses
├── utils/response.go         OK / Created helpers
└── server/
    ├── routes.go             Dependency wiring + route registration
    └── server.go             HTTP server lifecycle, graceful shutdown
```

---

## Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Python 3.12+ for local AI-service tests; Docker Compose for the optional Ollama/AI stack
- Local filesystem storage by default; AWS S3 and GCS are optional providers
- [golang-migrate CLI](https://github.com/golang-migrate/migrate) (for running migrations)
- [Task](https://taskfile.dev) (optional, replaces Make)

---

## Setup

### 1. Clone

```bash
git clone https://github.com/souvik03-136/Go-Store.git
cd Go-Store
```

### 2. Environment variables

```bash
cp .env.example .env
# Set a strong JWT secret and local PostgreSQL credentials. Cloud storage is optional.
openssl rand -hex 32
```

Key variables:

| Variable | Description | Default |
|---|---|---|
| `SERVER_PORT` | HTTP listen port | `8080` |
| `APP_ENV` | `development` or `production` | `development` |
| `DB_DRIVER` | PostgreSQL driver (migrations/queries target PostgreSQL) | `postgres` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_DATABASE` | Database name | `gostore` |
| `DB_USERNAME` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | *(required)* |
| `JWT_SECRET_KEY` | HMAC signing secret — keep long & random | *(required)* |
| `STORAGE_PROVIDER` | `local`, `s3`, or `gcs` | `local` |
| `LOCAL_STORAGE_ROOT` | Local object directory | `./data/files` |
| `STORAGE_QUOTA_BYTES` | Per-user quota shown in analytics | `5368709120` (5 GiB) |
| `MAX_UPLOAD_BYTES` | Maximum actual file bytes | `52428800` (50 MiB; max 1 GiB) |
| `CORS_ALLOWED_ORIGINS` | Comma-separated exact browser origins; no `*` | local Vite origins |
| `AI_SERVICE_URL` | Internal FastAPI origin; blank disables AI retrieval (returns a clear 503) | blank |
| `AI_SHARED_SECRET` | 32+ byte HMAC secret shared only by Go and AI containers | required for AI profile |
| `OLLAMA_EMBED_MODEL` | Local Ollama embedding model | `nomic-embed-text` (768 dimensions) |
| `OLLAMA_SUMMARY_MODEL` | Separate local text-generation model for summaries | `llama3.2:3b` |
| `OLLAMA_BASE_URL` | Local/private Ollama origin only; host-run service may use `http://127.0.0.1:11434` | `http://ollama:11434` in Compose |
| `AI_MAX_OCR_PAGES` | Maximum scanned pages locally OCR'd per PDF | `40` |

### 3. Run migrations

```bash
./scripts/migrate.sh up
# or: task migrate:up
```

### 4. Run the server

```bash
go run ./cmd/api
# or: task run
```

The API is available at `http://localhost:8080`.

### 5. Docker Compose (optional)

```bash
docker compose up --build
# or: task docker:up
```

The default Compose stack runs Go, pgvector-enabled PostgreSQL, and local file storage without
cloud credentials. Apply database migrations before starting the API (from the host, with the
migration CLI installed). Set a non-empty `DB_PASSWORD` and a `JWT_SECRET_KEY` of at least 32 bytes
in `.env` first. Uploaded local objects are persisted in the `local_files` volume.

To enable Phase 2 ingestion and semantic retrieval, set a separate internal-service secret in
`.env` with `openssl rand -hex 32`, then run the AI profile with the Go-to-AI URL explicitly
enabled (the default setup keeps it blank):

```bash
AI_SERVICE_URL=http://ai-api:8000 docker compose --profile ai up --build
```

The profile starts Ollama, the private FastAPI service, and the durable ingestion and summary workers.
Ollama downloads `nomic-embed-text` and the separate `llama3.2:3b` generation model into its local
volume; inference and OCR run locally and do not
use hosted AI APIs or API credentials. Compose passes the same `AI_SHARED_SECRET` from `.env` to
Go, FastAPI, and the worker; Go and Python validate its 32-byte minimum. The AI containers mount
local file storage read-only. With AI disabled, `POST /v1/search/semantic` returns 503 with
`AI retrieval is not configured` rather than attempting to contact a missing service.

---

## API Reference

All protected routes require these headers:

```
Authorization: Bearer <token>
X-Salt: <salt>
```

Both `token` and `salt` are returned by the login / register endpoints.

### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/auth/oauth/register` | No | Register with username + email + password |
| `POST` | `/v1/auth/oauth/login` | No | Login; returns `token` + `salt` |
| `POST` | `/v1/auth/anonymous/register` | No | Create an anonymous session |
| `POST` | `/v1/auth/logout` | Yes | Revoke the current token in PostgreSQL |
| `POST` | `/v1/auth/validate` | No | Validate bearer token and `X-Salt` header |

### Roles

Accounts are created with the `user` role. `/v1/admin/users` is guarded by a
database-backed admin-role check. There is deliberately no self-service role
promotion endpoint; an operator can promote a trusted account directly in the
database with `UPDATE users SET role='admin' WHERE email='admin@example.com';`.

### Storage analytics

`GET /v1/analytics/storage` returns the caller's file count and bytes used, enforced
quota, in-flight reservations, remaining bytes, MIME-type totals, and recent activity.
The default quota is 5 GiB and can be changed with `STORAGE_QUOTA_BYTES`. Upload
reservations serialize per-user quota checks, so simultaneous uploads cannot exceed it.

**Register example:**
```bash
curl -X POST http://localhost:8080/v1/auth/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"s3cur3pass"}'
```

### Users

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET`    | `/v1/users/:id` | Yes | Get user by ID |
| `PUT`    | `/v1/users/:id` | Yes | Update user; current password required for every profile update |
| `DELETE` | `/v1/users/:id` | Yes | Delete only after owned files/uploads are fully removed |
| `GET`    | `/v1/admin/users` | Admin | List user profiles without password hashes |

### Files

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST`   | `/v1/files`     | Yes | Upload a file (`multipart/form-data`, field: `file`) |
| `GET`    | `/v1/files`     | Yes | Search/filter/sort/paginate owned files; `folder_id=root` filters root files |
| `GET`    | `/v1/files/:id` | Yes | Get file metadata as owner or granted reader |
| `GET`    | `/v1/files/:id/download` | Yes | Download as owner or granted reader |
| `GET`    | `/v1/files/:id/ingestion` | Yes | Get ingestion state as the file owner |
| `GET`    | `/v1/files/:id/summary` | Yes | Poll summary state/read completed summary as owner or granted reader |
| `PATCH`  | `/v1/files/:id/folder` | Yes | Move an owned file to an owned folder; `{"folder_id":null}` moves it to root |
| `PUT`    | `/v1/files/:id` | Yes | Update metadata as owner or granted writer |
| `DELETE` | `/v1/files/:id` | Yes | Queue permanent deletion; owner only (202 Accepted) |
| `POST`   | `/v1/files/:id/shares` | Yes | Grant or replace recipient read/write/delete flags |
| `GET`    | `/v1/files/:id/shares` | Yes | List grants (owner only) |
| `DELETE` | `/v1/files/:id/shares/:user_id` | Yes | Revoke a recipient grant (owner only) |
| `POST`   | `/v1/files/:id/share-links` | Yes | Create an expiring optional-password link; token returned once |
| `GET`    | `/v1/files/:id/share-links` | Yes | List link metadata (never token hashes) |
| `DELETE` | `/v1/files/:id/share-links/:link_id` | Yes | Revoke an owner's link |
| `GET`    | `/v1/shares/with-me` | Yes | List files shared with the caller that grant read access |
| `DELETE` | `/v1/shares/with-me/:id` | Yes | Remove the caller's shared reference; does not delete the original |

### Local AI ingestion and semantic retrieval

New uploads enqueue an ingestion job in the same database transaction that commits file metadata.
Migration 010 also queues existing active files that already have a stored SHA-256 digest; older
records without a digest are not automatically ingested.
The worker claims jobs with PostgreSQL `SKIP LOCKED`, recovers expired worker leases, verifies the
stored SHA-256, extracts PDF/DOCX/text/CSV/JSON/XML/HTML content, and applies local Tesseract OCR to
images and scanned PDF pages. Extracted chunks and 768-dimensional Ollama embeddings are stored in
pgvector. Check `GET /v1/files/:id/ingestion` for `queued`, `processing`, `succeeded`, `failed`, or
`cancelled` status; status is owner-only.

Migration 011 adds an independent `document_summary_jobs` queue and `document_summaries` records.
Upload finalization enqueues summary work atomically with ingestion; the summary worker waits for
successful ingestion, then runs bounded hierarchical map/reduce over persisted page-aware chunks
with the separate local `OLLAMA_SUMMARY_MODEL`. Summary retries and failures do not change ingestion
state, chunks, or embeddings. `GET /v1/files/:id/summary` returns job state and, when complete,
summary text to the owner or a caller with explicit read access. Summary persistence is fenced by the
summary attempt and source SHA-256. Existing active files are queued by migration 011.

`POST /v1/search/semantic` accepts `{"query":"...","limit":10,"content_type":"application/pdf"}`;
`content_type` is optional and can be a MIME prefix such as `image/`. Results are limited to active
files the caller owns or can read through a share grant and to the currently configured embedding
model. The Go API signs its internal request to FastAPI with timestamp-bound HMAC; the shared secret
is never sent to the browser. This endpoint returns retrieval snippets, not a generated answer. The AI
Compose profile uses only the local Ollama model and does not call a hosted model API.

The RAG chat architecture and frontend integration sequence—including citations/page references,
abstention, access-control invariants, and known API contract gaps—are documented in
[`docs/rag-frontend-plan.md`](./docs/rag-frontend-plan.md). Chat and the React frontend are design-only.

`GET /v1/public/share-links/:token/download` is public but rate limited; tokens expire
within 30 days and may require `X-Share-Password`. The raw token is returned only
when created. Request logs use route templates and never log the token path value.

### Folders

| Method | Path | Auth | Description |
|---|---|---:|---|
| `POST` | `/v1/folders` | Yes | Create folder; optional `parent_id` must be owned by caller |
| `GET` | `/v1/folders?parent_id=...` | Yes | List direct child folders; omit `parent_id` for root |
| `PATCH` | `/v1/folders/:id` | Yes | Rename or move folder; empty `parent_id` moves it to root |
| `DELETE` | `/v1/folders/:id` | Yes | Delete an empty folder only |

Share grants use registered user IDs. `can_write` or `can_delete` automatically
implies `can_read`. `can_delete` does not authorize destruction of an owner's
canonical file: only the owner can permanently delete it. Recipients can revoke
their own access with `DELETE /v1/shares/with-me/:id`.

File-list parameters: `search` matches filenames, `content_type` matches a MIME
type prefix (for example `image/`), `folder_id` selects an owned folder (or
`root`), `sort` is `name`, `size`, `created_at`, or `updated_at`, `order` is
`asc`/`desc`, and `limit`/`offset` provide bounded pagination (maximum limit 100).

The API schema is in [`docs/openapi.yaml`](./docs/openapi.yaml).

Uploads default to a 50 MiB actual-file limit (`MAX_UPLOAD_BYTES`, configurable up
to 1 GiB); bounded multipart framing overhead is accounted separately. The server
sniffs MIME from file bytes, computes SHA-256, and deduplicates identical content
within an owner's namespace. File responses omit internal storage keys and provider
URLs. S3/GCS buckets must block public reads; downloads stream through the
authorization-checked API.

**Upload example:**
```bash
curl -X POST http://localhost:8080/v1/files \
  -H "Authorization: Bearer <token>" \
  -H "X-Salt: <salt>" \
  -F "file=@/path/to/document.pdf"
```

### Health

```
GET /healthz   →  {"status":"ok"}  or  {"status":"unhealthy"}
```

---

## Development

```bash
task test          # run tests with race detector
task test:ai       # run Python AI-service tests (after installing requirements-dev)
task lint          # golangci-lint
task fmt           # gofmt
task build         # compile to bin/api
task migrate:up    # apply migrations
task migrate:down  # roll back last migration
```

### Phase 2 verification snapshot (2026-11-21)

After adding summaries, `go test -race ./...`, `go vet ./...`, `go build ./...`, Python bytecode compilation, **36 Python tests**, Ruff, YAML parsing for Compose/OpenAPI, and `git diff --check` passed. The new summary tests use HTTP/database fakes; existing in-process pipeline coverage also uses a fake database and does not establish live pgvector or Ollama behavior.

Live PostgreSQL/pgvector, Docker Compose configuration/runtime, Ollama inference, and Tesseract OCR were **not** run: Docker is unavailable and no PostgreSQL integration DSN was provided. Compose and OpenAPI YAML were parsed, but `docker compose config`, migration execution, DB locking/lease behavior, and the complete Go-upload-to-pgvector/Ollama summarization flow remain to be verified in a runtime with those dependencies.

---

## Security notes

- Passwords are hashed with bcrypt; every profile update requires the current password. Changing email/password increments the session version, invalidating prior JWTs.
- Each JWT is signed with `HMAC-SHA256` using a per-token salt-derived key; the salt is returned to the client alongside the token.
- The `X-Salt` header is required to validate any token. Clients must store it alongside the token.
- File owner/share checks are enforced for metadata reads and downloads; only owners may permanently delete original files.
- Profile routes only allow the authenticated subject to access its own record.
- Quota is reserved transactionally before storage writes; failed writes release reservations after safe object cleanup.
- Uploads are limited by `MAX_UPLOAD_BYTES`; content MIME and SHA-256 are derived from actual bytes.
- Exact configured CORS origins only; sensitive public endpoints use bounded per-IP rate limits.
- Audit events retain the acting identity for file, share, credential, and account actions; deletion and upload cleanup jobs survive process restarts.
- File JSON never exposes internal paths or direct provider URLs; configure cloud buckets as private.
- Ingestion reads local-storage objects read-only, requires and verifies `files.sha256`, enforces extracted-text limits during parsing and a configurable scanned-PDF OCR page cap, and stores extracted text with local embeddings in PostgreSQL.
- The Go-to-FastAPI retrieval request is HMAC-signed with a timestamp and a private 32-byte-or-longer secret; never expose `AI_SHARED_SECRET` to browser clients.
- The Docker image runs as a non-root user (`nonroot:nonroot`).

Logout invalidates the presented token server-side. Clients should still
discard both token and salt after logout. Revocation records store only a SHA-256
fingerprint and expire with the token.

---

## License

MIT — see [LICENSE](./LICENSE).
