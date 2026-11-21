# RAG Chat Architecture and Frontend Integration Plan

**Status:** design document. RAG chat and the React frontend are not implemented here. The summary worker and authenticated summary-status/read route are implemented separately; do not interpret the designs below as runtime-verified.

## Existing API surface the frontend can use

Every protected API request currently requires `Authorization: Bearer <token>` and `X-Salt: <salt>`. The API does not currently issue an HttpOnly-cookie session or refresh token. Authentication is exposed as `POST /v1/auth/oauth/register`, `POST /v1/auth/oauth/login`, `POST /v1/auth/anonymous/register`, `POST /v1/auth/validate`, and authenticated `POST /v1/auth/logout`. Despite the `/oauth/` path, register/login currently use local email/password credentials, not an OAuth provider.

| Feature | Existing endpoint(s) | Frontend behavior |
|---|---|---|
| Authentication | `/v1/auth/oauth/register`, `/oauth/login`, `/anonymous/register`, `/validate`, `/logout` | Pair token and salt; attach both headers to protected calls; clear both on logout/401. Prefer in-memory credentials for an SPA until a cookie-based session contract is deliberately added. |
| Dashboard | `GET /v1/analytics/storage`, `GET /v1/files` | Display caller-scoped usage/quota, recent activity, and owned-file overview. Analytics currently covers storage, not AI usage. |
| Upload | `POST /v1/files` | Send multipart field `file`; handle both `201` new file and `200` owner-scoped duplicate; never infer ingestion completion from upload response. |
| File management | `GET /v1/files`, `GET /v1/files/:id`, `PUT /v1/files/:id`, `DELETE /v1/files/:id`, `GET /v1/files/:id/download` | Use current search/filter/sort/pagination; download only through the application route; treat delete `202` as queued, not complete. |
| Folders | `GET/POST /v1/folders`, `PATCH/DELETE /v1/folders/:id`, `PATCH /v1/files/:id/folder` | Build owned-folder navigation; `409` means a non-empty folder cannot be removed. |
| Document processing | `GET /v1/files/:id/ingestion`, `GET /v1/files/:id/summary` | Poll processing state with backoff. The ingestion endpoint is owner-only. The summary endpoint permits the owner or an explicit read-granted recipient and returns only sanitized status plus completed summary text. |
| Semantic search | `POST /v1/search/semantic` | Send query and optional limit/MIME prefix. Render returned text chunks as retrieval results; this endpoint does not generate an answer. |
| Secure sharing | `/v1/files/:id/shares`, `/v1/shares/with-me`, `/v1/shares/with-me/:id`, `/v1/files/:id/share-links` | Make owner-only grant management distinct from a recipient removing their own shared reference. Treat raw public-link tokens as one-time secrets. `can_delete` never permits permanent deletion of another owner's original. |

## Future RAG chat (not implemented)

Keep Go/Gin as the authenticated public boundary, Python/FastAPI as the private retrieval/generation service, PostgreSQL + pgvector as the source of document chunks, and local Ollama as the only inference runtime. The browser must never call Ollama or FastAPI directly and must never receive `AI_SHARED_SECRET` or a caller-supplied trusted `user_id`.

A future Go endpoint can accept a user message, optional conversation identifier, and optional file scope. Go derives the principal from JWT claims and signs an internal request to FastAPI. FastAPI must apply the same owner-or-`permissions.can_read=TRUE` and `files.state='active'` predicate already used by semantic retrieval on every turn. If the client supplies file IDs, they are a narrowing filter, never an authorization grant. Re-check access on every request so revoked shares or deleted files are unavailable to existing conversations.

Retrieval should return bounded top-k chunks with `file_id`, safe filename, `chunk_index`, nullable `page_number`, and similarity score. Prompt construction must clearly delimit untrusted document text, instruct the local generation model to answer only from those excerpts, and require abstention when evidence is absent or below a calibrated retrieval threshold. A no-evidence answer should say that the accessible documents do not contain enough information. Each factual answer should carry citations generated only from retrieved chunk metadata; validate that every emitted citation refers to a retrieved chunk. Never invent page numbers; omit them when unavailable. Return a structured answer plus references so the UI can link to the file and, where supported, its page.

The future chat design also needs bounded context/token budgets, a local generation-model setting distinct from the embedding model, per-user request limits, timeouts, cancellation, and sanitized failure states. Conversation persistence, retention/deletion behavior, prompt-injection handling, and the answer-quality/abstention threshold require explicit decisions before implementation. Do not persist more extracted text than needed for citations, and audit the acting user and file references without logging prompts or document contents.

A candidate public contract (not registered or present in OpenAPI) is `POST /v1/chat` with `{message, conversation_id?, file_ids?}` and a response `{conversation_id, answer, references:[{file_id,file_name,page_number,chunk_index}]}`. Do not build against this placeholder until the endpoint, limits, streaming behavior, and conversation-retention policy are agreed.

## Frontend implementation sequence

1. **API client and auth:** centralize the base URL, bearer token and `X-Salt` headers, JSON/error parsing, cancellation, and logout cleanup. Do not store credentials in URLs or expose the internal AI secret. Since the current API lacks cookie/refresh-session support, document the selected browser storage trade-off before production deployment.
2. **Dashboard:** combine `/v1/analytics/storage` with owned file/folder lists. Show quota and file counts; present analytics as storage/activity only, not AI usage.
3. **Upload and processing:** use a multipart upload progress indicator, then show duplicate/new-file outcomes. Poll ingestion; after ingestion succeeds, poll summary status. Display queued/processing/failed distinctly, allow retry UX only after a backend retry contract exists, and never claim a summary exists until `state=succeeded`.
4. **File and folder management:** implement list filters/sorts/pagination, safe metadata, folder navigation/moves, application-mediated downloads, and queued deletion. Hide controls the current principal cannot use, while treating server authorization as authoritative.
5. **Summaries and search:** render accessible summaries and semantic-search snippets with filename and page references. Clearly label semantic search as retrieved passages, not an AI-generated answer.
6. **Sharing:** keep grant and link management in owner-only views; separate “leave shared file” from owner deletion. Display public link tokens only at creation time and avoid analytics or client logs containing them.
7. **Future chat:** add only after the authenticated RAG endpoint and citation/abstention contract exists. Show answer citations as navigable references; provide a clear no-answer state and never silently replace it with an unsupported response.
8. **Analytics:** start with current storage quota and recent file activity. AI usage, summary latency, chat usage, and model health require new server-side metrics endpoints and privacy/retention decisions.

## Backend contract gaps from the audit

These are pre-existing findings, intentionally not changed as part of summary/RAG preparation:

- Share-link revocation should verify that `link_id` belongs to the `file_id` in the URL; current repository/service behavior should be fixed before building destructive link-management UI.
- `/v1/auth/oauth/*` names are password-based local auth, not provider OAuth; use the current contract carefully and plan a compatibility-preserving rename/alias if needed.
- Analytics failure currently maps to `500` while OpenAPI advertises `503`; align before frontend error handling depends on it.
- Profile-update validation and current-password requirements differ between controller and OpenAPI; align the response contract.
- `CreateUser` exists in a controller but is not registered as a route or documented; do not assume the UI can call it.
- Route tests do not yet provide broad OpenAPI parity or handler authorization coverage.
