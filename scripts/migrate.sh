#!/usr/bin/env bash
# scripts/migrate.sh
#
# Runs or rolls back database migrations using golang-migrate.
# Requires the `migrate` CLI: https://github.com/golang-migrate/migrate
#
# Usage:
#   ./scripts/migrate.sh          # apply all pending migrations
#   ./scripts/migrate.sh rollback # roll back the last applied migration
#   ./scripts/migrate.sh drop     # drop everything (DANGER: use only in dev)

set -euo pipefail

MIGRATION_DIR="./database/migrations"

# Build the DATABASE_URL from individual env vars when not already set.
if [[ -z "${DATABASE_URL:-}" ]]; then
  DB_DRIVER="${DB_DRIVER:-postgres}"
  if [[ "$DB_DRIVER" != "postgres" ]]; then
    echo >&2 "ERROR: PostgreSQL is the supported database; DB_DRIVER must be postgres."
    exit 1
  fi
  DB_HOST="${DB_HOST:-localhost}"
  DB_PORT="${DB_PORT:-5432}"
  DB_DATABASE="${DB_DATABASE:-gostore}"
  DB_USERNAME="${DB_USERNAME:-postgres}"
  DB_PASSWORD="${DB_PASSWORD:-}"
  DB_SSL_MODE="${DB_SSL_MODE:-disable}"

  urlencode() {
    local LC_ALL=C input="$1" output="" char encoded i
    for ((i=0; i<${#input}; i++)); do
      char="${input:i:1}"
      case "$char" in
        [a-zA-Z0-9.~_-]) output+="$char" ;;
        *) printf -v encoded '%%%02X' "'${char}"; output+="$encoded" ;;
      esac
    done
    printf '%s' "$output"
  }

  DATABASE_URL="postgres://$(urlencode "$DB_USERNAME"):$(urlencode "$DB_PASSWORD")@${DB_HOST}:${DB_PORT}/$(urlencode "$DB_DATABASE")?sslmode=${DB_SSL_MODE}"
fi

case "$DATABASE_URL" in
  postgres://*|postgresql://*) ;;
  *) echo >&2 "ERROR: DATABASE_URL must use postgres:// or postgresql://"; exit 1 ;;
esac

command -v migrate >/dev/null 2>&1 || {
  echo >&2 "ERROR: 'migrate' CLI not found. Install it:"
  echo >&2 "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
  exit 1
}

case "${1:-up}" in
  up)
    echo "Applying migrations…"
    migrate -path "$MIGRATION_DIR" -database "$DATABASE_URL" up
    echo "Migrations applied."
    ;;
  rollback|down)
    echo "Rolling back last migration…"
    migrate -path "$MIGRATION_DIR" -database "$DATABASE_URL" down 1
    echo "Rollback complete."
    ;;
  drop)
    echo "WARNING: dropping all tables in 5 seconds. Ctrl-C to abort."
    sleep 5
    migrate -path "$MIGRATION_DIR" -database "$DATABASE_URL" drop -f
    echo "All tables dropped."
    ;;
  *)
    echo "Usage: $0 [up|rollback|drop]"
    exit 1
    ;;
esac
