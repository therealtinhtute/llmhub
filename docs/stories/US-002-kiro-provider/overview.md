# Overview

## Current Behavior

llmhub supports OAuth-backed routing for Gemini CLI, Claude, Codex, Antigravity,
Kimi, and xAI, plus OpenAI-compatible API-key providers. Kiro credentials from
Kiro IDE or 9router exports are not normalized as first-class auth records, and
Kiro models are not exposed through the merged or provider-specific routing
surfaces.

## Target Behavior

llmhub accepts Kiro auth import input as either a raw refresh token or a
9router Kiro provider connection export, persists normalized Kiro metadata, and
routes OpenAI-shaped chat completion requests through AWS CodeWhisperer
`GenerateAssistantResponse`.

## Affected Users

- Operators importing existing Kiro credentials.
- API clients using llmhub's OpenAI-compatible routing.

## Affected Product Docs

- `README.md`

## Non-Goals

- Implementing first-party Kiro login flows for Builder ID, IDC, Google, or
  GitHub.
- Replacing Kiro's upstream model catalog discovery.
