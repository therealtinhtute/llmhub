---
id: 01KYKAKVXQ62ZRQC9N6HSR4TK0
type: plan
intake_id: 01KYKAM79F477BT30M0MW5ZA20
lane: high-risk
status: active
created: 2026-07-28
updated: 2026-07-29
---

# Plan: Database-backed quota alert monitoring

## Outcome
- result: LLMHub continuously evaluates quota for each eligible persisted auth, records current alert state in PostgreSQL, shows actionable in-app alerts, and sends deduplicated provider-grouped Telegram notifications when quota becomes low, exhausted, or recovers.
- success_signals:
  - Quota monitoring continues while the management UI is closed and produces normalized per-auth, per-resource, and per-window state for every supported provider.
  - A configurable threshold defaults to 10% remaining; crossing it, reaching exhaustion, and recovering produce the expected state transitions without duplicate alerts across polls or process restarts.
  - One notification batch per provider identifies affected auth labels and quota resources/windows without exposing credentials or raw tokens.
  - Global settings, provider overrides, Telegram destination configuration, alert state, and in-app events are persisted as structured PostgreSQL records and no quota-alert setting is added to config YAML.
  - Management APIs and the web panel can configure monitoring, configure Telegram through write-only secret handling, inspect current state, inspect recent events, acknowledge events, and trigger an explicit test notification.
  - Automated tests prove threshold evaluation, provider grouping, persistence, deduplication, recovery, multi-instance ownership, API authorization/secret redaction, and relevant provider normalization; Go tests and frontend type/lint/build checks pass.

## Authority and Requirements
- authority:
  - Owner instruction on 2026-07-28: warn when remaining quota is below a configurable limit such as 10% or is exhausted for individual auth files, grouped by provider, with configurable monitoring and Telegram notification.
  - Owner correction on 2026-07-28: all quota-monitoring and notification configuration must use database-backed structured records and must not use config YAML.
  - Current on-demand quota bridge and credential substitution: `internal/api/handlers/management/api_tools.go`.
  - Current provider parsing and quota-page state: `web/src/components/quota/quotaConfigs.ts`, `web/src/stores/useQuotaStore.ts`, and `web/src/pages/QuotaPage.tsx`.
  - Existing stable auth identity, runtime result hooks, token scheduler, and PostgreSQL store: `sdk/cliproxy/auth/types.go`, `sdk/cliproxy/auth/conductor.go`, `sdk/cliproxy/auth/auto_refresh_loop.go`, and `internal/store/postgresstore.go`.
- requirements:
  - R1 [accepted]: A server-side monitor must evaluate quota without requiring an open browser, using bounded concurrency and a configurable polling interval. | source: owner instruction and current browser-driven quota flow
  - R2 [accepted]: The monitor must represent quota per persisted auth ID, provider, quota resource, and quota window, and must retain a redacted human-readable auth label for presentation. | source: owner instruction and existing auth identity model
  - R3 [accepted]: The normalized state model must distinguish `healthy`, `warning`, `exhausted`, and `unknown`; the default warning threshold is 10% remaining, exhaustion requires zero remaining or an explicit provider exhaustion signal, and collection failures must not be treated as exhaustion. | source: owner instruction and provider-specific quota semantics
  - R4 [accepted]: Global monitoring settings, per-provider enablement and threshold overrides, Telegram destination settings, alert state, and in-app alert events must be structured PostgreSQL data; this initiative must not add or update quota-alert configuration through YAML. | source: owner database-only correction
  - R5 [accepted]: Alert delivery must be transition-driven and survive restart: send on entry to warning, escalation to exhausted, and configured recovery; suppress duplicate sends while state is unchanged and support an optional reminder interval that is disabled by default. | source: approved brainstorm recommendation
  - R6 [accepted]: Notifications must be grouped by provider while preserving each affected auth label, quota resource/window, remaining value when known, and reset time when known. | source: owner provider-grouping instruction
  - R7 [accepted]: MVP delivery channels are persisted in-app events and Telegram; Telegram configuration must support enabled state, destination chat ID, and a write-only bot token that is encrypted at rest and never returned by management read APIs. | source: owner notification instruction and database-only correction
  - R8 [accepted]: Unattended quota collection must use provider-specific backend collectors with fixed or allowlisted upstream hosts and must not drive the externally parameterized management `/api-call` endpoint. | source: existing `/api-call` security boundary
  - R9 [accepted]: Request-result signals may accelerate known quota transitions, but HTTP 429 alone must not be classified as exhausted unless provider-specific evidence or runtime quota state confirms exhaustion. | source: existing auth result hook and provider behavior
  - R10 [accepted]: In multi-instance deployments, database-backed ownership must ensure that only one active monitor dispatches a given polling cycle and notification transition. | source: PostgreSQL runtime architecture and deduplication requirement
  - R11 [accepted]: Database-backed management APIs and dedicated web controls must configure monitoring and Telegram directly, expose current/recent alert state, acknowledge in-app events, and perform an explicit Telegram test without using the YAML editor. | source: owner database-only correction
  - R12 [accepted]: Logs, API responses, UI state, persisted event payloads, and Telegram messages must not expose OAuth tokens, API keys, raw Telegram bot tokens, or unredacted credential material. | source: credential and outbound-notification security boundary

## Non-goals
- NG1: Add Slack, Discord, email, SMS, generic webhook, or browser-push delivery in the MVP.
- NG2: Add per-auth threshold overrides, a rule-expression engine, multi-level escalation policies, or advanced notification routing.
- NG3: Build a full analytics dashboard or indefinite alert-history warehouse; only current state and bounded recent in-app events are required.
- NG4: Change provider request routing, quota failover behavior, credential selection, or automatically disable auth records based on alert state.
- NG5: Migrate unrelated existing application configuration away from its current storage model.
- NG6: Reuse browser execution or the general management `/api-call` endpoint as the unattended monitoring engine.

## Approach and Risks
- approach:
  - Add a server-only `internal/quotaalert` domain that owns normalized observations, threshold evaluation, transition state, polling, provider grouping, in-app events, durable Telegram delivery, and lifecycle control. Keep the public auth store contract unchanged.
  - Persist all feature configuration and runtime state as structured PostgreSQL records through a dedicated quota-alert store contract. Add idempotent additive DDL to the existing `PostgresStore.EnsureSchema` path; seed one disabled singleton settings row with a 10% default threshold.
  - Store global settings and one Telegram destination in `quota_alert_settings`, provider enablement/threshold overrides in `quota_alert_provider_settings`, current normalized state in `quota_alert_state`, transition history in `quota_alert_events`, and immutable provider-grouped delivery work in `quota_notification_batches`.
  - Encrypt the Telegram bot token with versioned AES-256-GCM before persistence. Supply the 32-byte root key through `LLMHUB_QUOTA_SECRET_KEY_B64` and an optional `LLMHUB_QUOTA_SECRET_KEY_ID`; these are deployment secrets, not feature configuration. Management reads expose only `token_configured`.
  - Implement fixed-host backend collectors for Claude, Codex, Gemini CLI, Antigravity, Kimi, xAI, and Kiro. Reuse proxy/OAuth/executor behavior and existing Kiro/Antigravity backend seams; never invoke the caller-controlled management `/api-call` endpoint from unattended work.
  - Keep the existing browser-fetched quota cards unchanged in this initiative. The new monitor owns a separate canonical server-observed state; Go fixture tests must preserve the current provider grouping, percentage, reset, and missing-data semantics before later UI consolidation is considered.
  - Poll settings versions using the existing database-watcher cadence, acquire a PostgreSQL advisory lock for each collection cycle, collect with deadlines, jitter, and a bounded worker pool, and atomically persist state transitions, events, and deduplicated provider batches.
  - Dispatch Telegram batches through durable `FOR UPDATE SKIP LOCKED` claims with expiring leases and sanitized retries. In-app events are the persisted transition records exposed by management APIs.
  - Wire the monitor as an optional service dependency so SDK/file-store use remains compatible; normal PostgreSQL server startup supplies the store and cipher. Start after persisted auth loading and stop inside the existing idempotent shutdown path.
  - Add authorized `/v0/management/quota-alerts/*` APIs and a dedicated `/quota/monitoring` web page. The page reads and writes structured APIs directly and must not call, parse, or update config YAML.
- constraints:
  - PostgreSQL is the sole source of truth for quota-alert settings, provider overrides, Telegram configuration, state, events, and delivery work; no quota-alert field may be added to config YAML or the visual YAML editor.
  - Durable identity is persisted `Auth.ID`; filenames and derived `Auth.Index` are display/management aids only and cannot key database state or deduplication.
  - Default monitoring is disabled after schema installation; when enabled, the default poll interval is five minutes and the default warning threshold is 10% remaining.
  - Collection failures and bare HTTP 429 responses cannot produce exhaustion. A later collection failure retains the last reliable alert level while recording collection health; `unknown` is used when no reliable observation exists.
  - A notification is created only for warning entry, exhausted escalation, configured recovery, or an enabled reminder. Unchanged polls cannot create another logical event or batch.
  - Provider messages contain only redacted auth labels, resource/window, normalized remaining values, reset time, and transition type; raw provider bodies and credentials are excluded from persistence and logs.
  - Telegram supports one configured destination in MVP. Slack, email, Discord, generic webhooks, browser push, per-auth thresholds, and rule engines remain outside scope.
  - Frontend work must not add test files under `web/`; proof uses type checking, lint, production build, and browser runtime checks.
- dependencies:
  - Existing PostgreSQL `EnsureSchema`, transaction, `SKIP LOCKED`, and integration-test patterns in `internal/store/postgresstore.go` and `internal/store/postgresstore_integration_test.go`.
  - Existing auth enumeration, stable IDs, provider refresh/proxy behavior, service builder/start/stop seams, and storage version watcher under `sdk/cliproxy`.
  - Existing Kiro quota normalization and Antigravity credit collection under `internal/runtime/executor`.
  - Existing management authorization/routing and web management API client/form primitives.
- rejected_alternatives:
  - Browser-only evaluation: rejected because monitoring would stop when the quota page is closed and Telegram delivery could not be durable.
  - Request-hook or usage-plugin-only alerts: rejected as the primary mechanism because idle auths would never receive pre-exhaustion warnings; retained only as an accelerated wake signal when provider evidence is available.
  - Reusing management `/api-call`: rejected because it accepts arbitrary absolute URLs and can substitute credential tokens, which is unsafe for unattended polling.
  - YAML-backed settings: rejected by owner requirement; all feature settings use dedicated structured database records and APIs.
  - A generalized multi-channel notification bus: rejected for MVP scope; the store/outbox contract supports only persisted in-app events and one Telegram destination.
  - Replacing all existing browser quota-card fetches in the same initiative: rejected to avoid coupling alert reliability to a broad UI migration; server collectors receive parity fixtures and the current cards remain operational.
- risks:
  - risk: Provider payload semantics can drift or differ from the current TypeScript normalizers.
    mitigation: Capture provider fixtures and test resource/window keys, remaining percentages, reset selection, optional enrichment, and missing-data behavior for every supported provider before enabling its collector.
    recovery: Disable only the affected provider through its database override; retain prior reliable state and record collection health without emitting false exhaustion.
  - risk: Unattended provider calls could leak credentials or call attacker-controlled hosts.
    mitigation: Construct fixed or allowlisted endpoints inside provider collectors, reuse proxy-aware backend clients, cap one refresh retry on 401, sanitize errors, and prohibit raw bodies/tokens in events or logs.
    recovery: Cancel the provider collector and keep it disabled until its host and redaction tests pass.
  - risk: Polling can trigger provider rate limits or overload a deployment with many auths.
    mitigation: Default to five-minute polling, apply jitter, per-request deadlines, bounded concurrency, one advisory-lock owner per cycle, and provider-level failure isolation.
    recovery: Back off the affected provider without converting failures to quota exhaustion; continue healthy providers.
  - risk: Concurrent instances or restarts can create duplicate state transitions or Telegram work.
    mitigation: Use atomic state/event/batch transactions, unique transition and batch keys, advisory-lock polling, and leased `SKIP LOCKED` delivery claims.
    recovery: Reclaim expired delivery leases and suppress any batch whose event IDs already have a terminal delivery record.
  - risk: Telegram cannot guarantee mathematical exactly-once delivery if the process crashes after Telegram accepts a message but before PostgreSQL records success.
    mitigation: Document the narrow at-least-once crash window, use durable unique batches and short send timeouts, and eliminate duplicates across normal polls, retries, and restarts.
    recovery: Surface delivery state in the event UI so an operator can distinguish sent, failed, and lease-recovered attempts.
  - risk: A missing, changed, or invalid encryption root key can make the stored bot token unusable.
    mitigation: Validate the key at startup and on token writes, version ciphertext with key ID and authenticated purpose data, never return the token, and fail closed for Telegram only.
    recovery: Continue collection and in-app events with Telegram disabled until an operator restores the correct key or writes a replacement token.
  - risk: Schema or PostgreSQL availability failures could affect the proxy request path.
    mitigation: Keep quota monitoring optional and isolated from auth routing, install additive idempotent DDL, start workers only after store readiness, and return sanitized management errors.
    recovery: Do not start the monitor when its schema/store is unavailable; keep normal proxy service running and expose the monitor as unavailable.
  - risk: Management UI accidentally reads or writes YAML or retains the Telegram token in browser state.
    mitigation: Use a dedicated typed API service, a blank write-only token input cleared after save, `token_configured` metadata, and browser network/state inspection.
    recovery: Stop the UI phase until no quota-monitoring request touches config YAML and no GET response or React state contains the stored token.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: quota-alert-foundation
    story_id: 01KYKBFTE2YMWCCZVXQP1NKP2E
    status: checked
    goal: Define the quota-alert domain, encrypted database-only settings, durable state, events, and notification outbox persistence.
    depends_on: none
    allowed_surfaces:
      - `internal/quotaalert/types.go`
      - `internal/quotaalert/crypto.go`
      - `internal/quotaalert/*_test.go`
      - `internal/store/postgres_quota_alert.go`
      - `internal/store/postgres_quota_alert_integration_test.go`
      - `internal/store/postgresstore.go`
    avoided_surfaces:
      - `internal/config/config.go`
      - config YAML handlers and `web/src/hooks/useVisualConfig.ts`
      - provider executors and frontend files
      - the public `sdk/cliproxy/auth.Store` interface
    waves:
      - wave: 1
        goal: Lock the domain, validation, and secret boundary before persistence work.
        tasks:
          - task_id: F1
            task: Define domain contracts and validation.
            requirements: [R2, R3, R4, R5, R6, R7, R10, R12]
            depends_on: none
            touched_surfaces:
              - new `internal/quotaalert/types.go`
              - new `internal/quotaalert/types_test.go`
            avoided_surfaces:
              - provider HTTP code
              - PostgreSQL implementation
            expected_outputs:
              - Typed global settings, provider overrides, normalized observations, reliable/unknown collection health, alert states, transition events, immutable provider batches, pagination, and narrow store/collector/sender interfaces.
              - Validation for enabled state, poll interval bounds, 0–100 percentage thresholds, optional reminder interval, supported providers, and one Telegram destination.
              - Stable database identity based on persisted auth ID plus provider/resource/window; no filename or auth-index keys.
            verification:
              - `go test ./internal/quotaalert -run '^Test(Validate|Normalize|Identity)' -count=1`
              - `go test ./internal/quotaalert -count=1`
            stop_conditions:
              - Stop if a domain type requires importing web types, config YAML structs, or the public auth-store interface.
            escalation: Return to brainstorm refine only if a new product setting or channel is required.
          - task_id: F2
            task: Implement versioned write-only Telegram secret encryption.
            requirements: [R4, R7, R12]
            depends_on: none
            touched_surfaces:
              - new `internal/quotaalert/crypto.go`
              - new `internal/quotaalert/crypto_test.go`
            avoided_surfaces:
              - browser storage
              - application YAML
            expected_outputs:
              - AES-256-GCM cipher with a random nonce per write, key ID and purpose-bound authenticated data, explicit replace/preserve/clear semantics, and redacted read DTOs.
              - Tamper, wrong-key, wrong-purpose, invalid-key-length, and ciphertext-nondeterminism coverage.
            verification:
              - `go test ./internal/quotaalert -run '^Test(Cipher|Secret)' -count=1`
            stop_conditions:
              - Stop if plaintext or ciphertext can be serialized by a management read DTO.
            escalation: Keep Telegram disabled and proceed with in-app storage only until a safe root-key boundary exists.
      - wave: 2
        goal: Add idempotent PostgreSQL schema and transactional persistence.
        tasks:
          - task_id: F3
            task: Implement quota-alert PostgreSQL schema and store adapter.
            requirements: [R2, R4, R5, R6, R7, R10, R11, R12]
            depends_on: [F1, F2]
            touched_surfaces:
              - new `internal/store/postgres_quota_alert.go`
              - modified `internal/store/postgresstore.go`
            avoided_surfaces:
              - `sdk/cliproxy/auth/store.go`
              - existing usage-event queue semantics
            expected_outputs:
              - Additive idempotent DDL for `quota_alert_settings`, `quota_alert_provider_settings`, `quota_alert_state`, `quota_alert_events`, and `quota_notification_batches`.
              - A disabled singleton settings row with a five-minute interval and 10% default threshold, version increments on configuration changes, encrypted Telegram columns, provider overrides, and schema constraints.
              - Atomic state/revision/event/batch writes with unique transition and deterministic provider-batch keys.
              - Current-state/event listing, idempotent acknowledgement, bounded retention pruning, advisory-lock collection ownership, and expiring `FOR UPDATE SKIP LOCKED` delivery claims.
            verification:
              - `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1`
              - `LLMHUB_POSTGRES_TEST_DSN='postgres://user:password@127.0.0.1:5432/llmhub_test?sslmode=disable' go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1`
            stop_conditions:
              - Do not mark the task verified if the PostgreSQL integration suite only skipped; a real DSN is required for DDL, transaction, advisory-lock, and lease proof.
              - Stop if schema installation is destructive or requires a second migration framework.
            escalation: Record a blocker if the existing database bootstrap path cannot safely install additive tables.
      - wave: 3
        goal: Prove the database-only boundary and foundation concurrency behavior.
        tasks:
          - task_id: F4
            task: Complete foundation integration and race coverage.
            requirements: [R4, R5, R7, R10, R12]
            depends_on: [F3]
            touched_surfaces:
              - `internal/quotaalert/*_test.go`
              - `internal/store/postgres_quota_alert_integration_test.go`
            avoided_surfaces:
              - config YAML and frontend YAML editor files
            expected_outputs:
              - Tests for idempotent DDL, settings/provider round trips, encrypted-at-rest token storage, concurrent transition deduplication, event acknowledgement, retention, advisory-lock exclusivity, notification claim exclusivity, and expired-lease recovery.
              - No quota-alert configuration diff under YAML-owned surfaces.
            verification:
              - `go test -race ./internal/quotaalert -count=1`
              - `LLMHUB_POSTGRES_TEST_DSN='postgres://user:password@127.0.0.1:5432/llmhub_test?sslmode=disable' go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1`
              - `test -z "$(git diff --name-only -- internal/config/config.go web/src/hooks/useVisualConfig.ts)"`
            stop_conditions:
              - Stop on any race, plaintext token match in database assertions, duplicate event/batch under concurrency, or YAML-owned file change.
            escalation: Fix the foundation before creating any provider collector.
    phase_checks:
      - `go test ./internal/quotaalert ./internal/store -count=1`
      - `go test -race ./internal/quotaalert -count=1`
      - Required PostgreSQL integration command from F4 completes without skip or failure.
      - `git diff --check`

  - phase_slug: quota-provider-collectors
    story_id: 01KYKBG1WPYTBQW4P23Y2YFQEQ
    status: in-progress
    goal: Collect and normalize quota for every supported provider through fixed backend endpoints with provider-parity coverage.
    depends_on: quota-alert-foundation
    allowed_surfaces:
      - `internal/quotaalert/collectors.go`
      - `internal/quotaalert/collector_*.go`
      - `internal/quotaalert/collector_*_test.go`
      - `internal/quotaalert/testdata/**`
      - `internal/runtime/executor/kiro_quota.go`
      - `internal/runtime/executor/kiro_quota_test.go`
      - `internal/runtime/executor/antigravity_executor.go`
      - `internal/runtime/executor/antigravity_executor_credits_test.go`
      - narrowly required helpers under `internal/runtime/executor/helps/`
    avoided_surfaces:
      - management `/api-call` invocation
      - arbitrary credential-provided quota URLs
      - frontend parser removal or quota-card redesign
      - routing and credential-selection behavior
    waves:
      - wave: 1
        goal: Establish one fixed-host collector contract and deterministic normalization vocabulary.
        tasks:
          - task_id: C1
            task: Build the collector registry, shared HTTP boundary, and fixture contract.
            requirements: [R1, R2, R3, R8, R9, R12]
            depends_on: none
            touched_surfaces:
              - new `internal/quotaalert/collectors.go`
              - new `internal/quotaalert/collector_http.go`
              - new `internal/quotaalert/collectors_test.go`
              - new `internal/quotaalert/testdata/**`
            avoided_surfaces:
              - `internal/api/handlers/management/api_tools.go` as an execution dependency
            expected_outputs:
              - Registry keyed by supported provider, fixed/allowlisted hosts, proxy-aware clients, per-request deadlines, one provider refresh retry on 401, cloned auth inputs, sanitized errors, and deterministic resource/window keys.
              - Fixture assertions covering known observation, unknown collection health, explicit exhaustion evidence, reset time, optional enrichment, and no raw credential/body retention.
            verification:
              - `go test ./internal/quotaalert -run '^TestCollector(Registry|HTTP|Sanitization)' -count=1`
            stop_conditions:
              - Stop if any collector requires a caller-provided absolute URL or logs bearer/cookie/token data.
            escalation: Add a narrow backend helper only when existing executor/proxy behavior cannot be reused without duplication.
      - wave: 2
        goal: Port provider semantics in independent file groups.
        tasks:
          - task_id: C2
            task: Implement Claude and Codex collectors.
            requirements: [R1, R2, R3, R8, R9, R12]
            depends_on: [C1]
            touched_surfaces:
              - new `internal/quotaalert/collector_claude.go`
              - new `internal/quotaalert/collector_claude_test.go`
              - new `internal/quotaalert/collector_codex.go`
              - new `internal/quotaalert/collector_codex_test.go`
              - provider fixtures under `internal/quotaalert/testdata/`
            avoided_surfaces:
              - frontend quota parser behavior
            expected_outputs:
              - Claude recognized usage windows with `100 - utilization`, tolerant profile enrichment, bounded percentages, and reset timestamps.
              - Codex five-hour/weekly/code-review/additional limits, reset-credit enrichment, guarded explicit exhaustion, and deterministic resource keys.
            verification:
              - `go test ./internal/quotaalert -run '^Test(Claude|Codex)Collector' -count=1`
            stop_conditions:
              - Stop if usage failure is masked by optional enrichment or a bare denial/429 becomes exhausted.
            escalation: Disable the incomplete provider override rather than ship guessed semantics.
          - task_id: C3
            task: Implement Gemini CLI and Antigravity collectors.
            requirements: [R1, R2, R3, R8, R9, R12]
            depends_on: [C1]
            touched_surfaces:
              - new `internal/quotaalert/collector_gemini.go`
              - new `internal/quotaalert/collector_gemini_test.go`
              - new `internal/quotaalert/collector_antigravity.go`
              - new `internal/quotaalert/collector_antigravity_test.go`
              - modified `internal/runtime/executor/antigravity_executor.go`
              - modified `internal/runtime/executor/antigravity_executor_credits_test.go`
            avoided_surfaces:
              - process-local credit cache as durable state
            expected_outputs:
              - Gemini bucket normalization across snake/camel payloads, `_vertex` stripping, preferred/minimum resource selection, amount-only/reset-only unknown rules, and optional plan/credit enrichment.
              - Antigravity three-host fixed fallback, body/bodyText parsing, resource-family minimum selection, explicit host-error unknown state, and a reusable backend credit-fetch seam.
            verification:
              - `go test ./internal/quotaalert -run '^Test(Gemini|Antigravity)Collector' -count=1`
              - `go test ./internal/runtime/executor -run '^TestAntigravity.*Credits' -count=1`
            stop_conditions:
              - Stop if a failed host or missing remaining field silently normalizes to exhausted without the current reset/value evidence.
            escalation: Preserve the current provider as unknown until parity fixtures are decision-complete.
          - task_id: C4
            task: Implement Kimi and xAI collectors.
            requirements: [R1, R2, R3, R8, R9, R12]
            depends_on: [C1]
            touched_surfaces:
              - new `internal/quotaalert/collector_kimi.go`
              - new `internal/quotaalert/collector_kimi_test.go`
              - new `internal/quotaalert/collector_xai.go`
              - new `internal/quotaalert/collector_xai_test.go`
              - provider fixtures under `internal/quotaalert/testdata/`
            avoided_surfaces:
              - monetary/token values in logs
            expected_outputs:
              - Kimi summary and dynamic limit resources, direct/derived usage, absolute/relative reset handling, and zero-limit unknown rules.
              - xAI scalar/wrapped cent parsing, monthly-credit resource, remaining percentage, billing reset, pay-as-you-go metadata, and incomplete-period unknown handling.
            verification:
              - `go test ./internal/quotaalert -run '^Test(Kimi|XAI)Collector' -count=1`
            stop_conditions:
              - Stop if zero/absent limits are guessed into a percentage or currency amounts are logged unredacted.
            escalation: Keep the provider disabled when a reliable remaining measure is unavailable.
          - task_id: C5
            task: Adapt existing Kiro quota collection.
            requirements: [R1, R2, R3, R8, R9, R12]
            depends_on: [C1]
            touched_surfaces:
              - new `internal/quotaalert/collector_kiro.go`
              - new `internal/quotaalert/collector_kiro_test.go`
              - narrowly modified `internal/runtime/executor/kiro_quota.go`
              - modified `internal/runtime/executor/kiro_quota_test.go`
            avoided_surfaces:
              - Kiro routing/exhaustion behavior
              - Kiro raw upstream payload persistence
            expected_outputs:
              - Adapter over existing fixed-host `FetchQuota` behavior for subscription, free-trial, overage, request/token resources, reset times, and explicit exhausted state.
              - Monitor mode ignores arbitrary metadata quota URLs while preserving existing manual/test behavior outside the monitor.
            verification:
              - `go test ./internal/quotaalert -run '^TestKiroCollector' -count=1`
              - `go test ./internal/runtime/executor -run 'Test(ParseKiroUsageLimits|KiroQuotaStateExhausted|KiroExecutorFetchQuota)' -count=1`
            stop_conditions:
              - Stop if monitor adaptation changes existing Kiro selector/routing state or persists the upstream raw payload.
            escalation: Keep the existing Kiro behavior intact and isolate conversion in the quota-alert package.
      - wave: 3
        goal: Prove cross-provider normalization parity and failure isolation.
        tasks:
          - task_id: C6
            task: Complete provider fixture parity and registry integration.
            requirements: [R1, R2, R3, R8, R9, R12]
            depends_on: [C2, C3, C4, C5]
            touched_surfaces:
              - `internal/quotaalert/collector_*_test.go`
              - `internal/quotaalert/testdata/**`
            avoided_surfaces:
              - frontend test files
              - removal of current TypeScript quota normalization
            expected_outputs:
              - Table-driven fixtures for every supported resource/window, reset selection, partial payload, optional enrichment failure, timeout/cancellation, 401 refresh, 429 classification, malformed body, and redaction case.
              - One failing provider does not prevent observations from other providers and cannot alter their state.
            verification:
              - `go test ./internal/quotaalert -run '^Test.*Collector' -count=1`
              - `go test ./internal/runtime/executor -run 'Test(ParseKiroUsageLimits|KiroQuotaStateExhausted|KiroExecutorFetchQuota|Antigravity.*Credits)' -count=1`
              - `go test -race ./internal/quotaalert -run '^Test.*Collector' -count=1`
            stop_conditions:
              - Do not enable a provider whose fixture semantics conflict with the current parser and cannot be resolved from provider evidence.
            escalation: Record the provider as unsupported/disabled without blocking verified providers.
    phase_checks:
      - `go test ./internal/quotaalert ./internal/runtime/executor -count=1`
      - `go test -race ./internal/quotaalert -run '^Test.*Collector' -count=1`
      - No collector calls management `/api-call`, accepts arbitrary absolute URLs, or persists raw upstream bodies.
      - `git diff --check`

  - phase_slug: quota-alert-runtime
    story_id: 01KYKBG9G432XE1RBFB9NRDRZG
    status: planned
    goal: Run transition evaluation, single-owner polling, durable provider-grouped delivery, Telegram, and lifecycle integration.
    depends_on: quota-provider-collectors
    allowed_surfaces:
      - `internal/quotaalert/evaluator.go`
      - `internal/quotaalert/service.go`
      - `internal/quotaalert/telegram.go`
      - corresponding `internal/quotaalert/*_test.go`
      - `sdk/cliproxy/builder.go`
      - `sdk/cliproxy/service.go`
      - `sdk/cliproxy/auth/conductor.go` only for narrow hook composition
      - `cmd/server/db_runtime.go`
      - `cmd/server/db_runtime_test.go`
      - `cmd/server/main.go`
    avoided_surfaces:
      - request routing and selector policy
      - provider collector semantics already checked in the previous phase
      - management API and frontend
    waves:
      - wave: 1
        goal: Implement pure transition logic and isolated Telegram transport independently.
        tasks:
          - task_id: R1
            task: Implement deterministic quota state evaluation and provider batching.
            requirements: [R3, R5, R6, R9]
            depends_on: none
            touched_surfaces:
              - new `internal/quotaalert/evaluator.go`
              - new `internal/quotaalert/evaluator_test.go`
            avoided_surfaces:
              - database and network code
            expected_outputs:
              - Healthy/warning/exhausted/unknown evaluation, threshold crossing, prior-known-state retention on collection failure, warning-to-exhausted escalation, configurable recovery, disabled-by-default reminders, deterministic provider grouping, and stable dedupe inputs.
            verification:
              - `go test ./internal/quotaalert -run '^TestEvaluator' -count=1`
            stop_conditions:
              - Stop if a bare 429, timeout, malformed body, or missing percentage can become exhausted without explicit provider evidence.
            escalation: Refine only if a provider cannot map into the accepted four-state model.
          - task_id: R2
            task: Implement sanitized Telegram transport and explicit test-send behavior.
            requirements: [R6, R7, R12]
            depends_on: none
            touched_surfaces:
              - new `internal/quotaalert/telegram.go`
              - new `internal/quotaalert/telegram_test.go`
            avoided_surfaces:
              - generic notification channels
              - logging Telegram URLs or tokens
            expected_outputs:
              - Fixed Telegram API host, proxy-aware client, context deadlines, provider-grouped message rendering, redacted labels, bounded message size, sanitized retryable/permanent errors, and test send that creates no alert transition.
            verification:
              - `go test ./internal/quotaalert -run '^TestTelegram' -count=1`
            stop_conditions:
              - Stop if any error, log, or test output exposes the bot token or request URL containing it.
            escalation: Disable Telegram and preserve in-app events if transport safety cannot be proven.
      - wave: 2
        goal: Run single-owner polling and durable transition/outbox processing.
        tasks:
          - task_id: R3
            task: Implement the quota monitor service.
            requirements: [R1, R2, R3, R5, R6, R8, R9, R10, R12]
            depends_on: [R1, R2]
            touched_surfaces:
              - new `internal/quotaalert/service.go`
              - new `internal/quotaalert/service_test.go`
            avoided_surfaces:
              - unbounded goroutines
              - direct mutation of routing quota state
            expected_outputs:
              - Two-second settings-version watcher, resettable poll schedule, five-minute default with jitter, advisory-lock cycle ownership, persisted-auth enumeration, bounded collector pool, per-provider isolation, atomic transition/event/batch writes, immediate wake channel for qualified runtime evidence, leased outbox workers, retry/backoff, retention pruning, and cancellation-safe shutdown.
              - Database unavailability pauses monitoring without affecting normal proxy requests; provider failure retains reliable state and records collection health.
            verification:
              - `go test ./internal/quotaalert -run '^TestService' -count=1`
              - `go test -race ./internal/quotaalert -run '^Test(Service|Evaluator|Telegram)' -count=1`
            stop_conditions:
              - Stop on worker leaks, overlapping local cycles, duplicate transition writes, or request-path failure caused by monitor unavailability.
            escalation: Reduce concurrency or disable only the failing provider; do not weaken deduplication or request isolation.
      - wave: 3
        goal: Wire optional runtime construction, database secrets, hooks, startup, and shutdown.
        tasks:
          - task_id: R4
            task: Integrate quota monitoring into builder and service lifecycle.
            requirements: [R1, R4, R7, R9, R10, R12]
            depends_on: [R3]
            touched_surfaces:
              - modified `sdk/cliproxy/builder.go`
              - modified `sdk/cliproxy/service.go`
              - narrowly modified `sdk/cliproxy/auth/conductor.go` if hook composition is required
              - new `sdk/cliproxy/service_quota_alert_lifecycle_test.go`
              - modified `cmd/server/db_runtime.go`
              - modified `cmd/server/db_runtime_test.go`
              - modified `cmd/server/main.go`
            avoided_surfaces:
              - mandatory quota dependencies for SDK/file-store embedders
              - changes to selector or failover semantics
            expected_outputs:
              - Optional builder store/cipher options, quota service constructed with the resolved auth manager, startup after persisted auth loading/watcher/core refresh setup, stop inside the existing idempotent shutdown block, and composed result-hook wake signals that preserve existing hooks.
              - Strict base64 decoding of `LLMHUB_QUOTA_SECRET_KEY_B64` to 32 bytes plus optional key ID; absent key permits in-app monitoring but rejects token writes and disables Telegram.
              - Foreground and embedded/TUI server paths supply the shared PostgreSQL store and cipher without adding quota settings to YAML.
            verification:
              - `go test ./sdk/cliproxy -run '^TestServiceQuotaAlert' -count=1`
              - `go test ./cmd/server -run '^TestQuotaSecretKey' -count=1`
              - `go test ./sdk/cliproxy ./cmd/server -count=1`
            stop_conditions:
              - Stop if normal SDK construction now requires PostgreSQL, if shutdown can deadlock, or if an existing auth hook is replaced instead of composed.
            escalation: Keep the service optional/no-op outside normal PostgreSQL server startup.
      - wave: 4
        goal: Prove multi-instance ownership, durable delivery, and recovery.
        tasks:
          - task_id: R5
            task: Complete runtime integration and recovery tests.
            requirements: [R1, R5, R6, R7, R10, R12]
            depends_on: [R4]
            touched_surfaces:
              - `internal/quotaalert/service_test.go`
              - `internal/store/postgres_quota_alert_integration_test.go`
              - `sdk/cliproxy/service_quota_alert_lifecycle_test.go`
            avoided_surfaces:
              - production provider calls
            expected_outputs:
              - Two services sharing PostgreSQL produce one polling cycle and one logical provider batch; multiple outbox workers claim distinct batches; expired leases recover; settings version changes reset schedules; restart preserves dedupe; wrong encryption key disables Telegram only; shutdown releases locks and workers.
            verification:
              - `LLMHUB_POSTGRES_TEST_DSN='postgres://user:password@127.0.0.1:5432/llmhub_test?sslmode=disable' go test ./internal/store ./internal/quotaalert -run 'Test.*(AdvisoryLock|NotificationClaim|TransitionDedup|MultiInstance|LeaseRecovery)' -count=1`
              - `go test -race ./internal/quotaalert ./sdk/cliproxy -count=1`
            stop_conditions:
              - Do not mark the task verified if PostgreSQL tests skip, two instances send the same logical batch, or any monitor failure affects request serving.
            escalation: Block management/UI work until lifecycle and delivery ownership are deterministic.
    phase_checks:
      - `go test ./internal/quotaalert ./sdk/cliproxy ./cmd/server -count=1`
      - `go test -race ./internal/quotaalert ./sdk/cliproxy -count=1`
      - Required PostgreSQL multi-instance and outbox integration command from R5 completes without skip or failure.
      - `git diff --check`

  - phase_slug: quota-alert-management
    story_id: 01KYKBGFGPVBC58TAW8RBKEPCV
    status: planned
    goal: Expose authorized management APIs and a dedicated web UI for database-backed settings, quota states, events, and Telegram.
    depends_on: quota-alert-runtime
    allowed_surfaces:
      - `internal/api/handlers/management/quota_alerts.go`
      - `internal/api/handlers/management/quota_alerts_test.go`
      - `internal/api/handlers/management/handler.go`
      - `internal/api/server.go`
      - `internal/api/server_test.go`
      - `web/src/types/quotaAlert.ts`
      - `web/src/services/api/quotaAlerts.ts`
      - `web/src/pages/QuotaMonitoringPage.tsx`
      - `web/src/pages/QuotaPage.tsx`
      - `web/src/router/MainRoutes.tsx`
      - `web/src/i18n/locales/en.json`
      - `web/src/i18n/locales/vi.json`
    avoided_surfaces:
      - config YAML APIs and visual YAML editor
      - existing quota-card parser/store migration
      - new frontend test files
      - global notification-center or sidebar redesign
    waves:
      - wave: 1
        goal: Expose a redacted authorized API contract before building the page.
        tasks:
          - task_id: M1
            task: Add quota-alert management handlers and routes.
            requirements: [R4, R7, R11, R12]
            depends_on: none
            touched_surfaces:
              - new `internal/api/handlers/management/quota_alerts.go`
              - new `internal/api/handlers/management/quota_alerts_test.go`
              - modified `internal/api/handlers/management/handler.go`
              - modified `internal/api/server.go`
              - modified `internal/api/server_test.go`
            avoided_surfaces:
              - `ManagementConfigStore`
              - config YAML endpoints
            expected_outputs:
              - Authorized routes: `GET|PUT /v0/management/quota-alerts/settings`, `PUT /v0/management/quota-alerts/telegram`, `GET /v0/management/quota-alerts/state`, `GET /v0/management/quota-alerts/events`, `POST /v0/management/quota-alerts/events/:id/ack`, and `POST /v0/management/quota-alerts/telegram/test`.
              - Settings reads return global values, provider overrides, Telegram enabled/chat metadata, and `token_configured` only; omitted token preserves, non-empty token replaces, and explicit clear removes it.
              - Validation, pagination, idempotent acknowledgement, sanitized unavailable/provider/Telegram errors, and explicit test send with no persisted transition.
            verification:
              - `go test ./internal/api/handlers/management ./internal/api -run '^Test.*QuotaAlert' -count=1`
              - `go test ./internal/api/handlers/management ./internal/api -count=1`
            stop_conditions:
              - Stop if authorization can be bypassed, GET returns plaintext/ciphertext, PUT touches YAML storage, or a test send creates an alert event.
            escalation: Keep routes unavailable until the narrow quota-management dependency is injected safely.
      - wave: 2
        goal: Build a typed database-backed web client and monitoring page.
        tasks:
          - task_id: M2
            task: Add frontend quota-alert types and API service.
            requirements: [R4, R7, R11, R12]
            depends_on: [M1]
            touched_surfaces:
              - new `web/src/types/quotaAlert.ts`
              - new `web/src/services/api/quotaAlerts.ts`
            avoided_surfaces:
              - `configFileApi`
              - readable token response properties
            expected_outputs:
              - Typed settings/provider overrides, Telegram write/read shapes, current states, recent events, pagination, acknowledgement, and test-send calls through the existing management API client.
              - Response types contain only `tokenConfigured`; the replacement token exists only in the PUT request shape.
            verification:
              - `cd web && bun run type-check`
            stop_conditions:
              - Stop if the service imports YAML/config-file APIs or defines a readable stored-token property.
            escalation: Reconcile the backend DTO rather than weakening secret redaction.
          - task_id: M3
            task: Implement the dedicated quota monitoring page.
            requirements: [R2, R3, R4, R5, R6, R7, R11, R12]
            depends_on: [M2]
            touched_surfaces:
              - new `web/src/pages/QuotaMonitoringPage.tsx`
            avoided_surfaces:
              - existing browser quota cache/store
              - browser persistence of the bot token
            expected_outputs:
              - Monitor health summary; enablement, poll interval, default threshold, recovery, reminder, and provider-override controls; Telegram enabled/chat/write-only token/test controls; provider-grouped current-state rows; bounded recent events with acknowledgement and delivery status.
              - Uses existing secondary-screen shell, form controls, unsaved-changes guard, `Promise.allSettled`/stale-request protection, Sonner feedback, explicit loading/error/404-upgrade states, and accessible text labels in addition to color.
              - Token input is blank on load, cleared immediately after successful save, and never reconstructed from API data.
            verification:
              - `cd web && bun run type-check`
              - `cd web && bun run lint`
            stop_conditions:
              - Stop if page load or save makes config YAML requests, duplicate filenames merge server states, or token material survives a successful save in component/store state.
            escalation: Keep existing `/quota` usable and hide only the unsupported monitoring route on a 404 server response.
      - wave: 3
        goal: Register navigation, copy, and the bounded entry point without redesigning existing quota cards.
        tasks:
          - task_id: M4
            task: Wire the monitoring route, quota-page entry, and translations.
            requirements: [R7, R11, R12]
            depends_on: [M3]
            touched_surfaces:
              - modified `web/src/router/MainRoutes.tsx`
              - modified `web/src/pages/QuotaPage.tsx`
              - modified `web/src/i18n/locales/en.json`
              - modified `web/src/i18n/locales/vi.json`
            avoided_surfaces:
              - sidebar structure
              - `web/src/stores/useQuotaStore.ts`
              - `web/src/components/quota/quotaConfigs.ts`
              - new frontend test files
            expected_outputs:
              - `/quota/monitoring` route under the existing main layout, one clear Monitoring link/action from `/quota`, matching English/Vietnamese copy, and unchanged provider-card refresh behavior.
            verification:
              - `cd web && bun run type-check`
              - `cd web && bun run lint`
              - `cd web && bun run build`
            stop_conditions:
              - Stop if the existing `/quota` route, provider refresh, or sidebar active state regresses.
            escalation: Revert the entry-point wiring while retaining the directly addressable monitoring page.
      - wave: 4
        goal: Prove end-to-end management behavior, production builds, and browser safety.
        tasks:
          - task_id: M5
            task: Run backend, frontend, and browser acceptance checks.
            requirements: [R1, R2, R3, R4, R5, R6, R7, R10, R11, R12]
            depends_on: [M1, M4]
            touched_surfaces:
              - verification evidence only; fix only defects within this phase's allowed surfaces
            avoided_surfaces:
              - frontend test files
              - unrelated UI cleanup
            expected_outputs:
              - Database settings survive hard reload without config-YAML traffic; provider overrides affect later polls; warning, exhausted, recovery, and unchanged polls produce the expected deduplicated sequence; event acknowledgement persists.
              - `/quota` still refreshes existing cards; `/quota/monitoring` loads directly and keeps Quota navigation active; statuses use text plus color; 429/collection errors render unknown; token responses/state never contain the bot token; Telegram test is explicit and sanitized.
              - Production frontend build and full Go suite pass.
            verification:
              - `go test ./internal/api/handlers/management ./internal/api -count=1`
              - `go test ./... -count=1`
              - `cd web && bun run type-check`
              - `cd web && bun run lint`
              - `cd web && bun run build`
              - Start the PostgreSQL-backed server and `make dev-web`, then inspect browser UI, network requests, hard reload, transition sequence, token state, and existing quota-card refresh against the acceptance list above.
              - `git diff --check`
            stop_conditions:
              - Do not complete the phase with failing Go tests, frontend errors, config-YAML monitoring traffic, duplicate unchanged events, exposed token material, or an unverified browser flow.
            escalation: Route code defects back to the owning phase; route only scope changes to brainstorm refine.
    phase_checks:
      - `go test ./internal/api/handlers/management ./internal/api -count=1`
      - `go test ./... -count=1`
      - `cd web && bun run type-check && bun run lint && bun run build`
      - Browser acceptance checks from M5 are recorded with actual observed outcomes.
      - `git diff --check`

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- timestamp: 2026-07-28T05:27:33Z | phase: quota-alert-foundation | wave: 1 | task: phase-start | task_status: in-progress | run_id: 01KYKK2CDAB7SR569Y3ZXEES95 | trace_id: none | changed_surfaces: none | verification: run created; implementation pending
- timestamp: 2026-07-28T05:39:40Z | phase: quota-alert-foundation | wave: 1 | task: F1 | task_status: DONE | run_id: 01KYKK2CDAB7SR569Y3ZXEES95 | trace_id: 01KYKM47P0AEZ37S24GMR91ZPY | changed_surfaces: internal/quotaalert/types.go, internal/quotaalert/types_test.go | verification: `go test ./internal/quotaalert -run '^Test(Validate|Normalize|Identity)' -count=1` PASS; `go test ./internal/quotaalert -count=1` PASS
- timestamp: 2026-07-28T05:45:34Z | phase: quota-alert-foundation | wave: 1 | task: F2 | task_status: DONE | run_id: 01KYKK2CDAB7SR569Y3ZXEES95 | trace_id: 01KYKM47P0AEZ37S24GMR91ZPY | changed_surfaces: internal/quotaalert/crypto.go, internal/quotaalert/crypto_test.go | verification: `go test ./internal/quotaalert -run '^Test(Cipher|Secret)' -count=1` PASS; `go test ./internal/quotaalert -count=1` PASS
- timestamp: 2026-07-28T06:01:17Z | phase: quota-alert-foundation | wave: 2 | task: F3 | task_status: DONE | run_id: 01KYKK2CDAB7SR569Y3ZXEES95 | trace_id: 01KYKN1WJ1N0DBEGDJCD14ME68 | changed_surfaces: internal/quotaalert/types.go, internal/store/postgres_quota_alert.go, internal/store/postgres_quota_alert_integration_test.go, internal/store/postgresstore.go | verification: `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS; real PostgreSQL 16 DSN command PASS; `go test ./internal/quotaalert ./internal/store -count=1` PASS
- timestamp: 2026-07-28T06:07:13Z | phase: quota-alert-foundation | wave: 3 | task: F4 | task_status: DONE | run_id: 01KYKK2CDAB7SR569Y3ZXEES95 | trace_id: 01KYKNBVYYK3RMASMFECJ12X8G | changed_surfaces: internal/store/postgres_quota_alert_integration_test.go | verification: `go test -race ./internal/quotaalert -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS; YAML-owned surface diff check PASS
- timestamp: 2026-07-28T06:20:44Z | phase: quota-alert-foundation | wave: remediation | task: check-01KYKP0ZK8APNVVT0H4ZYGMWYP | task_status: in-progress | run_id: 01KYKP3RWHS0XMCQSS82ZQYBRA | trace_id: none | changed_surfaces: none | verification: remediation run created; four blocker groups pending
- timestamp: 2026-07-28T06:44:54Z | phase: quota-alert-foundation | wave: remediation | task: check-01KYKP0ZK8APNVVT0H4ZYGMWYP | task_status: DONE | run_id: 01KYKP3RWHS0XMCQSS82ZQYBRA | trace_id: 01KYKQFXKJ4W23WFRC8EWV351T | changed_surfaces: internal/quotaalert/types.go, internal/quotaalert/types_test.go, internal/store/postgres_quota_alert.go, internal/store/postgres_quota_alert_integration_test.go | verification: `go test ./internal/quotaalert ./internal/store -count=1` PASS; `go test -race ./internal/quotaalert -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS without skips; `go test ./... -count=1` PASS; YAML-owned surface check PASS; gofmt check PASS; `git diff --check` PASS
- timestamp: 2026-07-28T08:13:40Z | phase: quota-alert-foundation | wave: remediation | task: check-01KYKRAE46M01QSD2WG35N2TRJ | task_status: DONE | run_id: 01KYKP3RWHS0XMCQSS82ZQYBRA | trace_id: 01KYKWJ4X1NP4P9QBWAS4H4AKJ | changed_surfaces: internal/quotaalert/types.go, internal/quotaalert/types_test.go, internal/quotaalert/crypto.go, internal/quotaalert/crypto_test.go, internal/store/postgres_quota_alert.go, internal/store/postgres_quota_alert_integration_test.go | verification: canonical event/state/batch persistence, atomic settings-secret reads, explicit state reconciliation, immutable event replay, poison quarantine, bounded leases/inputs, PostgreSQL timestamp precision, and exclusive notification outcomes implemented; `go test ./internal/quotaalert ./internal/store -count=1` PASS; `go test -race ./internal/quotaalert -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS without skips; `go test ./... -count=1` PASS
- timestamp: 2026-07-28T09:45:21Z | phase: quota-alert-foundation | wave: remediation-4 | task: architecture-performance-follow-up | task_status: DONE | run_id: 01KYKP3RWHS0XMCQSS82ZQYBRA | trace_id: 01KYM1T7H87E9FRF6GPTR7BHPX | changed_surfaces: internal/quotaalert/types.go, internal/quotaalert/types_test.go, internal/store/postgres_quota_alert.go, internal/store/postgres_quota_alert_integration_test.go | verification: persisted transition provenance, exact replay, cross-store lease fencing, acknowledgement chronology, two-sided retention tombstones, seekable commit-refreshed notification leases, 1,000-item collection commits, and 16,384-identity state loads implemented; `go test ./internal/quotaalert ./internal/store -count=1` PASS; `go test -race ./internal/quotaalert -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS without skips; `go test ./... -count=1` PASS; gofmt, YAML-owned surface, `git diff --check`, changed-file whitespace, and secret scans PASS
- timestamp: 2026-07-29T00:00:00Z | phase: quota-alert-foundation | wave: remediation-5 | task: final-gate | task_status: DONE | run_id: 01KYKP3RWHS0XMCQSS82ZQYBRA | trace_id: 01KYNQZP2TZFZHX2T9VSZ6NXJG | changed_surfaces: internal/store/postgres_quota_alert.go, internal/store/postgres_quota_alert_integration_test.go, docs/plans/active/quota-alert-monitoring.md | verification: invalid initial transition history, batch-first retired replay, event-first pruned ID reuse, serialized opposite-order pruning, primary-key lease refresh, descending tuple state pagination, and corrected JSON constraint regression implemented; `go test ./internal/quotaalert ./internal/store -count=1` PASS; `go test -race ./internal/quotaalert -count=1` PASS; `go test ./... -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS without skips after restarting local PostgreSQL container; gofmt, YAML-owned surface, `git diff --check`, changed-file whitespace, secret scan, focused SQL review, and `zharness audit --json` PASS; check_id 01KYNR02XCV0EQBPJ44VZC074E APPROVED
- timestamp: 2026-07-29T02:03:37Z | phase: quota-provider-collectors | wave: 1 | task: phase-start | task_status: in-progress | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | trace_id: none | changed_surfaces: docs/plans/active/quota-alert-monitoring.md | verification: run created; C1 registry/shared HTTP boundary pending
- timestamp: 2026-07-29T02:07:54Z | phase: quota-provider-collectors | wave: 1 | task: C1 | task_status: DONE | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | trace_id: 01KYNT1QFG7WGRFZQAVQAWQ0B5 | changed_surfaces: internal/quotaalert/collectors.go, internal/quotaalert/collector_http.go, internal/quotaalert/collectors_test.go, internal/quotaalert/testdata/collector_known.json | verification: `go test ./internal/quotaalert -run '^TestCollector(Registry|HTTP|Sanitization)' -count=1` PASS
- timestamp: 2026-07-29T02:18:00Z | phase: quota-provider-collectors | wave: 2 | task: C2 | task_status: DONE | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | trace_id: 01KYNTKFVJS31JMMHXWZM87RS2 | changed_surfaces: internal/quotaalert/collectors.go, internal/quotaalert/collector_claude.go, internal/quotaalert/collector_claude_test.go, internal/quotaalert/collector_codex.go, internal/quotaalert/collector_codex_test.go, internal/quotaalert/collector_provider_helpers.go, internal/quotaalert/testdata/collector_claude_usage.json, internal/quotaalert/testdata/collector_codex_usage.json | verification: `go test ./internal/quotaalert -run '^Test(Claude|Codex)Collector' -count=1` PASS
- timestamp: 2026-07-29T02:28:00Z | phase: quota-provider-collectors | wave: 3 | task: C3 | task_status: DONE | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | trace_id: 01KYNTZX0Z2XP4GARDSGK7G1NY | changed_surfaces: internal/quotaalert/collector_http.go, internal/quotaalert/collector_gemini.go, internal/quotaalert/collector_gemini_test.go, internal/quotaalert/collector_antigravity.go, internal/quotaalert/collector_antigravity_test.go, internal/quotaalert/collector_provider_helpers.go, internal/quotaalert/testdata/collector_gemini_usage.json, internal/quotaalert/testdata/collector_antigravity_models.json | verification: `go test ./internal/quotaalert -run '^Test(Gemini|Antigravity)Collector' -count=1` PASS; `go test ./internal/runtime/executor -run '^TestAntigravity.*Credits' -count=1` PASS
- timestamp: 2026-07-29T02:36:00Z | phase: quota-provider-collectors | wave: 4 | task: C4 | task_status: DONE | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | trace_id: 01KYNV8E6STPCQS0Q7CC56ZQ5R | changed_surfaces: internal/quotaalert/collector_kimi.go, internal/quotaalert/collector_xai.go, internal/quotaalert/collector_kimi_xai_test.go, internal/quotaalert/testdata/collector_kimi_usage.json, internal/quotaalert/testdata/collector_xai_billing.json | verification: `go test ./internal/quotaalert -run '^Test(Kimi|XAI)Collector' -count=1` PASS
- timestamp: 2026-07-29T02:43:17Z | phase: quota-provider-collectors | wave: 5 | task: C5 | task_status: DONE | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | trace_id: 01KYNVYAVHNXWZ778T1WRTD92S | changed_surfaces: internal/quotaalert/collector_kiro.go, internal/quotaalert/collector_kiro_test.go, internal/quotaalert/testdata/collector_kiro_usage.json | verification: `go test ./internal/quotaalert -run '^TestKiroCollector' -count=1` PASS; `go test ./internal/runtime/executor -run 'Test(ParseKiroUsageLimits|KiroQuotaStateExhausted|KiroExecutorFetchQuota)' -count=1` PASS
- timestamp: 2026-07-29T02:43:17Z | phase: quota-provider-collectors | wave: 6 | task: C6 | task_status: DONE | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | trace_id: 01KYNW1A9BTD70BS6GJ0TRT6AR | changed_surfaces: internal/quotaalert/collectors.go, internal/quotaalert/collectors_test.go, internal/quotaalert/collector_kiro.go, internal/quotaalert/collector_kiro_test.go, internal/quotaalert/testdata/collector_kiro_usage.json | verification: `go test ./internal/quotaalert -run '^Test.*Collector' -count=1` PASS; `go test ./internal/runtime/executor -run 'Test(ParseKiroUsageLimits|KiroQuotaStateExhausted|KiroExecutorFetchQuota|Antigravity.*Credits)' -count=1` PASS; `go test -race ./internal/quotaalert -run '^Test.*Collector' -count=1` PASS

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- timestamp: 2026-07-28T05:39:40Z | phase/task: quota-alert-foundation/F1 | decision: Bound polling intervals inclusively from 1 minute to 24 hours while retaining the planned 5-minute default. | rationale: The accepted plan requires bounded polling but does not prescribe limits; these bounds prevent tight-loop provider pressure and impractically stale monitoring without adding configuration surface.
- timestamp: 2026-07-28T06:01:17Z | phase/task: quota-alert-foundation/F3 | decision: Add `PruneEvents(ctx, before, limit)` to the narrow store contract with a 1–100 deletion bound and deterministic oldest-first deletion. | rationale: F3 requires bounded retention pruning; making the operation explicit keeps deletion controlled, testable, and database-owned without introducing a scheduler or policy surface.
- timestamp: 2026-07-28T06:44:54Z | phase/task: quota-alert-foundation/remediation | decision: Read the singleton settings row and provider overrides in one read-only `REPEATABLE READ` transaction. | rationale: One stable PostgreSQL snapshot prevents management and runtime readers from combining fields from different settings revisions.
- timestamp: 2026-07-28T06:44:54Z | phase/task: quota-alert-foundation/remediation | decision: Bind collection commits to the active advisory-lock-owning PostgreSQL session and invalidate the physical connection when cleanup cannot prove unlock. | rationale: Session ownership fences stale collectors and prevents canceled cleanup from returning a potentially locked connection to the pool.
- timestamp: 2026-07-28T06:44:54Z | phase/task: quota-alert-foundation/remediation | decision: Use PostgreSQL `clock_timestamp()` for notification claims and lease expiry. | rationale: Database-owned time removes cross-replica application-clock skew from delivery ownership decisions.
- timestamp: 2026-07-28T06:44:54Z | phase/task: quota-alert-foundation/remediation | decision: Reject stale state writes and bound identity loading, cursors, batch cardinality, payload size, and terminal notification retention while authenticating all canonical delivery content in batch IDs. | rationale: Explicit limits and integrity checks preserve state freshness, avoid database/protocol exhaustion boundaries, and keep the durable outbox verifiable and prunable.
- timestamp: 2026-07-28T08:13:40Z | phase/task: quota-alert-foundation/remediation | decision: Replace separate runtime settings and secret reads with atomic `LoadSettingsWithSecret` snapshots while retaining redacted `LoadSettings` for management-facing callers. | rationale: A runtime cycle must never pair one settings revision or Telegram destination with encrypted token material from another revision.
- timestamp: 2026-07-28T08:13:40Z | phase/task: quota-alert-foundation/remediation | decision: Make `CurrentState.Normalize` the persistence invariant boundary and expose `CollectionCommit.RemovedStates` for explicit atomic reconciliation. | rationale: Reliable non-exhausted states require quota evidence, stale or contradictory states cannot persist, and the later runtime phase—not persistence—must decide when disabled/deleted auths or disappeared resources are obsolete without mistaking collection failure for deletion.
- timestamp: 2026-07-28T08:13:40Z | phase/task: quota-alert-foundation/remediation | decision: Require every notification-batch event to exactly match one canonical event in the same collection commit, reject same-ID event content changes, and exclude mutable acknowledgement state from immutable replay equality. | rationale: Event rows and delivery payloads remain atomically consistent and idempotent while a later acknowledgement cannot break a safe replay or be overwritten by it.
- timestamp: 2026-07-28T08:13:40Z | phase/task: quota-alert-foundation/remediation | decision: Canonicalize durable timestamps to PostgreSQL microsecond precision and validate event remaining evidence against the transition target. | rationale: Event-table rows, state rows, and immutable JSON payloads must share one canonical representation and cannot describe positive remaining quota as exhausted or zero remaining quota as non-exhausted.
- timestamp: 2026-07-28T08:13:40Z | phase/task: quota-alert-foundation/remediation | decision: Quarantine malformed outbox payloads as terminal `failed/invalid_payload`, bound claims from 1 second to 24 hours, and require exactly one sent/retry/permanent resolution intent. | rationale: Poison rows cannot starve valid work, invalid durations cannot create duplicate delivery or indefinite denial, and contradictory caller outcomes cannot silently select the wrong terminal state.
- timestamp: 2026-07-28T08:13:40Z | phase/task: quota-alert-foundation/remediation | decision: Bound Telegram chat IDs, secret key IDs, plaintext secret values, and encrypted payloads; keep quota schema readiness and obsolete-state selection owned by the later runtime phase. | rationale: Management input remains resource-bounded, while optional monitor readiness and deletion policy stay isolated from normal proxy startup and provider failure semantics.
- timestamp: 2026-07-28T09:45:21Z | phase/task: quota-alert-foundation/remediation-4 | decision: Validate each new transition against the locked previously persisted state, allow only exact immutable replay, bind collection leases to their creating store, enforce `transitioned <= observed <= updated`, and reject acknowledgements before occurrence. | rationale: A same-commit state alone cannot prove transition provenance, a lease from another store is not ownership, and impossible chronology would corrupt durable alert history.
- timestamp: 2026-07-28T09:45:21Z | phase/task: quota-alert-foundation/remediation-4 | decision: Persist event-to-batch assignment tombstones without parent foreign keys and delete a tombstone only in the bounded prune that removes the second surviving parent. | rationale: Batch-first and event-first retention must both preserve event delivery dedupe, while cleanup must touch only assignments associated with the bounded deleted set.
- timestamp: 2026-07-28T09:45:21Z | phase/task: quota-alert-foundation/remediation-4 | decision: Maintain one indexed `claimable_at` timestamp for pending notification eligibility and refresh every valid lease from PostgreSQL time immediately before claim commit. | rationale: A seekable schedule avoids nullable-expiry OR scans, and validation time cannot consume the lease returned to a delivery worker.
- timestamp: 2026-07-28T09:45:21Z | phase/task: quota-alert-foundation/remediation-4 | decision: Cap atomic collection commits at 1,000 top-level items and `LoadStates` requests at 16,384 identities while retaining 1,000-identity SQL chunks. | rationale: Explicit total bounds make per-row transactional writes and state-load allocation/query work finite without an unplanned bulk-write abstraction.
- timestamp: 2026-07-29T00:00:00Z | phase/task: quota-alert-foundation/remediation-5 | decision: Serialize event and batch retention through one transaction-scoped advisory lock, use a new all-descending state-list index/keyset predicate, and refresh notification leases by bounded claimed batch IDs plus lease ownership. | rationale: Opposite-order pruning cannot leak tombstones under MVCC, equal-timestamp state pages remain seekable, and lease refresh avoids scanning unrelated pending outbox rows.

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- timestamp: 2026-07-28T06:19:12Z | phase: quota-alert-foundation | commands/results: `go test ./internal/quotaalert ./internal/store -count=1` PASS; `go test -race ./internal/quotaalert -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS without skips; YAML-owned surface check PASS; `git diff --check` PASS; `go test ./... -count=1` PASS; gofmt check PASS; `zharness audit --json` PASS; full Security/Performance/Architecture/Code Quality review FAIL with four major finding groups and no critical findings | run_id: 01KYKK2CDAB7SR569Y3ZXEES95 | check_id: 01KYKP0ZK8APNVVT0H4ZYGMWYP | verdict: REQUEST_CHANGES | proof_gaps: settings snapshot/secret-boundary concurrency; cancellation-safe collection ownership and database-clock delivery leases; transition/state freshness invariants; bounded and integrity-checked terminal outbox retention
- timestamp: 2026-07-28T06:59:13Z | phase: quota-alert-foundation | commands/results: `go test ./internal/quotaalert ./internal/store -count=1` PASS; `go test -race ./internal/quotaalert -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS without skips; `go test ./... -count=1` PASS; YAML-owned surface check PASS; gofmt check PASS; `git diff --check` PASS; secret-pattern scan across 15 changed/untracked files PASS; full Security/Performance/Architecture/Code Quality review FAIL with one major finding and no critical findings: `CommitCollection` bypasses `TransitionEvent.Normalize` and persists the original noncanonical event | run_id: 01KYKP3RWHS0XMCQSS82ZQYBRA | check_id: 01KYKRAE46M01QSD2WG35N2TRJ | verdict: REQUEST_CHANGES | proof_gaps: persistence-level regression proving contradictory transitions are rejected and canonical event fields are stored consistently with notification batches
- timestamp: 2026-07-29T00:00:00Z | phase: quota-alert-foundation | commands/results: `go test ./internal/quotaalert ./internal/store -count=1` PASS; `go test -race ./internal/quotaalert -count=1` PASS; `go test ./... -count=1` PASS; real PostgreSQL 16 `go test ./internal/store -run '^TestPostgresQuotaAlert' -count=1` PASS without skips; changed-file static validation PASS for 18 files: gofmt, YAML-owned surface, `git diff --check`, whitespace, and secret patterns; focused in-process SQL review CLEAN for transition provenance, tombstones, retention locking, primary-key lease refresh, and seekable pagination; `zharness audit --json` PASS; prior background review agents were stopped and no incomplete result was used | run_id: 01KYKP3RWHS0XMCQSS82ZQYBRA | check_id: 01KYNR02XCV0EQBPJ44VZC074E | verdict: APPROVED | proof_gaps: none
- timestamp: 2026-07-29T02:43:17Z | phase: quota-provider-collectors | commands/results: `go test ./internal/quotaalert -run '^Test.*Collector' -count=1` PASS; `go test ./internal/runtime/executor -run 'Test(ParseKiroUsageLimits|KiroQuotaStateExhausted|KiroExecutorFetchQuota|Antigravity.*Credits)' -count=1` PASS; `go test -race ./internal/quotaalert -run '^Test.*Collector' -count=1` PASS; `go test ./internal/quotaalert ./internal/runtime/executor -count=1` PASS; `go test -race ./internal/quotaalert -run '^Test.*Collector' -count=1` PASS; `git diff --check` PASS; forbidden-pattern scan found only fixed provider constants, raw decode buffers, and negative attacker-URL tests | run_id: 01KYNSSJAVX9HF8J3MJEA7P65R | check_id: 01KYNW37X1678T95V3Z8N99NM7 | verdict: APPROVED | proof_gaps: none

## Current State and Next Action
- active_phase: quota-provider-collectors
- lifecycle_status: approved
- latest_run_id: 01KYNSSJAVX9HF8J3MJEA7P65R
- latest_trace_ids: [01KYKM47P0AEZ37S24GMR91ZPY, 01KYKN1WJ1N0DBEGDJCD14ME68, 01KYKNBVYYK3RMASMFECJ12X8G, 01KYKQFXKJ4W23WFRC8EWV351T, 01KYKWJ4X1NP4P9QBWAS4H4AKJ, 01KYM1T7H87E9FRF6GPTR7BHPX, 01KYNQZP2TZFZHX2T9VSZ6NXJG, 01KYNT1QFG7WGRFZQAVQAWQ0B5, 01KYNTKFVJS31JMMHXWZM87RS2, 01KYNTZX0Z2XP4GARDSGK7G1NY, 01KYNV8E6STPCQS0Q7CC56ZQ5R, 01KYNVYAVHNXWZ778T1WRTD92S, 01KYNW1A9BTD70BS6GJ0TRT6AR]
- latest_check_id: 01KYNW37X1678T95V3Z8N99NM7
- latest_handoff_id: 01KYKJGBHYY4VZ0GSC0GC9WERY
- completed_work:
  - Locked initiative `quota-alert-monitoring` with stable plan/intake IDs and database-only configuration requirements.
  - Completed full planning and created four DB-backed stories in dependency order.
  - Reconciled, committed, and pushed the Harness 0.6 docs-first migration as `e9baa966`.
  - Completed all F1-F4 implementation waves and automated phase checks, including non-skipping PostgreSQL 16 integration proof and the repository-wide Go suite.
  - Remediated all four blocker groups from check `01KYKP0ZK8APNVVT0H4ZYGMWYP` and recorded trace `01KYKQFXKJ4W23WFRC8EWV351T`.
  - Remediated check `01KYKRAE46M01QSD2WG35N2TRJ` plus final integrity-review findings and recorded trace `01KYKWJ4X1NP4P9QBWAS4H4AKJ`; focused, race, real-PostgreSQL, and repository-wide suites pass.
  - Completed architecture/performance follow-up remediation and recorded trace `01KYM1T7H87E9FRF6GPTR7BHPX`: persisted transition provenance and lease ownership, two-sided delivery tombstones, commit-refreshed seekable claims, and explicit total workload caps all pass PostgreSQL regressions.
  - Completed final gate trace `01KYNQZP2TZFZHX2T9VSZ6NXJG` and approved check `01KYNR02XCV0EQBPJ44VZC074E`: invalid initial transitions, retired replay, pruned ID reuse, opposite-order pruning, primary-key lease refresh, equal-timestamp pagination, and database JSON constraint regressions all pass.
  - Started `quota-provider-collectors` full-mode run `01KYNSSJAVX9HF8J3MJEA7P65R`; wave 1 C1 is complete with trace `01KYNT1QFG7WGRFZQAVQAWQ0B5`.
  - Completed C2 Claude/Codex collectors with trace `01KYNTKFVJS31JMMHXWZM87RS2`: usage failures stay fatal, profile/reset-credit enrichment stays non-fatal, percentage windows normalize to remaining quota, and bare Codex denial without reset/usage evidence is not treated as exhausted.
  - Completed C3 Gemini CLI/Antigravity collectors with trace `01KYNTZX0Z2XP4GARDSGK7G1NY`: fixed POST endpoints, grouped quota buckets/models, non-fatal supplementary calls, and existing Antigravity credits tests pass.
  - Completed C4 Kimi/xAI collectors with trace `01KYNV8E6STPCQS0Q7CC56ZQ5R`: fixed GET endpoints, row/billing normalization, reset timestamps, and sanitized failures pass.
  - Completed C5 Kiro collector adapter with trace `01KYNVYAVHNXWZ778T1WRTD92S`: fixed CodeWhisperer/Q collector attempts, executor parser parity, metadata quota URL isolation, and sanitized failures pass.
  - Completed C6 provider parity and default collector registry integration with trace `01KYNW1A9BTD70BS6GJ0TRT6AR`; collector, executor parity, race, full package, diff hygiene, and forbidden-pattern gates pass under approved check `01KYNW37X1678T95V3Z8N99NM7`.
- blockers:
  - none
- proof_gaps:
  - none
- open_items:
  - Commit and push the approved `quota-provider-collectors` phase when ready.
- exact_next_action: `/git cp` for the approved `quota-provider-collectors` phase changes
