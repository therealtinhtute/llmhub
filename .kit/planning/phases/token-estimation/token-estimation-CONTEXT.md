# Context: Token estimation

Phase: token-estimation
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: unit, stream integration, benchmark, full Go gate

## Goal
Port the v7.2.95 Claude and xAI input-token estimation changes plus the OpenAI
tool-setting allocation cleanup without changing request semantics.

## Scope Boundary
### Allowed Surfaces
- `internal/runtime/executor/`
- `internal/runtime/executor/helps/`
- `internal/translator/openai/openai/responses/`
- `go.mod` and `go.sum`
- focused tests and benchmarks

### Forbidden Surfaces
- auth selection and credential concurrency
- model routing or new model registration
- frontend, database, installers, and release artifacts

## Spec Hooks
- Token estimation/state handling and request-path performance improvements.
- Existing provider behavior and output payloads must remain compatible.

## Locked Decisions
- Upstream commits `3ad6dfe3`, `cb110ad4`, and `f3e36f19` are authoritative.
- Claude estimates patch the first downstream `message_start` only when upstream
  usage is absent/zero and must preserve SSE framing.
- xAI counts semantic text/tool segments with O200k and excludes encrypted or
  multimedia payloads.
- Tool choice and parallel-tool fields are emitted only when convertible tools exist.

## Assumptions
- The tokenizer upgrade to `v0.8.1` is compatible with Go 1.26 and current callers.

## Canonical Refs
- `.kit/planning/SPEC.md`
- CLIProxyAPI `v7.2.95`
- commits `3ad6dfe3`, `cb110ad4`, `f3e36f19`

## Rejected Options
- Counting serialized JSON bytes: it overcounts control and encoded media fields.
- Patching all stream events: it would duplicate usage and corrupt SSE framing.

## Deferred Ideas
- End-to-end live-provider token reconciliation.

## Escalate If
- Existing provider output includes a different authoritative usage event contract.
- The tokenizer upgrade changes unrelated provider counts.
