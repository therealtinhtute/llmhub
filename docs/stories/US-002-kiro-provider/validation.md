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

2026-06-08 provider quota tracking and runtime usage stats slice:

- Added mocked `getUsageLimits` proof for paid quota, trial quota, reset date,
  overage fields, empty/unavailable quota, endpoint fallback order, and 401
  refresh retry.
- Added mocked management proof that `POST /v0/management/auth-files/kiro/quota`
  updates auth runtime metadata and quota state.
- Added management proof that Kiro quota refresh 402/403/429 metadata failures
  do not mark the auth as permanently errored or disabled.
- Added selector proof that an exhausted persisted Kiro provider quota is
  skipped so routing can fall back to another account.
- Added executor proof for Kiro `metricsEvent` token usage and
  `contextUsageEvent` plus `meteringEvent` estimate fallback.
- Added auth manager proof that successful Kiro calls accumulate
  `kiro_usage_stats` on auth metadata.
- Updated web quota UI proof to render provider quota rows, trial/overage,
  runtime usage stats, runtime cooldown, and model states.
- Focused verification passed:
  - `go test ./internal/runtime/executor ./internal/api/handlers/management ./sdk/cliproxy/auth`
  - `cd web && bun run type-check`
- Follow-up review fixed fragmented Kiro tool-call argument coalescing so raw
  fragments preserve whitespace inside JSON string values.
- Final verification passed:
  - `go test ./internal/runtime/executor -run 'TestKiroExecutorExecute_CollapsesFragmentedToolUseEvents|TestKiroExecutorExecute_ParsesAWSEventStream|TestKiroExecutor'`
  - `go test ./...`
  - `cd web && bun run type-check`

2026-06-09 Kiro quota parity follow-up:

- Fixed Kiro provider quota normalization so enterprise `getUsageLimits`
  responses keep `displayName`, per-row `nextDateReset`, and
  `overageConfiguration.overageStatus`; per-row overage amount, cap, and rate
  now also feed the provider-level UI summary.
- Added compatibility for legacy/9router-style `quotas` object maps so saved
  Kiro quota snapshots no longer drop 2-3 quota rows when the data is not an
  array yet.
- Fixed the web refresh path to normalize snake_case Kiro quota responses
  before deriving `providerQuotaAvailable`.
- Verification passed:
  - `go test ./internal/runtime/executor ./internal/api/handlers/management ./sdk/cliproxy/auth`
  - `go test $(rg --files -g '*.go' | sed '/^data\\//d' | xargs -n1 dirname | sort -u | sed 's#^#./#')`
  - `cd web && bun run type-check`

2026-06-09 Kiro quota field-preservation follow-up:

- Extended normalized Kiro quota snapshots to keep raw `/getUsageLimits`
  payloads plus row metadata for `currency`, `unit`, `displayNamePlural`,
  `overageCharges`, `overageChargesWithPrecision`, and `bonuses`.
- Extended subscription metadata to keep `overageCapability`,
  `subscriptionManagementTarget`, and `upgradeCapability`; the quota UI now
  includes those compactly with plan/overage metadata and displays quota units
  or currency beside relevant amounts.
- Verification passed:
  - `go test ./internal/runtime/executor ./internal/api/handlers/management ./sdk/cliproxy/auth`
  - `go test $(rg --files -g '*.go' | sed '/^data\\//d' | xargs -n1 dirname | sort -u | sed 's#^#./#')`
  - `cd web && bun run type-check`

2026-06-04 investigation note:

- Reviewed `jwadow/kiro-gateway` issue `#41` and cached the upstream evidence in
  `.kit/cache/github/jwadow/kiro-gateway/`.
- Confirmed upstream Kiro rejects tool names longer than 64 characters and that
  `kiro-gateway` now fails early with explicit validation instead of letting
  Kiro return a generic malformed-request error.
- Confirmed llmhub currently forwards tool names unchanged in
  `internal/runtime/executor/kiro_executor.go` for both tool definitions and
  assistant tool calls, so long MCP/Codex tool names are the most likely cause
  of current "cannot call tool" failures.
- The older "tool_result lost during merge" failure mode did not match the
  llmhub Kiro translator shape during this review because tool results are kept
  as separate user turns rather than merged user content.
- Follow-up implementation should add a Kiro-specific validation error for tool
  names longer than 64 characters and cover it with focused executor tests.

2026-06-04 implementation and live verification:

- Added a Kiro-specific request-build guard in
  `internal/runtime/executor/kiro_executor.go` that rejects tool definitions
  and assistant tool-call history when any tool name exceeds 64 characters.
- Added focused executor tests covering:
  - oversized Kiro tool definitions
  - oversized assistant tool calls in prior conversation history
- Verification passed:
  - `go test ./internal/runtime/executor -run 'TestBuildKiroPayloadFromOpenAI|TestKiroExecutor'`
  - `go test ./sdk/cliproxy ./internal/registry ./internal/auth/kiro ./sdk/auth`
  - `go test ./...`
  - `go build -o /private/tmp/llmhub-kiro-smoke/llmhub ./cmd/server`
- Live Anthropic-compatible smoke against
  `http://127.0.0.1:31286/v1/messages` with model `claude-sonnet-4.5` and the
  existing auth file under `/private/tmp/llmhub-kiro-smoke/auths`:
  - 3 oversized-tool requests returned local `400 invalid_request_error` with
    explicit length details instead of a vague upstream malformed-request error.
  - 1 plain no-tool request reached Kiro and returned upstream `403` with an
    explicit account suspension message.
  - 1 short-tool request returned `503 auth_unavailable` after the same Kiro
    auth had been suspended for model use by the upstream provider.
- Conclusion: the local tool-name failure is fixed and now surfaces correctly,
  but successful end-to-end Kiro execution is currently blocked by the real
  account state, not by the llmhub Kiro translator.

2026-06-04 follow-up live verification with refreshed multi-auth Kiro pool:

- Loaded five refreshed Kiro auth files into `/private/tmp/llmhub-kiro-multi`
  and rebuilt `./cmd/server` to `/private/tmp/llmhub-kiro-multi/llmhub`.
- During the first successful real tool-use attempt, Kiro returned incremental
  `toolUseEvent` fragments and llmhub incorrectly surfaced them as many
  duplicate tool calls in non-stream OpenAI/Anthropic-compatible responses.
- Added a non-stream Kiro coalescing fix in
  `internal/runtime/executor/kiro_executor.go` so repeated `toolUseEvent`
  chunks are merged by `toolUseId`, argument fragments are concatenated, and
  placeholder `{}` payloads are only kept when the tool truly has no input.
- Added focused executor regression coverage for fragmented Kiro tool-use
  events in `internal/runtime/executor/kiro_executor_test.go`.
- Verification passed after the fix:
  - `go test ./internal/runtime/executor -run 'TestBuildKiroPayloadFromOpenAI|TestKiroExecutorExecute_ParsesAWSEventStream|TestKiroExecutorExecute_CollapsesFragmentedToolUseEvents'`
  - `go test ./...`
  - `go build -o /private/tmp/llmhub-kiro-multi/llmhub ./cmd/server`
- Final live Anthropic-compatible smoke against
  `http://127.0.0.1:31287/v1/messages`:
  - plain request to `claude-sonnet-4.5` returned `200`
  - forced short-tool request returned `200` with
    `stop_reason: "tool_use"` and a single `repo_starred` tool call whose input
    preserved `owner=therealtinhtute` and `repo=llmhub`
  - oversized tool name returned local `400 invalid_request_error` with the
    explicit 64-character limit message
  - thinking request to `claude-sonnet-4.5-thinking` returned `200`
  - `auto` request returned `200`
- Conclusion: llmhub now handles Kiro tool calls correctly on the real
  Anthropic-compatible path, while rejecting unsupported long tool names
  locally with a clear validation error.
