# Context: Translator content fidelity

Phase: translator-content-fidelity
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: unit, integration

## Goal

Preserve OpenAI file content, MIME types, structured tool outputs, and provider output indices across selected translator paths.

## Scope Boundary

### Allowed Surfaces

- `internal/translator/common/file_data.go`
- `internal/translator/common/file_data_test.go`
- `internal/translator/claude/openai/responses/*`
- `internal/translator/gemini/openai/chat-completions/*`
- `internal/translator/gemini-cli/openai/chat-completions/*`
- `internal/translator/codex/openai/chat-completions/*`
- `internal/translator/codex/claude/*`

### Forbidden Surfaces

- Google Interactions.
- Tool namespace/additional-tool behavior reserved for the next phase.
- Executor retry/auth behavior.
- Registry, config, storage, plugin, logging, UI, installer, release, and schema changes.
- Performance refactors not required for correctness.

## Spec Hooks

- Requirements 7-9.
- Empty Claude `content: []` behavior must remain valid.
- Translator helpers remain pure and side-effect free.

## Locked Decisions

- One shared file-data helper supports raw base64 and data URLs.
- Data-URL MIME is authoritative; filename extension is fallback only.
- Tool-output arrays remain structured only when every member maps to a valid Claude tool-result content block; arbitrary objects and invalid/mixed arrays use compact JSON text fallback, matching canonical upstream behavior.
- Provider `output_index` is used when present, including zero; current sequential logic remains fallback only when the field is absent.

## Assumptions

- Existing `gjson`/`sjson` patterns can preserve valid Claude content-block arrays and produce deterministic compact JSON fallback without a new type layer.
- Existing MIME extension mapping remains suitable as fallback for raw base64; malformed or MIME-less data URLs are rejected rather than guessed.

## Canonical Refs

- `.kit/planning/SPEC.md`
- `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- `.kit/cache/github/router-for-me/CLIProxyAPI/internal/translator/common/file_data.go:10-39`
- `internal/translator/claude/openai/responses/claude_openai-responses_request.go:253-270`
- `internal/translator/claude/openai/responses/claude_openai-responses_request.go:364-371`
- `internal/translator/codex/claude/codex_claude_response.go:163-207`

## Rejected Options

- Duplicate data-URL parsing in each translator: rejected because behavior would drift.
- Replace all translator construction with upstream performance helpers: rejected because correctness does not require a broad refactor.

## Deferred Ideas

- Shared cache-control preservation across all translator families.
- Translation-result caching or request payload reuse optimization.
- Google Interactions file handling.

## Escalate If

- Preserving structured output requires changing client-visible response schemas outside the selected translator.
- The common helper must depend on executor/config/storage state.
