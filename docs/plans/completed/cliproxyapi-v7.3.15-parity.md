---
id: plan-20260923-v7315
type: plan
intake_id: intake-20260923-v7315
lane: high-risk
status: active
created: 2026-09-23
updated: 2026-09-23
---

# Plan: CLIProxyAPI v7.3.11..v7.3.15 targeted parity

## Outcome
- result: llmhub ports the user-approved v7.3.11..v7.3.15 capability slices (11 slices, ~15 upstream commits) as independently verifiable semantic ports behind existing registry, executor, config, management, and translator interfaces — without wholesale merge, pluginhost, request-proxy surface, perf rewrites, or docs/test churn.
- success_signals:
  - Registry counting: a client that is both quota-exceeded and suspended counts once in model availability; regression tests equivalent to upstream `model_registry_credential_quota_regression_test.go` pass.
  - Claude fingerprint: outgoing `/v1/messages` beta headers match Claude Code 2.1.280 wire order, including `mid-conversation-tool-changes` and the eight feature-gated betas, emitted only under their capability/body gates.
  - Embedded catalog: `models.json` carries `claude-opus-5-5`, `gpt-6-luna`, `gpt-6-sol`, `grok-4.7`, `grok-4.7-build-fast`, no `gpt-5.3-codex-spark`; codex client catalog advertises `0.155.0` and marshals compact without dropping required null fields.
  - Management: `priority` is patchable on gemini/interactions/openai-compat/vertex/codex credential endpoints; `disable-codex-cloaking` is settable per Codex credential and wins over global; `trusted-proxies` config resolves forwarded client IPs through Gin.
  - Translators: Responses requests with string `input` and `input_video`/`video_url` parts translate correctly; codex tool schemas with octal NUL `\0` patterns pass strict upstream validators; Gemini/Antigravity schemas declaring uppercase `ARRAY`/`OBJECT` types keep/receive correct `items`.
  - Each accepted slice lands with focused tests citing its upstream commit(s) and local symbol.
  - `go test` on touched packages, `make build`, `git diff --check`, `gofmt` clean at gates.
  - Newer-than-v7.3.15 upstream delta is pinned as follow-up, never silent scope growth.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v7.3.15` at `673131f57484` / `refs/upstream-checkpoints/cliproxyapi/v7.3.15`; `scope_policy` (11 include / 3 exclude) is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.3.11..v7.3.15.json` — 116 paths (match 0, baseline 1, upstream-add-absent 23, diverged-absent 45, upstream-delete-present-local 0, semantic-review 47).
  - `docs/upstream/cliproxyapi-ledger-v7.3.11..v7.3.15.md` — 27 non-merge commits across 4 releases.
  - `docs/plans/completed/cliproxyapi-v7.3.10-parity.md` and prior parity plans — their slices must not regress; standing exclusions still hold.
  - `CLAUDE.md` / `docs/WORKFLOW.md` — Postgres-authoritative store, SDK additive-only, no `web/**/*_test.go`, llmhub branding.
- rejected_alternatives:
  - Wholesale merge of the 116-path delta — diverged-absent/semantic-review dominate; upstream splits (`codex_executor_auth.go`, `claude_executor_request.go`, `helps/codex_tool_schema.go`) map onto local monoliths and the translator-side schema path.
  - Porting upstream's `internal/util/gemini_schema.go` sjson sanitizer — local rewrote it (`normalizeMalformedSchemaObjects`/`repairSchemaNode`); port the case-insensitivity semantics, not the code.
  - Cherry-picking the perf batch without benchmarks — executor internals diverged; re-implementation is a separate profiling-driven initiative.
- requirements:
  - R1 [accepted]: **Quota/suspension single-counting** — `internal/registry/model_registry.go` counts a client that is both quota-exceeded and suspended only once in `modelRegistrationAvailability` and `GetAvailableModelsByProvider`; port upstream regression tests. | source: `ed70aeaa1627` (v7.3.13)
  - R2 [accepted]: **Claude Code 2.1.280 beta surface** — local baseline bumps 2.1.258→2.1.280 (device profile UA, `checkSystemInstructionsWithSigningMode` callers); add `per-turn-control-2026-07-01`, `timing-2026-09-09`, `mid-conversation-tool-changes-2026-07-01`, `inline-tools-2026-09-15`, `mid-conversation-system-clear-at-2026-08-21`, `dangerous-tool-use-2026-09-03`, `thinking-binding-controls-2026-08-01`, `thinking-resumption-2026-07-17`, `prompt-caching-evict-2026-05-12` with upstream's exact wire order and capability/body gates; managed-beta set covers all of them. | source: `bd584a752329` `779bf317e030` (v7.3.15)
  - R3 [accepted]: **Embedded model catalog sync** — `internal/registry/models/models.json` gains `claude-opus-5-5`, `gpt-6-luna` (all codex tiers), `gpt-6-sol`, `grok-4.7`, `grok-4.7-build-fast` with upstream field values; drops `gpt-5.3-codex-spark`; `codex_client_models.json` bumps `minimal_client_version` to `0.155.0`. | source: `130c879206ba` `b9b50a83cb9d` `94b7cc2ee0f0` `2430354330af` `937ebb8f11f7` `fc914b9debb9` (v7.3.12–v7.3.15)
  - R4 [accepted]: **Codex client catalog compact marshal** — `internal/client/codex/models/models.go` emits compact single-line JSON without HTML escaping, uses compact base instructions for non-template models to stay under the 1MiB limit, and preserves `apply_patch_tool_type`/`upgrade`/`availability_nux` as explicit nulls. | source: `673131f57484` (v7.3.15)
  - R5 [accepted]: **Priority patch on credential endpoints** — `PATCH` handlers for gemini, interactions, openai-compat, vertex, and codex credentials accept `priority` the way `PatchClaudeKey` already does (`config_lists.go:376`); omitted preserves, explicit replaces. | source: `cc77410866c2` (v7.3.13)
  - R6 [accepted]: **Per-credential disable codex cloaking** — `config.CodexKey` (or local equivalent) gains `disable-codex-cloaking` overriding global `codex.disable-codex-cloaking`; resolution order matches upstream: auth attribute → credential field → global; management PATCH accepts it; watcher diff/synthesizer preserve it. | source: `f351924f42cb` (v7.3.13)
  - R7 [accepted]: **Trusted proxies** — `trusted-proxies` config (IP/CIDR validated) applies to the Gin engine for client-IP resolution and records `ResolvedClientIP` in request metadata + usage queue details. | source: `a962b77d4383` (v7.3.14)
  - R8 [accepted]: **Video input in Responses translation** — `input_video`/`video_url` content parts convert to chat-completion video parts preserving URL and processing options in the openai→openai-compat path. | source: `4b5adbbe9a05` (v7.3.13)
  - R9 [accepted]: **String `input` in responses→claude** — `internal/translator/claude/openai/responses/claude_openai-responses_request.go` appends a string `input` as a user text message instead of dropping it. | source: `b989e34881c7` (v7.3.13)
  - R10 [accepted]: **Uppercase schema type handling** — `isArrayDeclaredType`/`isNonObjectDeclaredType` in `internal/util/gemini_schema.go` match types case-insensitively so `type: "ARRAY"`/`"OBJECT"` schemas keep and receive `items`/property repair; port upstream regression test shapes. | source: `2eb8dd11d248` (v7.3.12)
  - R11 [accepted]: **Octal NUL pattern strip for codex tool schemas** — regex `pattern` attributes containing `\0` escapes are stripped before upstream submission, in whichever local surface owns codex tool-schema normalization (upstream target `helps/codex_tool_schema.go` has no local file — locate the real normalization site first; verify whether the earlier `\p{...}` strip was ever ported). | source: `320100ecf767` (v7.3.13)
  - R12 [accepted]: **Final gate** — re-resolve latest upstream release (`upstream_sync.py sync --slug cliproxyapi`), refresh checkpoint; pin any newer-than-v7.3.15 delta as explicit follow-up in ledger + Progress — never silent scope growth. | source: skill contract

## Non-goals
- `d582067c066f` execution-scoped request-proxy overrides — implemented upstream through pluginhost adapters; no local consumer without pluginhost. Deferred until a local writer of the context exists.
- Perf batch `56518489ce92` `cbf8318315a1` `8a6a39684d11` `a26cf2a8c2e5` `639b7f1126b4` — SSE copy avoidance, Responses tool index, translation reuse; deferred until profiling shows the cost (recorded in `scope_policy.exclude` as `perf-batch`).
- `555662940411` `6ed58a7c5548` `e01806f971b1` `bf44a7f89206` — upstream README/docs/gofmt churn (recorded as `upstream-docs-gofmt`).
- Upstream file-split refactors, pluginhost, Home/gitstore, branding — standing exclusions from prior parity cycles.
- Meta/Muse catalog changes — none exist upstream in this range; `muse-spark-1.3` remains the newest model on both sides.

## Approach and Risks
- approach: eleven semantic ports grouped into eight phases across five waves. Catalog sync first (low risk, mechanical). Wave 1 parallelizes the four independent code slices (registry counting, claude betas, schema types, translator inputs). Wave 2 handles the config/management surface (credential patch + trusted proxies, file-disjoint). Wave 3 isolates the octal-NUL port because its local call site is unmapped and may land on `codex_executor.go` (conflicts with p5). Wave 4 is the upstream final gate.
- upstream→local surface map (verified during gap review):
  - upstream `internal/runtime/executor/claude_executor_request.go` beta assembly → local `claude_executor.go` (`claudeCodeCLIBetas` :1350, constants :1305-1322) + `helps/claude_device_profile.go` UA baseline
  - upstream `internal/config/trusted_proxies.go` (new) → local `internal/config/` + `internal/api/server.go` Gin engine setup
  - upstream `internal/runtime/executor/codex_executor_{auth,request}.go` cloaking → local `codex_executor.go` (`codexUserAgentForCloaking` :42, disable path :1277) + `sdk/cliproxy/executor/context.go`
  - upstream `helps/codex_tool_schema.go` → no local file; codex schema keywords live in `internal/translator/codex/claude/codex_claude_request.go` — locate actual normalize call site in R11 task
- risks:
  - r1: beta wire-order is a cloaking fingerprint — wrong order or over-eager gating regresses upstream parity; port the upstream ordering table verbatim and cover each gate with a focused test.
  - r2: R11's normalization site is unmapped — if codex tool schemas are normalized translator-side locally, the strip must run before upstream submission, not at translation.
  - r3: `models.go` compact marshal touches llmhub-specific `cpa_capabilities` code — preserve local fields while adopting compact/null-preserving output.
- recovery: per-phase revert of touched files; upstream end-state preserved at `refs/upstream-checkpoints/cliproxyapi/v7.3.15`.

## Phases and Verification

### Wave 0 — sequential gate
- phase: `p0-catalog` — story_id: `story-20260923-p0-catalog` — status: `checked` — deps: none
  - goal: embedded catalogs match upstream v7.3.15 (R3, R4).
  - tasks:
    - p0-t1 `internal/registry/models/models.json`: add `claude-opus-5-5` (claude), `gpt-6-luna` (codex-free/team/plus/pro), `gpt-6-sol` (codex-team/plus/pro), `grok-4.7` + `grok-4.7-build-fast` (xai) copying upstream field values; remove `gpt-5.3-codex-spark` (codex-plus/pro). Source values: `git show refs/upstream-checkpoints/cliproxyapi/v7.3.15:internal/registry/models/models.json`.
    - p0-t2 `internal/registry/models/codex_client_models.json`: port upstream v7.3.15 catalog including `minimal_client_version: 0.155.0`; check whether local `cmd/fetch_codex_models` exists and sync if so.
    - p0-t3 `internal/client/codex/models/models.go`: compact single-line marshal without HTML escaping, compact base instructions for non-template models, preserve `apply_patch_tool_type`/`upgrade`/`availability_nux` as explicit null — while keeping local `cpa_capabilities`/`applyCPAWebSearchCapability` behavior.
  - surfaces: `internal/registry/models/**`, `internal/client/codex/models/**`
  - verify: `go test ./internal/registry/... ./internal/client/codex/models/... -count=1` green; `python3 -c "import json; json.load(open('internal/registry/models/models.json'))"` valid.
  - stop condition: upstream field values ambiguous → halt, diff local entry vs upstream JSON for that model id.

### Wave 1 — parallel phases (file-disjoint)
- phase: `p1-registry-count` — story_id: `story-20260923-p1-registry-count` — status: `checked` — deps: none
  - goal: quota+suspended clients counted once (R1).
  - tasks:
    - p1-t1 `internal/registry/model_registry.go`: add `quotaAndOtherSuspended` correction in `modelRegistrationAvailability` (~:1338) and `GetAvailableModelsByProvider` (~:1496) matching upstream `ed70aeaa1627`.
    - p1-t2 port upstream regression tests `model_registry_credential_quota_regression_test.go` + `model_registry_quota_refresh_regression_test.go` + `sdk/cliproxy/auth/catalog_credential_quota_regression_test.go`, adapted to local types.
  - surfaces: `internal/registry/model_registry*.go`, `sdk/cliproxy/auth/` tests
  - verify: `go test ./internal/registry/... ./sdk/cliproxy/auth/... -count=1` green.
  - stop condition: local suspension model differs enough that upstream test doesn't map → halt, re-review `ed70aeaa1627`.

- phase: `p2-claude-betas` — story_id: `story-20260923-p2-claude-betas` — status: `checked` — deps: none
  - goal: beta headers match Claude Code 2.1.280 wire order and gates (R2).
  - tasks:
    - p2-t1 `internal/runtime/executor/claude_executor.go`: add the nine new beta constants + `claudeManagedBetaSet` entries + ordering in `claudeCodeCLIBetas` exactly per `bd584a752329` (incl. legacy-model else-branch).
    - p2-t2 gate predicates `claudeIncludePerTurnControl`/`Timing`/`InlineTools`/`MidConvSystemClearAt`/`DangerousToolUse`/`ThinkingBinding`/`ThinkingResumption`/`PromptCachingEvict` ported semantically (body-field + requested + model-capability gates).
    - p2-t3 baseline 2.1.258→2.1.280: `helps/claude_device_profile.go` UA constant, `claude_executor.go:1928,3134` call-site versions, related test expectations.
  - surfaces: `internal/runtime/executor/claude_executor*.go`, `internal/runtime/executor/helps/claude_device_profile.go`
  - verify: `go test ./internal/runtime/executor/... -run 'Beta|Claude|DeviceProfile' -count=1` green; assert emitted header string equals upstream ordering table for representative bodies (oauth, legacy model, each gated feature).
  - stop condition: upstream gate predicates reference body fields with no local detection helper → halt, list missing predicates.

- phase: `p3-schema-types` — story_id: `story-20260923-p3-schema-types` — status: `checked` — deps: none
  - goal: uppercase `ARRAY`/`OBJECT` types handled (R10).
  - tasks:
    - p3-t1 `internal/util/gemini_schema.go`: `isArrayDeclaredType` (:312) and `isNonObjectDeclaredType` (:297) match type strings case-insensitively (`strings.EqualFold`), incl. `[]any` type arrays.
    - p3-t2 port upstream `2eb8dd11d248` regression test shapes (uppercase OBJECT/ARRAY with nested items) for `CleanJSONSchemaForGemini`, `CleanJSONSchemaForAntigravity`, `CleanJSONSchemaForAntigravityResponse`, `CleanJSONSchemaForAntigravityTool`.
  - surfaces: `internal/util/gemini_schema*.go`
  - verify: `go test ./internal/util/... -count=1` green; new test asserts `"type":"ARRAY"` keeps `items` and gets `items` auto-added when absent.
  - stop condition: EqualFold changes break an existing case-sensitive expectation → halt, identify which path relies on exact case.

- phase: `p4-translator-inputs` — story_id: `story-20260923-p4-translator-inputs` — status: `checked` — deps: none
  - goal: string `input` and video parts translate (R8, R9).
  - tasks:
    - p4-t1 `internal/translator/claude/openai/responses/claude_openai-responses_request.go`: string `input` appended as user text message per `b989e34881c7` (initialize message capacity when input is a string).
    - p4-t2 `internal/translator/openai/openai/responses/openai_openai-responses_request.go`: `input_video`/`video_url` parts → chat-completion video parts preserving URL + processing options per `4b5adbbe9a05`; port `openai_compat_executor_video_test.go` shape.
  - surfaces: `internal/translator/{claude,openai}/openai/responses/`, `internal/runtime/executor/openai_compat_executor*test*`
  - verify: `go test ./internal/translator/... -count=1` green.
  - stop condition: local content-part loop structure can't carry a video part without refactor → halt, map upstream part types first.

### Wave 2 — parallel phases (config/management, file-disjoint from wave 1 and each other)
- phase: `p5-credential-config` — story_id: `story-20260923-p5-credential-config` — status: `checked` — deps: `p0-catalog`
  - goal: `priority` patchable on all credential endpoints; `disable-codex-cloaking` per credential (R5, R6).
  - tasks:
    - p5-t1 `internal/api/handlers/management/config_lists.go`: `priority` field + apply logic on gemini, interactions, openai-compat, vertex, codex patch handlers, mirroring `PatchClaudeKey` (:373-449); port upstream `config_priority_test.go` shape.
    - p5-t2 `internal/config/config_types.go` (or `config.go` wherever `CodexKey` lives): `DisableCodexCloaking *bool` field; watcher `diff/config_diff.go` + `synthesizer/config.go` preserve it; management PATCH accepts it.
    - p5-t3 `internal/runtime/executor/codex_executor.go` + `codex_websockets_*`: `isCodexCloakingDisabled(cfg, auth)` resolution order auth-attribute → credential → global, per `f351924f42cb`; auth attribute key `AttributeCodexDisableCloaking` on `cliproxyauth`.
  - surfaces: `internal/api/handlers/management/config_lists.go`, `internal/config/config_types.go`, `internal/watcher/{diff,synthesizer}/config*.go`, `internal/runtime/executor/codex_*.go`, `sdk/cliproxy/auth/`
  - verify: `go test ./internal/api/handlers/management/... ./internal/config/... ./internal/watcher/... ./internal/runtime/executor/... -count=1` green.
  - stop condition: auth attribute plumbing has no local path from credential config → halt, trace how `Attributes` are populated for codex auth.

- phase: `p6-trusted-proxies` — story_id: `story-20260923-p6-trusted-proxies` — status: `checked` — deps: `p0-catalog`
  - goal: `trusted-proxies` resolves forwarded client IPs (R7).
  - tasks:
    - p6-t1 `internal/config/trusted_proxies.go` (new): IP/CIDR validation + config field `trusted-proxies` on `config.Config`; wire `config_load.go`/`parse.go`; `config.example.yaml` entry.
    - p6-t2 `internal/api/server.go`: `engine.SetTrustedProxies` from config; record `ResolvedClientIP` in request metadata + usage queue details per `a962b77d4383`; port `server_test.go` additions.
  - surfaces: `internal/config/{trusted_proxies.go,config.go,config_load.go,parse.go}`, `internal/api/server*.go`, `config.example.yaml`
  - verify: `go test ./internal/api/... ./internal/config/... -count=1` green; `make build`.
  - stop condition: local server setup lacks a Gin engine hook point → halt, locate where `gin.New`/`gin.Default` is constructed.

### Wave 3 — sequential
- phase: `p7-octal-nul` — story_id: `story-20260923-p7-octal-nul` — status: `checked` — deps: `p5-credential-config`
  - goal: octal NUL `\0` patterns stripped from codex tool schemas (R11).
  - tasks:
    - p7-t1 investigate: locate the local codex tool-schema normalization site (`internal/translator/codex/claude/codex_claude_request.go` `codexSchema*Keywords` path vs executor-side); determine whether upstream's earlier `\p{...}`/`\\u` pattern-strip was ever ported; write findings to `## Progress`.
    - p7-t2 implement `\\0` (and `\p{`/`\\u` if absent) pattern strip at that site, schema-aware per `320100ecf767` (`util.HasUnsupportedUnicodePropertyEscape` or local equivalent); port the upstream octal-NUL test.
  - surfaces: TBD by p7-t1 — `internal/translator/codex/**` or `internal/runtime/executor/codex_*` or `internal/util/responses_tools.go`
  - verify: `go test` on owning package green; request with `^[^\\0]*$` pattern reaches upstream without the pattern.
  - stop condition: no schema-aware normalize site exists at all → halt; escalate whether to build the upstream `stripIncompatiblePatternsFromJSON` wholesale.

### Wave 4 — final gate
- phase: `p8-final-gate` — story_id: `story-20260923-p8-final-gate` — status: `checked` — deps: all
  - goal: checkpoint refreshed, newer delta pinned (R12).
  - tasks:
    - p8-t1 `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi`; if a newer release exists, run `upstream_gap.py`/`upstream_ledger.py` for the new range and pin findings as follow-up rows in `## Progress` — never fold into this plan's scope.
  - surfaces: `docs/upstream/**`, this plan's `## Progress`
  - verify: `docs/upstream/cliproxyapi-checkpoint.json` shows refreshed `generated_at` and checkpoint ≥ `v7.3.15`; `git diff --check` clean.
  - stop condition: none (read-mostly; failure = report fetch error).

## Progress
- 2026-09-23 | phase=p0-catalog wave=W0 task=phase-start task_status=in-progress | run anchor 10:30Z
- 2026-09-23 | phase=p0-catalog wave=W0 task=p0-t1 task_status=DONE | models.json: +claude-opus-5-5 (after fable-5-1), +gpt-6-luna (codex-free first), +gpt-6-sol/gpt-6-luna (after astra in team/plus/pro), +grok-4.7/grok-4.7-build-fast (xai first), -gpt-5.3-codex-spark (plus/pro); upstream field values verbatim | check `python3 json.load` -> valid
- 2026-09-23 | phase=p0-catalog wave=W0 task=p0-t2 task_status=DONE | codex_client_models.json: overlaid upstream v7.3.15 onto local conventions — +gpt-6-sol/gpt-6-luna (mcv 0.155.0, local 5.6-shape + new universal fields), -gpt-5.3-codex-spark, +guardian/supports_reasoning_effort_updates/available_access_programs/supports_experimental_context on shared models, +upgrade migrations (5.5→5.6-sol, 5.6-*→6-sol/luna), +ent26 plans, astra nux+instructions refresh; kept local-only models (5.4, 5.4-mini, 5.3-codex, 5.2) + local priorities + reasoning_summary_format | local fetch tool cmd/fetch_codex_models exists but regenerates upstream-fat shape — file stays hand-maintained
- 2026-09-23 | phase=p0-catalog wave=W0 task=p0-t3 task_status=DONE | models.go: +MarshalCompact (SetEscapeHTML(false), single-line), +nullCodexClientRequiredOptions (null not delete), +codexClientFallbackInstructions/useCompactCodexClientInstructions; call sites server.go:1057 + openai_handlers.go:63 emit c.Data compact bytes; tests ported (assertCodexNullableFieldCleared, single-line, <1MiB) | check `go test ./internal/registry/... ./internal/client/codex/models/... -count=1` -> ok; web_search_capability_test spark rows repointed to gpt-6-sol/luna
- 2026-09-23 | wave=W0 summary | p0-catalog DONE 3/3 tasks — catalogs match upstream v7.3.15 within local conventions; gate APPROVED (Validation entry)

- 2026-09-23 | phase=p1-registry-count wave=W1 task=phase-start task_status=in-progress | run anchor ~10:55Z
- 2026-09-23 | phase=p1-registry-count wave=W1 task=p1-t1 task_status=DONE | model_registry.go: quotaAndOtherSuspended correction at buildAvailableModelsLocked (:1328) + GetAvailableModelsByProvider (:1470) + GetModelCount (:1545 suspended clients skip quota-active) per ed70aeaa1627
- 2026-09-23 | phase=p1-registry-count wave=W1 task=p1-t2 task_status=DONE | 3 upstream tests ported; conductor.go cooldownAuthRestorable relaxed to upstream authCooldownStateRecord semantics (auth-level record saved alongside model records — was local divergence dropping credential_quota restore); cooldown_persistence_test expectations updated to auth+model record shape; memoryCooldownStateStore reused (Postgres-backed prod store)
- 2026-09-23 | phase=p3-schema-types wave=W1 task=phase-start task_status=in-progress
- 2026-09-23 | phase=p3-schema-types wave=W1 task=p3-t1 task_status=DONE | gemini_schema.go isNonObjectDeclaredType + isArrayDeclaredType -> strings.EqualFold per 2eb8dd11d248; upstream sanitizeArrayItems site has no local counterpart (local never strips items)
- 2026-09-23 | phase=p3-schema-types wave=W1 task=p3-t2 task_status=DONE | TestCleanJSONSchema_UppercaseTypeRepair: ARRAY missing-items repair + items preservation + OBJECT bare-prop folding across all 4 local cleaners
- 2026-09-23 | phase=p4-translator-inputs wave=W1 task=phase-start task_status=in-progress
- 2026-09-23 | phase=p4-translator-inputs wave=W1 task=p4-t1 task_status=DONE | claude_openai-responses_request.go: string input -> appendParts("user", textPart) per b989e34881c7; test adapted to local instructions-as-user-message shape (not upstream system array)
- 2026-09-23 | phase=p4-translator-inputs wave=W1 task=p4-t2 task_status=DONE | openai_openai-responses_request.go: input_video/video_url -> video_url chat parts preserving url/processing per 4b5adbbe9a05; input_image detail preservation via normalizeChatImageDetail added (upstream current semantics); video translator test + executor e2e test ported
- 2026-09-23 | phase=p2-claude-betas wave=W1 task=phase-start task_status=in-progress
- 2026-09-23 | phase=p2-claude-betas wave=W1 task=p2-t1 task_status=DONE | claude_executor.go: 9 new beta constants (per-turn-control, timing, mid-conv-tool-changes, inline-tools, clear-at, dangerous-tool-use, thinking-binding, thinking-resumption, prompt-caching-evict) + claudeManagedBetaSet + isManagedClaudeBeta + wire order per bd584a752329/779bf317e030 incl. legacy else-branch
- 2026-09-23 | phase=p2-claude-betas wave=W1 task=p2-t2 task_status=DONE | helpers claudeCanonicalModel/claudeModelHasPerTurnEffort/claudeModelHasPerTurnTiming/claudeIncludePerTurn{Control,Timing,InlineTools,MidConvClearAt} ported; caller-beta gating upgraded to managed-set semantics (unmanaged caller betas now forwarded on Anthropic — upstream #5738 semantics, was stale drop-all)
- 2026-09-23 | phase=p2-claude-betas wave=W1 task=p2-t3 task_status=DONE | baseline 2.1.258->2.1.280: device profile UA + DefaultClaudeVersion fallback + checkSystemInstructionsWithSigningMode call sites + comments; config.example.yaml N/A locally (Postgres-authoritative config, no example file)
- 2026-09-23 | wave=W1 summary | p1-p4 DONE — all verify commands green; gate APPROVED (Validation entry)

- 2026-09-23 | phase=p5-credential-config wave=W2 task=phase-start task_status=in-progress | run anchor ~13:00Z
- 2026-09-23 | phase=p5-credential-config wave=W2 task=p5-t1 task_status=DONE | priority patch added to PatchGeminiKey/PatchOpenAICompat/PatchVertexCompatKey/PatchCodexKey (claude + meta pre-existing); no local XAI/interactions credential handlers — upstream-only types, skipped; config_priority_test.go matrix ported over 6 local families via recordingConfigStore
- 2026-09-23 | phase=p5-credential-config wave=W2 task=p5-t2 task_status=DONE | CodexKey.DisableCodexCloaking *bool + PATCH tri-state (applyDisableCodexCloakingPatch, null=inherit) + watcher diff line + synthesizer attr; attr key = literal "disable_codex_cloaking" via new const cliproxyauth.AttributeCodexDisableCloaking (fork uses literal keys, no AttributeConfigIndex-style consts elsewhere); config.example.yaml N/A locally
- 2026-09-23 | phase=p5-credential-config wave=W2 task=p5-t3 task_status=DONE | resolveCodexConfig refactored -> resolveCodexKeyConfig(cfg, auth); isCodexCloakingDisabled(ctx, cfg, auth) precedence: auth attr -> credential entry -> global runtime-control ctx flag (local global = runtimecontrol.Cloaking.DisableCodex via middleware ctx, NOT upstream's static cfg.Codex.*); wired into applyCodexHeaders + applyCodexWebsocketHeaders
- 2026-09-23 | phase=p5-credential-config wave=W2 note | upstream hard-overrides User-Agent/Originator after custom headers — NOT ported: local layered precedence (existing > config-defaults > client > identity) is deliberate (8 pre-existing tests encode it); adapted tests assert observable local semantics instead
- 2026-09-23 | phase=p6-trusted-proxies wave=W2 task=p6-t1 task_status=DONE | config.TrustedProxies []string + validateTrustedProxies (IP/CIDR, whitespace reject) wired in LoadConfigOptional + ParseConfigBytes; trusted_proxies_test.go verbatim
- 2026-09-23 | phase=p6-trusted-proxies wave=W2 task=p6-t2 task_status=DONE | server.go SetTrustedProxies after gin.New with fallback-disable on error; ClientRequestMetadata.ResolvedClientIP populated in GetContextWithCancel (first client-metadata write site — local ClientIP/XFF/UA fields were dead); redisqueue requestDetail + resolved_client_ip JSON; config.example.yaml N/A; tests: server trusted/untrusted peer matrix, GetContextWithCancel capture, plugin payload field
- 2026-09-23 | phase=p6-trusted-proxies wave=W2 fix | server_test.go stale asserts from W0 catalog change: custom model base_instructions now compact literal (not template clone), apply_patch_tool_type/upgrade/availability_nux now present-and-null (not omitted) — updated to upstream 673131f57484 semantics
- 2026-09-23 | wave=W2 summary | p5+p6 DONE — all verify commands green incl. `go test ./internal/api/... ./internal/config/... ./internal/runtime/executor/... ./internal/watcher/...` + `make build`; gate APPROVED (Validation entry)

- 2026-09-23 | phase=p7-octal-nul wave=W3 task=phase-start task_status=in-progress | run anchor ~14:00Z
- 2026-09-23 | phase=p7-octal-nul wave=W3 task=p7-t1 task_status=DONE | investigation: upstream `helps/codex_tool_schema.go` never ported — earlier `\p{...}` strip (e56abd56/37ce368c/bf20b999) absent from ledger, whole pipeline missing locally; local codex schema keywords in `internal/translator/codex/claude/codex_claude_request.go` only serve the claude→codex translate path, not all codex upstream submissions → executor-side insertion chosen
- 2026-09-23 | phase=p7-octal-nul wave=W3 task=p7-t2 task_status=DONE | NEW `internal/runtime/executor/helps/codex_tool_schema.go`: NormalizeCodexToolSchemas + normalizeCodexToolList/normalizeCodexTool (batch-copy + namespace recursion) + stripIncompatiblePatternsFromJSON/stripIncompatiblePatterns (schema-aware walk, patternProperties key handling, fast-path, no-HTML-escape re-encode) — strip-only port; union-simplification machinery from bf20b999 stays deferred with perf batch. `internal/util/claude_schema.go`: +HasUnsupportedUnicodePropertyEscape (\p{...}/\P{...}/\0, \\0 escaped-backslash safe), +SchemaMapKeywords/+SchemaValueKeywords (upstream verbatim). Wired at all 5 request-sending sites: codex_executor.go after normalizeCodexParallelToolCalls (:313,:557,:679) + codex_websockets_executor.go after normalizeCodexWebsocketParallelToolCalls (:407,:1248); count-tokens site skipped (upstream doesn't normalize count path)
- 2026-09-23 | phase=p7-octal-nul wave=W3 task=p7-t3 task_status=DONE | tests: helps/codex_tool_schema_test.go (octal NUL strip + field preservation + hex-NUL/valid-pattern survival + idempotence, unicode-property strip, patternProperties key strip, non-schema user-data pattern preservation, nested $defs/namespace recursion, passthrough fast-path) + util predicate table test incl. \\0-literal and trailing-backslash edge cases
- 2026-09-23 | wave=W3 summary | p7 DONE — `go test ./internal/util/ ./internal/runtime/executor/helps/ ./internal/runtime/executor/` ok; `go build ./...` clean; `git diff --check` clean; gofmt on touched files clean; gate APPROVED (Validation entry)

- 2026-09-23 | phase=p8-final-gate wave=W4 task=phase-start task_status=in-progress | run anchor ~14:30Z
- 2026-09-23 | phase=p8-final-gate wave=W4 task=p8-t1 task_status=DONE | upstream_sync: "checkpoint already at v7.3.15 — no upstream delta"; `git ls-remote --tags` confirms v7.3.15 is still the latest release (no v7.3.16+/v7.4.x) → no follow-up rows needed; checkpoint `generated_at` refreshed
- 2026-09-23 | wave=W4 summary | p8 DONE — plan scope fully executed; R1–R11 ported/adapted, R12 gate clean. Deferred per scope_policy: request-proxy-overrides (pluginhost-dependent), perf-batch 56518489 et al., upstream-docs-gofmt. Residual divergence intentionally kept: codex header layered precedence (no post-attrs identity hard-override), count-tokens path not schema-normalized (matches upstream)

## Decisions
none

## Validation
- `2026-09-23T10:50Z` — phase: `p0-catalog` — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: codex_client_models.json overlay choices (local key-shape for gpt-6-sol/luna, retained local priorities and local-only models) are a judgment call reviewed against upstream end-state, not independently re-derived; catalog served to a live Codex client not exercised (unit-tested only)
  - commands:
    - `python3 -c "import json; json.load(open('internal/registry/models/models.json'))"` → valid
    - `/usr/local/go/bin/go test ./internal/registry/... ./internal/client/codex/models/... -count=1` → ok (registry 0.063s, models 0.094s)
    - `/usr/local/go/bin/go build ./...` → clean
    - `git diff --check` → clean; `/usr/local/go/bin/gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.15-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.15, docs/upstream/cliproxyapi-ledger-v7.3.11..v7.3.15.md
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 1 (assertCodexNullableFieldCleared + spark test repoint after first test run)
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: upstream field-level parity of new catalog entries on a live Codex 0.155+ client

- `2026-09-23T11:15Z` — phase: `p1-registry-count,p2-claude-betas,p3-schema-types,p4-translator-inputs` (wave 1) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: 2.1.280 beta wire order verified against upstream's published order table and test expectations, not against a live 2.1.280 binary capture; mythos-5-1 per-turn-timing capability list follows upstream's helper verbatim (no local model catalog cross-check needed since gate is prefix-based); cooldownAuthRestorable semantic change assumes upstream snapshot shape is desirable for local Postgres persistence (same record format, different backend)
  - commands:
    - `/usr/local/go/bin/go test ./internal/registry/ ./sdk/cliproxy/auth/ -count=1` → ok (registry 0.165s, auth 30.7s; 3 ported quota regression tests pass; cooldown persistence updated)
    - `/usr/local/go/bin/go test ./internal/util/ -count=1` → ok 1.195s (UppercaseTypeRepair 3/3)
    - `/usr/local/go/bin/go test ./internal/translator/openai/openai/responses/ ./internal/translator/claude/openai/responses/ -count=1` → ok (VideoInput 16/16 subtests, MixedVideoInputOrder, StringInput)
    - `/usr/local/go/bin/go test ./internal/runtime/executor/ -count=1` → ok 2.409s (21280GatedBetas 6/6, ForwardsUnmanagedCallerBetas, BetaAssemblyPerRequest updated; full suite green incl. prior fingerprint tests rebased to 2.1.280)
    - `/usr/local/go/bin/go test ./internal/runtime/executor/helps/ -count=1` → ok 5.856s
    - `/usr/local/go/bin/go build ./...` → BUILD_RC=0
    - `git diff --check` → clean; `/usr/local/go/bin/gofmt -l` on touched files → clean (4 unrelated pre-existing unformatted files from excluded upstream-docs-gofmt scope)
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.15-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.15 (commits ed70aeaa1627, 2eb8dd11d248, b989e34881c7, 4b5adbbe9a05, bd584a752329, 779bf317e030)
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 3 (GetModelCount third-site fix, memoryCooldownStateStore name collision, cooldownAuthRestorable divergence + persistence-test expectations)
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: live 2.1.280 client capture; Postgres cooldown store end-to-end restart (in-memory store used in tests)

- `2026-09-23T13:10Z` — phase: `p5-credential-config,p6-trusted-proxies` (wave 2) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: cloaking-disable precedence verified via unit tests on header fns, not a live Codex upstream; trusted-proxy resolution verified via gin test-engine, not a deployed reverse proxy; local requestDetail lacks upstream's client_ip/xff/ua fields (pre-existing payload minimization) so only resolved_client_ip was added
  - commands:
    - `/usr/local/go/bin/go test ./internal/api/handlers/management/ ./internal/config/ ./internal/watcher/... -count=1` → ok (priority matrix 6/6, cloaking PATCH tri-state, diff + synthesizer attr tests)
    - `/usr/local/go/bin/go test ./internal/runtime/executor/... -count=1` → ok (7 cloaking precedence tests; all pre-existing header contract tests pass after adaptation)
    - `/usr/local/go/bin/go test ./internal/api/... ./internal/config/... -count=1` → ok (trusted-proxy peer matrix, ResolvedClientIP capture, plugin payload)
    - `make build` → ok (web embed + go build)
    - `git diff --check` → clean; `gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.15-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.15 (commits cc77410866c2, f351924f42cb, a962b77d4383)
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 2 (upstream post-attrs identity hard-override conflicted with deliberate local header precedence — adapted; stale codex-catalog asserts in server_test.go updated to compact-fallback semantics)
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: live Codex upstream cloaking behavior; deployed reverse-proxy end-to-end

- `2026-09-23T14:20Z` — phase: `p7-octal-nul` (wave 3) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: strip verified on synthetic payloads mirroring upstream tests, not against a live Codex upstream rejection/acceptance cycle; union-simplification machinery (bf20b999decb) intentionally not ported — only the pattern strip
  - commands:
    - `/usr/local/go/bin/go test ./internal/util/ ./internal/runtime/executor/helps/ -count=1` → ok (util 0.981s, helps 8.195s — predicate table, octal-NUL strip, patternProperties, non-schema preservation, namespace recursion, idempotence)
    - `/usr/local/go/bin/go test ./internal/runtime/executor/ -count=1` → ok 2.351s (all pre-existing executor tests green with new normalize call sites wired)
    - `/usr/local/go/bin/go build ./...` → clean
    - `git diff --check` → clean; `/usr/local/go/bin/gofmt -l` on touched files → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.15-parity.md, refs/upstream-checkpoints/cliproxyapi/v7.3.15 (commit 320100ecf767, related e56abd56f142/37ce368c5002)
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: live Codex upstream submission with an offending \0 pattern

- `2026-09-23T14:40Z` — phase: `p8-final-gate` (wave 4) — verdict: `APPROVED`
  - mode: `gate`
  - judge: `same-session`
  - judge_model: `SWE-2 Max`
  - proof_gaps: none additional — upstream tag listing is the authority for "no newer delta"; cumulative plan-level proof gaps are recorded on their per-wave entries
  - commands:
    - `python3 .claude/skills/upstream/scripts/upstream_sync.py sync --slug cliproxyapi` → "checkpoint already at v7.3.15 — no upstream delta"
    - `git ls-remote --tags https://github.com/router-for-me/CLIProxyAPI` → v7.3.15 remains latest (no v7.3.16+, no v7.4.x)
    - `python3 -c "import json; json.load(...)"` on checkpoint → valid; `git diff --check` → clean
  - receipt:
    context_sources: docs/plans/active/cliproxyapi-v7.3.15-parity.md, docs/upstream/cliproxyapi-checkpoint.json
    policy: targeted-semantic-ports
    judge: same-session
    judge_model: SWE-2 Max
    retries: 0
    rollback_point: none
    failure_ledger: absent
    enforcement: local-only
    not_independently_verified: n/a

## Current State and Next Action
- active_phase: none — all 9 phases checked; plan scope complete
- lifecycle_status: complete
- latest anchors: wave 4 Progress lines (2026-09-23), Validation `2026-09-23T14:40Z` APPROVED
- blockers: none
- open items: none
- exact_next_action: none — move plan to `docs/plans/completed/` on next housekeeping pass; deferred upstream items (request-proxy-overrides, perf-batch, union-simplification) tracked in scope_policy + W4 summary
