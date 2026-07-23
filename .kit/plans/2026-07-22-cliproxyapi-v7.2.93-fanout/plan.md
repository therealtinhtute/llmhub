# CLIProxyAPI v7.2.93 backport — subagent fan-out plan

Status: awaiting approval
Updated At: 2026-07-22
Source SPEC: `.kit/planning/SPEC.md`
Roadmap: `.kit/planning/ROADMAP.md`
Upstream: `router-for-me/CLIProxyAPI` `v7.2.93` at `01f387f4`
Local baseline: `2960b690bf89232b8a23c5b8823fbe0ca831347f`

## Outcome

Finish the targeted upstream backport without widening scope, losing the existing Phase 1 work, or allowing parallel agents to edit the same working tree.

The remaining work is **not 43 product tasks**. The executable inventory is:

| Class | Count |
|---|---:|
| Phase 1 closure | 1 |
| Product implementation slices | 9 |
| Phase gates | 4 |
| Phase closures | 4 |
| Final evidence gate | 1 |
| **Total execution units** | **19** |

Read-only reviewer sidecars are mandatory but are not counted as execution units.

## Locked amendments

1. Phase 3 follows canonical upstream and Claude's schema:
   - preserve a tool-output array as structured content only when every member maps to a valid Claude tool-result content block;
   - serialize arbitrary objects, scalars, and invalid/mixed arrays as compact JSON text;
   - preserve valid empty-array behavior.
2. Codex-to-Claude events use provider `output_index` whenever the field exists, including zero; sequential indexing is fallback only when absent.
3. Cooldown wait calculation is:
   - `clamped = min(max(base, 0), maxRetryInterval)`;
   - `jitterCap = min(clamped/4, 2s, maxRetryInterval-clamped)`;
   - result is `clamped + jitter` where `0 <= jitter <= jitterCap`;
   - a non-positive maximum returns zero.
4. Tool namespace restoration uses an exact request-scoped declaration table. Effective-name collisions fail before network I/O. Existing `mcp__` names remain unchanged.
5. xAI internal-search filtering matches the original full client identity and suppresses the complete lifecycle only for undeclared provider-internal traces.

## Execution model

### Isolation

- Every implementation subagent works in an isolated worktree or scratch tree created from one immutable approved phase base.
- Parallel agents never edit the main working tree.
- Each agent returns one immutable patch, changed-path manifest, focused test log, and normalized review input.
- The main session applies reviewed patches serially in the declared order.
- Parallelism is allowed only when exact file allowlists are disjoint.

### Base representation without commits

An approved phase base is represented by:

- current `HEAD`;
- a cumulative governed-content patch from `HEAD`;
- SHA-256 manifest of governed paths;
- approved phase/check ID.

No commit, ref, push, PR, merge, release, or publication is created.

### Governed tree versus control-plane evidence

To avoid self-referential fingerprints:

- **Governed paths**: product code, tests, story files, decision records, parity report, SPEC, roadmap, and phase plans.
- **Append-only control plane**: `.kit/evidence/cliproxyapi-v7.2.93-backport/`, `.kit/runs/`, `.kit/reports/check/`, `.kit/changesets/`, and `.kit/harness.db`.
- Approved fingerprints cover governed paths only.
- A separate append-only evidence record references the governed fingerprint and the previous evidence-record hash.
- Working-tree drift checks allow only the current slice's exact governed allowlist and explicitly named append-only evidence paths; unrelated pre-existing WIP must remain byte-for-byte and status-category stable.

### Immutable slice revisions

Each slice revision is stored under:

```text
.kit/evidence/cliproxyapi-v7.2.93-backport/slices/{slice-id}/rNN/
  baseline.json
  baseline.tar
  patch
  test.log
  review.json
```

Rules:

- never overwrite a revision;
- `CHANGES_REQUESTED` creates `rNN+1` with a complete replacement artifact set;
- `baseline.json` records base ID, exact file allowlist, hashes, modes, and planned-new absent files;
- `patch` is binary/full-index and must contain no path outside the slice allowlist;
- forward and reverse application are checked in scratch trees, never by destructively resetting the main tree.

### Reviewer contract

One read-only reviewer follows each implementation slice.

`review.json` contains:

- `slice_id`, `revision`, `base_id`, `patch_sha256`;
- verdict: `APPROVED`, `CHANGES_REQUESTED`, or `BLOCKED`;
- exact scope check and unexpected paths;
- tests reviewed and output hashes;
- findings with severity, file, line, claim, evidence, and required fix;
- re-review reference and disposition of earlier findings.

An approved patch becomes phase-approved only after the phase gate and closure pass.

## Wave 0 — Close Phase 1

### P1-CLOSE

Goal: close the existing six-file WebSocket work without new product edits.

Exact product files:

- `internal/runtime/executor/codex_websockets_executor.go`
- `internal/runtime/executor/codex_websockets_executor_test.go`
- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- `sdk/api/handlers/openai/openai_responses_websocket.go`
- `sdk/api/handlers/openai/openai_responses_websocket_test.go`

Steps:

1. Build an imported Phase 1 patch from baseline `2960b690...` for exactly the six files.
2. Prove forward and reverse application in scratch trees.
3. Fingerprint the current six-file tree.
4. Run the three focused suites plus the three-package suite.
5. Recompute hashes and require equality.
6. Run a read-only high-risk review.
7. Create the US-016 story packet and update parity evidence.
8. Record a new full check against run `01KY22EK7T10ED5QZ10GQA5R7F` only if the tested fingerprint is the reviewed fingerprint.
9. Record handoff and confirm `audit`, `validate`, and `resume` are drift-free.

Verification:

```text
go test ./internal/runtime/executor -run 'Test.*(MessageTooBig|1009)' -count=1
go test ./sdk/cliproxy/auth -run 'Test.*(RequestScoped|Fallback)' -count=1
go test ./sdk/api/handlers/openai -run 'Test.*WebSocket.*(1009|MessageTooBig)' -count=1
go test ./internal/runtime/executor ./sdk/cliproxy/auth ./sdk/api/handlers/openai -count=1
```

If blocked: keep the pre-existing WIP untouched, record `BLOCKED`, and do not start Phase 2. Revertibility is proved in scratch; changing the user's imported WIP requires explicit authorization.

## Wave 1 — Phase 2: auth and CountTokens

Execution is serial because both slices own `sdk/cliproxy/auth/conductor.go`.

### P2-S1 — Cooldown escalation and jitter

Exact files:

- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/cooldown_backoff_test.go` (planned new)

Required proof:

- concurrent/repeated 429s inside one active recovery window advance once;
- deterministic jitter cases for maximum zero and base zero/below/equal/above maximum;
- persistence callbacks, scheduler state, registry updates, and Kiro accounting remain intact.

Verification:

```text
go test ./sdk/cliproxy/auth -run 'Test.*(Cooldown|Jitter|Backoff)' -count=1
```

### P2-S2 — invalid_grant and CountTokens classification

Depends on: approved and applied P2-S1 revision.

Exact files:

- `sdk/cliproxy/auth/conductor.go`
- `sdk/cliproxy/auth/conductor_overrides_test.go`
- `sdk/cliproxy/auth/conductor_count_tokens_test.go` (planned new)
- `sdk/cliproxy/auth/conductor_scheduler_refresh_test.go`
- `internal/api/modules/amp/routes_test.go`

Required proof:

- structured and textual `invalid_grant` for HTTP 400/401 uses the existing 30-minute suspension path;
- unrelated 400 errors retain normal behavior;
- generic unsupported CountTokens endpoint failures do not mutate model availability;
- explicit top-level or nested `model_not_found` remains availability-changing;
- Amp inherits the shared policy without production route changes.

Verification:

```text
go test ./sdk/cliproxy/auth -run 'Test.*(InvalidGrant|CountTokens|RequestScoped|Fallback)' -count=1
go test ./internal/api/modules/amp -run 'Test.*CountTokens' -count=1
```

### P2 gate and closure

Gate:

```text
go test ./sdk/cliproxy/auth ./internal/api/modules/amp -count=1
go test ./...
go vet ./...
go build ./...
git diff --check
```

Closure governed allowlist:

- `docs/decisions/0010-auth-cooldown-and-error-classification.md`
- `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`

The closure patch is baseline/reviewed/reversible like a product slice. Harness run/check records are append-only control-plane evidence.

## Wave 2 — Phase 3: translator content fidelity

Three slices fan out from the same Phase 2 approved base in isolated worktrees.

### P3-S1 — Shared file-data normalization

Exact files:

- `internal/translator/common/file_data.go` (planned new)
- `internal/translator/common/file_data_test.go` (planned new)
- `internal/translator/gemini/openai/chat-completions/gemini_openai_request.go`
- `internal/translator/gemini/openai/chat-completions/gemini_openai_request_test.go` (planned new if absent)
- `internal/translator/gemini-cli/openai/chat-completions/gemini-cli_openai_request.go`
- `internal/translator/gemini-cli/openai/chat-completions/gemini-cli_openai_request_test.go` (planned new if absent)
- `internal/translator/codex/openai/chat-completions/codex_openai_request.go`
- `internal/translator/codex/openai/chat-completions/codex_openai_request_test.go`

Verification:

```text
go test ./internal/translator/common ./internal/translator/gemini/openai/chat-completions ./internal/translator/gemini-cli/openai/chat-completions ./internal/translator/codex/openai/chat-completions -count=1
```

### P3-S2 — Claude tool-result content

Exact files:

- `internal/translator/claude/openai/responses/claude_openai-responses_request.go`
- `internal/translator/claude/openai/responses/claude_openai-responses_request_test.go`

Fixtures cover strings, empty arrays, valid text/image/document/search-result blocks, valid multi-block arrays, arbitrary objects, invalid/mixed arrays, nested arrays, numbers, booleans, null, and unchanged `content: []` behavior.

Verification:

```text
go test ./internal/translator/claude/openai/responses -run 'Test.*(FunctionCallOutput|ToolResult|Structured|Content)' -count=1
```

### P3-S3 — Codex-to-Claude output indices

Exact files:

- `internal/translator/codex/claude/codex_claude_response.go`
- `internal/translator/codex/claude/codex_claude_response_test.go`

Fixtures cover `output_index: 0`, non-zero provider indices, absent-field fallback, and interleaved reasoning/text/function-call lifecycles.

Verification:

```text
go test ./internal/translator/codex/claude -run 'Test.*(OutputIndex|Interleav|ContentBlock)' -count=1
```

Apply order after all reviews: P3-S1, P3-S2, P3-S3.

### P3 gate and closure

```text
go test ./internal/translator/common ./internal/translator/claude/openai/responses ./internal/translator/gemini/openai/chat-completions ./internal/translator/gemini-cli/openai/chat-completions ./internal/translator/codex/openai/chat-completions ./internal/translator/codex/claude -count=1
go test ./...
go vet ./...
go build ./...
git diff --check
```

Closure governed allowlist:

- `docs/decisions/0011-translator-structured-output-and-index-fidelity.md`
- `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`

Antigravity and Google Interactions are forbidden in this phase.

## Wave 3 — Phase 4: Codex/xAI tool protocol

Two slices fan out from the same Phase 3 approved base in isolated worktrees.

### P4-S1 — Codex declaration table and round trips

Exact files:

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

Required behavior:

- function/custom/additional/namespaced declarations build one exact request table;
- distinct original identities cannot share an outbound effective name;
- collisions return structured HTTP 400 `tool_name_collision` before network I/O;
- `mcp__` remains byte-stable;
- streaming and non-streaming outputs restore original namespace/name/type.

### P4-S2 — xAI tools and internal-search lifecycle

Exact files:

- `internal/runtime/executor/xai_executor.go`
- `internal/runtime/executor/xai_executor_test.go`

Required behavior:

- promote `additional_tools` before send;
- preserve declaration identity and choices;
- reject outbound collisions before network I/O;
- preserve declared search tools;
- suppress every event and completed output item associated with undeclared provider-internal `x_search` traces;
- compact later visible indices consistently.

Verification:

```text
go test ./internal/translator/codex/claude ./internal/translator/codex/openai/responses ./internal/translator/openai/openai/responses -count=1
go test ./internal/runtime/executor -run 'Test.*(Codex|XAI|Namespace|AdditionalTools|CustomTool|ParallelTool|XSearch|Collision|Declaration)' -count=1
```

Apply order: P4-S1 then P4-S2.

### P4 gate and closure

Run focused suites, full Go tests, vet, build, and diff check. Closure governed allowlist:

- `docs/decisions/0012-tool-namespace-and-x-search-lifecycle.md`
- `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`

## Wave 4 — Phase 5: model/display compatibility

Two slices fan out from the same Phase 4 approved base.

### P5-S1 — Model metadata

Exact files:

- `internal/registry/models/models.json`
- `internal/registry/model_definitions_test.go`

Add only Grok 4.5 and the approved additive Gemini production IDs; retain previews. Do not add GPT-5.6, Kimi K3, remote refresh, or unrelated catalog changes.

Verification:

```text
go test ./internal/registry -run 'Test.*(Grok45|GeminiProduction|Preview|ModelDefinition)' -count=1
```

### P5-S2 — Display-name propagation

Exact files:

- `internal/config/config.go`
- `internal/config/vertex_compat.go`
- `internal/config/model_display_name_test.go` (planned new)
- `internal/registry/model_registry.go`
- `sdk/cliproxy/service.go`
- `sdk/cliproxy/config_model_display_name_test.go` (planned new)
- `sdk/api/handlers/openai/openai_handlers.go`
- `sdk/api/handlers/openai/codex_client_models.go`
- `sdk/api/handlers/openai/model_display_name_test.go` (planned new)
- `sdk/api/handlers/claude/code_handlers.go`
- `sdk/api/handlers/claude/model_display_name_test.go` (planned new)
- `sdk/api/handlers/gemini/gemini_handlers.go`
- `sdk/api/handlers/gemini/gemini-cli_handlers.go`
- `sdk/api/handlers/gemini/model_display_name_test.go` (planned new)

Display names are presentation-only. IDs, upstream names, auth selection, aliases, and routing keys remain unchanged.

Verification:

```text
go test ./internal/config ./sdk/cliproxy ./sdk/api/handlers/openai ./sdk/api/handlers/claude ./sdk/api/handlers/gemini -run 'Test.*(DisplayName|ModelList|Models)' -count=1
```

Apply order: P5-S1 then P5-S2.

### P5 gate and closure

Run registry/config/service/handler packages, full Go tests, vet, build, and diff check. Closure governed allowlist:

- `docs/decisions/0013-model-display-name-presentation-contract.md`
- `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`

## Wave 5 — Final evidence

Before terminal validation:

1. ensure all story, parity, decision, roadmap, and phase-plan updates are written;
2. build a scratch-index cumulative patch that includes tracked and untracked governed paths;
3. validate the complete story packet and all decision records;
4. capture pre-command governed and unrelated-WIP manifests.

Then run:

```text
go test ./...
go vet ./...
make build-web
make build
git diff --check
```

Rebuild the scratch-index patch and manifests afterward. Final approval requires:

- all commands pass;
- governed content is unchanged by verification commands;
- unrelated pre-existing WIP is unchanged;
- only declared append-only control-plane records were added;
- all SPEC requirements 1–18 have evidence;
- no commit, push, PR, release, or publication occurred.

## Rollback

- Phase 1 imported WIP is never automatically overwritten. Its patch is proved reversible in scratch; a blocked check stops later phases.
- For Phases 2–5, reverse all current-phase implementation patches in reverse apply order if a slice blocks or the phase gate fails.
- Closure documentation is one governed closure patch with an exact allowlist; reverse it if closure validation fails.
- Append-only evidence records remain and record the failed revision/rollback result.
- After rollback, governed paths and unrelated WIP must match the prior approved manifest exactly; rerun the prior phase's focused gate.
- Never use destructive whole-tree reset or checkout operations.

## Pause points

1. After Phase 1 closure: require approval before Phase 2 implementation.
2. After every phase gate/closure: report evidence and require approval before the next phase.
3. Any missing in-scope requirement, failed test, unresolved reviewer finding, unexpected path, or non-reversible patch blocks progression.
4. After final evidence: leave the repository uncommitted and unpublished.
