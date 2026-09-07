# Logos Stories (Glossias)

Language-learning platform for introductory students: interactive stories with vocabulary, grammar, and translation exercises. Go backend + React frontend.

Frontend agent instructions live in [`frontend/AGENTS.md`](frontend/AGENTS.md).

## Architecture

**Backend:** Go 1.25 · Gorilla Mux · PostgreSQL (Supabase) · SQLC-generated queries · Clerk JWT auth · BigCache

**Three-layer backend pattern:** HTTP handlers → models (business logic) → SQLC generated queries → PostgreSQL

## Running Locally

**Backend** (port 8080):

```bash
go run main.go
```

Frontend (port 5173) is documented in `frontend/AGENTS.md`. Both must run concurrently for the app to work. The Vite dev server proxies `/api` to `:8080` — no CORS config needed in dev.

## Building

```bash
go build ./...
```

## After every non-trivial change

Run the checks for whichever side was touched before considering a task done. Do not skip steps. Frontend checks: `frontend/AGENTS.md`.

**DB query budgets:** every request logs `db_queries=N` and warns above 15 (`dbQueryWarnThreshold` in `main.go`). Handler tests should wrap the success case in `assertQueryBudget(t, max, handler, req)` (`src/apis/handlers/querybudget_test.go`). If a budget has to rise, the fix is almost always a batch SQLC query (`WHERE id = ANY($1::int[])`) rather than a bigger number.

### Backend (Go)

```bash
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

Optionally, run `go test -v ./src/pkg/models/...` for verbose package tests

**Go style:** use modern Go 1.22+ idioms — `for i := range n` instead of `for i := 0; i < n; i++`, `min`/`max` builtins, `slices`/`maps` packages over hand-rolled loops.

## Key Environment Variables

Backend (`.env`):

- `PORT` — defaults to 8080
- `LOG_LEVEL` — `DEBUG` (default), `INFO`, `WARN`, or `ERROR`; read after `.env` loads
- `CLERK_SECRET_KEY`, `AUTHORIZED_PARTY`
- `DATABASE_URL`
- `STORAGE_URL`, `STORAGE_API_KEY`
- `DEV_USER` — when set, bypasses Clerk auth (dev only)
- `ANTHROPIC_API_KEY` — enables AI grading of Produce submissions (`claude-haiku-4-5`, background, fail-open). Unset → submissions are stored ungraded and a warning is logged at startup.

## Directory Layout

```
main.go                     # Backend entrypoint
src/
  admin/                    # Admin API handlers
  apis/handlers/            # Student-facing API handlers
  auth/                     # JWT middleware
  logging/                  # Structured logger
  pkg/
    database/               # DB connection pool
    database/migrations/    # goose SQL migrations (also the SQLC schema source)
    database/queries/       # SQLC query definitions
    generated/db/           # SQLC-generated query code (do not edit manually)
    models/                 # Business logic layer
    cache/                  # BigCache wrapper
frontend/                   # React app — see frontend/AGENTS.md
bruno-reqs/                 # Bruno REST client request collection
scripts/                    # Python analytics scripts
```

## Database / SQLC

Queries live in `src/pkg/database/queries/*.sql` and are compiled by SQLC into `src/pkg/generated/db/`. To regenerate after changing a query or the schema:

```bash
sqlc generate
```

Never edit files under `src/pkg/generated/db/` by hand.

**The goose migrations in `src/pkg/database/migrations/` are the single source of truth for the schema** — SQLC reads that directory directly (see `sqlc.yaml`), and the backend applies the migrations on startup. To change the schema, add a numbered migration with `-- +goose Up` / `-- +goose Down` sections and re-run `sqlc generate`. There is no separate `schema.sql` to keep in sync.

### Verifying against the local database

`DATABASE_URL` in `.env` points at the local Supabase Postgres (`localhost:54322`); `psql` lives at `/opt/homebrew/opt/postgresql@18/bin/psql`. It is a development database, but the real accounts in it (the `DEV_USER` account included) carry state the developer is using.

- **Destructive checks only against `user_test_student`** ("Test Student", `test.student@example.com`, enrolled in course 1). It exists for this purpose. Seed whatever shape of data you need on that user first; never delete or reset rows belonging to any other account, even to "verify" an endpoint. Cleaning up: `DELETE /api/admin/stories/{id}/students/user_test_student/progress?phase=all`.
- To hit the running backend as an admin, send the `dev_auth: 12345678` header. Unauthenticated student routes return 401 without it.
- The backend does not hot-reload: new routes need a restart (or run a second instance with `PORT=8099 go run main.go`).

## Auth

Clerk is used for both frontend (ClerkProvider in `frontend/app/root.tsx`) and backend (JWT middleware in `src/auth/`). Role-based access: `super_admin`, `course_admin`, `student`. The `DEV_USER` env var bypasses auth entirely — never set it in production.
If you want to cURL a request, include `'dev_auth: 12345678'` as a header to be authenticated as admin.

## Routing Conventions

- Student API routes: `/api/*`
- Admin API routes: `/api/admin/*`
- Frontend routes: file-based under `frontend/app/routes/` — see `frontend/AGENTS.md`

## Known Issues

- Global `queries` variable shared across requests creates a race condition in transactions — should be scoped per-request.
- Rate limiter uses an unbounded map (memory leak under load).
- N+1 query pattern in story loading (no batching). The per-request `db_queries` log field / WARN exposes which endpoints are affected.
