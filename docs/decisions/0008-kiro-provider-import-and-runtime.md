# 0008 Kiro Provider Import And Runtime

Date: 2026-06-01

## Status

Accepted

## Context

Kiro support changes external provider behavior and introduces a new credential
import format. llmhub needs to accept 9router exports without storing their
schema as the runtime contract.

## Decision

Normalize Kiro import input at the boundary into llmhub auth metadata and use a
dedicated Kiro executor for token refresh, request translation, and stream
conversion.

## Alternatives Considered

1. Treat Kiro as a generic OpenAI-compatible upstream. Rejected because Kiro uses
   AWS CodeWhisperer `GenerateAssistantResponse`, not OpenAI wire format.
2. Store 9router connection JSON directly. Rejected because it would leak an
   external app schema into llmhub runtime behavior.

## Consequences

Positive:

- Existing 9router Kiro exports can be imported without manual conversion.
- Runtime routing uses the same auth manager and model registry surfaces as
  other OAuth providers.

Tradeoffs:

- Kiro's unofficial upstream protocol can drift and requires mocked tests plus
  clear error surfacing.

## Follow-Up

- Consider a first-party Kiro login flow after the import-only runtime path is
  stable.
