#!/usr/bin/env bash
set -euo pipefail

# Restore a backup to a local PostgreSQL database
# Usage: ./scripts/restore-backup.sh <backup-file.dump.gz.enc> <database-name>
#
# Prerequisites:
#   - PostgreSQL installed locally (brew install postgresql)
#   - ENCRYPTION_KEY environment variable set (from .env)
#
# Example:
#   source .env
#   ./scripts/restore-backup.sh backup.dump.gz.enc test_restore

if [ $# -ne 2 ]; then
    echo "Usage: $0 <backup-file> <database-name>"
    echo "Example: $0 backup.dump.gz.enc test_restore"
    exit 1
fi

BACKUP_FILE="$1"
DB_NAME="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Check if ENCRYPTION_KEY is set
if [ -z "${ENCRYPTION_KEY:-}" ]; then
    echo "Error: ENCRYPTION_KEY not set"
    echo "Run: source .env"
    exit 1
fi

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "==> Decrypting backup..."
go run "$PROJECT_ROOT/scripts/decrypt-backup.go" "$BACKUP_FILE" "$TEMP_DIR/backup.dump.gz"

echo "==> Decompressing backup..."
gunzip -c "$TEMP_DIR/backup.dump.gz" > "$TEMP_DIR/backup.dump"

echo "==> Creating database '$DB_NAME'..."
createdb "$DB_NAME" 2>/dev/null || echo "Database already exists, will restore into it"

echo "==> Restoring backup..."
pg_restore --clean --if-exists --no-owner --no-privileges -d "$DB_NAME" "$TEMP_DIR/backup.dump" || true

echo ""
echo "==> Done! Connect with: psql $DB_NAME"
echo ""
echo "Quick verification:"
psql "$DB_NAME" -c "\dt" 2>/dev/null || echo "(run 'psql $DB_NAME' to explore)"
