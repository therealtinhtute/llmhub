# 0007 Postgres Durable Runtime Ownership

Date: 2026-06-01

## Status

Accepted

## Context

Cliproxy local-file mode is the default runtime storage model. Existing
Postgres support moved config/auth/usage into DB, but server runtime still left
durable local logging surfaces and synthetic path assumptions around the running
process. Operators need a remote Postgres or Supabase-backed deployment where
Postgres is the only durable runtime owner once bootstrap finishes.

## Decision

When `PGSTORE_DSN` is set, llmhub uses Postgres as the only durable runtime
store for config, auth, and recent usage queue records. The service seeds DB
config and auth rows from local files only on first boot when the corresponding
DB tables are empty. After seeding, DB rows win. Startup fails clearly when the
DB connection fails, and the server does not fall back to local durable
runtime stores.

In this mode, llmhub disables durable local server logging and request archive
logging instead of introducing new Postgres log tables. Stdout/stderr remains
the operational log surface. Synthetic Postgres config/auth paths remain labels
for compatibility only; they are not authoritative durable mirrors.

Local file mode remains unchanged when `PGSTORE_DSN` is unset. `HOME_JWT` mode
is a separate runtime path and is not changed by this decision.

## Alternatives Considered

1. Keep durable local log files in Postgres mode. Rejected because that leaves
   a second durable runtime surface outside the DB contract.
2. Add Postgres log tables now. Rejected because this story only changes
   durable ownership for existing runtime state; log schema expansion is a
   separate decision.
3. Add a `config.yaml` storage block. Rejected to keep activation environment
   driven and avoid changing local config schema.

## Consequences

Positive:

- Postgres deployments no longer depend on file watcher state for config/auth.
- Management config/auth writes persist directly to DB.
- Recent usage survives process restarts within the configured short retention.
- Postgres mode no longer creates durable local auth directories, app log files,
  or request archive files during normal server runtime.

Tradeoffs:

- Postgres mode requires DB connectivity at startup.
- Integration proof requires an external Postgres DSN.
- Auth JSON payloads in Postgres are sensitive and must be protected like local
  auth files.
- Operators who want durable log retention in Postgres mode must rely on
  platform stdout/stderr capture for now.

## Follow-Up

- Add operator runbook examples for backup/restore if Postgres mode becomes the
  recommended production path.
- Revisit DB-backed operational logging only if product behavior requires
  durable in-app log search rather than platform log capture.
