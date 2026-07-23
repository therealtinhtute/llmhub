# Plan: Translator content fidelity

Phase: translator-content-fidelity
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-07-22

## Goal

Normalize selected OpenAI file inputs, preserve Claude-schema-valid tool-result arrays with canonical JSON-text fallback, and retain Codex provider output indices.

## Inputs

- Approved Phase 2 governed-tree fingerprint.
- `.kit/planning/SPEC.md` as amended on 2026-07-22.
- phase CONTEXT and detailed fan-out plan.

## Wave 1 — Parallel isolated slices

All three agents start from the same immutable Phase 2 base. Their exact file allowlists are disjoint.

### P3-S1 — File-data normalization

- exact touches:
  - `internal/translator/common/file_data.go` (planned new)
  - `internal/translator/common/file_data_test.go` (planned new)
  - `internal/translator/gemini/openai/chat-completions/gemini_openai_request.go`
  - `internal/translator/gemini/openai/chat-completions/gemini_openai_request_test.go` (planned new if absent)
  - `internal/translator/gemini-cli/openai/chat-completions/gemini-cli_openai_request.go`
  - `internal/translator/gemini-cli/openai/chat-completions/gemini-cli_openai_request_test.go` (planned new if absent)
  - `internal/translator/codex/openai/chat-completions/codex_openai_request.go`
  - `internal/translator/codex/openai/chat-completions/codex_openai_request_test.go`
- behavior:
  - raw base64 uses supplied MIME or filename extension fallback;
  - explicit-MIME data URLs strip the prefix and preserve MIME;
  - malformed, MIME-less, non-base64, empty, or payload-less data URLs fail deterministically.
- verification:
  - `go test ./internal/translator/common ./internal/translator/gemini/openai/chat-completions ./internal/translator/gemini-cli/openai/chat-completions ./internal/translator/codex/openai/chat-completions -count=1`

### P3-S2 — Claude tool-result content

- exact touches:
  - `internal/translator/claude/openai/responses/claude_openai-responses_request.go`
  - `internal/translator/claude/openai/responses/claude_openai-responses_request_test.go`
- behavior:
  - strings remain strings;
  - valid Claude content-block arrays remain structured;
  - arbitrary objects, scalars, and invalid/mixed arrays become compact JSON text;
  - valid empty-array and `content: []` behavior remains unchanged.
- verification:
  - `go test ./internal/translator/claude/openai/responses -run 'Test.*(FunctionCallOutput|ToolResult|Structured|Content)' -count=1`

### P3-S3 — Codex-to-Claude output indices

- exact touches:
  - `internal/translator/codex/claude/codex_claude_response.go`
  - `internal/translator/codex/claude/codex_claude_response_test.go`
- behavior:
  - provider `output_index` wins when present, including zero;
  - start/delta/stop use one stable selected index;
  - sequential fallback applies only when the field is absent.
- verification:
  - `go test ./internal/translator/codex/claude -run 'Test.*(OutputIndex|Interleav|ContentBlock)' -count=1`

Each slice must produce immutable patch/test/review artifacts. Stop on any path outside its exact allowlist.

## Wave 2 — Apply, gate, and close

- apply order: P3-S1, P3-S2, P3-S3
- closure touches:
  - `docs/decisions/0011-translator-structured-output-and-index-fidelity.md`
  - `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
  - `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
  - append-only harness/evidence paths
- verification:
  - `go test ./internal/translator/common ./internal/translator/claude/openai/responses ./internal/translator/gemini/openai/chat-completions ./internal/translator/gemini-cli/openai/chat-completions ./internal/translator/codex/openai/chat-completions ./internal/translator/codex/claude -count=1`
  - `go test ./...`
  - `go vet ./...`
  - `go build ./...`
  - `git diff --check`
- rollback:
  - reverse P3-S3, P3-S2, P3-S1; reverse closure docs patch; verify Phase 2 fingerprint.

## Risks / Watch-fors

- Antigravity and Google Interactions remain forbidden.
- Never reinterpret `output_index: 0` as absent.
- Never pass arbitrary JSON objects directly as invalid Claude content.
