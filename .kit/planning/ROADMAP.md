# ROADMAP: CLIProxyAPI v7.2.93 targeted parity backport

Entry Phase: `websocket-message-too-big`
Execution Mode: full

## Planning Basis

- source spec: `.kit/planning/SPEC.md`
- source plan: `.kit/plans/2026-07-21-cliproxyapi-v7.2.93-backport/plan.md`
- evidence: `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- planning mode: `full`
- execution inventory: 1 Phase 1 closure + 9 implementation slices + 4 gates + 4 closures + 1 final evidence gate = 19 execution units
- fan-out policy: parallel implementation only in isolated worktrees from one immutable approved phase base; reviewed patches apply serially to the main tree
- detailed orchestration: `.kit/plans/2026-07-22-cliproxyapi-v7.2.93-fanout/plan.md`

## Phase 1: WebSocket message-too-big

**Slug:** `websocket-message-too-big`

**Goal:** Propagate upstream WebSocket close code 1009 as a structured request-scoped error without credential fallback.

**Deliverables:**

- Close-1009 mapping to HTTP 413 / `message_too_big`.
- Request-scoped auth-manager classification.
- Handler propagation and regression tests.

**Dependencies:**

- Existing Codex WebSocket executor and OpenAI Responses WebSocket handler.

**Risks / Watch-fors:**

- Incorrectly suppressing fallback for unrelated transport errors.
- Losing structured error fields in the downstream handler.

## Phase 2: Auth and CountTokens reliability

**Slug:** `auth-count-reliability`

**Goal:** Correct quota-window escalation, cooldown jitter, `invalid_grant`, and CountTokens availability behavior while preserving local persistence and Kiro hooks.

**Deliverables:**

- Once-per-window quota escalation with bounded jitter.
- HTTP 400 `invalid_grant` recognition and suspension.
- Generic CountTokens endpoint-error neutrality with explicit model-not-found handling.
- Amp/shared-path regression coverage.

**Dependencies:**

- Existing auth manager cooldown and Postgres persistence abstractions.

**Risks / Watch-fors:**

- Provider-wide retry behavior regression.
- Dropping Kiro usage/cooldown persistence updates.

## Phase 3: Translator content fidelity

**Slug:** `translator-content-fidelity`

**Goal:** Preserve file data, MIME types, structured tool outputs, and provider output indices across selected translator paths.

**Deliverables:**

- Shared pure file-data normalization helper.
- Claude-schema-valid tool-result arrays preserved with canonical compact-JSON fallback for arbitrary values.
- Codex provider output-index consumption, including zero, with local fallback only when absent.
- Cross-provider fixtures.

**Dependencies:**

- Existing translator registry and byte-oriented transformation pipeline.

**Risks / Watch-fors:**

- Changing valid empty Claude content behavior.
- Extra payload copies or data-URL corruption.

## Phase 4: Codex and xAI tool protocol parity

**Slug:** `tool-protocol-parity`

**Goal:** Support custom, additional, and namespaced tools through Codex/xAI while filtering provider-internal xAI search traces.

**Deliverables:**

- Namespace qualification/restoration helpers.
- Codex HTTP and WebSocket custom and `additional_tools` round trips.
- xAI `additional_tools` promotion and namespace routing.
- Client-declaration-aware `x_search` filtering.

**Dependencies:**

- Existing Codex and xAI translator/executor tool normalization.

**Risks / Watch-fors:**

- Tool-name collisions or double-qualification.
- Filtering client-declared tools as provider-internal traces.

## Phase 5: Model and display-name compatibility

**Slug:** `model-display-compatibility`

**Goal:** Add safe model metadata and propagate configured display names consistently without advertising unsupported GPT-5.6 or Kimi K3 behavior.

**Deliverables:**

- Grok 4.5 metadata.
- Gemini production IDs retained alongside preview IDs.
- Config/OAuth alias display-name fields.
- Consistent model-list output across OpenAI, Claude, Gemini, Gemini CLI, and Codex clients.

**Dependencies:**

- Existing embedded model registry and listing handlers.

**Risks / Watch-fors:**

- Changing routing IDs while applying display names.
- Removing preview IDs used by existing clients.
