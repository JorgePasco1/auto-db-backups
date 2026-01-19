# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Auto DB Backups is a **template/forkable repository** for automated database backups. Users fork this repo, configure their secrets, and the included GitHub Actions workflow automatically backs up their databases.

## Usage Pattern

1. User forks this repository
2. User adds secrets to their fork (R2 credentials, database connections, encryption key)
3. The included `.github/workflows/backup-databases.yml` runs automatically on schedule (daily at 2 AM UTC)
4. Backups are stored in the user's Cloudflare R2 bucket

## Architecture

This is a Go-based CLI tool that runs in GitHub Actions:

### GitHub Actions Workflow
- `.github/workflows/backup-databases.yml` - Main workflow that builds and runs the tool
- Configured via environment variables from GitHub secrets
- Supports backing up multiple databases in one run (numbered suffixes: DATABASE_CONNECTION_1, DATABASE_CONNECTION_2, etc.)
- Runs on schedule or manual trigger

### Core Functionality
- Connect to any PostgreSQL, MySQL, or MongoDB database (Neon, Supabase, Railway, AWS RDS, PlanetScale, MongoDB Atlas, etc.)
- Export databases using native dump tools (pg_dump, mysqldump, mongodump)
- Compress with gzip (optional, enabled by default)
- Encrypt with AES-256-GCM (optional but recommended)
- Upload to Cloudflare R2 storage
- Apply retention policies (by age or count)
- Send webhook notifications (Slack, Discord, etc.)
- Support for local testing via `scripts/run-local.sh`

## Key Points

- This is NOT a reusable GitHub Action (no action.yml)
- This is NOT a published package
- This IS a template repository for forking
- Users customize the workflow in their fork, not reference this repo from theirs