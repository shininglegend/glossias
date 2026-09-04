#!/usr/bin/env bash
# Idempotent bootstrap for the Glossias Cloud Agent environment.
# Runs once to build the environment snapshot: installs PostgreSQL, project
# dependencies, applies the database schema, and seeds a super-admin dev user.
set -euo pipefail

cd "$(dirname "$0")/.."

SUDO=""
if command -v sudo >/dev/null 2>&1; then SUDO="sudo"; fi

PG_VERSION=16
DB_NAME=glossias
DB_URL="postgresql://postgres:postgres@127.0.0.1:5432/${DB_NAME}?sslmode=disable"

# 1. System dependency: PostgreSQL (the backend talks to Postgres directly).
if ! command -v pg_ctlcluster >/dev/null 2>&1; then
  $SUDO apt-get update -qq
  $SUDO DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
    postgresql postgresql-contrib >/dev/null
fi

# 2. Start the cluster so the schema can be created at build time.
$SUDO pg_ctlcluster "$PG_VERSION" main start 2>/dev/null || true
for _ in $(seq 1 30); do
  pg_isready -h 127.0.0.1 -p 5432 >/dev/null 2>&1 && break
  sleep 1
done

# 3. Role and database (idempotent).
$SUDO -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" >/dev/null
if ! $SUDO -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" | grep -q 1; then
  $SUDO -u postgres createdb "$DB_NAME"
fi

# 4. Project dependencies.
go mod download
(cd frontend && npm ci)

# 5. Apply migrations with goose (the same files the server embeds). Baking the
#    schema into the snapshot lets the server skip migrations on every boot.
GOBIN="$(go env GOPATH)/bin"
if [ ! -x "$GOBIN/goose" ]; then
  go install github.com/pressly/goose/v3/cmd/goose@v3.27.3
fi
"$GOBIN/goose" -dir src/pkg/database/migrations postgres "$DB_URL" up

# 6. Seed a super-admin dev user so the `dev_auth: 12345678` header shortcut
#    (see src/auth/auth.go) returns admin data without a live Clerk account.
PGPASSWORD=postgres psql -h 127.0.0.1 -U postgres -d "$DB_NAME" -v ON_ERROR_STOP=1 -c \
  "INSERT INTO users (user_id, email, name, is_super_admin)
   VALUES ('dev-admin', 'dev-admin@example.com', 'Dev Admin', true)
   ON CONFLICT (user_id) DO UPDATE SET is_super_admin = true;"

# 7. Backend .env with local, non-secret config. Injected secrets
#    (CLERK_SECRET_KEY, STORAGE_URL, STORAGE_API_KEY, ANTHROPIC_API_KEY) arrive
#    as environment variables and take precedence: godotenv does not override
#    variables already present in the environment.
if [ ! -f .env ]; then
  cat > .env <<EOF
PORT=8080
DATABASE_URL="${DB_URL}"
AUTHORIZED_PARTY=http://localhost:5173
LOG_LEVEL=INFO
DEV_USER=dev-admin
EOF
fi

echo "install.sh complete"
