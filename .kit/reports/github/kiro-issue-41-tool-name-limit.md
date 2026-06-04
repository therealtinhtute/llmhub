---
title: kiro issue 41 tool name limit
description: Kiro upstream rejects tool definitions with names longer than 64 characters; llmhub currently forwards names unchanged.
status: active
created: 2026-06-04
tags: [github, kiro-gateway, llmhub, kiro]
---

# Findings

- `jwadow/kiro-gateway` added explicit validation for Kiro's 64-character tool-name limit and raises a client-facing error before sending the upstream request. Evidence: `.kit/cache/github/jwadow/kiro-gateway/kiro/converters_core.py:560-599` and `.kit/cache/github/jwadow/kiro-gateway/kiro/converters_core.py:1435-1440`.
- In `llmhub`, Kiro tool definitions are forwarded unchanged from OpenAI payloads into `toolSpecification.name`. Evidence: `internal/runtime/executor/kiro_executor.go:607-627`.
- In `llmhub`, assistant tool calls are also forwarded unchanged into Kiro `toolUses[].name`. Evidence: `internal/runtime/executor/kiro_executor.go:659-673`.
- `llmhub` does sanitize Kiro schemas for `additionalProperties` and empty `required`, which aligns with other known Kiro request quirks. Evidence: `internal/runtime/executor/kiro_executor.go:629-656`.
- `llmhub` preserves tool-result turns as separate user messages and inserts placeholder assistant turns when needed, so the specific "tool_result lost during merge" bug from another proxy implementation is not the first match here. Evidence: `internal/runtime/executor/kiro_executor.go:541-571`, `internal/runtime/executor/kiro_executor.go:576-605`, and `internal/runtime/executor/kiro_executor.go:683-695`.

# Conclusion

The most likely current failure in `llmhub` is long MCP/Codex tool names exceeding Kiro's 64-character limit, not the tool-result merge bug.

# Next action

- Add a Kiro-specific tool-name guard in `buildKiroPayloadFromOpenAI` or `convertKiroTools`.
- Return a clear `400` error naming each oversized tool.
- Add focused tests for:
  - long tool definitions in `tools`
  - assistant `tool_calls` with matching long names
  - a preserved tool-result flow to prove the merge bug is not reintroduced
