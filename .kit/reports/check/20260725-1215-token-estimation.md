---
id: 01KYCVQEX866578AD3EZAHY6FK
type: check
phase: token-estimation
lane: high-risk
mode: full
run_id: 01KYCVPE9326H17R0KV7DHC2Z4
proof_links:
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go test ./internal/runtime/executor/helps ./internal/runtime/executor/... -run 'ClaudeInputToken|InputTokens' -v"
    output_ref: "inline — Gate Evidence, 11 tests PASS"
    artifact_path: "internal/runtime/executor/helps/claude_input_tokens.go"
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go test ./internal/runtime/executor ./internal/translator/openai/openai/responses"
    output_ref: "inline — Gate Evidence, both packages ok"
    artifact_path: "internal/translator/openai/openai/responses/"
  - command: "go test ./internal/translator/openai/openai/responses -bench 'BenchmarkConvertOpenAIResponsesRequestWithLargeNonConvertibleToolArray' -run '^$' -benchmem"
    output_ref: "inline — 15309 iters, 77905 ns/op, 25616 B/op, 23 allocs/op"
    artifact_path: "internal/translator/openai/openai/responses/openai_openai-responses_request_test.go"
  - command: "go test ./internal/translator/openai/openai/responses -run 'Tool' -v"
    output_ref: "inline — OmitsToolSettingsWithoutTools + PreservesToolSettingsWithTools PASS"
    artifact_path: "internal/translator/openai/openai/responses/openai_openai-responses_request.go"
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go test ./..."
    output_ref: "inline — Gate Evidence, 58 packages ok, exit 0"
    artifact_path: "."
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go vet ./..."
    output_ref: "inline — Gate Evidence, exit 0"
    artifact_path: "."
  - command: "GOCACHE=/private/tmp/llmhub-go-cache go build ./..."
    output_ref: "inline — Gate Evidence, exit 0"
    artifact_path: "."
  - command: "git diff --check"
    output_ref: "inline — Gate Evidence, clean"
    artifact_path: "."
  - command: "zharness audit --json"
    output_ref: "inline — Findings/Major, 9 contract_violations"
    artifact_path: ".kit/planning/SPEC.md"
created: 2026-07-25
updated: 2026-07-25
---

# CHECK REPORT

Run ID: check-20260725-1215-token-estimation
Scope: full
Artifact Alignment: drift
Review Verdict: APPROVE with requests
Phase: token-estimation
Spec: .kit/planning/SPEC.md
Plan: .kit/planning/phases/token-estimation/token-estimation-PLAN.md
Cook Run: .kit/runs/work/20260725-1210-token-estimation.md
Created At: 2026-07-25 12:15

Depth: deep (~1100 lines attributable to this phase; 932 new lines across 3 new files, +119/-4 in `xai_executor.go`, +76/-11 in translator + `go.mod`/`go.sum`)

## Gate Evidence
- tests: `go test ./...` → pass (58 packages ok, exit 0, 0 failures)
- types: `go vet ./...` → pass (exit 0)
- lint: `staticcheck ./...` → none (binary not installed in this environment)
- build: `go build ./...` → pass (exit 0)
- whitespace: `git diff --check` → pass (clean)
- benchmark: `BenchmarkConvertOpenAIResponsesRequestWithLargeNonConvertibleToolArray` → 15309 iters, 77905 ns/op, 290.13 MB/s, 25616 B/op, 23 allocs/op

### Validation Matrix — lane `high-risk`
| Proof class | Required | Evidence |
|---|---|---|
| unit | required | ✅ 11 Claude input-token tests + `TestCountXAIInputTokensExcludesRequestStructure` |
| integration | required | ✅ stream-translation tests crossing the translator boundary: `TestTranslateStreamWithClaudeInputTokensPatchesMessageStartOnce`, `TestClaudeInputTokenStatePreservesCRLFAndNonTargetEvents` (the "stream integration" proof the phase CONTEXT names) |
| e2e | optional | none gathered — CONTEXT defers live-provider reconciliation |
| manual-check | required | ✅ Phase 2 review of `claude_input_tokens.go`; no 🔴, no 🟠 code defects |
| command-output | required | ✅ test/vet/build/diff-check/benchmark output captured above |

All required cells have matching evidence → matrix PASS.

## Artifact Alignment
- status: drift
- notes:
  - **spec coverage**: both implementation waves map to spec hooks (token state handling + request-path perf). No requirement-shaped gap.
  - **boundary compliance — drift**: this phase's Forbidden Surfaces include "auth selection and credential concurrency", and the working tree carries the full uncommitted `auth-credential-concurrency` phase alongside it. This is the mirror image of the drift recorded in `20260725-1200-auth-credential-concurrency.md` — one working tree, two phases, neither independently auditable until committed apart.
  - **no other forbidden surface hit**: no model routing or new model registration; no frontend, database, installer, or release artifact change.
  - **shared-surface attribution**: `internal/runtime/executor/` is legitimately allowed for both phases, so attribution was done per-file. `xai_executor.go` and the six Claude-translating executors carry hunks from both phases; token-estimation owns only the token-counting hunks.
  - **proof trail**: every planned verification command was run this session, including the W2 allocation benchmark. The RUN artifact is explicitly labeled reconstructed — the implementation predates it and no `work` execution is claimed.
  - **locked decisions honored**: verified in code and by named tests (see Verified clean).
  - **plan/reality mismatch**: PLAN W2 step 4 names `internal/runtime/executor/xai_executor_test.go`; the test actually lives in a new `xai_token_count_test.go`. Same allowed surface, different filename.

## Findings

### Critical
- none

### Major
- **Harness artifacts lack YAML frontmatter fences; verdict cannot be persisted.** `zharness audit --json` now reports 9 `contract_violations` across the chain: `.kit/planning/SPEC.md` missing `id`, both RUN artifacts missing `id`/`phase`/`plan_id`, and the sibling CHECK report's `run_id` not being a valid ULID. Consequence for this gate: `zharness check record --verdict APPROVE_WITH_REQUESTS --run-id work-20260725-1210-token-estimation` returned `{"error":{"code":"unknown_run_id"}}` — the RUN is not registered in the `runs` table, so this verdict is file-only. Fix by fencing the metadata blocks in `---` and issuing ULID `id` values for SPEC and both RUNs.
- **Scope drift: one working tree holds two phases.** `auth-credential-concurrency` is uncommitted alongside this phase and its surfaces are explicitly forbidden here. Both phases are individually green, so behavior is not wrong — but committing them together makes each phase's boundary unauditable. Fix by splitting into two commits along phase lines.

### Minor / Suggestions
- 🟡 **PLAN references a test filename that does not exist.** W2 step 4 names `xai_executor_test.go`; the real file is `xai_token_count_test.go`. Recorded in the RUN log, but the PLAN itself would mislead a future `to-plan` refresh.
- 💡 **Typeless content blocks bypass media exclusion.** `internal/runtime/executor/helps/claude_input_tokens.go:199-200` — the `case "":` fallback serializes an entire content object as compacted JSON. The explicit media branch at line 197 (`image`, `input_audio`, `audio`, `video`, `redacted_thinking`) is only reached when `type` is present, so a content block lacking `type` but carrying base64 payload data would be counted in full, contradicting the locked decision to exclude multimedia. The Claude API always supplies `type`, so this is defensive hardening rather than a live defect — consider skipping objects whose serialized form exceeds a size threshold, or allow-listing keys in the fallback.
- 💡 `staticcheck` is not installed, so that gate stage has no evidence. Install it or drop it from the expected Go gate so its absence is not read as a pass.

### Verified clean (no finding)
- **Patch-once semantics correct.** `ClaudeInputTokenState.apply` (`claude_input_tokens.go:275-283`) breaks after the first chunk that reports `found` and sets `handled = true`, so at most one `message_start` is ever rewritten — matching `TestTranslateStreamWithClaudeInputTokensPatchesMessageStartOnce`.
- **Authoritative usage is never overwritten.** `applyChunk:313-316` returns the chunk untouched when `message.usage.input_tokens` exists and is non-zero, and a zero estimate (line 322) or estimation error (lines 317-321) also leaves the chunk unmodified. Covered by `TestClaudeInputTokenStatePatchesMissingAndPreservesNonZero`, `TestClaudeInputTokenStateCountErrorKeepsZero`, and `TestClaudeInputTokenStateInvalidJSONKeepsZeroWithoutLoggingRequest`.
- **SSE framing preserved.** `applyChunk` strips a trailing `\r` for parsing only (lines 297-299) and rebuilds the chunk by splicing the rewritten payload back into its exact byte span (lines 330-336), leaving line endings and non-target events byte-identical. Covered by `TestClaudeInputTokenStatePreservesCRLFAndNonTargetEvents`.
- **Media and control fields excluded** for typed blocks (`claude_input_tokens.go:197`), and document sources are counted only when `source.type == "text"` (lines 206-215). Covered by `TestCountClaudeInputTokensExcludesMultimediaAndControlFields`.
- **Tokenizer init is race-safe.** `claudeInputTokenizerOnce` guards O200k codec construction (lines 72-77); `TestClaudeInputTokenizerConcurrentCount` exercises concurrent counting and asserts a single codec instance.
- **`state.codec` is an intentional test seam, not dead state** — `claude_input_tokens_test.go:336` assigns `failingClaudeInputCodec{}` to drive the estimation-error path; production callers fall through to the shared codec.
- **Tool-setting gating proven both directions** — `OmitsToolSettingsWithoutTools` and `PreservesToolSettingsWithTools` are paired, so the optimization cannot silently drop settings when convertible tools do exist.

## Next Action
- Split the working tree into two commits along phase boundaries (`auth-credential-concurrency`, then `token-estimation`) — clears the scope-drift Major on both reports.
- Repair frontmatter fences and issue ULIDs for SPEC + both RUNs, then re-run `zharness check record` for both phases so the verdicts persist.
- Optional before PR: refresh the PLAN's W2 test filename; harden the `case "":` fallback.

### Harness verdict
`zharness check record` was **not** recorded. The RUN's `Mode: full` requires it, but the call returned `{"error":{"code":"unknown_run_id","message":"check record: run_id work-20260725-1210-token-estimation not found"}}` — the RUN has no ULID `id` and is absent from the `runs` table (first Major finding). This is a blocked recording, not a `mode: simple` exemption.
