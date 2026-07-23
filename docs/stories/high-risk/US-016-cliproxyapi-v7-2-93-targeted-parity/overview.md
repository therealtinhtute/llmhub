# Overview

## Current Behavior

LLMHub is a customized CLIProxyAPI fork whose runtime configuration and auth
state remain Postgres-owned. The upstream range from `v7.2.49` through
`v7.2.93` contains correctness and compatibility fixes that are useful locally,
but also includes plugin, release, provider, and runtime behavior that is not
compatible with llmhub's Amp, Kiro, embedded-management, and storage contracts.

## Target Behavior

Backport only the approved WebSocket, auth, translator, tool-protocol, model,
and display-name behavior from CLIProxyAPI `v7.2.93` at commit `01f387f4`.
Every phase is independently tested, reviewed, fingerprinted, and reversible.
No wholesale upstream merge, runtime storage change, or publication is part of
this story.

## Affected Users

- Coding clients using OpenAI Responses over Codex WebSockets.
- Operators relying on credential fallback, cooldown, and model availability.
- Clients translating file, tool-result, and tool-call content across protocols.
- Clients consuming model lists through OpenAI, Claude, Gemini, Gemini CLI, and
  Codex-compatible APIs.

## In Scope

1. Structured request-scoped close-1009 handling without credential fallback.
2. Cooldown, `invalid_grant`, and CountTokens reliability.
3. File/MIME, Claude tool-result, and provider-index fidelity.
4. Codex/xAI custom, additional, namespaced, and internal-search tool behavior.
5. Grok/Gemini metadata and presentation-only configured display names.

## Non-Goals

- Google Interactions, pluginhost/pluginstore, or plugin synchronization.
- Runtime remote model-catalog refresh.
- GPT-5.6, Kimi K3, native xAI API-key configuration, or unrelated provider work.
- Database schema, UI, installer, Docker, release, branding, or publication work.
- Commit, push, pull request, merge, or release creation.
