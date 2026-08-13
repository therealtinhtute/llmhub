---
id: 01KZX15BYCKAMRHF6JHDRNNK7T
type: plan
intake_id: 01KZX15RZ91ETQKBZC90E6PNA5
lane: normal
status: active
created: 2026-08-13
updated: 2026-08-13
---

# Plan: Audit finding remediation

## Outcome
- result: The confirmed audit findings in runtime startup discovery, storage-watcher delivery, native-provider config isolation, credential-file permissions, and watcher test synchronization are remediated without changing the agreed provider-routing behavior.
- success_signals:
  - A service started with combos in its initial runtime config exposes those virtual models through model discovery before any config reload.
  - Auth updates remain eventually reconcilable after a storage-watcher queue saturation event; no Add, Modify, or Delete transition is lost solely because the queue was full.
  - Native-provider hydration operates on an isolated config value and does not mutate the config pointer concurrently shared with service, management, executor, or request paths.
  - Provider credential files created or rewritten by the affected save paths have owner-only permissions (`0600`) without exposing credential contents.
  - The affected watcher tests are race-safe and the targeted race suite passes.
  - Existing Go tests, vet, frontend build/embed, and backend build remain passing after remediation.

## Authority and Requirements
- authority:
  - Owner instruction selected the bounded scope "Audit remediation" after the read-only audit: remediate the four confirmed findings and verify the watcher race result.
  - `CLAUDE.md`: minimal surgical changes, read before edit, prove before done, Postgres-only runtime configuration, and no new frontend test files under `web/`.
  - `sdk/cliproxy/service.go` and `internal/registry/model_registry.go`: initial service startup, combo registration, and model-discovery wiring.
  - `sdk/cliproxy/storage_watcher.go`: storage polling, auth snapshot reconciliation, synthesized native-provider updates, and queue dispatch behavior.
  - `internal/nativeproviders/nativeproviders.go`: native resource hydration and projection replacement semantics.
  - `internal/watcher/clients.go`, `internal/watcher/config_reload.go`, and `internal/watcher/watcher_test.go`: asynchronous persistence and debounce test synchronization paths covered by the race report.
  - Affected provider credential save paths previously identified by the audit: `internal/auth/gemini/gemini_token.go`, `internal/auth/codex/token.go`, `internal/auth/kimi/token.go`, `internal/auth/claude/token.go`, `internal/auth/xai/token.go`, and `internal/auth/vertex/vertex_credentials.go`.
  - Existing combo documentation and tests: `docs/plans/done/model-combos.md`, `internal/config/combos_test.go`, and `sdk/cliproxy/auth/combo_test.go`.
- requirements:
  - R1 [accepted]: Initial service startup must register configured combo models before serving requests, so `GET /v1/models` includes them without requiring a later config update. | source: `sdk/cliproxy/service.go`, `internal/registry/model_registry.go`, existing combo model-discovery contract.
  - R2 [accepted]: Storage-watcher Add, Modify, and Delete updates must remain eventually deliverable when the runtime update queue is temporarily full; reconciliation must not mark a change complete while its runtime update has been permanently dropped. | source: `sdk/cliproxy/storage_watcher.go` queue dispatch and snapshot diff audit.
  - R3 [accepted]: Native-provider hydration must not mutate a `*config.Config` value shared with concurrent service paths; synthesized auth generation must use an isolated config snapshot while preserving native projection replacement and persistence behavior. | source: `sdk/cliproxy/storage_watcher.go`, `internal/nativeproviders/nativeproviders.go`, Postgres runtime-storage contract.
  - R4 [accepted]: Every affected provider credential save path must create and rewrite its token file with owner-only mode `0600`, including the existing-file case, while preserving the current serialization and refresh behavior. | source: affected provider save paths and credential-file security audit.
  - R5 [accepted]: The watcher persistence and config-reload debounce tests must synchronize asynchronous state observations so `go test -race ./sdk/cliproxy/... ./internal/watcher/...` completes without data-race reports. | source: `internal/watcher/clients.go`, `internal/watcher/config_reload.go`, `internal/watcher/watcher_test.go`, recorded race-detector output.
  - R6 [accepted]: Remediation must preserve existing combo fallback/round-robin semantics, native projection persistence boundaries, and provider authentication behavior outside the confirmed findings. | source: `docs/plans/done/model-combos.md`, `internal/nativeproviders/nativeproviders.go`, repository audit scope.
  - R7 [accepted]: Verification must include focused regression checks for each remediation, the targeted race suite, `go test ./...`, `go vet ./...`, frontend build/embed, backend build, and clean working-tree inspection. | source: repository `CLAUDE.md` command contract and audit completion criteria.

## Non-goals
- NG1: Add new providers, models, combo strategies, native-provider resource types, or frontend product features.
- NG2: Redesign the Postgres runtime-storage schema, watcher polling interval, auth scheduler, or general executor routing beyond the smallest changes required by R1–R5.
- NG3: Change the documented streaming fallback boundary, fallback eligibility rules, sticky-limit behavior, or round-robin cursor semantics.
- NG4: Persist native-provider projections into generic YAML or expose raw API keys/OAuth credentials through logs, tests, management responses, or plan artifacts.
- NG5: Create frontend test files under `web/` or refactor unrelated watcher, auth, registry, or provider code.
- NG6: Remediate unconfirmed issues outside the four findings and the observed watcher test races, including host-binding policy or speculative concurrency behavior not reproduced or required by R3.

## Approach and Risks
- approach: Ship four surgical phases: register combo models during initial startup; make storage-watcher reconciliation lossless and isolate native-provider hydration; harden affected credential-file permissions; then run a dedicated race and repository verification gate. The first three remediation phases can proceed independently, while the verification gate consumes all three outputs. Keep the existing provider-routing, persistence, and streaming contracts unchanged.
- constraints:
  - Runtime configuration remains Postgres-backed; native-provider projections stay in-memory and are stripped before generic YAML persistence.
  - Credential values must not appear in source diffs, test output, logs, management responses, or plan artifacts; permission checks use temporary files and metadata only.
  - Changes must be limited to the confirmed findings and their direct regression tests; no frontend test files or unrelated refactors.
  - Shared-file changes within a phase are executed sequentially; independent provider permission paths may be tested in parallel only when they do not share mutable fixtures.
  - Phase definitions and story IDs are immutable after this planning pass; lifecycle status changes only through workflow transitions.
- decisions:
  - Register the initial combo model client at the same startup boundary that registers models for loaded auths, using the existing `llmhub-combos` client identity and idempotent registry behavior.
  - Preserve watcher snapshots while adding an owner-controlled pending-update path: an update is not considered delivered until accepted by the runtime queue, and pending updates are retried in order on later polls without blocking the poll loop forever.
  - Build an isolated config snapshot for synthesized auth generation by copying the config and the nested OpenAI-compatibility data that hydration reads or rewrites; keep `HydrateConfig`'s projection semantics unchanged and never pass the shared service config to it.
  - Replace permissive credential-file creation/truncation with explicit owner-only mode plus an explicit mode correction for existing files, applied minimally in each affected provider save path.
  - Make asynchronous watcher tests wait on deterministic signals and protect shared observations with the existing watcher mutex/atomic conventions instead of increasing sleeps or weakening race coverage.
- risks:
  - risk: Queue pressure can grow pending updates or expose ordering mistakes between Add, Modify, and Delete events.
    mitigation: Preserve update order, retry only unsent updates, cover queue saturation with a bounded test fixture, and verify eventual runtime convergence.
    recovery: Stop the phase if the queue consumer cannot drain safely; retain the existing snapshot behavior and escalate the exact event sequence before changing coalescing semantics.
  - risk: A shallow config copy could still share nested slices or maps with request paths.
    mitigation: Copy every nested OpenAI-compatibility collection touched by hydration and add a concurrent polling/read regression check.
    recovery: Do not proceed to the verification gate until the race test proves the shared config remains untouched.
  - risk: Credential mode fixes may alter existing serialization or fail to correct permissions on pre-existing files.
    mitigation: Assert file mode after both create and rewrite cases while keeping payload assertions metadata-only and preserving the current encoder flow.
    recovery: Revert only the affected save-path edits if provider tests show payload or refresh regressions, then narrow the mode change.
  - risk: Startup registration could duplicate or stale-register virtual models during reload.
    mitigation: Reuse the existing client registration/unregistration path and test initial startup plus one config update without changing combo resolution.
    recovery: Stop before broad registry changes if model discovery or reload tests show duplicate entries.
- stop_conditions:
  - Any regression in existing combo routing, native projection persistence, provider authentication, or streaming fallback behavior blocks progression to the verification gate.
  - Any test requires exposing credential contents or bypassing the privacy guard; use metadata-only assertions instead.
  - A proposed fix requires schema migration, a new public configuration surface, or an unrelated refactor; return to the locked requirements for review.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: combo-startup-discovery
    story_id: 01KZX2Z4160Q260Z229Y0MMCZA
    status: planned
    goal: Register configured combo models during initial service startup and prove model discovery before any config reload.
    depends_on: none
    allowed_surfaces:
      - sdk/cliproxy/service.go
      - internal/registry/model_registry.go
      - sdk/cliproxy/*_test.go and existing model-discovery tests
    avoided_surfaces:
      - combo resolution strategy and cursor behavior
      - HTTP streaming handlers and provider executors
    waves:
      - wave: 1
        tasks:
          - task: Trace the startup registration boundary and add the smallest call that registers initial combo models before requests are served, preserving the existing client ID and reload path.
            dependencies: none
            expected_output: Initial service startup registers the configured combo model set exactly once or idempotently, and later config updates still refresh it.
            checks:
              - go test ./sdk/cliproxy/... -count=1
              - go test ./sdk/api/... -count=1
            stop_conditions: Stop if startup registration changes provider resolution or creates duplicate registry entries.
            escalation: Compare the startup and applyConfigUpdate call order with the existing registry contract before broadening the change.
          - task: Add or extend a focused startup/model-discovery regression test without creating frontend test files.
            dependencies: task: Trace the startup registration boundary and add the smallest call that registers initial combo models before requests are served, preserving the existing client ID and reload path.
            expected_output: A combo present in initial config is visible through model discovery before a config reload, with existing combo routing tests unchanged.
            checks:
              - go test ./sdk/cliproxy/... -run 'Combo|Model' -count=1
            stop_conditions: Stop if the test requires a live external provider or exposes an unrelated startup defect.
            escalation: Use the existing service test fixtures and report any missing seam instead of adding a new integration dependency.

  - phase_slug: watcher-reconciliation-isolation
    story_id: 01KZX2Z439RKFFPBQQE1W2B7F3
    status: planned
    goal: Make storage-watcher updates eventually deliverable and isolate native-provider hydration from shared runtime config.
    depends_on: none
    allowed_surfaces:
      - sdk/cliproxy/storage_watcher.go
      - internal/nativeproviders/nativeproviders.go only where required by the isolated call contract
      - sdk/cliproxy/storage_watcher_test.go and focused watcher tests
    avoided_surfaces:
      - Postgres schema and polling interval
      - auth scheduler and executor routing
      - generic YAML persistence of native projections
    waves:
      - wave: 1
        tasks:
          - task: Add lossless queue-saturation reconciliation for Add, Modify, and Delete updates, retaining ordering and retrying unsent updates without permanently advancing the delivered baseline.
            dependencies: none
            expected_output: A full runtime update queue no longer causes a state transition to disappear from subsequent reconciliation.
            checks:
              - go test ./sdk/cliproxy/... -run 'StorageWatcher|Watcher' -count=1
              - Add a deterministic queue-full regression covering add, modify, and delete convergence.
            stop_conditions: Stop if the implementation blocks shutdown indefinitely or requires changing the queue owner outside the watcher boundary.
            escalation: Preserve the existing non-blocking poll behavior and surface the exact queue/order trade-off for review.
          - task: Verify native-provider hydration and synthesized auth diffs use an isolated config snapshot, including nested OpenAI-compatibility slices/maps, while persistence still strips projections.
            dependencies: none
            expected_output: Polling does not mutate the shared config pointer, and native resource create/update/delete still produces the expected auth updates.
            checks:
              - go test ./sdk/cliproxy/... -run 'StorageWatcher|Native|Synth' -count=1
              - go test ./internal/nativeproviders/... -count=1
              - Run a concurrent read/poll regression or targeted race check around the shared config path.
            stop_conditions: Stop if hydration changes generic config persistence or if the isolated copy omits a field required by synthesis.
            escalation: Compare the copied nested fields against `config.OpenAICompatibility` and existing projection tests before changing the public hydrator API.
      - wave: 2
        tasks:
          - task: Review the watcher diff and add regression coverage for unchanged native resources, resource deletion, queue saturation, and retry convergence.
            dependencies: wave: 1
            expected_output: Repeated polls remain idempotent, and all new tests describe observable runtime state rather than implementation details.
            checks:
              - go test ./sdk/cliproxy/... ./internal/nativeproviders/... -count=1
            stop_conditions: Stop if tests depend on timing sleeps instead of deterministic queue or store controls.
            escalation: Replace timing assumptions with channels, controlled stores, or bounded polling in the test fixture.

  - phase_slug: credential-file-hardening
    story_id: 01KZX2Z451B5M03VSS9P72VF8X
    status: planned
    goal: Enforce owner-only permissions for affected provider credential file creation and rewrites.
    depends_on: none
    allowed_surfaces:
      - internal/auth/gemini/gemini_token.go
      - internal/auth/codex/token.go
      - internal/auth/kimi/token.go
      - internal/auth/claude/token.go
      - internal/auth/xai/token.go
      - internal/auth/vertex/vertex_credentials.go
      - existing provider auth tests only
    avoided_surfaces:
      - credential serialization format and refresh protocol
      - management API responses and logging
      - unrelated auth providers
    waves:
      - wave: 1
        tasks:
          - task: Update each affected save path to enforce mode 0600 on both newly created and pre-existing credential files while preserving current write and refresh behavior.
            dependencies: none
            expected_output: All six audited provider token paths create or rewrite files readable only by the owner.
            checks:
              - go test ./internal/auth/gemini/... ./internal/auth/codex/... ./internal/auth/kimi/... ./internal/auth/claude/... ./internal/auth/xai/... ./internal/auth/vertex/... -count=1
              - Use temporary files and file metadata assertions only; do not print or compare credential contents in logs.
            stop_conditions: Stop if a provider path cannot enforce existing-file permissions without changing its serialization flow.
            escalation: Isolate the mode correction at the file-open/chmod boundary and preserve the provider-specific encoder.
          - task: Add focused metadata-only coverage for create and rewrite cases where the affected provider packages have suitable test seams.
            dependencies: task: Update each affected save path to enforce mode 0600 on both newly created and pre-existing credential files while preserving current write and refresh behavior.
            expected_output: Tests fail for permissive modes and pass for both new and existing files without exposing secrets.
            checks:
              - go test ./internal/auth/... -count=1
            stop_conditions: Stop if a test fixture would require real OAuth/API credentials or privacy-guard bypass.
            escalation: Keep the check at `os.FileMode.Perm()` and temporary paths, and leave provider-specific fixture setup unchanged.

  - phase_slug: remediation-verification-gate
    story_id: 01KZX2ZCFW013262C0JPG6QWMC
    status: planned
    goal: Run race-safe watcher checks and full repository verification without regressions after all audit remediations.
    depends_on: watcher-reconciliation-isolation
    allowed_surfaces:
      - internal/watcher/clients.go only if production synchronization is required by the observed race
      - internal/watcher/config_reload.go only if its lock usage needs a surgical correction
      - internal/watcher/watcher_test.go
      - repository build and verification outputs
    avoided_surfaces:
      - unrelated production behavior
      - new frontend tests
      - credential contents or sensitive fixture output
    waves:
      - wave: 1
        tasks:
          - task: Make asynchronous persistence and config-reload debounce observations deterministic and race-safe, preferring completion channels and existing mutexes over longer sleeps.
            dependencies: combo-startup-discovery, watcher-reconciliation-isolation, credential-file-hardening
            expected_output: The two previously reported watcher tests no longer race and retain their behavioral assertions.
            checks:
              - go test -race ./sdk/cliproxy/... ./internal/watcher/...
            stop_conditions: Stop if the race remains in production code after test synchronization or if the fix weakens the asserted behavior.
            escalation: Separate test-only synchronization from production changes and report the exact remaining race path.
          - task: Run focused regression suites for startup discovery, watcher reconciliation/isolation, credential modes, native projections, and combo semantics.
            dependencies: combo-startup-discovery, watcher-reconciliation-isolation, credential-file-hardening
            expected_output: Each accepted requirement has a passing focused proof and no existing combo/native-provider behavior regresses.
            checks:
              - go test ./sdk/cliproxy/... ./sdk/api/... ./internal/nativeproviders/... ./internal/watcher/... -count=1
              - go test ./internal/auth/... -count=1
            stop_conditions: Stop on any failure attributable to a remediation before running broad builds.
            escalation: Return to the owning phase and preserve the failing focused command/output.
      - wave: 2
        tasks:
          - task: Run the repository-wide quality gate and inspect generated/build state.
            dependencies: wave: 1
            expected_output: Go tests, vet, frontend build/embed, backend build, diff hygiene, and working-tree inspection all meet the repository contract.
            checks:
              - go test ./...
              - go vet ./...
              - make build-web
              - make build
              - git diff --check
              - git status --short
            stop_conditions: Stop and report exact output if any command fails; do not claim completion with a failing or skipped gate.
            escalation: Fix only regressions caused by this initiative or hand back the phase with the failing command and proof gap.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- none

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
