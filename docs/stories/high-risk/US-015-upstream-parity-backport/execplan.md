# Exec Plan

## Goal

Backport useful CLIProxyAPI upstream parity into LLMHub while preserving local
runtime, provider, web, and release customizations.

## Scope

In scope:

- Responses/WebSocket/translator fixes.
- XAI, Antigravity, and related provider reliability fixes.
- Model/media additions: `gpt-image-1.5`, `grok-imagine-video-1.5-preview`,
  Claude Sonnet 5, Gemini 3.5 refinements, and Kimi K2.7 Code.
- Postgres-compatible quota reset, cooldown, auth removal, and retry/dedupe
  improvements.
- Minimal management API/docs/web exposure for new behavior.

Out of scope:

- Upstream pluginhost/pluginstore and plugin SDK packages.
- Removing Amp or Gemini CLI routes.
- Replacing local persistence with upstream file-backed runtime state.
- Release publishing.

## Risk Classification

Risk flags:

- Auth.
- External systems/providers.
- Public contracts.
- Existing behavior.
- Multi-domain.
- Weak proof if live provider credentials are unavailable.

Hard gates:

- Auth.
- External provider behavior.
- Public API behavior.

## Work Phases

1. Create story packet and register Harness story.
2. Backport media/model parity with focused tests.
3. Backport Responses/WebSocket/translator fixes.
4. Backport XAI/Antigravity runtime reliability.
5. Backport quota/cooldown/auth management with Postgres persistence.
6. Update docs/web surfaces where behavior is exposed.
7. Run focused slice tests, then full verification gate.
8. Update this story and the upstream parity report with evidence.

## Stop Conditions

Pause for human confirmation if:

- A backport requires adding upstream pluginhost/pluginstore.
- A backport requires removing Amp, Gemini CLI, Kiro, or Postgres runtime.
- A change introduces file-backed runtime state outside existing local policy.
- Live provider verification becomes mandatory but credentials are unavailable.
