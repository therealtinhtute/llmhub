# Overview

## Current Behavior

Claude-compatible requests that route to the Kiro executor can fail with
`400 Improperly formed request` when the conversation contains tool usage.

The failure surface is not basic auth or model routing. It is the request shape
produced inside `internal/runtime/executor/kiro_executor.go` after the repo
translates `claude -> openai -> kiro`.

Known risk points:

- Kiro tool schemas can lose `required: []` and top-level object defaults.
- Role normalization can leave adjacent user turns in Kiro history.
- Follow-up turns can reference historical `tool_calls` even when the current
  request body omits `tools`.

## Target Behavior

Claude-compatible Kiro requests keep the strict request shape Kiro expects:

- tool schemas always include object defaults and `required: []`
- Kiro history does not contain adjacent user turns after normalization
- missing request `tools` are synthesized from prior assistant `tool_calls`

## Affected Users

- Claude Code users calling Kiro models through `/v1/messages`
- OpenAI-compatible agent clients that hit the same Kiro executor path

## Affected Product Docs

- `docs/stories/US-002-kiro-provider/*`

## Non-Goals

- Rewriting the Kiro path to a native Claude-to-Kiro translator
- Changing auth import, refresh, or scheduler/failover behavior
- Changing public API handler contracts outside the Kiro executor path
