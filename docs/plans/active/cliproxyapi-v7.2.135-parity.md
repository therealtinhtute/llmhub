---
id: 01M079W4067Y7FD4GJEY7M42VR
type: plan
intake_id: 01M079W9YWHHJ9Q7A2QA2Q8KKY
lane: high-risk
status: active
created: 2026-08-17
updated: 2026-08-17
---

# Plan: CLIProxyAPI v7.2.135 parity

## Outcome
- result: llmhub reaches functional parity with CLIProxyAPI v7.2.135 across the nine approved capability slices — Claude OAuth wire, Codex multi-agent v2, translator correctness, auth/cooldown robustness, registry/model updates, websocket/realtime, perf, config-as-DB-records, plugin schema-v3 — with no regression to llmhub's DB-only config, embedded panel, or SDK boundaries.
- success_signals:
  - `go test ./...` green at every phase end; `make build` succeeds (embedded panel path intact)
  - Claude OAuth flow includes TLS session resumption and BIP-39 MCP tool aliases (absent at baseline, verified 2026-08-17)
  - Codex surface contains `multi_agent` handling (zero hits at baseline) and passes the local codex test suite
  - `gpt-5.6-sol-wm` resolvable from the local registry with 921k context window
  - All new feature settings stored as DB records via management API; zero new `config.example.yaml` fields from the ported slices
  - Final gate: checkpoint re-resolved to latest stable release, ledger refreshed, delta pinned or approved

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint v7.2.135 (856ddd8df746), local baseline `f08bec353156`, `scope_policy` recorded 2026-08-17
  - `docs/upstream/cliproxyapi-gap-v7.2.113..v7.2.135.json` — 448 changed paths, 161 semantic-review
  - `docs/upstream/cliproxyapi-ledger-v7.2.113..v7.2.135.md` — 203 non-merge commits, 22 releases
  - User-approved triage (AskUserQuestion, 2026-08-17): include all nine slices; exclude docs-readme-churn; config adapts behind DB records; plugin schema-v3 include; gpt-5.6-sol-wm include (2026-08-17)
  - `CLAUDE.md` — standing invariants (Postgres authoritative runtime store; provider route semantics; embedded web panel; SDK backward compat; llmhub branding)
- requirements:
  - R1 [accepted]: Claude OAuth wire parity — TLS session resumption, OAuth exchange/identity alignment with Claude Code 2.1.220, Fast error pass-through, cloaked system-prompt/beta policies, BIP-39 MCP tool aliases with malformed-ID recovery, native Haiku helpers, `service_tier`→`speed` mapping — re-expressed behind llmhub's `internal/auth/claude` + `internal/runtime/executor` interfaces. | source: `f3e25ab2bae6` `707934917a74` `fa6bc77f28e0` `f63a925d15f6` `ef89c6a69d0f` `a8bbbea2b9b5` `f0034ca66376` `f6f03e4de99c` `f2d272da817d` `7eefab98b8e5` (v7.2.116–v7.2.134)
  - R2 [accepted]: Codex multi-agent v2 + input sanitization — MAv2 tool definitions at the Responses boundary, `multi_agent_version` v2 flag, `is-compat` model flag rewrites, deterministic collision-resistant ID sanitization (`ctc_`/`ctco_`/sub2api), `max-context-length` overrides, sequential cutoff reasoning summaries, spawn-agent model caching, `response.completed` ID hydration — landed in the local codex executor/translator/client surfaces. | source: `7fe847376667` `4f5ec105b11c` `7fef08ea2bf4` `e5ea945ed93d` `5e25566c240e` `197f52042637` `673bac5fc606` `db143aebac93` `a303fd869b03` `5314b29da963` `98c98d66be1a` `134a66738c35` `8d675e690ddf` `f43aad7637ad` `133047de66bf` `b08fe3b49264` `c30e60a11b49` (v7.2.115–v7.2.132)
  - R3 [accepted]: Translator correctness — idempotent `response.completed` and `function_call_arguments.done` emission, content block/tool input preservation (incl. non-object inputs, base64 PDFs, order), Responses reasoning chain rebuild, `reasoning_tokens` usage detail, open-item finalization on `[DONE]`, truncated/filtered responses marked incomplete, `content_filter` stop-reason mapping, belated `tool_use` synthesis, deterministic tool-call IDs, duplicate tool output dedup — through the local `sdk/translator` registry matrix. | source: `872070259ef6` `0cc24b2e0006` `0879929821cb` `aea71c0070fa` `9b1142399c29` `82d6242098a7` `9b8d97441e86` `4906ead34fa5` `78f0c4079e3e` `fab077a04b4a` `37609fa17993` `8d670b98ffac` `1ecb7df228cf` `37411842e859` `71e87111e9d8` `f1a21a9b95bf` `2e91e99e0339` `94808f1087d2` `aff2095abd3b` `44d5e0bebcbc` `ea37d13a9ece` `934da2379d62` `e9c44ae256c5` `8b54db36ae7a` `0a95fa62a106` (v7.2.115–v7.2.133)
  - R4 [accepted]: Auth/cooldown/selector robustness — credential rotation on unknown upstream failures, no cooldown penalty for client faults (incl. request-scoped 401s and connection-lifecycle disconnects), session affinity across priorities and model variant suffixes, rate-limit status prioritized over error body, unified Anthropic rate-limit parsing, centralized client error status mapping, Home OAuth retry after unauthorized — landed in `sdk/cliproxy/auth` + `internal/clienterror`. | source: `579f5e30fbd6` `c1d69e7b4778` `fe28d582f43b` `8392b180ce37` `1674aaf43b52` `65fc536e6e32` `9169ad56e18b` `00c4377a21b8` `9c8e4a07e638` `d0d77182ee8e` `45ffd115fdf0` `ac82bedfaf5b` `203f5b1a1d84` `b8fbe70b37a6` `4b3cc55cdc93` `1e38a3a544ec` `063f9d341285` `d952cb429706` `a81b9e9cedab` `076ec64c16e0` (v7.2.116–v7.2.134)
  - R5 [accepted]: Registry/model updates — Grok Imagine Video 1.5 GA (incl. preview alias routing), Grok Image 2.0, Gemini 3.7 Flash High, `max_completion_tokens` on model definitions and responses, model modality metadata, `gpt-5.6-sol-wm` registrations (incl. Codex client config and 921k context limits), Kimi K2.7 alias canonicalization — merged into local `internal/registry` definitions and validated JSON. | source: `abaeb55bb20e` `61d9a30d126e` `db35b91e2a46` `7ea9c670ea0d` `046b59ecc519` `323b7276bc5b` `dd214445ef7a` `745fb38dbbe6` `c1ff55fc2f1e` `36936340a33a` `84232747e20e` (v7.2.118–v7.2.135)
  - R6 [accepted]: Websocket/realtime — realtime hangup forwarding, local client-secret support, websocket transcript merge/repair (case-insensitive item metadata, duplicate input semantics, reduced allocations), premature SSE/websocket termination with terminal error emission, `response.failed` stream errors, upstream stream error preservation — landed in `sdk/api` openai websocket/handlers. | source: `bd34ceca0420` `c845ce15c9de` `75d2c4a4b4e2` `baa11ed6dd75` `49b2f891ac32` `522b4de54a8d` `3522e481aa7b` `92f03e68e3f1` `f9bd9def2b6a` (v7.2.124–v7.2.134)
  - R7 [accepted]: Perf — payload-reuse guards, no-copy GJSON parse helper, batched raw-array assembly (`SetRawArrayItems`), reduced request amplification for Codex/Gemini/Antigravity large payloads, connection reuse for Antigravity — landed in `internal/util` + translators behind existing interfaces. | source: `b921b5d03264` `1737596e020a` `ea8882bf26b2` `8b3b304952ed` `0c58c3c83d3c` `c30e60a11b49` `34364fff469e` `b7a441522a90` `99929209840b` `124dab6cc198` `f9bd9def2b6a` `bb7278a1af58` `cf8c27fe90ab` `d5f68856f6da` `c33a33e14a6d` (v7.2.123–v7.2.133)
  - R8 [accepted]: Config features as DB records — per-credential request-retry override, request-scoped proxy override, request-scoped error action handling, cooling management refactor — behavior ported with settings stored as structured database records via the management API; no new `config.example.yaml` fields. | source: `6f2cea948451` `17a479a8db98` `aa10847e1271` `5bffd1514fba` (v7.2.130–v7.2.135), invariant: Postgres authoritative runtime store
  - R9 [accepted]: Plugin schema-v3 + auth delegation — schema-v3 stream chunk contract omitting payload request bodies, OpenAI-compatible OAuth refresh delegation to plugin auth providers, Responses usage token detail fields — landed in `internal/pluginhost` behind local ABI. | source: `ba5ab795a2de` `01a21b77f4dc` `e0b4956242e5` (v7.2.124–v7.2.133)
  - R10 [accepted]: Local behavior preservation — llmhub DB/file token stores, DB-only feature config, embedded management panel, and `sdk/` backward compatibility remain intact; `go test ./...` and `make build` green after every phase. | source: `CLAUDE.md`, plan baseline `f08bec353156`
  - R11 [accepted]: The final gate must re-resolve the latest stable upstream release, fetch its commit into `refs/upstream-checkpoints/cliproxyapi/{tag}`, and refresh the checkpoint and ledger before closure. | source: checkpoint integrity contract
  - R12 [accepted]: If the latest stable upstream release changes after planning, implementation must not silently widen scope; the ledger must identify the delta and either include it through an approved plan refinement or pin the completed scope with an explicit follow-up. | source: targeted-scope policy

## Non-goals
- NG1: Upstream README/docs/sponsorship churn (`docs-readme-churn`) — llmhub branding and release contracts are not upstream's (invariant #5).
- NG2: Carried-forward deferred items — `home-401-refresh-revert`, `gitstore-recovery` — no port work this cycle.
- NG3: No wholesale file-for-file merge; every port lands behind existing llmhub interfaces (translator registry, executor interfaces, store backends).
- NG4: No new test files under `web/`; frontend verified by type check, lint, production build, and browser runtime checks only.

## Approach and Risks
- approach: targeted semantic ports — for each slice, read the upstream diff at `refs/upstream-checkpoints/cliproxyapi/v7.2.135`, classify already-present vs gap per file (ledger dispositions), then re-express the gap behavior behind llmhub's existing interfaces (translator registry, executor interfaces, store backends). Chosen over wholesale merge because the fork baseline is mature (448 changed paths, 161 semantic-review) and a file-for-file merge would invert llmhub invariants.
- constraints:
  - Feature settings from ported slices land as structured DB records via the management API, never as `config.example.yaml` fields (invariant #1).
  - No upstream branding/docs/README content (invariant #5). No new test files under `web/`.
  - `sdk/` public boundaries stay backward compatible (invariant #4).
  - All citations of upstream behavior use the checkpoint ref, never `upstream/main`.
- risks:
  - Auth/runtime cross-cutting changes (R1, R4, R8) can regress credential selection and streaming — mitigation: focused package tests per phase, full `go test ./...` per phase end.
  - translator-correctness (R3) and perf-work (R7) touch the same translator files — mitigation: strict sequencing (perf after correctness), no parallel waves across phases.
  - Local behavior already diverged on some fixes (e.g. `response.completed` partially present) — mitigation: per-path already-present classification with local symbol citation before writing code.
  - Upstream keeps moving during implementation — mitigation: R11/R12 final gate pins the delta instead of silently widening scope.
- rejected alternatives:
  - Wholesale file-for-file merge: would invert DB-only config and SDK-compat invariants; rejected at triage.
  - Per-commit ports: produce incoherent phases; rejected in favor of capability slices.
  - Skipping config-adapts: loses user-approved behavior (per-credential retry, proxy override); rejected.
- recovery: each phase ends at a green `go test ./...`; if a phase regresses a local behavior, revert that phase's changes and re-verify; escalation route: `work` → `check` gate → `handoff`.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase: registry-models | story_id: 01M079Y7894RYN2H233HAPCEM9 | status: in-progress | goal: Merge upstream model registry and definition updates (R5) | depends_on: none
    - waves:
      - wave 1: `git diff {from_ref}..{to_ref} -- internal/registry/models/models.json internal/registry/models/codex_client_models.json internal/registry/model_definitions.go` — map upstream additions (Grok Imagine Video 1.5 GA + preview alias `84232747e20e` `abaeb55bb20e`, Image 2.0 `db35b91e2a46`, Gemini 3.7 Flash High `7ea9c670ea0d`, gpt-5.6-sol-wm `dd214445ef7a` + 921k limits `745fb38dbbe6`, Kimi K2.7 aliases `36936340a33a`) against local registry; identify local-only definitions to preserve
      - wave 2: apply `max_completion_tokens` to model definitions and responses (`046b59ecc519`) and modality metadata (`323b7276bc5b`) in registry code; merge entries into local JSON + Go definitions
    - tasks:
      - T1: register Grok Imagine Video 1.5 GA and Image 2.0 with auth routing aliases
      - T2: register Gemini 3.7 Flash High
      - T3: register gpt-5.6-sol-wm incl. codex_client_models.json entries and 921k context limits
      - T4: add max_completion_tokens field + modality metadata; validate JSON
      - T5: canonicalize Kimi K2.7 aliases
    - checks: `go test ./internal/registry/...`; `python3 -m json.tool internal/registry/models/models.json internal/registry/models/codex_client_models.json`; `go build ./...`; grep `gpt-5.6-sol-wm` present in both JSON files
  - phase: translator-correctness | story_id: 01M079Y78HDM1P6YF0J968EZP3 | status: planned | goal: Port translator correctness fixes (R3) | depends_on: none
    - waves:
      - wave 1: per-path classify the R3 commit set across the translator matrix — already-present (cite local symbol, e.g. `claude_openai-responses_response.go:320` emits `response.completed`) vs gap
      - wave 2: implement gaps: idempotent `response.completed`/`function_call_arguments.done` (`872070259ef6` `0cc24b2e0006`), non-object tool input + block order preservation (`0879929821cb` `aea71c0070fa`), reasoning chain rebuild (`9b1142399c29`), `reasoning_tokens` usage detail (`82d6242098a7`), [DONE] finalization (`4906ead34fa5`), incomplete marking (`78f0c4079e3e`), content_filter mapping (`fab077a04b4a`), tool_use synthesis (`37609fa17993`), deterministic tool IDs (`8d670b98ffac`), dedup (`1ecb7df228cf`), base64 PDF preservation (`44d5e0bebcbc`)
    - tasks:
      - T1: classify all R3 paths with disposition + local citation
      - T2: port confirmed gaps per translator pair, keeping existing local behavior where stronger
    - checks: `go test ./internal/translator/... ./sdk/translator/...`; `go test ./internal/runtime/executor/... -run Thinking`; `git diff --check`
  - phase: claude-oauth-wire | story_id: 01M079Y78RNXCME3J787TPQ17W | status: checked | goal: Reach Claude OAuth wire parity with Claude Code 2.1.220 (R1) | depends_on: none
    - waves:
      - wave 1: map upstream auth/runtime diffs (`f3e25ab2bae6` OAuth identity + TLS, `707934917a74` TLS session resumption — absent locally, `fa6bc77f28e0` stacked encodings, `ef89c6a69d0f` cloaked system prompts, `a2933c7737a4` beta/count_tokens policies) onto `internal/auth/claude` + `internal/runtime/executor/claude_executor.go` + `helps/`
      - wave 2: port wire fixes: Fast error pass-through (`497cf491aab4`), MCP tool aliases incl. BIP-39 (`f6f03e4de99c` `93c378b791f7` `5b5f428ad9a6` `f2d272da817d`), Haiku helpers (`a8bbbea2b9b5`), prompt-cache ownership (`f0034ca66376`), `service_tier`→`speed` (`7eefab98b8e5`)
    - tasks:
      - T1: implement TLS session resumption in the claude auth transport
      - T2: align OAuth exchange/identity and betas with CC 2.1.220
      - T3: port cloak/beta/count_tokens policies and system-prompt reconstruction
      - T4: BIP-39 MCP tool aliases with malformed-ID recovery
      - T5: Haiku helpers, prompt-cache ownership, service_tier→speed
    - checks: `go test ./internal/auth/claude/... ./internal/runtime/executor/... -run Claude`; `go vet ./internal/auth/claude/...`; grep TLS session resumption symbol in claude auth transport
  - phase: websocket-realtime | story_id: 01M079Y87R30TPWN50EKYWMQ3P | status: done | goal: Port websocket/realtime fixes (R6) | depends_on: none
    - waves:
      - wave 1: diff `sdk/api/handlers/openai/openai_responses_websocket.go` + handlers against upstream (`c845ce15c9de` merge/repair refactor, `75d2c4a4b4e2` no-copy repair, `baa11ed6dd75` case-insensitive metadata, `49b2f891ac32` duplicate input semantics, `f9bd9def2b6a` allocations)
      - wave 2: port realtime hangup forwarding + local client-secret (`bd34ceca0420`), premature-termination terminal errors (`522b4de54a8d`), `response.failed` stream errors (`3522e481aa7b`), upstream stream error preservation (`92f03e68e3f1`)
    - tasks:
      - T1: port websocket transcript merge/repair fixes
      - T2: port hangup forwarding and client-secret support
      - T3: port terminal/failed stream error emission paths
    - checks: `go test ./sdk/api/... -run WebSocket`; `go test ./sdk/api/... -run Responses`
  - phase: auth-cooldown-robustness | story_id: 01M079Y87ZZB1S37THPF3T4ZGX | status: planned | goal: Port auth/cooldown/selector robustness (R4) | depends_on: none
    - waves:
      - wave 1: port rotation/penalty semantics in `sdk/cliproxy/auth` + `internal/clienterror`: rotate on unknown failures (`579f5e30fbd6`), client-fault immunity (`c1d69e7b4778` `203f5b1a1d84` `8392b180ce37`), error status mapping (`4b3cc55cdc93`)
      - wave 2: port session affinity (priorities `65fc536e6e32`, variant suffixes `00c4377a21b8`, rebound safety `9169ad56e18b`), thinking-suffix canonicalization (`1674aaf43b52`), rate-limit priority (`9c8e4a07e638`), unified Anthropic rate-limit parsing (`ac82bedfaf5b`), Home OAuth retry (`1e38a3a544ec` `063f9d341285` `d952cb429706` `a81b9e9cedab`)
    - tasks:
      - T1: port rotation + client-fault cooldown semantics
      - T2: port session affinity and canonicalization
      - T3: port rate-limit parsing and client error mapping
      - T4: port Home OAuth retry chain
    - checks: `go test ./sdk/cliproxy/... ./internal/clienterror/...`; `go test ./internal/home/... ./internal/runtime/executor/helps/... -run Home`
  - phase: plugin-schema-v3 | story_id: 01M079Y8875J0QEJTQ9VGETEWR | status: planned | goal: Port plugin schema-v3 + OAuth delegation (R9) | depends_on: none
    - waves:
      - wave 1: diff upstream `internal/pluginhost` changes (`ba5ab795a2de` schema-v3 chunk contract, `01a21b77f4dc` OAuth refresh delegation, `e0b4956242e5` usage token detail) against local pluginhost ABI
      - wave 2: implement schema-v3 chunk contract and delegation behind local ABI; keep schema-v2 compatibility
    - tasks:
      - T1: implement schema-v3 stream chunk contract (omit payload request bodies)
      - T2: delegate OpenAI-compatible OAuth refresh to plugin auth providers
      - T3: emit Responses usage token detail fields
    - checks: `go test ./internal/pluginhost/...`; `go build ./...`
  - phase: codex-multiagent-v2 | story_id: 01M079YBYRK3S6ME13F9XM94PS | status: planned | goal: Port Codex multi-agent v2 + input sanitization (R2) | depends_on: translator-correctness
    - waves:
      - wave 1: port MAv2 tool definitions at the Responses boundary + `multi_agent_version` v2 flag (`7fe847376667` `4f5ec105b11c`) and is-compat rewrites (`7fef08ea2bf4` `e5ea945ed93d`)
      - wave 2: port ID sanitization (`ctc_`/`ctco_`/sub2api) collision-resistant deterministic (`5e25566c240e` `197f52042637` `673bac5fc606` `db143aebac93` `8d675e690ddf`), `max-context-length` overrides (`a303fd869b03`), sequential cutoff reasoning (`5314b29da963`), spawn-agent cache (`98c98d66be1a`), `response.completed` ID hydration (`134a66738c35`), namespace conflict clearing (`133047de66bf`), session header normalization (`f43aad7637ad`)
    - tasks:
      - T1: implement MAv2 tool definitions and flag propagation
      - T2: implement collision-resistant ID sanitization
      - T3: port max-context-length, cutoff reasoning, spawn-agent cache, header normalization
    - checks: `go test ./internal/runtime/executor/... -run Codex ./internal/translator/codex/... ./internal/client/...`; grep `multi_agent` present in codex executor
  - phase: perf-work | story_id: 01M079YBYZRHF9QD3D9NC46RM0 | status: planned | goal: Port perf work (R7) | depends_on: translator-correctness
    - waves:
      - wave 1: port `internal/util` helpers: payload-reuse guards (`b921b5d03264` `1737596e020a`), no-copy GJSON parse (`ea8882bf26b2`)
      - wave 2: port batched array assembly (`8b3b304952ed` `0c58c3c83d3c`), reduced amplification for Codex/Gemini/Antigravity (`c30e60a11b49` `34364fff469e` `b7a441522a90`), connection reuse (`c33a33e14a6d`), replay index perf (`bb7278a1af58` `cf8c27fe90ab` `d5f68856f6da`)
    - tasks:
      - T1: port payload-reuse and no-copy parse helpers
      - T2: switch translators to batched assembly
      - T3: port amplification/connection reuse fixes
    - checks: `go test ./internal/util/... ./internal/translator/... ./internal/runtime/executor/...`; `go test -race ./internal/runtime/executor/... -run Antigravity`
  - phase: config-adapts-db | story_id: 01M079YBZ72AQNR8HMM9XFEGEX | status: planned | goal: Port config features as DB records (R8) | depends_on: auth-cooldown-robustness
    - waves:
      - wave 1: port per-credential request-retry override behavior (`6f2cea948451`) and request-scoped proxy override (`17a479a8db98`) as DB records via management API
      - wave 2: port request-scoped error action handling (`aa10847e1271`) and cooling management refactor (`5bffd1514fba`) behind the conductor; zero new `config.example.yaml` fields
    - tasks:
      - T1: add DB records + management API surface for retry/proxy overrides
      - T2: wire overrides into request path
      - T3: port error-action handling and cooling refactor
    - checks: `go test ./internal/api/... ./sdk/cliproxy/... ./internal/config/...`; `git diff -- config.example.yaml` shows no new feature fields; management API round-trip test for one override record
  - phase: final-gate | story_id: 01M079YBZEJFZYB4VE6SVRX1T9 | status: planned | goal: Re-resolve latest release and pin delta (R11, R12) | depends_on: registry-models, translator-correctness, claude-oauth-wire, websocket-realtime, auth-cooldown-robustness, plugin-schema-v3, codex-multiagent-v2, perf-work, config-adapts-db
    - waves:
      - wave 1: `upstream_sync.py sync --slug cliproxyapi` — re-resolve latest stable release, fetch into checkpoint ref
      - wave 2: refresh gap + ledger; compare against v7.2.135; for every new commit either get approved plan refinement or record explicit follow-up; update checkpoint
    - tasks:
      - T1: re-resolve and sync latest stable release
      - T2: refresh gap matrix and ledger
      - T3: pin delta as follow-up or approved refinement; final `go test ./...` + `make build`
    - checks: sync output reports tag (must be v7.2.135 or higher with delta list); `go test ./...`; `make build`; `git diff --check`

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- 2026-08-17 | claude-oauth-wire | wave 1 | T1-T5 mapping | run 01M07CWNR6QR84MHQ642TBARVM | traces 01M07DJQ5QFM2CRK0CFEDTXG9Y, 01M07DJQ5QFM2CRK0CFEQKZDF1, 01M07DJQ5QFM2CRK0CFGBWVKR7, 01M07DJQ5QFM2CRK0CFKVFX1DC, 01M07DJQ5QFM2CRK0CFMFA73MN | Mapping complete: TLS resumption absent→port, beta policy static→per-request port, BIP-39 aliases into existing rename seam, OAuth identity already-present, cloak reconstruction + Haiku helpers adaptation pending
- 2026-08-17 | claude-oauth-wire | wave 1 | summary | run 01M07CWNR6QR84MHQ642TBARVM | trace 01M07DJWA2R2EH51CRNGR36C5J | wave 1 outcome: surfaces mapped, dispositions recorded
- 2026-08-17 | claude-oauth-wire | wave 2 | T1-T5 + stacked-encodings | run 01M07CWNR6QR84MHQ642TBARVM | traces 01M07DJWAASAHJKWNCQJ0HFSA0, 01M07DJWAASAHJKWNCQNQH5G49, 01M07DJWAASAHJKWNCQPFKM6FB, 01M07DJWAASAHJKWNCQR8MVA2N, 01M07DJWAASAHJKWNCQRD7W124, 01M07DJWAASAHJKWNCQW0HJZ12 | TLS resumption (both transports), 2.1.220 per-request beta assembly + count_tokens fixed profile, BIP-39 aliases (helps/ + executor seam), service_tier→speed, stacked encodings decode; fast-errors + header-casing rejected as inapplicable locally; cloak/Haiku deferred (user-approved)
- 2026-08-17 | claude-oauth-wire | wave 2 | summary | run 01M07CWNR6QR84MHQ642TBARVM | trace 01M07DK24SR6ZDV0D1D2X23MQH | wave 2 outcome: wire-fix slice ported and verified; surfaces: internal/auth/claude/utls_transport.go, helps/utls_client.go, helps/claude_mcp_alias*.go + claude_bip39_words.txt, claude_executor.go, claude_executor_test.go, claude_openai-responses_request.go
- 2026-08-18 | registry-models | wave 1 | T1-T5 mapping | run 01M0ACR6J9XAS0WQ8NPCYGD708 | trace 01M0ACRJGNPNJ70VKS2QKWSPRE | Classification vs upstream v7.2.135 (checkpoint tag): GAP Grok Image 2.0 `xaiBuiltinImage20ModelID="grok-imagine-image-2.0"` + Video 1.5 GA split (local `xaiBuiltinVideo15ModelID` currently `-preview`); GAP Gemini 3.7 Flash High (antigravity section, local has 3.6 only); GAP Kimi K2.7 alias canonicalization (local stripKimiPrefix lacks normalizeKimiUpstreamModel switch); GAP codex_client_models.json Sol/Terra/Luna 372000→272000/921000 (745fb38dbbe6) + models.json codex-pro sol ctx 372000→921000; GAP modality metadata data (struct + /v1/models emission already local, JSON lacks supportedInputModalities/Output on all 126 models); xai preview alias routing ALREADY-PRESENT (canonicalXAIVideosModel + xaiVideos15PreviewModel preserve alias; upstream 61d9a30d126e is a cosmetic UpstreamModel→RoutingModel rename); max_completion_tokens ALREADY-PRESENT on definitions (87 local vs 73 upstream) + ModelInfo emission; codex client models response emission gap (046b59ecc519)
- 2026-08-18 | registry-models | wave 2 | T1-T5 | run 01M0ACR6J9XAS0WQ8NPCYGD708 | trace 01M0AD2H7QRW58Z8CXGWVJP1BP | T1 model_definitions.go: xaiBuiltinImage20ModelID + xaiBuiltinVideo15PreviewID split, WithXAIBuiltins + xaiBuiltinImage20/Video15GA/Video15Preview model-info funcs; T2 gemini-3.7-flash-high inserted after 3.6 in models.json antigravity; T3 models.json codex-pro gpt-5.6-sol ctx 921000 + codex_client_models.json Sol/Terra/Luna 272000/921000; T4 modality merge (244 fields / 123 models) + applyCodexClientMaxTokens max_completion_tokens→max_tokens emission (both template + synthesized paths, mirrors 046b59ecc519); T5 normalizeKimiUpstreamModel K2.7-code/highspeed alias switch, both executor call sites; surfaces: internal/registry/{model_definitions.go,model_definitions_test.go}, models/{models.json,codex_client_models.json}, sdk/api/handlers/openai/codex_client_models.go, internal/runtime/executor/kimi_executor.go | verification: go build ./... OK; go test ./internal/registry/... OK; go test ./sdk/api/handlers/openai/... OK; go test ./internal/runtime/executor/ -run Kimi OK; python3 -m json.tool both JSON OK; git diff --check clean; full go test ./... 63/65 ok — cmd/server + internal/updater fail at clean base too (pre-existing, self-update needs mock release host)

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- 2026-08-17 | claude-oauth-wire/T1 | traces 01M07DK251VWWSSZSX87A1EQK9 | Do not port upstream HelloCustom HTTP/1.1 transport rewrite — inverts local H2 + HelloChrome_Auto design, needs internal/httpwire absent locally; TLS session resumption ported instead
- 2026-08-17 | claude-oauth-wire/T2 | trace 01M07DK251VWWSSZSX8AKRJ2XJ | Reject header-name casing pass (a20626f1eee9) — H2-only local transport makes casing unobservable (HPACK lowercases); fix targets upstream HTTP/1.1 transport
- 2026-08-17 | claude-oauth-wire/T3 | trace 01M07DK251VWWSSZSX8AQTEX4W | Reject Fast error pass-through (497cf491aab4) — Fast fallback machinery absent locally (no fast_fallback.go, no RequestTerminatedError); fast-mode beta still follows the body
- 2026-08-17 | claude-oauth-wire/T2 | trace 01M07DK251VWWSSZSX8DFRZXBC | Skip confirmedClaudeCode passthrough branches — no local client detection; strict policy applies: known caller betas placed at captured positions, unknown dropped on api.anthropic.com
- 2026-08-17 | claude-oauth-wire/T4 | trace 01M07DK251VWWSSZSX8H0WKGPX | Aliases only for third-party tools — local oauthToolRenameMap keeps OpenCode builtins TitleCase (fingerprint/billing invariant); rename map first, BIP-39 aliases for the rest
- 2026-08-17 | claude-oauth-wire/T3 | trace 01M07DK251VWWSSZSX8M7XJVSD | Auth-side stacked-encoding fix N/A — oauth_response.go absent locally; executor-side decode chain ported
- 2026-08-17 | claude-oauth-wire | trace 01M07EK49Q112BMMEJRY0Q9Q64 | Defer cloak reconstruction (ef89c6a69d0f 3fac4a09d80d f0034ca66376) + Haiku helpers (a8bbbea2b9b5) to explicit follow-up — user-approved wrap; both are large adaptations (cloak adds config keys needing DB records; Haiku needs new claude_client_detection.go)
- 2026-08-18 | registry-models/T3 | run 01M0ACR6J9XAS0WQ8NPCYGD708 | plan wording `gpt-5.6-sol-wm` superseded by upstream state: `dd214445ef7a` added `-wm`, `c1ff55fc2f1e` removed it; v7.2.135 ships `gpt-5.6-sol` with 921000 context (codex-pro models.json) + 272000/921000 codex client config. Port the final v7.2.135 state (gpt-5.6-sol, not -wm); success signal interpreted as "gpt-5.6-sol resolvable with 921k context window"
- 2026-08-18 | websocket-realtime | wave 1 | T1 | run (main-agent takeover, user-approved) | merge/repair wave mapping: upstream c845ce15c9de refactor + 75d2c4a4b4e2 no-copy + f9bd9def2b6a alloc NOT-PORTED (local openai_responses_websocket.go structurally independent; merge/dedupe/repair + rollback-on-forward-failure already present, tests green); baa11ed6dd75 case-insensitive metadata PORTED to dedupeFunctionCallsByCallID + shouldReplaceWebsocketTranscript (responsesWebsocketItemMetadata); 49b2f891ac32 duplicate-input semantics PORTED to responsesWebsocketInputField (last-wins, case-insensitive key, any non-array duplicate invalidates) | surfaces: sdk/api/handlers/openai/openai_responses_websocket.go + _test.go | commit b2e23a37 | checks: go test ./sdk/api/... ok
- 2026-08-18 | websocket-realtime | wave 2 | T2 | user-approved "port gaps only" for bd34ceca0420: hangup forwarding + capability-not-supported (501) onto local /backend-api/codex realtime contract; client-secret/sessions/translations/transcription OpenAI /v1/realtime surface NOT-PORTED (different client contract, local serves ChatGPT AVAS shape); upstream home-dispatch selection + 401-refresh-on-hangup N/A locally (no home-dispatch subsystem); session owner principal/provider recorded at CreateCall for hangup scope checks | surfaces: internal/api/handlers/codexlive/handler.go + _test.go, internal/api/server.go, internal/client/codex/live/session.go (Session owner fields) | commit 70866194 | checks: go test ./internal/api/handlers/codexlive/ ./internal/client/codex/live/ ./internal/api/... ok
- 2026-08-18 | websocket-realtime | wave 2 | T3 | terminal/failed stream error emission committed earlier under R6 umbrella: 522b4de54a8d premature SSE/websocket termination + 3522e481aa7b response.failed + 92f03e68e3f1 upstream stream-error preservation (sdk/api handlers + openai responses websocket + claude/gemini close-path) | commits ffe4666d, a5ee94bf | checks: go test ./sdk/api/... ok

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- 2026-08-17 | claude-oauth-wire | gate | run 01M07CWNR6QR84MHQ642TBARVM | check 01M07EKCB2TM25616ST8EMC91K | verdict APPROVE_WITH_REQUESTS (judge same-session, deepseek-v4-flash) | proofs re-run by CLI, all exit 0: `go build ./...`; `go test ./...`; `go vet ./internal/auth/claude/ ./internal/runtime/executor/ ./internal/runtime/executor/helps/`; `go test ./internal/runtime/executor/ -run TestRemapOAuthToolNames -v`; `go test ./internal/runtime/executor/ -run TestApplyClaudeHeaders_BetaAssemblyPerRequest -v`; `go test ./internal/runtime/executor/ -run TestDecodeResponseBody -v`; `git diff --check` | proof_gaps: live wire capture vs api.anthropic.com not re-verified in-session (no live OAuth credential); upstream alias/beta test suites exercised only through ported tests | requests: cloak reconstruction + Haiku helpers shipped as follow-up
- 2026-08-18 | registry-models | phase checks | run 01M0ACR6J9XAS0WQ8NPCYGD708 | verdict green | `go build ./...` exit 0; `go test ./internal/registry/...` ok; `go test ./sdk/api/handlers/openai/...` ok; `go test ./internal/runtime/executor/ -run Kimi` ok; `python3 -m json.tool` both JSON files valid; grep gpt-5.6-sol present in both JSON files (sol-wm removed upstream by c1ff55fc2f1e — see Decisions); `git diff --check` clean; full `go test ./...` 63/65 ok, cmd/server + internal/updater fail at clean base f9df6880 (pre-existing, self-update tests require mock release host)
- 2026-08-18 | websocket-realtime | phase checks | verdict green | `go build ./...` exit 0; `go test ./sdk/api/handlers/openai/ -run 'WebSocket|Responses|DedupeFunctionCalls|ShouldReplaceWebsocketTranscript'` ok; `go test ./sdk/api/...` ok; `go test ./internal/api/handlers/codexlive/ ./internal/client/codex/live/` ok; `go test ./internal/api/...` ok; `go vet ./internal/api/handlers/codexlive/ ./internal/client/codex/live/ ./internal/api/` clean; `git diff --check` clean via commits b2e23a37 + 70866194 | proof_gaps: live realtime hangup against chatgpt.com not re-verified (no live OAuth credential); capability errors exercised through ported unit tests

## Current State and Next Action
- active_phase: translator-correctness
- lifecycle_status: in-progress
- latest_run_id: 01M0ACR6J9XAS0WQ8NPCYGD708
- latest_trace_ids: [01M0ACRJGNPNJ70VKS2QKWSPRE, 01M0AD2H7QRW58Z8CXGWVJP1BP]
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- open_items: [deferred follow-up: cloak reconstruction (ef89c6a69d0f 3fac4a09d80d f0034ca66376) + Haiku helpers (a8bbbea2b9b5); websocket-realtime (R6) complete — commits ffe4666d, a5ee94bf, b2e23a37, 70866194; translator-correctness partial edits uncommitted in main tree]
- exact_next_action: finish translator-correctness (R3) in-session — review + complete the uncommitted translator edits (antigravity_claude_request.go, claude_openai-responses_response.go, openai_openai-responses_response.go + tests) against the plan's target behaviors, then `go test ./internal/translator/... ./sdk/translator/...` + `-run Thinking` + commit; then auth-cooldown-robustness (R4) relaunch on opus
- merged: PR #17 (d26f640f) merged to master 2026-08-17; work/claude-oauth-wire-v7.2.135 branch deleted
