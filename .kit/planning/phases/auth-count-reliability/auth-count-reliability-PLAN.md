# Plan: Auth and CountTokens reliability

Phase: auth-count-reliability
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-07-22

## Goal

Correct cooldown escalation, bounded jitter, invalid-grant suspension, and CountTokens availability behavior while preserving Postgres, Amp, Kiro, and existing request-scoped semantics.

## Inputs

- Approved Phase 1 governed-tree fingerprint.
- `.kit/planning/SPEC.md`
- phase CONTEXT
- detailed fan-out plan under `.kit/plans/2026-07-22-cliproxyapi-v7.2.93-fanout/`

## Wave 1

### P2-S1 — Cooldown escalation and jitter

- type: implementation
- isolation: dedicated worktree from Phase 1 approved base
- exact touches:
  - `sdk/cliproxy/auth/conductor.go`
  - `sdk/cliproxy/auth/cooldown_backoff_test.go` (planned new)
- avoid:
  - request-time 401 refresh
  - Kiro executor changes
  - storage/schema changes
- steps:
  1. Add failing concurrency and jitter-bound fixtures.
  2. Prevent backoff advancement while `NextRecoverAt` is active.
  3. Clamp base wait and add only the remaining bounded jitter budget.
  4. Produce immutable patch/test/review artifacts.
- verification:
  - `go test ./sdk/cliproxy/auth -run 'Test.*(Cooldown|Jitter|Backoff)' -count=1`
- stop if:
  - persistence callbacks or Kiro accounting cannot remain unchanged.

## Wave 2

### P2-S2 — invalid_grant and CountTokens classification

- type: implementation
- depends on: approved and serially applied P2-S1
- isolation: dedicated worktree from P2-S1 accepted base
- exact touches:
  - `sdk/cliproxy/auth/conductor.go`
  - `sdk/cliproxy/auth/conductor_overrides_test.go`
  - `sdk/cliproxy/auth/conductor_count_tokens_test.go` (planned new)
  - `sdk/cliproxy/auth/conductor_scheduler_refresh_test.go`
  - `internal/api/modules/amp/routes_test.go`
- steps:
  1. Add structured/textual HTTP 400/401 `invalid_grant` fixtures plus unrelated-400 controls.
  2. Add generic CountTokens endpoint and explicit nested `model_not_found` fixtures.
  3. Implement narrow shared classification without handler-only policy.
  4. Prove Amp inherits the shared behavior without production route edits.
  5. Produce immutable patch/test/review artifacts.
- verification:
  - `go test ./sdk/cliproxy/auth -run 'Test.*(InvalidGrant|CountTokens|RequestScoped|Fallback)' -count=1`
  - `go test ./internal/api/modules/amp -run 'Test.*CountTokens' -count=1`
- stop if:
  - a public error schema or shared request-time refresh is required.

## Wave 3

### P2-GATE — Integrate and close

- type: test/docs
- steps:
  1. Apply approved slice patches serially.
  2. Verify changed paths equal the union of both exact allowlists.
  3. Run package, full Go, vet, build, and diff checks.
  4. Write and review the governed closure patch.
  5. Record run/check and approved fingerprint.
- closure touches:
  - `docs/decisions/0010-auth-cooldown-and-error-classification.md`
  - `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
  - `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
  - append-only harness/evidence paths
- verification:
  - `go test ./sdk/cliproxy/auth ./internal/api/modules/amp -count=1`
  - `go test ./...`
  - `go vet ./...`
  - `go build ./...`
  - `git diff --check`
- rollback:
  - on failure reverse P2-S2 then P2-S1, reverse the closure docs patch if applied, and verify the Phase 1 governed fingerprint.

## Risks / Watch-fors

- `conductor.go` ownership makes both implementation slices strictly serial.
- Preserve Phase 1 request-scoped metadata and all Kiro/Postgres callbacks.
