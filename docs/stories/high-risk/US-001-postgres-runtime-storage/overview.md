# Overview

## Current Behavior

Without `PGSTORE_DSN`, llmhub reads `config.yaml`, stores OAuth/auth JSON files
under `auth-dir`, watches the local filesystem, and serves recent management
usage from an in-memory queue.

The previous Postgres path mirrored DB rows into a local spool directory and
then reused file-based config/auth behavior.

## Target Behavior

When `PGSTORE_DSN` is set, Postgres is the runtime source of truth for:

- `config_store`: YAML config content and version.
- `auth_store`: auth JSON payloads plus runtime auth state.
- `usage_events`: short-retention management usage queue payloads.

On first boot only, the service seeds config from the local `-config` path or
`config.yaml`. If the auth table is also empty, it imports existing local auth
JSON files from the configured `auth-dir`. After that, DB rows win and local
files are not watched as the source of truth.

## Affected Users

- Operators running llmhub on a VPS or hosted environment.
- Operators using Supabase/Postgres for durable runtime state.
- Management API users editing config/auth records.

## Affected Product Docs

- `README.md`
- `docs/stories/high-risk/US-001-postgres-runtime-storage/*`
- `docs/decisions/0007-postgres-runtime-storage.md`

## Non-Goals

- No logs migration.
- No model cache migration.
- No Harness SQLite migration.
- No long-term analytics dashboard.
- No `config.yaml` storage block.
