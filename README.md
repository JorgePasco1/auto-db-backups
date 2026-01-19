# Auto DB Backups

[![CI](https://github.com/JorgePasco1/auto-db-backups/actions/workflows/ci.yml/badge.svg)](https://github.com/JorgePasco1/auto-db-backups/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/JorgePasco1/auto-db-backups)](https://goreportcard.com/report/github.com/JorgePasco1/auto-db-backups)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A template repository for automatically backing up PostgreSQL, MySQL, or MongoDB databases to Cloudflare R2 storage using GitHub Actions.

## Features

- **Multi-database support** - Back up multiple databases in a single run
- **Multiple database types** - PostgreSQL, MySQL, MongoDB
- **Cloudflare R2 storage** - Cost-effective S3-compatible object storage
- **Compression** - Gzip compression to reduce storage costs
- **Encryption** - AES-256-GCM encryption for sensitive data
- **Retention policies** - Automatically delete old backups by age or count
- **Webhook notifications** - Get notified on success or failure (Slack, Discord, etc.)
- **Template repository** - Fork and configure with your own secrets
- **Local testing** - Run backups locally before deploying

## Setup Guide

### Prerequisites

- A GitHub account
- A database to back up (PostgreSQL, MySQL, or MongoDB)
- A Cloudflare account (free tier is sufficient)

### Step 0: Fork This Repository

1. Click the **Fork** button at the top of this repository
2. This creates your own copy where backups will run automatically

### Step 1: Set Up Cloudflare R2

Cloudflare R2 is an S3-compatible object storage service with zero egress fees, making it ideal for backups.

1. **Create a Cloudflare account** (if you don't have one):
   - Go to [dash.cloudflare.com](https://dash.cloudflare.com/sign-up)
   - Sign up for a free account

2. **Enable R2**:
   - From the Cloudflare dashboard, navigate to **R2** in the left sidebar
   - Click **Purchase R2 Plan** (the free tier includes 10 GB storage)
   - Accept the terms and enable R2

3. **Create an R2 bucket**:
   - Click **Create bucket**
   - Enter a bucket name (e.g., `db-backups`)
   - Choose a location (optional, or use Automatic)
   - Click **Create bucket**

4. **Create R2 API tokens**:
   - In the R2 section, click **Manage R2 API Tokens**
   - Click **Create API Token**
   - Configure the token:
     - **Token name**: `auto-db-backups`
     - **Permissions**: Select **Object Read & Write**
     - **Specify bucket(s)**: Choose the bucket you created
   - Click **Create API Token**
   - **Important**: Copy these values immediately (they won't be shown again):
     - Access Key ID
     - Secret Access Key
     - Endpoint URL (you'll extract the Account ID from this)

5. **Get your Account ID**:
   - Your endpoint URL looks like: `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`
   - Extract the `<ACCOUNT_ID>` portion - this is your R2 Account ID
   - Alternatively, find it on the R2 overview page

### Step 2: Get Your Database Connection String

#### For Neon (PostgreSQL)

1. Go to your [Neon Console](https://console.neon.tech)
2. Select your project and database
3. Click **Connection Details**
4. Copy the connection string (it looks like):
   ```
   postgresql://user:password@ep-cool-name-123456.us-east-2.aws.neon.tech/dbname?sslmode=require
   ```

#### For Other PostgreSQL Providers

- **Supabase**: Project Settings → Database → Connection string → URI
- **Railway**: Your Project → Database → Connect → Connection URL
- **Heroku**: App Dashboard → Settings → Config Vars → DATABASE_URL
- **AWS RDS**: Endpoint + credentials in the format:
  ```
  postgresql://username:password@endpoint:5432/dbname
  ```

#### For MySQL Providers

- **PlanetScale**: Dashboard → Connect → Select "General" → Copy connection string
- **AWS RDS MySQL**: Similar to PostgreSQL format:
  ```
  mysql://username:password@endpoint:3306/dbname
  ```

#### For MongoDB Providers

- **MongoDB Atlas**: Clusters → Connect → Connect your application → Copy connection string
- Format:
  ```
  mongodb+srv://username:password@cluster.mongodb.net/dbname
  ```

### Step 3: Generate an Encryption Key (Optional but Recommended)

Encrypt your backups for security:

```bash
openssl rand -base64 32
```

Save this key securely - you'll need it to decrypt backups later.

### Step 4: Configure GitHub Secrets

Add the following secrets to your GitHub repository:

1. Go to your repository on GitHub
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret** for each:

| Secret Name | Value | Where to Find It |
|-------------|-------|------------------|
| `R2_ACCOUNT_ID` | Your Cloudflare account ID | From Step 1.5 above |
| `R2_ACCESS_KEY_ID` | R2 API access key | From Step 1.4 above |
| `R2_SECRET_ACCESS_KEY` | R2 API secret key | From Step 1.4 above |
| `R2_BUCKET_NAME` | Your R2 bucket name | From Step 1.3 above (e.g., `db-backups`) |
| `DATABASE_CONNECTION_1` | Your database connection string | From Step 2 above |
| `DATABASE_NAME_1` | Friendly name for backups | Any name (e.g., `my-app-prod`) |
| `ENCRYPTION_KEY` | Base64 encryption key | From Step 3 above (optional) |

Using the GitHub CLI (faster):
```bash
gh secret set R2_ACCOUNT_ID --body "your-account-id"
gh secret set R2_ACCESS_KEY_ID --body "your-access-key"
gh secret set R2_SECRET_ACCESS_KEY --body "your-secret-key"
gh secret set R2_BUCKET_NAME --body "db-backups"
gh secret set DATABASE_CONNECTION_1 --body "postgresql://..."
gh secret set DATABASE_NAME_1 --body "my-app-prod"
gh secret set ENCRYPTION_KEY --body "your-base64-key"
```

### Step 5: Configure the Workflow (Optional)

The forked repository already includes `.github/workflows/backup-databases.yml` which is configured to:
- Run daily at 2 AM UTC (`cron: '0 2 * * *'`)
- Back up 3 databases (you can add more by adding `DATABASE_CONNECTION_4`, etc.)
- Use PostgreSQL 17

**To customize the workflow:**

1. Edit `.github/workflows/backup-databases.yml` in your fork
2. Adjust the schedule if needed:
   ```yaml
   schedule:
     - cron: '0 2 * * *'  # Change time here (UTC)
   ```
3. Add more database connections:
   ```yaml
   DATABASE_CONNECTION_4: ${{ secrets.DATABASE_CONNECTION_4 }}
   DATABASE_NAME_4: ${{ secrets.DATABASE_NAME_4 }}
   ```
4. Adjust retention settings:
   ```yaml
   RETENTION_DAYS: 7      # Keep backups for 7 days
   RETENTION_COUNT: 30    # Keep last 30 backups
   ```

**For MySQL or MongoDB:**
- Change `DATABASE_TYPE: postgres` to `mysql` or `mongodb`
- Update the client installation step in the workflow

### Step 6: Test Your Setup

1. **Manual trigger**:
   - Go to **Actions** tab in your GitHub repository
   - Select **Backup Databases** workflow
   - Click **Run workflow** → **Run workflow**

2. **Check the results**:
   - Watch the workflow run and check for any errors
   - If successful, verify the backup in your R2 bucket:
     - Go to Cloudflare dashboard → R2 → Your bucket
     - You should see a file like: `backups/my-app-prod/postgres-my-app-prod-20240115-140532.dump.gz.enc`

3. **Common issues**:
   - **Connection refused**: Check your database connection string and firewall rules
   - **Access denied**: Verify your R2 API token has read/write permissions
   - **Command not found**: Make sure the workflow has the correct Go version

### Optional: Set Up Notifications

Get notified when backups succeed or fail via Slack, Discord, or any webhook-compatible service.

#### For Slack:

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click **Create New App** → **From scratch**
3. Name your app (e.g., "Database Backups") and select your workspace
4. Go to **Incoming Webhooks** and toggle **Activate Incoming Webhooks**
5. Click **Add New Webhook to Workspace**
6. Choose a channel and click **Allow**
7. Copy the webhook URL (looks like: `https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX`)

#### For Discord:

1. Go to your Discord server settings
2. Navigate to **Integrations** → **Webhooks**
3. Click **New Webhook**
4. Name it (e.g., "DB Backups"), choose a channel
5. Copy the webhook URL

#### Add to GitHub Secrets:

```bash
gh secret set WEBHOOK_URL --body "https://hooks.slack.com/services/..."
```

Or add it manually in GitHub repository settings.

#### Update your workflow:

Add these environment variables to your `.github/workflows/backup.yml`:
```yaml
env:
  # ... existing env vars ...
  WEBHOOK_URL: ${{ secrets.WEBHOOK_URL }}
  NOTIFY_ON_SUCCESS: true
  NOTIFY_ON_FAILURE: true
```

Now you'll receive notifications for all backup operations.

## Quick Start

1. **Fork this repository** to your GitHub account

2. **Add secrets** to your fork (Settings → Secrets and variables → Actions):
   ```bash
   gh secret set R2_ACCOUNT_ID --body "your-account-id"
   gh secret set R2_ACCESS_KEY_ID --body "your-access-key"
   gh secret set R2_SECRET_ACCESS_KEY --body "your-secret-key"
   gh secret set R2_BUCKET_NAME --body "db-backups"
   gh secret set DATABASE_CONNECTION_1 --body "postgresql://..."
   gh secret set DATABASE_NAME_1 --body "my-app-prod"
   gh secret set ENCRYPTION_KEY --body "$(openssl rand -base64 32)"
   ```

3. **Enable GitHub Actions** in your fork (Actions tab → Enable workflows)

4. **Test it** by going to Actions → Backup Neon Databases → Run workflow

That's it! Backups will run daily at 2 AM UTC automatically.

See the [Setup Guide](#setup-guide) below for detailed step-by-step instructions.

### Local Execution (Alternative)

If you prefer to run backups locally or want to test before setting up GitHub Actions:

1. **Clone and configure**:
```bash
git clone https://github.com/jorgepascosoto/auto-db-backups.git
cd auto-db-backups
```

2. **Create a `.env` file** with your credentials:
```bash
# Copy the example file
cp .env.example .env

# Edit .env with your actual values
nano .env  # or use your preferred editor
```

Example `.env` file:
```bash
R2_ACCOUNT_ID=your-account-id
R2_ACCESS_KEY_ID=your-access-key-id
R2_SECRET_ACCESS_KEY=your-secret-access-key
R2_BUCKET_NAME=db-backups

DATABASE_TYPE=postgres
DATABASE_CONNECTION_1=postgresql://user:pass@host:5432/dbname
DATABASE_NAME_1=my-app-prod

ENCRYPTION_KEY=your-base64-encryption-key
COMPRESSION=true
RETENTION_DAYS=7
```

3. **Install Go** (if not already installed):
```bash
# macOS
brew install go

# Ubuntu/Debian
sudo apt install golang

# Verify installation
go version  # Should be 1.21 or higher
```

4. **Run the backup**:
```bash
./scripts/run-local.sh
```

The script will build the binary and run the backup using your `.env` configuration.

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

### Step 1: Download from R2

#### Option A: Using the Cloudflare Dashboard

1. Go to your Cloudflare dashboard → R2 → Your bucket
2. Navigate to the backup file you want to restore
3. Click the file and select **Download**

#### Option B: Using AWS CLI

Configure the AWS CLI for R2:
```bash
aws configure
# AWS Access Key ID: <your-r2-access-key-id>
# AWS Secret Access Key: <your-r2-secret-access-key>
# Default region: auto
```

Download the backup:
```bash
aws s3 cp s3://your-bucket/backups/my-app/postgres-my-app-20240115-140532.dump.gz.enc ./backup.dump.gz.enc \
  --endpoint-url https://<account-id>.r2.cloudflarestorage.com
```

### Step 2: Restore Using the Automated Script (Easiest)

This repository includes a convenience script that handles decryption, decompression, and restoration automatically.

1. **Set up environment**:
```bash
# Clone the repository if you haven't already
git clone https://github.com/jorgepascosoto/auto-db-backups.git
cd auto-db-backups

# Create .env file with your encryption key
echo "ENCRYPTION_KEY=your-base64-encryption-key" > .env
source .env
```

2. **Install PostgreSQL locally** (if not already installed):
```bash
# macOS
brew install postgresql@17

# Ubuntu/Debian
sudo apt install postgresql-17

# The script uses PostgreSQL 17 to match the version used for backups
```

3. **Run the restore script**:
```bash
./scripts/restore-backup.sh backup.dump.gz.enc my_restored_db
```

This will:
- Decrypt the backup using your `ENCRYPTION_KEY`
- Decompress the file
- Create a new database called `my_restored_db`
- Restore the data into it
- Display connection instructions

4. **Connect to the restored database**:
```bash
psql my_restored_db
```

### Manual Restoration (Advanced)

If you prefer to restore manually or need more control:

#### Step 2a: Decrypt (if encrypted)

```bash
# Set your encryption key
export ENCRYPTION_KEY="your-base64-encryption-key"

# Decrypt using the provided script
go run scripts/decrypt-backup.go backup.dump.gz.enc backup.dump.gz
```

#### Step 2b: Decompress (if compressed)

```bash
gunzip backup.dump.gz
```

#### Step 2c: Restore to PostgreSQL

```bash
# Create the database
createdb my_database

# Restore the backup
pg_restore --clean --if-exists --no-owner --no-privileges -d my_database backup.dump
```

For MySQL:
```bash
mysql -u user -p dbname < backup.sql
```

For MongoDB:
```bash
mongorestore --archive=backup.archive
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
│   ├── run-local.sh        # Local execution script
│   ├── decrypt-backup.go   # Decrypt backup files
│   └── restore-backup.sh   # Automated restore script
└── .github/workflows/
    ├── backup-databases.yml # Main backup workflow
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

4. Update `.github/workflows/backup-databases.yml` to install the required database client.

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

# Build with CGO (required for proper TLS/crypto support)
CGO_ENABLED=1 go build -o auto-db-backups .

# Run locally
./scripts/run-local.sh
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
