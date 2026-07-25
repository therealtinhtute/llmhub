# Plan: Token estimation

Phase: token-estimation
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-07-23

## Goal
Provide bounded semantic Claude/xAI input-token estimates and avoid emitting
or processing tool settings when no convertible tools exist.

## Inputs
- `.kit/planning/SPEC.md`
- `token-estimation-CONTEXT.md`
- CLIProxyAPI v7.2.95 commits `3ad6dfe3`, `cb110ad4`, `f3e36f19`

## Wave 1
### T1 — Implement Claude input-token state
- type: implementation
- inputs:
  - current executor translation helpers
- touches:
  - `internal/runtime/executor/helps/`
  - provider executors that translate Claude responses
- avoid:
  - auth and routing
- steps:
  1. Collect semantic Claude request segments while excluding media/control fields.
  2. Estimate with O200k and patch only the first zero/missing `message_start`.
  3. Preserve CRLF/LF SSE framing, combined events, and non-target events.
  4. Wire state into streaming executor paths.
- expected outputs:
  - deterministic Claude input-token estimation across supported providers
- verification:
  - `go test ./internal/runtime/executor/helps ./internal/runtime/executor/... -run 'ClaudeInputToken|InputTokens'`
- stop if:
  - an executor lacks enough source/response format information
- escalate to:
  - to-plan phase

## Wave 2
### T2 — Implement xAI counting and tool-setting optimization
- type: implementation
- inputs:
  - current xAI executor and OpenAI Responses translator
- touches:
  - `internal/runtime/executor/xai_executor.go`
  - `internal/runtime/executor/xai_executor_test.go`
  - `internal/translator/openai/openai/responses/`
  - `go.mod`
  - `go.sum`
- avoid:
  - model registry and auth
- steps:
  1. Upgrade tokenizer to v0.8.1.
  2. Count xAI semantic input segments with O200k.
  3. Emit tool choice and parallel-tool settings only with convertible tools.
  4. Add behavior tests and allocation benchmarks for large tool arrays.
- expected outputs:
  - compatible payloads with bounded semantic counts and reduced redundant work
- verification:
  - `go test ./internal/runtime/executor ./internal/translator/openai/openai/responses`
- stop if:
  - dependency upgrade causes unrelated count regressions
- escalate to:
  - user clarification

## Wave 3
### T3 — Gate the phase
- type: test
- inputs:
  - Waves 1–2
- touches:
  - phase run and validation evidence only
- avoid:
  - unrelated source changes
- steps:
  1. Run focused benchmarks and the full Go gate.
  2. Record exact test/build evidence.
- expected outputs:
  - clean token-estimation phase gate
- verification:
  - `go test ./... && go vet ./... && go build ./... && git diff --check`
- stop if:
  - any gate fails twice after one targeted correction
- escalate to:
  - check

## Risks / Watch-fors
- Estimation must never overwrite authoritative non-zero provider usage.
- SSE patching must preserve chunk ordering and line endings.
- O200k dependency changes may expose unrelated count assumptions.
