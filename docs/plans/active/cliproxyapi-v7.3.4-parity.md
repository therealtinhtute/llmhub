---
id: plan-20260916-v734
type: plan
intake_id: intake-20260916-v734
lane: high-risk
status: active
created: 2026-09-16
updated: 2026-09-16
---

# Plan: CLIProxyAPI v7.3.4 targeted parity (Meta / Muse Code provider)

## Outcome
- result: llmhub ports the owner-approved v7.3.3..v7.3.4 capability slices as independently verifiable semantic ports behind existing translator, executor, auth, registry, and SDK interfaces — centered on the new Meta (Muse Code) provider with full device-flow UX in TUI and WebUI — without wholesale merge, pluginhost, or branding churn.
- success_signals:
  - Each accepted include slice lands with focused tests citing its upstream commit(s) and local symbol.
  - Meta authenticates via RFC 8628 device flow, mints LLM API keys via `/muse-code/key`, executes through the local `Executor` interface (codex/OpenAI-Responses protocol), and resolves `muse-spark-*` models from the shared catalog.
  - TUI shows the device `user_code` and skips callback input; Web panel exposes meta OAuth with device-code display (`meta` in WEBUI_SUPPORTED).
  - Postgres remains the authoritative runtime store; no new file/YAML source of truth.
  - `go test` on touched packages, `make build`, and `git diff --check` pass at gates.
  - Newer-than-v7.3.4 upstream delta is pinned as follow-up, never silent scope growth.

## Authority and Requirements
- authority:
  - `docs/upstream/cliproxyapi-checkpoint.json` — checkpoint `v7.3.4` at `8335eac73194` / `refs/upstream-checkpoints/cliproxyapi/v7.3.4`; `scope_policy` strategy `targeted-semantic-ports` (9 include, 6 exclude, 6 defer) is the scope authority.
  - `docs/upstream/cliproxyapi-gap-v7.3.3..v7.3.4.json` — 80 paths (upstream-add-absent 13, diverged-absent 25, semantic-review 41, baseline 1). [gitignored artifact — regenerate via `upstream_gap.py`]
  - `git log --no-merges v7.3.3..v7.3.4` — 26 non-merge commits, one release.
  - `docs/plans/completed/cliproxyapi-v7.3.3-parity.md` — prior cycle; Decisions hold resolved remainders (oauth-session cancel NOW PORTED — use it), do not regress R1–R15 slices.
  - `CLAUDE.md` / `docs/PROJECT.md` — Postgres-authoritative store, llmhub management UI, SDK compatibility, pluginhost non-goal.
  - Handoff analysis `7d91f4db` (2026-09-16) — per-slice feasibility verified against local code.
- rejected_alternatives:
  - Wholesale merge of 80 paths — diverged-absent/semantic-review classes make it unsafe.
  - Copying upstream's split conductor files — local `conductor.go` is monolithic; port semantics into local structure.
  - Excluding TUI/WebUI device UX — owner explicitly approved full device-flow UX on both surfaces.
- requirements:
  - R1 [accepted]: Misc upstream fixes — tool-call argument validation for finish-reason logic in `internal/translator/openai/claude/openai_claude_response.go` (emit `length` instead of `tool_calls` on empty/invalid accumulated args; `length`/`content_filter` passthrough; uses `util.FixJSON`); management.html no-cache headers in `serveManagementControlPanel`. | source: `772c63c8` `78c14b80`
  - R2 [accepted]: Meta model definitions — `muse-spark-1.3`, `muse-spark-1.3-contributor`, `muse-spark-1.2`, `muse-spark-1.2-contributor`, `muse-spark-1.1` (1M context, thinking levels incl. xhigh/max) in `internal/registry/models/models.json` `"meta"` section; `staticModelsJSON.Meta`/`GetMetaModels`/`GetStaticModelDefinitionsByChannel` meta+muse channel; `model_updater.go` requiredSections/detectChangedProviders/Meta-backfill. | source: `54d4f4c0` `65348b95` `cee799f6` (models hunks)
  - R3 [accepted]: Meta auth + OAuth device flow + key mint — `internal/auth/meta/**`, `sdk/auth/meta.go` authenticator, `internal/cmd/meta_login.go`, RFC 8628 device flow (`auth.meta.com/oidc/device/*`, client `1031625952748946`), `POST api.meta.ai/muse-code/key` minting with `Bearer dca:`, TokenStorage integration, `RefreshLead()=nil` (on-demand recovery via `dca_token`), DCA rejection in `meta-api-key` config, minted-key persistence under epoch/lock, cleared-`expired` persist exclusion, `NormalizeOAuthProvider` meta/muse. | source: `54d4f4c0` `30191c3a` `cee799f6` `47cc31ae` `be7323f3` `1144ae70` `4a0131c0` `d09042a5` `21aa46b6` (auth halves)
  - R4 [accepted]: Meta executor + conductor credential lifecycle — `meta_executor{,_execute,_stream}.go` behind local `Executor`/`ProviderExecutor` interfaces (upstream API is OpenAI-Responses-compatible: `POST {base}/responses`, `to=FromString("codex")`), request preparer, unconditional-refresh removal, 429 quota scoping, credential minting/persist inside `updateInternal` via `refreshLocks`/`authRefreshLock`, `authHasRefreshCredential` +`dca_token`, `MergeExistingAuthMetadata` meta skip, `indexSeed` +`meta-api-key`, `Manager.PrepareRequestAuth` export. Local shims expected: `IsConfigAPIKeyAuth`→`AccountInfo()`/`auth_kind`, translate fallback to `helps.TranslateRequestWithCodexMultiAgentV2`, `sanitizeOpenAIResponsesReasoningEncryptedContent` (port or reuse codex bootstrap handling), `ApplyRequestThinking`→`thinking.ApplyThinking` pattern; `reporter.SetTranslatedReasoningEffort` remains absent (recorded). | source: `65348b95` `54d4f4c0`(exec) `d09042a5` `be7323f3` `06660dd6` `4a0131c0` `47cc31ae` `cee799f6`(exec)
  - R5 [accepted]: `meta-api-key` config family + alias/error-rules channel — `MetaKey=CodexKey`-shaped list, `SanitizeMetaKeys` (drop empty/`dca:` keys, default base `https://api.meta.ai/v1`), synthesizer via `synthesizeCodexStyleKeys`, config-diff block, `BuildAPIKeyClients` 6th return, `resolveConfigMetaKey`/`buildMetaConfigModels`/`case "meta"` in service registration, mgmt CRUD `Get/Put/Patch/DeleteMetaKeys` + `metaKeyWithAuthIndex` + routes, `OAuthModelAliasChannel`+`extractRequestScopedErrorRules` meta cases. | source: `54d4f4c0` `cee799f6` `1144ae70` `e475807a` `8335eac7` `47cc31ae`(alias hunks)
  - R6 [accepted]: Meta management OAuth + FULL device-flow UX — `RequestMetaToken` (~140 LOC, new `auth_files_meta_oauth.go` mirroring `auth_files_devin_oauth.go`) using the NOW-PORTED cancel machinery (`watchOAuthSessionCancel`/`guardOAuthSessionPendingForSave`/`CancelOAuthSession` exist locally as of `dc3aa689`), routes in `server.go` mgmt block; TUI full device UX (`userCode`/`expiresIn`/`deviceFlow` fields, `deviceOAuthPollTimeout`, `shouldFailOAuthStatusPoll`/`maxOAuthStatusPollErrors`, `renderDeviceMode`, device i18n keys, `Meta` provider entry); WebUI `'meta'` in `OAuthProvider` union + `WEBUI_SUPPORTED` with device-code display from start response (owner-approved net-new UI work). | source: `23c16e29` `e475807a` `18385de0` + TUI deviceFlow share of `6e819ab62257`
  - R7 [accepted]: Meta `/api-call` token resolution — `resolveMetaToken` + `metaManagementPreparer` + `metaTokenFromAuth` + `case "meta"` in `proxyURLFromAPIKeyConfig`, beside existing gemini-cli/antigravity branches in `resolveTokenForAuth`. | source: `4a0131c0` `06660dd6` `be7323f3` `cee799f6` (api_tools hunks)
  - R8 [accepted]: Invariants — Postgres authoritative runtime store; public SDK additive-only; no new `web/**/*_test.go` (verify frontend via typecheck/lint/build); oauth-session cancel machinery (ported in `dc3aa689`) is the local convention for meta.
  - R9 [accepted]: Final gate — re-resolve latest stable upstream release, refresh checkpoint + ledger; any newer release pinned as follow-up.

## Non-goals
- NG1: plugin-quota hardening (`acb0eae2` + 7 validation commits) — depends on `sdk/pluginapi`/`h.pluginHost`, inside excluded `pluginhost-platform` scope.
- NG2: `config-optional commandMode` (`fbf74645`) — REJECTED: llmhub loads runtime config from Postgres unconditionally; no config.yaml path exists to make optional (Postgres-authoritative invariant).
- NG3: `assets/logo/meta.svg` (`e475807a` hunk) — no local `assets/` dir; web panel owns UI assets.
- NG4: `config.example.yaml` hunks — file deleted locally (383222ee); `x/text` promotion moot (only needed by excluded plugin-quota).
- NG5: wholesale upstream merge; non-additive SDK breakage; branding churn.
- NG6: xAI device-flow refactor (`6e819ab6` non-cancel share) — recorded follow-up, not this initiative's scope.
- NG7: antigravity web-search probe stack (`48dcadd9`/`e30de3d5` + `60e5b8bd` `antigravity_models.go`) — separate feature; `ApplyClientModelCapabilities` machinery already in place for when it lands.

## Approach and Risks
- Semantic ports behind local interfaces, never text-applied — local conductor is monolithic, management files are differently split, devin OAuth pattern is the local template.
- Dependency order matters: models before executor registration; auth before executor lifecycle; executor before config-apikey synthesis; all before mgmt/api-call surfaces.
- oauth-session cancel machinery is already local (`dc3aa689`) — meta mgmt handler uses it directly, no second port.
- Risks: conductor lifecycle surgery in monolithic file (refreshLocks/persist paths); `compileAPIKeyModelCapabilitiesForAuth` meta case folds into the landed capability machinery; Meta endpoints untestable live (httptest only, same as devin).

## Phases and Verification
Phase gate (each phase): `go test -count=1` on touched packages + `make build` + `git diff --check master..HEAD` + `gofmt -l` on changed files.

### Phase `misc-fixes` (story-20260916-misc-fixes) — status: in-progress
- goal: R1 — translator finish-reason validation + mgmt no-cache.
- dependencies: none.
- allowed surfaces: `internal/translator/openai/claude/**`, `internal/api/server.go` (serveManagementControlPanel only).
- avoided surfaces: everything else.
- waves:
  - W1 (single agent, both are tiny):
    - T1 `772c63c8` — `hasValidToolCallArguments` + `effectiveOpenAIFinishReason` rework (~50 LOC impl + ~97 LOC test). check: `go test ./internal/translator/openai/...`
    - T2 `78c14b80` — 3 no-cache headers before `c.Data` in `serveManagementControlPanel` (~`internal/api/server.go:900`). check: `go test ./internal/api/...`

### Phase `meta-models` (story-20260916-meta-models) — status: in-progress
- goal: R2 — muse-spark catalog + registry/updater plumbing.
- dependencies: none.
- allowed surfaces: `internal/registry/models/models.json`, `internal/registry/model_definitions.go`, `internal/registry/model_updater.go`, related tests.
- waves:
  - W1:
    - T1 models.json `"meta"` section + `staticModelsJSON.Meta`/`GetMetaModels` + channel wiring + updater requiredSections/detectChangedProviders/backfill + tests. check: `go test ./internal/registry/...`

### Phase `meta-auth` (story-20260916-meta-auth) — status: in-progress
- goal: R3 — device OAuth + key mint + auth record.
- dependencies: none (cancel machinery already landed).
- allowed surfaces: new `internal/auth/meta/**`, `sdk/auth/meta.go`, `internal/cmd/meta_login.go`, `sdk/auth/refresh_registry.go`, `internal/cmd/auth_manager.go`, `internal/api/handlers/management/oauth_sessions.go` (NormalizeOAuthProvider case only), `internal/util` (shared helpers if needed).
- waves:
  - W1:
    - T1 `internal/auth/meta/**` + `sdk/auth/meta.go` + refresh-registry + auth-manager + `-meta-login` cmd. check: `go test ./internal/auth/meta/... ./sdk/auth/... ./internal/cmd/...`

### Phase `meta-executor` (story-20260916-meta-executor) — status: planned
- goal: R4 — executor trio + conductor lifecycle.
- dependencies: `meta-models` (model resolution), `meta-auth` (auth records).
- allowed surfaces: new `internal/runtime/executor/meta_executor{,_execute,_stream}.go` + tests, `sdk/cliproxy/auth/conductor.go`, `sdk/cliproxy/auth/types.go`, `sdk/cliproxy/auth/metadata_merge.go`, `sdk/cliproxy/service.go` (executor registration + PrepareRequestAuth export).
- waves:
  - W1:
    - T1 conductor lifecycle deltas (mint serialization, persistMetaMint, dca_token credential detection, indexSeed, metadata-merge skip). check: `go test ./sdk/cliproxy/...`
    - T2 executor trio + service registration + shims. check: `go test ./internal/runtime/executor/... ./sdk/cliproxy/...`

### Phase `meta-config-apikey` (story-20260916-meta-config-apikey) — status: planned
- goal: R5 — MetaKey config family + alias/error-rules cases.
- dependencies: `meta-executor` (synthesized auths need registered executor).
- allowed surfaces: `internal/config/**`, `internal/watcher/synthesizer/config.go`, `internal/watcher/clients.go`, `internal/watcher/diff/config_diff.go`, `sdk/cliproxy/service.go` (resolve/build/case hunks), `sdk/cliproxy/auth/oauth_model_alias.go`, `sdk/cliproxy/auth/conductor_request_scoped_errors.go`, `sdk/cliproxy/auth/api_key_model_capabilities.go` (meta case folds into landed machinery), `internal/api/handlers/management/config_lists.go` + `config_auth_index.go` + `config_meta_keys*.go` (new), `internal/api/server.go` (routes).
- waves:
  - W1:
    - T1 config family: MetaKey type/sanitize/parse + synthesizer + diff + BuildAPIKeyClients + service resolve/build/case. check: `go test ./internal/config/... ./internal/watcher/... ./sdk/cliproxy/...`
    - T2 mgmt CRUD + auth-index + routes + alias/error-rules meta cases. check: `go test ./internal/api/... ./sdk/cliproxy/...`

### Phase `meta-mgmt-oauth` (story-20260916-meta-mgmt-oauth) — status: planned
- goal: R6 — management OAuth endpoint + FULL TUI device UX + WebUI support.
- dependencies: `meta-auth` (auth service), `meta-executor` (display only — soft).
- allowed surfaces: new `internal/api/handlers/management/auth_files_meta_oauth.go`, `internal/api/server.go`, `internal/tui/**`, `web/src/services/api/oauth.ts` + device-code display components (owner-approved UI work), `internal/api/handlers/management/auth_files.go` (wiring only).
- waves:
  - W1 (parallel-safe pair — disjoint files):
    - T5a `RequestMetaToken` + routes + TUI full deviceFlow UX. check: `go test ./internal/api/... ./internal/tui/...`
    - T5b web panel: `'meta'` union + `WEBUI_SUPPORTED` + device-code display. check: `cd web && bun run build` (or repo lint/typecheck script — no new test files per invariant)

### Phase `meta-api-call` (story-20260916-meta-api-call) — status: planned
- goal: R7 — `/api-call` meta token resolution.
- dependencies: `meta-executor` (`Manager.PrepareRequestAuth` export).
- allowed surfaces: `internal/api/handlers/management/api_tools.go` + tests.
- waves:
  - W1:
    - T6 `resolveMetaToken`/`metaManagementPreparer`/`metaTokenFromAuth`/`proxyURLFromAPIKeyConfig` meta case. check: `go test ./internal/api/...`

### Phase `final-gate` (story-20260916-final-gate) — status: planned
- goal: R9 — checkpoint refresh and delta pinning.
- dependencies: all phases above.
- allowed surfaces: `docs/upstream/**`, checkpoint refs.
- waves:
  - W1:
    - T1 `upstream_sync.py sync --slug cliproxyapi` → newer tag or confirms v7.3.4; newer delta → `follow-up:` Decisions, never worked.

## Decisions
- 2026-09-16 | phase=init task=triage | decision: approved FULL TUI device-flow UX (user_code display, skip callback input, deviceOAuthPollTimeout, device i18n) AND `meta` in `WEBUI_SUPPORTED` with device-code display — owner chose the complete option on both triage questions | rationale: meta's device flow is unusable without visible user_code; devin/Kimi patterns make the work bounded.
- 2026-09-16 | phase=init task=triage | decision: meta mgmt OAuth uses the oauth-cancel machinery ported in `dc3aa689` (watchOAuthSessionCancel/guardOAuthSessionPendingForSave/CancelOAuthSession) | rationale: resolves the "port cancel now or stay devin-consistent" triage question — it landed as debt cleanup first.
- 2026-09-16 | phase=init task=triage | decision: `config-optional commandMode` (`fbf74645`) rejected — llmhub is Postgres-authoritative with no config.yaml path; `plugin-quota-hardening` excluded (pluginhost scope); `meta.svg` excluded (no local assets dir) | rationale: recorded in checkpoint scope_policy.

## Progress
- 2026-09-16 | phase=init task=plan-seed task_status=DONE | handoff analysis `7d91f4db` scoped the full delta (26 commits / 80 paths / +5275−85 → 9 include slices, 3 excluded); checkpoint scope_policy recorded (9 include / 6 exclude / 6 defer); owner triage answers received (full TUI UX + WEBUI_SUPPORTED); plan written from `docs/plans/completed/cliproxyapi-v7.3.3-parity.md` conventions

- 2026-09-16 | phase=misc-fixes+meta-models+meta-auth task=phase-start | wave-1 parallel fanout — zero-dep trio; disjoint surfaces (translator+server.go vs registry vs auth/cmd); per plan wave-1 ordering

## Validation
- (populated at each phase gate — see work-full.md step 11 / check-validation.md format)

## Current State and Next Action
- active_phase: misc-fixes + meta-models + meta-auth — wave-1 in flight
- lifecycle_status: in-progress
- latest_anchors: prior initiative `docs/plans/completed/cliproxyapi-v7.3.3-parity.md` (v0.0.39 released); debt-cleanup commits on master `dc3aa689`/`c5a96047`/`4729ed69`/`73a2c234`
- blockers: none — all deps satisfied (cancel machinery landed, capability machinery landed, devin pattern established)
- open_items:
  - remainders folded here: `compileAPIKeyModelCapabilitiesForAuth` meta case (phase meta-config-apikey), `RequestToFormat`/`SetTranslatedReasoningEffort` absences (recorded, meta tolerates them like codex did)
  - environmental: `TestUpdateCommandRollback` + `internal/updater` `TestRollbackFailure` fail under uid=0 (pre-existing, unrelated)
- exact_next_action: kick off `misc-fixes` + `meta-models` + `meta-auth` — three parallel-safe phases with no dependencies
