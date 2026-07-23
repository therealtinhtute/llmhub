---
id: 01KY227ZRJJX9STD7B29Y3ECHK
type: check
phase: websocket-message-too-big
lane: high-risk
mode: full
run_id: 01KY20GD3482KG0KK2V5GJ21NY
proof_links:
  - command: focused executor websocket tests
    output_ref: check session output
    artifact_path: internal/runtime/executor
  - command: focused auth request-scoped tests
    output_ref: check session output
    artifact_path: sdk/cliproxy/auth
  - command: focused handler rollback tests
    output_ref: check session output
    artifact_path: sdk/api/handlers/openai
  - command: focused race tests with count 20
    output_ref: check session output
    artifact_path: phase product files
  - command: go test ./...
    output_ref: check session output
    artifact_path: .
  - command: go vet ./...
    output_ref: check session output
    artifact_path: .
  - command: go build ./...
    output_ref: check session output
    artifact_path: .
  - command: git diff --check
    output_ref: check session output
    artifact_path: .
created: 2026-07-21
updated: 2026-07-21
---

# CHECK REPORT

Run ID: check-20260721-1705-websocket-message-too-big
Scope: full
Artifact Alignment: aligned
Review Verdict: REQUEST CHANGES
Phase: websocket-message-too-big
Spec: .kit/planning/SPEC.md
Plan: .kit/planning/phases/websocket-message-too-big/websocket-message-too-big-PLAN.md
Cook Run: .kit/runs/work/20260721-1635-websocket-message-too-big.md
Created At: 2026-07-21 17:05

## Gate Evidence

- secrets: added `api_key` values are test-only `sk-test` fixtures → pass
- tests: focused normal and repeated race tests → pass
- tests: `go test ./...` → pass
- types/static analysis: `go vet ./...` → pass
- lint: `staticcheck ./...` → unavailable (`command not found`)
- build: `go build ./...` → pass
- whitespace: `git diff --check` → pass
- traces: both wave traces scored `detailed`
- validation matrix: unit, integration, and command-output evidence present; required manual-check is not clean because two Major findings remain

## Artifact Alignment

- status: aligned
- notes:
  - Six product files stay within the phase's allowed surfaces.
  - Requirements 1-2 are covered by focused and integration fixtures.
  - Audit's temporary out-of-order pointer is resolved by recording this check against the latest run.
  - Generic `SPEC->PLAN not_yet_implemented` remains a Harness limitation because PLAN artifacts do not carry `spec_id`.

## Findings

### Critical

- none

### Major

1. `internal/runtime/executor/codex_websockets_executor.go:872-882` waits for reader-side classification after every write failure. If a non-1009 write fails while the peer leaves the read half open, the existing retry can stall until the five-minute read deadline or request cancellation, contradicting the locked non-1009 behavior. Use a nonblocking upstream-error snapshot or another mechanism that cannot delay ordinary retries.
2. `sdk/cliproxy/auth/conductor.go:832-838,899-905,920-926,1448-1455,2313-2315,2936-2944` loses the typed request-scoped marker when converting execution errors to `Result.Error`, then reconstructs it from status 413 plus message text. An untyped 413 containing `message_too_big` can skip credential accounting while still falling back, and a typed request-scoped error with different text can mutate auth state. Preserve the classification explicitly in `Result`.

### Minor / Suggestions

- Sessionless requests have two connection cleanup owners: the request defer and the transient reader goroutine. Successful requests can log a false upstream disconnect and a second-close error. Consolidate cleanup ownership while addressing the writer lifecycle.
- Add an untyped 413 control fixture and a typed request-scoped fixture whose text does not contain `message_too_big`.
- `staticcheck` is unavailable in the environment; required Go tests, vet, build, integration fixtures, and race tests passed.

## Next Action

- Return to `work` for one bounded fix containing only the two Major findings and the sessionless cleanup ownership issue, then rerun `check full`.

scope:              on target
depth:              deep
artifact_alignment: ✅ aligned
gate:               ❌ fail: required high-risk manual-check is not clean
review:             REQUEST CHANGES
blockers:           0 critical, 2 major
autofix:            0 safe_auto proposed, 0 gated_auto awaiting confirmation
verification:       go test ./... → pass
harness_verdict:    01KY227ZRJJX9STD7B29Y3ECHK
