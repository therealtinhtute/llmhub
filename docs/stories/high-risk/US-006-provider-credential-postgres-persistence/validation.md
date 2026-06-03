# Validation

## Proof Strategy

Use code audit plus focused tests around shared auth-manager persistence errors,
then run focused provider/persistence packages and the full Go suite.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | `Register` and `Update` return store persistence failures; token-store save registers the auth in memory without double persistence. |
| Integration | Management auth handlers, API server, CLI login/import packages, SDK auth packages compile and pass tests. |
| E2E | Not run; real provider OAuth requires external provider/browser interaction. |
| Platform | Postgres mode uses token/config stores, not auth-dir files, for runtime credential persistence. |
| Performance | Not applicable. |
| Logs/Audit | Persistence failures now surface as API/command errors instead of silent in-memory success. |

## Commands

```text
env GOCACHE=/private/tmp/llmhub-gocache go test ./sdk/cliproxy/auth -run 'TestWithSkipPersist|TestManager(Register|Update)_ReturnsPersistenceError'
env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/cmd ./sdk/auth ./sdk/cliproxy/auth
env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/api/handlers/management ./internal/api ./cmd/server
env GOCACHE=/private/tmp/llmhub-gocache go test ./...
git diff --check
```

## Acceptance Evidence

All listed commands passed on 2026-06-03.
