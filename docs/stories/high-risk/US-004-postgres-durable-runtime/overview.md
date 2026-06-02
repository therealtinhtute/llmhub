# Overview

## Current Behavior

`PGSTORE_DSN` already moves config, auth, and recent usage into Postgres, but
server runtime still leaves durable local surfaces behind:

- app logs can still rotate into local `logs/`
- request and forced error archives can still write local files
- startup still treats a local auth directory as required

That weakens the claimed source-of-truth contract for remote Postgres mode.

## Target Behavior

When `PGSTORE_DSN` is set:

- Postgres is the only durable runtime owner for config, auth, and usage queue
- local config/auth files remain bootstrap-only for first boot into an empty DB
- durable local app logs and request archives are disabled
- stdout/stderr remains the operator-visible log surface
- `HOME_JWT` mode remains unchanged and out of scope

## Affected Users

- operators running llmhub against hosted Postgres or Supabase
- management users editing config/auth in Postgres mode
- operators who previously relied on local request/error archive files

## Affected Docs

- `README.md`
- `docs/decisions/0007-postgres-runtime-storage.md`
- `docs/stories/high-risk/US-004-postgres-durable-runtime/*`
