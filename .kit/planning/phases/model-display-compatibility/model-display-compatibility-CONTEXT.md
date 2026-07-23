# Context: Model and display-name compatibility

Phase: model-display-compatibility
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: unit, integration

## Goal

Add safe Grok/Gemini model metadata and propagate configured display names across model-list APIs without changing routing IDs.

## Scope Boundary

### Allowed Surfaces

- `internal/registry/models/models.json`
- `internal/registry/model_definitions_test.go`
- `internal/registry/model_registry.go`
- `internal/config/config.go`
- `internal/config/vertex_compat.go`
- `internal/config/model_display_name_test.go`
- `sdk/cliproxy/service.go`
- `sdk/cliproxy/config_model_display_name_test.go`
- `sdk/api/handlers/openai/openai_handlers.go`
- `sdk/api/handlers/openai/codex_client_models.go`
- `sdk/api/handlers/claude/code_handlers.go`
- `sdk/api/handlers/gemini/gemini_handlers.go`
- `sdk/api/handlers/gemini/gemini-cli_handlers.go`
- adjacent focused model-list tests

### Forbidden Surfaces

- GPT-5.6 model entries, header overrides, or search routing.
- Kimi K3 metadata, native thinking, normalization, or CountTokens.
- Remote Codex catalog refresh.
- xAI API-key config.
- Executor, auth, storage, plugin, logging, UI, installer, release, and schema changes.

## Spec Hooks

- Requirements 13-15.
- Existing preview model IDs remain discoverable.
- Display names must never alter model IDs or routing keys.

## Locked Decisions

- Add Grok 4.5 using upstream metadata.
- Add Gemini production IDs alongside previews rather than replacing them.
- Add optional `display-name` to configured model and OAuth alias structures.
- Propagate display names consistently through registry cloning and model-list handlers.

## Assumptions

- Existing registry `DisplayName` support can be extended without changing public model IDs.
- Duplicate production/preview entries are acceptable and testable as compatibility aliases.

## Canonical Refs

- `.kit/planning/SPEC.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/registry/models/models.json:2424-2430`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/config/config.go:366-367`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/config/config.go:501-621`
- `internal/registry/model_registry.go:33`
- `sdk/api/handlers/openai/openai_handlers.go:70`

## Rejected Options

- Replace Gemini preview IDs: rejected because existing clients may depend on them.
- Add GPT-5.6 metadata only: rejected because header/search behavior is coupled.
- Add Kimi K3 metadata only: rejected because thinking and normalization are coupled.
- Enable periodic remote catalog fetch: rejected because it adds runtime outbound dependency and nondeterminism.

## Deferred Ideas

- GPT-5.6 coupled executor support.
- Kimi K3 coupled executor support.
- Remote catalog refresh behind a future explicit operator setting.

## Escalate If

- Adding production IDs causes registry key collisions rather than additive entries.
- Display-name propagation requires changing model routing identity.
