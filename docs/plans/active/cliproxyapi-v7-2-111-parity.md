---
id: 01KYTYDKMCXHN0TQAPH1RM5K71
type: plan
intake_id: 01KYTYE1AK42PW4BGH0V6H5G8D
lane: high-risk
status: active
created: 2026-07-31
updated: 2026-07-31
---

# Plan: CLIProxyAPI v7.2.111 Targeted Parity with v7.2.112 Checkpoint Delta

## Outcome
- result: llmhub gains the approved fixes, compatibility improvements, model updates, operator controls, full Codex Live subsystem, and full Home runtime evolution available through CLIProxyAPI `v7.2.111`, adapted behind llmhub's existing architecture rather than merged wholesale; the final checkpoint records current upstream release `v7.2.112` with its newly published delta classified as explicit follow-up/reject scope per R23.
- success_signals:
  - The source checkpoint records latest `router-for-me/CLIProxyAPI` release `v7.2.112`, commit `a63da8ae76b1a4e0c0486c3eb0fb7ccf8f33e69d`, published `2026-07-31T08:39:29Z`, and local comparison baseline `234daa3fe1aab28e6a0b849b2f4c81d8a383c5e1`; completed implementation scope remains pinned to `v7.2.111`, commit `4a315136730baa8b3a436d12b74e5a702c70be5c`.
  - Every product-relevant upstream change from the prior approved `v7.2.93` checkpoint through `v7.2.111` has an explicit disposition: already present, adapt, reject, or superseded locally; every `v7.2.112` final-gate commit has an explicit follow-up or reject disposition.
  - Approved behavior is implemented without replacing Postgres runtime authority, Amp routing, Kiro support, embedded management web, public SDK boundaries, or llmhub branding and release contracts.
  - Codex Live sessions, sideband WebSocket relay, WebRTC media relay, and TCP relay work through llmhub's runtime and configuration model with deterministic tests.
  - Home membership, takeover, discovery, dispatch, affinity, replay/CAS, refresh, recovery, and concurrency behavior is adapted without regressing credential selection or lifecycle release, pinned to the approved `v7.2.111` behavior pending review of the `v7.2.112` Home 401 revert.
  - Applicable credential-weight and cloaking controls are persisted as structured database settings and exposed through the existing management API and current web UI/UX patterns.
  - Focused tests, `go test ./...`, Go build, frontend type/lint/build checks, `make build`, JSON validation, and `git diff --check` pass before closure.
  - Final validation re-checks the latest stable upstream release and updates the checkpoint to that release/version and commit; if the latest release introduces product deltas outside the locked requirements, those are pinned as explicit follow-up rather than silently declared complete.

## Authority and Requirements
- authority:
  - User-approved decisions on 2026-07-31: targeted semantic ports; include full Codex Live; include full Home parity; expose applicable controls through Postgres, management API, and the current llmhub web UI/UX.
  - Upstream repository: `https://github.com/router-for-me/CLIProxyAPI`.
  - Initial implementation checkpoint: release `v7.2.111`, commit `4a315136730baa8b3a436d12b74e5a702c70be5c`, published `2026-07-30T18:56:58Z`; `upstream/main` matched this commit when checked on 2026-07-31.
  - Final latest-release checkpoint: release `v7.2.112`, commit `a63da8ae76b1a4e0c0486c3eb0fb7ccf8f33e69d`, published `2026-07-31T08:39:29Z`; the final gate pins implemented scope to `v7.2.111` and records the `v7.2.112` delta as follow-up/reject scope per R23.
  - Local immutable comparison baseline: `234daa3fe1aab28e6a0b849b2f4c81d8a383c5e1` on `master`.
  - Local Git checkpoint refs: `refs/upstream-checkpoints/cliproxyapi/v7.2.93`, `refs/upstream-checkpoints/cliproxyapi/v7.2.96`, `refs/upstream-checkpoints/cliproxyapi/v7.2.111`, and `refs/upstream-checkpoints/cliproxyapi/v7.2.112`.
  - Prior parity authority: `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/` and decisions `0010` through `0013`.
  - Prior selective post-checkpoint adaptation: local commit `f6294be54166cb481edb032b1e6f86052431cf2a`, which ports credential concurrency and token-estimation behavior from upstream `v7.2.96`.
  - Repository architecture and verification rules in `CLAUDE.md`, including Postgres runtime authority and the prohibition on new frontend test files under `web/`.
  - Structural gap scan on 2026-07-31: 411 paths changed upstream from `v7.2.93` to `v7.2.111`; 11 exactly match the target, 17 remain at the upstream baseline, 166 upstream additions are absent locally, 68 changed upstream paths are absent because of local divergence, and 149 paths require semantic review.
- requirements:
  - R1 [accepted]: Maintain one durable upstream ledger covering every non-merge product commit and release from `v7.2.94` through the checkpoint release, with source commit, affected surface, local status, decision, dependencies, and verification owner. | source: upstream checkpoint; user request to check all gaps, updates, fixes, and features
  - R2 [accepted]: Adapt behavior in bounded slices behind existing llmhub interfaces; do not merge or overwrite the upstream tree wholesale. | source: user-approved targeted semantic ports; prior parity design
  - R3 [accepted]: Preserve Postgres as the authoritative runtime source for configuration, credentials, cooldown state, usage, and new feature settings; no new runtime dependency may read authoritative state from local YAML or files. | source: `CLAUDE.md`; database-only feature-config project constraint
  - R4 [accepted]: Preserve Amp routes and behavior, Kiro provider support, Gemini CLI paths, embedded management web, public SDK contracts, and existing provider-specific route semantics unless an individually approved parity fix requires an additive compatible change. | source: repository architecture; prior parity authority
  - R5 [accepted]: Port or formally supersede the v7.2.94 management auth-file identity/index filtering fix using llmhub's database-backed credential identity model. | source: upstream commits `d25b6b41`, `36b45d57`
  - R6 [accepted]: Reconcile v7.2.95-v7.2.98 translator performance, xAI token counting, Claude token estimation, Codex Alpha Search routing, credential concurrency, WebSocket continuity, atomic tool-cache state, normalized usage accounting, and Codex multi-agent-v2 behavior against already-ported local equivalents before adding code. | source: upstream releases `v7.2.95`-`v7.2.98`; local commit `f6294be5`
  - R7 [accepted]: Implement the full upstream Codex Live capability set: session handling, sideband protocol, realtime WebRTC media relay, TCP proxy, configuration/diff propagation, lifecycle shutdown, logging, and deterministic tests, adapted to llmhub service startup and Postgres configuration. | source: user-approved full Codex Live; upstream release `v7.2.99`
  - R8 [accepted]: Implement Codex and Claude client model catalogs and response builders without bypassing llmhub's registry, configured display-name contract, model visibility rules, or provider/auth routing. | source: upstream release `v7.2.99`; decision `0013`
  - R9 [accepted]: Port applicable Antigravity replay, signature, schema-sanitization, tool-provenance, and response-format fixes while preserving llmhub's existing translator and cache boundaries. | source: upstream releases `v7.2.100`-`v7.2.105`
  - R10 [accepted]: Port executor/runtime correctness changes for deferred-tool cache control, tool-result ordering, derived sessions, executor binding, xAI output controls, OAuth tool-name restoration, video failure reporting, and terminal/completion handling without weakening request-scoped error classification. | source: upstream releases `v7.2.100`-`v7.2.103`; decisions `0010`-`0012`
  - R11 [accepted]: Implement full Home parity through the final checkpoint, including live alias groups, native session affinity, edge-control validation, membership and takeover eligibility, cluster discovery, dispatch/reconnect state, replay CAS, credential refresh before 401 retry, OAuth recovery after 401, and lifecycle/concurrency release semantics. | source: user-approved full Home parity; upstream releases `v7.2.101`-`v7.2.111`
  - R12 [accepted]: Add weighted credential scheduling and validation across credential ingestion, selection, service configuration, and management operations; persist weights as structured database data and preserve existing provider-level scheduling semantics unless explicitly superseded by tests. | source: upstream commits `5dcca50f`, `e8e39526`; user-approved database/API/UI controls
  - R13 [accepted]: Port PostgreSQL cooldown persistence only through llmhub's existing database schema/repository patterns, with migrations, restart recovery, concurrency tests, and no upstream file-store authority. | source: upstream commit `f329b9d1`; Postgres runtime authority
  - R14 [accepted]: Port applicable translator fidelity changes for tool-call streaming, `[DONE]` and completion handling, `input_image` tool output, Claude tool schema normalization, Gemini structured output, token metadata including cached creation tokens, Codex function calls, grouped Claude tool results, and `json_schema`/`json_object` correctness. | source: upstream releases `v7.2.102`-`v7.2.107`
  - R15 [accepted]: Port applicable model/catalog updates through the final checkpoint, including approved Claude, Gemini, Codex reasoning-level, and Kimi K3 256K metadata; removals or changed model defaults must be reconciled against llmhub's registry JSON, static definitions, display-name behavior, and remote model updater. | source: upstream releases `v7.2.100`-`v7.2.111`; decision `0013`
  - R16 [accepted]: Add configurable Claude model-list cloaking and Codex cloaking/header behavior only as Postgres-backed structured settings with management API validation, safe defaults, and current-UX web controls. | source: upstream commits `69144785`, `a80e8082`; user-approved database/API/UI controls
  - R17 [accepted]: Reconcile Codex configured-model resolution so built-in IDs are not force-injected when local configuration says otherwise, while preserving llmhub aliases, prefix clones, display names, auth selection, and Codex-compatible catalogs. | source: upstream releases `v7.2.106`-`v7.2.109`; decision `0013`
  - R18 [accepted]: Any management API change must have Go handler/service/store coverage, authorization behavior consistent with existing endpoints, and backward-compatible response shapes unless a versioned additive contract is documented. | source: repository architecture; high-risk public-contract lane
  - R19 [accepted]: Any web change must reuse current layout, typography, design tokens, shadcn-style components, navigation, interaction patterns, loading/error states, and i18n structure; upstream visual design must not replace llmhub UI/UX. | source: user UI/UX constraint; existing `web/` system
  - R20 [accepted]: Frontend verification must use type checking, linting, production build, and browser runtime checks; no new test file may be created under `web/`. | source: `CLAUDE.md`
  - R21 [accepted]: Each phase must start from the checkpointed sources, use bounded file ownership, include focused regression tests, and record exact evidence sufficient to distinguish adapted, already-present, rejected, and blocked upstream behavior. | source: prior high-risk parity validation model
  - R22 [accepted]: The final gate must run the complete Go and frontend verification contract, re-query the latest non-prerelease upstream release, fetch its immutable commit into `refs/upstream-checkpoints/cliproxyapi/<version>`, and update this plan's checkpoint/ledger before completion. | source: user checkpoint constraint; repository prove-before-done rule
  - R23 [accepted]: If the latest stable upstream release changes after planning, implementation must not silently widen scope; the ledger must identify the delta and either include it through an approved plan refinement or pin the completed scope with an explicit follow-up. | source: targeted-scope policy; checkpoint integrity

## Non-goals
- NG1: Wholesale merging, rebasing, or replacing llmhub with the upstream source tree is excluded; this initiative ports approved semantics only.
- NG2: Upstream pluginhost, pluginstore, plugin SDK/ABI, request-lifecycle plugin framework, plugin examples, and Home plugin synchronization are excluded even where they appear in the checkpoint range.
- NG3: Replacing Postgres runtime authority with `config.yaml`, local auth files, watcher-owned state, Git-backed runtime state, or any other upstream file-based source is excluded.
- NG4: Removing or redesigning Amp, Kiro, Gemini CLI, embedded management web, provider-specific routes, quota-alert features, or existing llmhub SDK surfaces is excluded.
- NG5: Importing upstream web styling, branding, logos, sponsors, translated README churn, release assets, Docker/release workflows, installer behavior, or documentation showcases is excluded.
- NG6: Refactoring large local files merely to match upstream file splits is excluded unless a bounded split is necessary to safely implement and verify an approved behavior.
- NG7: Adding controls to the UI when the underlying feature is rejected or not operator-configurable is excluded; UI work follows approved backend contracts only.
- NG8: Production deployment, credential-based live-provider validation, commit, push, pull request, merge, or release publication is excluded from this specification and requires separate authorization.

## Approach and Risks
- approach: Use an immutable-source, ledger-first semantic backport. Each upstream behavior is classified against local code, reproduced with a focused test where applicable, then implemented behind existing llmhub interfaces. Database contracts precede runtime consumers; protocol/runtime work precedes model and UI exposure; Codex Live is split into session/control and media/relay phases; the final phase refreshes the checkpoint and closes any newly published release delta without silently widening scope.
- constraints:
  - Source comparisons use immutable refs under `refs/upstream-checkpoints/cliproxyapi/`; `upstream/main` is informative only.
  - The canonical initiative artifact remains this plan; the machine-readable checkpoint and commit ledger live in one additive `docs/upstream/cliproxyapi-checkpoint.json` file.
  - Runtime configuration and new feature settings are database-authoritative. Existing YAML structs may remain as decoded compatibility shapes, but startup cannot make files authoritative.
  - Existing Amp, Kiro, Gemini CLI, provider routes, public SDK boundaries, embedded web, and local design system remain intact.
  - Upstream file splits are copied only when they reduce implementation risk for an approved behavior; source layout parity is not a goal.
  - No new frontend test files are created under `web/`; web proof uses type-check, lint, build, and browser runtime checks.
  - Live-provider credentials are optional evidence. Deterministic local protocol, lifecycle, relay, and persistence tests are mandatory.
  - Phase definitions and task contracts below are immutable after planning; execution status is append-only under Progress.
- dependencies:
  - Public GitHub access to query and fetch `router-for-me/CLIProxyAPI` releases and immutable commits.
  - A Postgres test DSN for integration tests that cannot be proved with unit-level stores; tests must skip cleanly when the documented env var is absent.
  - Local network loopback, TLS fixtures, and WebRTC/TCP test doubles for Codex Live and Home protocol verification.
  - Bun for frontend verification and embedding.
- rejected_alternatives:
  - Wholesale upstream merge: rejected because histories and architecture have diverged and the local tree contains Postgres, Amp, Kiro, quota-alert, SDK, branding, and web contracts absent upstream.
  - Critical-fixes-only update: rejected because the approved scope explicitly includes full Codex Live, full Home parity, models, weights, and management controls.
  - File/YAML-backed feature toggles: rejected because new llmhub feature configuration is database-only.
  - One giant parity phase: rejected because Home, translators, database state, media relay, and UI have different failure modes and independent verification needs.
  - Importing upstream plugin infrastructure to simplify patch application: rejected as an explicit non-goal and an unnecessary execution surface.
- risks:
  - risk: Patch-equivalent behavior may already exist under different local names, causing duplicate or conflicting implementations.
    mitigation: The ledger phase classifies each commit using tests, symbols, and behavior; implementation starts only from `adapt` entries.
    recovery: Revert only the duplicate slice and mark the ledger entry `already-present` or `superseded-local` with evidence.
  - risk: Full Home parity introduces distributed-state races, stale ownership, duplicate release, or 401 retry loops.
    mitigation: Preserve exactly-once executionregistry release, add deterministic fake-Home protocol tests, bound retries, and test shutdown/reconnect interleavings under `-race`.
    recovery: Disable the new Home path through its database setting while retaining the prior v7.2.96 concurrency behavior; reverse only the active Home slice.
  - risk: Credential weights and cooldown persistence alter scheduler fairness or availability semantics.
    mitigation: Define validation and deterministic selection invariants before wiring persistence; test mixed providers, aliases, disabled credentials, restarts, and concurrent updates.
    recovery: Fall back to neutral weight `1` and existing in-memory cooldown behavior without deleting stored values.
  - risk: Codex Live media relay expands the attack and resource surface through UDP/WebRTC/TCP forwarding.
    mitigation: Bind only configured interfaces, validate targets, enforce deadlines and size/concurrency bounds, redact credentials, and test cancellation and half-close behavior.
    recovery: Keep Live disabled by default and independently disable media/TCP relay while preserving non-Live Codex execution.
  - risk: Translator and signature changes can corrupt streaming state or tool identity across protocols.
    mitigation: Add golden event-sequence tests for fragmented calls, indices, completion, schemas, signatures, and replay before changing shared helpers.
    recovery: Reverse the smallest translator/provider slice and retain the prior decision contracts `0010`-`0013`.
  - risk: Model catalog changes can silently alter routing, aliases, or visible IDs.
    mitigation: Separate presentation metadata from routing identity and assert registry, configured-only Codex, prefix, alias, and display-name behavior across all catalog endpoints.
    recovery: Remove only affected model entries or catalog exposure while keeping unrelated runtime fixes.
  - risk: Database migrations or settings updates can leave a partially upgraded runtime.
    mitigation: Use idempotent `EnsureSchema` operations, seeded defaults, revisioned writes, transaction tests, and restart round trips.
    recovery: Keep old columns/read paths compatible and ignore new tables/settings when absent; never destructively migrate existing rows.
  - risk: Upstream may publish another stable release during implementation.
    mitigation: Re-query latest stable release at phase 1 and final gate; record the delta without silently changing planned tasks.
    recovery: Refine this same plan for required product changes or pin the completed checkpoint and create an explicit follow-up.
- stop_conditions:
  - Stop if an approved behavior requires pluginhost/pluginstore, file-authoritative runtime state, removal of a protected local subsystem, or an undocumented public API break.
  - Stop a phase when deterministic tests expose a conflict with prior decisions `0010`-`0013`, a database migration is not backward-compatible, or a relay cannot be bounded safely.
  - Stop final closure if the checkpoint file, plan validation record, Git ref, release tag, and commit SHA disagree.
- recovery_policy: Reverse only the current bounded slice in dependency order, preserve database rows through additive compatibility, append the failure and evidence under Progress/Validation, and resume from the last checked phase.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: upstream-parity-ledger
    story_id: 01KYTYK4EWSZJW4PPRJC2BYY1B
    status: done
    goal: Classify every product-relevant CLIProxyAPI change from the prior checkpoint through the current stable release and establish the durable machine-readable checkpoint.
    depends_on: none
    requirement_trace: [R1, R2, R6, R21, R22, R23]
    allowed_surfaces:
      - `docs/upstream/cliproxyapi-checkpoint.json`
      - append-only `Progress`, `Decisions`, and `Validation` entries in this plan
      - local Git refs under `refs/upstream-checkpoints/cliproxyapi/`
    avoided_surfaces:
      - production Go and web code
      - upstream plugin, branding, release, and documentation showcase content
    waves:
      - wave: ledger-source-freeze
        tasks:
          - task_id: ledger-refresh-source
            goal: Re-query the latest non-prerelease release, fetch its tag commit into an immutable checkpoint ref, and record release time, tag, SHA, URL, local baseline, prior checkpoints, and comparison timestamp.
            depends_on: none
            touched_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`, Git checkpoint refs]
            expected_output: A schema-versioned checkpoint object whose source metadata matches GitHub and `git show-ref` exactly.
            checks:
              - `gh api repos/router-for-me/CLIProxyAPI/releases/latest --jq '{tag_name,published_at,html_url,target_commitish}'`
              - `git show-ref refs/upstream-checkpoints/cliproxyapi/v7.2.93 refs/upstream-checkpoints/cliproxyapi/v7.2.96 refs/upstream-checkpoints/cliproxyapi/v7.2.111`
              - `python3 -m json.tool docs/upstream/cliproxyapi-checkpoint.json >/dev/null`
          - task_id: ledger-commit-inventory
            goal: Enumerate every non-merge product commit and release from `v7.2.94` through the checkpoint, grouping commits by management, auth/Home, persistence, translator, executor, model, Codex Live, plugin, docs/release, and local-conflict surfaces.
            depends_on: ledger-refresh-source
            touched_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`]
            expected_output: Complete ordered commit entries with upstream files, release membership, dependency notes, and no unclassified product commit.
            checks:
              - `git log --reverse --no-merges --format='%H%x09%s' refs/upstream-checkpoints/cliproxyapi/v7.2.93..refs/upstream-checkpoints/cliproxyapi/v7.2.111`
              - `git diff --name-status refs/upstream-checkpoints/cliproxyapi/v7.2.93..refs/upstream-checkpoints/cliproxyapi/v7.2.111`
      - wave: ledger-local-disposition
        tasks:
          - task_id: ledger-semantic-disposition
            goal: Assign every product commit one disposition from `already-present`, `adapt`, `superseded-local`, or `reject`, with local paths, tests, rationale, and owning phase.
            depends_on: ledger-commit-inventory
            touched_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`]
            expected_output: Zero unresolved product entries and explicit rejection of plugin/release/branding churn.
            checks:
              - `python3 -m json.tool docs/upstream/cliproxyapi-checkpoint.json >/dev/null`
              - `python3 - <<'PY'
import json
p='docs/upstream/cliproxyapi-checkpoint.json'
d=json.load(open(p))
entries=d['entries']
assert entries
assert all(e.get('disposition') in {'already-present','adapt','superseded-local','reject'} for e in entries)
assert all(e.get('owner_phase') or e['disposition'] == 'reject' for e in entries)
print(len(entries))
PY`
    phase_checks:
      - `git diff --check`
      - `zharness query phases --json`
    stop_conditions:
      - Any product commit lacks a disposition, local evidence, or owning phase.
      - The queried stable release differs from the locked checkpoint and the delta cannot be classified without changing scope.
    escalation: Refine this same plan before implementation if the stable-release delta adds product behavior outside existing requirements.

  - phase_slug: postgres-control-foundation
    story_id: 01KYTYK4F3MEEFC38M0RHCWPS9
    status: done
    goal: Create database-backed settings and state foundations required by approved controls, cooldown persistence, Home, and Codex Live.
    depends_on: upstream-parity-ledger
    requirement_trace: [R3, R12, R13, R16, R18, R21]
    allowed_surfaces:
      - `internal/store/postgresstore.go`
      - additive `internal/store/postgres_*` files and integration tests
      - focused setting/domain packages under `internal/`
      - service/store interfaces required to load revisioned settings
    avoided_surfaces:
      - YAML as runtime authority
      - web UI
      - provider protocol behavior
    waves:
      - wave: postgres-schema-domain
        tasks:
          - task_id: define-runtime-control-domain
            goal: Define validated, defaulted domain types for credential weights, Claude/Codex cloaking, Codex Live controls, Home controls, and persisted cooldown snapshots.
            depends_on: none
            touched_surfaces: [`internal/config/`, additive focused domain packages under `internal/`]
            expected_output: Pure validation/default contracts with neutral backward-compatible defaults and no file persistence.
            checks:
              - `go test ./internal/config/... -count=1`
          - task_id: add-idempotent-postgres-schema
            goal: Add idempotent tables/columns/indexes and seeded singleton rows using existing `EnsureSchema` conventions and revisioned writes.
            depends_on: define-runtime-control-domain
            touched_surfaces: [`internal/store/postgresstore.go`, additive `internal/store/postgres_*` files]
            expected_output: Additive schema that upgrades empty and existing databases without destructive changes.
            checks:
              - `go test ./internal/store -run 'TestPostgresStore.*(Init|Schema|Settings|Cooldown)' -count=1`
              - `LLMHUB_POSTGRES_TEST_DSN="$LLMHUB_POSTGRES_TEST_DSN" go test ./internal/store -run 'TestPostgres.*(Settings|Cooldown|Migration|Restart)' -count=1`
      - wave: postgres-repository-contracts
        tasks:
          - task_id: implement-revisioned-settings-store
            goal: Implement atomic load/save APIs for runtime controls, including revision conflicts, defaults, and restart round trips.
            depends_on: add-idempotent-postgres-schema
            touched_surfaces: [`internal/store/`, `sdk/cliproxy/` service dependencies]
            expected_output: Transactional database repository with deterministic in-memory/unit seams and no YAML write path.
            checks:
              - `go test ./internal/store ./sdk/cliproxy -run 'Test.*(RuntimeSettings|Revision|Restart)' -count=1`
          - task_id: implement-cooldown-snapshot-store
            goal: Persist and restore cooldown windows without changing request-scoped availability classification or writing stale snapshots over newer state.
            depends_on: add-idempotent-postgres-schema
            touched_surfaces: [`internal/store/`, `sdk/cliproxy/auth/` store interfaces only]
            expected_output: Restart-safe cooldown store with expiry cleanup and compare-before-write semantics.
            checks:
              - `go test ./internal/store ./sdk/cliproxy/auth -run 'Test.*Cooldown.*(Persist|Restore|Expiry|Concurrent)' -count=1`
    phase_checks:
      - `go test ./internal/config/... ./internal/store/... ./sdk/cliproxy/... -count=1`
      - `go vet ./internal/config/... ./internal/store/... ./sdk/cliproxy/...`
      - `git diff --check`
    stop_conditions:
      - A schema operation drops/rewrites existing data or startup requires local config files.
      - Revisioned writes cannot prevent lost updates.
    escalation: Keep the new feature disabled and stop consumers until additive schema and repository contracts are proven.

  - phase_slug: auth-weight-management-parity
    story_id: 01KYTYK4F9VD1FZRB2BS4Q1E36
    status: done
    goal: Adapt auth filtering, credential weights, weighted scheduling, cooldown restoration, and management contracts.
    depends_on: postgres-control-foundation
    requirement_trace: [R5, R10, R12, R13, R18, R21]
    allowed_surfaces:
      - `sdk/cliproxy/auth/`
      - `internal/api/handlers/management/auth_files.go` and focused additive auth-management files
      - `internal/store/`
      - `sdk/cliproxy/builder.go`, `sdk/cliproxy/service.go`, and focused service wiring
      - existing auth-file web DTO compatibility types only when backend contracts require them
    avoided_surfaces:
      - Home membership protocol internals
      - Codex Live
      - unrelated provider translators
    waves:
      - wave: auth-identity-weight
        tasks:
          - task_id: port-auth-identity-filtering
            goal: Filter and address credentials by stable database identity/auth index without filename assumptions or cross-provider collisions.
            depends_on: none
            touched_surfaces: [`internal/api/handlers/management/auth_files.go`, focused handler tests, auth store lookups]
            expected_output: Backward-compatible auth list/filter/update behavior for name, provider, and auth index.
            checks:
              - `go test ./internal/api/handlers/management -run 'Test.*Auth.*(Filter|Index|Identity)' -count=1`
          - task_id: implement-weight-validation-ingestion
            goal: Parse, validate, default, persist, and expose credential weights without accepting zero, negative, overflow, or malformed values.
            depends_on: none
            touched_surfaces: [`sdk/cliproxy/auth/`, `internal/store/`, management DTOs]
            expected_output: Neutral default weight `1`, canonical serialization, and validation at every ingestion path.
            checks:
              - `go test ./sdk/cliproxy/auth ./internal/api/handlers/management ./internal/store -run 'Test.*Weight' -count=1`
      - wave: scheduler-cooldown
        tasks:
          - task_id: implement-weighted-scheduler
            goal: Apply deterministic weighted round-robin among ready credentials while preserving provider rotation, aliases, disabled credentials, cooldowns, and session affinity.
            depends_on: implement-weight-validation-ingestion
            touched_surfaces: [`sdk/cliproxy/auth/scheduler.go`, `selector.go`, conductor selection paths]
            expected_output: Fair deterministic selection with no starvation and stable neutral behavior when all weights are `1`.
            checks:
              - `go test ./sdk/cliproxy/auth -run 'Test.*(Weighted|Weight|Scheduler|PickNext)' -count=1`
              - `go test -race ./sdk/cliproxy/auth -run 'Test.*(Weighted|Scheduler|Cooldown)' -count=1`
          - task_id: wire-cooldown-restart-state
            goal: Load persisted cooldowns at service startup and persist meaningful transitions without changing `invalid_grant`, CountTokens, or request-scoped error semantics.
            depends_on: implement-weighted-scheduler
            touched_surfaces: [`sdk/cliproxy/auth/`, `sdk/cliproxy/service.go`, `internal/store/`]
            expected_output: Availability survives restart and expired/stale cooldowns do not suppress usable credentials.
            checks:
              - `go test ./sdk/cliproxy/auth ./sdk/cliproxy ./internal/store -run 'Test.*Cooldown.*(Restart|Persist|Availability)' -count=1`
      - wave: auth-management-api
        tasks:
          - task_id: expose-auth-weight-management
            goal: Add authorized management read/update contracts for weight and identity filtering with revision/error behavior matching existing endpoints.
            depends_on: port-auth-identity-filtering
            touched_surfaces: [`internal/api/handlers/management/`, route registration, API tests]
            expected_output: Additive authenticated API with stable response shapes and explicit validation errors.
            checks:
              - `go test ./internal/api/handlers/management ./internal/api -run 'Test.*Auth.*(Weight|Filter|Management)' -count=1`
    phase_checks:
      - `go test ./sdk/cliproxy/auth ./sdk/cliproxy ./internal/api/handlers/management ./internal/store -count=1`
      - `go vet ./sdk/cliproxy/auth ./sdk/cliproxy ./internal/api/handlers/management ./internal/store`
      - `git diff --check`
    stop_conditions:
      - Neutral weights change existing selection order or a restart revives an expired cooldown.
      - Auth filtering can target a credential outside the authenticated operator's intended identity.
    escalation: Disable weighted selection and retain stored neutral values while isolating the failing scheduler or handler slice.

  - phase_slug: home-runtime-full-parity
    story_id: 01KYTYK4FGP9CGTDWN08PT26BM
    status: done
    goal: Adapt complete Home membership, affinity, dispatch, recovery, CAS replay, and concurrency behavior through the checkpoint.
    depends_on: auth-weight-management-parity
    requirement_trace: [R4, R6, R10, R11, R21]
    allowed_surfaces:
      - `internal/home/`
      - `sdk/cliproxy/auth/home_*`
      - `sdk/cliproxy/executionregistry/`
      - `sdk/cliproxy/service.go` and focused Home lifecycle files
      - `internal/runtime/executor/helps/home_refresh.go`
      - database-backed Home settings interfaces
    avoided_surfaces:
      - plugin-based Home synchronization
      - file-authoritative Home config
      - unrelated scheduler behavior after the prior phase gate
    waves:
      - wave: home-protocol-state
        tasks:
          - task_id: port-membership-discovery-takeover
            goal: Implement membership state, cluster discovery, takeover eligibility, alias groups, native client affinity signals, and edge-control validation.
            depends_on: none
            touched_surfaces: [`internal/home/`, `sdk/cliproxy/auth/home_session.go`, additive session identity helpers]
            expected_output: Deterministic membership state machine with bounded discovery and validated session identity.
            checks:
              - `go test ./internal/home ./sdk/cliproxy/auth -run 'Test.*Home.*(Membership|Discovery|Takeover|Alias|Affinity|Control)' -count=1`
              - `go test -race ./internal/home ./sdk/cliproxy/auth -run 'Test.*Home.*(Membership|Takeover|Affinity)' -count=1`
          - task_id: port-home-cas-replay
            goal: Use Home CAS semantics for replay state and reject stale compare-and-swap updates without losing valid transcript state.
            depends_on: port-membership-discovery-takeover
            touched_surfaces: [`internal/home/`, Home request/testdata fixtures, request-scoped replay consumers]
            expected_output: Atomic replay ownership transitions with deterministic stale-writer tests.
            checks:
              - `go test ./internal/home ./sdk/cliproxy/auth -run 'Test.*Home.*(CAS|Replay|CompareAndSwap)' -count=1`
      - wave: home-dispatch-recovery
        tasks:
          - task_id: port-dispatch-reconnect-state
            goal: Adapt dispatch, reconnect failover, NewLifetime semantics, error reporting, takeover-aware release, and cluster config recovery.
            depends_on: port-membership-discovery-takeover
            touched_surfaces: [`internal/home/client.go`, `requests.go`, `concurrency_release.go`, `sdk/cliproxy/auth/home_*`]
            expected_output: Bounded reconnect and failover with exactly-once resource accounting.
            checks:
              - `go test ./internal/home ./sdk/cliproxy/auth ./sdk/cliproxy/executionregistry -run 'Test.*Home.*(Dispatch|Reconnect|Lifetime|Release|Failover)' -count=1`
              - `go test -race ./internal/home ./sdk/cliproxy/auth ./sdk/cliproxy/executionregistry -run 'Test.*(Dispatch|Release|Reconnect)' -count=1`
          - task_id: port-home-401-refresh-recovery
            goal: Refresh Home credentials before a single bounded 401 retry and recover OAuth material without infinite loops or duplicate in-flight accounting.
            depends_on: port-dispatch-reconnect-state
            touched_surfaces: [`internal/runtime/executor/helps/home_refresh.go`, Home auth/dispatch paths, provider executor auth hooks]
            expected_output: One refresh/retry path with explicit terminal failure and preserved fallback behavior.
            checks:
              - `go test ./internal/runtime/executor/helps ./sdk/cliproxy/auth ./internal/home -run 'Test.*Home.*(401|Unauthorized|Refresh|Recovery)' -count=1`
      - wave: home-service-lifecycle
        tasks:
          - task_id: integrate-home-full-lifecycle
            goal: Wire database settings, startup, reload, reconnect, release flushing, and shutdown into the existing service without plugin synchronization.
            depends_on: port-home-cas-replay
            touched_surfaces: [`sdk/cliproxy/service.go`, focused service Home files/tests, `internal/config/home.go`, database settings loader]
            expected_output: Idempotent start/reload/stop behavior with no leaked goroutines or stale ownership.
            checks:
              - `go test ./sdk/cliproxy ./internal/home -run 'Test.*Home.*(Lifecycle|Reload|Shutdown|Stale)' -count=1`
              - `go test -race ./sdk/cliproxy ./internal/home -run 'Test.*Home.*(Lifecycle|Shutdown)' -count=1`
    phase_checks:
      - `go test ./internal/home ./sdk/cliproxy/auth ./sdk/cliproxy/executionregistry ./sdk/cliproxy ./internal/runtime/executor/helps -count=1`
      - `go vet ./internal/home ./sdk/cliproxy/auth ./sdk/cliproxy/executionregistry ./sdk/cliproxy ./internal/runtime/executor/helps`
      - `git diff --check`
    stop_conditions:
      - Any path can release a credential twice, retry 401 indefinitely, accept invalid session control bytes, or allow stale replay ownership.
      - Full parity requires Home plugins or file-backed state.
    escalation: Keep new Home behavior disabled and restore the prior v7.2.96 lifecycle path while preserving database schema and evidence.

  - phase_slug: translator-protocol-parity-111
    story_id: 01KYTYK4FPD9V2YQ16F3WP82B2
    status: done
    goal: Adapt translator streaming, schema, tool-output, function-call, and usage fidelity through the checkpoint.
    depends_on: home-runtime-full-parity
    requirement_trace: [R6, R9, R10, R14, R21]
    allowed_surfaces:
      - `internal/translator/`
      - `sdk/translator/`
      - focused shared helpers under `internal/util/`
      - protocol handler tests only where transport completion semantics require them
    avoided_surfaces:
      - provider auth selection
      - model registry identity
      - UI
    waves:
      - wave: translator-request-fidelity
        tasks:
          - task_id: port-tool-schema-and-output-inputs
            goal: Normalize Claude tool input schemas, preserve `input_image` tool outputs, map Gemini structured output, and keep valid tool-result ordering/grouping.
            depends_on: none
            touched_surfaces: [`internal/translator/claude/`, `gemini/`, `gemini-cli/`, `codex/`, `common/`, `internal/util/`]
            expected_output: Protocol-valid request shapes with deterministic fallback for invalid mixed content.
            checks:
              - `go test ./internal/translator/... -run 'Test.*(Schema|InputImage|ToolResult|Structured|ResponseFormat)' -count=1`
          - task_id: reconcile-openai-tool-performance
            goal: Port only behavior-preserving tool conversion and multi-agent optimizations not already present from v7.2.96.
            depends_on: none
            touched_surfaces: [`internal/translator/openai/`, `internal/runtime/executor/helps/` translation helpers]
            expected_output: Equivalent output with focused allocation/benchmark evidence and no duplicate helper path.
            checks:
              - `go test ./internal/translator/openai/... ./internal/runtime/executor/helps -run 'Test.*(Tool|MultiAgent|Translate)' -count=1`
              - `go test ./internal/translator/openai/... -run '^$' -bench 'Benchmark.*Tool' -benchmem`
      - wave: translator-response-streaming
        tasks:
          - task_id: port-stream-completion-tool-calls
            goal: Handle fragmented tool names, custom/function conflicts, duplicate Claude deltas, `[DONE]`, terminal completion, provider output indices, and Codex function-call conversion without reopening retired state.
            depends_on: port-tool-schema-and-output-inputs
            touched_surfaces: [`internal/translator/*/*response*.go`, focused event-sequence tests]
            expected_output: Stable streaming and non-streaming tool identity, indices, completion, and event ordering.
            checks:
              - `go test ./internal/translator/... -run 'Test.*(Done|Completion|ToolCall|FunctionCall|OutputIndex|Delta)' -count=1`
          - task_id: port-usage-metadata-normalization
            goal: Normalize partial/canonical token accounting and preserve cached creation tokens across supported response protocols.
            depends_on: port-stream-completion-tool-calls
            touched_surfaces: [`internal/translator/`, `sdk/cliproxy/usage/`, executor usage helpers]
            expected_output: One canonical usage mapping with explicit partial accounting and no double counting.
            checks:
              - `go test ./internal/translator/... ./sdk/cliproxy/usage ./internal/runtime/executor/helps -run 'Test.*(Usage|Token|CachedCreation|Partial)' -count=1`
      - wave: translator-response-format
        tasks:
          - task_id: fix-json-response-formats
            goal: Correct `json_schema` and `json_object` mappings across OpenAI, Codex, Gemini, and Claude conversions using structured schema-cleaning options.
            depends_on: port-tool-schema-and-output-inputs
            touched_surfaces: [`internal/translator/`, `internal/util/` schema helpers]
            expected_output: Provider-correct structured-output requests with preserved strict/name/schema fields.
            checks:
              - `go test ./internal/translator/... ./internal/util/... -run 'Test.*(JSONSchema|JSONObject|ResponseFormat|SchemaClean)' -count=1`
    phase_checks:
      - `go test ./internal/translator/... ./sdk/translator/... ./sdk/cliproxy/usage ./internal/runtime/executor/helps -count=1`
      - `go vet ./internal/translator/... ./sdk/translator/... ./sdk/cliproxy/usage ./internal/runtime/executor/helps`
      - `git diff --check`
    stop_conditions:
      - A shared helper changes Kiro/Amp request shapes without explicit tests.
      - Streaming tests show duplicate, missing, reordered, or reopened events.
    escalation: Split the failing provider/protocol into a follow-up refinement rather than weakening shared correctness contracts.

  - phase_slug: antigravity-signature-parity
    story_id: 01KYTYK4FXWCAR9718NRRMBMEH
    status: done
    goal: Adapt Antigravity replay, provenance, signature, and schema-sanitization correctness.
    depends_on: translator-protocol-parity-111
    requirement_trace: [R9, R10, R14, R21]
    allowed_surfaces:
      - `internal/runtime/executor/antigravity_executor.go` and focused additive files
      - `internal/translator/antigravity/`
      - additive focused signature package under `internal/signature/`
      - existing cache package for bounded replay state
      - Antigravity-focused tests
    avoided_surfaces:
      - unrelated Claude/Gemini direct executors
      - global cache semantics
      - plugin signature adapters
    waves:
      - wave: signature-validation
        tasks:
          - task_id: implement-provider-signature-validation
            goal: Add CAIS, Gemini, Claude, GPT, and Grok signature validation/sanitization needed by approved replay paths with provider compatibility tests.
            depends_on: none
            touched_surfaces: [`internal/signature/`, `internal/translator/antigravity/`]
            expected_output: Pure validation/sanitization APIs that reject malformed signatures without discarding recoverable context.
            checks:
              - `go test ./internal/signature/... ./internal/translator/antigravity/... -run 'Test.*Signature' -count=1`
          - task_id: scope-antigravity-schema-cleaning
            goal: Preserve tool arguments and provenance while applying schema cleaning only to the intended Antigravity surfaces.
            depends_on: implement-provider-signature-validation
            touched_surfaces: [`internal/runtime/executor/antigravity*`, `internal/translator/antigravity/`, schema helpers]
            expected_output: No cross-provider mutation and complete tool argument preservation.
            checks:
              - `go test ./internal/runtime/executor ./internal/translator/antigravity/... -run 'Test.*Antigravity.*(Schema|Tool|Provenance)' -count=1`
      - wave: replay-provenance
        tasks:
          - task_id: port-antigravity-reasoning-replay
            goal: Preserve complete reasoning signature chains, recover provenance when context drifts, replay text thoughts safely, and clear bounded cache state at terminal boundaries.
            depends_on: implement-provider-signature-validation
            touched_surfaces: [`internal/cache/`, `internal/runtime/executor/antigravity*`, `internal/translator/antigravity/`]
            expected_output: Session-safe bounded replay that degrades gracefully instead of killing valid sessions.
            checks:
              - `go test ./internal/cache ./internal/runtime/executor ./internal/translator/antigravity/... -run 'Test.*Antigravity.*(Replay|Reasoning|Provenance|Drift)' -count=1`
              - `go test -race ./internal/cache ./internal/runtime/executor -run 'Test.*Antigravity.*Replay' -count=1`
    phase_checks:
      - `go test ./internal/cache ./internal/signature/... ./internal/runtime/executor ./internal/translator/antigravity/... -count=1`
      - `go vet ./internal/cache ./internal/signature/... ./internal/runtime/executor ./internal/translator/antigravity/...`
      - `git diff --check`
    stop_conditions:
      - Signature recovery accepts malformed provider data or replay leaks across sessions.
      - Schema cleaning removes tool arguments or changes non-Antigravity requests.
    escalation: Disable replay recovery while retaining strict validation and isolate the incompatible provider variant.

  - phase_slug: executor-websocket-runtime-parity
    story_id: 01KYTYK4G41ER484MG7D6MSTBA
    status: done
    goal: Adapt executor binding, lifecycle, WebSocket continuity, completion, cloaking, and runtime reliability changes.
    depends_on: antigravity-signature-parity
    requirement_trace: [R4, R6, R10, R14, R16, R21]
    allowed_surfaces:
      - `internal/runtime/executor/`
      - `internal/runtime/executor/helps/`
      - `sdk/api/handlers/`
      - `sdk/cliproxy/executor/`, `sdk/cliproxy/pipeline/`, and focused service binding files
      - database-backed cloaking settings consumers
    avoided_surfaces:
      - model catalog content
      - management web
      - Codex Live media transport
    waves:
      - wave: executor-binding-cloaking
        tasks:
          - task_id: reconcile-executor-binding
            goal: Preserve custom/local executors during auth sync, use config-aware executor binding, and keep Amp/Kiro/provider registrations stable.
            depends_on: none
            touched_surfaces: [`sdk/cliproxy/service.go`, `builder.go`, provider registration, executor interfaces]
            expected_output: Deterministic executor registry across startup, reload, auth updates, and configured providers.
            checks:
              - `go test ./sdk/cliproxy ./sdk/cliproxy/auth -run 'Test.*(Executor|Binding|Registration|AuthSync|Kiro|Amp)' -count=1`
          - task_id: implement-database-cloaking-controls
            goal: Apply Claude model-list and Codex request/header cloaking settings from database snapshots with backward-compatible defaults.
            depends_on: reconcile-executor-binding
            touched_surfaces: [`internal/runtime/executor/claude_executor.go`, `codex_executor.go`, helps/cloak files, service settings reload]
            expected_output: Hot-reloadable controls with no YAML authority and no effect when defaults are unchanged.
            checks:
              - `go test ./internal/runtime/executor ./sdk/cliproxy -run 'Test.*(Cloak|Header|ModelList|DeferredTool)' -count=1`
      - wave: websocket-continuity
        tasks:
          - task_id: port-websocket-context-continuity
            goal: Preserve transcript, compacted xAI state, replay state, and tool cache atomically across WebSocket transport changes and failures.
            depends_on: none
            touched_surfaces: [`internal/runtime/executor/codex_websockets_executor.go`, xAI WebSocket paths if applicable, `sdk/api/handlers/openai/openai_responses_websocket*`]
            expected_output: Request-scoped rollback/commit semantics with no state loss or partial cache mutation.
            checks:
              - `go test ./internal/runtime/executor ./sdk/api/handlers/openai -run 'Test.*WebSocket.*(Continuity|Transcript|Cache|Rollback|Compact)' -count=1`
              - `go test -race ./internal/runtime/executor ./sdk/api/handlers/openai -run 'Test.*WebSocket' -count=1`
          - task_id: port-runtime-terminal-reliability
            goal: Restore OAuth tool names, report video failures, preserve tool-result ordering, suppress duplicate completion, support derived sessions, and keep request-scoped error/fallback semantics.
            depends_on: port-websocket-context-continuity
            touched_surfaces: [`internal/runtime/executor/`, `sdk/api/handlers/`, `sdk/cliproxy/session/` or local equivalent]
            expected_output: Stable terminal output and lifecycle accounting for HTTP, stream, and WebSocket paths.
            checks:
              - `go test ./internal/runtime/executor ./sdk/api/handlers/... ./sdk/cliproxy/... -run 'Test.*(Terminal|Completion|ToolName|Video|DerivedSession|ToolResult)' -count=1`
      - wave: local-token-counting
        tasks:
          - task_id: reconcile-local-token-counters
            goal: Retain existing Claude/xAI estimation behavior, add only missing local counting semantics, and prevent double estimation when upstream usage is present.
            depends_on: port-runtime-terminal-reliability
            touched_surfaces: [`internal/runtime/executor/helps/claude_input_tokens.go`, token helpers, Claude/xAI executors]
            expected_output: Bounded deterministic counts with explicit media/control exclusions and first-event patching only when needed.
            checks:
              - `go test ./internal/runtime/executor ./internal/runtime/executor/helps -run 'Test.*(InputToken|TokenCount|Estimate|Usage)' -count=1`
    phase_checks:
      - `go test ./internal/runtime/executor/... ./sdk/api/handlers/... ./sdk/cliproxy/... -count=1`
      - `go vet ./internal/runtime/executor/... ./sdk/api/handlers/... ./sdk/cliproxy/...`
      - `git diff --check`
    stop_conditions:
      - WebSocket rollback loses transcript/tool state or any executor lifecycle leaks/release duplication occur.
      - Cloaking controls require file-backed config or alter default requests.
    escalation: Disable the affected optional control and reverse only the transport/provider slice while retaining proven shared lifecycle fixes.

  - phase_slug: model-catalog-resolution-parity
    story_id: 01KYTYK4GCR07WKC3HJC40AH1Z
    status: done
    goal: Adapt model catalogs, metadata, registry behavior, and configured-model resolution without changing local routing contracts.
    depends_on: executor-websocket-runtime-parity
    requirement_trace: [R8, R15, R17, R21]
    allowed_surfaces:
      - `internal/registry/`
      - `internal/registry/models/models.json`
      - `internal/registry/models/codex_client_models.json`
      - additive client catalog builders under `internal/client/{codex,claude}/models/` when local registry delegation is preserved
      - `sdk/api/handlers/{openai,claude,gemini}/` model responses
      - focused Codex resolution code in `sdk/cliproxy/`
    avoided_surfaces:
      - provider auth selection
      - upstream remote model authority replacing local registry
      - UI styling
    waves:
      - wave: model-data-catalogs
        tasks:
          - task_id: reconcile-model-metadata
            goal: Add, update, or reject Claude, Gemini, Codex reasoning, and Kimi K3 256K metadata based on executor support and local registry schema; reconcile upstream removals explicitly.
            depends_on: none
            touched_surfaces: [`internal/registry/model_definitions.go`, `internal/registry/models/models.json`, registry tests]
            expected_output: Valid static/JSON parity with no unsupported visible model and no routing identity changes from display metadata.
            checks:
              - `go test ./internal/registry -run 'Test.*(Model|Definition|Registry|Kimi|Gemini|Claude|Reasoning)' -count=1`
          - task_id: implement-client-catalog-builders
            goal: Adapt Codex and Claude client-specific catalog response builders by delegating model visibility, IDs, aliases, and display names to llmhub's registry.
            depends_on: reconcile-model-metadata
            touched_surfaces: [`internal/client/codex/models/`, `internal/client/claude/models/`, API model handlers]
            expected_output: Protocol-correct client catalogs with one registry source of truth.
            checks:
              - `go test ./internal/client/... ./sdk/api/handlers/openai ./sdk/api/handlers/claude -run 'Test.*(ClientModel|Catalog|DisplayName)' -count=1`
      - wave: configured-model-resolution
        tasks:
          - task_id: fix-codex-configured-only-resolution
            goal: Stop force-injecting built-in Codex model IDs when configured-only behavior applies while preserving aliases, prefix clones, OAuth forks, and credential validation.
            depends_on: implement-client-catalog-builders
            touched_surfaces: [`sdk/cliproxy/`, `sdk/cliproxy/auth/`, Codex model handlers/tests]
            expected_output: Configured models resolve deterministically and unconfigured built-ins stay hidden without breaking valid fallback.
            checks:
              - `go test ./sdk/cliproxy ./sdk/cliproxy/auth ./sdk/api/handlers/openai -run 'Test.*Codex.*(Model|Configured|Resolution|Credential|Alias)' -count=1`
          - task_id: verify-cross-protocol-presentation
            goal: Assert OpenAI, Codex, Claude, Gemini, and Gemini CLI catalogs preserve IDs, routing names, display names, visibility, and excluded-model behavior.
            depends_on: fix-codex-configured-only-resolution
            touched_surfaces: [model handler tests across `sdk/api/handlers/`, registry tests]
            expected_output: Cross-protocol presentation matrix with no routing/presentation conflation.
            checks:
              - `go test ./internal/registry ./sdk/api/handlers/... ./sdk/cliproxy -run 'Test.*(Model|DisplayName|Excluded|Catalog)' -count=1`
    phase_checks:
      - `go test ./internal/registry ./internal/client/... ./sdk/api/handlers/... ./sdk/cliproxy ./sdk/cliproxy/auth -count=1`
      - `go vet ./internal/registry ./internal/client/... ./sdk/api/handlers/... ./sdk/cliproxy ./sdk/cliproxy/auth`
      - `git diff --check`
    stop_conditions:
      - A model is exposed without executor support or display metadata changes routing/auth identity.
      - Configured-only behavior hides a valid explicitly configured alias or provider-prefixed clone.
    escalation: Remove only the affected catalog entry/exposure and retain validated resolution fixes.

  - phase_slug: codex-live-session-core
    story_id: 01KYTYK4GJ1CJFRMZCBB2PYDMC
    status: done
    goal: Implement Codex Live session, sideband, configuration, lifecycle, and service integration.
    depends_on: model-catalog-resolution-parity
    requirement_trace: [R3, R7, R10, R18, R21]
    allowed_surfaces:
      - additive `internal/client/codex/live/` session and sideband files
      - database-backed Codex Live setting domain/store
      - `internal/api/`, `sdk/cliproxy/`, and Codex executor lifecycle wiring
      - focused logging and configuration diff hooks
    avoided_surfaces:
      - WebRTC/TCP forwarding implementation, reserved for the next phase
      - upstream file/YAML runtime authority
      - management web controls
    waves:
      - wave: live-protocol-core
        tasks:
          - task_id: define-live-session-sideband
            goal: Implement session state, sideband message framing, request correlation, cancellation, deadlines, and terminal errors using deterministic in-memory transports.
            depends_on: none
            touched_surfaces: [`internal/client/codex/live/live.go`, `sideband.go`, focused tests]
            expected_output: Race-safe protocol core independent of concrete media transport.
            checks:
              - `go test ./internal/client/codex/live -run 'Test.*(Live|Session|Sideband|Cancel|Deadline)' -count=1`
              - `go test -race ./internal/client/codex/live -run 'Test.*(Session|Sideband)' -count=1`
          - task_id: define-live-database-config
            goal: Load validated Live enablement, limits, bind/target policy, and timeouts from revisioned Postgres settings with disabled defaults.
            depends_on: none
            touched_surfaces: [Codex Live domain/store files, service settings loader]
            expected_output: Database-only settings snapshot and reload contract.
            checks:
              - `go test ./internal/store ./sdk/cliproxy ./internal/client/codex/live -run 'Test.*CodexLive.*(Config|Settings|Reload|Default)' -count=1`
      - wave: live-service-integration
        tasks:
          - task_id: wire-live-service-lifecycle
            goal: Register Live routes/handlers, bind Codex executors, propagate config diffs, and stop sessions cleanly during reload/shutdown.
            depends_on: define-live-session-sideband
            touched_surfaces: [`internal/api/`, `sdk/cliproxy/service.go`, Codex executor binding, logging hooks]
            expected_output: Disabled-by-default lifecycle with authorized routing and no impact on ordinary Codex requests.
            checks:
              - `go test ./internal/api ./sdk/cliproxy ./internal/runtime/executor ./internal/client/codex/live -run 'Test.*CodexLive.*(Route|Lifecycle|Reload|Shutdown|Binding)' -count=1`
              - `go test -race ./sdk/cliproxy ./internal/client/codex/live -run 'Test.*CodexLive.*(Lifecycle|Shutdown)' -count=1`
    phase_checks:
      - `go test ./internal/client/codex/live ./internal/api ./sdk/cliproxy ./internal/runtime/executor ./internal/store -count=1`
      - `go vet ./internal/client/codex/live ./internal/api ./sdk/cliproxy ./internal/runtime/executor ./internal/store`
      - `git diff --check`
    stop_conditions:
      - Live affects non-Live Codex routing, leaks sessions, or accepts unbounded/unvalidated sideband input.
      - Configuration becomes file-authoritative.
    escalation: Keep Live disabled and retain only isolated protocol tests until lifecycle integration is safe.

  - phase_slug: codex-live-media-relay
    story_id: 01KYTYK4GR4MR6FEJGDX3392R2
    status: checked
    goal: Implement Codex Live WebRTC media and TCP relay with bounded resources, observability, and safe shutdown.
    depends_on: codex-live-session-core
    requirement_trace: [R7, R10, R18, R21]
    allowed_surfaces:
      - additive `internal/client/codex/live/media.go`, `tcp_proxy.go`, and tests
      - focused service/listener lifecycle wiring
      - request logging/redaction helpers
      - database settings fields already defined in the prior phase
    avoided_surfaces:
      - general-purpose unrestricted proxying
      - public bind by default
      - credential logging
      - management web controls
    waves:
      - wave: media-relay
        tasks:
          - task_id: implement-webrtc-media-relay
            goal: Relay negotiated realtime media with bounded packet/frame sizes, deadlines, cancellation, target allow policy, and per-session resource accounting.
            depends_on: none
            touched_surfaces: [`internal/client/codex/live/media.go`, test doubles/fixtures]
            expected_output: Deterministic loopback media relay and clean cancellation/timeout behavior.
            checks:
              - `go test ./internal/client/codex/live -run 'Test.*(Media|WebRTC|Relay|Packet|Cancel|Timeout)' -count=1`
              - `go test -race ./internal/client/codex/live -run 'Test.*(Media|Relay)' -count=1`
      - wave: tcp-relay-observability
        tasks:
          - task_id: implement-bounded-tcp-proxy
            goal: Add half-close-aware TCP forwarding with validated target, bind restrictions, connection limits, idle deadlines, byte accounting, and shutdown cancellation.
            depends_on: implement-webrtc-media-relay
            touched_surfaces: [`internal/client/codex/live/tcp_proxy.go`, listener lifecycle, tests]
            expected_output: Loopback-proven TCP relay that cannot become an unrestricted open proxy.
            checks:
              - `go test ./internal/client/codex/live -run 'Test.*TCP.*(Proxy|HalfClose|Limit|Timeout|Shutdown|Target)' -count=1`
              - `go test -race ./internal/client/codex/live -run 'Test.*TCP' -count=1`
          - task_id: add-live-relay-observability
            goal: Emit structured start/stop/failure and byte/session metrics without logging tokens, SDP secrets, or media payloads.
            depends_on: implement-bounded-tcp-proxy
            touched_surfaces: [`internal/logging/`, Codex Live relay lifecycle]
            expected_output: Redacted structured logs and failure reporting suitable for operations.
            checks:
              - `go test ./internal/logging ./internal/client/codex/live -run 'Test.*(Live|Media|TCP).*(Log|Redact|Failure)' -count=1`
      - wave: media-service-integration
        tasks:
          - task_id: wire-media-relay-lifecycle
            goal: Attach media/TCP relays to Live sessions and service shutdown/reload without leaking listeners or goroutines.
            depends_on: add-live-relay-observability
            touched_surfaces: [`sdk/cliproxy/`, `internal/api/`, `internal/client/codex/live/`]
            expected_output: Independent enable/disable controls and complete cleanup on reload/shutdown.
            checks:
              - `go test ./sdk/cliproxy ./internal/api ./internal/client/codex/live -run 'Test.*CodexLive.*(Media|TCP|Reload|Shutdown)' -count=1`
              - `go test -race ./sdk/cliproxy ./internal/client/codex/live -run 'Test.*(Media|TCP|Shutdown)' -count=1`
    phase_checks:
      - `go test ./internal/client/codex/live ./internal/logging ./internal/api ./sdk/cliproxy -count=1`
      - `go vet ./internal/client/codex/live ./internal/logging ./internal/api ./sdk/cliproxy`
      - `git diff --check`
    stop_conditions:
      - The relay can connect to arbitrary unvalidated targets, binds publicly by default, leaks secrets, or cannot terminate promptly.
    escalation: Ship no media/TCP enablement; preserve disabled protocol core and isolate the unsafe transport for plan refinement.

  - phase_slug: management-web-control-parity
    story_id: 01KYTYK4GYRNFA5SYBMSKKFX5E
    status: checked
    goal: Expose approved weights, cloaking, Home, and Codex Live controls through authorized management APIs and the current llmhub web UI/UX.
    depends_on: codex-live-media-relay
    requirement_trace: [R12, R16, R18, R19, R20, R21]
    allowed_surfaces:
      - `internal/api/handlers/management/`
      - management route registration and store interfaces
      - `web/src/services/api/`, `web/src/types/`, `web/src/pages/`, `web/src/features/`, `web/src/components/ui/`
      - `web/src/i18n/locales/en.json` and `vi.json`
      - existing design tokens in `web/src/index.css` only if an approved control cannot be expressed without an additive token
    avoided_surfaces:
      - new frontend test files
      - upstream web layout, branding, or component system
      - YAML/config editor as the persistence path for new controls
    waves:
      - wave: management-control-api
        tasks:
          - task_id: expose-runtime-control-api
            goal: Add authorized revisioned GET/PUT contracts for cloaking, Home, and Codex Live settings with redaction, validation, conflict handling, and hot-reload hooks.
            depends_on: none
            touched_surfaces: [`internal/api/handlers/management/`, API server route wiring, handler/store tests]
            expected_output: Additive database-backed management API with no secret disclosure and deterministic reload behavior.
            checks:
              - `go test ./internal/api/handlers/management ./internal/api ./internal/store ./sdk/cliproxy -run 'Test.*(RuntimeControl|CodexLive|Cloak|Home|Revision|Management)' -count=1`
          - task_id: finalize-auth-weight-api-contract
            goal: Confirm credential weight fields and identity filtering compose with existing auth-file DTOs, quota actions, and optimistic updates.
            depends_on: none
            touched_surfaces: [`internal/api/handlers/management/auth_files*`, focused API tests]
            expected_output: One coherent auth management contract used by the web client.
            checks:
              - `go test ./internal/api/handlers/management -run 'Test.*Auth.*(Weight|Identity|Filter|Quota)' -count=1`
      - wave: web-data-model
        tasks:
          - task_id: add-web-api-types-services
            goal: Add typed API clients and state adapters for runtime controls and credential weights using existing error/revision patterns.
            depends_on: expose-runtime-control-api
            touched_surfaces: [`web/src/services/api/`, `web/src/types/`, existing stores/hooks only when shared state is required]
            expected_output: Typed, localized error handling with no direct YAML writes.
            checks:
              - `cd web && bun run type-check`
              - `cd web && bun run lint`
      - wave: web-current-ux
        tasks:
          - task_id: add-auth-weight-control-ui
            goal: Add credential weight editing to the existing auth-file/provider resource flow using current cards, sheets, form controls, confirmation, loading, and error patterns.
            depends_on: add-web-api-types-services
            touched_surfaces: [`web/src/features/authFiles/`, relevant provider sheets, existing UI components, i18n]
            expected_output: Accessible validated weight editing without redesigning Auth Files UX.
            checks:
              - `cd web && bun run type-check`
              - `cd web && bun run lint`
              - `cd web && bun run build`
          - task_id: add-runtime-control-ui
            goal: Add cloaking, Home, and Codex Live settings to the existing System/Config information architecture using current typography, cards, switches, forms, save/conflict feedback, and responsive behavior.
            depends_on: add-web-api-types-services
            touched_surfaces: [`web/src/pages/SystemPage.tsx` or existing config surface, focused feature components, i18n]
            expected_output: Current-design operator controls with safe defaults, dependent-field disabling, and no upstream visual import.
            checks:
              - `cd web && bun run type-check`
              - `cd web && bun run lint`
              - `cd web && bun run build`
          - task_id: browser-verify-management-controls
            goal: Run the built app against a local database-backed server and verify desktop/mobile navigation, load/save/conflict/error states, disabled dependencies, weight editing, and persisted reload behavior.
            depends_on: add-auth-weight-control-ui
            touched_surfaces: [runtime evidence only]
            expected_output: Browser evidence for current UI/UX consistency and API persistence; no code changes unless a defect is found.
            checks:
              - `make build-web`
              - `make build`
              - Browser runtime check at desktop and mobile widths for Auth Files and System/Config control surfaces
    phase_checks:
      - `go test ./internal/api/handlers/management ./internal/api ./internal/store ./sdk/cliproxy -count=1`
      - `cd web && bun run type-check`
      - `cd web && bun run lint`
      - `cd web && bun run build`
      - `make build`
      - `git diff --check`
    stop_conditions:
      - A control writes YAML/files, exposes a secret, bypasses revision conflicts, or requires replacing existing UI patterns.
      - Browser verification shows broken responsive/navigation/loading/error behavior.
    escalation: Keep backend behavior at safe defaults and defer only the affected web control while preserving API compatibility.

  - phase_slug: upstream-checkpoint-final-gate
    story_id: 01KYTYK4H5G6W1Y7A7Z2KPF9VE
    status: planned
    goal: Run full integration verification, reconcile the current stable upstream release, and close the immutable checkpoint with complete evidence.
    depends_on: management-web-control-parity
    requirement_trace: [R1, R2, R4, R18, R20, R21, R22, R23]
    allowed_surfaces:
      - `docs/upstream/cliproxyapi-checkpoint.json`
      - append-only `Progress`, `Decisions`, and `Validation` entries in this plan
      - defects strictly required to pass an already-defined check, routed back to their owning phase surfaces
      - final Git checkpoint ref
    avoided_surfaces:
      - new product features
      - scope expansion without plan refinement
      - commit, push, PR, merge, release, or deployment
    waves:
      - wave: full-product-gate
        tasks:
          - task_id: run-full-go-gate
            goal: Run complete deterministic Go tests, race-focused high-risk packages, vet, build, and whitespace checks.
            depends_on: none
            touched_surfaces: [validation evidence only unless an owning-phase defect is found]
            expected_output: Clean Go product gate with exact command outputs recorded.
            checks:
              - `go test ./... -count=1`
              - `go test -race ./internal/home ./sdk/cliproxy/auth ./sdk/cliproxy/executionregistry ./internal/client/codex/live ./internal/runtime/executor ./sdk/api/handlers/openai -count=1`
              - `go vet ./...`
              - `go build ./...`
              - `git diff --check`
          - task_id: run-full-web-binary-gate
            goal: Run frontend type/lint/build, embed the panel, compile the final binary, and repeat browser smoke checks against the embedded management asset.
            depends_on: none
            touched_surfaces: [validation evidence only unless an owning-phase defect is found]
            expected_output: Production web and binary artifacts build successfully and changed controls work from the embedded panel.
            checks:
              - `cd web && bun run type-check`
              - `cd web && bun run lint`
              - `cd web && bun run build`
              - `make embed`
              - `make build`
              - Browser runtime smoke check against the embedded management panel
      - wave: checkpoint-refresh
        tasks:
          - task_id: refresh-final-upstream-checkpoint
            goal: Re-query the latest stable release, fetch its tag commit, update the checkpoint metadata and ledger, and classify every delta since the initial checkpoint.
            depends_on: run-full-go-gate
            touched_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`, final Git checkpoint ref, append-only plan validation]
            expected_output: Final checkpoint version/tag/SHA/time agree across GitHub, Git ref, JSON, and plan validation.
            checks:
              - `gh api repos/router-for-me/CLIProxyAPI/releases/latest --jq '{tag_name,published_at,html_url,target_commitish}'`
              - `python3 -m json.tool docs/upstream/cliproxyapi-checkpoint.json >/dev/null`
              - `git show-ref | rg 'refs/upstream-checkpoints/cliproxyapi/'`
          - task_id: reconcile-final-release-delta
            goal: For a changed stable release, prove every new commit is already covered, non-product/rejected, or blocked pending refinement; never silently declare parity over an unreviewed delta.
            depends_on: refresh-final-upstream-checkpoint
            touched_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`, plan Decisions/Validation]
            expected_output: Zero unclassified final-delta commits and an explicit completed checkpoint or refinement blocker.
            checks:
              - `python3 - <<'PY'
import json
p='docs/upstream/cliproxyapi-checkpoint.json'
d=json.load(open(p))
assert d['checkpoint']['tag']
assert d['checkpoint']['commit']
assert not [e for e in d['entries'] if not e.get('disposition')]
print(d['checkpoint']['tag'], d['checkpoint']['commit'])
PY`
      - wave: closure-consistency
        tasks:
          - task_id: verify-harness-plan-consistency
            goal: Confirm phase/story IDs, lifecycle states, evidence, checkpoint metadata, and next action are internally consistent before handing off to check.
            depends_on: reconcile-final-release-delta
            touched_surfaces: [append-only plan Validation and Current State lifecycle metadata]
            expected_output: One story per phase, no missing evidence, and no second initiative markdown.
            checks:
              - `zharness query phases --json`
              - `find docs/plans/active -maxdepth 1 -type f -name '*cliproxyapi*parity*.md' -print`
              - `git status --short`
    phase_checks:
      - `go test ./... -count=1`
      - `go vet ./...`
      - `go build ./...`
      - `cd web && bun run type-check && bun run lint && bun run build`
      - `make build`
      - `git diff --check`
      - `zharness query phases --json`
    stop_conditions:
      - Any required test/build/lint/browser check fails.
      - Latest stable release metadata, Git ref, JSON checkpoint, and plan evidence disagree.
      - A final release delta contains product behavior outside the locked requirements.
    escalation: Route defects to the owning phase; route new product scope to `brainstorm refine`; do not mark the initiative complete or publish repository state.

- first_executable_phase: upstream-parity-ledger
- execution_handoff: `work full` starts `upstream-parity-ledger` and may not begin production-code phases until its checkpoint and dispositions are verified.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- timestamp: 2026-07-31T02:14:54Z
  phase: upstream-parity-ledger
  wave: phase-start
  task: phase-start
  task_status: in-progress
  run_id: 01KYTZ7RXA43KZVEZ1GNQJX2T6
  trace_id: none
  changed_surfaces: [`docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `zharness run create --slug upstream-parity-ledger --plan-id 01KYTYDKMCXHN0TQAPH1RM5K71 --json` -> created run `01KYTZ7RXA43KZVEZ1GNQJX2T6`
  blocker: none
- timestamp: 2026-07-31T02:21:55Z
  phase: upstream-parity-ledger
  wave: ledger-source-freeze
  task: ledger-refresh-source
  task_status: DONE
  run_id: 01KYTZ7RXA43KZVEZ1GNQJX2T6
  trace_id: 01KYTZMH4843YFMA0FZHFDE2HW
  changed_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`, `refs/upstream-checkpoints/cliproxyapi/v7.2.93`, `refs/upstream-checkpoints/cliproxyapi/v7.2.96`, `refs/upstream-checkpoints/cliproxyapi/v7.2.111`]
  verification: `gh api repos/router-for-me/CLIProxyAPI/releases/latest --jq '{tag_name,published_at,html_url,target_commitish}'` -> `v7.2.111`, published `2026-07-30T18:56:58Z`; `git show-ref refs/upstream-checkpoints/cliproxyapi/v7.2.93 refs/upstream-checkpoints/cliproxyapi/v7.2.96 refs/upstream-checkpoints/cliproxyapi/v7.2.111` -> all three immutable refs present; `python3 -m json.tool docs/upstream/cliproxyapi-checkpoint.json >/dev/null` -> pass
  blocker: none
- timestamp: 2026-07-31T02:21:55Z
  phase: upstream-parity-ledger
  wave: ledger-source-freeze
  task: ledger-commit-inventory
  task_status: DONE
  run_id: 01KYTZ7RXA43KZVEZ1GNQJX2T6
  trace_id: 01KYTZMH4843YFMA0FZHFDE2HW
  changed_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`]
  verification: `git log --reverse --no-merges --format='%H%x09%s' refs/upstream-checkpoints/cliproxyapi/v7.2.93..refs/upstream-checkpoints/cliproxyapi/v7.2.111` -> 102 ordered commits; `git diff --name-status refs/upstream-checkpoints/cliproxyapi/v7.2.93..refs/upstream-checkpoints/cliproxyapi/v7.2.111` -> 411 touched paths; ledger release membership totals 102 with zero duplicate SHAs
  blocker: none
- timestamp: 2026-07-31T02:21:55Z
  phase: upstream-parity-ledger
  wave: ledger-local-disposition
  task: ledger-semantic-disposition
  task_status: DONE
  run_id: 01KYTZ7RXA43KZVEZ1GNQJX2T6
  trace_id: 01KYTZMNZBGWZ9TW1BKC6GCCK0
  changed_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`]
  verification: `python3 -m json.tool docs/upstream/cliproxyapi-checkpoint.json >/dev/null` plus ledger invariant assertions -> pass; 102 entries classified as 78 `adapt`, 4 `already-present`, 2 `superseded-local`, and 18 `reject`; every non-rejected entry has an owner phase and every entry has paths, evidence, and rationale
  blocker: none
- timestamp: 2026-07-31T02:24:40Z
  phase: upstream-parity-ledger
  wave: ledger-local-disposition
  task: ledger-semantic-disposition
  task_status: DONE
  run_id: 01KYTZ7RXA43KZVEZ1GNQJX2T6
  trace_id: 01KYTZSQ4WZENCCA39ZY27D9Q2
  changed_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`]
  verification: full manual classification review plus corrected semantic ledger assertions -> pass; 102 entries classified as 80 `adapt`, 4 `already-present`, 2 `superseded-local`, and 16 `reject`; all five plugin-only entries are rejected, while weighted auth scheduling and custom-executor preservation remain targeted adaptations
  concern: Supersedes the earlier wave-2 count after correcting primary-behavior ownership for commits that also touched incidental plugin paths.
  blocker: none
- timestamp: 2026-07-31T02:59:56Z
  phase: postgres-control-foundation
  wave: phase-start
  task: phase-start
  task_status: in-progress
  run_id: 01KYV1T9VE29TR5Q43G1AJE32Y
  trace_id: none
  changed_surfaces: [`docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `zharness run create --slug postgres-control-foundation --plan-id 01KYTYDKMCXHN0TQAPH1RM5K71 --json` -> created run `01KYV1T9VE29TR5Q43G1AJE32Y`
  blocker: none
- timestamp: 2026-07-31T03:24:38Z
  phase: postgres-control-foundation
  wave: postgres-schema-domain
  task: define-runtime-control-domain
  task_status: DONE
  run_id: 01KYV1T9VE29TR5Q43G1AJE32Y
  trace_id: 01KYV37NSMJ102SBAP5A9CMS30
  changed_surfaces: [`internal/runtimecontrol/types.go`, `internal/runtimecontrol/types_test.go`, `sdk/cliproxy/auth/cooldown_state.go`, `sdk/cliproxy/auth/cooldown_state_test.go`]
  verification: `go test ./internal/config/... -count=1` -> pass; focused `go test ./internal/runtimecontrol ./sdk/cliproxy/auth ... -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T03:24:38Z
  phase: postgres-control-foundation
  wave: postgres-schema-domain
  task: add-idempotent-postgres-schema
  task_status: DONE
  run_id: 01KYV1T9VE29TR5Q43G1AJE32Y
  trace_id: 01KYV37NSMJ102SBAP5A9CMS30
  changed_surfaces: [`internal/store/postgresstore.go`, `internal/store/postgres_runtime_controls.go`, `internal/store/postgres_runtime_controls_integration_test.go`, `internal/store/postgres_cooldown_store.go`, `internal/store/postgres_cooldown_store_test.go`, `internal/store/postgres_cooldown_store_integration_test.go`]
  verification: `go test ./internal/store -run 'TestPostgresStore.*(Init|Schema|Settings|Cooldown)' -count=1` -> pass; `LLMHUB_POSTGRES_TEST_DSN="$LLMHUB_POSTGRES_TEST_DSN" go test ./internal/store -run 'TestPostgres.*(Settings|Cooldown|Migration|Restart)' -count=1` -> pass with env-gated integration tests skipped when DSN is unset
  blocker: none
- timestamp: 2026-07-31T03:24:38Z
  phase: postgres-control-foundation
  wave: postgres-repository-contracts
  task: implement-revisioned-settings-store
  task_status: DONE
  run_id: 01KYV1T9VE29TR5Q43G1AJE32Y
  trace_id: 01KYV37NSX513R0ZD2TEECC5TY
  changed_surfaces: [`internal/runtimecontrol/types.go`, `internal/store/postgres_runtime_controls.go`, `internal/store/postgres_runtime_controls_integration_test.go`, `internal/store/postgresstore.go`]
  verification: `go test ./internal/store ./sdk/cliproxy -run 'Test.*(RuntimeSettings|Revision|Restart)' -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T03:24:38Z
  phase: postgres-control-foundation
  wave: postgres-repository-contracts
  task: implement-cooldown-snapshot-store
  task_status: DONE
  run_id: 01KYV1T9VE29TR5Q43G1AJE32Y
  trace_id: 01KYV37NSX513R0ZD2TEECC5TY
  changed_surfaces: [`sdk/cliproxy/auth/cooldown_state.go`, `sdk/cliproxy/auth/cooldown_state_test.go`, `internal/store/postgres_cooldown_store.go`, `internal/store/postgres_cooldown_store_test.go`, `internal/store/postgres_cooldown_store_integration_test.go`, `internal/store/postgresstore.go`]
  verification: `go test ./internal/store ./sdk/cliproxy/auth -run 'Test.*Cooldown.*(Persist|Restore|Expiry|Concurrent)' -count=1` -> pass; phase checks `go test ./internal/config/... ./internal/store/... ./sdk/cliproxy/... -count=1`, `go vet ./internal/config/... ./internal/store/... ./sdk/cliproxy/...`, and `git diff --check` -> pass
  blocker: none
- timestamp: 2026-07-31T03:47:00Z
  phase: postgres-control-foundation
  wave: phase-check
  task: full-phase-gate
  task_status: CHECKED
  run_id: 01KYV1T9VE29TR5Q43G1AJE32Y
  trace_id: none
  changed_surfaces: [`internal/runtimecontrol/`, `sdk/cliproxy/auth/cooldown_state.go`, `internal/store/postgresstore.go`, `internal/store/postgres_runtime_controls.go`, `internal/store/postgres_cooldown_store.go`, focused tests]
  verification: `gofmt -w internal/runtimecontrol/types.go internal/store/postgres_runtime_controls_integration_test.go && go test ./internal/runtimecontrol ./internal/store ./sdk/cliproxy/auth -count=1 && go test ./...` -> pass; `go vet ./internal/config/... ./internal/store/... ./sdk/cliproxy/...` -> pass; `git diff --check` -> pass; `zharness check record --verdict APPROVED --judge same-session --judge-model claude-opus-5 --run-id 01KYV1T9VE29TR5Q43G1AJE32Y ... --json` -> check `01KYV3X3GFGC008ZC8H0TPNG2P`
  blocker: none
- timestamp: 2026-07-31T03:51:00Z
  phase: auth-weight-management-parity
  wave: phase-start
  task: phase-start
  task_status: in-progress
  run_id: 01KYV433N09V7MVKTSJCQE5FEY
  trace_id: none
  changed_surfaces: [`docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `zharness run create --slug auth-weight-management-parity --plan-id 01KYTYDKMCXHN0TQAPH1RM5K71 --json` -> created run `01KYV433N09V7MVKTSJCQE5FEY`
  blocker: none
- timestamp: 2026-07-31T04:08:00Z
  phase: auth-weight-management-parity
  wave: auth-identity-weight
  task: port-auth-identity-filtering
  task_status: DONE
  run_id: 01KYV433N09V7MVKTSJCQE5FEY
  trace_id: 01KYV4RG9VQAFR5B7CEGSYN2ZN
  changed_surfaces: [`internal/api/handlers/management/auth_files.go`, `internal/api/handlers/management/auth_files_identity_test.go`]
  verification: `go test ./internal/api/handlers/management -run 'Test.*Auth.*(Filter|Index|Identity)' -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T04:08:00Z
  phase: auth-weight-management-parity
  wave: auth-identity-weight
  task: implement-weight-validation-ingestion
  task_status: DONE
  run_id: 01KYV433N09V7MVKTSJCQE5FEY
  trace_id: 01KYV4RG9VQAFR5B7CEGSYN2ZN
  changed_surfaces: [`sdk/cliproxy/auth/credential_weight.go`, `sdk/cliproxy/auth/credential_weight_test.go`, `internal/api/handlers/management/auth_files.go`, `internal/api/handlers/management/auth_files_patch_fields_test.go`, `internal/api/handlers/management/auth_files_postgres_test.go`]
  verification: `go test ./sdk/cliproxy/auth ./internal/api/handlers/management ./internal/store -run 'Test.*Weight' -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T04:19:00Z
  phase: auth-weight-management-parity
  wave: scheduler-cooldown
  task: implement-weighted-scheduler
  task_status: DONE
  run_id: 01KYV433N09V7MVKTSJCQE5FEY
  trace_id: 01KYV6F6G3S5QFBQHFETD9BVJF
  changed_surfaces: [`sdk/cliproxy/auth/scheduler.go`, `sdk/cliproxy/auth/scheduler_test.go`]
  verification: `go test ./sdk/cliproxy/auth -run 'Test.*(Weighted|Weight|Scheduler|PickNext)' -count=1` -> pass; `go test -race ./sdk/cliproxy/auth -run 'Test.*(Weighted|Scheduler|Cooldown)' -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T04:19:00Z
  phase: auth-weight-management-parity
  wave: scheduler-cooldown
  task: wire-cooldown-restart-state
  task_status: DONE
  run_id: 01KYV433N09V7MVKTSJCQE5FEY
  trace_id: 01KYV6F6G3S5QFBQHFETD9BVJF
  changed_surfaces: [`sdk/cliproxy/auth/conductor.go`, `sdk/cliproxy/auth/cooldown_persistence_test.go`, `sdk/cliproxy/builder.go`, `sdk/cliproxy/service.go`]
  verification: `go test ./sdk/cliproxy/auth ./sdk/cliproxy ./internal/store -run 'Test.*Cooldown.*(Restart|Persist|Availability)|TestManagerCooldownState' -count=1` -> pass; `go test -race ./sdk/cliproxy/auth -run 'TestManagerCooldownState|Test.*Cooldown.*Availability' -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T04:23:00Z
  phase: auth-weight-management-parity
  wave: auth-management-api
  task: expose-auth-weight-management
  task_status: DONE
  run_id: 01KYV433N09V7MVKTSJCQE5FEY
  trace_id: 01KYV6J9PCAQ7RC18Z5WJ1M5G5
  changed_surfaces: [`internal/api/handlers/management/auth_files_identity_test.go`]
  verification: `go test ./internal/api/handlers/management ./internal/api -run 'Test.*Auth.*(Weight|Filter|Management)' -count=1` -> pass; existing management endpoints already expose stable ID/auth_index selectors and weight list/update/upload contracts, with a test-only delete fixture corrected to respect safe name validation
  blocker: none
- timestamp: 2026-07-31T04:27:00Z
  phase: home-runtime-full-parity
  wave: phase-start
  task: phase-start
  task_status: in-progress
  run_id: 01KYV6T0F7EW0X1JAXFPG7YF3E
  trace_id: none
  changed_surfaces: [`docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `zharness run create --slug home-runtime-full-parity --plan-id 01KYTYDKMCXHN0TQAPH1RM5K71 --json` -> created run `01KYV6T0F7EW0X1JAXFPG7YF3E`
  blocker: none
- timestamp: 2026-07-31T05:19:41Z
  phase: home-runtime-full-parity
  wave: home-protocol-state
  task: port-membership-discovery-takeover
  task_status: DONE
  run_id: 01KYV6T0F7EW0X1JAXFPG7YF3E
  trace_id: 01KYV7VTS8ASC548NQCZJ2AQA5
  changed_surfaces: [`internal/home/client.go`, `internal/home/client_test.go`, `sdk/cliproxy/auth/home_*`]
  verification: `go test ./internal/home -run 'TestMembership|TestSubscriptionParameters|TestNewLifetime|TestConcurrencyReleaseDoesNotOpen|TestAmbiguousDispatchSuppresses' -count=1` -> pass; protocol-one membership args, native session IDs, alias canonicalization, takeover eligibility, legacy downgrade, and release gating covered
  blocker: none
- timestamp: 2026-07-31T05:19:41Z
  phase: home-runtime-full-parity
  wave: home-protocol-state
  task: port-home-cas-replay
  task_status: DONE
  run_id: 01KYV6T0F7EW0X1JAXFPG7YF3E
  trace_id: 01KYV91D0A6A11HP3XB0B8ZK52
  changed_surfaces: [`internal/home/client.go`, `internal/home/global.go`, `internal/cache/antigravity_reasoning_replay_cache.go`, `internal/runtime/executor/antigravity_reasoning_replay.go`, `internal/runtime/executor/antigravity_executor.go`]
  verification: `go test ./internal/home ./internal/cache ./internal/runtime/executor -count=1` -> pass; Home CAS, stale-writer rejection, replay tombstones, and Antigravity replay request/response integration covered
  blocker: none
- timestamp: 2026-07-31T05:19:41Z
  phase: home-runtime-full-parity
  wave: home-dispatch-recovery
  task: port-dispatch-reconnect-state
  task_status: DONE
  run_id: 01KYV6T0F7EW0X1JAXFPG7YF3E
  trace_id: 01KYV9GZZBNPR9Z9AX3AVQ2SEP
  changed_surfaces: [`internal/home/client.go`, `internal/home/client_test.go`]
  verification: `go test ./internal/home ./sdk/cliproxy/auth -count=1` -> pass; ambiguous dispatch marker, fenced client reuse, async Redis client/socket detach-close, cluster discovery transport propagation, failover state, and takeover-safe lifetime recovery covered
  blocker: none
- timestamp: 2026-07-31T05:19:41Z
  phase: home-runtime-full-parity
  wave: home-service-lifecycle
  task: integrate-home-full-lifecycle
  task_status: DONE
  run_id: 01KYV6T0F7EW0X1JAXFPG7YF3E
  trace_id: 01KYV9R9CFBV9NFAT0NM8SD8HG
  changed_surfaces: [`sdk/cliproxy/service.go`, `sdk/cliproxy/service_home_lifecycle_test.go`, `sdk/config/config.go`]
  verification: `go test ./internal/home ./sdk/config ./sdk/cliproxy ./sdk/cliproxy/auth -count=1` -> pass; Home enablement reload, monotonic dispatch generations, disabled lifetime clearing, release flushing, and SDK-visible Home config covered
  blocker: none

- timestamp: 2026-07-31T05:22:00Z
  phase: translator-protocol-parity-111
  wave: phase-start
  task: phase-start
  task_status: in-progress
  run_id: 01KYV9XPGD0QKRVJD53EF0F78Q
  trace_id: none
  changed_surfaces: [`docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `zharness run create --slug translator-protocol-parity-111 --plan-id 01KYTYDKMCXHN0TQAPH1RM5K71 --json` -> created run `01KYV9XPGD0QKRVJD53EF0F78Q`
  blocker: none
- timestamp: 2026-07-31T05:31:00Z
  phase: translator-protocol-parity-111
  wave: translator-request-fidelity
  task: port-tool-schema-and-output-inputs
  task_status: DONE
  run_id: 01KYV9XPGD0QKRVJD53EF0F78Q
  trace_id: 01KYVA7VE17BCKVQ7FCWRRBPAT
  changed_surfaces: [`internal/util/claude_schema.go`, `internal/util/claude_schema_test.go`, `internal/translator/claude/openai/chat-completions/claude_openai_request.go`, `internal/translator/claude/openai/chat-completions/claude_openai_request_test.go`, `internal/translator/gemini/openai/chat-completions/gemini_openai_request.go`, `internal/translator/gemini/openai/chat-completions/gemini_openai_request_test.go`]
  verification: `go test ./internal/translator/... ./internal/util/... -run 'Test.*(Schema|InputImage|ToolResult|Structured|ResponseFormat)' -count=1` -> pass; `go test ./internal/translator/openai/... ./internal/runtime/executor/helps -run 'Test.*(Tool|MultiAgent|Translate)' -count=1 && go test ./internal/translator/openai/... -run '^$' -bench 'Benchmark.*Tool' -benchmem` -> pass, benchmark `BenchmarkConvertOpenAIResponsesRequestWithLargeNonConvertibleToolArray-20` 129838 ns/op, 25616 B/op, 23 allocs/op
  blocker: none
- timestamp: 2026-07-31T05:31:00Z
  phase: translator-protocol-parity-111
  wave: translator-request-fidelity
  task: reconcile-openai-tool-performance
  task_status: DONE
  run_id: 01KYV9XPGD0QKRVJD53EF0F78Q
  trace_id: 01KYVA7VE17BCKVQ7FCWRRBPAT
  changed_surfaces: [`internal/translator/openai/openai/responses/openai_openai-responses_request.go`, `internal/runtime/executor/helps/codex_multi_agent_v2.go`]
  verification: no code change; ledger entry was already-present, verified by `go test ./internal/translator/openai/... ./internal/runtime/executor/helps -run 'Test.*(Tool|MultiAgent|Translate)' -count=1` and OpenAI translator tool benchmark
  blocker: none
- timestamp: 2026-07-31T05:45:00Z
  phase: translator-protocol-parity-111
  wave: translator-streaming-responses
  task: port-streaming-response-parity
  task_status: DONE
  run_id: 01KYV9XPGD0QKRVJD53EF0F78Q
  trace_id: 01KYVAXHGXMD0JGVDEJT5QJ528
  changed_surfaces: [`internal/translator/codex/openai/chat-completions/codex_openai_response.go`, `internal/translator/codex/openai/chat-completions/codex_openai_response_test.go`, `internal/translator/gemini/openai/responses/gemini_openai-responses_response.go`, `internal/translator/gemini/openai/responses/gemini_openai-responses_response_test.go`]
  verification: `go test ./internal/translator/... -run 'Test.*(Done|Completion|ToolCall|FunctionCall|OutputIndex|Delta|Incomplete|Custom)' -count=1 && git diff --check` -> pass after gofmt
  blocker: none
- timestamp: 2026-07-31T05:55:00Z
  phase: translator-protocol-parity-111
  wave: translator-usage-normalization
  task: normalize-translator-usage
  task_status: DONE
  run_id: 01KYV9XPGD0QKRVJD53EF0F78Q
  trace_id: 01KYVB18QF55HVV41631H7EA3Z
  changed_surfaces: [`internal/translator/codex/openai/chat-completions/codex_openai_response.go`, `internal/translator/codex/openai/chat-completions/codex_openai_response_test.go`, `internal/translator/openai/claude/openai_claude_response.go`, `internal/translator/openai/claude/openai_claude_response_test.go`]
  verification: `go test ./internal/translator/... -run 'Test.*(Usage|Token|Cache|Cached|MessageDelta|Completion)' -count=1 && git diff --check` -> pass after gofmt
  blocker: none
- timestamp: 2026-07-31T06:00:00Z
  phase: translator-protocol-parity-111
  wave: translator-response-formats
  task: fix-translator-response-formats
  task_status: DONE
  run_id: 01KYV9XPGD0QKRVJD53EF0F78Q
  trace_id: 01KYVB6GWDKT2FMEYZWQB029E4
  changed_surfaces: [`internal/translator/gemini/openai/chat-completions/gemini_openai_request.go`, `internal/translator/gemini/openai/chat-completions/gemini_openai_request_test.go`]
  verification: `go test ./internal/translator/... -run 'Test.*(ResponseFormat|JSONSchema|JSONObject|Structured|Format)' -count=1 && git diff --check` -> pass after gofmt
  blocker: none

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- timestamp: 2026-07-31T02:24:40Z
  phase: upstream-parity-ledger
  task: ledger-semantic-disposition
  decision: Classify commits by their primary portable product behavior before incidental touched paths; reject plugin-only behavior, but retain weighted auth scheduling and custom-executor preservation as targeted adaptations.
  rationale: Path-only plugin classification would have incorrectly discarded two approved llmhub behaviors because the upstream commits also updated pluginhost integration.

- timestamp: 2026-07-31T06:31:00Z
  phase: antigravity-signature-parity
  wave: signature-validation
  task: implement-provider-signature-validation
  task_status: DONE
  run_id: 01KYVBA78TVMFE6V43NFFEMP9X
  trace_id: 01KYVBRY94P72702NMVFXCNBVG
  changed_surfaces: [`internal/signature/`, `internal/runtime/executor/antigravity_reasoning_replay.go`, `internal/runtime/executor/antigravity_executor_signature_test.go`]
  verification: `go test ./internal/signature ./internal/runtime/executor -run 'TestValidateGeminiFunctionCallPairing|TestPrepareAntigravityGeminiReasoningReplayPayloadRejectsBadFunctionPairing|TestAntigravityExecutor_.*Bypass|TestAntigravityExecutor_CacheMode' -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T06:39:00Z
  phase: antigravity-signature-parity
  wave: signature-validation
  task: scope-antigravity-schema-cleaning
  task_status: DONE
  run_id: 01KYVBA78TVMFE6V43NFFEMP9X
  trace_id: 01KYVBZ006Z1K8PAY139K8VQVJ
  changed_surfaces: [`internal/runtime/executor/antigravity_executor.go`, `internal/runtime/executor/antigravity_executor_buildrequest_test.go`, `internal/util/gemini_schema.go`]
  verification: `go test ./internal/runtime/executor ./internal/util -run 'TestAntigravityBuildRequest.*(Schema|Schemas|Response|History|JSONObject|Generation)|TestCleanJSONSchemaForAntigravity|TestCleanJSONSchemaForGemini' -count=1` -> pass after scoping schema cleaning to declaration/generation schema paths and adding response-schema behavior
  blocker: none
- timestamp: 2026-07-31T06:42:00Z
  phase: antigravity-signature-parity
  wave: phase-gate
  task: validate-antigravity-signature-parity
  task_status: DONE
  run_id: 01KYVBA78TVMFE6V43NFFEMP9X
  trace_id: none
  changed_surfaces: [`internal/signature/`, `internal/runtime/executor/`, `internal/util/`, `internal/cache/`]
  verification: `go test ./internal/signature ./internal/runtime/executor ./internal/util ./internal/cache -run 'Test.*(Antigravity|GeminiFunctionCallPairing|GeminiThought|Signature|Schema|Replay)' -count=1 && go vet ./internal/signature ./internal/runtime/executor ./internal/util ./internal/cache && git diff --check` -> pass; `zharness check record ... --run-id 01KYVBA78TVMFE6V43NFFEMP9X --json` -> check `01KYVBZKTZEPZ0ST02SBY0E2ZS`
  blocker: none
- timestamp: 2026-07-31T06:14:33Z
  phase: executor-websocket-runtime-parity
  wave: executor-binding-cloaking
  task: port-derived-session-runtime-reliability
  task_status: DONE
  run_id: 01KYVC25QYPE3PHA3JNCTCZRSB
  trace_id: 01KYVCYGRCWJ179BHMS0AEWZE1
  changed_surfaces: [`sdk/cliproxy/executor/`, `internal/runtime/executor/helps/`, `internal/runtime/executor/xai_executor.go`, `internal/runtime/executor/antigravity_reasoning_replay.go`]
  verification: `go test ./sdk/cliproxy/executor ./internal/runtime/executor/helps ./internal/runtime/executor -run 'TestBindExecutionResource|TestCodexWebsocketSessionBindsSameLifecycleAndConnectionOnce|TestDerivedSessionProviderMappings|TestProviderSessionUUIDPrefersExecutionSession|TestXAIExecutionSessionIDUsesDerivedFallback|TestAntigravityReasoningReplayClientSessionKeyUsesDerivedMetadata' -count=1` -> pass
  blocker: none
- timestamp: 2026-07-31T06:14:33Z
  phase: executor-websocket-runtime-parity
  wave: websocket-continuity
  task: port-websocket-continuity-cache-control-ordering
  task_status: DONE
  run_id: 01KYVC25QYPE3PHA3JNCTCZRSB
  trace_id: 01KYVCYGTA4PP50GFHEKQ1F3AJ
  changed_surfaces: [`internal/runtime/executor/claude_executor.go`, `internal/runtime/executor/claude_executor_test.go`, `sdk/api/handlers/openai/openai_responses_websocket*`]
  verification: `go test ./internal/runtime/executor ./sdk/api/handlers/openai -run 'TestPrependToFirstUserMessage|TestInjectToolsCacheControlSkipsDeferredTools|TestResponsesWebSocket1009ToolCacheRollbackIsTransactional|TestRepairResponsesWebsocket|TestResponsesWebsocket.*Rollback' -count=1` -> pass; local WebSocket transaction rollback was already present and validated
  blocker: none
- timestamp: 2026-07-31T06:14:33Z
  phase: executor-websocket-runtime-parity
  wave: phase-validation
  task: validate-executor-websocket-runtime-parity
  task_status: DONE
  run_id: 01KYVC25QYPE3PHA3JNCTCZRSB
  trace_id: none
  changed_surfaces: [`sdk/cliproxy/executor/`, `internal/runtime/executor/`, `internal/runtime/executor/helps/`, `sdk/api/handlers/openai/`]
  verification: `go test ./sdk/cliproxy/executor ./internal/runtime/executor/helps ./internal/runtime/executor ./sdk/api/handlers/openai -run 'TestBindExecutionResource|TestCodexWebsocketSessionBindsSameLifecycleAndConnectionOnce|TestDerivedSessionProviderMappings|TestProviderSessionUUIDPrefersExecutionSession|TestDerivedSessionProviderMappingsRequireIdentity|TestXAIExecutionSessionIDUsesDerivedFallback|TestAntigravityReasoningReplayClientSessionKeyUsesDerivedMetadata|TestPrependToFirstUserMessage|TestInjectToolsCacheControlSkipsDeferredTools|TestResponsesWebSocket1009ToolCacheRollbackIsTransactional|TestRepairResponsesWebsocket|TestResponsesWebsocket.*Rollback' -count=1 && go vet ./sdk/cliproxy/executor ./internal/runtime/executor/helps ./internal/runtime/executor ./sdk/api/handlers/openai && git diff --check` -> pass; `zharness check record ... --run-id 01KYVC25QYPE3PHA3JNCTCZRSB --json` -> check `01KYVCYGW3K2CX3ADHSVW0RNTK`
  blocker: none
- timestamp: 2026-07-31T07:05:00Z
  phase: model-catalog-resolution-parity
  wave: model-data-catalogs
  task: reconcile-model-metadata
  task_status: DONE
  run_id: 01KYVD3482T1X8944ZEXBJ6SEP
  trace_id: 01KYVDN8QM013AW7BZ9KXPTPVM
  changed_surfaces: [`internal/registry/models/models.json`, `internal/registry/model_definitions_test.go`]
  verification: `go test ./internal/registry ./sdk/api/handlers/openai -run 'TestUpstreamCheckpointModelCatalogAdditions|TestLocalKiroModelDefinitionsRemainAvailable|TestCodexClientModelsIncludesCheckpointTemplates|TestCodexClientModelsUsesConfiguredDisplayNameForTemplate' -count=1` -> pass; pinned upstream missing-ID comparison returned no remaining missing model IDs
  blocker: none
- timestamp: 2026-07-31T07:08:00Z
  phase: model-catalog-resolution-parity
  wave: model-data-catalogs
  task: implement-client-catalog-builders
  task_status: DONE
  run_id: 01KYVD3482T1X8944ZEXBJ6SEP
  trace_id: 01KYVDN8QM013AW7BZ9KXPTPVM
  changed_surfaces: [`internal/registry/models/codex_client_models.json`, `sdk/api/handlers/openai/model_display_name_test.go`]
  verification: Existing llmhub Codex client catalog builder preserved registry delegation; pinned upstream Codex templates for `gpt-5.6-*` and `gpt-5.3-codex-spark` render through `buildCodexClientModels` with explicit template display names and list visibility
  blocker: none
- timestamp: 2026-07-31T07:10:00Z
  phase: model-catalog-resolution-parity
  wave: phase-validation
  task: validate-model-catalog-resolution-parity
  task_status: DONE
  run_id: 01KYVD3482T1X8944ZEXBJ6SEP
  trace_id: 01KYVDN8QWVF5ZHQSV7M40VKRJ
  changed_surfaces: [`internal/registry/`, `internal/registry/models/`, `sdk/api/handlers/openai/`]
  verification: `go test ./internal/registry ./sdk/api/handlers/openai -count=1 && go vet ./internal/registry ./sdk/api/handlers/openai && git diff --check` -> pass; `zharness check record ... --run-id 01KYVD3482T1X8944ZEXBJ6SEP --json` -> check `01KYVDMY4W9Z1CW83TDE3AK64T`
  blocker: none
- timestamp: 2026-07-31T07:34:00Z
  phase: codex-live-session-core
  wave: live-protocol-core
  task: define-live-session-sideband
  task_status: DONE
  run_id: 01KYVDT2JBWRG7YBTSRKCHH0XP
  trace_id: 01KYVEKFQTQPMF4XR29E22BPF8
  changed_surfaces: [`internal/client/codex/live/session.go`, `internal/client/codex/live/protocol.go`, `internal/client/codex/live/session_test.go`, `internal/client/codex/live/protocol_test.go`]
  verification: `go test ./internal/client/codex/live -count=1` -> pass; session store, strict call ID validation, request body preparation, protocol/header allowlists, sideband URL construction, and body-size rejection covered
  blocker: none
- timestamp: 2026-07-31T07:34:00Z
  phase: codex-live-session-core
  wave: live-service-integration
  task: wire-live-service-lifecycle
  task_status: DONE
  run_id: 01KYVDT2JBWRG7YBTSRKCHH0XP
  trace_id: 01KYVEKFR35Q8F21BBDB9M3068
  changed_surfaces: [`internal/api/handlers/codexlive/handler.go`, `internal/api/handlers/codexlive/handler_test.go`, `internal/api/server.go`, `sdk/cliproxy/builder.go`, `sdk/cliproxy/service.go`]
  verification: `go test ./internal/api/handlers/codexlive ./internal/api ./sdk/cliproxy ./internal/client/codex/live -count=1` -> pass; Codex Live routes are registered only with a runtime settings store, disabled settings return 404, create-call forwards through prepared Codex auth and stores sessions, sideband remains a 501 placeholder for the media-relay phase
  blocker: none
- timestamp: 2026-07-31T07:34:00Z
  phase: codex-live-session-core
  wave: phase-check
  task: full-phase-gate
  task_status: CHECKED
  run_id: 01KYVDT2JBWRG7YBTSRKCHH0XP
  trace_id: none
  changed_surfaces: [`internal/client/codex/live/`, `internal/api/handlers/codexlive/`, `internal/api/server.go`, `sdk/cliproxy/builder.go`, `sdk/cliproxy/service.go`]
  verification: focused tests, focused vet, and `git diff --check` -> pass; `zharness check record --verdict APPROVED --judge same-session --judge-model gpt-5.5 --run-id 01KYVDT2JBWRG7YBTSRKCHH0XP ... --json` -> check `01KYVEMTD5JSTJWS3J5VFY5N12`
  blocker: none

- timestamp: 2026-07-31T08:32:00Z
  phase: codex-live-media-relay
  wave: phase-start
  task: phase-start
  task_status: in-progress
  run_id: 01KYVMT19Z4ZNQADEE2RDWHP7V
  trace_id: none
  changed_surfaces: [`docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `zharness run create --slug codex-live-media-relay --plan-id 01KYTYDKMCXHN0TQAPH1RM5K71 --json` -> created run `01KYVMT19Z4ZNQADEE2RDWHP7V`
  blocker: none
- timestamp: 2026-07-31T09:00:01Z
  phase: codex-live-media-relay
  wave: media-relay + tcp-relay-observability + media-service-integration
  task: implement-webrtc-media-relay, implement-bounded-tcp-proxy, add-live-relay-observability, wire-media-relay-lifecycle
  task_status: CHECKED
  run_id: 01KYVMT19Z4ZNQADEE2RDWHP7V
  trace_id: none
  changed_surfaces: [`internal/client/codex/live/media.go`, `internal/client/codex/live/tcp_proxy.go`, `internal/client/codex/live/protocol.go`, `internal/client/codex/live/session.go`, `internal/api/handlers/codexlive/handler.go`, `internal/api/server.go`, `sdk/proxyutil/proxy.go`, `go.mod`, `go.sum`]
  verification: focused Codex Live relay/race/API tests, `go test ./...`, focused `go vet`, and `git diff --check` -> pass; `zharness check record --verdict APPROVED --judge same-session --judge-model claude-sonnet-5 --run-id 01KYVMT19Z4ZNQADEE2RDWHP7V ... --json` -> check `01KYVPF015XXH6D9FWSJ4WERJP`
  blocker: none
- timestamp: 2026-07-31T09:00:01Z
  phase: management-web-control-parity
  wave: phase-start
  task: phase-start
  task_status: in-progress
  run_id: 01KYVPH81P5CFGMQFCPEW04MGZ
  trace_id: none
  changed_surfaces: [`docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `zharness run create --slug management-web-control-parity --plan-id 01KYTYDKMCXHN0TQAPH1RM5K71 --json` -> created run `01KYVPH81P5CFGMQFCPEW04MGZ`
  blocker: none

- timestamp: 2026-07-31T09:33:25Z
  phase: management-web-control-parity
  wave: management-control-api + web-data-model + web-current-ux
  task: expose-runtime-control-api, finalize-auth-weight-api-contract, add-web-api-types-services, add-auth-weight-control-ui, add-runtime-control-ui
  task_status: CHECKED
  run_id: 01KYVPH81P5CFGMQFCPEW04MGZ
  trace_id: none
  changed_surfaces: [`internal/api/handlers/management/runtime_controls.go`, `internal/api/handlers/management/runtime_controls_test.go`, `internal/api/handlers/management/handler.go`, `internal/api/server.go`, `web/src/services/api/runtimeControls.ts`, `web/src/types/runtimeControls.ts`, `web/src/pages/SystemPage.tsx`, `web/src/features/authFiles/`, `web/src/i18n/locales/`, `internal/managementasset/static/management.html`]
  verification: `go test ./internal/api/handlers/management -run 'TestRuntimeControls|TestPutCodexKeys|TestAuthFileFieldsPatchWeight|TestListAuthFilesPostgresIncludesWeight' -count=1` -> pass; `go test ./internal/api ./internal/api/handlers/management -count=1` -> pass; locale JSON validation -> pass; `cd web && bun run type-check` -> pass; `cd web && bun run lint` -> 0 errors and 8 pre-existing warnings; `cd web && bun run build` -> pass; `git diff --check` -> pass
  blocker: none
- timestamp: 2026-07-31T09:33:25Z
  phase: upstream-checkpoint-final-gate
  wave: full-product-gate + checkpoint-refresh + closure-consistency
  task: run-full-go-gate, run-full-web-binary-gate, refresh-final-upstream-checkpoint, reconcile-final-release-delta, verify-harness-plan-consistency
  task_status: CHECKED_WITH_FOLLOW_UP
  run_id: none
  trace_id: none
  changed_surfaces: [`docs/upstream/cliproxyapi-checkpoint.json`, `docs/plans/active/cliproxyapi-v7-2-111-parity.md`]
  verification: `go test ./...` -> pass; `make build` -> pass and embeds `web/dist/index.html` into `internal/managementasset/static/management.html`; latest release metadata -> `v7.2.112` published `2026-07-31T08:39:29Z`; `git fetch https://github.com/router-for-me/CLIProxyAPI a63da8ae76b1a4e0c0486c3eb0fb7ccf8f33e69d:refs/upstream-checkpoints/cliproxyapi/v7.2.112` -> ref fetched; `git log 4a315136730baa8b3a436d12b74e5a702c70be5c..a63da8ae76b1a4e0c0486c3eb0fb7ccf8f33e69d` -> 11 final-gate commits; `git diff --stat` for that range -> 61 files, 2515 insertions, 1063 deletions; checkpoint JSON updated to latest release with all 11 `v7.2.112` commits classified as follow-up or reject
  blocker: `v7.2.112` introduced product deltas outside locked requirements: thinking-summary behavior and Home 401 revert need explicit refinement before implementation; completed implementation remains pinned to `v7.2.111` per R23

- timestamp: 2026-07-31T09:33:25Z
  phase: upstream-checkpoint-final-gate
  task: reconcile-final-release-delta
  decision: Do not widen the completed three-phase implementation to `v7.2.112`; update the checkpoint to latest release metadata, pin verified code scope to `v7.2.111`, reject repo-hygiene-only delta, and create explicit follow-up classifications for thinking-summary behavior plus the Home 401 revert.
  rationale: The latest release was published during final validation and contains product behavior outside the approved locked requirements; R23 requires explicit follow-up/refinement rather than silently declaring parity over unimplemented product deltas.

- timestamp: 2026-07-31T09:41:32Z
  phase: codex-live-media-relay
  wave: sideband-websocket-relay
  task: implement-sideband-websocket-relay
  task_status: DONE
  run_id: 01KYVMT19Z4ZNQADEE2RDWHP7V
  trace_id: none
  changed_surfaces: [`internal/api/handlers/codexlive/handler.go`, `internal/api/handlers/codexlive/handler_test.go`, `internal/client/codex/live/protocol.go`]
  verification: `go test ./internal/api/handlers/codexlive ./internal/client/codex/live -run 'Test.*(Sideband|HandleSideband|Protocol|CallID|PionMediaRelay|TCPCandidate|PrepareProxied|ReadValidated|BundledICE|IsPublicRemoteIP|CallSDP)' -count=1` -> pass; `go test ./internal/api/handlers/codexlive ./internal/api ./internal/client/codex/live -count=1` -> pass; `go test -race ./internal/api/handlers/codexlive ./internal/client/codex/live -run 'Test.*(Sideband|HandleSideband|PionMediaRelay|TCPCandidate)' -count=1` -> pass
  blocker: none

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- timestamp: 2026-07-31T02:26:55Z
  phase: upstream-parity-ledger
  commands:
    - `python3 -m json.tool docs/upstream/cliproxyapi-checkpoint.json >/dev/null` plus semantic and full artifact assertions -> pass; 102 unique commits, 18 releases, 411 touched paths, complete owner/evidence/verification fields, no secret-bearing content
    - `go test ./...` -> pass across all Go packages
    - `git diff --check` plus new-file whitespace assertions -> pass
    - full Security, Performance, Architecture, and Code Quality review -> pass after correcting primary-behavior ownership for weighted auth, custom executors, translator schema normalization, and Antigravity schema cleaning
    - `zharness check record --verdict APPROVED --judge same-session --judge-model claude-opus-5 --run-id 01KYTZ7RXA43KZVEZ1GNQJX2T6 ... --json` -> check `01KYTZXTX1EGT7V8XRERP8EXT1`
  run_id: 01KYTZ7RXA43KZVEZ1GNQJX2T6
  check_id: 01KYTZXTX1EGT7V8XRERP8EXT1
  verdict: APPROVED
  proof_gaps: none
- timestamp: 2026-07-31T03:47:00Z
  phase: postgres-control-foundation
  commands:
    - `gofmt -w internal/runtimecontrol/types.go internal/store/postgres_runtime_controls_integration_test.go && go test ./internal/runtimecontrol ./internal/store ./sdk/cliproxy/auth -count=1 && go test ./...` -> pass across focused foundation packages and all Go packages after removing plaintext ICE credential material from runtime settings
    - `go vet ./internal/config/... ./internal/store/... ./sdk/cliproxy/...` -> pass
    - `git diff --check` -> pass
    - same-session Security, Performance, Architecture, and Code Quality review -> pass; runtime controls remain Postgres-authoritative with safe disabled defaults, revision CAS prevents lost settings updates, cooldown snapshot CAS rejects stale deletes/resurrections, and ICE credentials are intentionally excluded until a future encrypted-secret contract exists
    - `zharness check record --verdict APPROVED --judge same-session --judge-model claude-opus-5 --run-id 01KYV1T9VE29TR5Q43G1AJE32Y ... --json` -> check `01KYV3X3GFGC008ZC8H0TPNG2P`
  run_id: 01KYV1T9VE29TR5Q43G1AJE32Y
  check_id: 01KYV3X3GFGC008ZC8H0TPNG2P
  verdict: APPROVED
  proof_gaps: none
- timestamp: 2026-07-31T04:24:00Z
  phase: auth-weight-management-parity
  commands:
    - `go test ./internal/api/handlers/management ./internal/api -run 'Test.*Auth.*(Weight|Filter|Management)' -count=1` -> pass; existing management API exposes stable selector and weight contracts
    - `go test ./sdk/cliproxy/auth ./sdk/cliproxy ./internal/api/handlers/management ./internal/store -count=1` -> pass after correcting a test fixture to use a safe stable ID for delete
    - `go vet ./sdk/cliproxy/auth ./sdk/cliproxy ./internal/api/handlers/management ./internal/store` -> pass
    - `git diff --check` -> pass
    - `zharness check record --verdict APPROVED --judge same-session --judge-model gpt-5.5 --run-id 01KYV433N09V7MVKTSJCQE5FEY ... --json` -> check `01KYV6N7FEZDN6PS7AVP3DC2QB`
  run_id: 01KYV433N09V7MVKTSJCQE5FEY
  check_id: 01KYV6N7FEZDN6PS7AVP3DC2QB
  verdict: APPROVED
  proof_gaps: none
- timestamp: 2026-07-31T05:19:41Z
  phase: home-runtime-full-parity
  commands:
    - `go test ./internal/home ./sdk/cliproxy/auth ./sdk/cliproxy/executionregistry ./sdk/cliproxy ./internal/runtime/executor/helps -count=1` -> pass
    - `go vet ./internal/home ./sdk/cliproxy/auth ./sdk/cliproxy/executionregistry ./sdk/cliproxy ./internal/runtime/executor/helps` -> pass
    - `git diff --check` -> pass
    - `zharness check record --verdict APPROVED --judge same-session --judge-model gpt-5.5 --run-id 01KYV6T0F7EW0X1JAXFPG7YF3E --proof-links ... --json` -> check `01KYV9TKK5ER9F82XA8MEHA7JR`
  run_id: 01KYV6T0F7EW0X1JAXFPG7YF3E
  check_id: 01KYV9TKK5ER9F82XA8MEHA7JR
  verdict: APPROVED
  proof_gaps: none
- timestamp: 2026-07-31T06:04:00Z
  phase: translator-protocol-parity-111
  commands:
    - `go test ./internal/translator/... ./internal/util/... -count=1` -> pass
    - `go vet ./internal/translator/... ./internal/util/...` -> pass
    - `git diff --check` -> pass
    - `zharness check record --verdict APPROVED --judge same-session --judge-model gpt-5.5 --run-id 01KYV9XPGD0QKRVJD53EF0F78Q --proof-links ... --json` -> check `01KYVB8W6EG7GGB56WQNDKAQ0R`
  run_id: 01KYV9XPGD0QKRVJD53EF0F78Q
  check_id: 01KYVB8W6EG7GGB56WQNDKAQ0R
  verdict: APPROVED
  proof_gaps: none

- timestamp: 2026-07-31T06:42:00Z
  phase: antigravity-signature-parity
  commands:
    - `go test ./internal/signature ./internal/runtime/executor ./internal/util ./internal/cache -run 'Test.*(Antigravity|GeminiFunctionCallPairing|GeminiThought|Signature|Schema|Replay)' -count=1` -> pass
    - `go vet ./internal/signature ./internal/runtime/executor ./internal/util ./internal/cache` -> pass
    - `git diff --check` -> pass
    - `zharness check record --verdict APPROVED --judge same-session --judge-model gpt-5.5 --run-id 01KYVBA78TVMFE6V43NFFEMP9X --proof-links ... --json` -> check `01KYVBZKTZEPZ0ST02SBY0E2ZS`
  run_id: 01KYVBA78TVMFE6V43NFFEMP9X
  check_id: 01KYVBZKTZEPZ0ST02SBY0E2ZS
  verdict: APPROVED
  proof_gaps: none
- timestamp: 2026-07-31T06:14:33Z
  phase: executor-websocket-runtime-parity
  commands:
    - `go test ./sdk/cliproxy/executor ./internal/runtime/executor/helps ./internal/runtime/executor ./sdk/api/handlers/openai -run 'TestBindExecutionResource|TestCodexWebsocketSessionBindsSameLifecycleAndConnectionOnce|TestDerivedSessionProviderMappings|TestProviderSessionUUIDPrefersExecutionSession|TestDerivedSessionProviderMappingsRequireIdentity|TestXAIExecutionSessionIDUsesDerivedFallback|TestAntigravityReasoningReplayClientSessionKeyUsesDerivedMetadata|TestPrependToFirstUserMessage|TestInjectToolsCacheControlSkipsDeferredTools|TestResponsesWebSocket1009ToolCacheRollbackIsTransactional|TestRepairResponsesWebsocket|TestResponsesWebsocket.*Rollback' -count=1` -> pass
    - `go vet ./sdk/cliproxy/executor ./internal/runtime/executor/helps ./internal/runtime/executor ./sdk/api/handlers/openai` -> pass
    - `git diff --check` -> pass
    - `zharness check record --verdict APPROVED --judge same-session --judge-model gpt-5.5 --run-id 01KYVC25QYPE3PHA3JNCTCZRSB --proof-links ... --json` -> check `01KYVCYGW3K2CX3ADHSVW0RNTK`
  run_id: 01KYVC25QYPE3PHA3JNCTCZRSB
  check_id: 01KYVCYGW3K2CX3ADHSVW0RNTK
  verdict: APPROVED
  proof_gaps: none
- timestamp: 2026-07-31T07:10:00Z
  phase: model-catalog-resolution-parity
  commands:
    - `go test ./internal/registry ./sdk/api/handlers/openai -run 'TestUpstreamCheckpointModelCatalogAdditions|TestLocalKiroModelDefinitionsRemainAvailable|TestCodexClientModelsIncludesCheckpointTemplates|TestCodexClientModelsUsesConfiguredDisplayNameForTemplate' -count=1` -> pass
    - `python3 comparison against pinned CLIProxyAPI 4a315136730baa8b3a436d12b74e5a702c70be5c models.json` -> pass; no upstream model IDs remain missing in llmhub catalog channels covered by the phase
    - `go test ./internal/registry ./sdk/api/handlers/openai -count=1` -> pass
    - `go vet ./internal/registry ./sdk/api/handlers/openai` -> pass
    - `git diff --check` -> pass
    - `zharness check record --verdict APPROVED --judge same-session --judge-model gpt-5.5 --run-id 01KYVD3482T1X8944ZEXBJ6SEP --proof-links ... --json` -> check `01KYVDMY4W9Z1CW83TDE3AK64T`
  run_id: 01KYVD3482T1X8944ZEXBJ6SEP
  check_id: 01KYVDMY4W9Z1CW83TDE3AK64T
  verdict: APPROVED
  proof_gaps: none
- timestamp: 2026-07-31T09:00:01Z
  phase: codex-live-media-relay
  commands:
    - `go test -race ./internal/client/codex/live -run 'Test(PionMediaRelaySelectsRemoteProxyMode|TCPCandidateTunnel|PrepareProxied|ReadValidated|BundledICE|IsPublicRemoteIP|CallSDP)' -count=1` -> pass
    - `go test ./internal/api/handlers/codexlive ./internal/api ./sdk/proxyutil -count=1` -> pass
    - `go test ./...` -> pass across all Go packages
    - `go vet ./internal/client/codex/live ./internal/api/handlers/codexlive ./internal/api ./sdk/proxyutil` -> pass
    - `git diff --check` -> pass
    - `zharness check record --verdict APPROVED --judge same-session --judge-model claude-sonnet-5 --run-id 01KYVMT19Z4ZNQADEE2RDWHP7V --proof-links ... --json` -> check `01KYVPF015XXH6D9FWSJ4WERJP`
  run_id: 01KYVMT19Z4ZNQADEE2RDWHP7V
  check_id: 01KYVPF015XXH6D9FWSJ4WERJP
  verdict: APPROVED
  proof_gaps: none


- timestamp: 2026-07-31T09:33:25Z
  phase: management-web-control-parity
  commands:
    - `go test ./internal/api/handlers/management -run 'TestRuntimeControls|TestPutCodexKeys|TestAuthFileFieldsPatchWeight|TestListAuthFilesPostgresIncludesWeight' -count=1` -> pass
    - `go test ./internal/api ./internal/api/handlers/management -count=1` -> pass
    - `python3 locale JSON validation for web/src/i18n/locales/en.json and vi.json` -> pass
    - `cd web && bun run type-check` -> pass
    - `cd web && bun run lint` -> 0 errors, 8 warnings from pre-existing lint debt
    - `cd web && bun run build` -> pass
    - `git diff --check` -> pass
  run_id: 01KYVPH81P5CFGMQFCPEW04MGZ
  check_id: none
  verdict: APPROVED_WITH_LOCAL_EVIDENCE
  proof_gaps: browser runtime smoke was not executed in this environment; production build and embedded binary build passed
- timestamp: 2026-07-31T09:33:25Z
  phase: upstream-checkpoint-final-gate
  commands:
    - `go test ./...` -> pass across all Go packages
    - `make build` -> pass; frontend production build and Go binary build completed, and `internal/managementasset/static/management.html` was regenerated
    - latest release metadata query -> `v7.2.112`, published `2026-07-31T08:39:29Z`, target `main`
    - `git fetch https://github.com/router-for-me/CLIProxyAPI a63da8ae76b1a4e0c0486c3eb0fb7ccf8f33e69d:refs/upstream-checkpoints/cliproxyapi/v7.2.112` -> pass
    - `git log --reverse --no-merges --format='%H %aI %s' 4a315136730baa8b3a436d12b74e5a702c70be5c..a63da8ae76b1a4e0c0486c3eb0fb7ccf8f33e69d` -> 11 final-gate commits
    - `git diff --stat 4a315136730baa8b3a436d12b74e5a702c70be5c..a63da8ae76b1a4e0c0486c3eb0fb7ccf8f33e69d` -> 61 files changed, 2515 insertions, 1063 deletions
  run_id: none
  check_id: none
  verdict: CHECKED_WITH_FOLLOW_UP
  proof_gaps: `v7.2.112` thinking-summary and Home 401 revert deltas are classified but not implemented; no commit/push performed in this final gate


- timestamp: 2026-07-31T09:41:32Z
  phase: codex-live-media-relay
  commands:
    - `go test ./internal/api/handlers/codexlive ./internal/client/codex/live -run 'Test.*(Sideband|HandleSideband|Protocol|CallID|PionMediaRelay|TCPCandidate|PrepareProxied|ReadValidated|BundledICE|IsPublicRemoteIP|CallSDP)' -count=1` -> pass
    - `go test ./internal/api/handlers/codexlive ./internal/api ./internal/client/codex/live -count=1` -> pass
    - `go test -race ./internal/api/handlers/codexlive ./internal/client/codex/live -run 'Test.*(Sideband|HandleSideband|PionMediaRelay|TCPCandidate)' -count=1` -> pass
  run_id: 01KYVMT19Z4ZNQADEE2RDWHP7V
  check_id: none
  verdict: APPROVED_WITH_LOCAL_EVIDENCE
  proof_gaps: no live ChatGPT credential/runtime validation; deterministic websocket relay test covers auth pinning, upstream URL shape, bidirectional frames, and session completion


- timestamp: 2026-07-31T09:42:22Z
  phase: upstream-checkpoint-final-gate
  commands:
    - `go test ./...` -> pass across all Go packages after sideband relay closure
    - `python3 checkpoint/plan invariant script && git diff --check` -> pass; checkpoint remains `v7.2.112`, implementation scope remains pinned to `v7.2.111`, and stale sideband `501` references are removed
    - `make build` -> pass; frontend production build, embedded management asset regeneration, and Go binary build completed
  run_id: none
  check_id: none
  verdict: CHECKED_WITH_FOLLOW_UP
  proof_gaps: `v7.2.112` thinking-summary and Home 401 revert deltas remain classified follow-up; no live ChatGPT credential/runtime validation; no commit/push performed


- timestamp: 2026-07-31T09:51:55Z
  phase: handoff
  commands:
    - `zharness preflight handoff --json` -> durable ready with playbook `docs/playbooks/handoff.md`
    - `zharness resume --json` -> drifted; current DB phase `management-web-control-parity` remains `in-progress`, latest check `01KYVPF015XXH6D9FWSJ4WERJP` belongs to `codex-live-media-relay` run `01KYVMT19Z4ZNQADEE2RDWHP7V`, not latest management run `01KYVPH81P5CFGMQFCPEW04MGZ`
    - `zharness handoff record --run-id 01KYVPH81P5CFGMQFCPEW04MGZ --open-items ... --json` -> handoff `01KYVSBYYMMYBJY21F7P7G6HMT`
    - `git status --short --branch` -> branch `chore/upstream-parity-v7-2-111` synced with origin; only untracked `.kit/`
  run_id: 01KYVPH81P5CFGMQFCPEW04MGZ
  check_id: none
  verdict: INCOMPLETE_HANDOFF_RECORDED
  proof_gaps: lifecycle DB/plan cannot be closed until a proper management/final check is recorded or harness drift is reconciled

## Current State and Next Action
- active_phase: upstream-checkpoint-final-gate
- lifecycle_status: checked-with-follow-up; handoff-recorded; zharness-lifecycle-drifted
- latest_run_id: 01KYVPH81P5CFGMQFCPEW04MGZ
- latest_trace_ids: []
- latest_check_id: none
- latest_handoff_id: 01KYVSBYYMMYBJY21F7P7G6HMT
- branch_state: `chore/upstream-parity-v7-2-111` pushed to `origin/chore/upstream-parity-v7-2-111` at `ebfff0aa`
- blockers: [`zharness` drift: DB current phase remains `management-web-control-parity` in-progress and latest approved check belongs to `codex-live-media-relay`, not the management/final gate; `v7.2.112` product delta requires explicit refinement before implementation: thinking-summary behavior and Home 401 revert]
- open_items: [record/reconcile a proper check for `management-web-control-parity` and/or `upstream-checkpoint-final-gate` before closing phases, review `v7.2.112` thinking-summary delta, review `v7.2.112` Home 401 revert, ignore or clean local untracked `.kit/` scratch]
- exact_next_action: run `zharness query check --latest --json` and record a check for `management-web-control-parity` or final gate if ní wants lifecycle closure; otherwise start a new refinement for `v7.2.112` thinking-summary/Home deltas
