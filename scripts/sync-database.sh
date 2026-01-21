#!/usr/bin/env bash
set -euo pipefail

# Sync a single database by name
# Usage: ./scripts/sync-database.sh <database-name>
#
# Prerequisites:
#   - PostgreSQL 17 installed locally (brew install postgresql@17)
#
# Example:
#   ./scripts/sync-database.sh my-production-db

# Use PostgreSQL 17 tools (required for connecting to PG 17 servers)
export PATH="/opt/homebrew/opt/postgresql@17/bin:$PATH"

if [ $# -ne 1 ]; then
    echo "Usage: $0 <database-name>"
    echo "Example: $0 my-production-db"
    exit 1
fi

DATABASE_NAME="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Load .env file if it exists
if [[ -f "$PROJECT_ROOT/.env" ]]; then
    echo "Loading configuration from .env..."
    set -a
    source "$PROJECT_ROOT/.env"
    set +a
else
    echo "Error: .env file not found"
    echo "Copy .env.example to .env and fill in your credentials:"
    echo "  cp .env.example .env"
    exit 1
fi

# Build the binary
echo "Building auto-db-backups..."
cd "$PROJECT_ROOT"
CGO_ENABLED=1 go build -o auto-db-backups .

# Run the backup for specific database
echo "Running backup for database: $DATABASE_NAME"
./auto-db-backups --database "$DATABASE_NAME"
