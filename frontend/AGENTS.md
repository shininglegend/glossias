# Frontend

React 19 · React Router 7 (SPA mode) · Vite 6 · Tailwind CSS 4 · TypeScript 5 · Clerk React

Repo-wide backend, DB, and auth conventions live in the root [`AGENTS.md`](../AGENTS.md).

## Running locally

Port 5173. Proxies `/api` to the Go backend on `:8080`. Both must run concurrently — no CORS config needed in dev. If you start Vite or the backend to test, stop it when finished — see the root `AGENTS.md`.

```bash
cd frontend
npm install
npm run dev
```

## Building

```bash
cd frontend && npm run build   # output: frontend/build/
```

## After every non-trivial change

`format`, `lint`, `typecheck`, `test`, and `build` are scripts on `frontend/package.json`. The repo root has no those scripts — `npm run format` from the root fails. `cd frontend` first; do not invoke `prettier` / `eslint` / `tsc` / `vitest` binaries directly.

```bash
cd frontend
npm run format    # prettier
npm run lint      # eslint
npm run typecheck
npm run test
npm run build
```

## Environment variables

- `VITE_CLERK_PUBLISHABLE_KEY`

## Directory layout

```
app/
  routes/                 # File-based page components (admin.*, stories-*)
  components/             # Reusable UI components
  contexts/               # React Context (UserContext)
  services/               # API call helpers
  types/                  # Shared TypeScript types
vite.config.ts            # Proxy config
react-router.config.ts    # SPA mode (SSR: false)
```

## Auth

Clerk via `ClerkProvider` in `app/root.tsx`. Roles: `super_admin`, `course_admin`, `student`. Backend JWT and `DEV_USER` are documented in the root `AGENTS.md`.

## Routing

File-based under `app/routes/`. Student API: `/api/*`. Admin API: `/api/admin/*`.

## Known issues

- Several large "god components" (~400–640 lines with 15+ state variables).
