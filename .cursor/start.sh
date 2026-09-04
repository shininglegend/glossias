#!/usr/bin/env bash
# Per-boot reconciliation: bring PostgreSQL up and materialize the frontend
# Clerk key from the injected secret. Must tolerate restarts and then return.
set -euo pipefail

cd "$(dirname "$0")/.."

SUDO=""
if command -v sudo >/dev/null 2>&1; then SUDO="sudo"; fi

PG_VERSION=16

# Start PostgreSQL (idempotent) and wait for readiness.
$SUDO pg_ctlcluster "$PG_VERSION" main start 2>/dev/null || true
for _ in $(seq 1 60); do
  pg_isready -h 127.0.0.1 -p 5432 >/dev/null 2>&1 && break
  sleep 1
done
pg_isready -h 127.0.0.1 -p 5432

# Materialize the frontend Clerk publishable key from the injected secret every
# boot. Falls back to a well-formed placeholder so the dev server always starts;
# the placeholder cannot reach Clerk, so real sign-in needs the real secret.
if [ -n "${VITE_CLERK_PUBLISHABLE_KEY:-}" ]; then
  printf 'VITE_CLERK_PUBLISHABLE_KEY=%s\n' "$VITE_CLERK_PUBLISHABLE_KEY" > frontend/.env
elif [ ! -f frontend/.env ]; then
  printf 'VITE_CLERK_PUBLISHABLE_KEY=%s\n' 'pk_test_Y2xlcmsuZXhhbXBsZS5jb20k' > frontend/.env
fi

echo "start.sh complete"
