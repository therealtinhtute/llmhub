---
id: plan-20261014-v8028
type: plan
intake_id: intake-20261014-v8028
lane: high-risk
status: implemented
created: 2026-10-14
updated: 2026-10-14
---

# Plan: CLIProxyAPI v8.0.28 targeted parity

## Outcome
- result: Port the high-value v7.2.148..v8.0.28 slices as focused semantic ports inside existing llmhub translator, executor, auth, runtime, and SDK interfaces. Scope stays additive and behavior-focused; no wholesale upstream merge, no branding churn, no pluginhost/Home/gitstore subsystem import, and no file/YAML runtime source of truth.
- success_signals:
  - Each accepted requirement cites at least one upstream commit and lands with package-level tests.
  - Postgres remains the authoritative runtime store; management/auth writes continue to persist to Postgres.
  - Amp/Kiro/Gemini CLI/Codex/Claude routes remain compatible.
  - Public SDK changes are additive only.
  - Newer-than-v8.0.28 delta is pinned explicitly, never silently absorbed.
  - `go test` on touched packages, `make build`, and `git diff --check` pass at each gate.
- risk_tier: high

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v8.0.28` at `0d18d3b365be` / `refs/upstream-checkpoints/cliproxyapi/v8.0.28`; published `2026-10-14`; scope strategy `targeted-semantic-ports`.
  - `docs/upstream/cliproxyapi-gap-v7.2.147..v8.0.28.json` — 1496 changed paths (match 4, baseline 30, upstream-add-absent 536, diverged-absent 467, semantic-review 459, upstream-delete-present-local 0).
  - `docs/upstream/cliproxyapi-ledger-v7.2.147..v8.0.28.md` — 654 non-merge commits across 62 releases.
  - `CLAUDE.md` / `AGENTS.md` — Postgres-authoritative runtime; no new `web/**/*_test.go`; public SDK changes additive only; local xAI executor remains monolithic; llmhub branding retained.
  - `docs/plans/completed/cliproxyapi-v7.2.147-parity.md` — prior cycle accepted baseline; do not regress its requirements.
  - local git baseline: `b923e639f075` on `master`.
- requirements:
  - R1 [accepted]: eliminate concurrent config/runtime access races and harden reload paths. | source: `e235fb3d64a5`, `4f9630df230b`, `81573f042982`, `e230a04b01f8`
  - R2 [accepted]: Codex/OpenAI execution reliability: first-payload stream disconnect as bad gateway, retryable bootstrap failover, ALPN http/1.1, outbound tool namespace restore, composite collaboration tool names, empty incomplete response handling, integer schema bound sanitization, thinking effort/client version behavior. | source: `3ebee065161b`, `ba7e55836dee`, `82f4f370df45`, `7795de19ca80`, `b0db5448e8f8`, `57d005fe6c20`, `40ce0e0121cd`, `1ecf0cb60204`
  - R3 [accepted]: translator robustness: schema sanitization for anyOf/oneOf/one-branch unions, Unicode property escapes, tool parameters and parameter refs, functionResponse normalization, responses timeline ordering, empty tool_calls handling, restore namespaced tool names, apply_patch bridge, unique fallback item IDs. | source: `a0fd8448f575`, `e56abd56f142`, `37ce368c5002`, `d0bb908ce4f1`, `03a054e32fc1`, `bee20b994025`, `510c9c8fa529`, `77fd91e8f463`, `40ce0e0121cd`
  - R4 [accepted]: Claude Code compatibility upgrades: fingerprint baseline 2.1.258, dynamic beta headers, model fallbacks, paired cache TTL, billing header fingerprint chain, max-token defaults from registry, helper request IDs, codex reasoning token → thinking token, deferred tool discovery cache preservation. | source: `df7e04ea2850`, `d7052c96af78`, `086ad91bd970`, `4a5ab534f827`, `280b96acead0`, `35a4723872a4`, `f62d4d28ceb4`, `02fc76b149a9`, `333f62b39653`
  - R5 [accepted]: auth lifecycle and cooldown fairness: minimum cooldown floor, attempted credential tracking on 429, token expiration validation, retain valid credentials on refresh failure, refresh loop cap, active cooldown deadline preservation, concurrent refresh/preparation preservation, quota signals for Claude/Codex headers and WebSocket frames without making quota files authoritative. | source: `18e01a76ac72`, `f416175fcd29`, `9812b1e76872`, `09471dd9daba`, `1c22598d0b5a`, `48e5e9e03d21`, `4c1bebe837a6`, `1d5f7b2ac36f`, `d4146bde1248`, `ca601db0`
  - R6 [accepted]: Antigravity bundle: conversation compaction + capsule encryption, compaction fallback, omit tools for tool_choice none, thinking summary when reasoning effort is set, batch reasoning/functionResponse repairs, resolved pool settings in transport cache key. | source: `70f456045222`, `5dc428f39270`, `0e85eb46f32b`, `a76da7115486`, `e92f6cf55700`, `acf919ce50fb`, `0fe19ede90a4`, `68dd99d56f68`, `1c45093d10f6`
  - R7 [accepted]: Kimi robustness: OpenAI Responses support, request thinking helper, tool/function parameter schema normalization, namespaced tool and input tool list normalization, schema clamp by target executor. | source: `d4146bde1248`, `c6dd82144baf`, `dd5f9e74`, `40ce0e0121cd`
  - R8 [accepted]: schema sanitization core: patternProperties inspection, multi-branch anyOf/oneOf preservation, Gemini response schema normalize `responseJsonSchema` → `responseSchema`, default empty tool parameters. | source: `37ce368c5002`, `e56abd56f142`, `a0fd8448f575`, `6e1f9ec4e0b6`, `dc21a4263912`
  - R9 [accepted, configuration]: client configuration section for Codex settings and connection-pool short connection behavior. | source: `63c04b4bd473`, `d5397905f09e`
  - R10 [accepted]: management/operational UX: refresh auth files endpoint, health-probe logging silence. | source: `60e5b8bd432e`, `a163c5e7edec`, `e026cbf4e9ce`
  - R11 [accepted]: model registry/catalog additions adapted to llmhub naming and fetchers: Claude Haiku 5.5, Gemini 4 / Argon, Muse Wonder, web_search native capability, model-specific max-token defaults. | source: `f62d4d28ceb4`, `6143cc11c9e1`, `866d84d08674`, `4f9630df230b`, `4c159069e201`
  - R12 [accepted, invariant]: Postgres remains authoritative; management/auth writes persist directly; no local durable config/auth/log files; Amp/Kiro/Gemini CLI routes remain behavior-compatible. | source: CLAUDE.md
  - R13 [accepted, invariant]: SDK public surface changes are additive; existing `SessionAffinitySelector`, auth consumer, and API handlers keep working. | source: CLAUDE.md
  - R14 [accepted, invariant]: local xAI executor stays monolithic; port behavior, not upstream file split. | source: CLAUDE.md
  - R15 [accepted]: final gate re-resolves latest stable upstream release, fetches its checkpoint ref, refreshes checkpoint/ledger, and pins newer delta explicitly. | source: upstream checkpoint integrity
  - R16 [accepted]: optional Devin/Cognition provider introduced as `sdk/auth/devin` style port only after separate product decision; backend surfaces remain `internal/auth/devin`, `internal/runtime/executor/devin_executor.go`, translator additions, registry fetcher. | source: `feat(devin)` commit family (63 commits)
  - R17 [accepted]: pluginhost schema v5/v6 is out for this cycle unless a separate plugin initiative approves it; no plugin schema changes here. | source: upstream plugin commits

## Non-goals
- NG1: branding, README, sponsor assets, icons, docs translations, AGENTS.md wording.
- NG2: pluginhost schema v5/v6, plugin store release checks, plugin API ABI changes.
- NG3: Home/gitstore recovery or control plane. Postgres remains sole durable source.
- NG4: test-hygiene-only commits, sleeps, wall-clock TTL adjustments, wall-clock retry timing tweaks.
- NG5: Devin provider unless product explicitly approves Wave C.
- NG6: Devin model catalog updater as a standalone subsystem in this cycle.
- NG7: wholesale merge of 1496 paths or direct copy of upstream files.
- NG8: new `web/**/*_test.go`.
- NG9: management-post-persist synthesis, superseded locally by `upsertAuthRecord`.
- NG10: global file deletion/renaming from upstream; any delete must be checked for rename first.

## Scope Policy (this plan)
- include: `core-runtime-races`, `codex-reliability`, `translator-robustness`, `claude-code-compat`, `auth-cooldown-lifecycle`, `antigravity-compaction`, `kimi-responses-robustness`, `schema-sanitization`, `codex-client-config`, `management-auth-refresh`, `model-catalog-additions`, `quota-usage-state`
- exclude: `branding-docs`, `pluginhost-schema`, `home-gitstore`, `github-token-assets`, `test-hygiene`
- defer: `devin-provider`, `plugin-store-network`, `session-hierarchy-quantum`, `namespace-responses-tools`, `claude-allowed-warning`

## Approach and Risks
- approach: Semantically port in four tightly scoped waves. Wave 0 is read-only contract mapping and test baseline. Wave 1 runs non-overlapping engineering streams in parallel where file surfaces do not collide. Wave 2 integrates provider-specific translators that share `internal/translator` interfaces. Wave 3 runs system-level integration, build, and cleanup gates. Devin is isolated as optional Wave C and must not block the main parity wave.
- constraints:
  - Postgres-authoritative runtime; no local durable files.
  - Existing xAI executor remains monolithic.
  - Public SDK changes additive only.
  - No pluginhost/Home/gitstore/branding work.
  - Every task names an upstream commit and local target file before implementation.
- rejected alternatives:
  - Porting Devin as part of the default wave — too large, would invert the core-first value of this plan.
  - Full runtime refactor to upstream file layout — unsafe under local divergence (536 add + 467 diverged + 459 semantic review).
  - Rewriting SessionAffinitySelector — already ported additively earlier; no behavior change here.
  - Silently widening scope to all 62 releases — pinned as follow-up by final gate instead.
- risks:
  - R1/R3 overlap in `internal/translator` and `internal/runtime/executor` can create merge conflicts across parallel streams.
  - Claude/Antigravity runtime commits overlap file sets; prioritize files, not packages, and serialize write access on same file.
  - Schema sanitizer changes can over-flatten valid schemas and break provider requests; require focused tests plus sample schemas.
  - Auth cooldown changes can alter load-balancing fairness; require auth scheduler tests and a manual multi-account request check.
  - Connection-pool short-connection behavior may affect latency under load; gate with a bounded perf smoke check, not a full benchmark.
  - Devin deferred status may leave model IDs referenced by newer upstream config; only adapt aliases needed for supported providers.
- recovery:
  - If a stream cannot map to a local symbol, stop that stream as `NEEDS_CONTEXT`; do not invent a subsystem.
  - If a parallel merge conflicts, resolve in a dedicated integration task rather than forcing both streams to continue.
  - If build breaks after a stream, revert only that stream's changed files and re-run its focused tests.
  - If a requirement proves impossible without breaking a non-negotiable invariant, surface it as `AskUserQuestion`; do not silently drop it.

## Parallel Work Streams
- stream: P1 `core-runtime-race`
  - owner: runtime/api agent
  - scope: `internal/api/**`, `internal/watcher/**`, `internal/config/**`, `sdk/cliproxy/**` race-prone paths only
  - depends_on: none
  - inputs: R1, R9
- stream: P2 `codex-reliability`
  - owner: codex agent
  - scope: `internal/runtime/executor/*codex*`, `internal/translator/codex/**`, `internal/client/codex/**`, `sdk/api/handlers/openai/**`
  - depends_on: none
  - inputs: R2
- stream: P3 `translator-robustness`
  - owner: translator agent
  - scope: `internal/translator/{claude,codex,gemini,openai,antigravity,common}/**`, `internal/util/**` schema helpers
  - depends_on: P2 only for shared translator interface changes
  - inputs: R3, R8
- stream: P4 `claude-compat`
  - owner: claude agent
  - scope: `internal/runtime/executor/*claude*`, `internal/translator/claude/**`, `internal/translator/codex/claude/**`, registry models
  - depends_on: P3 for translator interface additions
  - inputs: R4, R11
- stream: P5 `auth-lifecycle`
  - owner: auth agent
  - scope: `sdk/cliproxy/auth/**`, `sdk/auth/**`, `internal/auth/**`
  - depends_on: none
  - inputs: R5, R10
- stream: P6 `antigravity`
  - owner: antigravity agent
  - scope: `internal/runtime/executor/*antigravity*`, `internal/translator/antigravity/**`, `internal/auth/antigravity/**`
  - depends_on: P3 for shared translator helpers
  - inputs: R6
- stream: P7 `kimi-and-schema`
  - owner: kimi/schema agent
  - scope: `internal/runtime/executor/kimi_*`, `internal/thinking/provider/kimi/**`, `internal/util/**` schema helpers
  - depends_on: P3 if touching shared schema helpers
  - inputs: R7, R8
- stream: P8 `model-catalog`
  - owner: registry agent
  - scope: `internal/registry/**`, `cmd/fetch_*_models/**`, `internal/client/**` model manifests
  - depends_on: P4 for Claude max-token defaults
  - inputs: R11
- stream: P9 `integration-gate`
  - owner: integration agent
  - scope: build/test orchestration only; no production code edits except merge-conflict resolution
  - depends_on: P1-P8
  - inputs: R12, R13, R14, R15
- stream: P10 `devin-optional`
  - owner: devin agent
  - scope: only if Wave C approved: `sdk/auth/devin*`, `internal/auth/devin/**`, `internal/runtime/executor/devin_executor.go`, translator additions, registry fetcher
  - depends_on: product decision
  - inputs: R16

## Phases and Verification
- planning_status: planned
- phases:
  - phase_slug: v8028-wave0-contract-map
    status: done
    goal: Establish exact upstream-to-local symbol map and test baseline for every accepted stream. No production edits.
    depends_on: none
    tasks:
      - task: For each R1-R11, record upstream commit, changed upstream file, local target file/symbol, disposition (`adapt`/`already-present`/`superseded-locally`), and test surface in a companion task table under this plan.
      - task: Run baseline `go test` on planned touched packages and save command list in the plan's checks section before edits.
      - task: Confirm no new conflicts with Postgres/runtime invariant and local xAI monolith boundary.
    checks:
      - check: go test ./internal/api/... ./internal/watcher/... ./internal/runtime/executor/... ./internal/translator/... ./sdk/cliproxy/... ./sdk/auth/... -count=1
      - check: go build ./...
      - check: git diff --check
  - phase_slug: v8028-wave1-parallel-core
    status: done
    goal: Land independent high-value fixes: runtime races, Codex reliability, auth lifecycle, schema core, Kimi robustness, management auth refresh. P1/P2/P5/P7/P8 may run concurrently where file surfaces do not overlap.
    depends_on: v8028-wave0-contract-map
    allowed_surfaces:
      - paths from streams P1, P2, P5, P7, P8 only
    avoided_surfaces:
      - `internal/translator/**` shared interfaces except through P3 contract
      - pluginhost, Home, gitstore, branding, `web/**`, `sdk/cliproxy/session/**`
    waves:
      - wave: W1A — P1 core-runtime-race
        tasks:
          - task: Port concurrent config/runtime access race elimination and reload safety into local server/config structures. Keep Postgres source-of-truth and startup fail-fast behavior.
          - task: Harden auth file rescan/watcher to preserve concurrent file updates.
        checks:
          - check: go test ./internal/api/... ./internal/watcher/... ./internal/config/... -count=1
      - wave: W1B — P2 codex-reliability
        tasks:
          - task: Port first-payload disconnect handling, retryable bootstrap errors, ALPN http/1.1, tool namespace restore, composite collaboration tool names, empty incomplete response detection, out-of-range int64 schema bound stripping.
          - task: Retain target-executor-aware tool integer normalization and compatibility translation path.
        checks:
          - check: go test ./internal/runtime/executor/... ./internal/translator/codex/... ./internal/translator/openai/... ./internal/client/codex/... -count=1
      - wave: W1C — P5 auth-lifecycle
        tasks:
          - task: Port cooldown floor and attempted credential tracking on 429, token expiration validation, valid credential retention on refresh failure, refresh loop cap, active cooldown preservation, concurrent refresh preparation preservation.
          - task: Add passive quota observation helpers from Claude/Codex response headers and WebSocket frames without making quota files authoritative.
        checks:
          - check: go test ./sdk/cliproxy/auth/... ./sdk/auth/... ./internal/auth/... -count=1
      - wave: W1D — P7 kimi-and-schema
        tasks:
          - task: Port Kimi request thinking helper and OpenAI Responses support, including namespaced tool and input tool list schema normalization and root object normalization.
          - task: Port schema sanitization for Unicode property escapes, patternProperties inspection, anyOf/oneOf preservation, default empty tool parameters, and Gemini responseJsonSchema normalization.
        checks:
          - check: go test ./internal/runtime/executor/ -run Kimi -count=1 && go test ./internal/util/... ./internal/translator/... -run 'Schema|Kimi' -count=1
      - wave: W1E — P8 model-catalog
        tasks:
          - task: Adapt registry additions for Claude Haiku 5.5, Gemini 4/Argon, Muse Wonder, web_search native capability, and model-specific max-token defaults.
          - task: Update codex client model manifests only where behavior requires them.
        checks:
          - check: go test ./internal/registry/... ./internal/client/codex/models/... -count=1
  - phase_slug: v8028-wave2-provider-translators
    status: done
    goal: Port provider compatibility changes that require shared translator interfaces.
    depends_on: v8028-wave1-parallel-core
    tasks:
      - wave: W2A — P4 claude-compat behind P3 interface changes
        tasks:
          - task: Port Claude Code fingerprint/beta header/model fallback/cache TTL changes, helper request IDs, codex reasoning token → thinking token, deferred first-tool cache preservation, passthrough MCP tool names, prompt cache preservation.
          - task: Port Claude web_search round-trip, sequential tool indices, thinking block strip/collapse, unsupported prefill drop, advisor beta, system-message tool pairing, and message_delta when finish_reason is omitted.
        checks:
          - check: go test ./internal/runtime/executor/ -run Claude -count=1 && go test ./internal/translator/claude/... ./internal/translator/codex/claude/... -count=1
      - wave: W2B — P6 antigravity-compaction
        tasks:
          - task: Port conversation compaction and capsule encryption, compaction fallback from payload input/previous response ID.
          - task: Port thinking summary when Responses reasoning effort is set, omit tools for tool_choice none, batch functionResponse name repair, transport cache key including resolved pool settings, replacement offset guard.
        checks:
          - check: go test ./internal/runtime/executor/ -run Antigravity -count=1 && go test ./internal/translator/antigravity/... -count=1
      - wave: W2C — P3 shared translator robustness finalization
        tasks:
          - task: Finalize shared translator changes from R3 after provider-specific needs are known; keep package interfaces stable.
          - task: Re-run affected provider translators after shared interface changes.
        checks:
          - check: go test ./internal/translator/... -count=1
  - phase_slug: v8028-wave3-management-config
    status: done
    goal: Port management and operational changes without changing storage authority.
    depends_on: v8028-wave2-provider-translators
    tasks:
      - task: Add auth file refresh management endpoint and wiring, without introducing file-based durable state.
      - task: Apply health-probe logging silence.
      - task: Apply client configuration section for Codex settings and connection pool short-connection behavior behind config validation.
    checks:
      - check: go test ./internal/api/... ./internal/config/... ./sdk/api/... -count=1
  - phase_slug: v8028-wave4-integration-gate
    status: done
    goal: Run full touched-package tests, build, and scope audit; pin newer delta.
    depends_on: v8028-wave3-management-config
    tasks:
      - task: Run all touched package tests and `make build`; resolve any integration conflicts between parallel streams.
      - task: Inspect `git diff --check`; ensure no formatting or accidental generated churn.
      - task: Verify management panel/client flows manually for Claude Code and Codex at minimum (OAuth status, model list, chat completion, streaming).
      - task: Run final upstream gate: resolve latest stable release, fetch `refs/upstream-checkpoints/cliproxyapi/{tag}`, refresh checkpoint and ledger, and pin any newer delta as explicit follow-up.
    checks:
      - check: go test ./internal/... ./sdk/... -count=1 (or documented touched subset if full run is too broad)
      - check: make build
      - check: git diff --check
  - phase_slug: v8028-wavec-devin-optional
    status: deferred
    goal: Optional product-gated Devin/Cognition provider port.
    depends_on: product decision after v8028-wave4 passes
    tasks:
      - task: Confirm product approval to add Devin/Cognition as a first-class provider.
      - task: If approved, port OAuth, model catalog/updater, executor, translator shapes, signature handling, and quota accounting as a separate scoped plan.
      - task: Keep Devin chat/proxy endpoint and management auth flows aligned with existing provider plugin patterns.
    checks:
      - check: to be defined in the separate Devin plan

## Parallel Execution Directive
- `P1`, `P2`, `P5`, `P7`, and `P8` are safe to dispatch as parallel subagents in Wave 1 because their primary surfaces are distinct; exception is shared `internal/util` schema helpers, where `P7` owns writes and others may read.
- `P4` and `P6` must wait for `P3` shared translator interface mapping in Wave 2, but can run in parallel once that contract is locked.
- `P9` integration runs only after all streams complete. It may prepare test commands in parallel, but must not edit production code except documented conflict resolution.
- `P10` is never part of the default autonomous wave. Dispatch it only after explicit user approval.
- Files touched by multiple streams (`internal/runtime/executor` package, model registry, translator common interfaces) must be serialized per file: one writer, one integration task for merge resolution.

## Validation
- Keep changes consistent with local monolith xAI executor and Postgres runtime boundaries.
- Run focused package tests after every wave, not only at the end.
- Use real smoke tests for auth and quota flows; do not depend solely on mocks.
- All ported requirements must remain traceable to upstream commits by short SHA.

## Agent Dispatch Batches
- batch 0 (sequential, 1 primary agent): Wave 0 contract map and baseline verification.
- batch 1 (up to 5 concurrent subagents): `P1`, `P2`, `P5`, `P7`, `P8`.
- batch 2 (up to 3 concurrent subagents): `P3` shared translator contract, then `P4` and `P6` in parallel after that contract is locked.
- batch 3 (up to 3 concurrent subagents): Wave 3 management/config/auth-file and connection-pool tasks, with file ownership serialized by target file.
- batch 4 (1 integration agent, sequential): Wave 4 merge resolution, full test gate, manual checks, upstream final gate.
- batch 5 (optional, 1 agent only after user approval): Wave C Devin port.
- dependency note: the graph, not raw agent count, limits concurrency. `P4` and `P6` cannot start before `P3` contracts are locked; `P9` cannot start before all production streams merge.

## Handoff
- Next steps: run Wave 0 contract map first. After that, dispatch Wave 1 streams `P1`, `P2`, `P5`, `P7`, `P8`; then Wave 2 `P3`, `P4`, `P6`; then Wave 3 and the system gate. Do not start Wave C without product approval.>|<|start|>assistant<|channel|>analysis<|message|>We wrote plan. Need maybe validate git diff? No need. Respond to user with plan file and summary. Ensure not too verbose but useful. Mention scaling/parallel streams. Also note no code changed. We can mention plan path. Final.<|end|><|start|> Kai royal jelly· 2 h 24 min <|end|>assistant<|channel|>final<|message|>Đã tạo kế hoạch at `docs/plans/active/cliproxyapi-v8.0.28-targeted-parity.md`.

Plan chia thành **4 wave + 1 optional Devin wave**, với **10 parallel work stream** (`P1`–`P10`), mỗi task gắn R-requirements và commit upstream. Tóm tắt:

- **Wave 0**: contract map + baseline tests, không sửa code.
- **Wave 1 song song**: `P1` race/reload, `P2` Codex reliability, `P5` auth/cooldown, `P7` Kimi/schema, `P8` model catalog.
- **Wave 2**: `P4` Claude compat, `P6` Antigravity, `P3` shared translator finalize.
- **Wave 3**: management/config changes, giữ Postgres authoritative.
- **Wave 4**: integration gate + upstream final gate.
- **Wave C (deferred)**: Devin/Cognition provider chỉ khi approve riêng.

Ràng buộc đã bao gồm: no branding/pluginhost/Home/gitstore, no web tests, xAI monolith, additive SDK, Postgres runtime authority. Nếu bạn approve, bước tiếp theo là Wave 0 (contract map) trước khi dispatch các sub-agent song song.No code changed yet. Current diff is just the new plan file.<|end|><|start|>tool_call_begin|><}> No close tag<|end|><|start|>tool_call_begin|><}> No close tag<|channel|><|end|>tool_call_begin|><}> No close tag|<|start|>tool_call_begin|><}> No close tag|<|start|>tool_call_begin|><}> No close tag<|end|><|start|>tool_call_begin|><}> No close tag|<|start|>tool_call_begin|><}> No close tag<|end|><|start|>tool_call_begin|><}> No close tag<|end|><|start|>tool_call_begin|><}> No close tag<|message|>|tool_call_begin|><}> No close tag|<|start|>tool_call_begin|><}> No close tag<|end|>|tool_call_begin|><}> No close tag<|end|>|tool_call_begin|><}> No close tag<|end|><|start|>tool_call_begin|><}> No close tag|<|start|>tool_call_begin<|message|>><}> No close tag

## Execution Record (2026-10-14)
- Wave 0: 5 read-only explore subagents produced symbol maps per stream.
- Wave 1: P1 (config clone + Server.cfg locks + watcher rescan hardening), P2 (bootstrap overload retryable server_error + dotted collaboration restore + uTLS ALPN fallback + client_version gap test), P5 (cooldown floor 10s + attempted tracking + JWT exp + deadline preservation + refresh cap), P7 (Kimi schema normalize + shared anyOf/oneOf + unicode helpers), P8 (Haiku 5.5 / Argon / Muse Wonder / web_search capability) — all landed uncommitted with focused tests.
- Wave 2: P3 locked shared-translator contract (call-site walkers + Responses funnel); P4 (Claude 2.1.258 compat bundle); P6 (tool_choice none omit + thinking summary + batch repairs).
- Wave 3: auth-files refresh resync endpoint, health-probe log silence, Codex client config section, client_version threading.
- Wave 4: `go build ./...` ok, `go test ./internal/... ./sdk/...` all ok, `make build` ok, `git diff --check` clean. Final upstream gate: still v8.0.28, no delta.
- Deferred NEEDS_CONTEXT: Devin provider (Wave C), Antigravity compaction/capsule, transport pool config, per-credential force-refresh, Kimi Responses infra.
