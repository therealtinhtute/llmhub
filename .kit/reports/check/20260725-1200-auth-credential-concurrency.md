---
id: 01KYCVQEX1N4W28X5SFB751THG
type: check
phase: auth-credential-concurrency
lane: high-risk
mode: full
run_id: 01KYCVPE8V5CP8TWVN3FX5JEEK
proof_links:
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go test ./..."
    output_ref: "inline — Gate Evidence, 58 packages ok, exit 0"
    artifact_path: "."
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go vet ./..."
    output_ref: "inline — Gate Evidence, exit 0, no diagnostics"
    artifact_path: "."
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go build ./..."
    output_ref: "inline — Gate Evidence, exit 0"
    artifact_path: "."
  - command: "git diff --check"
    output_ref: "inline — Gate Evidence, clean after whitespace fix"
    artifact_path: ".kit/planning/phases/websocket-message-too-big/"
  - command: "go test ./sdk/cliproxy/ -run TestStopHomeLifetimeDetachesDrainsAndFlushes -v"
    output_ref: "inline — integration proof class, PASS"
    artifact_path: "sdk/cliproxy/service_home_lifecycle_test.go"
  - command: "go test ./sdk/cliproxy/auth/ -run 'TestHomeWebsocketSessionRetainsOneAccountedSelectionUntilClose|TestCloseExecutionSessionClearsHomeRuntimeAuthForSession' -v"
    output_ref: "inline — integration proof class, 2 PASS"
    artifact_path: "sdk/cliproxy/auth/home_websocket_reuse_test.go"
  - command: "go test ./internal/runtime/executor/ -run TestCodexWebsocketSessionBindsSameLifecycleAndConnectionOnce -v"
    output_ref: "inline — integration proof class, PASS"
    artifact_path: "internal/runtime/executor/websocket_lifecycle_bind_test.go"
  - command: "go test ./internal/home/ -run 'InFlight|Contract|ConcurrencyRelease' -v"
    output_ref: "inline — unit/wire-contract proof class, 3 PASS"
    artifact_path: "internal/home/"
  - command: "zharness audit --json"
    output_ref: "inline — Findings/Major, 4 contract_violations"
    artifact_path: ".kit/planning/SPEC.md"
created: 2026-07-25
updated: 2026-07-25
---

# CHECK REPORT

Run ID: check-20260725-1200-auth-credential-concurrency
Scope: full
Artifact Alignment: drift
Review Verdict: APPROVE with requests
Phase: auth-credential-concurrency
Spec: .kit/planning/SPEC.md
Plan: .kit/planning/phases/auth-credential-concurrency/auth-credential-concurrency-PLAN.md
Cook Run: .kit/runs/work/20260723-2210-auth-credential-concurrency.md
Created At: 2026-07-25 12:00

Depth: deep (313 files, +948/-36873, touches auth)

## Gate Evidence
- tests: `go test ./...` → pass (58 packages ok, exit 0, 0 failures)
- types: `go vet ./...` → pass (exit 0, no diagnostics)
- lint: `staticcheck ./...` → none (binary not installed in this environment)
- build: `go build ./...` → pass (exit 0)
- whitespace: `git diff --check` → pass (clean; two planning markdown files corrected this session)
- secrets scan: `git diff HEAD | grep -iE "(password|secret|token|api_key|private_key)"` → pass (all hits are token-counting identifiers; no credential literals)

### Validation Matrix — lane `high-risk`
| Proof class | Required | Evidence |
|---|---|---|
| unit | required | ✅ `go test ./...` 58 pkg; `internal/config`, `executionregistry`, `executor` unit suites |
| integration | required | ✅ 4 tests crossing package public API boundaries (service drain, Home websocket accounting, executor lifecycle bind) |
| e2e | optional | none gathered |
| manual-check | required | ✅ Phase 2 review pass — Security/Performance/Architecture/Code Quality; no 🔴, no 🟠 code defects |
| command-output | required | ✅ test/vet/build/diff-check output captured above |

All required cells have matching evidence → matrix PASS.

## Artifact Alignment
- status: drift
- notes:
  - **spec coverage**: the phase goal (Home-dispatched credential accounting with exactly-once lifecycle release, bounded drain, in-flight observation) is implemented across all 3 implementation waves. No requirement-shaped gap found.
  - **boundary compliance — drift**: the working tree co-mingles the sibling `token-estimation` phase. `internal/translator/openai/openai/responses/` and the `go.mod`/`go.sum` tokenizer v0.8.1 bump are **not** in this phase's Allowed Surfaces. `internal/runtime/executor/helps/claude_input_tokens.go` sits inside an allowed directory but serves token estimation, not lifecycle binding. The gate cannot cleanly attribute the diff to this phase alone.
  - **no forbidden surface hit**: no frontend/management-panel change; no model-routing or registry change; no llmhub DB schema change (the deleted `scripts/schema/*.sql` are zharness harness SQLite schema, pre-existing cleanup acknowledged by the RUN preflight note); `internal/store/` untouched.
  - **proof trail**: every wave's planned verification command was run this session and recorded. The RUN artifact previously showed W2 as `running` with `changed files: none` while the code was already present; corrected this session before gating.
  - **locked decisions honored**: release strictly follows resource close (verified in code, see Findings); busy dispatch preserves retry timing without mutating credentials; Postgres auth persistence extended, not replaced.

## Findings

### Critical
- none

### Major
- **Harness artifacts lack YAML frontmatter delimiters.** `zharness audit --json` reports 4 `contract_violations`: `.kit/planning/SPEC.md` missing key `id`, and the RUN artifact missing `id`, `phase`, `plan_id`. Both files begin their metadata without the opening `---` fence, so the parser cannot read fields that are physically present (SPEC.md line 1 literally is `id: 01KY1T7GGY3DBEW2NAAV620VCK`). Direct consequence: `zharness check record --run-id work-20260723-2210-auth-credential-concurrency` fails with `unknown_run_id` — the RUN was never registered in the `runs` table, so this verdict could not be persisted to the DB. Fix by wrapping both files' metadata blocks in `---` fences and giving the RUN a ULID `id` plus `phase` / `plan_id` keys.
- **Scope drift: two phases share one working tree.** `token-estimation` work (translator response request builder, tokenizer dependency bump, Claude input-token helper) is uncommitted alongside `auth-credential-concurrency`. `internal/translator/` is outside this phase's Allowed Surfaces. Both phases are individually code-complete and green, so behavior is not wrong — but committing them as one unit would make either phase's boundary unauditable. Fix by splitting into two commits along phase lines before merging.

### Minor / Suggestions
- 🟡 **Dead error field yields a misleading contract.** `sdk/cliproxy/executionregistry/registry.go:50` declares `closeErr error`; it is read at lines 406 and 437 but never assigned anywhere in the package. `Registry.Close()` therefore always returns `nil`, so callers cannot distinguish a clean shutdown from a failed resource close (individual close failures are only logged, in `closeResource`). Either assign `closeErr` from the close path or drop the field and change `Close()` to return no error.
- 🟡 **Phase plan lists two surfaces that do not match the repo.** `internal/homeplugins/` (W2 `touches`) does not exist in llmhub, and `internal/api/` (W3 `touches`) required no change because lifecycle binding lands in `sdk/api/handlers` and the executors. Both are now documented in the RUN log, but the PLAN itself is stale and would mislead a future `to-plan` refresh.
- 💡 `staticcheck` is not installed, so that gate stage has no evidence. Install it or drop it from the expected Go gate so the absence is not mistaken for a pass.
- 💡 Postgres integration is unexercised: `internal/store` has 2 tests that skip on missing `LLMHUB_POSTGRES_TEST_DSN`. Not blocking — this phase does not touch `internal/store` — but a `high-risk` lane whose locked decisions concern Postgres auth persistence would be better served by running that suite once with a DSN set.

### Verified clean (no finding)
- **Exactly-once release ordering is correct** — the phase's top stated risk. `Scope.EndWithRelease` (`registry.go:271`) guards the whole body in `s.ended.Do`, calls `waitForBoundResourceClose()` at line 284, and only then calls `markReleasedLocked` at line 287. Release therefore cannot precede resource close, and cannot happen twice.
- **All four `Bind` error paths release their resource.** `executor/lifecycle.go:30` joins `closeResource()`; `auth/home_selection.go:167` closes `resources` and ends the scope; `auth/home_session.go:169-170` rolls back `runtimeAuthBound` and forgets the runtime auth; `executor/codex_websockets_executor.go:193` returns the error and both callers (lines 452, 690) route through `invalidateUpstreamConn`, which closes the conn via `lifecycle.End(reason)` or `conn.Close()` (lines 1705-1709).
- **Lock ordering is consistent** — `registry.mu` is always acquired before `scope.mu`; the release sink is invoked with no registry mutex held (`registry.go:288-292`), and `startBoundResourceClose` is called outside `r.mu` in both `Drain` and `EndWithRelease`.
- **No hardcoded credentials** in the diff.

## Next Action
- Split the working tree into two commits along phase boundaries (`auth-credential-concurrency`, then `token-estimation`) before merging — addresses the scope-drift Major.
- Repair the frontmatter fences on `.kit/planning/SPEC.md` and the RUN artifact, then re-run `zharness check record` so this verdict persists to the DB.
- Optional cleanups before PR: resolve `closeErr`, refresh the phase PLAN's `touches` lists.

### Harness verdict
`zharness check record` was **not** recorded. The RUN artifact's `Mode: full` requires it, but the call returned `{"error":{"code":"unknown_run_id","message":"check record: run_id work-20260723-2210-auth-credential-concurrency not found"}}` — the RUN has no ULID `id` and was never registered in the `runs` table (see the first Major finding). This is a blocked recording, not a `mode: simple` exemption.
