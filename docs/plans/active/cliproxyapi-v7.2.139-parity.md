---
id: 01M0M64B2TYGG69CS1SSMVYF57
type: plan
intake_id: 01M0MPP37ZXX9HY4VVVHXWS32G
lane: high-risk
status: active
created: 2026-08-22
updated: 2026-08-22
---

# Plan: CLIProxyAPI v7.2.139 targeted parity

## Outcome
- result: llmhub ports the user-approved v7.2.137..v7.2.139 parity deltas as independently verifiable capability slices — translator correctness (six mechanical fixes), xAI incomplete-terminal handling, Gemini thought-signature sanitization, Gemini 3.7 Flash registry entries, warn-level auth diagnostics, global OAuth request-scoped error rules, Codex stream bootstrap buffering with overload failover, base_URL-only credentials with stale-header skip, and the Home credential retry-round contract.
- success_signals:
  - Each accepted slice lands behind existing llmhub interfaces with focused regression tests citing its upstream commit(s).
  - Config-dependent features (OAuth error rules, Codex buffering flag) are stored as DB-backed settings; no new `config.example.yaml` feature fields appear.
  - The base_URL-credential slice changes the auth-ID scheme without corrupting existing index/cooldown mappings (migration or explicit invalidation proven by test).
  - `go test ./...`, `make build`, and `git diff --check` run at the aggregate gate; known baseline failures are recorded honestly.
  - No deferred or excluded delta (namespace Responses tools, plugin-executor parsing, Fable cooldown fix) is silently implemented.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint v7.2.139 at `0a14eb70ce19`; `scope_policy` strategy `targeted-semantic-ports` recorded 2026-08-22 is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.2.137..v7.2.139.json` — structural gap matrix (98 paths: 16 upstream-add-absent, 44 diverged-absent, 38 semantic-review).
  - `docs/upstream/cliproxyapi-ledger-v7.2.137..v7.2.139.md` — 18 non-merge commits across v7.2.138/v7.2.139.
  - `docs/plans/completed/cliproxyapi-v7.2.137-follow-up.md` — completed prior cycle and its final-gate dispositions.
  - `CLAUDE.md` / repo invariants — Postgres-authoritative runtime state, SDK compatibility, embedded management panel ownership.
- requirements:
  - R1 [accepted]: Gemini→OpenAI Responses streaming emits empty `annotations` and `logprobs` arrays in the `response.output_item.done` message template at the local site lacking them (`gemini_openai-responses_response.go:197`). | source: `b1c000590b47`.
  - R2 [accepted]: Claude→OpenAI non-stream conversion emits `choices.0.message.reasoning_content`, replacing the legacy `reasoning` key, after a grep confirms no local consumer reads the legacy key. | source: `aa5dccc23688`.
  - R3 [accepted]: OpenAI Responses non-stream conversion falls back to `choices.0.message.reasoning` when `reasoning_content` is missing. | source: `8eb3ac2e036b`.
  - R4 [accepted]: Antigravity request conversion maps `max_completion_tokens` to `maxOutputTokens` while preserving existing `max_tokens` precedence. | source: `68e96c27165e`.
  - R5 [accepted]: Claude←Gemini and Codex←Gemini conversions generate deterministic sequential tool-call IDs (`toolu_gemini_%016d`, `call_gemini_%016d`) instead of random IDs. | source: `5b232e3e981b`.
  - R6 [accepted]: Gemini→Claude non-stream conversion preserves thought signatures into thinking blocks, including signature-only part skip, matching the streaming path's behavior. | source: `3db591eecd6a`.
  - R7 [accepted]: The xAI executor treats `response.incomplete` as terminal success in both execute and stream switches, without feeding incomplete events into any replay cache. | source: `9d6b5cdd163b`.
  - R8 [accepted]: A Gemini request thought-signature sanitizer exists under `internal/signature/` and is wired into Gemini and Vertex executors across their execute/stream/count-tokens call sites. | source: `4053c026e79c`.
  - R9 [accepted]: Model registry registers `gemini-3.7-flash` in the gemini, vertex, and gemini-cli sections via manual JSON merge of the locally diverged `models.json`. | source: `85e7add6adf3`.
  - R10 [accepted]: Auth cooldown entry and upstream execution failure paths emit warn-level diagnostics that identify the affected credential. | source: `48749717645e`.
  - R11 [accepted]: OAuth-kind request-scoped error rules work end-to-end: DB-backed setting plus sanitize hook, an OAuth branch beside the per-credential engine in `conductor_request_scoped_errors.go:52`, and management CRUD handlers on local routes. | source: `9dc51b1f8777`.
  - R12 [accepted]: Codex opt-in stream bootstrap buffering with overload failover works on both HTTP and WebSocket paths; the opt-in flag is a DB-backed setting, not a YAML field. | source: `4b9d404fb04f`.
  - R13 [accepted]: base_URL-only credentials are supported in two layers — synthesizer/config dedup and synthesis first, then executor stale-auth-header else-Del branches — with the auth-ID scheme change proven not to orphan existing index/cooldown mappings. | source: `92d96e0b729d`.
  - R14 [accepted]: The credential retry-round contract works inside the monolithic conductor: dispatch carries round exclusion metadata, rounds are counted, exhausted rounds are marked, and per-round admission filtering excludes failed credentials. | source: `601ca43090f7`, `0a14eb70ce19`.
  - R15 [accepted]: Every port preserves Postgres-authoritative config, public SDK compatibility, llmhub management panel layout, and uses semantic ports rather than wholesale merges. | source: repo invariants.
  - R16 [accepted] final gate: after all slices land, re-resolve the latest stable CLIProxyAPI release, refresh the checkpoint/gap/ledger trio, and pin any newer delta as an explicit follow-up plan — never silent scope growth. | source: upstream skill final-gate contract.

## Non-goals
- NG1: Namespace-aware OpenAI Responses tool resolution (`556328c12253`) is deferred; trigger to pull it back in: custom tools or `additional_tools` namespaces needed on the Gemini Responses path.
- NG2: Plugin-executor usage parsing (`1d5b7612c6ba`) is excluded — the prerequisite plugin-executor host subsystem does not exist locally.
- NG3: Fable-only `7d_oi` cooldown fix (`42d8e746e540`) is excluded as superseded locally: `classify.go:38` text rules plus the `quotaBackoffMax = 30m` clamp already prevent week-long cooldowns; bundle with any future port of upstream's Claude rate-limit header parser.
- NG4: No new `config.example.yaml` feature fields, no new `web/**/*_test.go` files, no pushes, no wholesale upstream source merges.
- NG5: Splitting the monolithic conductor into upstream's `conductor_home/execution/selection` files is not in scope — only the retry-round behavior is ported inside the existing structure.

## Approach and Risks
- approach: semantic ports of the nine approved slices in four independently gated phases — translator fixes first (lowest risk, shared package), runtime hardening second, DB-backed managed settings third, and credential lifecycle last where conductor surfaces overlap. Every task cites its upstream commit; every port lands behind existing llmhub interfaces with focused regression tests.
- constraints:
  - Config-dependent behavior (R11 OAuth error rules, R12 Codex buffering flag) is stored as DB-backed settings; no new `config.example.yaml` feature fields (NG4).
  - The monolithic conductor is preserved (NG5); only retry-round behavior is ported inside existing structure.
  - Postgres-authoritative config, public SDK compatibility, embedded management panel ownership (R15).
  - No `web/**/*_test.go` files, no pushes from phase work, no wholesale upstream merges.
- rejected_alternatives:
  - Wholesale upstream merge — inverts repo invariants: plugin host, config storage, and panel layout diverge locally.
  - Splitting the conductor into upstream `conductor_home/execution/selection` files — excluded by NG5.
  - Cherry-picking upstream patches — local hunks have diverged across R8–R10 cycles; each fix must be re-derived at its local site.
- risks:
  - R13 auth-ID scheme change orphans existing index/cooldown mappings → mitigation: mapping-preservation test proves migration or explicit invalidation before landing.
  - R2 legacy `reasoning` key replacement breaks an unnoticed consumer → mitigation: grep proof of zero local consumers is a wave gate before editing.
  - Adjacent conductor regions between r11c (OAuth error-rule branch) and r11d (dispatch/admission edits) → mitigation: r11d depends_on r11c.
  - Diverged translator hunks mislead a direct patch application → recovery: revert only the incomplete slice's edits per wave, record `NEEDS_CONTEXT`, never widen scope.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: r11a-translator-fixes
    story_id: 01M0MPWHT5Q93NYTAKB9SA6RYK
    status: planned
    goal: Port the six approved mechanical translator correctness fixes R1-R6 from upstream v7.2.138 with focused regression tests.
    depends_on: none
    allowed_surfaces:
      - `internal/translator/` conversion paths cited by R1-R6 and their focused tests
    avoided_surfaces:
      - `config.example.yaml`, `internal/config/`, `sdk/`, `web/`, `internal/managementasset/static/`, `internal/runtime/`
      - deferred/rejected deltas: namespace Responses tools (556328c12253), plugin-executor parsing (1d5b7612c6ba), Fable cooldown (42d8e746e540)
    waves:
      - wave: 1 — reasoning-key correctness (R2, R3)
        tasks:
          - task: Prove no local consumer reads the legacy `reasoning` key on the Claude→OpenAI non-stream path (`rg` over internal/ and sdk/), then emit `choices.0.message.reasoning_content` replacing the legacy key per `aa5dccc23688`; expected output is the grep transcript plus updated conversion test.
          - task: Fall back to `choices.0.message.reasoning` when `reasoning_content` is missing in OpenAI Responses non-stream conversion per `8eb3ac2e036b`; expected output is fallback regression test.
        checks:
          - check: `rg -n "reasoning_content|\.reasoning\b" internal/translator internal/runtime sdk | review transcript recorded in Progress`
          - check: `go test ./internal/translator/... -count=1`
        stop_conditions:
          - A local consumer reads the legacy `reasoning` key on the affected path.
          - Upstream semantics require changing a public SDK type.
        escalation: Record `NEEDS_CONTEXT` with the consuming symbol; do not keep both keys without user decision.
      - wave: 2 — request-field correctness (R4, R1)
        tasks:
          - task: Map `max_completion_tokens` to Antigravity `maxOutputTokens` preserving existing `max_tokens` precedence per `68e96c27165e`; expected output is precedence matrix test.
          - task: Emit empty `annotations` and `logprobs` arrays in the Gemini→OpenAI Responses streaming `response.output_item.done` message template at the local site lacking them (`gemini_openai-responses_response.go:197`) per `b1c000590b47`.
        checks:
          - check: `go test ./internal/translator/... -run 'Antigravity|Annotations|Responses' -count=1`
          - check: `go test ./internal/translator/... -count=1`
        stop_conditions:
          - Local template structure has no corresponding output_item.done emission point.
        escalation: Record `NEEDS_CONTEXT` with the diverged hunk; do not restructure templates.
      - wave: 3 — Gemini tool-call identity (R5, R6)
        tasks:
          - task: Generate deterministic sequential tool-call IDs (`toolu_gemini_%016d`, `call_gemini_%016d`) in Claude←Gemini and Codex←Gemini conversions per `5b232e3e981b`; expected output is two-conversion determinism test.
          - task: Preserve Gemini thought signatures into thinking blocks in non-stream Claude conversion including signature-only part skip, matching the streaming path per `3db591eecd6a`.
        checks:
          - check: `go test ./internal/translator/... -count=1`
          - check: `git diff --check && git diff -- config.example.yaml sdk web internal/managementasset/static`
        stop_conditions:
          - Deterministic IDs collide with locally cached tool-call ID formats elsewhere.
        escalation: Mark blocked with the colliding format evidence; do not change ID scheme unilaterally.
  - phase_slug: r11b-runtime-hardening
    story_id: 01M0MPWHTF122GTAAM3DZ5KMHG
    status: planned
    goal: Port xAI incomplete-terminal handling, the Gemini thought-signature sanitizer, and Gemini 3.7 Flash registry entries (R7, R8, R9).
    depends_on: none
    allowed_surfaces:
      - `internal/runtime/executor/` xAI execute/stream switches and Gemini/Vertex executor call sites plus tests
      - `internal/signature/` new sanitizer package and focused tests
      - `internal/registry/` models.json gemini, vertex, and gemini-cli sections
    avoided_surfaces:
      - `internal/translator/`, `sdk/`, `config.example.yaml`, `web/`, `internal/managementasset/static/`
    waves:
      - wave: 1 — xAI terminal state (R7)
        tasks:
          - task: Treat `response.incomplete` as terminal success in both execute and stream switches per `9d6b5cdd163b`, feeding no incomplete event into any replay cache; expected output is incomplete-terminal regression test plus cache-absence assertion.
        checks:
          - check: `go test ./internal/runtime/executor/... -run 'XAI|Xai' -count=1`
          - check: `go test ./internal/runtime/executor/... -count=1`
        stop_conditions:
          - Local xAI stream switch lacks a terminal-state seam matching upstream semantics.
        escalation: Record `NEEDS_CONTEXT` with the local switch excerpt.
      - wave: 2 — thought-signature sanitizer (R8)
        tasks:
          - task: Add a request thought-signature sanitizer under `internal/signature/` per `4053c026e79c` with unit coverage for malformed/missing/oversized signatures; expected output is sanitizer test matrix.
          - task: Wire it into Gemini and Vertex executors across execute/stream/count-tokens call sites after payload/replay rewrites and before upstream request construction (same boundary as R10c normalization); expected output is wiring regression tests for both executors.
        checks:
          - check: `go test ./internal/signature/... ./internal/runtime/executor/... -count=1`
          - check: `go test ./internal/runtime/executor/... -run 'Signature|Gemini|Vertex' -count=1`
        stop_conditions:
          - Wiring would touch Antigravity or Claude paths beyond the approved slice.
        escalation: Stop at the boundary map; record which call sites cannot be reached within allowed surfaces.
      - wave: 3 — model registry (R9)
        tasks:
          - task: Register `gemini-3.7-flash` in the gemini, vertex, and gemini-cli sections via manual JSON merge of the locally diverged models.json per `85e7add6adf3`; expected output is registry lookup test across all three sections.
        checks:
          - check: `go test ./internal/registry/... -count=1`
          - check: `go test ./internal/runtime/executor/... -run 'Model' -count=1 || true # only if registry feeds executor selection tests`
        stop_conditions:
          - Local models.json schema diverges so that manual merge cannot express the entries.
        escalation: Record `NEEDS_CONTEXT` with the schema diff; do not regenerate models.json from upstream wholesale.
  - phase_slug: r11c-managed-settings
    story_id: 01M0MPWQFPVZB8VCSKH6BS1DMX
    status: planned
    goal: Port OAuth request-scoped error rules and Codex stream bootstrap buffering as DB-backed settings with management CRUD (R11, R12).
    depends_on: none
    execution_gate: Land before r11d because its OAuth branch sits beside the credential engine regions r11d later edits.
    allowed_surfaces:
      - DB-backed settings store and management API handlers on local routes plus tests
      - `conductor_request_scoped_errors.go` OAuth branch beside the per-credential engine
      - Codex executor HTTP and WebSocket buffering/failover paths plus tests
    avoided_surfaces:
      - `config.example.yaml` feature fields (NG4), `web/**/*_test.go`, wholesale upstream merges, translator files
    waves:
      - wave: 1 — OAuth request-scoped error rules (R11)
        tasks:
          - task: Add the DB-backed setting plus sanitize hook for OAuth-kind request-scoped error rules per `9dc51b1f8777`; expected output is sanitize-hook unit test and settings round-trip test.
          - task: Branch on OAuth kind beside the per-credential engine at `conductor_request_scoped_errors.go:52` and expose management CRUD handlers on local routes; expected output is end-to-end handler test proving rule application.
        checks:
          - check: `go test ./internal/api/... ./sdk/cliproxy/... -count=1`
          - check: `git diff -- config.example.yaml` → empty
        stop_conditions:
          - Required route or storage surface falls outside allowed list.
          - Sanitize hook needs a public SDK signature change.
        escalation: Record `NEEDS_CONTEXT` naming the missing surface; do not add YAML fields as fallback.
      - wave: 2 — Codex bootstrap buffering (R12)
        tasks:
          - task: Implement opt-in stream bootstrap buffering with overload failover on the HTTP path per `4b9d404fb04f`, flag stored as DB-backed setting; expected output is buffered-start and failover-on-overload tests.
          - task: Apply identical semantics on the WebSocket Codex path without changing public SDK contracts; expected output is WS-path regression test.
        checks:
          - check: `go test ./internal/runtime/executor/... -run 'Codex|Buffer|Overload|WebSocket' -count=1`
          - check: `go test ./internal/runtime/executor/... -count=1`
        stop_conditions:
          - The WS path lacks a seam to buffer without altering the SDK interface.
        escalation: Ship HTTP-only behind the opt-in flag, mark WS follow-up explicitly in Progress; do not force the SDK change.
  - phase_slug: r11d-credential-lifecycle
    story_id: 01M0MPWQFYTQKRQ0WE7DZE2BV0
    status: planned
    goal: Port warn-level auth diagnostics, base_URL-only credentials, and the credential retry-round contract inside the monolithic conductor (R10, R13, R14).
    depends_on: r11c-managed-settings
    execution_gate: Begin only after r11c lands; both phases edit adjacent conductor regions.
    allowed_surfaces:
      - `sdk/cliproxy/` cooldown-entry and upstream-failure diagnostics plus tests
      - credential synthesizer/config dedup layers and executor stale-auth-header branches plus tests
      - conductor dispatch/round-count/exhausted-marking/admission-filter logic plus tests
    avoided_surfaces:
      - Conductor file split into upstream `conductor_home/execution/selection` (NG5), translator, web, config YAML fields
    waves:
      - wave: 1 — diagnostics (R10)
        tasks:
          - task: Emit warn-level diagnostics identifying the affected credential on auth cooldown entry and upstream execution failure paths per `48749717645e`; expected output is log-capture unit test naming the credential identifier without leaking secrets.
        checks:
          - check: `go test ./sdk/cliproxy/... -run 'Cooldown|Diagnostic|Warn' -count=1`
          - check: `rg -n "api[_-]?key|secret|token" $(git diff --name-only -- sdk/)` → reviewed transcript in Progress
        stop_conditions:
          - Diagnostic content can only identify credentials by printing secret material.
        escalation: Propose hashed/truncated identifiers in NEEDS_CONTEXT; never ship raw keys.
      - wave: 2 — base_URL-only credentials (R13)
        tasks:
          - task: Support base_URL-only credentials in the synthesis/config dedup layer per `92d96e0b729d`; expected output is synthesis-first dedup unit test.
          - task: Change the auth-ID scheme with a mapping-preservation test proving existing index/cooldown mappings survive via migration or explicit invalidation, then add executor stale-auth-header else-Del branches; expected output is both layer test suites green.
        checks:
          - check: `go test ./sdk/cliproxy/... ./internal/runtime/executor/... -run 'BaseURL|AuthID|StaleHeader' -count=1`
          - check: `go test ./sdk/cliproxy/... ./internal/runtime/executor/... -count=1`
        stop_conditions:
          - Existing persisted mappings cannot be migrated or invalidated without data loss.
        escalation: Halt before landing the ID change; present mapping-diff evidence for user decision.
      - wave: 3 — retry-round contract (R14) + final gate (R16)
        tasks:
          - task: Carry round-exclusion metadata through dispatch, count rounds, mark exhausted rounds, and apply per-round admission filtering that excludes failed credentials per `601ca43090f7` + `0a14eb70ce19`, all inside the monolithic conductor; expected output is round-count/exhaust/filter unit test suite.
          - task: Final gate per R16 — re-resolve the latest stable CLIProxyAPI release, refresh checkpoint/gap/ledger trio, pin any newer delta as an explicit follow-up plan; expected output is refreshed checkpoint JSON and follow-up note, no silent scope growth.
        checks:
          - check: `go test ./sdk/cliproxy/... -run 'RetryRound|Round|Admission' -count=1`
          - check: `go test ./sdk/cliproxy/... ./internal/conductor/... -count=1 || go test ./sdk/cliproxy/... -count=1 # adjust to real conductor package at execution time`
          - check: `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi && git status --porcelain docs/upstream/ | wc -m # refreshed checkpoint visible`
        stop_conditions:
          - Round contract requires splitting the conductor files.
          - Final gate finds new upstream releases mid-initiative.
        escalation: Conductor split demand → REQUEST_CHANGES scope; new upstream delta → pin as explicit follow-up plan, never absorb silently.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- `2026-08-22T12:21:51Z` — wave 0. summary: Initiative re-registered on local DB after branch merge 92bce00e: intake 01M0MPP37ZXX9HY4VVVHXWS32G (spec-slice/high-risk) linked to plan 01M0M64B2TYGG69CS1SSMVYF57; plan frontmatter intake_id synced; scope authority remains checkpoint scope_policy targeted-semantic-ports; next action to-plan..

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- none

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- none

## Current State and Next Action
- active_phase: none
- lifecycle_status: planned
- latest_run_id: none
- latest_trace_ids: [01M0MPQ5E0Y36TVPX6RNB7JXCF]
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- open_items: [execute four planned phases r11a-r11d]
- exact_next_action: start `work full` on phase r11a-translator-fixes wave 1 (reasoning-key correctness, R2+R3)
