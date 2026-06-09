# Exec Plan

## Goal

Port Kiro-Go profile ARN and upstream overage behavior into llmhub while
preserving the current Kiro import/runtime boundary.

## Scope

In scope:

- Kiro profile ARN resolver via `ListAvailableProfiles`.
- Overage-aware quota exhaustion and selector fallback.
- Management-only upstream overage toggle.
- Kiro retry body reapply after 401 refresh.
- Focused Go tests, web type/build checks, and docs updates.

Out of scope:

- New database schema.
- New import schema.
- Global allow-overage config.
- Live Kiro account verification.

## Risk Classification

Risk flags:

- Auth
- External systems
- Public contracts
- Existing behavior
- Weak proof
- Multi-domain

Hard gates:

- External provider behavior.

## Work Phases

1. Inspect current Kiro executor, quota parser, selector, management API, and UI renderer.
2. Implement backend resolver, overage toggle, quota semantics, and retry body fix.
3. Add focused mocked-provider tests.
4. Add UI API and toggle wiring.
5. Update ADR and validation notes.
6. Run focused and broad verification.

## Stop Conditions

Pause for human confirmation if:

- Kiro-Go behavior conflicts with the approved default decisions.
- A database migration becomes necessary.
- Existing auth disable behavior must change outside Kiro quota/overage.
