# Execution Plan

## Goal

Complete the approved five-phase targeted parity backport while preserving the
existing llmhub runtime and without publishing repository state.

## Execution Units

- 1 Phase 1 closure.
- 9 product implementation slices.
- 4 later-phase gates.
- 4 later-phase documentation closures.
- 1 final evidence gate.

Detailed ownership and command contracts live in
`.kit/plans/2026-07-22-cliproxyapi-v7.2.93-fanout/plan.md`.

## Phase Order

1. Close the existing WebSocket message-too-big implementation.
2. Implement auth cooldown, invalid-grant, and CountTokens reliability.
3. Implement translator content fidelity in three isolated slices.
4. Implement Codex/xAI tool-protocol parity in two isolated slices.
5. Implement model metadata and display-name compatibility in two isolated
   slices.
6. Run the final full test, vet, web-build, binary-build, fingerprint, and
   documentation evidence gate.

## Isolation and Apply Rules

- Parallel agents never edit the main working tree.
- Each slice has an exact allowlist and immutable evidence revision.
- Reviewed patches are applied to the main tree serially in declared order.
- A blocked slice or gate reverses current-phase patches in reverse order.
- The imported Phase 1 WIP is never destructively reset; its reversibility is
  proved only in scratch.

## Stop Conditions

Stop rather than expand scope if work requires a forbidden provider, storage,
schema, UI, installer, release, or publication change; if an unexpected path is
modified; if a required test fails; if a Critical/Major review finding remains;
or if a patch cannot be applied and reversed cleanly.
