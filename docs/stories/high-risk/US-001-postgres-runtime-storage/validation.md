# Validation

## Proof Strategy

Prove local-mode regressions through existing unit tests and prove Postgres
contracts with guarded integration tests that run when
`LLMHUB_POSTGRES_TEST_DSN` is available.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | Config/auth/usage store compile checks, management usage/auth handlers, redisqueue behavior |
| Integration | Guarded Postgres schema idempotency, config versioning, auth save/list/delete, usage append/pop |
| E2E | Existing API/server tests in `go test ./...` |
| Platform | Supabase-style DSN documented for operator verification |
| Performance | Usage retention remains bounded at existing max 3600 seconds |
| Logs/Audit | Startup logs identify Postgres activation and first-boot auth import count |

## Fixtures

- Postgres integration tests create an isolated schema named
  `llmhub_test_<timestamp>`.
- Tests skip unless `LLMHUB_POSTGRES_TEST_DSN` is set.

## Commands

```text
go test ./internal/store ./internal/redisqueue ./internal/api/handlers/management ./sdk/cliproxy/...
go test ./cmd/server
go test ./...
LLMHUB_POSTGRES_TEST_DSN=... go test ./internal/store
```

## Acceptance Evidence

- `go test ./internal/store ./internal/redisqueue ./internal/api/handlers/management ./sdk/cliproxy/...` passed on 2026-06-01.
- `go test ./cmd/server` passed on 2026-06-01.
- `go test ./...` passed on 2026-06-01.
- Guarded Postgres integration coverage was added in
  `internal/store/postgresstore_integration_test.go`; it skips unless
  `LLMHUB_POSTGRES_TEST_DSN` is configured.
