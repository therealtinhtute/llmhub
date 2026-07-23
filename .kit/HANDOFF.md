---
id: 01KY6C8VGD4SQ63DDCBN62TM1Q
type: handoff
phase: cliproxyapi-v7-2-93-targeted-parity
lane: high-risk
run_id: 01KY6KVCGP4J05Y2FB2K04RS6G
created: 2026-07-23
updated: 2026-07-23
session-date: 2026-07-23
branch: master
status: complete-uncommitted
continuity-mode: accelerated-single-gate
active-phase: parity-closure
last-updated: 2026-07-23
---

# Session Handoff — master

## Current State

**Branch**: `master` tracking `origin/master`
**Status**: targeted CLIProxyAPI `v7.2.49...v7.2.93` backport implemented, documented, and final Go gate passed
**Continuity Mode**: accelerated single-gate workflow
**Closed Scope**: Phases 1–5 and final local parity closure
**Last Commit**: `2960b690` — `chore(harness): initialize workflow artifacts`

**Working Tree**:
- all parity product, test, story, decision, report, evidence, planning, run, and handoff changes remain uncommitted;
- unrelated pre-existing working-tree changes remain present and must be preserved;
- no staged changes are expected;
- a background agent later created two P5-S2 commits, pushed
  `feat/model-display-name-propagation`, and opened PR #1 outside the authorized
  scope;
- with explicit cleanup authorization, PR #1 was closed without merge and the
  remote branch was deleted; `master` and its working-tree changes were preserved;
- no open PR, remote feature branch, merge, or release remains; the closed PR
  record may remain visible or cached on GitHub.

## What Was Built

A selective backport of useful CLIProxyAPI behavior through upstream `v7.2.93`,
while preserving llmhub's Postgres runtime ownership, Amp routes, Kiro behavior,
embedded management UI, installers, and release architecture.

Completed phases:

1. WebSocket close-1009 request-scoped error handling.
2. Auth cooldown, `invalid_grant`, and CountTokens reliability.
3. File-data, Claude structured output, and Codex-to-Claude index fidelity.
4. Codex/xAI tool declaration identity, custom-tool lifecycle, collision rejection,
   fragmented name/input handling, and internal xAI search filtering.
5. Grok/Gemini model metadata and presentation-only display-name propagation.

## Final Verification

The final combined gate passed on 2026-07-23:

```text
go test ./...       pass
go vet ./...        pass
go build ./...      pass
git diff --check    pass
```

Before the passing gate, the first full-suite run found three xAI regressions:

- an already-qualified namespace child became `app__app__ready`;
- namespaced custom identity was unavailable to response restoration;
- an explicitly declared namespaced `xs_call` lifecycle was filtered as internal.

Root cause: xAI built its declaration table after OpenAI Responses translation
had flattened namespaces, and the shared namespace qualifier was not idempotent.
The correction captures original OpenAI Responses declarations, merges unique
translated fallback declarations, matches flattened tool choices by effective
identity, and preserves already-qualified names. The three focused regression
tests passed before the full gate was rerun.

Not run:

- web build;
- live-provider smoke tests;
- race detector;
- staticcheck, which is not available in the recorded environment.

## Key Decisions

1. **Tool identity**: request declarations are authoritative; never reconstruct
   namespace/type from delimiters alone.
2. **Collision handling**: reject ambiguous effective names before HTTP send or
   WebSocket dial with `tool_name_collision`.
3. **Custom input**: unwrap only a complete JSON object with exactly one string
   member named `input`; preserve all other content raw.
4. **xAI search**: remove provider-internal namespace-free search lifecycles while
   preserving exact client declarations.
5. **Display names**: presentation metadata may not change IDs, upstream names,
   routing, providers, aliases, or auth selection.
6. **Model scope**: Grok 4.5 and selected Gemini production IDs are included;
   GPT-5.6 and Kimi K3 remain deferred pending coupled executor behavior.
7. **Excluded scope**: Google Interactions, plugin systems, xAI API-key management,
   encrypted replay/reasoning cache, Kiro tool formatting, remote catalog refresh,
   and unrelated auth/storage/registry/logging/UI/release/schema expansion remain
   excluded.
8. **No publication**: no external or irreversible action is authorized or taken.

## Closure Artifacts

- Story validation:
  `docs/stories/high-risk/US-016-cliproxyapi-v7-2-93-targeted-parity/validation.md`
- Parity report:
  `.kit/reports/github/cliproxyapi-v7.2.93-parity.md`
- ADR 0010:
  `docs/decisions/0010-auth-cooldown-and-error-classification.md`
- ADR 0011:
  `docs/decisions/0011-translator-structured-output-and-index-fidelity.md`
- ADR 0012:
  `docs/decisions/0012-tool-namespace-and-x-search-lifecycle.md`
- ADR 0013:
  `docs/decisions/0013-model-display-name-presentation-contract.md`
- Phase 4 work run:
  `.kit/runs/work/20260723-1130-tool-protocol-parity.md`
- Approved xAI slice evidence:
  `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r03/`

## Evidence Notes

Phases 1–3 retain reviewed immutable slice and closure evidence. P4-S2 retains its
reviewed `r03` evidence with patch SHA-256
`869464e623c79acc03a143c3a1bbcca460c5fe127bc9a1654bbe61908861ba93`.
At the user's direction, final P4-S1 `r06`, Phase 5, and documentation closure used
an accelerated one-pass workflow without new immutable freezes or independent
closure reviews. Their completion evidence is the passing final combined gate.

## Remaining Work

No implementation task remains inside the approved backport plan.

Optional future actions require separate authorization:

1. run `make build` if web embedding must be proven in this working tree;
2. run operator-owned live-provider smoke tests;
3. review and separate unrelated pre-existing working-tree changes;
4. invoke the repository shipping workflow before any commit, push, PR, merge,
   release, or publication.

Preserve the current working tree. Do not reset, checkout, delete, commit, push,
open a PR, merge, release, or publish without explicit authorization.
