#!/usr/bin/env bash
set -euo pipefail

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

# Run the backup
echo "Running backup..."
./auto-db-backups
