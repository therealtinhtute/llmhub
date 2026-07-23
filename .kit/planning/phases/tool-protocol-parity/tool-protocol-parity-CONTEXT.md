# Context: Codex and xAI tool protocol parity

Phase: tool-protocol-parity
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: unit, integration

## Goal

Round-trip custom, additional, and namespaced tools through Codex/xAI while filtering provider-internal xAI search traces.

## Scope Boundary

### Allowed Surfaces

- `internal/translator/codex/claude/*`
- `internal/translator/codex/openai/responses/*`
- `internal/translator/openai/openai/responses/*`
- `internal/runtime/executor/xai_executor.go`
- `internal/runtime/executor/xai_executor_test.go`
- `internal/runtime/executor/codex_executor.go`
- `internal/runtime/executor/codex_executor_parallel_tool_calls_test.go`
- `internal/runtime/executor/codex_websockets_executor.go`
- `internal/runtime/executor/codex_websockets_executor_test.go`

### Forbidden Surfaces

- Google Interactions.
- xAI API-key configuration or management.
- Encrypted reasoning replay cache.
- Kiro tool formatting.
- Auth, storage, registry, model metadata, logging, plugin, UI, installer, release, and schema changes.

## Spec Hooks

- Requirements 10-12.
- Existing `mcp__` names must remain stable.
- Provider-internal traces are hidden unless client-declared.

## Locked Decisions

- Build a request-scoped declaration table before normalization; use `<namespace>__<tool>` as the effective name and leave existing `mcp__` names byte-for-byte unchanged.
- Reject distinct original identities that collide on one effective outbound name before network I/O, using the existing structured invalid-request error path.
- Restore namespace/name/type from the exact request declaration table, never from delimiter parsing or response-name inference alone.
- Promote xAI `additional_tools` before upstream execution.
- Match xAI `x_search` against the original full client identity and suppress the complete lifecycle only for undeclared provider-internal traces.
- Codex HTTP and WebSocket routes use the same declaration contract: reject collisions before HTTP send or WebSocket dial, and restore exact namespace/name/type across streaming and non-streaming responses.
- Restore the complete custom-tool streaming lifecycle, including event types and consistent `ctc_*` item references; partial output-item-only conversion is invalid.
- The WebSocket executor file expansion is limited to the two listed executor/test files and is authorized by the reproduced P4-S1 r01 review defect; no other scope expands.

## Assumptions

- Request payload is available to response translators/executors for namespace restoration.
- Existing schema simplification and built-in web search handling can remain in place.

## Canonical Refs

- `.kit/planning/SPEC.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/translator/openai/openai/responses/openai_openai-responses_tools.go:17-83`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/translator/openai/openai/responses/openai_openai-responses_tools.go:236-328`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/runtime/executor/xai_executor.go:1472-1839`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/runtime/executor/xai_executor.go:1939-2098`
- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S1-codex-tool-declarations/r01/review.json`
- `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r03/review.json`

## Rejected Options

- Flatten namespaces without restoration: rejected because returned calls cannot be routed safely.
- Filter every xAI search-like call: rejected because clients may explicitly declare matching tools.
- Import encrypted reasoning replay in the same phase: rejected because it is independent functional expansion.

## Deferred Ideas

- xAI API-key config/management.
- Encrypted reasoning replay and agent-scoped cache isolation.
- Additional image-generation function-tool checks beyond required tool normalization.

## Escalate If

- Namespace round trips require changing public tool schemas beyond existing protocol fields.
- Client declaration cannot be recovered at response time within current executor/translator interfaces.
- Correct WebSocket parity requires product edits outside `codex_websockets_executor.go`, its test, or the existing P4-S1 allowlist.
