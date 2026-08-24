---
id: 01M0SZEF7HGRXZT8NC0DXKTB68
type: plan
intake_id: 01M0SZG9KJ2PW4REZWH3J8CMT8
lane: high-risk
status: active
created: 2026-08-23
updated: 2026-08-23
---

# Plan: CLIProxyAPI v7.2.140 targeted parity

## Outcome
- result: llmhub ports the user-approved v7.2.139..v7.2.140 parity deltas as independently verifiable capability slices — Gemini schema normalization merge, Claude stable request-scoped user_id via a shared helper, Gemini←Claude thinking-signature normalization, xAI orphaned tool_choice and grok-4.6+ image_generation fixes, management key delete/patch base-URL validation, and the antigravity fallback client version bump.
- success_signals:
  - Each accepted slice lands behind existing llmhub interfaces with focused regression tests citing its upstream commit(s).
  - No config.example.yaml feature fields, no pluginhost/interactions subsystems invented.
  - Existing credential IDs remain content-stable (StableIDGenerator) — no mapping orphans.
  - `go test ./...` scoped suites, `make build`, and `git diff --check` run at gates; known baseline failures recorded honestly.
  - The v7.2.141+ delta and deferred slices (grok keepalive, credential metadata keys) stay explicit follow-ups.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint v7.2.140 at `a7e3596b7e35`; `scope_policy` strategy `targeted-semantic-ports` recorded 2026-08-23 is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.2.139..v7.2.140.json` — structural gap matrix (55 paths).
  - `docs/upstream/cliproxyapi-ledger-v7.2.139..v7.2.140.md` — 13 non-merge commits in v7.2.140.
  - `docs/plans/completed/cliproxyapi-v7.2.139-parity.md` — completed prior cycle; signature/synthesizer surfaces it owns must not regress.
  - Repo invariants — Postgres-authoritative config, SDK compatibility, embedded management panel, monolithic conductor (NG5 lineage).
- requirements:
  - R1 [accepted]: Merge upstream malformed-schema-node normalization before schema cleanup into the locally diverged `internal/util/gemini_schema.go`. | source: `a7e3596b7e35`.
  - R2 [accepted]: Introduce shared `internal/translator/common` Claude user_id derivation and replace the three duplicated request-converter copies so `metadata.user_id` is stable per account/session. | source: `ab8f00dbd91f`.
  - R3 [accepted]: Normalize Claude thinking signatures in Gemini response conversion (plus its request-path touch) matching upstream behavior. | source: `b3f72cef6565`.
  - R4 [accepted]: xAI drops orphaned `tool_choice` after compact strips tools. | source: `87fb01b23788`.
  - R5 [accepted]: xAI keeps `image_generation` on grok-4.6+ conversation requests. | source: `dfdf183fcfb6`.
  - R6 [accepted]: Management key deletion and patching validate base URLs before applying. | source: `ebda7509114d`.
  - R7 [accepted]: Antigravity fallback client version raised to 2.9.1 with refreshed fallback test. | source: `a834917e871d`.
  - R8 [accepted] final gate: re-resolve latest upstream release, refresh checkpoint/gap/ledger trio, pin newer delta as explicit follow-up — never silent scope growth. | source: upstream skill final-gate contract.

## Non-goals
- NG1: Gemini interactions API paths (`d5b57a2d8ac0`, `71c3c144a078`) excluded — no local interactions subsystem.
- NG2: Grok build keepalive client (`4d68ca8a6349`) deferred — `internal/client/grokbuild` does not exist locally.
- NG3: Credential metadata-key normalization (`e04d620cc1a5`) deferred — spans pluginhost/stores/filestore needing dedicated assessment.
- NG4: No new `config.example.yaml` feature fields, no `web/**/*_test.go`, no wholesale upstream merges, no conductor file split.
- NG5: Test-only upstream commits land absorbed into their sibling slices, never standalone.

## Approach and Risks
- approach: semantic ports of six approved slices in two phases — translator cluster first (schema merge, user_id dedupe, thinking-sig normalization), then runtime/auth cluster (xAI fixes, management validation, version bump). Every task cites its upstream commit; every slice lands with focused regression tests.
- constraints:
  - DB-backed settings only; no new config.example.yaml fields (NG4).
  - Preserve StableIDGenerator content-addressed identity (no mapping orphans).
  - Monolithic conductor untouched by this initiative.
- rejected_alternatives:
  - Wholesale upstream merge — diverged local hunks across r8-r11 cycles make it unsafe.
  - Porting interactions/pluginhost subsystems to enable small fixes — cost inverts value.
- risks:
  - R1 gemini_schema.go divergence may hide behavioral coupling → mitigation: hunk-level merge + full util suite.
  - R2 replacing three user_id copies could shift emitted IDs → mitigation: assert byte-stable output for identical account/session inputs before/after.
  - R3 signature normalization interacts with r11b sanitize layer → mitigation: run internal/signature + executor suites together.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: r12a-translator-userid-sigs
    story_id: 01M0SZH1Z5RJ1D0KVB9YTD8AYZ
    status: planned
    goal: Port the Gemini schema normalization merge, shared stable Claude user_id helper, and Gemini-from-Claude thinking-signature normalization (R1-R3).
    depends_on: none
    allowed_surfaces:
      - `internal/util/gemini_schema.go` merge hunk plus focused tests
      - `internal/translator/common/` new claude_user_id helper and tests
      - `internal/translator/claude/` three request-converter call sites and their conversion tests
      - `internal/translator/gemini/claude/gemini_claude_response.go` and request touch plus tests
    avoided_surfaces:
      - interactions paths (`d5b57a2d8ac0`, `71c3c144a078`), pluginhost/stores, config.example.yaml, sdk public types, web/
    waves:
      - wave: 1 — stable Claude user_id (R2)
        tasks:
          - task: Add internal/translator/common Claude user_id derivation per ab8f00dbd91f with stability/regression tests; expected output is helper test matrix.
          - task: Replace the three duplicated user_%%s_account_%%s_session_%%s copies (claude_openai_request.go, claude_gemini_request.go, claude_openai-responses_request.go) asserting byte-stable output for identical account/session inputs before and after.
        checks:
          - check: go test ./internal/translator/common/... ./internal/translator/claude/... -count=1
          - check: rg -n "user_%s_account_%s_session_%s" internal/translator | only common helper remains
      - wave: 2 — Gemini schema normalize merge (R1)
        tasks:
          - task: Merge upstream malformed-schema-node normalization hunks into local gemini_schema.go per a7e3596b7e35 with upstream-derived focused tests adapted to local divergence.
        checks:
          - check: go test ./internal/util/ -run Schema -count=1 && go test ./internal/util/... -count=1
        stop_conditions:
          - Local gemini_schema.go diverges so hunks cannot map without inventing behavior.
        escalation: Record NEEDS_CONTEXT with the diverged symbol list.
      - wave: 3 — thinking-signature normalization (R3)
        tasks:
          - task: Port Claude thinking-signature normalization in gemini_claude_response.go plus its request-path touch per b3f72cef6565, absorbing applicable upstream test cases; run signature + executor suites together for interaction safety.
        checks:
          - check: go test ./internal/translator/gemini/claude/... ./internal/signature/... -count=1
          - check: go build ./... && git diff --check
  - phase_slug: r12b-runtime-auth-fixes
    story_id: 01M0SZH1DW73THPXTMX8JWPFC
    status: planned
    goal: Port xAI orphaned tool_choice and grok-4.6+ image_generation fixes, management key delete/patch base-URL validation, and the antigravity fallback client bump (R4-R7).
    depends_on: none
    allowed_surfaces:
      - `internal/runtime/executor/xai_executor.go` compact/tool_choice and image-generation paths plus tests
      - `internal/api/handlers/management/config_lists.go` delete/patch validation plus tests
      - `internal/misc/antigravity_version.go` and its test
    avoided_surfaces:
      - grokbuild client subsystem, pluginhost/stores/filestore, interactions paths, translator files owned by r12a, web/, config.example.yaml
    waves:
      - wave: 1 — xAI fixes (R4, R5)
        tasks:
          - task: Drop orphaned tool_choice after compact strips tools per 87fb01b23788; expected output is regression test.
          - task: Keep image_generation on grok-4.6+ conversation requests per dfdf183fcfb6; expected output is gating matrix test.
        checks:
          - check: go test ./internal/runtime/executor/ -run 'XAI' -count=1
          - check: go test ./internal/runtime/executor/ -count=1
      - wave: 2 — auth validation + version bump (R6, R7)
        tasks:
          - task: Validate base URLs in management key deletion and patching per ebda7509114d; expected output is delete/patch validation tests.
          - task: Raise antigravity fallback client version per a834917e871d refreshing its fallback test.
        checks:
          - check: go test ./internal/api/handlers/management/... ./internal/misc/... -count=1
          - check: go build ./... && git diff --check && git diff -- config.example.yaml
        stop_conditions:
          - Local management handler shapes cannot express upstream validation without new public API surface.
        escalation: Record NEEDS_CONTEXT naming the missing seam.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- `2026-08-24T13:31:47Z` — wave 0. summary: Initiative v7.2.140 targeted parity locked: intake 01M0SZG9KJ2PW4REZWH3J8CMT8 (spec-slice/high-risk) linked to plan 01M0SZEF7HGRXZT8NC0DXKTB68; scope authority = checkpoint scope_policy targeted-semantic-ports (6 include, interactions+grok+metadata-keys excluded/deferred); next action to-plan.
- `2026-08-24T13:33:31Z` — wave 0. summary: to-plan complete for v7.2.140 targeted parity: r12a-translator-userid-sigs (01M0SZH1Z5RJ1D0KVB9YTD8AYZ, waves user_id/schema/thinking-sigs) and r12b-runtime-auth-fixes (01M0SZH1DW73THPXTMX8JWPFC, waves xAI/auth+version) planned with tasks citing upstream commits; next action work full r12a wave 1.

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
- latest_trace_ids: []
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- open_items: [execute r12a then r12b; R8 final gate after both]
- exact_next_action: start `work full` on phase r12a-translator-userid-sigs wave 1 (stable Claude user_id helper)
