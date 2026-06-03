# Overview

## Current Behavior

Provider credential creation uses several surfaces:

- CLI OAuth logins for Gemini, Codex, Codex device flow, Claude, Antigravity,
  Kimi, and xAI.
- CLI Vertex service-account import.
- Management OAuth login and Vertex import.
- Management raw auth JSON upload/import.
- Management API-key provider config for Gemini, Claude, Codex, Vertex-compatible,
  OpenAI-compatible, and Amp upstream mappings.

Most paths already routed through the configured token store or config store in
Postgres mode. However:

- Shared auth manager `Register` and `Update` ignored store persistence errors,
  so management imports or edits could appear successful in memory even when
  the database write failed.
- Management OAuth and Vertex import saved token records through the store but
  did not immediately register the saved record in the runtime auth manager, so
  local `make dev-pg` flows could complete while the management list/runtime
  did not see the new credential until a later reload.

## Target Behavior

Every provider credential path either persists through the configured Postgres
auth/config store or reports the persistence failure to the caller. Token-store
save paths also update the runtime auth manager after a successful save without
writing back to the store a second time.

## Affected Users

- Operators adding OAuth credentials in Postgres durable runtime mode.
- Operators importing raw auth files or Vertex service-account credentials.
- Operators editing provider API-key configuration from the management API.

## Affected Product Docs

- `docs/decisions/0007-postgres-runtime-storage.md`
- `docs/stories/high-risk/US-004-postgres-durable-runtime/*`
- `docs/stories/high-risk/US-005-oauth-callback-postgres-runtime/*`

## Non-Goals

- Reworking standalone local model-fetch utilities.
- Running real external OAuth browser flows.
- Changing provider token exchange or API-key routing semantics.
