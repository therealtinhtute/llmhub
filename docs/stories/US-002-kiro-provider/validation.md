# Validation

## Proof Strategy

Use unit tests and mocked upstream HTTP servers. Do not require live Kiro
credentials for acceptance.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | Raw token import, 9router import, refresh parsing, translator cases |
| Integration | Executor headers/body, 401 refresh retry, event-stream conversion |
| E2E | Not required for v1 |
| Platform | Go package test and full Go test suite |
| Performance | Not required for v1 |
| Logs/Audit | Existing upstream request/response hooks remain in use |

## Fixtures

- 9router Kiro connection JSON with active and inactive variants.
- Mock Kiro refresh JSON.
- Mock AWS event-stream chunks represented by embedded JSON frames.

## Commands

```text
go test ./internal/auth/kiro ./sdk/auth ./sdk/cliproxy ./internal/runtime/executor
go test ./...
```

## Acceptance Evidence

2026-06-01:

- `go test ./internal/api/handlers/management ./internal/auth/kiro ./sdk/auth ./sdk/cliproxy ./internal/runtime/executor` passed.
- `go test ./internal/auth/kiro ./sdk/auth ./sdk/cliproxy ./internal/runtime/executor && go test ./...` passed through `scripts/bin/harness-cli story verify US-002`.
- Proof is mocked-provider proof; no live Kiro account was required.
- `go test ./internal/registry ./internal/auth/kiro ./sdk/auth ./internal/runtime/executor ./sdk/cliproxy` passed after adding live Kiro model catalog loading and Kiro built-in fallback protection.
- `go test ./...` passed after the Kiro model loading fix.
- Local smoke with `/private/tmp/llmhub-kiro-smoke/config.yaml`:
  - `GET /v1/models` returned Kiro fallback models after the remote shared model catalog omitted Kiro.
  - `GET /v0/management/auth-files/models?name=kiro-account16.json` returned all five Kiro auth-file models.
  - `POST /v1/chat/completions` with model `auto` routed to Kiro and returned upstream HTTP 403 because the imported Kiro account is suspended.

2026-06-02:

- Compared llmhub Kiro runtime against `decolua/9router` Kiro executor,
  request translator, model constants, token refresh, and model catalog source.
- Fixed runtime request mismatch: Kiro model IDs are now sent upstream unchanged
  instead of being uppercased and rewritten with underscores.
- Fixed response conversion mismatch: AWS EventStream binary frames are now
  parsed for assistant content, reasoning content, tool calls, metrics, and the
  previous JSON-text parser remains as a test fallback.
- Fixed Kiro refresh/import metadata preservation: `profileArn` from refresh
  responses is stored as `profile_arn` and reused by inference/model requests.
- `go test ./internal/auth/kiro ./sdk/auth ./sdk/cliproxy ./internal/runtime/executor ./internal/registry` passed.
- `go test ./...` could not run directly because local `data/postgres` is not
  readable. Equivalent tracked Go package sweep passed via:
  `go test $(rg --files -g '*.go' | sed '/^data\//d' | xargs -n1 dirname | sort -u | sed 's#^#./#')`.

2026-06-02 runtime quota UI slice:

- Expected Go proof:
  `go test ./internal/api/handlers/management ./internal/auth/kiro ./sdk/auth ./internal/runtime/executor`.
- Expected web proof: `bun run type-check` from `web/`.
- Acceptance focuses on management auth-file responses exposing `quota` and
  `model_states`, and the quota UI showing Kiro runtime state while clearly
  marking provider quota unavailable.
