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
- approach: not-planned
- constraints:
  - none
- risks:
  - none

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: not-planned
- phases: none

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
- lifecycle_status: not-planned
- latest_run_id: none
- latest_trace_ids: []
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- open_items: [to-plan must define stable phases, stories, waves, tasks, and checks]
- exact_next_action: to-plan
