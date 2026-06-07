# Design

## Execution Path

`/v1/messages` requests enter `sdk/api/handlers/claude/code_handlers.go`, keep
`SourceFormat=claude`, then route through `internal/runtime/executor/kiro_executor.go`.
The Kiro executor already uses `sdktranslator.TranslateRequest(...)` to convert
the source payload into OpenAI shape before building the final Kiro request.

## Scope Of Fix

Keep the current executor boundary and patch only the request-builder logic.

Planned changes:

- treat tool-result context as a list so merged user turns can preserve all
  normalized tool results
- normalize only the top-level Kiro tool schema with `type`, `properties`, and
  `required`, while still recursively stripping unsupported keys such as
  `additionalProperties`
- merge adjacent user turns inside Kiro history instead of inserting assistant
  placeholders
- synthesize minimal tool declarations from assistant `tool_calls` when the
  current request omits `tools`
- strip completed historical structured tool turns from Kiro history, narrate
  their results into user content, and keep only the active final assistant
  tool turn paired with the current request
- flatten orphan current `tool_result` payloads into message text instead of
  sending unmatched structured `toolResults` upstream

## Boundaries

- Do not touch auth bootstrap or model registry behavior.
- Do not change Claude handler request parsing.
- Do not add a second translator path unless the parity patch fails verification.
