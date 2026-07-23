# Plan: Model and display-name compatibility

Phase: model-display-compatibility
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-07-22

## Goal

Add the approved model metadata and propagate configured display names across listing APIs without changing IDs, auth selection, routing keys, previews, or unsupported provider behavior.

## Inputs

- Approved Phase 4 governed-tree fingerprint.
- `.kit/planning/SPEC.md`
- phase CONTEXT and detailed fan-out plan.

## Wave 1 — Parallel isolated slices

### P5-S1 — Model metadata

- exact touches:
  - `internal/registry/models/models.json`
  - `internal/registry/model_definitions_test.go`
- steps:
  1. Add Grok 4.5 with pinned upstream limits and thinking metadata.
  2. Add the approved Gemini production IDs additively.
  3. Retain every corresponding preview ID.
  4. Add exact-definition and coexistence fixtures.
- verification:
  - `go test ./internal/registry -run 'Test.*(Grok45|GeminiProduction|Preview|ModelDefinition)' -count=1`
- stop if:
  - an ID cannot be represented additively;
  - GPT-5.6, Kimi K3, or remote refresh becomes necessary.

### P5-S2 — Display-name propagation

- exact touches:
  - `internal/config/config.go`
  - `internal/config/vertex_compat.go`
  - `internal/config/model_display_name_test.go` (planned new)
  - `internal/registry/model_registry.go`
  - `sdk/cliproxy/service.go`
  - `sdk/cliproxy/config_model_display_name_test.go` (planned new)
  - `sdk/api/handlers/openai/openai_handlers.go`
  - `sdk/api/handlers/openai/codex_client_models.go`
  - `sdk/api/handlers/openai/model_display_name_test.go` (planned new)
  - `sdk/api/handlers/claude/code_handlers.go`
  - `sdk/api/handlers/claude/model_display_name_test.go` (planned new)
  - `sdk/api/handlers/gemini/gemini_handlers.go`
  - `sdk/api/handlers/gemini/gemini-cli_handlers.go`
  - `sdk/api/handlers/gemini/model_display_name_test.go` (planned new)
- steps:
  1. Add optional display-name fields to the supported configured model and OAuth alias structures.
  2. Preserve presentation metadata through sanitization, registration, cloning, replacement, and alias forks.
  3. Expose protocol-specific display fields in OpenAI, Claude, Gemini, Gemini CLI, and Codex model listings.
  4. Prove IDs, upstream names, auth selection, aliases, and routing are unchanged.
- verification:
  - `go test ./internal/config ./sdk/cliproxy ./sdk/api/handlers/openai ./sdk/api/handlers/claude ./sdk/api/handlers/gemini -run 'Test.*(DisplayName|ModelList|Models)' -count=1`
- stop if:
  - any listing requires a breaking public-schema change;
  - display names become routing or auth identity.

Both slices start from the same immutable Phase 4 base in separate worktrees.

## Wave 2 — Apply, gate, and close

- apply order: P5-S1, P5-S2
- closure touches:
  - `docs/decisions/0013-model-display-name-presentation-contract.md`
  - `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
  - `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
  - append-only harness/evidence paths
- verification:
  - `go test ./internal/registry ./internal/config ./sdk/cliproxy ./sdk/api/handlers/openai ./sdk/api/handlers/claude ./sdk/api/handlers/gemini -count=1`
  - `go test ./...`
  - `go vet ./...`
  - `make build-web`
  - `make build`
  - `git diff --check`
- rollback:
  - reverse P5-S2 then P5-S1; reverse closure docs patch; verify Phase 4 fingerprint.

## Risks / Watch-fors

- Preview IDs and routing IDs are compatibility invariants.
- Build commands must not silently mutate lockfiles or generated tracked assets.
