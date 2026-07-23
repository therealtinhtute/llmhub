# Context: WebSocket message-too-big

Phase: websocket-message-too-big
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: unit, integration

## Goal

Propagate Codex upstream WebSocket close code 1009 as a structured request-scoped HTTP-413-equivalent error without credential fallback.

## Scope Boundary

### Allowed Surfaces

- `internal/runtime/executor/codex_websockets_executor.go`
- `internal/runtime/executor/codex_websockets_executor_test.go`
- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- `sdk/api/handlers/openai/openai_responses_websocket.go`
- `sdk/api/handlers/openai/openai_responses_websocket_test.go`

### Forbidden Surfaces

- xAI WebSocket feature expansion.
- Auth refresh, quota, or CountTokens behavior.
- Translator, registry, config, storage, plugin, logging, web UI, installer, release, or database schema changes.

## Spec Hooks

- Requirements 1-2.
- No credential mark, cooldown, refresh, or fallback for request-size failures.
- No migration or external dependency.

## Locked Decisions

- Map Gorilla close code 1009 to HTTP 413 with code `message_too_big`.
- Represent the error as request-scoped so the shared auth manager stops fallback.
- Preserve existing behavior for all non-1009 WebSocket errors.

## Assumptions

- The current auth manager request-scoped error contract can express this behavior without a new public API.
- OpenAI Responses WebSocket handler can forward the structured error without schema changes.

## Canonical Refs

- `.kit/planning/SPEC.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/runtime/executor/codex_websockets_executor.go:932-962`
- `internal/runtime/executor/codex_websockets_executor.go:333-341`
- `sdk/cliproxy/auth/conductor.go:2260-2291`

## Rejected Options

- String-match raw Gorilla errors in the handler: rejected because retry/fallback decisions belong in executor/auth boundaries.
- Disable fallback for all WebSocket close errors: rejected because only request-size failures are request-scoped.

## Deferred Ideas

- xAI WebSocket connection tracking and 1009 parity.
- Active connection scoping improvements.

## Escalate If

- Request-scoped classification requires changing a public SDK interface used outside the allowed surfaces.
- The handler cannot preserve structured status/code without a public response-contract change.
