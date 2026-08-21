---
id: 01M0FR0JSNWR6FKSBBZCZSYYEH
type: plan
intake_id: 01M0FR170W7VPYF2QB0S0K2S7S
lane: high-risk
status: active
created: 2026-08-20
updated: 2026-08-20
---

# Plan: CLIProxyAPI v7.2.137 parity follow-up

## Outcome
- result: llmhub ports only the approved v7.2.136-v7.2.137 parity deltas in three independently verifiable slices: custom headers, native-parity micro-fixes, and Gemini/Antigravity leading-user turns.
- success_signals:
  - R10a preserves existing header behavior while resolving approved `$VAR` custom headers and caller-owned Claude headers.
  - R10b passes focused Claude, Codex, Gemini, Antigravity, client-error, and executor tests without changing DB-only config or SDK boundaries.
  - R10c prevents model-first Gemini/Antigravity requests from being rejected by upstream turn-order validation.
  - `go test ./...`, `make build`, and `git diff --check` are run at the aggregate gate; known baseline failures are recorded honestly.
  - No deferred or rejected v7.2.137 delta is silently implemented.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint v7.2.137 at `85d2faddd17e`.
  - `docs/upstream/cliproxyapi-gap-v7.2.135..v7.2.137.json` — structural gap matrix for the pinned release delta.
  - `docs/upstream/cliproxyapi-ledger-v7.2.135..v7.2.137.md` — 25 non-merge commits and release surfaces.
  - `docs/plans/done/cliproxyapi-v7.2.135-parity.md` — completed parity scope and final-gate dispositions.
  - `CLAUDE.md` — DB-only runtime config, SDK boundaries, and frontend test constraints.
- requirements:
  - R1 [accepted]: R10a resolves approved `$VAR` custom-header values, threads request headers through the affected executor paths, and preserves incoming headers in Claude caller-owned mode without weakening existing header precedence. | source: `e424bfad00a6`, `5fef17e2ecd1`.
  - R2 [accepted]: Claude OpenAI request conversion always emits `stop` as an array when converting `stop_sequences`. | source: `4ac37ed3cd53`.
  - R3 [accepted]: Claude OpenAI request conversion accepts both `max_tokens` and `max_completion_tokens` with deterministic precedence. | source: `3230e37023db`.
  - R4 [accepted]: Claude executor preserves approved native subagent, environment, client-app, and additional-protection headers. | source: `85d2faddd17e`.
  - R5 [accepted]: Codex Responses request conversion removes `prompt_cache_retention`, and Gemini tool-parameter schema sanitization removes `encrypted`. | source: `2005788fc396`, `ac0d1888c04e`.
  - R6 [accepted]: Antigravity OpenAI request conversion supports `video_url`, and Gemini translation attaches sibling tool images to the nearest function response. | source: `497673bf6bda`, `79ef3618d650`.
  - R7 [accepted]: HTTP 499 and client-cancellation failures do not become forced actionable error logs while other actionable failures retain existing logging behavior. | source: `0f69f09a70b0`.
  - R8 [accepted]: Claude CCH signing is gated by the approved upstream-origin condition; fingerprint-profile configuration remains deferred. | source: `aec70dfec4ab`.
  - R9 [accepted]: Gemini, Vertex, AI Studio, and Antigravity model-first request/count-token paths prepend the required empty user turn without changing already-valid turn ordering. | source: `62f5a2798c52`.
  - R10 [accepted]: All ports use existing llmhub interfaces and preserve Postgres-authoritative config, public SDK compatibility, and the embedded management panel. | source: `CLAUDE.md`, completed parity plan.

## Non-goals
- NG1: Fingerprint-profile configuration and Claude fingerprint-policy capability (`f1b0431c7718`, related bundled portions of `aec70dfec4ab`).
- NG2: Structured Gemini `function_call_output` refactor, Claude `tool_search_tool_result` remapping, Responses compact cooldown semantics, and setup-tokens/403 OAuth handling (`20f84e78c162`, `788e9b792852`, `ec105dac9416`, `8aa6868d0dc1`).
- NG3: README/sponsor churn, upstream-only rate-limit or compaction-trigger machinery, and the already-present Claude identity refactor are not ports.
- NG4: No new `config.example.yaml` feature fields, `XAIKey`/`InteractionsKey` types, frontend test files, wholesale upstream merges, or pushes.

## Approach and Risks
- approach: port only the accepted release-delta behaviors against the pinned upstream checkpoint; map each behavior to existing llmhub symbols, add focused regression coverage, and keep each story independently reviewable and commit-scoped to its owned paths.
- sequencing:
  - R10a is the header-signature slice. Complete its shared helper/signature change and all affected call sites before R10b touches overlapping Claude executor paths.
  - R10c has disjoint executor surfaces and may run in parallel with R10a; R10b's translator, client-error, and non-Claude tasks may fan out after the R10a ownership boundary is established, but the Claude-header task waits for R10a completion.
  - Each agent stages explicit owned paths only; no shared-file edits are accepted without reconciling ownership first.
- constraints:
  - Runtime configuration remains Postgres-authoritative; do not add feature fields to `config.example.yaml` or introduce upstream-only config types.
  - Preserve public `sdk/` interfaces and the embedded management panel; do not add frontend test files under `web/`.
  - Use semantic ports rather than wholesale upstream merges; preserve stronger local behavior and existing header precedence.
  - Do not push; commit each completed slice only after the `check` gate and secret scan.
- risks:
  - Custom-header interpolation or incoming-header precedence could leak caller values or override configured security headers; mitigate with explicit precedence, missing-variable, caller-owned, and non-caller-owned regression tests.
  - Changing executor signatures can leave a provider path unthreaded or break stream/token-count flows; mitigate with call-graph mapping, package-wide compilation, and focused executor tests before fan-out.
  - Native provider payload fixes can alter already-valid requests; keep transformations idempotent and test both model-first/valid ordering and existing request shapes.
  - Aggregate `go test ./...` may retain known baseline failures in self-update and stale-worktree no-copy checks; record exact output rather than masking or broadening scope.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: r10a-custom-headers
    story_id: 01M0FR4267GSYRQ5GQ6EKS5H8A
    status: checked
    goal: Resolve approved `$VAR` custom headers and preserve caller-owned Claude incoming headers (R10a).
    depends_on: none
    allowed_surfaces:
      - `internal/util/` custom-header helpers and focused tests
      - `internal/runtime/executor/` affected executor call paths and focused tests
    avoided_surfaces:
      - `config.example.yaml`, `internal/config/`, `sdk/`, `web/`, and `internal/managementasset/static/`
      - unrelated translator behavior and any deferred/rejected v7.2.137 delta
    waves:
      - wave: 1 — map the custom-header helper and every affected executor call site against `e424bfad00a6` and `5fef17e2ecd1`
        tasks:
          - task: Inventory the shared custom-header helper, affected executor signatures/callers, and existing configured-versus-incoming header precedence; expected output is a local symbol/path map and explicit ownership list.
          - task: Identify caller-owned Claude mode and the stream, non-stream, and token-count paths that must carry request headers; expected output is a no-unthreaded-call-site checklist.
        checks:
          - check: `rg -n "custom.?header|CustomHeader|caller.?owned|request headers|RequestHeaders" internal/util internal/runtime/executor`
          - check: `go test ./internal/util/... ./internal/runtime/executor/...`
        stop_conditions:
          - A required call site is outside the allowed surfaces or public SDK signatures would need to change.
          - Existing header precedence cannot be stated precisely from local code and tests.
        escalation: Record `NEEDS_CONTEXT` with the unresolved symbol and stop before implementation; do not infer provider precedence from upstream alone.
      - wave: 2 — implement request-header-aware interpolation and thread headers through the affected executor paths
        tasks:
          - task: Implement `$VAR` resolution from downstream request headers with safe missing-variable handling and unchanged literal/static behavior; expected output is a shared helper regression test matrix.
          - task: Thread request headers through affected Gemini, Codex, Claude, and other executor paths without changing public SDK contracts; expected output is a compiling call graph with no dropped request-header path.
          - task: Preserve existing configured-header precedence and incoming native headers in Claude caller-owned mode; expected output is explicit caller-owned and non-caller-owned request tests.
        checks:
          - check: `go test ./internal/util/...`
          - check: `go test ./internal/runtime/executor/... -run 'Header|Claude|Codex|Gemini|Antigravity'`
          - check: `go test ./internal/runtime/executor/...`
        stop_conditions:
          - A configured header can be overridden by an untrusted incoming value outside the approved caller-owned rule.
          - Any stream, non-stream, or token-count executor path fails to compile or loses request headers.
        escalation: Revert only the incomplete R10a edits, report the failing path and precedence evidence, and keep R10b header work blocked.
      - wave: 3 — verify the slice and capture scope evidence
        tasks:
          - task: Run the focused regression matrix and inspect the diff for DB-only config, SDK, embedded-panel, and ownership invariants; expected output is a clean R10a diff plus captured command results.
        checks:
          - check: `go test ./internal/util/... ./internal/runtime/executor/...`
          - check: `git diff --check`
          - check: `git diff -- config.example.yaml sdk web internal/managementasset/static`
        stop_conditions:
          - Focused tests fail after one targeted correction or the diff leaves the approved surfaces.
        escalation: Mark the phase blocked at the first second failure, append the exact output to Progress, and do not stage or commit.
  - phase_slug: r10b-native-parity
    story_id: 01M0FR426FG9XE3JNAGAY0J6EE
    status: checked
    goal: Port approved Claude, Codex, Gemini, Antigravity, logging, and CCH native-parity micro-fixes (R10b).
    depends_on: none
    execution_gate: Begin after R10a has established the shared header ownership boundary; the Claude-header task must not edit a file still owned by an active R10a task.
    allowed_surfaces:
      - `internal/translator/` approved Claude, Codex, Gemini, and Antigravity conversion paths and tests
      - `internal/runtime/executor/` approved Claude native-header/CCH and provider-specific behavior plus tests
      - `internal/api/middleware/` and `internal/clienterror/` HTTP 499/client-cancellation logging paths and tests
    avoided_surfaces:
      - `internal/config/`, `config.example.yaml`, `sdk/cliproxy/`, `web/`, `internal/managementasset/static/`, README/assets, and all non-goal commits
    waves:
      - wave: 1 — fan out disjoint translator and client-error micro-fixes
        tasks:
          - task: Make Claude `stop` serialization array-based and accept `max_tokens` plus `max_completion_tokens` with deterministic precedence; expected output is focused OpenAI-to-Claude translator coverage.
          - task: Filter Codex `prompt_cache_retention` and strip Gemini tool-schema `encrypted` metadata; expected output is request-payload tests proving both fields are absent without altering unrelated fields.
          - task: Support Antigravity `video_url` and attach sibling tool images to the nearest function response; expected output is translation tests for adjacent and non-adjacent image/tool cases.
          - task: Exclude HTTP 499 and client cancellations from forced actionable logs while preserving other actionable failures; expected output is middleware/client-error/runtime logging coverage.
        checks:
          - check: `go test ./internal/translator/...`
          - check: `go test ./internal/api/middleware/... ./internal/clienterror/...`
          - check: `go test ./internal/runtime/executor/...`
        stop_conditions:
          - A translator changes an already-valid field shape outside the accepted behavior or logging suppresses a non-client actionable failure.
        escalation: Isolate the failing task, record `DONE_WITH_CONCERNS` or `BLOCKED_VERIFICATION` with output, and leave unrelated fan-out results intact.
      - wave: 2 — port Claude native headers and the CCH origin gate
        tasks:
          - task: Preserve approved native subagent, environment, client-app, and additional-protection headers, reusing R10a's caller-owned boundary; expected output is header-presence and precedence tests.
          - task: Gate Claude CCH signing by the approved upstream-origin condition while leaving fingerprint-profile capability deferred; expected output is origin-positive/origin-negative signing tests.
        checks:
          - check: `go test ./internal/runtime/executor/... -run 'Claude|Header|CCH'`
          - check: `go test ./internal/runtime/executor/...`
        stop_conditions:
          - CCH signing occurs for a non-approved origin or native headers are copied across caller-owned boundaries incorrectly.
        escalation: Stop the Claude task, preserve the evidence, and resolve the security-sensitive precedence/origin decision before any commit.
      - wave: 3 — verify the slice and reconcile only owned conflicts
        tasks:
          - task: Run the native-parity package matrix, inspect staged paths, and confirm no deferred/rejected delta was included; expected output is a reviewable R10b diff and exact verification record.
        checks:
          - check: `go test ./internal/translator/... ./internal/runtime/executor/... ./internal/api/middleware/... ./internal/clienterror/...`
          - check: `git diff --check`
          - check: `git status --short`
        stop_conditions:
          - Any conflict crosses a phase ownership boundary or the focused matrix needs a second fix attempt.
        escalation: Do not stage conflicting files; append the conflict and owner to Progress and route back to the phase lead.
  - phase_slug: r10c-gemini-leading-user-turn
    story_id: 01M0FR426Q85GCF3G8VQY1EGF0
    status: checked
    goal: Prepend the required empty user turn for model-first Gemini-family requests (R10c).
    depends_on: none
    allowed_surfaces:
      - `internal/runtime/executor/` Gemini, Vertex, AI Studio, and Antigravity model/count-token paths and focused tests
    avoided_surfaces:
      - `internal/translator/`, `sdk/`, `internal/config/`, `config.example.yaml`, `web/`, and unrelated executor behavior
    waves:
      - wave: 1 — map model-first validation and request/count-token construction against `62f5a2798c52`
        tasks:
          - task: Identify the shared turn-order helper and every Gemini, Vertex, AI Studio, and Antigravity model/count-token call path; expected output is a provider-matrix map distinguishing model-first and already-valid histories.
        checks:
          - check: `rg -n "contents|role|count.?tokens|GenerateContent|Vertex|AI.?Studio|Antigravity" internal/runtime/executor`
          - check: `go test ./internal/runtime/executor/...`
        stop_conditions:
          - A required path is a translator or SDK surface, or local validation semantics contradict the accepted requirement.
        escalation: Record `NEEDS_CONTEXT` and stop without modifying provider payload code.
      - wave: 2 — normalize only model-first histories
        tasks:
          - task: Prepend one empty user turn where provider validation requires it for normal model and count-token requests, without duplicating a leading user turn or mutating valid ordering; expected output is an idempotent helper and provider wiring.
          - task: Preserve streaming and non-streaming request behavior across all four provider families; expected output is unchanged payloads except for the required leading turn.
        checks:
          - check: `go test ./internal/runtime/executor/... -run 'Gemini|Vertex|AIStudio|Antigravity'`
          - check: `go test ./internal/runtime/executor/...`
        stop_conditions:
          - A valid existing user turn is duplicated, or any provider receives a leading turn outside the model-first condition.
        escalation: Revert the targeted normalization, retain the failing fixture, and escalate as `BLOCKED_VERIFICATION`.
      - wave: 3 — verify the provider matrix and scope invariants
        tasks:
          - task: Run request/count-token regression coverage and inspect the diff for translator, SDK, config, and frontend drift; expected output is a clean R10c diff with provider-matrix evidence.
        checks:
          - check: `go test ./internal/runtime/executor/... -run 'Gemini|Vertex|AIStudio|Antigravity'`
          - check: `git diff --check`
          - check: `git diff -- internal/translator sdk internal/config config.example.yaml web`
        stop_conditions:
          - Focused tests fail after one targeted correction or scope inspection finds an unapproved path.
        escalation: Stop before staging, append exact output to Progress, and route the issue to the phase lead.
- aggregate_gate:
  order: after all three phases are checked
  checks:
    - check: `go test ./...`
    - check: `make build`
    - check: `git diff --check`
  evidence: Record exact pass/fail output, known baseline failures, changed paths, and deferred/rejected delta confirmation in Validation and Progress; never claim clean aggregate success when baseline failures remain.
  invariants:
    - `config.example.yaml` has no new feature fields.
    - Public SDK signatures remain compatible.
    - No new `web/**/*_test.go` files exist.
    - Only approved R10 paths are staged.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- `2026-08-20T14:19:41Z` — wave 1. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Phase r10a-custom-headers started (in-progress); executing the planned header-helper inventory before implementation..
- `2026-08-20T14:26:54Z` — wave 1, task Inventory the shared custom-header helper, affected executor signatures/callers, and existing configured-versus-incoming header precedence; expected output is a local symbol/path map and explicit ownership list.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Mapped internal/util/header_helpers.go and all affected Claude, Codex, Gemini/Vertex, OpenAI-compatible, and XAI call paths. Focused executor tests pass; the internal/util package check retains the known stale-worktree no-copy invariant failure..
- `2026-08-20T14:26:54Z` — wave 1, task Identify caller-owned Claude mode and the stream, non-stream, and token-count paths that must carry request headers; expected output is a no-unthreaded-call-site checklist.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Confirmed Options.Headers originates in sdk/api/handlers and is dropped by Claude custom-header calls, including stream and count-token paths; caller-owned/native restoration boundaries are identified. Known util no-copy baseline remains isolated..
- `2026-08-20T14:27:03Z` — wave 1. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Wave 1 complete: R10a header helper, request-header sources, caller-owned Claude boundaries, and provider call-site ownership are mapped. Proceeding to implementation; internal/util no-copy failures are a known stale-worktree baseline and focused executor tests pass..
- `2026-08-20T14:48:19Z` — wave 2, task Implement `$VAR` resolution from downstream request headers with safe missing-variable handling and unchanged literal/static behavior; expected output is a shared helper regression test matrix.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Implemented case-insensitive `$VAR` resolution with first non-empty incoming value, omitted unresolved variables, static-header preservation, Gin-context fallback, and regression coverage. Focused util tests pass; full util remains affected only by the known stale-worktree no-copy invariant..
- `2026-08-20T14:48:19Z` — wave 2, task Thread request headers through affected Gemini, Codex, Claude, and other executor paths without changing public SDK contracts; expected output is a compiling call graph with no dropped request-header path.. task_status: `DONE`. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Threaded Options.Headers through Claude execute/stream/count, Codex normal/compact/stream/image/websocket/alpha-search, Gemini/Vertex, OpenAI-compatible, and XAI request paths; package-wide executor tests pass and no SDK signatures changed..
- `2026-08-20T14:48:19Z` — wave 2, task Preserve existing configured-header precedence and incoming native headers in Claude caller-owned mode; expected output is explicit caller-owned and non-caller-owned request tests.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Explicit incoming headers now feed Claude header preservation and custom resolution, with caller-owned/static/stream regression coverage. Full utility gate retains only the pre-existing stale-worktree no-copy failure; executor tests pass..
- `2026-08-20T14:48:28Z` — wave 2. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Wave 2 complete: custom-header interpolation and request-header threading are implemented across all live Options.Headers executor paths, including local Codex alpha-search. Focused utility and full executor tests pass; full utility remains blocked only by stale sibling-worktree no-copy invariant findings..
- `2026-08-20T14:49:07Z` — wave 3, task Run the focused regression matrix and inspect the diff for DB-only config, SDK, embedded-panel, and ownership invariants; expected output is a clean R10a diff plus captured command results.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Focused utility and executor tests pass; full internal/util fails only on known stale sibling-worktree no-copy invariant findings. git diff --check and forbidden-surface diff checks pass; changed source paths are limited to R10a executors/util plus two tests..
- `2026-08-20T14:49:20Z` — wave 3. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: Wave 3 complete: R10a focused verification and forbidden-surface checks pass; only the known stale sibling-worktree internal/util no-copy invariant remains. R10a is ready for the durable phase gate and commit review..
- `2026-08-20T14:56:27Z` — wave 3. run: `01M0FRM5Y916HYE7EFXWFMR14D`. summary: R10a durable gate recorded APPROVE_WITH_REQUESTS; focused utility and executor proofs pass, while the combined utility gate retains only the stale sibling-worktree no-copy baseline concern. Phase is checked and ready for an owned-path commit..
- `2026-08-20T14:59:25Z` — wave 1. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: Phase r10b-native-parity started (in-progress); R10a header ownership is checked and the disjoint translator/client-error fan-out is ready..
- `2026-08-20T23:56:36Z` — wave 1, task Make Claude stop serialization array-based and accept max_tokens plus max_completion_tokens with deterministic precedence; expected output is focused OpenAI-to-Claude translator coverage.. task_status: `DONE`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: Claude translator tests pass: stop always serializes as an array; max_tokens takes precedence over max_completion_tokens, with fallback behavior preserved..
- `2026-08-20T23:56:36Z` — wave 1, task Filter Codex prompt_cache_retention and strip Gemini tool-schema encrypted metadata; expected output is request-payload tests proving both fields are absent without altering unrelated fields.. task_status: `DONE`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: Codex and Gemini schema tests pass: prompt_cache_retention is removed, encrypted metadata is stripped, and an actual encrypted property remains intact..
- `2026-08-20T23:56:36Z` — wave 1, task Support Antigravity video_url and attach sibling tool images to the nearest function response; expected output is translation tests for adjacent and non-adjacent image/tool cases.. task_status: `DONE`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: Antigravity translator tests pass for video_url conversion and adjacent/non-adjacent sibling image attachment with ordering preserved..
- `2026-08-20T23:56:36Z` — wave 1, task Exclude HTTP 499 and client cancellations from forced actionable logs while preserving other actionable failures; expected output is middleware/client-error/runtime logging coverage.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: Middleware and executor logging tests pass; this branch has no internal/clienterror package or upstream StreamUsageBuffer equivalent, so no upstream-only API was introduced..
- `2026-08-20T23:56:43Z` — wave 1. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: R10b wave 1 complete: all four disjoint translator and logging slices are applied and package-verified; middleware path is used because this branch has no internal/clienterror equivalent, and the known stale sibling-worktree utility no-copy baseline remains outside the slice..
- `2026-08-20T23:57:54Z` — wave 2, task Preserve approved native subagent, environment, client-app, and additional-protection headers, reusing R10a caller-owned boundary; expected output is header-presence and precedence tests.. task_status: `DONE`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. trace: `01M0GSRQM2MX3EVRMQR5VBSTYJ`. summary: Claude incoming native headers are copied case-insensitively, configured headers retain final precedence, and focused/full executor tests pass..
- `2026-08-20T23:57:54Z` — wave 2, task Gate Claude CCH signing by the approved upstream-origin condition while leaving fingerprint-profile capability deferred; expected output is origin-positive/origin-negative signing tests.. task_status: `DONE`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. trace: `01M0GSRQM2MX3EVRMQR825026G`. summary: API-key experimental CCH signing is restricted to HTTPS api.anthropic.com with default/443 ports; custom or unapproved origins omit CCH, OAuth behavior remains independent, and origin matrix tests pass..
- `2026-08-20T23:57:54Z` — wave 2. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. trace: `01M0GSRQMASXBN410NYTZFV3G8`. summary: R10b wave 2 complete: Claude native-header preservation and approved-origin CCH gating are implemented and verified by focused and full executor tests; fingerprint-profile configuration remains deferred..
- `2026-08-20T23:57:54Z` — wave 3, task Run the native-parity package matrix, inspect staged paths, and confirm no deferred/rejected delta was included; expected output is a reviewable R10b diff and exact verification record.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. trace: `01M0GSRQMJ0H84DZAB6Y2ZC8JK`. summary: Translator, executor, middleware, focused schema, and staged/unstaged whitespace checks pass. The local branch has no internal/clienterror package, and full internal/util remains affected only by stale sibling-worktree no-copy findings; neither was expanded into R10b..
- `2026-08-20T23:57:54Z` — wave 3. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. trace: `01M0GSRQMSBYEXAT855ERMPGBC`. summary: R10b durable gate recorded APPROVE_WITH_REQUESTS; approved paths are verified, no SDK/config/frontend/deferred surfaces were touched, and the known baseline concerns are explicit..
- `2026-08-20T23:58:11Z` — wave 2, task Preserve approved native subagent, environment, client-app, and additional-protection headers, reusing R10a caller-owned boundary; expected output is header-presence and precedence tests.. task_status: `DONE`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: Claude incoming native headers are copied case-insensitively, configured headers retain final precedence, and focused/full executor tests pass..
- `2026-08-20T23:58:11Z` — wave 2, task Gate Claude CCH signing by the approved upstream-origin condition while leaving fingerprint-profile capability deferred; expected output is origin-positive/origin-negative signing tests.. task_status: `DONE`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: API-key experimental CCH signing is restricted to HTTPS api.anthropic.com with default/443 ports; custom or unapproved origins omit CCH, OAuth behavior remains independent, and origin matrix tests pass..
- `2026-08-20T23:58:11Z` — wave 2. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: R10b wave 2 complete: Claude native-header preservation and approved-origin CCH gating are implemented and verified by focused and full executor tests; fingerprint-profile configuration remains deferred..
- `2026-08-20T23:58:11Z` — wave 3, task Run the native-parity package matrix, inspect staged paths, and confirm no deferred/rejected delta was included; expected output is a reviewable R10b diff and exact verification record.. task_status: `DONE_WITH_CONCERNS`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: Translator, executor, middleware, focused schema, and staged/unstaged whitespace checks pass. The local branch has no internal/clienterror package, and full internal/util remains affected only by stale sibling-worktree no-copy findings; neither was expanded into R10b..
- `2026-08-20T23:58:11Z` — wave 3. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: R10b durable gate recorded APPROVE_WITH_REQUESTS; approved paths are verified, no SDK/config/frontend/deferred surfaces were touched, and the known baseline concerns are explicit..
- `2026-08-20T23:59:14Z` — wave 3. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. summary: R10b plan synchronization recorded after the durable gate: phase checked, check 01M0GSQ4NJ9Y3A8A54G5FTA9EP linked, wave traces and proof gaps captured, and next action is owned-path commit followed by R10c..
- `2026-08-21T00:06:15Z` — wave 1. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: Phase r10c-gemini-leading-user-turn started (in-progress); mapping model-first Gemini-family validation and all normal, streaming, and count-token construction paths..
- `2026-08-21T00:11:01Z` — wave 1, task Identify the shared turn-order helper and every Gemini, Vertex, AI Studio, and Antigravity model/count-token call path; expected output is a provider-matrix map distinguishing model-first and already-valid histories.. task_status: `DONE`. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: Inventory complete: no existing shared leading-turn helper; Gemini Execute/stream/CountTokens and Vertex normal/stream/count paths use top-level contents, AI Studio translateRequest covers normal/stream/count, and Antigravity Execute/stream/CountTokens use request.contents after replay. Exact rg inventory and go test ./internal/runtime/executor/... pass; Claude models must remain untouched in Antigravity..
- `2026-08-21T00:11:01Z` — wave 1. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: R10c wave 1 complete: provider matrix and insertion points are mapped, with normalization deferred to executor boundaries after payload/replay rewrites and before upstream request construction..
- `2026-08-21T12:54:05Z` — wave 2, task Prepend one empty user turn where provider validation requires it for normal model and count-token requests, without duplicating a leading user turn or mutating valid ordering. task_status: `DONE`. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: Added idempotent helps.EnsureGeminiLeadingUserContent with no-copy valid-path behavior; wired Gemini, Vertex, AI Studio, and Antigravity normal/count-token boundaries, including Antigravity Claude bypass. Verified with go test ./internal/runtime/executor/... -run 'Gemini|Vertex|AIStudio|Antigravity' -count=1 and full executor tests..
- `2026-08-21T12:54:05Z` — wave 2, task Preserve streaming and non-streaming request behavior across all four provider families. task_status: `DONE`. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: Applied normalization after payload/replay rewrites and before upstream request construction for Gemini, Vertex, AI Studio, and Antigravity streaming/non-streaming paths. Regression fixtures cover normal, stream, count-token, nested payload, idempotency, and Claude bypass; focused tests and git diff --check pass..
- `2026-08-21T12:54:11Z` — wave 2. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: R10c wave 2 complete: shared leading-user normalization is implemented and wired across Gemini, Vertex, AI Studio, and Antigravity normal/stream/count-token paths; focused and full executor tests pass, with Claude bypass and scope checks verified..
- `2026-08-21T12:54:46Z` — wave 3, task Run request/count-token regression coverage and inspect the diff for translator, SDK, config, and frontend drift. task_status: `DONE`. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: R10c provider-matrix regression command passes; git diff --check passes; forbidden-surface diff is empty and working tree contains only the active plan plus approved executor/helper/test files..
- `2026-08-21T12:54:46Z` — wave 3. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: R10c wave 3 complete: provider-matrix regression coverage and scope invariants pass; no translator, SDK, config, or frontend drift detected..
- `2026-08-21T13:02:44Z` — wave 3. run: `01M0GT77XCHB5BVSQMP34790VB`. summary: R10c durable gate synchronized: check 01M0J6J3SYKAX3P1A1YXYKHS94 recorded APPROVE_WITH_REQUESTS; aggregate baseline failures and Vertex fixture gap are explicit; phase is checked and ready for owned-path commit without push..

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- `2026-08-20T14:24:25Z` — Treat the internal/util no-copy invariant failures from stale sibling worktrees as a pre-existing baseline and do not modify the reviewed-write list or worktree state for R10a (phase: `r10a-custom-headers`), task: wave-1-gate. rationale: The focused executor packages pass; internal/util fails only because its invariant scan includes unrelated .claude/worktrees/agent-* paths. Changing those paths would violate minimal scope and the R10a ownership boundary..
- `2026-08-20T14:28:46Z` — Sequence R10c Gemini leading-turn edits in gemini_executor.go and gemini_vertex_executor.go after R10a completes because both phases modify the same request-path regions (phase: `r10a-custom-headers`), task: wave-2-ownership. rationale: The R10c inventory found definite overlap where R10a threads Options.Headers and R10c inserts turn normalization. Keeping R10a as the first owner prevents concurrent hunk conflicts while preserving parallelism for disjoint future surfaces..
- `2026-08-20T14:47:41Z` — Include the local Codex alpha-search path in R10a header threading (phase: `r10a-custom-headers`), task: wave-2-implementation. rationale: Although the upstream delta does not contain this legacy method, the local executor already passes Options into executeAlphaSearch and applies the shared custom-header helper. Threading opts.Headers closes a real local request path without changing any public interface or adding provider scope..
- `2026-08-20T14:47:50Z` — Leave Claude PrepareRequest and other static-only executor helper callers unchanged for dynamic `$VAR` resolution (phase: `r10a-custom-headers`), task: wave-2-implementation. rationale: Those paths have no request-scoped Options.Headers or public interface slot; changing sdk RequestPreparer or static-only callers would expand scope and violate SDK compatibility. All live request paths that carry Options.Headers are threaded..

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- `2026-08-20T14:55:18Z` — check. verdict: `APPROVE_WITH_REQUESTS`. check: `01M0FTPPNFWEAD96M2RBDZX51X`. run: `01M0FRM5Y916HYE7EFXWFMR14D`. phase: `r10a-custom-headers`. judge: `same-session` (gpt-5.6-luna).
  - `go test ./internal/util -run TestApplyCustomHeadersFromAttrs -count=1` → Validation entry 2026-08-20T21:55:30+07:00: pass
  - `go test ./internal/runtime/executor/... -run TestCustomMagicHeaders -count=1` → Validation entry 2026-08-20T21:55:30+07:00: pass
  - `go test ./internal/runtime/executor/... -count=1` → Validation entry 2026-08-20T21:55:30+07:00: pass
  - `git diff --check` → Validation entry 2026-08-20T21:55:30+07:00: pass
  - `git diff -- config.example.yaml sdk web internal/managementasset/static` → Validation entry 2026-08-20T21:55:30+07:00: pass
  - `go test ./internal/util/... ./internal/runtime/executor/... -count=1` → FAIL: `internal/util` TestInPlaceByteWritesAreReviewed reports only stale `.claude/worktrees/agent-*` writes; executor packages pass. Accepted pre-existing baseline; no R10a-owned path is implicated.
  - proof_gaps: same-session gate; no independent manual review. The full utility package remains red on the stale sibling-worktree invariant, so the approval proofs intentionally use focused utility coverage and passing executor/package checks.
- `2026-08-20T23:57:18Z` — check. verdict: `APPROVE_WITH_REQUESTS`. check: `01M0GSQ4NJ9Y3A8A54G5FTA9EP`. run: `01M0FTXH2WRSCTAF4RGFRBNM6Q`. phase: `r10b-native-parity`. judge: `same-session` (gpt-5.6-luna).
  - `go test ./internal/translator/... -count=1` → R10b wave-1 translator matrix: pass
  - `go test ./internal/runtime/executor/... -count=1` → R10b wave-2 executor matrix: pass
  - `go test ./internal/api/middleware/... -count=1` → R10b cancellation logging matrix: pass
  - `go test ./internal/util -run TestCleanJSONSchemaStripsEncryptedMetadata -count=1` → R10b schema sanitization focused proof: pass
  - `go test ./internal/runtime/executor/... -run TestCustomMagicHeaders -count=1` → R10b custom-header focused proof: pass
  - `git diff --check` → R10b unstaged whitespace check: pass
  - `git diff --cached --check` → R10b staged whitespace check: pass
  - proof_gaps: same-session gate; no independent manual review. Full internal/util remains red only on stale `.claude/worktrees/agent-*` no-copy findings, and this branch has no internal/clienterror package or upstream StreamUsageBuffer equivalent.
- `2026-08-21T13:01:00Z` — check. verdict: `APPROVE_WITH_REQUESTS`. check: `01M0J6J3SYKAX3P1A1YXYKHS94`. run: `01M0GT77XCHB5BVSQMP34790VB`. phase: `r10c-gemini-leading-user-turn`. judge: `same-session` (gpt-5.6-luna).
  - `go test ./internal/runtime/executor/... -run 'Gemini|Vertex|AIStudio|Antigravity' -count=1` → 2026-08-21 R10c full check: provider-matrix regression tests passed
  - `go test ./internal/runtime/executor/... -count=1` → 2026-08-21 R10c full check: all executor package tests passed
  - `make build` → 2026-08-21 R10c full check: web and Go build passed
  - `git diff --check` → 2026-08-21 R10c full check: no whitespace errors
  - `go test ./...` → FAIL: `cmd/server/TestUpdateCommandStages` reports release `v9.9.9` has no `llmhub-darwin-arm64` asset, and `internal/util/TestInPlaceByteWritesAreReviewed` reports only stale `.claude/worktrees/agent-*` writes; all R10c-owned executor packages pass. Accepted as pre-existing baseline; no R10c-owned path is implicated.
  - `git diff -- config.example.yaml sdk web internal/managementasset/static` → pass: forbidden surfaces are unchanged.
  - `rg` credential-pattern scan over the final R10c scope → pass: no high-confidence credential patterns found.
  - Full Security/Performance/Architecture/Code Quality review → pass: normalization is bounded to approved executor boundaries, preserves Antigravity Claude behavior, avoids valid-path copies, and introduces no new secret, logging, retry, concurrency, SDK, translator, config, or frontend surface.
  - proof_gaps: same-session review; Vertex normal/stream/count-token request bodies were verified by complete call-site inspection and package tests but lack dedicated HTTP capture fixtures. Aggregate repository tests remain red only on the unrelated baseline failures recorded above.

## Current State and Next Action
- active_phase: r10c-gemini-leading-user-turn
- lifecycle_status: checked
- latest_run_id: 01M0GT77XCHB5BVSQMP34790VB
- latest_trace_ids: [01M0GTG7ZVKER9BG22PQ8Q00HG, 01M0GTG805E5HJ0J0815W6EC0M]
- latest_check_id: 01M0J6J3SYKAX3P1A1YXYKHS94
- latest_handoff_id: none
- blockers: none
- open_items: [commit the verified R10c implementation, helper, tests, and plan update without pushing; retain the unrelated aggregate baseline failures as explicit context]
- exact_next_action: run the pre-commit security/scope review, stage only approved R10c paths, and create the commit without pushing
