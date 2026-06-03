# Validation

## Proof Strategy

Use focused management handler tests to prove callback submission works when
`auth-dir` is synthetic or missing, then compile the broader API/server packages
that own the redirect routes.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | OAuth session store accepts and returns callback payloads by provider/state. |
| Integration | Management callback endpoint succeeds without a real auth directory and exposes the payload to the waiting flow. |
| E2E | Not run; real provider OAuth requires external browser/provider interaction. |
| Platform | Postgres mode remains pathless for auth-dir callback handoff. |
| Performance | Not applicable; payloads are one per pending OAuth session. |
| Logs/Audit | Existing OAuth error status handling is preserved. |

## Fixtures

- Synthetic Codex OAuth state: `codex-test-state`.
- Synthetic callback code: `callback-code`.
- Missing synthetic auth directory under `t.TempDir()`.

## Commands

```text
env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/api/handlers/management
env GOCACHE=/private/tmp/llmhub-gocache go test ./internal/api
env GOCACHE=/private/tmp/llmhub-gocache go test ./cmd/server
```

## Acceptance Evidence

All listed commands passed on 2026-06-03.
