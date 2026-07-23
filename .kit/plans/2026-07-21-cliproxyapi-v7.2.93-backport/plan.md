# CLIProxyAPI v7.2.93 targeted backport plan

## Goal

Bring the highest-value correctness and compatibility changes from `router-for-me/CLIProxyAPI` `v7.2.49...v7.2.93` into llmhub without merging upstream wholesale or weakening local Postgres, Amp, Kiro, embedded UI, and installer behavior.

Comparison evidence: `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`.

## Success criteria

- Request-size WebSocket failures are returned as structured request errors and never trigger credential fallback.
- Generic unsupported `count_tokens` endpoint errors do not disable otherwise valid models.
- Repeated concurrent quota errors do not escalate backoff more than once per cooldown window, and cooldown waits include bounded jitter without exceeding `max-retry-interval`.
- HTTP 400 OAuth `invalid_grant` failures enter the intended suspension path.
- Structured tool outputs, file data, MIME types, output indices, custom tools, additional tools, namespaces, and xAI internal search traces round-trip correctly through selected protocol paths.
- Grok 4.5 and Gemini production IDs are visible while existing preview IDs remain compatible.
- Configured model display names appear consistently in OpenAI, Claude, Gemini, and Codex client model listings.
- Amp, Kiro, Postgres runtime storage, and current management behavior remain intact.
- Focused tests, `go test ./...`, `make build-web`, and `make build` pass.

## Scope

### In scope

1. WebSocket 1009 request-scoped propagation.
2. Shared auth/count-token reliability fixes that do not require schema migration.
3. Translator content fidelity fixes.
4. Codex/xAI custom, additional, namespace, and internal-search tool correctness.
5. Safe model/catalog additions: Grok 4.5, Gemini production IDs alongside existing previews, and complete configured display-name propagation.
6. Tests and parity evidence updates.

### Not in scope

- Google Interactions protocol, routes, executors, translators, or configuration.
- Upstream `pluginhost/pluginstore`, Redis/Home plugin synchronization, or plugin management UI.
- Runtime remote refresh of the Codex client model catalog.
- GPT-5.6 Sol/Terra/Luna until model-header overrides and provider search routing are planned together.
- Kimi K3 until native thinking, `[1m]` model normalization, Claude-path normalization, and token counting are planned together.
- Shared request-time 401 refresh until Kiro's executor-local refresh behavior has an explicit double-refresh prevention contract.
- OAuth session cancellation until the Postgres-safe pre-persistence guard is designed as a dedicated auth-management slice.
- Native xAI API-key configuration and management endpoints.
- Service-tier persistence, deferred request-body logging, CPA trace IDs, encrypted xAI reasoning replay, or unproven performance micro-optimizations.
- Release, Docker, FreeBSD, sponsor, branding, logo, and README churn.

## Architecture boundary

This work spans more than eight files and five components. Changes must remain directional and must not create a dependency cycle:

```text
Client request
    |
    v
HTTP / WebSocket handler
    |
    v
Translator registry -----> protocol translators
    |                              |
    v                              v
Auth manager -----------> provider executor -----> provider
    |                              |
    v                              v
Postgres cooldown/auth state   usage/error result
```

Rules:

- Handlers do not implement provider-specific retry state.
- Translator helpers remain pure and do not access config, storage, or network state.
- Executors classify provider/protocol errors; the auth manager decides retry, cooldown, and fallback.
- Existing Postgres persistence hooks in the auth manager remain authoritative.
- Kiro-specific refresh and usage updates remain in place.

## Work phases

Each phase is independently mergeable and leaves the application usable if later phases do not ship.

### Phase 1 — WebSocket message-too-big correctness

Change:

- Add a request-scoped mapped error for Gorilla WebSocket close code `1009` in `internal/runtime/executor/codex_websockets_executor.go`.
- Map it to HTTP `413` with structured code `message_too_big`.
- Ensure `sdk/cliproxy/auth/conductor.go` treats the mapped error as request-scoped and does not mark, cool down, or fall back credentials.
- Preserve the structured error through `sdk/api/handlers/openai/openai_responses_websocket.go`.

Primary files:

- `internal/runtime/executor/codex_websockets_executor.go`
- `internal/runtime/executor/codex_websockets_executor_test.go`
- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- `sdk/api/handlers/openai/openai_responses_websocket.go`
- `sdk/api/handlers/openai/openai_responses_websocket_test.go`

Verification:

```text
go test ./internal/runtime/executor -run 'Test.*(MessageTooBig|1009)'
go test ./sdk/cliproxy/auth -run 'Test.*(RequestScoped|Fallback)'
go test ./sdk/api/handlers/openai -run 'Test.*WebSocket.*(1009|MessageTooBig)'
```

Acceptance:

- A provider close frame `1009` reaches the client as a 413-style structured error.
- No second credential or provider is attempted.
- Other WebSocket errors retain existing fallback behavior.

### Phase 2 — Auth cooldown and CountTokens reliability

Change:

- In `sdk/cliproxy/auth/conductor.go`, escalate quota backoff only when the current recovery window has expired.
- Add jitter bounded to `min(wait/4, 2s)` and cap total wait at `max-retry-interval`.
- Recognize OAuth `invalid_grant` from structured errors and HTTP 400 text; suspend for the upstream-compatible 30-minute window without treating unrelated 400 errors as auth failures.
- Make generic unsupported `count_tokens` endpoint errors availability-neutral.
- Continue normal suspension for explicit nested or structured `model_not_found` results.
- Preserve local cooldown snapshots, Kiro usage updates, and Postgres persistence hooks.

Primary files:

- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/cooldown_backoff_test.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- `sdk/cliproxy/auth/conductor_count_tokens_test.go`
- `sdk/cliproxy/auth/conductor_scheduler_refresh_test.go`
- `internal/api/modules/amp/routes_test.go`

Verification:

```text
go test ./sdk/cliproxy/auth -run 'Test.*(Cooldown|Jitter|InvalidGrant|CountTokens)'
go test ./internal/api/modules/amp -run 'Test.*CountTokens'
```

Acceptance:

- Concurrent 429 results inside one cooldown window increment the level once.
- Jitter never exceeds two seconds or the configured maximum retry interval.
- HTTP 400 `invalid_grant` suspends; ordinary HTTP 400 does not.
- Generic endpoint 404 from CountTokens does not hide or suspend the model.
- Explicit `model_not_found` still changes model availability.

### Phase 3 — Translator content fidelity

Change:

- Add `internal/translator/common/file_data.go` with one OpenAI file-data parser that supports raw base64 and `data:<mime>;base64,<payload>` forms, using filename extension only as fallback.
- Apply the helper to Gemini, Gemini CLI, Codex, Antigravity, and relevant OpenAI/Interactions-independent paths that currently forward `file_data` unchanged.
- Preserve structured `function_call_output.output` arrays and objects when translating OpenAI Responses to Claude instead of flattening through `.String()`.
- Consume provider `output_index` when present and retain current sequential fallback when absent.
- Preserve valid empty Claude `content: []` behavior unchanged.

Primary files:

- `internal/translator/common/file_data.go`
- `internal/translator/common/file_data_test.go`
- `internal/translator/claude/openai/responses/claude_openai-responses_request.go`
- `internal/translator/claude/openai/responses/claude_openai-responses_request_test.go`
- `internal/translator/gemini/openai/chat-completions/gemini_openai_request.go`
- `internal/translator/gemini/openai/chat-completions/gemini_openai_request_test.go`
- `internal/translator/gemini-cli/openai/chat-completions/gemini-cli_openai_request.go`
- `internal/translator/codex/openai/chat-completions/codex_openai_request.go`
- `internal/translator/codex/openai/chat-completions/codex_openai_request_test.go`
- `internal/translator/codex/claude/codex_claude_response.go`
- `internal/translator/codex/claude/codex_claude_response_test.go`

Verification:

```text
go test ./internal/translator/common
go test ./internal/translator/claude/openai/responses
go test ./internal/translator/gemini/openai/chat-completions
go test ./internal/translator/gemini-cli/openai/chat-completions
go test ./internal/translator/codex/openai/chat-completions
go test ./internal/translator/codex/claude
```

Acceptance:

- Raw base64 and data URLs produce identical provider payloads and correct MIME types.
- Filename extension is used only when the data URL has no MIME type.
- Structured tool output remains structured through OpenAI Responses to Claude.
- Interleaved output events keep stable provider indices.

### Phase 4 — Codex/xAI tool protocol parity

Change:

- Add custom tool, `additional_tools`, and namespace handling to Codex request/response translations.
- Qualify namespaced tools deterministically as `<namespace>__<tool>` internally and restore the protocol namespace fields on output.
- Preserve existing `mcp__` names without double qualification.
- Promote xAI `additional_tools` input items into top-level `tools` before upstream execution.
- Normalize xAI namespaced tool choices and restore namespaces in returned calls.
- Filter provider-internal `x_search` call traces unless the client declared the corresponding tool.
- Keep existing tool-schema simplification and local Kiro/Amp behavior unchanged.

Primary files:

- `internal/translator/codex/claude/codex_claude_request.go`
- `internal/translator/codex/claude/codex_claude_request_test.go`
- `internal/translator/codex/openai/responses/codex_openai-responses_request.go`
- `internal/translator/codex/openai/responses/codex_openai-responses_request_test.go`
- `internal/translator/openai/openai/responses/openai_openai-responses_tools.go`
- `internal/translator/openai/openai/responses/openai_openai-responses_tools_test.go`
- `internal/translator/openai/openai/responses/openai_openai-responses_response.go`
- `internal/translator/openai/openai/responses/openai_openai-responses_response_test.go`
- `internal/runtime/executor/xai_executor.go`
- `internal/runtime/executor/xai_executor_test.go`
- `internal/runtime/executor/codex_executor.go`
- `internal/runtime/executor/codex_executor_parallel_tool_calls_test.go`

Verification:

```text
go test ./internal/translator/codex/claude
go test ./internal/translator/codex/openai/responses
go test ./internal/translator/openai/openai/responses
go test ./internal/runtime/executor -run 'Test.*(XAI|Namespace|AdditionalTools|CustomTool|ParallelTool)'
```

Acceptance:

- Namespace collisions cannot route to the wrong tool.
- Custom and additional tools round-trip through streaming and non-streaming paths.
- Existing `mcp__` tool names are stable.
- Internal xAI search traces are not exposed as client-executable calls unless declared by the client.

### Phase 5 — Safe model and display-name compatibility

Change:

- Add Grok 4.5 metadata using upstream context, token, and thinking limits.
- Add Gemini production model IDs alongside existing preview IDs; do not remove preview aliases in this phase.
- Add `display-name` fields to supported configured model and OAuth alias types.
- Propagate configured display names through registry cloning and OpenAI, Claude, Gemini, Gemini CLI, and Codex client model listings.
- Do not add GPT-5.6 or Kimi K3 metadata in this phase.

Primary files:

- `internal/registry/models/models.json`
- `internal/registry/model_definitions_test.go`
- `internal/registry/model_registry.go`
- `internal/config/config.go`
- `internal/config/model_display_name_test.go`
- `sdk/cliproxy/service.go`
- `sdk/cliproxy/config_model_display_name_test.go`
- `sdk/api/handlers/openai/openai_handlers.go`
- `sdk/api/handlers/openai/codex_client_models.go`
- `sdk/api/handlers/claude/code_handlers.go`
- `sdk/api/handlers/gemini/gemini_handlers.go`
- `sdk/api/handlers/gemini/gemini-cli_handlers.go`

Verification:

```text
go test ./internal/registry ./internal/config
go test ./sdk/cliproxy
go test ./sdk/api/handlers/openai ./sdk/api/handlers/claude ./sdk/api/handlers/gemini
```

Acceptance:

- Grok 4.5 is returned by the xAI model registry.
- Gemini production and preview IDs both remain discoverable.
- A configured display name appears consistently across all model-list APIs.
- Model IDs and routing keys remain unchanged by display-name customization.

## Final verification

After every phase passes its focused tests:

```text
go test ./...
make build-web
make build
git diff --check
```

Manual acceptance using mocked or operator-owned credentials:

1. Send a WebSocket request that causes a synthetic close 1009 and confirm no fallback attempt appears in logs.
2. Return generic CountTokens 404 and explicit model-not-found 404 from a test upstream; confirm only the explicit model error affects availability.
3. Exercise structured tool output, data-URL file content, namespaced tools, and xAI internal search fixtures through streaming and non-streaming paths.
4. Query OpenAI, Claude, Gemini, and Codex model endpoints and compare display names and IDs.

No new API key, external account, or third-party service is required for deterministic verification. Live smoke tests may use existing operator-owned provider credentials but are not required to prove the core transformations.

## Risks and mitigations

- **Auth regression:** Shared conductor changes can affect all providers. Keep changes local to error classification/cooldown helpers and run the full auth suite after each edit.
- **Kiro double behavior:** Do not implement shared 401 refresh in this plan; preserve Kiro executor-local refresh and usage updates.
- **Tool name collision:** Use request-derived namespace maps and round-trip fixtures; do not infer namespaces from response names alone.
- **Model advertisement mismatch:** Do not add GPT-5.6 or Kimi K3 until their coupled executor behavior is implemented.
- **External dependency failure:** Do not add periodic remote model-catalog fetch. Runtime remains deterministic and offline-capable.
- **Scale:** Translator performance commits are excluded until benchmarks show a local bottleneck. Correctness changes must avoid unbounded payload copies.

## Premise collapse

This plan assumes the local translator/executor interfaces remain close enough to adapt the selected upstream tests and algorithms without importing upstream service, plugin, storage, or logging architecture. If that assumption fails for a slice, stop that slice and implement the same externally observable behavior behind the current local interface rather than widening the import boundary.

## Rollback

- No database migration or destructive data change is planned.
- Each phase can be reverted independently.
- Model additions are additive; preview model IDs are retained.
- If a phase causes provider regressions, revert only that phase and keep the earlier verified phases.

## Documentation and handoff

During implementation:

- Create a new high-risk story packet using the next unambiguous story identifier rather than reusing `US-015`, because the repository currently has conflicting `US-015` meanings in markdown and durable Harness data.
- Update `.kit/reports/github/cliproxyapi-v7.2.93-parity.md` with implemented/deferred status and exact test evidence.
- Update the new story validation file after each mergeable phase.
- Record any provider behavior decision that changes API shape, routing, auth state, or data ownership in `docs/decisions/`.
