# Plan: Auth credential concurrency

Phase: auth-credential-concurrency
Status: ready
Wave Count: 4
Execution Owner: work
Updated At: 2026-07-23

## Goal
Implement Home-dispatched credential accounting with exactly-once lifecycle
release, bounded drain, and in-flight observation without regressing local auth.

## Inputs
- `.kit/planning/SPEC.md`
- `auth-credential-concurrency-CONTEXT.md`
- CLIProxyAPI `v7.2.96`, commit `3ecd4afe`
- current Home, auth manager, handler, executor, and service code

## Wave 1
### T1 — Add configuration and lifecycle primitives
- type: implementation
- inputs:
  - upstream credential concurrency/in-flight contracts
- touches:
  - `internal/config/`
  - `sdk/cliproxy/executionregistry/`
  - `sdk/cliproxy/executor/`
- avoid:
  - persistence and provider routing
- steps:
  1. Add validated defaulted concurrency and in-flight config contracts.
  2. Add execution lifecycle binding and an exactly-once execution registry.
  3. Add unit tests for defaults, invalid bounds, bind/end, release, cancel, and drain.
- expected outputs:
  - reusable lifecycle primitives with deterministic tests
- verification:
  - `go test ./internal/config ./sdk/cliproxy/executionregistry ./sdk/cliproxy/executor`
- stop if:
  - public behavior requires non-additive API replacement
- escalate to:
  - to-plan phase

## Wave 2
### T2 — Add Home dispatch, release, and observation contracts
- type: implementation
- inputs:
  - Wave 1 lifecycle primitives
- touches:
  - `internal/home/`
  - `internal/homeplugins/`
  - `sdk/cliproxy/auth/home_*.go`
- avoid:
  - non-Home persistence
- steps:
  1. Parse accounted/busy Home dispatch responses and expose trusted retry timing.
  2. Add selection attempt binding and exactly-once release batching.
  3. Publish bounded in-flight snapshots and lifecycle configuration revisions.
  4. Add unit and integration tests for dispatch, release, retries, and snapshots.
- expected outputs:
  - Home lifecycle completes accounting across success, error, cancellation, and drain
- verification:
  - `go test ./internal/home ./internal/homeplugins ./sdk/cliproxy/auth`
- stop if:
  - Home protocol evidence differs from upstream fixtures
- escalate to:
  - user clarification

## Wave 3
### T3 — Integrate concurrency across request execution
- type: implementation
- inputs:
  - Waves 1–2
- touches:
  - `sdk/cliproxy/auth/conductor.go`
  - `sdk/api/handlers/`
  - `internal/api/`
  - `internal/runtime/executor/`
  - `sdk/cliproxy/`
- avoid:
  - model routing and frontend
- steps:
  1. Select Home credentials through lifecycle-aware selection paths.
  2. Bind HTTP bodies, streams, websocket sessions, and terminal paths before release.
  3. Preserve busy retry headers and request-scoped failure semantics.
  4. Wire service startup, config reload, Home plugins, and bounded shutdown drain.
  5. Add regression coverage for every terminal execution path.
- expected outputs:
  - all accounted requests release once and only after resources close
- verification:
  - `go test ./internal/api ./internal/runtime/executor ./sdk/api/handlers/... ./sdk/cliproxy/...`
- stop if:
  - a local provider requires behavior outside the locked phase
- escalate to:
  - to-plan phase

## Wave 4
### T4 — Gate the phase
- type: test
- inputs:
  - Waves 1–3
- touches:
  - phase run and validation evidence only
- avoid:
  - unrelated source changes
- steps:
  1. Run the full Go test, vet, build, and whitespace gates.
  2. Record exact evidence and remaining non-Go verification gaps.
- expected outputs:
  - clean phase gate
- verification:
  - `go test ./... && go vet ./... && go build ./... && git diff --check`
- stop if:
  - any gate fails twice after one targeted correction
- escalate to:
  - check

## Risks / Watch-fors
- Double release or release-before-close corrupts Home concurrency accounting.
- Shutdown and websocket reuse paths can race with lifecycle drain.
- Wholesale upstream replacements can erase llmhub-specific Postgres and Kiro behavior.
