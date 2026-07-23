# Plan: WebSocket message-too-big

Phase: websocket-message-too-big
Status: ready
Wave Count: 1
Execution Owner: work
Updated At: 2026-07-22

## Goal

Close the already-implemented close-1009 behavior against the exact current six-file tree, record an approved full check, and leave a drift-free handoff without further product edits.

## Inputs

- `.kit/planning/SPEC.md`
- `.kit/planning/phases/websocket-message-too-big/websocket-message-too-big-CONTEXT.md`
- `.kit/runs/work/20260721-1709-websocket-message-too-big.md`
- `.kit/plans/2026-07-22-cliproxyapi-v7.2.93-fanout/plan.md`

## Wave 1

### P1-CLOSE — Fingerprint, retest, review, and close Phase 1

- type: test/docs
- touches:
  - `.kit/reports/check/*-websocket-message-too-big.md`
  - `.kit/HANDOFF.md`
  - `.kit/changesets/*.changeset.jsonl`
  - `.kit/harness.db`
  - `.kit/evidence/cliproxyapi-v7.2.93-backport/**`
  - `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/**`
  - `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- avoid:
  - all Phase 1 product files except read-only fingerprint/test/review
  - the known low-severity sessionless duplicate-cleanup telemetry issue
  - Phase 2 implementation
  - commit, push, PR, release, publication
- steps:
  1. Build an imported six-file patch relative to local baseline `2960b690...`; prove forward/reverse application in a scratch tree.
  2. Fingerprint the exact current six product files.
  3. Run the three focused suites and the combined three-package suite.
  4. Recompute fingerprints and require equality.
  5. Run a read-only high-risk review; no Critical or Major finding may remain.
  6. Create the US-016 story packet and update parity evidence.
  7. Record a full APPROVED check for run `01KY22EK7T10ED5QZ10GQA5R7F` only when proof belongs to the reviewed fingerprint.
  8. Record handoff; require drift-free audit, validation, and resume state.
- expected outputs:
  - Phase 1 approved check and anchored handoff.
  - Reversible imported Phase 1 patch evidence.
  - No product-code delta from this closure task.
- verification:
  - `go test ./internal/runtime/executor -run 'Test.*(MessageTooBig|1009)' -count=1`
  - `go test ./sdk/cliproxy/auth -run 'Test.*(RequestScoped|Fallback)' -count=1`
  - `go test ./sdk/api/handlers/openai -run 'Test.*WebSocket.*(1009|MessageTooBig)' -count=1`
  - `go test ./internal/runtime/executor ./sdk/cliproxy/auth ./sdk/api/handlers/openai -count=1`
  - `zharness audit --json`
  - `zharness validate --json`
  - `zharness resume --json`
- stop if:
  - the tested, reviewed, and current fingerprints differ;
  - a Critical or Major defect is verified;
  - an unexpected product path changed;
  - check/handoff anchors do not match the latest run.
- escalate to:
  - `work` for verified in-scope blockers; otherwise user clarification.

## Risks / Watch-fors

- Do not approve from stale command evidence unless the current fingerprint is proven identical.
- A blocked closure leaves the user's existing six-file WIP untouched and stops Phase 2.
