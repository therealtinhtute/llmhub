---
id: plan-20261019-v8033
type: plan
intake_id: intake-20261019-v8033
lane: high-risk
status: active
created: 2026-10-19
updated: 2026-10-19
---

# Plan: CLIProxyAPI v8.0.33 targeted parity

> Reduced mode: `zharness preflight brainstorm --mode lock --json` is unavailable in zharness 0.23.1 (`error: unrecognized subcommand 'preflight'`). This lock is recorded from repository authority; rerun preflight with a zharness that ships the subcommand before treating the lock as durably gated.

## Outcome
- result: Port the approved v8.0.28..v8.0.33 capability slices as focused semantic ports inside existing llmhub auth, watcher, translator, executor, registry, and management interfaces. Scope stays additive and behavior-focused; no wholesale upstream merge, no branding churn, no pluginhost runtime import, and no file/YAML runtime source of truth.
- success_signals:
  - Each accepted requirement cites at least one upstream `{tag} {sha}` and lands with package-level tests.
  - Postgres remains the authoritative runtime store; management/auth writes continue to persist to Postgres.
  - Amp/Kiro/Gemini CLI/Codex/Claude routes remain behavior-compatible; the local xAI executor stays monolithic.
  - Public SDK changes are additive only.
  - Newer-than-v8.0.33 delta is pinned explicitly, never silently absorbed.
  - `go test` on touched packages, `make build`, and `git diff --check` pass at each gate.
- risk_tier: high

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v8.0.33` at `2422c6b0f5fe16495320fa29456493874c1977a0` / `refs/upstream-checkpoints/cliproxyapi/v8.0.33`; published `2026-10-18`; scope strategy `targeted-semantic-ports`.
  - `docs/upstream/cliproxyapi-gap-v8.0.28..v8.0.33.json` — 215 changed paths (match 0, baseline 3, upstream-add-absent 52, diverged-absent 71, upstream-delete-present-local 0, semantic-review 89). Derived and gitignored; regenerate with `upstream_gap.py`.
  - `docs/upstream/cliproxyapi-ledger-v8.0.28..v8.0.33.md` — 54 non-merge commits across 5 releases; every row carries a disposition (`adapt` 41, `defer` 6, `reject` 7).
  - `docs/upstream/cliproxyapi-semantic-review-v8.0.28..v8.0.33.md` — 89 semantic-review paths, per-slice local citations and upstream commits.
  - `~/quaqua/raw/2026-10-19-llmhub-upsync.md` — owner-approved triage decision (2026-10-19): include the 18 slices below; exclude `docs-branding` and `pluginhost-shadow`; defer exactly `vertex-config-upload`, `tls-handshake-timeout`, `devin-quota`, `discovery-mdns`.
  - `CLAUDE.md` / `AGENTS.md` — Postgres-authoritative runtime; no new `web/**/*_test.go`; public SDK changes additive only; local xAI executor remains monolithic; llmhub branding retained; pluginhost is a stub.
  - `docs/plans/completed/cliproxyapi-v8.0.28-targeted-parity.md` — prior cycle accepted baseline; do not regress its requirements.
  - local git baseline: `771e5cefb` on `quaqua/tieude-uplock`.
- requirements:
  - R1 [accepted]: restore per-account excluded-model priority and tolerate invalid excluded-model values on load, keeping plugin-model binding out of scope. | source: `v8.0.33 b015f208d28c`, `v8.0.33 94d1a9a04646`, `v8.0.33 1a28c61e5632`, `v8.0.33 3e309fb47471`, `v8.0.33 bb4282471a88`, `v8.0.30 5fc580b1d72d`
  - R2 [accepted]: add epoch-checked model-registration fencing and latest-auth-state reconciliation so superseded registrations cannot publish or overwrite newer state. | source: `v8.0.29 08e5762a9a3c`, `v8.0.33 94d1a9a04646`, `v8.0.33 f75ea0733657`, `v8.0.33 16b53545e006`
  - R3 [accepted]: track auth file removals and bind single-file snapshots so stale snapshots cannot delete or overwrite later persisted auth state. | source: `v8.0.29 5fc580b1d72d`, `v8.0.33 c4c1d43eaf8c`
  - R4 [accepted]: protect active model and credential cooldowns from stale in-flight successes by carrying an execution sequence through auth results. | source: `v8.0.30 35af74f92721`
  - R5 [accepted]: support Claude preserved-thinking block binding, adaptive fallbacks, `between_tools` disable type, and operator header overrides (beta strip/append, inject-diagnostics, client-request-id modes). | source: `v8.0.32 911f49cbd49f`, `v8.0.32 7604d3df3e62`, `v8.0.33 e13bc9a1f0ca`, `v8.0.33 ad9e456d4a07`, `v8.0.30 9db0729f5fef`, `v8.0.30 4e8a4125ace0`
  - R6 [accepted]: clean and normalize Claude tool arguments against original schemas, including tool-name mapping and cached-token usage correction. | source: `v8.0.32 202b8b8c80bd`, `v8.0.29 912a038b8b39`
  - R7 [accepted]: make Gemini→Claude translation treat `MALFORMED_FUNCTION_CALL` as retryable and keep tool-use IDs unique across turns. | source: `v8.0.29 7f2f17d76fe1`, `v8.0.29 582570076282`, `v8.0.29 a19a8ecde3eb`
  - R8 [accepted]: sanitize Gemini/Antigravity JSON schemas by flattening multi-branch tool unions, stripping `discriminator`, `contentMediaType`, and `contentEncoding`, and scaling linearly with declaration count. | source: `v8.0.29 5efe97582c05`, `v8.0.30 e9ba5c507f46`, `v8.0.33 a43026d75196`, `v8.0.33 b8d4e14a8e65`
  - R9 [accepted]: raise the Antigravity fallback client version, expose user-agent helpers, correct cached-token usage, and drop trailing thought-only turns. | source: `v8.0.29 8fb633c9fa21`, `v8.0.30 ceb4397f1e83`, `v8.0.33 acaab599988f`
  - R10 [accepted]: map incomplete OpenAI Responses tool status to `tool_result is_error` and pin the validated request body per stream. | source: `v8.0.30 59930ba36a82`, `v8.0.31 7a511e4db78a`
  - R11 [accepted]: narrow the Claude per-minute rate-limit cooldown when a short `Retry-After` is present. | source: `v8.0.33 db1bab011d98`
  - R12 [accepted]: add session-date pin retention so cloak-date continuity survives credential switches. | source: `v8.0.32 7604d3df3e62`
  - R13 [accepted]: detect upstream stream truncation across executors and proxy handlers and surface it instead of a silent EOF. | source: `v8.0.33 4f2e52709dcd`
  - R14 [accepted]: filter `response.subscription_usage` telemetry from Meta SSE streams and responses, including graceful client disconnect after terminal events. | source: `v8.0.32 088e445715e2`, `v8.0.33 3a0722984ab7`
  - R15 [accepted, partial]: serialize management config mutations and add ETag/If-Match optimistic concurrency to v8 config endpoints. Upstream's file-snapshot `SourceHash` guard is not applicable to the Postgres-authoritative store. | source: `v8.0.30 1631875b9ac3`, `v8.0.31 fa68946ad572`
  - R16 [accepted]: update the registry catalog with `muse-wonder-full-1.0` variants, Gemini 3.8/4-Argon definitions, and Claude family date/bracket lookup. | source: `v8.0.29 f9ac397be288`, `v8.0.29 a19a8ecde3eb`, `v8.0.30 7459255675f7`, `v8.0.33 ad9e456d4a07`
  - R17 [accepted]: drop xAI Responses `web_search` options the upstream API rejects while keeping the local xAI executor monolithic. | source: `v8.0.30 bbfd7886ee13`
  - R18 [accepted]: strip Codex SIWC `configuration_update` payloads and carry `KeepTopLevelEffort`; port only the non-pluginhost portion of the carrying commit. | source: `v8.0.31 355fd10e76b8`
  - R19 [accepted, invariant]: Postgres remains authoritative; management/auth writes persist directly; no local durable config/auth/log files; Amp/Kiro/Gemini CLI routes remain behavior-compatible. | source: CLAUDE.md
  - R20 [accepted, invariant]: SDK public surface changes are additive; existing `SessionAffinitySelector`, auth consumer, and API handlers keep working. | source: CLAUDE.md
  - R21 [accepted, invariant]: the local xAI executor stays monolithic; port behavior, not upstream file split. | source: CLAUDE.md
  - R22 [accepted, invariant]: llmhub branding and the embedded management web layout/tokens/components/i18n stay local; no pluginhost runtime import. | source: CLAUDE.md
  - R23 [accepted]: the final gate re-resolves the latest stable upstream release, fetches its commit into `refs/upstream-checkpoints/cliproxyapi/{tag}`, and refreshes checkpoint/ledger before closure. | source: upstream checkpoint integrity
  - R24 [accepted]: if the latest stable upstream release changes after planning, implementation must not silently widen scope; the ledger identifies the delta and either includes it through an approved plan refinement or pins the completed scope with an explicit follow-up. | source: targeted-scope policy

## Non-goals
- NG1: branding, README, sponsor assets, icons, docs translations, AGENTS.md wording (`docs-branding`). Invariant: llmhub branding is not upstream's.
- NG2: pluginhost shadow copies, plugin runtime, plugin API config defaults, plugin model binding/exclusions (`pluginhost-shadow`). Invariant: local `internal/pluginhost` is a registration stub, not a runtime.
- NG3: Vertex full `/publishers/` prefix and `PrepareConfigFileForUpload` seeding (`vertex-config-upload`). Trigger to pull in: an operator needs full publishers-prefix API-key base URLs, or a non-Postgres config store needs upload seeding.
- NG4: TLS handshake timeout/retry (`tls-handshake-timeout`). Trigger to pull in: silent TLS handshake hangs are observed in production or an operator requests an opt-in bound.
- NG5: Devin account-wide quota exhaustion tracking (`devin-quota`). Trigger to pull in: a separate Devin/Cognition provider cycle is approved.
- NG6: mDNS host-conflict resolution and IP binding fallback (`discovery-mdns`). Trigger to pull in: mDNS/multicast users report host conflicts; the upstream change is a 2572-line zeroconf rewrite.
- NG7: wholesale merge of 215 paths or direct copy of upstream files.
- NG8: new `web/**/*_test.go` files.
- NG9: splitting the local xAI executor into upstream's file layout.
- NG10: `already-present` upstream commits are recorded as evidence only; this plan does not re-port them.

## Scope Policy (this plan)
- source: `docs/upstream/cliproxyapi-checkpoint.json` `scope_policy`, written with `upstream_sync.py policy` (include/exclude) plus the approved defer list.
- include: `auth-excluded-models`, `model-registration-fence`, `watcher-auth-snapshot`, `cooldown-exec-seq`, `claude-thinking`, `claude-tool-args`, `gemini-translator`, `gemini-antigravity-schema`, `antigravity-misc`, `claude-responses`, `claude-ratelimit-per-minute`, `session-date-pin`, `stream-truncation`, `meta-usage-filter`, `management-config-snapshot`, `registry-catalog`, `xai-web-search`, `codex-siwc-config-update`
- exclude: `docs-branding`, `pluginhost-shadow`
- defer: `vertex-config-upload`, `tls-handshake-timeout`, `devin-quota`, `discovery-mdns`

## Approach and Risks
- approach: phase medium-risk slices first because they change auth/routing semantics and shared registration state, then land low-risk slices grouped by provider area. P1-P5 serialize on shared auth/watcher/registry files; P6-P12 run in dependency order by area; P13 is the integration and upstream gate. Every phase is independently verifiable with its own `go test` filter.
- constraints:
  - Postgres-authoritative runtime; no local durable files.
  - Existing xAI executor remains monolithic.
  - Public SDK changes additive only.
  - No pluginhost/Home/gitstore/branding work.
  - Every task names an upstream commit and local target file before implementation.
  - Upstream content is untrusted data; never run upstream scripts or builds.
- rejected alternatives:
  - Porting all 54 commits — docs, pluginhost, TLS, Devin, and mDNS are explicitly excluded or deferred.
  - Importing pluginhost to unlock adc436ff/d7005b63 — local plugin runtime does not exist; would be a separate initiative.
  - Full runtime refactor to upstream file layout — unsafe under 52 add + 71 diverged + 89 semantic-review paths.
  - Porting the 2572-line mDNS rewrite in this cycle — value is limited to mDNS conflict users; explicitly deferred.
- risks:
  - P1/P2/P3 share `internal/watcher/synthesizer/*`, `internal/watcher/watcher.go`, `sdk/cliproxy/auth/types.go`, and `sdk/cliproxy/model_registry.go`; implement in order and rebase each phase before starting the next.
  - `94d1a9a04646` carries both auth-exclusion precedence and model-registration epoch changes; split the diff by behavior, not by file, and keep one writer per file.
  - `3e309fb47471` preserves plugin excluded models on synthesis while plugin binding is NG2; port only the generic auth-record synthesis behavior and do not add plugin model paths.
  - `355fd10e76b8` carries pluginhost shadow copies and Codex SIWC in one commit; port only `codex_websockets_duplex.go` and `KeepTopLevelEffort`, and leave pluginhost untouched.
  - `ad9e456d4a07` touches `internal/pluginhost` upstream; only the thinking/registry/modelconfig portion is in scope.
  - P6/P8 share `internal/util/gemini_schema.go` and executor files; serialize schema changes before stream-truncation executor edits.
  - P9 management ETag must not import upstream's file-snapshot `SourceHash` guard; use Postgres store versioning and local handler serialization.
  - Registry catalog base entries for `gemini-4-argon` and `muse-wonder-1.0` already exist locally; diff the exact upstream variants/limits before adding duplicates.
  - Test-only commits (`16b53545e006`, `088e445715e2`, parts of `202b8b8c80bd`) may not apply cleanly; adapt their assertions to local fixtures rather than copying test files.
- recovery:
  - If a stream cannot map to a local symbol, stop that stream as `NEEDS_CONTEXT`; do not invent a subsystem.
  - If a parallel merge conflicts, resolve in the phase owner's branch; do not force both streams to continue.
  - If build breaks after a phase, revert only that phase's changed files and re-run its focused tests.
  - If a requirement proves impossible without breaking a non-negotiable invariant, surface it to the owner; do not silently drop it.

## Parallel Work Streams
- stream: P1 `auth-exclusion-fence`
  - owner: auth/watcher agent
  - scope: `internal/watcher/synthesizer/**`, `internal/api/handlers/management/auth_files.go`, `sdk/cliproxy/model_registry.go`, `internal/registry/model_registry.go`
  - depends_on: none
  - inputs: R1, R2
- stream: P2 `watcher-snapshot`
  - owner: watcher agent
  - scope: `internal/watcher/**`, `sdk/cliproxy/{watcher,types}.go`, `sdk/cliproxy/auth/types.go`
  - depends_on: P1
  - inputs: R3
- stream: P3 `cooldown-seq`
  - owner: auth agent
  - scope: `sdk/cliproxy/auth/conductor.go`, `sdk/cliproxy/auth/types.go`
  - depends_on: P2
  - inputs: R4
- stream: P4 `claude-thinking`
  - owner: claude/thinking agent
  - scope: `internal/thinking/**`, `internal/modelconfig/**`, `internal/registry/model_registry.go`, `internal/runtime/executor/claude_*`, Claude translators
  - depends_on: P1 (registry file), P3 (auth types) where touched
  - inputs: R5
- stream: P5 `claude-tool-args`
  - owner: claude/translator agent
  - scope: `internal/util/translator.go`, `internal/translator/antigravity/claude/**`, Claude translator call sites
  - depends_on: P4
  - inputs: R6
- stream: P6 `gemini-antigravity`
  - owner: gemini/antigravity agent
  - scope: `internal/translator/gemini/**`, `internal/translator/antigravity/**`, `internal/util/gemini_schema.go`, `internal/misc/**`, `sdk/auth/antigravity.go`, `internal/runtime/executor/helps/gemini_content_turns.go`
  - depends_on: P5 (shared `internal/util` and translator ownership)
  - inputs: R7, R8, R9
- stream: P7 `claude-responses-session`
  - owner: claude/runtime agent
  - scope: `internal/translator/claude/openai/responses/**`, `internal/runtime/executor/helps/claude_ratelimit.go`, `internal/runtime/executor/helps/claude_diagnostics.go`, `internal/config/parse.go`
  - depends_on: P4, P5
  - inputs: R10, R11, R12
- stream: P8 `runtime-stream-meta`
  - owner: runtime agent
  - scope: `sdk/api/handlers/claude/code_handlers.go`, `sdk/api/handlers/openai/openai_responses_handlers.go`, `internal/runtime/executor/{gemini,kimi,openai_compat,gemini_vertex,claude_executor_diagnostics,meta}_*`
  - depends_on: P6, P7
  - inputs: R13, R14
- stream: P9 `management-etag`
  - owner: management agent
  - scope: `internal/api/handlers/management/**`, `internal/config/config.go`, `internal/util/provider.go`, `internal/watcher/config_reload.go`
  - depends_on: P1, P2
  - inputs: R15
- stream: P10 `registry-catalog`
  - owner: registry agent
  - scope: `internal/registry/**`
  - depends_on: P4, P6
  - inputs: R16
- stream: P11 `xai-web-search`
  - owner: xai agent
  - scope: `internal/runtime/executor/xai_executor.go`
  - depends_on: P8
  - inputs: R17
- stream: P12 `codex-siwc`
  - owner: codex agent
  - scope: `internal/runtime/executor/codex_websockets_duplex.go`, `sdk/cliproxy/executor/types.go`, `internal/thinking/suffix.go`
  - depends_on: P8
  - inputs: R18
- stream: P13 `integration-gate`
  - owner: integration agent
  - scope: build/test orchestration only; no production code edits except merge-conflict resolution
  - depends_on: P1-P12
  - inputs: R19, R20, R21, R22, R23, R24

## Phases and Verification
- planning_status: planned
- phases:
  - phase_slug: v8033-p1-auth-exclusion-fence
    story_id: story-20261019-v8033-p1
    status: planned
    risk: medium
    goal: Restore per-account excluded-model priority, tolerate invalid excluded values, and add model-registration publication fencing/reconciliation.
    upstream_sources:
      - v8.0.29 08e5762a9a3c
      - v8.0.30 5fc580b1d72d
      - v8.0.33 b015f208d28c
      - v8.0.33 94d1a9a04646
      - v8.0.33 1a28c61e5632
      - v8.0.33 3e309fb47471
      - v8.0.33 bb4282471a88
      - v8.0.33 f75ea0733657
      - v8.0.33 16b53545e006
    depends_on: none
    requirements: R1, R2
    allowed_surfaces:
      - `internal/watcher/synthesizer/helpers.go`, `config.go`, `file.go`
      - `internal/api/handlers/management/auth_files.go`
      - `sdk/cliproxy/model_registry.go`
      - `internal/registry/model_registry.go`
      - associated `_test.go` files under those packages
    avoided_surfaces:
      - `internal/pluginhost/**`, `sdk/pluginapi/**`
      - branding/README
    tasks:
      - task: [R1] Change `ApplyAuthExcludedModelsMeta` at `internal/watcher/synthesizer/helpers.go:58` so per-account excluded models override, rather than merge with, the global list; keep invalid values tolerated on load per `1a28c61e5632`.
      - task: [R1] Adapt `b015f208d28c`: restore credential exclusion priority in management auth-file extraction and synthesis; port only the generic auth-record behavior from `3e309fb47471`.
      - task: [R1] Update `config.example.yaml` comment for per-account excluded-models priority (`bb4282471a88`); no runtime behavior change.
      - task: [R2] Add epoch-checked publication fencing in `sdk/cliproxy/model_registry.go:21-28` and `internal/registry/model_registry.go`; reject superseded registrations per `08e5762a9a3c`.
      - task: [R2] Reconcile latest auth state when a registration is superseded mid-flight per `f75ea0733657`; adapt the lifecycle stabilization assertions from `16b53545e006`.
    checks:
      - check: `go test ./internal/watcher/... ./sdk/cliproxy/... ./internal/registry/... ./internal/api/handlers/management/... -run 'Excluded|Registration|ModelRegistry|Epoch|Superseded' -count=1`
      - check: `go test ./sdk/cliproxy/... -run Excluded -count=1`
  - phase_slug: v8033-p2-watcher-auth-snapshot
    story_id: story-20261019-v8033-p2
    status: planned
    risk: medium
    goal: Track auth file removals and bind single-file snapshots so stale snapshots cannot delete or overwrite later persisted auth state.
    upstream_sources:
      - v8.0.29 5fc580b1d72d
      - v8.0.33 c4c1d43eaf8c
    depends_on: v8033-p1-auth-exclusion-fence
    requirements: R3
    allowed_surfaces:
      - `internal/watcher/watcher.go`, `clients.go`, `dispatcher.go`, `config_reload.go`
      - `internal/watcher/synthesizer/context.go`, `file.go`
      - `sdk/cliproxy/watcher.go`, `sdk/cliproxy/types.go`, `sdk/cliproxy/auth/types.go`
    avoided_surfaces:
      - auth exclusion precedence logic from P1 beyond the shared synthesizer interface
      - `internal/pluginhost/**`
    tasks:
      - task: [R3] Extend `AuthUpdate` at `internal/watcher/watcher.go:78` with file-removal tracking and adapt the synthesizer file snapshot binding (`BindFileSnapshot`) from `5fc580b1d72d`.
      - task: [R3] Add snapshot generation/binding in `sdk/cliproxy/auth/types.go` and reject stale persistence per `c4c1d43eaf8c`; keep Postgres as the authoritative store.
      - task: [R3] Wire the new snapshot through `internal/watcher/dispatcher.go`, `clients.go`, `synthesizer/context.go`, and `sdk/cliproxy/watcher.go` without changing provider route semantics.
    checks:
      - check: `go test ./internal/watcher/... ./sdk/cliproxy/... ./sdk/cliproxy/auth/... -run 'Snapshot|Removal|Stale|Watcher' -count=1`
  - phase_slug: v8033-p3-cooldown-exec-seq
    story_id: story-20261019-v8033-p3
    status: planned
    risk: medium
    goal: Carry an execution sequence so stale in-flight successes cannot clear active model or credential cooldowns.
    upstream_sources:
      - v8.0.30 35af74f92721
    depends_on: v8033-p2-watcher-auth-snapshot
    requirements: R4
    allowed_surfaces:
      - `sdk/cliproxy/auth/conductor.go`
      - `sdk/cliproxy/auth/types.go`
      - `sdk/cliproxy/auth/*_test.go`
    avoided_surfaces:
      - watcher snapshot logic
      - pluginhost
    tasks:
      - task: [R4] Add `ExecutionSeq` to the auth `Result` and propagate it through selection/execution/completion in `sdk/cliproxy/auth/conductor.go` per `35af74f92721`.
      - task: [R4] Drop cooldown-clearing effects from results whose sequence predates the active cooldown; preserve current fairness and model-lock behavior.
    checks:
      - check: `go test ./sdk/cliproxy/auth/... -run 'Cooldown|ExecutionSeq|Stale|Result' -count=1`
  - phase_slug: v8033-p4-claude-thinking
    story_id: story-20261019-v8033-p4
    status: planned
    risk: medium
    goal: Support preserved-thinking block binding, adaptive fallbacks, `between_tools` disable type, and Claude header overrides.
    upstream_sources:
      - v8.0.30 9db0729f5fef
      - v8.0.30 4e8a4125ace0
      - v8.0.32 911f49cbd49f
      - v8.0.32 7604d3df3e62
      - v8.0.33 ad9e456d4a07
      - v8.0.33 e13bc9a1f0ca
    depends_on: v8033-p1-auth-exclusion-fence
    requirements: R5
    allowed_surfaces:
      - `internal/thinking/validate.go`, `apply.go`, `provider/claude/apply.go`
      - `internal/modelconfig/model_info.go`
      - `internal/registry/model_registry.go`
      - `internal/config/config.go`, `parse.go`, `claude_header_defaults_test.go`
      - `internal/runtime/executor/claude_executor_diagnostics.go`, `claude_executor.go`
      - Claude request translators under `internal/translator/claude/**`
    avoided_surfaces:
      - `internal/pluginhost/**` (upstream `ad9e456d4a07` touches it)
      - branding/README
    tasks:
      - task: [R5] Add `DisableType`/`between_tools` handling at `internal/thinking/validate.go:162`, `internal/thinking/provider/claude/apply.go:72`, and `internal/registry/model_registry.go:165` per `ad9e456d4a07`.
      - task: [R5] Synthesize adaptive thinking block binding for preserved-thinking models in `internal/runtime/executor/claude_executor_diagnostics.go` per `911f49cbd49f` and `e13bc9a1f0ca`; keep beta policy additive.
      - task: [R5] Add Claude header overrides (`AnthropicBetaStrip`, `AnthropicBetaAppend`, `InjectDiagnostics`, `ClientRequestID` on/off/passthrough) at `internal/config/config.go:264` and the request path near `claude_executor.go:2104`, per `9db0729f5fef`; adapt tests from `4e8a4125ace0`.
      - task: [R5] Port the session-date pin hooks only where they feed thinking/continuity; the full retention window is P7 (R12).
    checks:
      - check: `go test ./internal/thinking/... ./internal/modelconfig/... ./internal/registry/... ./internal/config/... ./internal/runtime/executor/ -run 'Thinking|Adaptive|BetweenTools|ClaudeHeader|RequestID|Preserved' -count=1`
  - phase_slug: v8033-p5-claude-tool-args
    story_id: story-20261019-v8033-p5
    status: planned
    risk: medium
    goal: Clean and normalize Claude tool arguments against original schemas, including tool-name mapping and cached-token usage correction.
    upstream_sources:
      - v8.0.29 912a038b8b39
      - v8.0.32 202b8b8c80bd
    depends_on: v8033-p4-claude-thinking
    requirements: R6
    allowed_surfaces:
      - `internal/util/translator.go`
      - `internal/translator/antigravity/claude/antigravity_claude_response.go`
      - Claude/Gemini response translator call sites
      - `internal/util/*_test.go`
    avoided_surfaces:
      - `internal/pluginhost/**`
      - registry catalog (P10)
    tasks:
      - task: [R6] Add `cleanClaudeToolObject`/`MapClaudeToolName` behavior behind existing `internal/util/translator.go` helpers per `202b8b8c80bd`; port behavior, not the 1282-line upstream shape.
      - task: [R6] Apply the cleaner at Claude tool-call emission sites, including `internal/translator/antigravity/claude/antigravity_claude_response.go` and Gemini→Claude response translation, preserving unknown fields and namespaced names.
      - task: [R6] Correct cached-token usage math per `912a038b8b39` without changing total-token accounting for non-cached responses.
    checks:
      - check: `go test ./internal/util/... ./internal/translator/... -run 'ClaudeTool|ToolArgs|CachedToken|Sanitize' -count=1`
  - phase_slug: v8033-p6-gemini-antigravity
    story_id: story-20261019-v8033-p6
    status: planned
    risk: low
    goal: Land Gemini translator, Gemini/Antigravity schema, and Antigravity misc fixes as one provider-area phase.
    upstream_sources:
      - v8.0.29 5efe97582c05
      - v8.0.29 7f2f17d76fe1
      - v8.0.29 582570076282
      - v8.0.29 8fb633c9fa21
      - v8.0.29 a19a8ecde3eb
      - v8.0.30 e9ba5c507f46
      - v8.0.30 ceb4397f1e83
      - v8.0.33 a43026d75196
      - v8.0.33 acaab599988f
      - v8.0.33 b8d4e14a8e65
    depends_on: v8033-p5-claude-tool-args
    requirements: R7, R8, R9
    allowed_surfaces:
      - `internal/translator/gemini/claude/gemini_claude_response.go`
      - `internal/translator/antigravity/**`
      - `internal/util/gemini_schema.go`
      - `internal/misc/antigravity_version.go`
      - `sdk/auth/antigravity.go`
      - `internal/runtime/executor/helps/gemini_content_turns.go`
      - `internal/runtime/executor/aistudio_executor.go`
    avoided_surfaces:
      - `internal/util/translator.go` (P5 owns writes)
      - `internal/pluginhost/**`
    tasks:
      - task: [R7] Treat `MALFORMED_FUNCTION_CALL` as retryable in `internal/translator/gemini/claude/gemini_claude_response.go:53` per `7f2f17d76fe1`; keep tool-use IDs unique across turns per `582570076282`.
      - task: [R8] Extend `internal/util/gemini_schema.go:40,948` with multi-branch Claude-tool union flattening (`5efe97582c05`), `discriminator` strip (`e9ba5c507f46`), `contentMediaType`/`contentEncoding` strip (`a43026d75196`), and the linear declaration walk from `b8d4e14a8e65`.
      - task: [R9] Raise `antigravityFallbackVersion` from 2.9.1 to 2.23.0 at `internal/misc/antigravity_version.go:23` (`8fb633c9fa21`), expose UA helpers from `sdk/auth/antigravity.go:273` (`acaab599988f`), and drop trailing thought-only turns in `helps/gemini_content_turns.go` (`ceb4397f1e83`).
      - task: [R7] Add the upstream Gemini 3.8 model variants/test cases only if they are catalog-only; otherwise defer the catalog part to P10.
    checks:
      - check: `go test ./internal/translator/gemini/... ./internal/translator/antigravity/... ./internal/util/... ./internal/misc/... ./sdk/auth/... ./internal/runtime/executor/ -run 'Gemini|Antigravity|Schema|Malformed|ToolUse|ContentMedia|Discriminator|Thought' -count=1`
  - phase_slug: v8033-p7-claude-responses-session
    story_id: story-20261019-v8033-p7
    status: planned
    risk: low
    goal: Land Claude Responses incomplete-status/pinned-body, per-minute rate-limit narrowing, and session-date pin retention.
    upstream_sources:
      - v8.0.30 59930ba36a82
      - v8.0.31 7a511e4db78a
      - v8.0.32 7604d3df3e62
      - v8.0.33 db1bab011d98
    depends_on: v8033-p4-claude-thinking, v8033-p5-claude-tool-args
    requirements: R10, R11, R12
    allowed_surfaces:
      - `internal/translator/claude/openai/responses/claude_openai-responses_request.go`
      - `internal/translator/claude/openai/responses/claude_openai-responses_response.go`
      - `internal/runtime/executor/helps/claude_ratelimit.go`
      - `internal/runtime/executor/helps/claude_diagnostics.go`
      - `internal/config/parse.go`
    avoided_surfaces:
      - P4 thinking apply logic beyond the shared diagnostics hook
      - `internal/pluginhost/**`
    tasks:
      - task: [R10] Map incomplete OpenAI Responses tool status to `tool_result is_error` in `claude_openai-responses_request.go:749` per `59930ba36a82`; pin the validated request body per stream in `claude_openai-responses_response.go:401` per `7a511e4db78a`.
      - task: [R11] Narrow the Claude per-minute cooldown when a short `Retry-After` is present, starting from `helps/claude_ratelimit.go:111`, per `db1bab011d98`.
      - task: [R12] Add a retention window to the legacy continuity-scoped pin at `helps/claude_diagnostics.go:25,144` and parse it in `internal/config/parse.go` per `7604d3df3e62`; preserve across credential switches.
    checks:
      - check: `go test ./internal/translator/claude/openai/responses/... ./internal/runtime/executor/... ./internal/config/... -run 'Responses|Incomplete|Pinned|RateLimit|RetryAfter|SessionDate|Pin' -count=1`
  - phase_slug: v8033-p8-runtime-stream-meta
    story_id: story-20261019-v8033-p8
    status: planned
    risk: low
    goal: Detect stream truncation across executors/handlers and filter Meta subscription-usage telemetry.
    upstream_sources:
      - v8.0.29 7f2f17d76fe1
      - v8.0.32 088e445715e2
      - v8.0.33 4f2e52709dcd
      - v8.0.33 3a0722984ab7
    depends_on: v8033-p6-gemini-antigravity, v8033-p7-claude-responses-session
    requirements: R13, R14
    allowed_surfaces:
      - `sdk/api/handlers/claude/code_handlers.go`
      - `sdk/api/handlers/openai/openai_responses_handlers.go`
      - `internal/runtime/executor/gemini_executor.go`, `gemini_vertex_executor.go`, `kimi_executor.go`, `openai_compat_executor.go`
      - `internal/runtime/executor/claude_executor_diagnostics.go`
      - `internal/runtime/executor/meta_executor_stream.go`, `meta_executor_execute.go`
    avoided_surfaces:
      - provider route semantics
      - `internal/pluginhost/**`
    tasks:
      - task: [R13] Add finish-signal tracking to `forwardClaudeStream` at `sdk/api/handlers/claude/code_handlers.go:296` and to executor stream loops (`gemini_executor.go:229`, vertex, kimi, openai-compat) per `4f2e52709dcd`; convert silent EOF into an explicit upstream-truncation error/502.
      - task: [R14] Filter `response.subscription_usage` from Meta SSE streams and responses in `meta_executor_stream.go:21` and `openai_responses_handlers.go` per `3a0722984ab7`; handle client disconnect gracefully after terminal events per `088e445715e2`.
      - task: [R13] Keep the error surface stable for non-truncated clean EOF and for providers that legitimately omit a finish signal.
    checks:
      - check: `go test ./sdk/api/handlers/claude/... ./sdk/api/handlers/openai/... ./internal/runtime/executor/ -run 'StreamTrunc|Truncat|Finishing|Meta|SubscriptionUsage|Disconnect' -count=1`
  - phase_slug: v8033-p9-management-etag
    story_id: story-20261019-v8033-p9
    status: planned
    risk: low
    goal: Serialize management config mutations and add ETag/If-Match optimistic concurrency without importing the file-snapshot guard.
    upstream_sources:
      - v8.0.30 1631875b9ac3
      - v8.0.31 fa68946ad572
    depends_on: v8033-p1-auth-exclusion-fence, v8033-p2-watcher-auth-snapshot
    requirements: R15
    allowed_surfaces:
      - `internal/api/handlers/management/config_lists.go`
      - `internal/api/handlers/management/config_basic.go`
      - `internal/api/handlers/management/handler.go`
      - `internal/config/config.go`
      - `internal/util/provider.go`
      - `internal/watcher/config_reload.go`
    avoided_surfaces:
      - local YAML/file authoritative paths
      - upstream `SourceHash` file-snapshot guard (not applicable to Postgres)
    tasks:
      - task: [R15] Wrap the mutate+save path in `config_lists.go:57` with `h.mu` and return an ETag derived from the Postgres store version, per `1631875b9ac3`.
      - task: [R15] Add `If-Match`/stale-snapshot checks to the v8 config endpoints in `handler.go:128,352` and `config_basic.go`, per `fa68946ad572`; reject stale writes with the upstream status/contract.
      - task: [R15] Add the `value` mask for the management delete-by-key path at `internal/util/provider.go:285` per `fa68946ad572`.
    checks:
      - check: `go test ./internal/api/handlers/management/... ./internal/util/... ./internal/config/... ./internal/watcher/... -run 'ETag|IfMatch|Config|Provider|Stale' -count=1`
  - phase_slug: v8033-p10-registry-catalog
    story_id: story-20261019-v8033-p10
    status: planned
    risk: low
    goal: Update the registry catalog and Claude family lookup without duplicating entries already ported in the v8.0.28 cycle.
    upstream_sources:
      - v8.0.29 f9ac397be288
      - v8.0.29 a19a8ecde3eb
      - v8.0.30 7459255675f7
      - v8.0.33 ad9e456d4a07
    depends_on: v8033-p4-claude-thinking, v8033-p6-gemini-antigravity
    requirements: R16
    allowed_surfaces:
      - `internal/registry/model_definitions.go`
      - `internal/registry/models/models.json`
      - `internal/registry/*_test.go`
    avoided_surfaces:
      - runtime executors
      - `internal/pluginhost/**`
    tasks:
      - task: [R16] Add `muse-wonder-full-1.0` variants and updated context limits per `f9ac397be288`; diff against existing `muse-wonder-1.0` entries first.
      - task: [R16] Reconcile `gemini-4-argon` definitions/limits per `7459255675f7` and add the missing Gemini 3.8 variants per `a19a8ecde3eb`; keep existing base entries.
      - task: [R16] Add `claudeFamilyModelIDs` date/bracket fallback at `internal/registry/model_definitions.go:564` per `ad9e456d4a07`.
      - task: [R16] Validate `internal/registry/models/models.json` with `python3 -m json.tool` and run the catalog tests.
    checks:
      - check: `go test ./internal/registry/... -count=1`
      - check: `python3 -m json.tool internal/registry/models/models.json >/dev/null`
  - phase_slug: v8033-p11-xai-web-search
    story_id: story-20261019-v8033-p11
    status: planned
    risk: low
    goal: Drop xAI Responses web_search options the upstream API rejects while preserving the monolithic executor.
    upstream_sources:
      - v8.0.30 bbfd7886ee13
    depends_on: v8033-p8-runtime-stream-meta
    requirements: R17
    allowed_surfaces:
      - `internal/runtime/executor/xai_executor.go`
      - `internal/runtime/executor/xai_executor_test.go`
    avoided_surfaces:
      - upstream xAI file split
      - `internal/pluginhost/**`
    tasks:
      - task: [R17] Strip unsupported `web_search` options from xAI Responses requests at the request-building path around `xai_executor.go:44` per `bbfd7886ee13`; keep the executor monolithic and leave supported search tools intact.
    checks:
      - check: `go test ./internal/runtime/executor/ -run 'XAI|WebSearch|Responses' -count=1`
  - phase_slug: v8033-p12-codex-siwc
    story_id: story-20261019-v8033-p12
    status: planned
    risk: low
    goal: Strip Codex SIWC configuration_update payloads and carry KeepTopLevelEffort, without the pluginhost portion of the carrying commit.
    upstream_sources:
      - v8.0.31 355fd10e76b8
    depends_on: v8033-p8-runtime-stream-meta
    requirements: R18
    allowed_surfaces:
      - `internal/runtime/executor/codex_websockets_duplex.go`
      - `sdk/cliproxy/executor/types.go`
      - `internal/thinking/suffix.go`
      - `internal/runtime/executor/codex_websockets_duplex_test.go`
    avoided_surfaces:
      - `internal/pluginhost/**`, `internal/runtime` plugin shadow copies
    tasks:
      - task: [R18] Strip `configuration_update` events in `codex_websockets_duplex.go:283` per the non-plugin part of `355fd10e76b8`; preserve usage telemetry and existing stream ordering.
      - task: [R18] Add `KeepTopLevelEffort` to `sdk/cliproxy/executor/types.go` and thread it through `internal/thinking/suffix.go`; keep the SDK change additive.
    checks:
      - check: `go test ./internal/runtime/executor/ -run 'Codex|SIWC|ConfigurationUpdate|KeepTopLevelEffort' -count=1`
  - phase_slug: v8033-p13-integration-gate
    story_id: story-20261019-v8033-p13
    status: planned
    risk: high
    goal: Run full touched-package tests, build, scope audit, and the upstream final gate; pin any newer delta.
    upstream_sources:
      - v8.0.33 2422c6b0f5fe (checkpoint / final re-resolve)
    depends_on: v8033-p1-auth-exclusion-fence, v8033-p2-watcher-auth-snapshot, v8033-p3-cooldown-exec-seq, v8033-p4-claude-thinking, v8033-p5-claude-tool-args, v8033-p6-gemini-antigravity, v8033-p7-claude-responses-session, v8033-p8-runtime-stream-meta, v8033-p9-management-etag, v8033-p10-registry-catalog, v8033-p11-xai-web-search, v8033-p12-codex-siwc
    requirements: R19, R20, R21, R22, R23, R24
    allowed_surfaces:
      - merge-conflict resolution only; no new product behavior
    avoided_surfaces:
      - silent scope growth
      - excluded/deferred slices
    tasks:
      - task: [R19-R22] Run the full touched-package test set and `make build`; verify Postgres authority, additive SDK, monolithic xAI, and no pluginhost/branding drift by diff review.
      - task: [R23] Re-resolve the latest stable upstream release, fetch `refs/upstream-checkpoints/cliproxyapi/{tag}`, refresh the checkpoint and ledger, and record the delta.
      - task: [R24] If a newer release exists, either include it through an approved `brainstorm refine` or pin it as an explicit follow-up; do not silently widen this plan.
      - task: [R19-R22] Inspect `git diff --check` and confirm no `web/**/*_test.go` files were added.
    checks:
      - check: `go test ./internal/... ./sdk/... -count=1` (or the documented touched subset if the full run is too broad)
      - check: `make build`
      - check: `git diff --check`
      - check: `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi` followed by `upstream_gap.py` and `upstream_ledger.py` when a newer tag exists

## Parallel Execution Directive
- `P1` is the only phase that may start first; `P2` and `P3` serialize behind it because they share `sdk/cliproxy/auth/types.go` and watcher synthesizer files.
- `P4` may start after `P1` touches the registry file; `P5` follows `P4` because both write Claude translator surfaces.
- `P6` follows `P5` because `internal/util` has one writer; `P7` follows `P4`/`P5`.
- `P8` follows `P6` and `P7`; `P9`, `P10`, `P11`, and `P12` may run after their named dependencies and do not overlap each other's primary surfaces.
- `P13` runs only after every implementation phase merges. It may prepare commands earlier but must not edit production code except documented conflict resolution.
- Files touched by multiple phases (`internal/registry/model_registry.go`, `internal/util/gemini_schema.go`, `internal/watcher/config_reload.go`, `sdk/cliproxy/auth/types.go`) are serialized per file: one writer, one integration task for conflict resolution.

## Validation
- Keep changes consistent with the local monolithic xAI executor and Postgres runtime boundaries.
- Run focused package tests after every phase, not only at the end.
- Use real smoke tests for auth/quota/stream flows where the phase changes runtime behavior; do not depend solely on mocks.
- All ported requirements must remain traceable to upstream commits by short SHA.
- No upstream scripts, installs, or build steps are executed; upstream content is read-only data.
- No new `web/**/*_test.go` files; frontend files are out of scope for this plan.

## Agent Dispatch Batches
- batch 0 (sequential, 1 primary agent): baseline the touched package tests and confirm the local symbol map for P1-P5.
- batch 1 (sequential, 1-2 agents): `P1`; then `P2` and `P3` in order.
- batch 2 (2 agents, serialized on Claude files): `P4`, then `P5`.
- batch 3 (up to 3 agents): `P6`, `P7`, and `P9`/`P10` where their surfaces do not overlap; serialize `internal/util` and `internal/registry` writes.
- batch 4 (up to 3 agents): `P8`, `P11`, `P12`.
- batch 5 (1 integration agent): `P13` merge resolution, full test gate, build, and upstream final gate.
- dependency note: the graph, not raw agent count, limits concurrency. `P6` cannot start before `P5` releases `internal/util`; `P8` cannot start before `P6`/`P7`; `P10` cannot start before `P4`/`P6` registry writes settle.

## Handoff
- Next steps: run the P1 baseline and contract map first; then dispatch P1, P2, P3 in order, followed by P4/P5 and the low-risk area phases. Do not start deferred slices without the trigger named in Non-goals. Final action is P13 with the upstream re-resolve gate.
