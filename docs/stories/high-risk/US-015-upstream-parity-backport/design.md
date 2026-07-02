# Design

## Domain Model

The relevant domain objects are provider auth records, model registry entries,
OpenAI-compatible request/response translations, runtime cooldown/quota state,
and management-visible operational controls.

## Application Flow

OpenAI-compatible and provider-specific handlers select a model/auth, translate
requests to the selected provider protocol, execute through the runtime
executor, then translate responses back to the client protocol. Management
routes mutate runtime config or provider state through the existing auth manager
and Postgres-backed store.

## Interface Contract

- Existing `/v1` and provider routes remain compatible.
- Add media/model support without changing existing `gpt-image-2` default.
- Add `/openai/v1/videos` aliases only alongside existing `/v1/videos`.
- Add quota reset using the existing management authentication and auth
  identifier style.

## Data Model

Runtime state remains Postgres-owned. Any cooldown/quota persistence must use
existing Postgres store patterns or schema extensions; no upstream file-backed
runtime state may be introduced.

## UI / Platform Impact

Management web may expose new quota reset or model visibility only after backend
contracts are stable. Amp routes and embedded web packaging stay unchanged.

## Observability

Keep existing request/error logging behavior. Add tests for route inclusion
where new video aliases should be treated as AI API paths. Record Harness
validation evidence after each slice.

## Alternatives Considered

1. Wholesale upstream merge: rejected because upstream removes Amp, adds plugin
   execution, and assumes runtime shapes that conflict with llmhub.
2. Plugin parity now: rejected as a separate high-risk initiative.
3. Targeted backports: selected because it keeps changes reviewable and
   preserves local custom behavior.
