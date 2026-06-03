# Overview

## Current Behavior

Management OAuth flows wrote callback code/state payloads as short-lived marker
files under `auth-dir`, then provider goroutines polled those files before
exchanging tokens and saving auth records.

In Postgres durable runtime mode, `auth-dir` is a synthetic compatibility label
and is not created as a durable runtime directory. Codex OAuth callbacks could
therefore fail or time out before the token exchange reached the Postgres token
store.

## Target Behavior

Management OAuth callbacks are handed from the callback endpoint to the waiting
provider goroutine through the existing in-memory OAuth session store. The
provider goroutine then exchanges the code and persists the resulting auth
record through the configured token store, including Postgres.

## Affected Users

- Operators using Codex OAuth login from the management UI or TUI in Postgres
  durable runtime mode.
- Operators using the same management OAuth callback flow for Claude, Gemini,
  Antigravity, or xAI.

## Affected Product Docs

- `docs/decisions/0007-postgres-runtime-storage.md`
- `docs/stories/high-risk/US-004-postgres-durable-runtime/*`

## Non-Goals

- Changing provider OAuth token exchange behavior.
- Persisting temporary callback codes in Postgres.
- Changing CLI login flows that run their own localhost OAuth callback server.
