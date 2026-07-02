# Validation

## Proof Strategy

Each slice must pass focused Go tests before moving to the next slice. The final
gate must prove backend correctness, web build compatibility, and whitespace
cleanliness. Live provider checks are optional unless a changed behavior cannot
be proven with deterministic tests.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | Model registry, media request builders, translators, auth/cooldown reset logic. |
| Integration | Management quota reset, Postgres store persistence paths, API route registration. |
| E2E | Browser/API smoke only for changed management or video endpoints. |
| Platform | No release or installer changes expected. |
| Performance | No benchmark required unless auth/model alias rebuild performance changes. |
| Logs/Audit | New video aliases should be classified consistently as AI API paths. |

## Fixtures

- Mock OpenAI/XAI/Antigravity responses from upstream tests.
- In-memory/mock auth manager tests for quota reset.
- Postgres integration tests only when `LLMHUB_POSTGRES_TEST_DSN` is available.

## Commands

```text
go test ./sdk/api/handlers/openai ./sdk/cliproxy/auth ./internal/translator/...
go test ./internal/runtime/executor ./internal/translator/antigravity/... ./internal/auth/...
go test ./sdk/api/handlers/openai ./internal/registry ./internal/api ./internal/logging
go test ./internal/api/handlers/management ./sdk/cliproxy/auth ./internal/store ./internal/redisqueue
go test ./...
cd web && bun run type-check && bun run build
git diff --check
```

## Acceptance Evidence

Completed:

```text
go test ./sdk/api/handlers/openai ./internal/registry ./internal/api ./internal/logging
ok  	github.com/therealtinhtute/llmhub/sdk/api/handlers/openai	0.447s
ok  	github.com/therealtinhtute/llmhub/internal/registry
ok  	github.com/therealtinhtute/llmhub/internal/api
ok  	github.com/therealtinhtute/llmhub/internal/logging

go test ./internal/api/handlers/management ./sdk/cliproxy/auth ./internal/store ./internal/redisqueue
ok  	github.com/therealtinhtute/llmhub/internal/api/handlers/management	0.751s
ok  	github.com/therealtinhtute/llmhub/sdk/cliproxy/auth	0.708s
ok  	github.com/therealtinhtute/llmhub/internal/store	1.548s
ok  	github.com/therealtinhtute/llmhub/internal/redisqueue	1.033s

go test ./sdk/api/handlers/openai ./sdk/cliproxy/auth ./internal/translator/...
ok  	github.com/therealtinhtute/llmhub/sdk/api/handlers/openai	0.301s
ok  	github.com/therealtinhtute/llmhub/sdk/cliproxy/auth
ok  	github.com/therealtinhtute/llmhub/internal/translator/...

go test ./internal/runtime/executor ./internal/translator/antigravity/... ./internal/auth/...
ok  	github.com/therealtinhtute/llmhub/internal/runtime/executor	5.170s
ok  	github.com/therealtinhtute/llmhub/internal/translator/antigravity/...
ok  	github.com/therealtinhtute/llmhub/internal/auth/...
```

```text
go test ./...
ok  	github.com/therealtinhtute/llmhub/... 

cd web && bun run type-check && bun run build
$ tsc --noEmit
$ tsc && vite build
✓ built in 331ms

git diff --check
ok
```
