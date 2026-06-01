# Exec Plan

## Goal

Add first-class Kiro import and routing support compatible with 9router exports.

## Scope

In scope:

- Raw Kiro refresh token import.
- 9router Kiro OAuth connection JSON import.
- Token refresh and runtime credential update.
- Kiro request translation and event-stream conversion.
- Static fallback model registration.
- Focused tests for import, refresh, translation, executor behavior, and service
  binding.

Out of scope:

- Interactive Kiro login flows.
- Web management UI changes.

## Risk Classification

Risk flags:

- Auth.
- External systems.
- Public contracts.
- Existing behavior.
- Weak proof.

Hard gates:

- Auth.
- External provider behavior.

## Work Phases

1. Record Harness intake, story, and decision.
2. Add Kiro auth parsing and refresh helpers.
3. Wire provider registry and executor binding.
4. Add executor translation and streaming parser.
5. Add docs/config example.
6. Run focused and broad Go verification.

## Stop Conditions

Pause for human confirmation if:

- Kiro upstream authentication requires a login flow in v1.
- Import requires storing raw, unnormalized 9router records.
- Validation has to rely on live Kiro credentials.
