# Glossias

A web application for engaging with stories to increase the fluency of introductionary-level students.

## Installation & Setup

The app has three moving parts that all need to be up: **Supabase** (Postgres + file storage), the **Go backend** (port 8080), and the **React frontend** (port 5173). The frontend proxies `/api` to the backend, and the backend needs the database, so start them in that order.

Supabase provides both the database and the audio-file storage. The backend talks to Postgres directly over the wire with `pgx` — Supabase is simply the thing hosting that Postgres — and talks to storage over Supabase's HTTP API.

### Prerequisites

- **Go 1.25 or later** from [golang.org](https://golang.org). `go.mod` pins a `toolchain` of go1.26.5; with the default `GOTOOLCHAIN=auto` the Go command downloads it for you.
- **Node.js 20 or later** (required by React Router 7)
- **[Supabase](https://supabase.com)** — either the CLI running the stack locally (recommended for development) or a hosted project
- **Docker**, if you are running Supabase locally — the CLI runs the stack in containers
- A [Clerk](https://clerk.com) application, for auth keys

### Step 1: Clone and install dependencies

```bash
git clone https://github.com/shininglegend/glossias
cd glossias
go mod tidy
cd frontend && npm install && cd ..
```

### Step 2: Start Supabase

**Option A — local stack via the CLI (recommended for development).** Install the CLI and start it from the repository root:

```bash
brew install supabase/tap/supabase
supabase start
```

The first start pulls several Docker images and takes a few minutes. When it finishes it prints the local URLs and keys — keep that output, step 3 needs it. You can reprint it at any time with:

```bash
supabase status
```

The local ports are set in `supabase/config.toml`, which is committed:

| Service | URL |
| --- | --- |
| Postgres | `postgresql://postgres:postgres@localhost:54322/postgres` |
| API / storage gateway | `http://127.0.0.1:54321` |
| Studio (web UI) | `http://127.0.0.1:54323` |

**Option B — hosted project.** Create a project at [supabase.com](https://supabase.com), then take the connection string from *Project Settings → Database* and the storage URL and service-role key from *Project Settings → API*.

> If `DATABASE_URL` is unset the backend silently falls back to an in-memory mock store. The server will start and the UI will load, but nothing persists — so if data keeps vanishing, check this variable first.

### Step 3: Configure environment variables

Create a `.env` in the repository root. The values below match the local CLI stack from Option A; for a hosted project, substitute the ones from your dashboard.

```bash
PORT=8080                  # required — the server exits if this is unset, and the frontend proxy expects 8080

# From `supabase status` → DB URL
DATABASE_URL="postgresql://postgres:postgres@localhost:54322/postgres"

# From `supabase status` → API URL (+ /storage/v1) and service_role key
STORAGE_URL="http://127.0.0.1:54321/storage/v1"
STORAGE_API_KEY="..."      # audio uploads fail without this

CLERK_SECRET_KEY=sk_test_...
AUTHORIZED_PARTY=http://localhost:5173
# DEV_USER=some-user-id    # bypasses Clerk auth entirely — local development only, never in production
```

And a `frontend/.env`:

```bash
VITE_CLERK_PUBLISHABLE_KEY=pk_test_...
```

### Step 4: Database schema

Nothing to do by hand. The backend embeds the SQL files in `src/pkg/database/migrations/` and runs them through [goose](https://github.com/pressly/goose) on every startup, so a fresh database migrates itself the first time you launch. Already-applied migrations are skipped.

To add a schema change, drop a new numbered file in `src/pkg/database/migrations/` with goose's `-- +goose Up` / `-- +goose Down` annotations. It is applied on the next start.

> Migrations run before the connection pool opens, so a failing migration stops the server with the error rather than leaving it half-started.

### Step 5: Run the backend

```bash
go run main.go
```

Listens on `http://localhost:8080`. Check it with `curl http://localhost:8080/api/health`.

### Step 6: Run the frontend

In a second terminal:

```bash
cd frontend
npm run dev
```

Open `http://localhost:5173`. The Vite dev server proxies `/api` to the backend, so no CORS configuration is needed in development.

### To stop

Ctrl-C in the backend and frontend terminals. If you are running the local Supabase stack, stop its containers too:

```bash
supabase stop
```

Add `--no-backup` if you want to discard the local database contents rather than restore them on the next `supabase start`.


## Adding Content

### Stories
Add stories via the admin interface at `/admin`.

## Credits

### Content
- Most story text and audio files are by Dr. Jesse Scheumann, all rights reserved, used with permission
- All other story text and audio files were created by Titus Murphy, all rights reserved.

### Development
- Code written by Titus unless otherwise noted.
- AI assistance provided by claude.ai, GitHub Copilot, and Ollama using multiple models. Some documentation is in AiUsage.md. Developed with the Zed IDE.

## Architecture

### Frontend versus backend
This project is modular and split up into at least two main parts. The frontend is in `/frontend/*`, and is a react vite SPA. See `./frontend/routes.ts` for the available routes. The backend most of the rest of the code, but runs `./main.go` to provide the APIs for the frontend.
This uses the supabase APIs.

### Database Layer Architecture

The application uses a layered architecture for database access:

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Handlers                            │
│              (admin/handler.go, apis/*)                     │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ calls functions
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                 Models Package                              │
│                (src/pkg/models/*)                           │
│  ┌─────────────────┬────────────────────────────────────┐   │
│  │ UpsertUser()    │ GetStoryData()    │ SaveStory()    │   │
│  │ CanUserAccess() │ GetAllStories()   │ DeleteStory()  │   │
│  │ IsUserAdmin()   │ GetLineText()     │ EditStory()    │   │
│  └─────────────────┴────────────────────────────────────┘   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ uses SQLC queries
                      ▼
┌─────────────────────────────────────────────────────────────┐
│            Generated SQLC Queries                           │
│            (src/pkg/generated/db/*)                          │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ queries.UpsertUser()    │ queries.GetStoryData()    │    │
│  │ queries.CanUserAccess() │ queries.GetAllStories()   │    │
│  │ queries.IsUserAdmin()   │ queries.SaveStory()       │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ uses database connection
                      ▼
┌─────────────────────────────────────────────────────────────┐
│          Database Connection                                │
│          (pgxpool.Pool)                                     │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ executes type-safe SQL
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   Supabase                                  │
└─────────────────────────────────────────────────────────────┘
```

**Key Points:**
- HTTP handlers call model functions, never database directly
- Models package uses generated SQLC queries for type-safe database operations
- SQLC generates Go code from SQL queries, providing compile-time safety
- Models package adds business logic layer on top of generated queries
- Authentication middleware calls model functions for user operations

### Academic Context
This project was, in its first part, developed under the oversight of Dr. Derrick Tate for academic credit at Sattler College.
