# Validation

## Proof Strategy

Use focused unit tests to prove that Postgres durable mode suppresses local log
surfaces and local auth-dir creation, then run the existing server/store/API Go
suite to catch regressions.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | Postgres durable mode skips log directory creation; disabled request logger writes no files; management log surfaces report disabled; embedded service no longer requires local auth-dir |
| Integration | Guarded Postgres store tests continue to cover config/auth/usage behavior |
| E2E | Existing `go test ./...` coverage for management/server/runtime paths |
| Platform | Runtime docs now state stdout/stderr-only logging for Postgres mode |
| Logs/Audit | startup messaging explicitly states local file-backed logs are disabled in Postgres durable mode |

## Commands

```text
go test ./internal/logging ./internal/api/handlers/management ./sdk/cliproxy
go test ./internal/store ./internal/redisqueue ./internal/api/handlers/management ./sdk/cliproxy/...
go test ./cmd/server
go test ./...
LLMHUB_POSTGRES_TEST_DSN=... go test ./internal/store
```
