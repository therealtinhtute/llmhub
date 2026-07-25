# Context: Auth credential concurrency

Phase: auth-credential-concurrency
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: unit, integration, race-sensitive lifecycle, full Go gate

## Goal
Port the Home credential-concurrency lifecycle from CLIProxyAPI v7.2.96 while
preserving llmhub's existing Postgres auth persistence and provider behavior.

## Scope Boundary
### Allowed Surfaces
- `internal/config/`
- `internal/home/`
- `internal/homeplugins/`
- `internal/api/` and `sdk/api/handlers/`
- `internal/runtime/executor/` where execution resources must bind to lifecycle
- `sdk/cliproxy/auth/`
- `sdk/cliproxy/executionregistry/`
- `sdk/cliproxy/executor/`
- `sdk/cliproxy/` service and Home wiring
- focused fixtures and tests for these surfaces

### Forbidden Surfaces
- frontend and management-panel UI
- provider additions or model-routing changes
- database schema or Postgres ownership changes
- pluginhost behavior unrelated to lifecycle resource binding
- release, installer, and documentation work

## Spec Hooks
- Credential concurrency for Home/general request execution.
- Existing providers must retain current routing and persistence behavior.
- No breaking public API changes outside additive lifecycle hooks.

## Locked Decisions
- Upstream commit `3ecd4afe` is the behavioral reference.
- Home-synthesized concurrency policy is authoritative in Home mode.
- Every accounted dispatch must release exactly once after bound resources close.
- Busy dispatch errors preserve trusted retry timing without mutating credentials.
- Existing local Postgres auth persistence remains authoritative outside Home dispatch.

## Assumptions
- Current llmhub already contains the v7.2.93 Home/auth architecture required by
  the additive lifecycle integration.
- Dynamic/plugin execution may bind a closer, but pluginhost stream changes from
  v7.2.94 remain outside this phase.

## Canonical Refs
- `.kit/planning/SPEC.md`
- `.kit/planning/ROADMAP.md`
- CLIProxyAPI `v7.2.96`, commit `3ecd4afe`
- `sdk/cliproxy/auth/conductor.go`
- `internal/home/client.go`

## Rejected Options
- Copying the complete upstream tree: it would overwrite llmhub-specific Postgres,
  Amp, Kiro, and provider behavior.
- A semaphore local to the auth manager: it would not satisfy Home accounting,
  release acknowledgement, drain, or in-flight observation contracts.

## Deferred Ideas
- Credential migration scripts.
- Frontend visualization of concurrency state.

## Escalate If
- The lifecycle requires a database schema change.
- Existing Postgres auth persistence must be replaced rather than extended.
- A required upstream dependency falls inside unrelated pluginhost behavior.
