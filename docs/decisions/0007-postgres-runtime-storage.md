# 0007 Postgres Runtime Storage

Date: 2026-06-01

## Status

Accepted

## Context

Cliproxy local-file mode is the default runtime storage model. Existing
Postgres support mirrored records into a local spool directory, leaving file
watching and file paths in the runtime path. Operators need a remote Postgres or
Supabase-backed deployment where DB connectivity is explicit and DB rows are the
source of truth.

## Decision

When `PGSTORE_DSN` is set, llmhub uses Postgres as the runtime store for config,
auth, and recent usage queue records. The service seeds DB config and auth rows
from local files only on first boot when the corresponding DB tables are empty.
After seeding, DB rows win. Startup fails clearly when the DB connection fails.

Local file mode remains unchanged when `PGSTORE_DSN` is unset.

## Alternatives Considered

1. Keep the local spool mirror. Rejected because the spool preserves path-based
   behavior and competing sources of truth.
2. Add a `config.yaml` storage block. Rejected to keep activation environment
   driven and avoid changing local config schema.
3. Store long-term analytics. Rejected because this migration only covers recent
   management usage queue compatibility.

## Consequences

Positive:

- Postgres deployments no longer depend on file watcher state for config/auth.
- Management config/auth writes persist directly to DB.
- Recent usage survives process restarts within the configured short retention.

Tradeoffs:

- Postgres mode requires DB connectivity at startup.
- Integration proof requires an external Postgres DSN.
- Auth JSON payloads in Postgres are sensitive and must be protected like local
  auth files.

## Follow-Up

- Add operator runbook examples for backup/restore if Postgres mode becomes the
  recommended production path.
