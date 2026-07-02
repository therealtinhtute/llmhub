# Overview

## Current Behavior

LLMHub is a customized fork of CLIProxyAPI with Postgres-only runtime storage,
embedded management web, Kiro provider support, Amp routes, and custom release
installers. The upstream CLIProxyAPI range from `v7.1.23` to `v7.2.49` contains
provider/API/runtime fixes and model additions that are not fully present
locally.

## Target Behavior

LLMHub backports the useful upstream parity items without importing
`pluginhost/pluginstore` or removing local custom behavior. The proxy supports
new upstream media/model entries, selected Responses/WebSocket/translator
fixes, XAI/Antigravity reliability updates, and Postgres-compatible
auth/cooldown/quota management.

## Affected Users

- CLI users calling OpenAI-compatible `/v1` and provider-specific routes.
- Management users operating auth files, quota state, and model visibility.
- Operators running llmhub with Postgres-backed runtime storage.

## Affected Product Docs

- `plans/cliproxyapi-upstream-parity-2026-07-02.md`
- `README.md`
- `web/README.md` when management-visible behavior changes.

## Non-Goals

- Do not add upstream `pluginhost/pluginstore` or external plugin execution.
- Do not remove Amp, Gemini CLI paths, Kiro, Postgres runtime, or embedded web.
- Do not import upstream release/Docker/sponsor/branding churn.
