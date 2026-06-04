# Execplan

## Steps

1. Inspect the current Claude-to-Kiro request path and map it to upstream
   request-shape fixes.
2. Normalize Kiro tool schemas so top-level object defaults and `required: []`
   are always present.
3. Replace adjacent-user repair in Kiro history with a merge pass that preserves
   accumulated tool results.
4. Synthesize a minimal tool schema from assistant `tool_calls` when follow-up
   requests omit `tools`.
5. Add focused Kiro executor regression tests for all three cases.
6. Run focused tests, broader Go verification, build the server, and verify the
   binary with repeated local API calls through a mocked Kiro upstream.

## Status

Implemented.
