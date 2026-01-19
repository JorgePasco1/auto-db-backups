# Auto DB Backups

[![CI](https://github.com/JorgePasco1/auto-db-backups/actions/workflows/ci.yml/badge.svg)](https://github.com/JorgePasco1/auto-db-backups/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/JorgePasco1/auto-db-backups)](https://goreportcard.com/report/github.com/JorgePasco1/auto-db-backups)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A GitHub Action and CLI tool that automatically backs up PostgreSQL, MySQL, or MongoDB databases to Cloudflare R2 storage.

## Features

- **Multi-database support** - Back up multiple databases in a single run
- **Multiple database types** - PostgreSQL, MySQL, MongoDB
- **Cloudflare R2 storage** - Cost-effective S3-compatible object storage
- **Compression** - Gzip compression to reduce storage costs
- **Encryption** - AES-256-GCM encryption for sensitive data
- **Retention policies** - Automatically delete old backups by age or count
- **Webhook notifications** - Get notified on success or failure (Slack, Discord, etc.)
- **Flexible execution** - Run as GitHub Action or locally via CLI

## Quick Start

### GitHub Actions (Recommended)

1. **Create the workflow file** `.github/workflows/backup.yml`:

```yaml
name: Backup Databases

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM UTC
  workflow_dispatch:       # Manual trigger

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run backup
        env:
          R2_ACCOUNT_ID: ${{ secrets.R2_ACCOUNT_ID }}
          R2_ACCESS_KEY_ID: ${{ secrets.R2_ACCESS_KEY_ID }}
          R2_SECRET_ACCESS_KEY: ${{ secrets.R2_SECRET_ACCESS_KEY }}
          R2_BUCKET_NAME: ${{ secrets.R2_BUCKET_NAME }}
          DATABASE_TYPE: postgres
          DATABASE_CONNECTION_1: ${{ secrets.DATABASE_CONNECTION_1 }}
          DATABASE_NAME_1: ${{ secrets.DATABASE_NAME_1 }}
          ENCRYPTION_KEY: ${{ secrets.ENCRYPTION_KEY }}
          COMPRESSION: true
          RETENTION_DAYS: 7
        run: |
          go build -o auto-db-backups .
          ./auto-db-backups
```

2. **Add secrets** in your repository (Settings → Secrets and variables → Actions):

| Secret | Description |
|--------|-------------|
| `R2_ACCOUNT_ID` | Your Cloudflare account ID |
| `R2_ACCESS_KEY_ID` | R2 API access key |
| `R2_SECRET_ACCESS_KEY` | R2 API secret key |
| `R2_BUCKET_NAME` | Name of your R2 bucket |
| `DATABASE_CONNECTION_1` | Database connection string |
| `DATABASE_NAME_1` | Friendly name for the backup |
| `ENCRYPTION_KEY` | Base64-encoded 32-byte key (optional) |

You can set secrets via the GitHub CLI:
```bash
gh secret set R2_ACCOUNT_ID
gh secret set DATABASE_NAME_1 --body "my-app-db"
```

3. **Trigger the workflow** manually or wait for the schedule.

### Local Execution

1. **Clone and configure**:
```bash
git clone https://github.com/jorgepascosoto/auto-db-backups.git
cd auto-db-backups
cp .env.example .env
# Edit .env with your credentials
```

2. **Run the backup**:
```bash
./scripts/run-local.sh
```

## Configuration

### Environment Variables

#### R2 Storage (Required)

| Variable | Description |
|----------|-------------|
| `R2_ACCOUNT_ID` | Cloudflare account ID |
| `R2_ACCESS_KEY_ID` | R2 access key ID |
| `R2_SECRET_ACCESS_KEY` | R2 secret access key |
| `R2_BUCKET_NAME` | Bucket name for storing backups |

#### Database Connections

For multiple databases, use numbered suffixes:

| Variable | Description |
|----------|-------------|
| `DATABASE_TYPE` | `postgres`, `mysql`, or `mongodb` (default: `postgres`) |
| `DATABASE_CONNECTION_1` | Connection string for first database |
| `DATABASE_NAME_1` | Custom name for backup files (optional) |
| `DATABASE_PREFIX_1` | Custom R2 prefix path (optional) |
| `DATABASE_CONNECTION_2` | Connection string for second database |
| `DATABASE_NAME_2` | Custom name for second database |
| ... | Add more as needed |

Connection string formats:
```
# PostgreSQL
postgresql://user:password@host:5432/dbname?sslmode=require

# MySQL
mysql://user:password@host:3306/dbname

# MongoDB
mongodb://user:password@host:27017/dbname
```

#### Backup Options

| Variable | Default | Description |
|----------|---------|-------------|
| `COMPRESSION` | `true` | Enable gzip compression |
| `ENCRYPTION_KEY` | - | Base64-encoded 32-byte key for AES-256-GCM |
| `RETENTION_DAYS` | `0` | Delete backups older than N days (0 = disabled) |
| `RETENTION_COUNT` | `0` | Keep only last N backups (0 = disabled) |

#### Notifications

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_URL` | - | Webhook URL (Slack, Discord, etc.) |
| `NOTIFY_ON_SUCCESS` | `true` | Send notification on success |
| `NOTIFY_ON_FAILURE` | `true` | Send notification on failure |

### Generating an Encryption Key

```bash
openssl rand -base64 32
```

Store this key securely - you'll need it to decrypt backups.

## Adding a New Database

To add another database to your backups:

1. Add secrets:
```bash
gh secret set DATABASE_CONNECTION_4
gh secret set DATABASE_NAME_4 --body "new-database"
```

2. Add to workflow env section:
```yaml
DATABASE_CONNECTION_4: ${{ secrets.DATABASE_CONNECTION_4 }}
DATABASE_NAME_4: ${{ secrets.DATABASE_NAME_4 }}
```

## Backup File Naming

Backup files follow this pattern:
```
backups/<database-name>/<type>-<name>-<timestamp>.<ext>[.gz][.enc]
```

Example:
```
backups/my-app/postgres-my-app-20240115-140532.dump.gz.enc
```

## Restoring Backups

### Download from R2

Using the AWS CLI (configured for R2):
```bash
aws s3 cp s3://your-bucket/backups/my-app/postgres-my-app-20240115-140532.dump.gz.enc ./backup.dump.gz.enc \
  --endpoint-url https://<account-id>.r2.cloudflarestorage.com
```

### Decrypt (if encrypted)

```bash
# Using OpenSSL (the first 12 bytes are the nonce)
# You'll need a decryption tool that supports AES-256-GCM
# See the decrypt example in scripts/ directory
```

### Decompress (if compressed)

```bash
gunzip backup.dump.gz
```

### Restore PostgreSQL

```bash
pg_restore -h localhost -U user -d dbname backup.dump
```

---

# Developer Documentation

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Config    │────▶│    Backup    │────▶│   Storage   │
│   Loader    │     │   Pipeline   │     │   (R2)      │
└─────────────┘     └──────────────┘     └─────────────┘
                           │
                    ┌──────┴──────┐
                    ▼             ▼
              ┌──────────┐  ┌──────────┐
              │ Compress │  │ Encrypt  │
              │  (gzip)  │  │(AES-256) │
              └──────────┘  └──────────┘
```

The backup pipeline:
1. **Config Loader** - Reads environment variables and validates configuration
2. **Database Exporter** - Executes native dump tools (`pg_dump`, `mysqldump`, `mongodump`)
3. **Compression** - Optional gzip compression via streaming
4. **Encryption** - Optional AES-256-GCM encryption
5. **Upload** - Streams data to Cloudflare R2
6. **Retention** - Applies cleanup policies
7. **Notifications** - Sends webhook notifications

## Project Structure

```
.
├── main.go                 # Entry point, orchestrates backup flow
├── action.yml              # GitHub Action definition
├── Dockerfile              # Container for GitHub Action
├── internal/
│   ├── backup/
│   │   ├── exporter.go     # Exporter interface
│   │   ├── factory.go      # Creates exporters by database type
│   │   ├── postgres.go     # PostgreSQL exporter (pg_dump)
│   │   ├── mysql.go        # MySQL exporter (mysqldump)
│   │   └── mongodb.go      # MongoDB exporter (mongodump)
│   ├── compress/
│   │   └── gzip.go         # Gzip compression with streaming
│   ├── config/
│   │   └── config.go       # Configuration loading and validation
│   ├── encrypt/
│   │   └── aes.go          # AES-256-GCM encryption
│   ├── errors/
│   │   └── errors.go       # Custom error types
│   ├── notify/
│   │   ├── webhook.go      # Webhook notifications
│   │   └── summary.go      # GitHub Actions summary
│   └── storage/
│       ├── r2.go           # Cloudflare R2 client
│       └── retention.go    # Backup retention policies
├── scripts/
│   └── run-local.sh        # Local execution script
└── .github/workflows/
    ├── backup-databases.yml # Example backup workflow
    └── ci.yml              # CI pipeline
```

## Key Interfaces

### Exporter Interface

```go
type Exporter interface {
    Export(ctx context.Context) (io.ReadCloser, error)
}
```

Each database type implements this interface using native dump tools.

### Adding a New Database Type

1. Create `internal/backup/newdb.go`:
```go
type NewDBExporter struct {
    db *config.DatabaseConfig
}

func (e *NewDBExporter) Export(ctx context.Context) (io.ReadCloser, error) {
    cmd := exec.CommandContext(ctx, "newdb-dump", ...)
    // Set up stdout pipe and return
}
```

2. Register in `internal/backup/factory.go`:
```go
case config.DatabaseTypeNewDB:
    return NewNewDBExporter(db), nil
```

3. Add type in `internal/config/config.go`:
```go
const DatabaseTypeNewDB DatabaseType = "newdb"
```

4. Update the Dockerfile to include the dump tool.

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/config/...
```

## Building

```bash
# Build binary
go build -o auto-db-backups .

# Build with CGO (required for some TLS features)
CGO_ENABLED=1 go build -o auto-db-backups .

# Build Docker image
docker build -t auto-db-backups .
```

## CI Pipeline

The CI workflow (`.github/workflows/ci.yml`) runs:
- `gofmt` - Code formatting check
- `go vet` - Static analysis
- `go test` - Unit tests

## Environment Variable Resolution

The config loader checks multiple sources in order:
1. `INPUT_*` prefixed variables (GitHub Actions convention)
2. Regular environment variables
3. Numbered suffixes for multi-database support (`DATABASE_CONNECTION_1`, `DATABASE_CONNECTION_2`, etc.)

## Error Handling

Custom error types in `internal/errors/`:
- `BackupError` - Database export failures
- `StorageError` - R2 upload/download failures
- `ConfigError` - Configuration validation errors

Errors include context about the database and operation for easier debugging.

## Security Considerations

- Connection strings and encryption keys should only be stored in secrets
- The encryption key must be 32 bytes (256 bits) for AES-256
- Backup files in R2 should have appropriate access controls
- Consider enabling R2 bucket versioning for additional protection

## License

MIT
