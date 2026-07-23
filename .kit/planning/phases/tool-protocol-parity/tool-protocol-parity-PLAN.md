# Plan: Codex and xAI tool protocol parity

Phase: tool-protocol-parity
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-07-22

## Goal

Round-trip custom, additional, and namespaced tools through Codex/xAI using request-scoped declaration tables while suppressing only undeclared provider-internal xAI search lifecycles.

## Inputs

- Approved Phase 3 governed-tree fingerprint.
- `.kit/planning/SPEC.md`
- phase CONTEXT and detailed fan-out plan.

## Wave 1 — Parallel isolated slices

### P4-S1 — Codex declaration table and HTTP/WebSocket round trips

- exact touches:
  - `internal/translator/codex/claude/codex_claude_request.go`
  - `internal/translator/codex/claude/codex_claude_request_test.go`
  - `internal/translator/codex/openai/responses/codex_openai-responses_request.go`
  - `internal/translator/codex/openai/responses/codex_openai-responses_request_test.go`
  - `internal/translator/openai/openai/responses/openai_openai-responses_tools.go`
  - `internal/translator/openai/openai/responses/openai_openai-responses_tools_test.go`
  - `internal/translator/openai/openai/responses/openai_openai-responses_response.go`
  - `internal/translator/openai/openai/responses/openai_openai-responses_response_test.go`
  - `internal/runtime/executor/codex_executor.go`
  - `internal/runtime/executor/codex_executor_parallel_tool_calls_test.go`
  - `internal/runtime/executor/codex_websockets_executor.go`
  - `internal/runtime/executor/codex_websockets_executor_test.go`
- steps:
  1. Build one request-scoped table for top-level and `additional_tools` declarations.
  2. Preserve original namespace/name/type and calculate effective outbound identity.
  3. Leave `mcp__` byte-stable.
  4. Reject distinct originals that collide on one effective name before HTTP send or WebSocket dial with structured 400 `tool_name_collision`.
  5. Restore exact identity for HTTP and WebSocket streaming/non-streaming outputs.
  6. Restore custom calls as one consistent lifecycle: output items, custom input delta/done event types, and `ctc_*` item references must agree.
  7. Add permanent regressions for WebSocket collision zero-network behavior, WebSocket namespace restoration, and complete custom streaming identity.
- expected outputs:
  - one declaration contract enforced by both Codex transports;
  - no delimiter-only restoration or stale function-call events after custom conversion;
  - P4-S1 r01 remains immutable rejected evidence and the corrected revision supersedes it.
- verification:
  - `go test ./internal/translator/codex/claude ./internal/translator/codex/openai/responses ./internal/translator/openai/openai/responses -count=1`
  - `go test ./internal/runtime/executor -run 'Test.*(Codex|WebSocket|Namespace|AdditionalTools|CustomTool|ParallelTool|Collision|Declaration)' -count=1`
  - `go test -race ./internal/runtime/executor -run 'Test.*(Codex|WebSocket|Namespace|AdditionalTools|CustomTool|ParallelTool|Collision|Declaration)' -count=1`
- stop if:
  - correctness requires product edits outside these twelve files;
  - declaration state cannot remain request-scoped across WebSocket reconnect/replay paths.

### P4-S2 — xAI tools and internal-search lifecycle

- execution state: immutable `r03` is independently `APPROVED`; retain it unchanged and do not reimplement this slice during the P4-S1 scope refresh.
- evidence: `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r03/`.
- exact touches:
  - `internal/runtime/executor/xai_executor.go`
  - `internal/runtime/executor/xai_executor_test.go`
- steps:
  1. Promote `additional_tools` before send.
  2. Use the same original/effective declaration contract for names and choices.
  3. Reject collisions before network I/O.
  4. Match xAI search calls against original full client identity.
  5. Suppress the complete streaming/non-stream lifecycle only for undeclared provider-internal traces and compact later visible indices consistently.
- verification:
  - `go test ./internal/runtime/executor -run 'Test.*(XAI|Namespace|AdditionalTools|CustomTool|ParallelTool|XSearch|Collision|Declaration)' -count=1`

Both slices use the same immutable Phase 3 base in separate worktrees and produce immutable patch/test/review artifacts.

## Wave 2 — Apply, gate, and close

- apply order: P4-S1, P4-S2
- closure touches:
  - `docs/decisions/0012-tool-namespace-and-x-search-lifecycle.md`
  - `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
  - `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
  - append-only harness/evidence paths
- verification:
  - focused translator suites
  - focused executor suites
  - `go test ./...`
  - `go vet ./...`
  - `go build ./...`
  - `git diff --check`
- rollback:
  - reverse P4-S2 then P4-S1; reverse closure docs patch; verify Phase 3 fingerprint.

## Risks / Watch-fors

- Never restore namespaces through delimiter parsing alone.
- A short-name match must not authorize a namespaced declaration.
- CodexAuto WebSocket must validate before dial and restore every streamed/completed output consistently with HTTP.
- Custom-tool conversion must rewrite the entire event lifecycle, not only output items.
- No xAI API-key config, replay cache, Kiro tools, or Google Interactions.
