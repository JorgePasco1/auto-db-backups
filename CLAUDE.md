# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Auto DB Backups is a GitHub Action that automatically backs up databases (from providers like Neon) to Cloudflare R2 storage.

## Architecture

This is a GitHub Action project. The action should:
- Connect to database providers (e.g., Neon)
- Create database backups
- Upload backups to Cloudflare R2 storage