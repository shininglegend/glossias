# Logos Stories (Glossias)

Language-learning platform for introductory students: interactive stories with vocabulary, grammar, and translation exercises. Go backend + React frontend.

## Architecture

**Backend:** Go 1.25 · Gorilla Mux · PostgreSQL (Supabase) · SQLC-generated queries · Clerk JWT auth · BigCache

**Frontend:** React 19 · React Router 7 (SPA mode) · Vite 6 · Tailwind CSS 4 · TypeScript 5 · Clerk React

**Three-layer backend pattern:** HTTP handlers → models (business logic) → SQLC generated queries → PostgreSQL

## Running Locally

**Backend** (port 8080):
```bash
go run main.go
```

**Frontend** (port 5173, proxies `/api` to `:8080`):
```bash
cd frontend
npm install
npm run dev
```

Both must run concurrently for the app to work. The Vite dev server handles the proxy — no CORS config needed in dev.

## Building

```bash
# Backend
go build ./...

# Frontend
cd frontend && npm run build   # output: frontend/build/
```

## Type Checking / Lint

```bash
cd frontend && npm run typecheck
```

There are currently no Go unit tests in CI — the workflow only builds and hits a health endpoint.

## Key Environment Variables

Backend (`.env`):
- `PORT` — defaults to 8080
- `CLERK_SECRET_KEY`, `AUTHORIZED_PARTY`
- `DATABASE_URL`
- `STORAGE_URL`, `STORAGE_API_KEY`
- `DEV_USER` — when set, bypasses Clerk auth (dev only)

Frontend:
- `VITE_CLERK_PUBLISHABLE_KEY`

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
    generated/db/           # SQLC-generated query code (do not edit manually)
    models/                 # Business logic layer
    cache/                  # BigCache wrapper
migrations/                 # SQL migration files
frontend/
  app/
    routes/                 # File-based page components (admin.*, stories-*)
    components/             # Reusable UI components
    contexts/               # React Context (UserContext)
    services/               # API call helpers
    types/                  # Shared TypeScript types
  vite.config.ts            # Proxy config
  react-router.config.ts    # SPA mode (SSR: false)
bruno-reqs/                 # Bruno REST client request collection
scripts/                    # Python analytics scripts
```

## Database / SQLC

SQL queries live in source files and are compiled by SQLC into `src/pkg/generated/db/`. To regenerate after changing SQL:

```bash
sqlc generate
```

Never edit files under `src/pkg/generated/db/` by hand.

## Auth

Clerk is used for both frontend (ClerkProvider in `root.tsx`) and backend (JWT middleware in `src/auth/`). Role-based access: `super_admin`, `course_admin`, `student`. The `DEV_USER` env var bypasses auth entirely — never set it in production.

## Routing Conventions

- Student API routes: `/api/*`
- Admin API routes: `/api/admin/*`
- Frontend routes: file-based under `frontend/app/routes/`

## Known Issues

- Global `queries` variable shared across requests creates a race condition in transactions — should be scoped per-request.
- Rate limiter uses an unbounded map (memory leak under load).
- Several large "god components" in the frontend (~400–640 lines with 15+ state variables).
- N+1 query pattern in story loading (no batching).
