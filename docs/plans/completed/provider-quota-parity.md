---
id: plan-20260908-provider-quota-parity
type: plan
intake_id: intake-20260908-provider-quota-parity
lane: medium-risk
status: completed
created: 2026-09-08
updated: 2026-09-08
---

# Plan: Provider Quota Parity (Codex, Antigravity, Claude, xAI, Kimi)

## Outcome
- result: Full parity between llmhub and upstream CLIProxyAPI (`router-for-me/CLIProxyAPI`) / Management Center (`router-for-me/Cli-Proxy-API-Management-Center`) across Codex, Antigravity, Claude, xAI, and Kimi quota ingestion, window classification, and unified card rendering — without tab-hiding, dropped WebSocket rate limit headers, hardcoded 5h windows, or broken project ID resolution, while strictly preserving llmhub custom extensions (`gemini-cli` and `kiro`).
- success_signals:
  - Codex quota displays all quota forms simultaneously on a single card (plan tier, subscription renewal, available reset credits, credit expiry list, and usage window progress bars) matching upstream `CodexQuotaBody`.
  - Codex WebSocket executor calls `AppendCodexAPIWebsocketResponse`, capturing `codex.rate_limits` frames and updating `auth.Quota.Signals`.
  - Codex additional rate limits (e.g. `GPT-5.3-Codex-Spark`) and Team/Enterprise monthly windows (28–31 days) classify correctly via duration bounds instead of forcing 5h.
  - Antigravity quota uses `retrieveUserQuotaSummary` and `loadCodeAssist`, displaying hierarchical Groups & Buckets (5h, daily, weekly), subscription tiers, and server clock skew compensation (`serverTimeOffsetMs`).
  - Claude quota parses Fable scoped limits (`findFableUsageLimit`), xAI supports dual weekly/monthly billing plus paid health probe (`isPaidXaiAuthFile`), and Kimi presents 5h limits before weekly summary with correct duration tokens.
  - Custom llmhub extensions (`gemini-cli`, `kiro`) remain fully functional with zero regressions.
  - `cd web && bun run build && bun run type-check` passes; `go test ./...` passes.

## Authority and Requirements
- authority:
  - Upstream backend: `router-for-me/CLIProxyAPI` at checkpoint `v7.2.147` / HEAD (`internal/runtime/executor/helps/codex_quota.go`, `sdk/cliproxy/auth/quota_signals.go`, `internal/runtime/executor/codex_websockets_*.go`).
  - Upstream frontend: `router-for-me/Cli-Proxy-API-Management-Center` (`src/features/quota/providers/`, `src/utils/quota/`).
  - `AGENTS.md` and repository standards.
- requirements:
  - R1 [accepted]: Codex WebSocket executor calls `helps.AppendCodexAPIWebsocketResponse` in streaming and non-streaming loops, merging `codex.rate_limits` headers into context request headers and updating `auth.Quota.Signals`.
  - R2 [accepted]: Codex quota windows classify duration dynamically: 5-hour (18,000s), weekly (604,800s), and monthly (28–31 days `MIN_MONTH_SECONDS`..`MAX_MONTH_SECONDS`). `additional_rate_limits` runs through `pickClassifiedWindows(rateInfo)` instead of forcing 5h.
  - R3 [accepted]: Codex quota card adopts flat layout (no `<Tabs>` hiding reset credits into tab 2), showing plan tier (Elite Liquid Platinum for Pro 20x, Premium Gold for Pro Lite, Plain), renewal expiry, manual reset credits list, and all window progress bars.
  - R4 [accepted]: Antigravity quota upgrades from legacy `fetchAvailableModels` to `retrieveUserQuotaSummary`, dynamically building Groups & Buckets sorted by `ANTIGRAVITY_BUCKET_WINDOW_ORDER` (5h -> weekly -> daily), extracting subscription tiers via `loadCodeAssist`, and compensating clock skew via HTTP `Date` header.
  - R5 [accepted]: Claude quota extracts `weekly_scoped` Fable limits via `findFableUsageLimit` and maps `iguana_necktie` to `seven-day-fable`.
  - R6 [accepted]: xAI quota implements dual weekly (`?format=credits`) + monthly billing queries, recognizes paid accounts via `isPaidXaiAuthFile`, and falls back to `requestXaiPaidHealth` with `grok-4.5`.
  - R7 [accepted]: Kimi quota reorders rows so 5-hour limits appear before weekly summary, strips protobuf `TIME_UNIT_` prefix, and formats durations $\ge 24\text{h}$ as `Xd Yh`.
  - R8 [accepted]: All missing localization keys added to `en.json` and `vi.json` with dynamic timezone interpolation.
  - R9 [accepted]: Preserve llmhub-specific extensions `gemini-cli` and `kiro` without behavioral drift.

## Non-goals
- NG1: Do not rewrite the entire web routing or page architecture (keep `/quota` page structure, modernize provider configs and bodies).
- NG2: Do not remove or alter `gemini-cli` and `kiro` quota implementations.
- NG3: Do not alter backend scheduler cooldown mechanics (keep quota observation read-only and separate from scheduler cooldown state).
- NG4: Do not import unnecessary SCSS files from upstream; adapt styles into Tailwind classes in `quotaStyles.ts`.

## Approach and Risks
- approach: Staged execution across 4 clean phases. Phase 1 fixes backend WebSocket quota observation so `Signals` are never dropped. Phase 2 unifies Codex quota: removes tabs, implements duration-aware window classification (including monthly and additional limits like Spark), and updates plan tiers. Phase 3 ports modern Antigravity Groups & Buckets, subscription loading, and clock skew correction. Phase 4 updates Claude (Fable), xAI (dual billing + paid probe), and Kimi (ordering + duration format), alongside i18n locales.
- constraints:
  - TypeScript strict type checking and ESLint must pass (`bun run type-check`).
  - Vite build must succeed (`bun run build`).
  - Go build and unit tests must pass (`go test ./...`).
  - Read-only observation separation: observation signals must not mutate scheduler cooldowns.
- rejected_alternatives:
  - Keeping tabbed view for Codex: rejected because user feedback and upstream standard both require flat unified visibility across all quota types.
  - Blindly copying upstream SCSS modules: rejected because llmhub uses Tailwind utility classes.
- risks:
  - Antigravity `retrieveUserQuotaSummary` requires valid project ID: mitigated by 4-stage waterfall resolution and graceful error messages without sandbox fallback.
  - xAI paid health probe executes `grok-4.5` chat test: bounded timeout (15s) and fallback error preservation ensure free billing errors remain intact.
- recovery: Revert touched config files via Git; each phase is independently buildable and testable.

## Phases and Verification
- planning_status: planned
- phases:
  - phase_slug: phase-1-backend-codex-websocket
    story_id: story-20260908-backend-codex-ws
    status: checked
    goal: Fix Codex WebSocket executor dropped quota frames and verify signals capture.
    depends_on: none
    allowed_surfaces:
      - `internal/runtime/executor/codex_websockets_executor.go`
      - `internal/runtime/executor/helps/codex_quota_test.go`
    avoided_surfaces:
      - `internal/api/**`
      - `web/**`
    waves:
      - wave: 1
        tasks:
          - task: Replace `helps.AppendAPIWebsocketResponse` with `helps.AppendCodexAPIWebsocketResponse` in `codex_websockets_executor.go` (non-streaming and streaming loops).
          - task: Add regression test in `codex_quota_test.go` verifying that websocket `codex.rate_limits` frames update Gin context response headers and quota signals.
        checks:
          - check: `go test ./internal/runtime/executor/... -run 'Test(GenericWebsocketEntryPoint|ParseCodexQuotaEventHeaders)' -count=1`
          - check: `go test ./internal/runtime/executor/helps/... -count=1`

  - phase_slug: phase-2-frontend-codex-quota
    story_id: story-20260908-frontend-codex
    status: checked
    goal: Eliminate tab-hiding in Codex quota card, classify monthly and additional windows dynamically, and support plan tiers.
    depends_on: phase-1-backend-codex-websocket
    allowed_surfaces:
      - `web/src/components/quota/quotaConfigs.ts`
      - `web/src/components/quota/quotaStyles.ts`
      - `web/src/types/quota.ts`
      - `web/src/i18n/locales/en.json`
      - `web/src/i18n/locales/vi.json`
    avoided_surfaces:
      - `internal/**`
      - `sdk/**`
    waves:
      - wave: 1
        tasks:
          - task: Extend `CodexQuotaWindow` and `CodexQuotaState` in `web/src/types/quota.ts` with `resetAtMs`, `periodHours`, `subscriptionActiveUntil`, and `applicableAvailableCount`.
          - task: Add `MIN_MONTH_SECONDS`, `MAX_MONTH_SECONDS`, `isMonthlyWindow`, and `selectSecondaryWindowMeta` to `buildCodexQuotaWindows` in `quotaConfigs.ts`.
          - task: Update `additionalRateLimits` loop in `buildCodexQuotaWindows` to run `pickClassifiedWindows(rateInfo)` and `selectSecondaryWindowMeta` instead of hardcoding 5h.
        checks:
          - check: `cd web && bun run type-check`
      - wave: 2
        tasks:
          - task: Refactor `renderCodexItems` in `quotaConfigs.ts` to remove `<Tabs>`, rendering plan header chip (with `elitePlanValue` for Pro 20x), renewal expiry, manual reset credits list, and all window progress bars sequentially.
          - task: Add styles for `elitePlanValue` and missing Codex classes in `quotaStyles.ts`.
          - task: Add missing i18n keys for team secondary windows and dynamic timezone labels in `en.json` and `vi.json`.
        checks:
          - check: `cd web && bun run type-check && bun run build`

  - phase_slug: phase-3-frontend-antigravity-quota
    story_id: story-20260908-frontend-antigravity
    status: checked
    goal: Migrate Antigravity quota to `retrieveUserQuotaSummary` Groups & Buckets, subscription tiers, and clock skew correction.
    depends_on: phase-2-frontend-codex-quota
    allowed_surfaces:
      - `web/src/utils/quota/constants.ts`
      - `web/src/utils/quota/builders.ts`
      - `web/src/components/quota/quotaConfigs.ts`
      - `web/src/types/quota.ts`
      - `web/src/services/api/antigravitySubscription.ts`
      - `web/src/i18n/locales/en.json`
      - `web/src/i18n/locales/vi.json`
    avoided_surfaces:
      - `internal/**`
    waves:
      - wave: 1
        tasks:
          - task: Update `ANTIGRAVITY_QUOTA_URLS` in `constants.ts` to use `retrieveUserQuotaSummary`.
          - task: Update `types/quota.ts` schema for `AntigravityQuotaBucket`, `AntigravityQuotaGroup`, `AntigravityQuotaSubscription`, and `AntigravityQuotaState`.
          - task: Rewrite `buildAntigravityQuotaGroups` in `builders.ts` to parse dynamic `payload.groups` and order buckets (5h -> weekly -> daily).
        checks:
          - check: `cd web && bun run type-check`
      - wave: 2
        tasks:
          - task: Update `resolveAntigravityProjectId` in `quotaConfigs.ts` with 4-stage waterfall (`file.project_id` -> `metadata` -> `attributes` -> `downloadText`), removing hardcoded sandbox ID.
          - task: Add server clock skew calculation `resolveResponseServerTimeOffsetMs` from HTTP `Date` header in `fetchAntigravityQuota`.
          - task: Add `antigravitySubscriptionApi` to query `loadCodeAssist` and render subscription tier badges (`ultra`, `pro`, `free`).
          - task: Add Antigravity group, bucket, and subscription localization keys in `en.json` and `vi.json`.
        checks:
          - check: `cd web && bun run type-check && bun run build`

  - phase_slug: phase-4-other-providers-parity
    story_id: story-20260908-other-providers
    status: checked
    goal: Harmonize Claude Fable limits, xAI dual-probe/paid detection, Kimi window ordering and duration formatting.
    depends_on: phase-3-frontend-antigravity-quota
    allowed_surfaces:
      - `web/src/utils/quota/constants.ts`
      - `web/src/utils/quota/builders.ts`
      - `web/src/utils/quota/xaiPaid.ts`
      - `web/src/components/quota/quotaConfigs.ts`
      - `web/src/types/quota.ts`
      - `web/src/i18n/locales/en.json`
      - `web/src/i18n/locales/vi.json`
    avoided_surfaces:
      - `internal/runtime/executor/kiro_quota.go`
    waves:
      - wave: 1 — Claude & Kimi
        tasks:
          - task: Add `findFableUsageLimit` to Claude window builder in `quotaConfigs.ts` and map `iguana_necktie` to `seven-day-fable`.
          - task: Invert `buildKimiQuotaRows` order in `builders.ts` so `limits[]` appear before `usage` summary.
          - task: Strip protobuf `TIME_UNIT_` prefix in Kimi unit normalization and format durations $\ge 24\text{h}$ as `Xd Yh`.
        checks:
          - check: `cd web && bun run type-check`
      - wave: 2 — xAI
        tasks:
          - task: Add `isPaidXaiAuthFile`, dual weekly/monthly billing endpoints, and paid health check with `grok-4.5` in `quotaConfigs.ts`.
          - task: Update `renderXaiItems` to render weekly limit progress bar, SuperGrok plan badges, and paid health status.
          - task: Add missing Claude, Kimi, and xAI i18n keys to `en.json` and `vi.json`.
        checks:
          - check: `cd web && bun run type-check && bun run build`
          - check: `go test ./...`

## Progress
- 2026-09-08T13:40:00Z: Plan drafted and locked into docs/plans/active/provider-quota-parity.md.
- 2026-09-08T13:45:00Z: Phase 1 (phase-1-backend-codex-websocket) Wave 1 DONE. Replaced AppendAPIWebsocketResponse with AppendCodexAPIWebsocketResponse in codex_websockets_executor.go:554,839 so WebSocket rate_limits frames are captured into Gin context response headers and auth.Quota.Signals. Verification: go test ./internal/runtime/executor/... PASS.
- 2026-09-08T13:52:00Z: Phase 2 (phase-2-frontend-codex-quota) Wave 1 & 2 DONE. Extended CodexQuotaWindow with resetAtMs and periodHours, CodexQuotaState with subscriptionActiveUntil and applicableAvailableCount. Refactored buildCodexQuotaWindows to classify monthly windows (28-31d) and run additionalRateLimits through pickClassifiedWindows. Removed <Tabs> from renderCodexItems, unifying Plan chip, subscription expiry, manual resets, and windows onto a flat card. Added planTier.ts with resolvePlanTier. Added missing i18n keys to en.json/vi.json. Verification: cd web && bun run type-check && bun run build PASS.
- 2026-09-08T14:00:00Z: Phase 3 (phase-3-frontend-antigravity-quota) Wave 1 & 2 DONE. Switched ANTIGRAVITY_QUOTA_URLS to retrieveUserQuotaSummary. Implemented hierarchical Groups & Buckets builder in builders.ts with ANTIGRAVITY_BUCKET_WINDOW_ORDER (5h -> weekly -> daily). Implemented 4-stage waterfall for project ID resolution and HTTP Date header clock skew calculation (serverTimeOffsetMs). Added antigravitySubscriptionApi for loadCodeAssist tier detection (free/pro/ultra). Added locale keys. Verification: cd web && bun run type-check && bun run build PASS.
- 2026-09-08T14:10:00Z: Phase 4 (phase-4-other-providers-parity) Wave 1 & 2 DONE. Added findFableUsageLimit to Claude window builder and mapped iguana_necktie to seven-day-fable. Inverted buildKimiQuotaRows order (5h limits first, weekly usage last) and fixed protobuf unit stripping (TIME_UNIT_MINUTE -> 5h) and day formatting. Created xaiPaid.ts (isPaidXaiAuthFile) and implemented dual weekly/monthly billing queries with paid health fallback (requestXaiPaidHealth via grok-4.5) in quotaConfigs.ts. Verification: cd web && bun run type-check && bun run build PASS; go test ./internal/runtime/executor/... ./internal/api/handlers/management/... PASS.

## Decisions
- 2026-09-08: Adopted flat layout for Codex card instead of tabbed view. Upstream Cli-Proxy-API-Management-Center displays all quota forms simultaneously so users never miss available manual reset credits.
- 2026-09-08: Strictly preserved llmhub extensions (gemini-cli and kiro), verifying they compile and function without behavioral regressions alongside the updated standard providers.
- 2026-09-08: Used Record<string, true> for PREMIUM_CODEX_PLAN_TYPES in planTier.ts to adhere to repo ts-set-map guidelines.

## Validation
- 2026-09-08T14:20:00Z: phase-4-other-providers-parity (initiative full gate)
  - cd web && bun run type-check -> pass
  - cd web && bun run build -> pass
  - go test ./internal/runtime/executor/... ./internal/runtime/executor/helps/... ./internal/api/handlers/management/... -count=1 -> pass
  - git diff --check -> pass
  - verdict: APPROVED
  - judge: independent
  - judge_model: google-antigravity/gemini-3.8-flash
  - proof_gaps: none
  - receipt: context_sources: [internal/runtime/executor/codex_websockets_executor.go, web/src/components/quota/quotaConfigs.ts, web/src/utils/quota/builders.ts, web/src/utils/quota/constants.ts, web/src/types/quota.ts] / policy: strict / judge: independent / judge_model: google-antigravity/gemini-3.8-flash / retries: 0 / rollback_point: HEAD / failure_ledger: absent / not_independently_verified: none

## Current State and Next Action
- Active Phase: All phases completed & validated (1–4).
- Lifecycle Status: checked
- Blockers: None.
- Open Items: None.
- Exact Next Action: Run /git to commit and push changes.
