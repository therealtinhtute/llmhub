---
id: 013F0JDYKX1E8Q0ADQZC61N4TF
type: plan
intake_id: 011873R0W1JDJB5MXV74BGH5AB
lane: high-risk
status: active
created: 2026-09-25
updated: 2026-09-25
---

# Plan: Quota alert accuracy, parity, and notification unblocking

## Outcome
- result: Quota monitoring produces the same per-auth, per-window remaining values the quota management screen renders for all supported providers; warning/exhausted/recovery transitions — including for Claude and Codex — produce deduplicated provider-grouped Telegram notifications; qualified request-path quota evidence accelerates detection without waiting for the poll timer; collector failures are visibly diagnosable instead of silent `unknown` rows.
- success_signals:
  - Claude and Codex auths whose OAuth tokens live only in `Metadata` produce `reliable` observations; collector tests use production-shaped snapshots so the divergence cannot regress.
  - For every supported provider, collector fixture outputs map to the same window identities and remaining percentages the quota cards render; Codex monthly/team windows are not labeled `weekly` and Claude `limits[]`-scoped windows (e.g., fable) are represented.
  - Runtime-only/virtual auths (e.g., `runtime_only` Gemini virtual credentials) never create quota-alert states.
  - A request-path quota-exhaustion signal triggers an accelerated collection for the affected auth while bare HTTP 429 still cannot classify exhaustion.
  - One provider-grouped Telegram batch per logical transition is delivered; delivery failures expose a sanitized `failure_code` and log line distinguishing destination misconfiguration from upstream send errors.
  - Monitoring rows surface collection-health/error detail for `unknown` states and use the same user-facing labels as the quota cards.
  - Threshold oscillation cannot emit alternating warning/recovery notification pairs when hysteresis is configured; defaults preserve existing dedup semantics.
  - Sustained collection failure produces a monitor-degraded signal distinct from quota transition events.

## Authority and Requirements
- authority:
  - Owner instruction on 2026-09-25: current quota monitoring is inaccurate and does not match the quota management screen; no Telegram notification fires on quota exhaustion or threshold crossing; a longer monitoring interval is acceptable in exchange for accuracy; benchmark how other monitoring systems solve this.
  - Prior initiative `docs/plans/done/quota-alert-monitoring.md`: accepted requirements R1–R12 remain binding (database-only configuration, transition-driven deduplicated delivery, no bare-429 exhaustion, credential redaction). Its recorded proof gap defers the `Wake()` request-path hook to this initiative.
  - Root-cause evidence in code: `internal/quotaalert/collector_claude.go:62-68` and `internal/quotaalert/collector_codex.go:91-95` read `access_token`/`id_token` only from `Attributes`, while OAuth token material actually lives in `Auth.Metadata` (`internal/watcher/synthesizer/file.go:138`, `internal/store/postgresstore.go:884-885`); collector tests stub tokens into `attributes` (`internal/quotaalert/collector_claude_test.go:51-53`), masking the production failure.
  - `sdk/cliproxy/quota_alert.go:44-51` admits `runtime_only` virtual auths whose credentials live in `Runtime`, not `Metadata`/`Attributes`; the auth-files API excludes them via `isRuntimeOnlyAuth` (`internal/api/handlers/management/auth_files.go:1187-1191,1339`).
  - `Service.Wake()` (`internal/quotaalert/service.go:150`) has no production callers; only `service_test.go:310` exercises it.
  - Codex monthly-window divergence: `classifyCodexWindows` (`collector_codex.go:186-204`) recognizes only 18000s/604800s while the cards treat 28–31-day windows as monthly (`web/src/components/quota/quotaConfigs.ts:331-401`); Claude `limits[]` fable handling exists only in the cards (`quotaConfigs.ts:1308-1321`).
  - External benchmark references: Prometheus `for:`/staleness, Alertmanager `group_wait`/`group_interval`/`repeat_interval`, Datadog multi-sample evaluation + separate recovery threshold + no-data monitors, AWS Budgets/GCP billing low-frequency quota evaluation, Kubernetes HPA stabilization windows, uptime-monitor consecutive-failure confirmation, SRE multi-window burn-rate alerts.
- requirements:
  - R1 [accepted]: Claude and Codex collectors must resolve credentials from the production layout — Metadata-backed lookup (e.g., `snapshotString` with metadata keys) — and collector tests must build snapshots whose tokens exist only in `Metadata`, never in `Attributes`.
  - R2 [accepted]: The monitored auth set must exclude `runtime_only` auths and any auth whose credential material is unreachable through Attributes/Metadata snapshots; such auths must not produce `unknown` state rows.
  - R3 [accepted]: Qualified request-path quota evidence must trigger an accelerated collection for the affected auth/provider via `Service.Wake()` or an equivalent seam; bare HTTP 429 without provider-specific evidence must not classify exhaustion.
  - R4 [accepted]: Server normalization must match the quota management screen for supported providers: Codex secondary windows of 28–31-day duration map to a monthly/team identity rather than `weekly`, Claude `limits[]`-scoped windows are collected, and persisted resource/window keys map to the same user-facing labels the cards render (shared label map or aligned naming).
  - R5 [accepted]: Failed/`unknown` states must carry a diagnosable, sanitized reason — collection error class and last successful observation time — through the state API and monitoring UI.
  - R6 [accepted]: Evaluation must support optional consecutive-sample confirmation (`for:`-style) for entering `warning`/`exhausted` and a recovery hysteresis margin, so threshold oscillation cannot emit alternating notification pairs; defaults preserve the existing transition/dedup contract.
  - R7 [accepted]: Sustained collection failure across a configurable number of consecutive cycles must produce a visible monitor-degraded signal distinct from quota transition events.
  - R8 [accepted]: Telegram delivery failures must log a sanitized reason and expose a `failure_code` that distinguishes destination misconfiguration (`sender_unavailable`-class) from upstream send errors.
  - R9 [accepted]: All new tunables (confirmation samples, hysteresis margin, failure-signal threshold, and any interval policy) are database-backed quota alert settings; nothing is added to config YAML.
  - R10 [accepted]: Browser quota cards keep their existing fetch path; parity is achieved by aligning server-side normalization and monitoring-page labels, not by rewiring the cards.

## Non-goals
- NG1: Rewire `/quota` provider cards to read `quota_alert_state` or otherwise replace the browser-fetch path (consolidation remains deferred, consistent with the prior plan's rejected alternative).
- NG2: Adaptive or per-window poll scheduling (e.g., poll faster near `reset_at` or near threshold) — accepted trade-off discussion, deferred.
- NG3: New delivery channels (Slack, Discord, email, webhooks, browser push) or multiple Telegram destinations.
- NG4: Per-auth threshold overrides, rule-expression engines, or multi-level escalation policies.
- NG5: Changes to request routing, quota failover, credential selection, or auto-disabling auths based on alert state.
- NG6: New frontend test files under `web/` — verification stays type-check/lint/build plus browser runtime checks.

## Approach and Risks
- approach: Two dependency-ordered phases. Phase 1 (`quota-unblock`) repairs the notification path end-to-end — Metadata credential lookup, runtime-only auth exclusion, the request-result wake seam — and makes every silent failure mode diagnosable (collection failure codes, Telegram delivery code taxonomy). Phase 2 (`quota-parity`) aligns normalization and display labels with the quota management screen, then hardens the evaluator (consecutive-sample confirmation, recovery hysteresis, monitor-degraded signal). Parity work must land after Phase 1: window/resource identity changes produce one-time re-transition events, and those events are only verifiable once delivery actually works. Evaluator tunables reuse the additive-column pattern established by Phase 1's state-schema extension.
- design_decisions:
  - Wake seam: `sdk/cliproxy/builder.go:295` already accepts a `coreauth.Hook` (currently `nil`). Inject a mutable `quotaAlertResultHook` there, then bind the constructed `*quotaalert.Service` after `NewService` (~builder.go:322). `OnResult` wakes only on qualified evidence: `!result.Success` AND (`result.CredentialScope` OR `result.Error.HTTPStatus == 429` OR quota-classified error) AND provider ∈ monitored set; per-auth min-interval gate (default 60s) prevents wake storms during sustained 429s — the capacity-1 `wake` channel already coalesces bursts. No edits inside `sdk/cliproxy/auth` — the existing `Hook` interface is the whole seam.
  - Failure codes (collection): bounded sanitized enum on `Observation`/`CurrentState` — `collector_missing`, `credential_missing`, `upstream_http`, `timeout`, `transport`, `decode`, `internal`. Raw error text never persists; log lines keep the real error server-side only.
  - Failure codes (delivery): split the current binary codes — `telegram_unconfigured` (disabled/missing chat ID/missing token), `telegram_unavailable` (missing `LLMHUB_QUOTA_SECRET_KEY_B64` or decrypt failure), `send_failed` (Telegram HTTP/upstream), `sender_unavailable` (nil sender). Each maps to an i18n label in the Events tab.
  - Parity mechanism: server `resource`/`window` keys stay canonical and stable; a new frontend map `web/src/components/quota/quotaAlertLabels.ts` translates `(provider, resource, window)` → the same i18n `labelKey`s the cards use. Codex secondary windows of 28–31 days get a `monthly` window key (aligns with card id `monthly`); Claude `limits[]` entries collect under a `fable` resource family. Grouped min-remaining granularity stays — parity is identities + labels, not per-model rows.
  - Evaluator tunables (all DB settings, additive columns): `confirmation_samples` int default `1` (= current immediate behavior), `recovery_margin` Percentage default `0` (= current behavior), `degraded_failure_threshold` int default `3`, `0` disables. `ExplicitlyExhausted` observations bypass multi-sample confirmation. Per-state `consecutive_below` counter column persists pending evidence across restarts.
  - Monitor-degraded signal: new `TransitionKind` `monitor_degraded` emitted once per (auth, provider) when a dedicated `quota_alert_collection_health` row's consecutive-failure counter reaches the threshold; counter resets on the next reliable cycle, re-arming the signal. Events ride the existing provider-grouped batch pipeline and Events tab.
- constraints:
  - PostgreSQL remains the sole source of truth for quota-alert settings, state, events, and delivery work; no quota-alert field may be added to config YAML.
  - Logs, API responses, UI state, persisted payloads, and Telegram messages must not expose OAuth tokens, API keys, raw bot tokens, or unredacted credential material.
  - Collection failures and bare HTTP 429 responses cannot produce exhaustion; `unknown` retains the last reliable alert level.
  - Notification creation remains transition-driven: warning entry, exhausted escalation, configured recovery, enabled reminder, or the new monitor-degraded signal; unchanged polls create no new events or batches.
  - Default monitoring stays disabled after schema install; new tunables must have safe defaults that preserve existing behavior for already-configured deployments.
- dependencies:
  - Existing `internal/quotaalert` domain, collectors, evaluator, Telegram outbox, and `internal/store/postgres_quota_alert.go` persistence.
  - Existing request-result/auth-failure surfaces in `sdk/cliproxy/auth` (conductor result classification, cooldown view) for the wake seam; prior plan notes no existing composition surface — the seam may need to be created.
  - Card-side normalization authority in `web/src/components/quota/quotaConfigs.ts` and `web/src/utils/quota/constants.ts` for parity fixtures.
- rejected_alternatives:
  - Shorter polling as an accuracy fix: rejected — correctness bugs dominate; faster polling increases provider rate-limit pressure and `unknown` churn without fixing normalization.
  - Browser-card consolidation (single server-state source for both screens): deferred per prior plan scope decision; this initiative only aligns semantics and labels.
  - Request-hook-only alerting: rejected previously — idle auths would never warn; retained here only as an accelerator with verification.
- risks:
  - risk: No clean request-result composition surface exists for `Wake()`; inventing one may couple the monitor to hot request paths.
    mitigation: Restrict wiring to the existing auth result/cooldown classification seam, require provider-qualified evidence, and cap wake rate (coalesced channel already drops duplicate pending wakes).
    recovery: Ship collection fixes without wake wiring; document the gap explicitly rather than adding speculative coupling.
  - risk: Aligning resource/window keys changes persisted `StateIdentity` values, orphaning or duplicating state/event history.
    mitigation: Treat identity changes as a migration concern — map old keys or accept one-time re-transition events; verify dedup across restart after the change.
    recovery: Keep old keys for alerting-only resources where renaming has no user-facing benefit; confine renames to displayed labels.
  - risk: Confirmation/hysteresis delays genuine exhaustion notifications.
    mitigation: Keep explicit provider exhaustion signals exempt from multi-sample confirmation; bound confirmation to one extra cycle; keep hysteresis opt-in with a conservative default margin.
    recovery: Disable the new evaluator options via database settings without rollback of collectors.
  - risk: Parity work drifts card behavior instead of monitoring behavior.
    mitigation: Card fetch path and rendering stay read-only references; only shared label maps (display-only) may be touched on the card side.
    recovery: Revert label-map sharing; keep server keys canonical.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:

### Phase `quota-unblock` — story_id `quota-unblock-324b9eb1c2a84736`
- status: checked
- goal: Every supported provider produces reliable observations from production-shaped credentials; the monitored auth set matches the Quota Management eligibility boundary; qualified request-path evidence wakes collection; collection and delivery failures carry sanitized, diagnosable codes through state, API, logs, and UI.
- depends_on: none
- allowed_surfaces: `internal/quotaalert/`, `sdk/cliproxy/builder.go`, `sdk/cliproxy/quota_alert.go` (+ new `quota_alert_hook.go`), `internal/store/postgres_quota_alert.go`, `internal/api/handlers/management/quota_alerts.go`, `web/src/types/quotaAlert.ts`, `web/src/services/api/quotaAlerts.ts`, `web/src/pages/QuotaMonitoringPage.tsx`, web i18n locale files.
- avoided_surfaces: `sdk/cliproxy/auth/*` (existing `Hook` interface is the entire seam — no conductor edits), `web/src/components/quota/quotaConfigs.ts`, executor/translator paths, config YAML.

#### Wave 1 — unblock (tasks independent, disjoint surfaces)
- task `t1-metadata-credential-lookup` — traces R1
  - Change `collector_claude.go` / `collector_codex.go` to resolve `access_token`/`id_token` via `snapshotString` over Attributes+Metadata (or add the metadata keys to the cloned key set), matching production layout (`internal/watcher/synthesizer/file.go:138`, `internal/store/postgresstore.go:884-885`).
  - Rewrite collector tests so fixtures place OAuth material only in `Metadata`; add a regression test asserting attributes-only tokens are NOT required.
  - output: both collectors produce `reliable` observations from metadata-only snapshots.
  - check: `go test ./internal/quotaalert/ -run 'TestClaudeCollector|TestCodexCollector' -count=1` green; `grep -n 'attributes\[' internal/quotaalert/collector_claude_test.go internal/quotaalert/collector_codex_test.go` shows no token keys in attribute fixtures.
  - stop: if metadata layout differs per auth source (file vs Postgres), trace both paths before editing — escalate to owner if token location is provider-dependent.

- task `t2-runtime-only-auth-filter` — traces R2
  - In `newQuotaAlertAuthSnapshot` (`sdk/cliproxy/quota_alert.go:44`), exclude auths flagged `runtime_only` (mirror `isRuntimeOnlyAuth` logic in `internal/api/handlers/management/auth_files.go:1187-1191`) and auths whose credential material lives only in `Runtime`.
  - output: virtual Gemini multi-project and other runtime-only auths produce zero state rows.
  - check: `go test ./sdk/cliproxy/ -run 'TestQuotaAlertAuthSource' -count=1` with a runtime_only fixture asserting exclusion.
  - stop: if the `runtime_only` marker is not visible on `coreauth.Auth` fields available at snapshot time, find the actual marker field first — do not guess the field name.

- task `t3-request-result-wake` — traces R3
  - New `sdk/cliproxy/quota_alert_hook.go`: `quotaAlertResultHook` implementing `coreauth.Hook` (embed `NoopHook`); `OnResult` gates on `!result.Success` AND (`CredentialScope` OR `HTTPStatus==429` OR quota-classified error) AND provider ∈ `quotaalert.SupportedProviders()`; per-auth min-interval 60s via in-memory map; then `target.Wake()`. Mutable `atomic.Pointer` target.
  - `builder.go:295` — construct hook before `NewManager`, pass it in; after `NewService` (~line 322) bind the service.
  - output: a 429/quota result on a monitored auth triggers a collection cycle within seconds; bare-429 still cannot classify exhaustion (evaluator unchanged).
  - check: `go test ./sdk/cliproxy/ -run 'TestQuotaAlertResultHook' -count=1` — fake target records wake count; assert gate cases (success, non-quota error, unsupported provider, min-interval dedup).
  - stop: if `OnResult` is invoked while holding `m.mu` (deadlock risk calling back into manager), escalate — hook must not call manager methods; `Wake()` only touches a channel.

#### Wave 2 — diagnosability (depends on wave 1 collectors being correct so unknown states are rare)
- task `t4-collection-failure-visibility` — traces R5
  - Add `FailureCode` + `LastReliableObservedAt` to `CurrentState` (and failure code on `Observation`); classify collector errors in `collectAuthObservations` into the bounded enum (`collector_missing`, `credential_missing`, `credential_rejected` (upstream 401/403 — distinct from absent material), `upstream_http`, `timeout`, `transport`, `decode`, `internal`); `unknownObservations` propagates the code.
  - `postgres_quota_alert.go`: additive `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` on the state table; extend row scan/upsert.
  - `quota_alerts.go` handler + `quotaAlert.ts` DTO + `QuotaMonitoringPage.tsx`: render failure code chip + last-good observation time on `unknown` rows.
  - output: an `unknown` state row shows why and when it last saw real data.
  - check: `go test ./internal/quotaalert/ -count=1`; `LLMHUB_POSTGRES_TEST_DSN` integration test for column round-trip (skips cleanly without DSN); `cd web && bun run type-check`.
  - stop: if state `Normalize()` rejects the new fields, extend invariants rather than bypassing validation.

- task `t5-delivery-failure-taxonomy` — traces R8
  - `telegram_store_sender.go`: replace blanket `ErrTelegramUnavailable` with typed causes; `service.go` `DeliverNotificationsOnce` maps them to `telegram_unconfigured` / `telegram_unavailable` / `send_failed` / `sender_unavailable`; add sanitized Warn log (`failure_code`, provider, batch ID — never token/chat content).
  - Events tab: map `failureCode` to i18n labels.
  - output: Events UI and logs distinguish "bot token missing/decrypt failed" from "Telegram API rejected the send".
  - check: `go test ./internal/quotaalert/ -run 'TestDeliver|TestTelegram' -count=1`; `cd web && bun run type-check`.
  - stop: none — self-contained.

- phase_checks: `go test -race ./internal/quotaalert ./sdk/cliproxy -count=1`; `go test ./...`; `cd web && bun run lint && bun run build`; `git diff --check`.

### Phase `quota-parity` — story_id `quota-parity-2ce32704c9ed4e74`
- status: checked
- goal: Persisted resource/window identities and monitoring-page labels match what the quota cards render (Codex monthly, Claude fable, shared label map); evaluator gains opt-in confirmation samples and recovery hysteresis; sustained collection failure produces a monitor-degraded event.
- depends_on: `quota-unblock` — parity identity changes create one-time re-transition events that are only verifiable once delivery works; T10 reuses T4's failure-code plumbing.
- allowed_surfaces: `internal/quotaalert/` (collectors, evaluator, types, service, telegram), `internal/store/postgres_quota_alert.go`, `internal/api/handlers/management/quota_alerts.go`, `web/src/components/quota/quotaAlertLabels.ts` (new), `web/src/pages/QuotaMonitoringPage.tsx`, `web/src/types/quotaAlert.ts`, web i18n locale files.
- avoided_surfaces: `web/src/components/quota/quotaConfigs.ts` and card renderers (read-only reference — reuse existing labelKeys, do not modify card behavior), `sdk/cliproxy/auth/*`, config YAML.

#### Wave 1 — normalization parity (independent collectors)
- task `t6-codex-monthly-window` — traces R4
  - `classifyCodexWindows` (`collector_codex.go:186-204`): windows of 28–31-day duration → `monthly` window key (mirror `isMonthlyWindow`, `quotaConfigs.ts:399-411`, bounds 2419200–2678400s at lines 331-332); code-review monthly variant likewise.
  - Also fix `appendCodexWindows` for `additional` limits (`classify=false`, collector_codex.go:136): windows are named positionally `five-hour`/`weekly` regardless of duration — a monthly secondary on an additional limit is mislabeled today; classify by `limit_window_seconds` there too.
  - output: monthly/team secondary windows persist `window=monthly`, never `weekly`.
  - check: `go test ./internal/quotaalert/ -run 'TestCodexCollector' -count=1` with monthly-duration fixtures.
  - stop: none.

- task `t7-claude-fable-limits` — traces R4
  - `collector_claude.go`: collect `limits[]`-scoped entries (fable) under a `fable` resource family with the card's window semantics (`quotaConfigs.ts:1308-1321`).
  - Payload restructure required first: `claudeUsagePayload` is `map[string]claudeUsageWindow`, so the `limits` ARRAY fails unmarshal → whole collect errors → `unknown` for fable-plan accounts even after t1. Decode `limits` as a typed field (or `json.RawMessage` per key).
  - Fable match (confirmed from `findFableUsageLimit`, quotaConfigs.ts:1308-1321): `kind=="weekly_scoped"` AND `scope.model.display_name` ∈ {"fable","fable 5"} AND `percent` present; prefer `is_active==true` candidate; `resets_at` when present.
  - Schema-drift guard: keep old-shape fallback and the existing "no recognized windows" error → `decode` failure code (the endpoint already drifted once — `rate_limit_info` top-level shape on 2026-07-05, agent-harness#45).
  - output: fable limits appear as monitoring state rows with correct remaining/reset; accounts whose payload carries `limits[]` decode successfully.
  - check: `go test ./internal/quotaalert/ -run 'TestClaudeCollector' -count=1` with `limits[]` fixtures (active + inactive fable, non-fable scoped limits ignored) and a `rate_limit_info`-shape fixture asserting `decode` failure, not silent healthy.
  - stop: none — payload shape confirmed from repo authority.

#### Wave 2 — labels + evaluator tunables (T8 depends on T6/T7 final key set)
- task `t8-display-label-parity` — traces R4
  - New `web/src/components/quota/quotaAlertLabels.ts`: `(provider, resource, window)` → existing card `labelKey`; `QuotaMonitoringPage.tsx` renders mapped labels with raw-key fallback.
  - output: monitoring rows show "5-hour", "7-day Fable", "Monthly (team)" etc. identical to cards.
  - check: `cd web && bun run type-check && bun run lint && bun run build`; browser spot-check of the monitoring page.
  - stop: if a server key has no card counterpart, document it and fall back to raw key — do not invent new i18n keys beyond what cards already define.

- task `t9-confirmation-and-hysteresis` — traces R6, R9
  - Settings: additive columns `confirmation_samples` (default 1), `recovery_margin` (default 0); validation in `Settings.Validate`; settings API + `QuotaMonitoringPage` settings controls.
  - `CurrentState`: `consecutive_below` counter column; evaluator requires N consecutive below-threshold observations to enter `warning`/`exhausted` (`ExplicitlyExhausted` exempt); recovery requires `remaining > threshold + margin`.
  - output: `confirmation_samples=2` delays a warning one extra cycle; oscillation around 10% with `recovery_margin=5` emits no alternating pairs.
  - check: `go test ./internal/quotaalert/ -run 'TestEvaluate' -count=1` — new table cases for pending, exempt-exhausted, hysteresis recovery; defaults reproduce existing golden transitions.
  - stop: if pending state cannot persist cleanly on `CurrentState`, escalate before adding parallel tables.

#### Wave 3 — monitor health (depends on T4 failure codes + T9 settings pattern)
- task `t10-monitor-degraded-signal` — traces R7, R9
  - New table `quota_alert_collection_health(auth_id, provider, consecutive_failures, last_failure_code, updated_at)`; service increments on `cycle.failed`, resets on reliable cycle.
  - `TransitionKind` += `monitor_degraded` (`from==to==unknown` on the `collection/latest` identity); emitted once at `degraded_failure_threshold` (default 3, 0=off); rides provider-grouped batches + Events tab; settings column + UI control.
  - output: 3 consecutive collection failures for an auth produce one `monitor_degraded` event and Telegram line.
  - check: `go test ./internal/quotaalert/ -count=1`; DSN-gated integration test for counter persistence; `cd web && bun run type-check`.
  - stop: if re-arm semantics interact with reminder_interval dedup, keep degraded events exempt from reminders — one-shot per failure streak.

- phase_checks: `go test -race ./internal/quotaalert ./sdk/cliproxy -count=1`; `go test ./...`; `cd web && bun run type-check && bun run lint && bun run build`; `git diff --check`; DSN-gated `go test ./internal/store/ -run 'TestQuotaAlert' -count=1` when `LLMHUB_POSTGRES_TEST_DSN` is available.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- 2026-09-25 | quota-unblock | wave 1 | phase-start | task_status=in-progress | run anchor 2026-09-25; phase status planned→in-progress
- 2026-09-25 | quota-unblock | wave 1 | t1-metadata-credential-lookup | DONE | `go test ./internal/quotaalert/ -run 'TestClaudeCollector|TestCodexCollector' -count=1` → ok 0.146s | changed: `collector_claude.go`, `collector_codex.go` (CloneAuthSnapshot now includes token metadata keys; reads via `snapshotString` Attributes→Metadata), `collector_claude_test.go`, `collector_codex_test.go` (fixtures carry tokens only in `metadata`, attributes keep `source` only — production shape)
- 2026-09-25 | quota-unblock | wave 1 | t2-runtime-only-auth-filter | DONE | `go test ./sdk/cliproxy/ -run 'TestQuotaAlert' -count=1` → ok 0.064s | changed: `sdk/cliproxy/quota_alert.go` (EqualFold `runtime_only` check mirroring `isRuntimeOnlyAuth`, auth_files.go:1187); test asserts `runtime_only:"TRUE"` gemini-cli virtual excluded
- 2026-09-25 | quota-unblock | wave 1 | t3-request-result-wake | DONE | `go test ./sdk/cliproxy/ -run 'TestQuotaAlertResultHook' -count=1` → ok; gate cases verified (success/unsupported-provider/non-quota ignored; 429, CredentialScope, quota & rate_limit codes wake; 60s per-auth min-interval dedup; unbound target is a no-op) | changed: `sdk/cliproxy/quota_alert_hook.go` (new), `quota_alert_hook_test.go` (new), `builder.go` (hook into `NewManager`, `SetTarget` after `NewService`)
- 2026-09-25 | quota-unblock | wave 1 | wave-summary | DONE | `go build ./internal/quotaalert/ ./sdk/cliproxy/` clean; `go test ./internal/quotaalert/ ./sdk/cliproxy/ -count=1` → ok, ok | wave 1 complete; wake wiring found the composition surface (builder.go `NewManager` hook param) so the R3 deferred gap is closed without touching `sdk/cliproxy/auth`
- 2026-09-25 | quota-unblock | wave 2 | t4-collection-failure-visibility | DONE | `go test ./internal/quotaalert/ -count=1` → ok; `cd web && bun run type-check` → clean | changed: `types.go` (CollectionFailureCode enum + Observation/CurrentState fields + invariants), `service.go` (classifyCollectorError + unknownObservations propagates code), `evaluator.go` (unknown obs → health=unknown + code + last-reliable tracking), `postgres_quota_alert.go` (additive `failure_code` + `last_reliable_observed_at` columns on CREATE+ALTER, scan/upsert/list/lock SELECTs), `quota_alerts.go` (DTO fields), `quotaAlert.ts`, `quotaAlerts.ts` normalize, `QuotaMonitoringPage.tsx` (failure label + last-good tooltip on unknown rows), en/vi locale keys
- 2026-09-25 | quota-unblock | wave 2 | t5-delivery-failure-taxonomy | DONE | `go test ./internal/quotaalert/ -count=1` → ok (deliver + telegram suites) | changed: `telegram_store_sender.go` (`ErrSenderUnavailable`/`ErrTelegramUnconfigured`/`ErrTelegramUnavailable` sentinels split by cause), `types.go` (DeliveryFailureCode enum), `service.go` (`classifyDeliveryError`: config/decrypt/sender errors → permanent immediately, upstream → `send_failed` retry-until-max; Warn log with failure_code+provider+batch_id), `QuotaMonitoringPage.tsx` (`delivery_failure_<code>` i18n labels), en/vi locale keys
- 2026-09-25 | quota-unblock | wave 2 | wave-summary | DONE | `go test -race ./internal/quotaalert ./sdk/cliproxy -count=1` → ok, ok; `bun run lint` → 0 errors (8 pre-existing warnings); `bun run build` → ok; `git diff --check` → clean | wave 2 complete; retained-state semantics changed deliberately — unknown collection now flips health to `unknown` while keeping the alert value, so stale-but-failing rows are diagnosable instead of silently green
- 2026-09-25 | quota-unblock | gate | phase-gate | APPROVED | phase status in-progress→checked; full `go test ./...` green; Validation entry recorded | quota-unblock complete; advanced to quota-parity
- 2026-09-25 | quota-parity | wave 1 | phase-start | task_status=in-progress | phase status planned→in-progress
- 2026-09-25 | quota-parity | wave 1 | t6-codex-monthly-window | DONE | `go test ./internal/quotaalert/ -run 'TestCodexCollector' -count=1` → ok (new TestCodexCollectorClassifiesMonthlyWindows) | changed: `collector_codex.go` — `classifyCodexWindows` now returns named windows by duration (5h→five-hour, 7d→weekly, 2419200–2678400s→monthly) for ALL limits including additional; positional fallback never relabels a duration-claimed window; monthly fixtures assert code/monthly + additional-team-plan/monthly, no weekly/five-hour mislabels
- 2026-09-25 | quota-parity | wave 1 | t7-claude-fable-limits | DONE | `go test ./internal/quotaalert/ -run 'TestClaudeCollector' -count=1` → ok (new TestClaudeCollectorCollectsFableLimits + TestClaudeCollectorRejectsDriftedPayloadShape) | changed: `collector_claude.go` — `claudeUsagePayload` restructured to custom UnmarshalJSON (raw map; `limits` decodes as typed array, unknown keys decode-tolerant → latent `limits[]` decode bug fixed); `findClaudeFableLimit` mirrors findFableUsageLimit (weekly_scoped + display_name fable/fable 5 + percent, prefers is_active); iguana_necktie skipped when fable exists (card parity); fable rows persist as resource=fable/window=seven-day; rate_limit_info-only payload → error → `decode` failure code
- 2026-09-25 | quota-parity | wave 2 | t8-display-label-parity | DONE | `bun run type-check`/`lint`/`build` → clean | changed: `web/src/components/quota/quotaAlertLabels.ts` (new — `(provider, resource, window)` → existing card labelKeys: claude `claude_quota.*` incl. iguana-necktie+fable→`seven_day_fable`; codex `code`/`code-review`/`additional-*` → `codex_quota.*` with `{name}` humanized from the additional-limit slug; dynamic providers return undefined → raw fallback), `QuotaMonitoringPage.tsx` Window column renders mapped label + raw `resource/window` tooltip
- 2026-09-25 | quota-parity | wave 2 | t9-confirmation-and-hysteresis | DONE | `go test ./internal/quotaalert/ -count=1` → ok (5 new evaluator tests + 7 new validation cases); `go test -race ./internal/quotaalert` → ok | changed: `types.go` (`confirmation_samples` 1..10 default 1, `recovery_margin` 0..100 default 0, threshold+margin≤100 incl. per-override; `CurrentState.ConsecutiveBelow` + normalize invariant), `evaluator.go` (severity-gated confirmation — escalation into warning/exhausted requires N consecutive below-threshold samples, `ExplicitlyExhausted` bypasses, recovery to healthy requires remaining > threshold+margin), `postgres_quota_alert.go` (additive `consecutive_below` on state + `confirmation_samples`/`recovery_margin` on settings — CREATE+ALTER, all 4 SELECTs, upsert incl. unchanged-detection clause), `quota_alerts.go` (settings+state DTO fields; nil request fields → defaults), frontend types/api/page (settings controls + `confirming n/m` hint on held rows), en/vi locale keys
- 2026-09-25 | quota-parity | wave 2 | wave-summary | DONE | `go test ./...` → all green; `go test -race ./internal/quotaalert ./sdk/cliproxy` → ok, ok; `bun run type-check`/`lint`/`build` → clean; `git diff --check` → clean | wave 2 complete; note: `go test -race ./internal/store` shows a PRE-EXISTING race in `TestEnsureRepositoryKeepsCurrentBranchWhenRemoteDefaultCannotBeResolved` (git-store backend, reproduces on clean tree — unrelated to quota alert SQL)
- 2026-09-25 | quota-parity | wave 3 | t10-monitor-degraded-signal | DONE | `go test ./internal/quotaalert -count=1` → ok (new TestServiceMonitorDegradedSignalCrossesThresholdOncePerStreak + TestServiceMonitorDegradedDisabledNeverFires + TestTelegramMonitorDegradedMessageSkipsQuotaFields); `go test -race ./internal/quotaalert ./sdk/cliproxy` → ok; `go test ./...` → all green; `bun run type-check`/`lint`/`build` → clean; `git diff --check` → clean | changed: `types.go` (TransitionMonitorDegraded kind + CollectionHealthKey/Record types + `degraded_failure_threshold` settings 0..100 default 3), `service.go` (`advanceCollectionHealth` — per-auth streak increments on failed cycles, deletes on reliable; emits one `monitor_degraded` event at threshold crossing on `collection/latest` identity, appended to `collection.observed` when no prior rows exist), `postgres_quota_alert.go` (new `quota_alert_collection_health` table + `ListCollectionHealth`/upsert/delete inside CommitCollection txn; events `kind` CHECK dropped+recreated with `monitor_degraded`; transition-history validation exemption for no-history degraded events), `telegram.go` (🛠️ icon + dedicated "Monitoring degraded" line skipping quota fields), handler DTO + frontend types/api/settings control + `event_kind_monitor_degraded` i18n (en/vi), `quota_alert_route_test.go`+`quota_alerts_test.go` stub impls, `postgres_quota_alert_integration_test.go` health round-trip test (DSN-gated)

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- 2026-09-25 | to-plan research sharpening (owner-requested) | Findings folded into task specs:
  - Claude `limits[]` is a latent decode bug, not just an omission: `claudeUsagePayload = map[string]claudeUsageWindow` cannot unmarshal the `limits` array → whole collect fails → `unknown` for fable-plan accounts even after the t1 token fix. T7 now includes the payload restructure. (Source: `collector_claude.go:36` + `quotaConfigs.ts:1308-1321`.)
  - Fable limit shape confirmed from repo authority — `kind=="weekly_scoped"`, `scope.model.display_name ∈ {"fable","fable 5"}`, `percent`, `is_active` preference (`findFableUsageLimit`, `quotaConfigs.ts:1308-1321`). T7 stop condition resolved.
  - Codex monthly bounds confirmed: `MIN_MONTH_SECONDS=2419200`, `MAX_MONTH_SECONDS=2678400` (`quotaConfigs.ts:331-332`). Additional finding: `appendCodexWindows` names additional-limit windows positionally (`five-hour`/`weekly`) without duration classification — T6 now covers that path too.
  - Claude endpoint schema drift is a real precedent: on 2026-07-05 `/api/oauth/usage` began returning top-level `rate_limit_info` (`isUsingOverage`, `overageResetsAt`, `overageStatus`, `rateLimitType`, `resetsAt`); parsers silently went nil → fail-closed gates (github.com/nyelbangash/agent-harness issue #45). Validates R7's monitor-degraded signal and adds a `decode`-failure guard to T7 (old-shape fallback; nil-parse errors, never silent healthy).
  - Independent design validation from `sealofyou/cpa-quota-alert-plugin` (CLIProxyAPI native plugin): low threshold 1.5 / recovery 1.6 (separate recovery threshold = our `recovery_margin`), reminder every 24h, failure-alert-count 3 (= our `degraded_failure_threshold` default), terminal error codes (`token_invalidated`, `token_revoked`) — hence `credential_rejected` added to the T4 enum to distinguish revoked credentials from missing ones. Its `data_error` event kind ≈ our `monitor_degraded`; per-account failures stay partial (matches our per-auth `cycle.failed`). 5-minute timer cadence matches our default poll interval.
  - Community usage-endpoint header parity (optional, zero-cost): callers send `User-Agent: claude-code/x.y.z` alongside `anthropic-beta: oauth-2025-04-20` (gist.github.com/chrisns/3400a77287bbafb400b87810b006e7e5). Not required for correctness.
- 2026-09-25 | quota-unblock/wave 1 execution | Findings during t1–t3:
  - `CollectorDeps.Refresh` is never populated — `builder.go` leaves `ServiceConfig.CollectorDeps` zero-valued, so collectors run with `refresh == nil` and a 401 on a usage endpoint yields `credential_rejected`/`unknown` forever with no in-monitor refresh path. Deferred rather than fixed here: refresh mutates auth state and needs a manager-bound callback, a separate design decision. t4's `credential_rejected` code and t10's degraded signal make it visible; wiring refresh is tracked as an open item.
  - Custom `b.coreManager` supplied via SDK embedding gets no wake hook — `coreauth.NewManager` takes the hook only at construction and no `sdk/cliproxy/auth` edits are allowed. Accepted: embedded deployments keep poll-only monitoring (same as before); default server path is wired.
  - Hook dedup state is in-memory per-process (`lastWake` map with 60s min-interval, swept at 256 entries); restart resets it — acceptable since `Wake()` is a coalescing hint, not a dedup guarantee.
- 2026-09-25 | quota-unblock/wave 2 execution | Findings during t4–t5:
  - Retained-state semantics changed: pre-t4, an unknown collection kept `health=reliable` on the retained row so failing collectors looked indistinguishable from fresh data. Now `health` flips to `unknown` + `failure_code` while `alert` keeps its last reliable value; `observed_at` stays the last reliable observation time and `last_reliable_observed_at` backfills from it for pre-migration rows. This is the visibility feature itself — staleness must be inspectable.
  - Config-class delivery failures are permanent on first failure: `telegram_unconfigured` (disabled/no chat/no stored token), `telegram_unavailable` (missing key/cipher/decrypt), and `sender_unavailable` cannot self-heal on retry — marking them permanent immediately surfaces the real code in the Events tab instead of burning 3 retries as `send_failed`. Only upstream send errors retry to `MaxNotificationAttempts`.
- 2026-09-25 | quota-parity/wave 2 execution | Findings during t8–t9:
  - Confirmation gate semantics: `consecutive_below` counts consecutive reliable below-threshold observations on the durable `CurrentState`; escalation requires `counter >= confirmation_samples` (severity-rank compare so warning→exhausted escalation is also gated; severity decreases never wait). `ExplicitlyExhausted` bypasses — provider evidence is authoritative. Unknown collections retain the counter (a failure is not evidence quota recovered), so confirmation resumes across collection gaps. Non-explicit `remaining==0` still requires confirmation — a 0 payload could be drift.
  - Hysteresis: recovery to healthy requires `remaining > threshold + recovery_margin`; `threshold + margin ≤ 100` enforced in `Settings.Validate` globally and per provider-override, else recovery would be unreachable. `recovery_margin=0` and `confirmation_samples=1` reproduce pre-t9 behavior exactly (defaults are the unchanged path).
  - Label map scope: only claude + codex have fixed identity maps (other providers emit dynamic model-id resources → safe raw fallback). `iguana-necktie/default` maps to `claude_quota.seven_day_fable` because the cards treat `iguana_necktie` as the fable row (`constants.ts` `CLAUDE_USAGE_WINDOW_KEYS`).

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- 2026-09-25 | quota-unblock | gate | APPROVED | judge: same-session | judge_model: SWE-2 Max
  - `go test -race ./internal/quotaalert ./sdk/cliproxy -count=1` → ok internal/quotaalert 2.535s; ok sdk/cliproxy 1.643s
  - `go test ./... -count=1` → exit 0; every package ok, no failures
  - `cd web && bun run lint` → 0 errors, 8 pre-existing warnings, none in changed files
  - `cd web && bun run build` → tsc + vite build ok; dist/index.html 2172 kB
  - `cd web && bun run type-check` → clean
  - `git diff --check` → clean
  - `go test ./internal/store/ -run 'TestQuotaAlert' -count=1` → skips cleanly; LLMHUB_POSTGRES_TEST_DSN unset
  - scope: on target — 21 files, all inside allowed surfaces (`internal/quotaalert/`, `sdk/cliproxy/{builder,quota_alert,quota_alert_hook}`, `internal/store/postgres_quota_alert.go`, management handler, web types/api/page/locales); no avoided-surface edits; no config-YAML changes
  - alignment: t4/t5 outputs landed as specified — failure codes on unknown states end-to-end, delivery taxonomy split with permanent-vs-retry semantics, i18n event labels; retained-state semantics change (health flips unknown while alert retains) recorded in Decisions
  - proof_gaps: Postgres round-trip of `failure_code`/`last_reliable_observed_at` and ResolveNotification failure-code persistence unverified without DSN; browser runtime check not performed (type-check/lint/build only)
  - receipt: context_sources=plan+diff+phase_checks / policy=work-full phase gate / judge=same-session / judge_model=SWE-2 Max / retries=0 / rollback_point=HEAD / failure_ledger: absent / enforcement: local-only / not_independently_verified: Postgres column round-trip (no DSN), browser-rendered UI, independent review
- 2026-09-25 | quota-parity | gate | APPROVED | judge: same-session | judge_model: SWE-2 Max
  - `go test -race ./internal/quotaalert ./sdk/cliproxy -count=1` → ok internal/quotaalert 2.668s; ok sdk/cliproxy 1.804s
  - `go test ./... -count=1` → all packages ok, no failures
  - `cd web && bun run type-check` → clean; `bun run lint` → 0 errors, 8 pre-existing warnings; `bun run build` → ok dist/index.html 2177 kB
  - `git diff --check` → clean; `gofmt -l` on touched files → clean (pre-existing drift in 5 untouched files left alone)
  - `go test ./internal/store/ -run 'TestPostgresQuotaAlert' -count=1` with throwaway postgres:16 container DSN → all pass incl. fresh-schema CREATEs, ALTER idempotence, new `TestPostgresQuotaAlertCollectionHealthRoundTrip` + `TestPostgresQuotaAlertMonitorDegradedEventCommitsWithoutPriorHistory` (CHECK constraint + validator exemption verified on real DB); container removed after run
  - scope: on target — 26 files + 3 new (plan dir, quota_alert_hook.go+test, quotaAlertLabels.ts), all inside allowed surfaces; no card/`quotaConfigs.ts` edits; no config-YAML changes
  - alignment: t6–t10 outputs landed as specified — duration-based window naming, fable `limits[]` collection with decode-bug fix, shared label map with raw fallback, confirmation+margin tunables, `monitor_degraded` one-shot streak signal exempt from reminder dedup
  - proof_gaps: browser-rendered UI not exercised (type-check/lint/build only); `internal/store` git-backend race reproduced on clean tree — pre-existing, unrelated
  - receipt: context_sources=plan+diff+phase_checks+postgres16-dsn / policy=work-full phase gate / judge=same-session / judge_model=SWE-2 Max / retries=1 (degraded fixture missing TransitionedAt — normalize invariant, fixture fixed) / rollback_point=HEAD / failure_ledger: absent / enforcement: local-only / not_independently_verified: independent review
- 2026-09-25 | quota-parity | browser-verify | APPROVED | judge: same-session | judge_model: SWE-2 Max
  - env: throwaway postgres:16 container + `llmhub init-db-from-env` seeded config + built binary serving current embedded UI; Playwright 1.63 headless chromium; teardown complete
  - server log: `quota alert monitor started` under real Postgres runtime
  - `GET /v0/management/quota-alerts/settings` → 200 with `confirmation_samples:1, recovery_margin:0, degraded_failure_threshold:3`
  - states tab (4 seeded rows incl. unknown+failure_code): `fable/seven-day`→"7-day Fable", `messages/five-hour`→"5-hour limit", `sonnet/seven-day`→"7-day Sonnet", `code/monthly`→"Monthly limit" (= `codex_quota.team_secondary_window`, card-identical), `code-review/weekly`→"Code review weekly limit"; unknown row shows "Upstream HTTP error" label; `consecutive_below=2` + `confirmation_samples=3` renders "confirming 2/3"
  - settings tab: all three new inputs render; save via UI → API returns `{3,5,2}` → reload → persisted values re-render — full UI→API→Postgres round-trip
  - events tab renders; zero console errors across all flows
  - receipt: playwright headless chromium / policy=owner-requested browser check / retries=0 / not_independently_verified: independent review only
- 2026-09-25 | post-completion | open-items | DONE | `go test ./...` → all green; `go test -race ./internal/quotaalert ./sdk/cliproxy ./sdk/cliproxy/auth ./internal/store` → ok ×4 | closed remaining open items: (1) `CollectorDeps.Refresh` wired — `CollectorRefreshFunc` now returns a fresh `AuthSnapshot`, `JSON`/`JSONBody` take `headersFor func(AuthSnapshot)` rebuilt per attempt, 401→refresh→retry carries renewed credentials; `NewQuotaAlertAuthRefresher` binds `coreauth.Manager.ForceRefreshAuth` and re-snapshots the refreshed auth (verified by `TestQuotaAlertAuthRefresherReturnsRenewedSnapshot` + retry-header assertion in `collectors_test.go`); all 7 collectors converted to dynamic header builders. (2) Custom `coreManager` wake hook — `Manager.AddHook` composes late hooks atomically after the construction hook (`hookChain`, `atomic.Pointer`); builder calls `AddHook(quotaAlertHook)` on the custom-manager path (verified by `TestManagerAddHookComposesAfterConstructionHook`). (3) Pre-existing `internal/store` git-backend race fixed — was upstream go-git local-transport `stderrBuf` race on any filesystem remote; test helpers now serve remotes through `git daemon` (`git://` TCP transport) + push via `git` CLI; `go test -race ./internal/store` clean.
- 2026-09-25 | post-completion | NG2-adaptive-polling | DONE | `go test -race ./internal/quotaalert` → ok; `TestAdaptivePollIntervalShortensNearThresholdAndReset` → pass (9 cases: near-threshold, cool, near/far/stale reset, unknown-health skip, disabled-provider skip, override threshold, min floor) | NG2 closed — `adaptivePollInterval` in `service.go` shortens the cycle to `max(MinPollInterval, base/5)` while any reliable enabled-provider window is hot (reset within ±1 base interval, or remaining ≤ threshold + 10pt headroom incl. per-provider overrides); `nextPollInterval` reads settings+states after every cycle on both wake and timer paths, falls back to base on any read failure or monitoring disabled; no new settings — adaptive only ever shortens, never lengthens.

## Current State and Next Action
- active_phase: quota-parity
- lifecycle_status: completed — both phases checked; Postgres + browser verification closed all proof gaps
- latest_run_id: none
- latest_trace_ids: none
- latest_check_id: none
- latest_handoff_id: none
- completed_work:
  - Diagnosed quota-monitoring inaccuracy and Telegram silence to concrete root causes: attribute-only credential reads in Claude/Codex collectors masked by test stubs; `runtime_only` virtual auths admitted to monitoring; unwired `Wake()` seam; Codex monthly/Claude `limits[]` normalization divergence; silent collection-failure states; missing confirmation/hysteresis and monitor-degraded signaling.
  - quota-unblock wave 1 (t1–t3): metadata credential lookup for Claude/Codex, runtime_only auth exclusion, request-result wake hook wired through `coreauth.Hook` at builder.go.
  - quota-unblock wave 2 (t4–t5): collection failure codes (`collector_missing`/`credential_missing`/`credential_rejected`/`upstream_http`/`timeout`/`transport`/`decode`/`internal`) threaded observation→state→Postgres→API→monitoring UI with last-reliable-observation timestamps; Telegram delivery taxonomy (`telegram_unconfigured`/`telegram_unavailable`/`send_failed`/`sender_unavailable`) with permanent-vs-retry semantics and i18n event labels.
  - quota-parity wave 1 (t6–t7): duration-based Codex window naming (monthly incl. additional/code-review limits); Claude `limits[]` fable collection + payload restructure fixing the latent decode bug.
  - quota-parity wave 2 (t8–t9): shared label map `quotaAlertLabels.ts` rendering card-identical labels on monitoring rows; `confirmation_samples` + `recovery_margin` persisted tunables gating alert escalation (durable `consecutive_below` counter) and hysteresis recovery, exposed through API + settings UI.
  - quota-parity wave 3 (t10): `quota_alert_collection_health` table + per-auth failure streaks; `monitor_degraded` event fires once at `degraded_failure_threshold` (default 3, 0=off) on `collection/latest` identity, re-arms after a reliable cycle; dedicated Telegram line + i18n event label + settings control.
- blockers: none
- proof_gaps:
  - None — Postgres verified against a throwaway postgres:16 container (fresh CREATEs, ALTER idempotence, health round-trip, `monitor_degraded` event commit); browser-rendered UI verified via headless Chromium (label parity, failure labels, confirming hint, settings save round-trip, zero console errors). `internal/store` race resolved post-completion (git-daemon transport in test helpers).
- open_items: none — NG2 adaptive polling implemented post-completion (`adaptivePollInterval`: reliable window polls at max(MinPollInterval, base/5) while reset lands within one base interval or remaining sits within 10 points of the effective threshold; unknown-health and disabled-provider rows never shorten; run loop reschedules adaptively after every wake and timer cycle).
- exact_next_action: none — plan complete, all items closed.
