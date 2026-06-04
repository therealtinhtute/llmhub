---
title: Kiro Anthropic-compatible proxy fix plan
description: Research-backed plan for Claude Code to Kiro request-shape failures
status: active
created: 2026-06-04
tags: [brainstorm, kiro, claude, anthropic-compatible, proxy]
---

# Recommendation

Implement a narrow parity patch in `internal/runtime/executor/kiro_executor.go` and its tests first.

Reason:

- The current `llmhub` Kiro path already uses a dedicated executor and a single `claude -> openai -> kiro` translation seam.
- The concrete failures reported upstream map directly to 3 local translator gaps, not to auth or model routing.
- A localized patch is reversible, testable, and does not require changing public API handlers.

# Evidence

## External

- OmniRoute issue [#2113](https://github.com/diegosouzapw/OmniRoute/issues/2113) reports `400 Improperly formed request` only for agent flows, while direct chat works.
- OmniRoute PR [#2104](https://github.com/diegosouzapw/OmniRoute/pull/2104) fixed Kiro request acceptance by:
  - forcing tool schemas to include `required: []`
  - lowercasing tool result `status` to `success`
  - preserving trailing assistant history and using a synthetic continue turn
- OmniRoute PR [#2105](https://github.com/diegosouzapw/OmniRoute/pull/2105) fixed adjacent user history turns after role normalization by merging them instead of leaving invalid history shape.
- OmniRoute PR [#2149](https://github.com/diegosouzapw/OmniRoute/pull/2149) fixed cases where history references `tool_calls` but request `tools` is omitted, by synthesizing a minimal tool schema from history.
- Exa search found the same failure family in other Kiro proxies: tools-only or tool-history requests are the common trigger, not plain chat.

## Local

- `sdk/api/handlers/claude/code_handlers.go` sends Claude requests through `SourceFormat=claude`.
- `internal/runtime/executor/kiro_executor.go` then translates that source to OpenAI and builds the Kiro payload in `buildKiroPayloadFromOpenAI(...)`.
- `sanitizeKiroSchema(...)` currently removes `required: []` and does not guarantee `type: "object"` plus `properties: {}`.
- `ensureKiroAlternating(...)` currently injects assistant placeholders between adjacent user turns instead of merging normalized user turns.
- There is no local synthesis path when history contains `tool_calls` but `payload["tools"]` is empty.

# Options Considered

## Option 1: Narrow Kiro translator parity patch

Change only the Kiro executor request builder to match the known upstream fixes.

Pros:

- Smallest blast radius
- Directly addresses observed request-shape bugs
- Easy to cover with focused Go tests

Cons:

- Still keeps double translation (`claude -> openai -> kiro`)
- May leave some obscure Claude-specific edge cases for later

## Option 2: Add a native Claude-to-Kiro translator path

Bypass the OpenAI intermediate representation for Claude `/v1/messages` requests.

Pros:

- Cleaner conceptual mapping for Claude Code
- Avoids format loss across two translators

Cons:

- Much larger change
- Higher regression risk for existing OpenAI/Kiro flows
- Not justified until parity patch fails on real payload replays

## Option 3: Add replay tooling first, fix later

Build request capture/replay infrastructure before touching translator logic.

Pros:

- Better long-term debugging
- Useful for future provider issues

Cons:

- Does not solve the user-visible bug
- Overkill for a failure pattern already confirmed by upstream fixes

# Chosen Path

Choose Option 1 now.

Option 2 is only justified if real Claude Code traces still fail after parity fixes.
Option 3 is a follow-up improvement, not the first move.

# Fix Plan

## Phase 1: Match known Kiro schema constraints

Update `sanitizeKiroSchema(...)` so every emitted tool schema always has:

- `type: "object"`
- `properties: {}`
- `required: []`

Rules:

- Keep stripping unsupported `additionalProperties`
- Preserve caller-provided properties
- Never drop an empty `required` array

Tests to add:

- empty schema becomes `{type:"object",properties:{},required:[]}`
- schema without `required` gets `required:[]`
- schema with `required:[]` keeps it

## Phase 2: Fix normalized history shape

Replace the current placeholder-based alternation repair with a merge pass for adjacent user turns in Kiro history.

Rules:

- concatenate adjacent user contents with `\n\n`
- merge `userInputMessageContext` arrays such as `toolResults`
- preserve assistant turns in order

Tests to add:

- `system -> user` collapses into one user history turn
- `tool -> user` becomes one merged user turn, not user/assistant/user

## Phase 3: Synthesize minimal tools from history when request tools are absent

If `payload["tools"]` is empty but assistant history includes `tool_calls`, synthesize a minimal schema per tool name:

- `type: function`
- `function.name`
- `function.description = "Tool: <name>"`
- `function.parameters = {type:"object",properties:{},required:[]}`

Reason:

- Claude/OpenAI clients sometimes omit `tools` on later turns while still sending tool-call history.
- Upstream evidence shows Kiro rejects that shape.

Tests to add:

- history has tool calls and body tools empty -> synthesized tools present
- body tools already present -> do not override

## Phase 4: Revisit trailing assistant fallback

Evaluate whether `"(empty placeholder)"` should become a `Continue` synthetic user turn when the request ends on an assistant turn.

Priority:

- secondary
- ship only if replay or focused tests show current placeholder causes rejection or degraded behavior

# Verification Plan

Minimum focused proof:

- `go test ./internal/runtime/executor`

Required regression bundle before merge:

- `go test ./internal/runtime/executor ./sdk/cliproxy ./internal/registry`
- `go test ./...`

Recommended live smoke after tests:

- replay one Claude-compatible payload with tools
- replay one follow-up turn with prior `tool_calls` but omitted `tools`
- confirm Kiro no longer returns `ValidationException` / `Improperly formed request`

# Implementation Order

1. Schema normalization
2. Adjacent-user merge
3. Tool synthesis from history
4. Optional continue-turn fallback

# Out Of Scope

- Auth import or token refresh changes
- Model catalog changes
- New public endpoints
- Native Claude-to-Kiro translator rewrite
- Scheduler/failover behavior

# Exit Criteria

- Claude-compatible Kiro requests with tools stop failing on the known malformed-request shapes
- Focused Kiro executor tests cover all three regression families
- No OpenAI or Claude handler changes are required outside the Kiro executor path
