---
id: 01KY6KVCGP4J05Y2FB2K04RS6G
type: run
phase: tool-protocol-parity
lane: high-risk
mode: full
plan_id: none
trace_ids: []
created: 2026-07-23
updated: 2026-07-23
---

# COOK RUN — Tool Protocol Parity

Run ID: 01KY6KVCGP4J05Y2FB2K04RS6G
Mode: full
Status: running
Spec: `.kit/planning/SPEC.md`
Roadmap: `.kit/planning/ROADMAP.md`
Phase: `tool-protocol-parity`
Plan: `.kit/planning/phases/tool-protocol-parity/tool-protocol-parity-PLAN.md`
Started At: 2026-07-23 11:30

## Preflight

- scope drift: no;
- working tree note: approved Phase 1–3 changes and harness/evidence artifacts form the governed Phase 3 base; no Phase 4 allowed-surface product files were already modified;
- required artifacts present: yes — locked SPEC, ready CONTEXT, and ready PLAN;
- selected phase: `tool-protocol-parity`, explicitly approved at the Phase 3 boundary;
- harness note: `query state` still reports the historical entry phase, so the explicit `zharness next phase tool-protocol-parity --json` result is authoritative for this run;
- publication boundary: no commit, push, PR, merge, release, or publication.

## Wave / Task Log

### Wave 1 — Parallel isolated slices

#### P4-S1 — Codex declaration table and round trips

- status: corrected `r05` implementation PASS and independent immutable review active after four preserved rejected revisions;
- refreshed immutable base: `.kit/evidence/cliproxyapi-v7.2.93-backport/phases/P4-BASE/r02/` (`prepared_full_tree 851498d3cde20b09e889cc2f45e7feef70a8c241`, `allowlist_tree de5c44884e286229d15ffb61d4be2733472b46fa`);
- rejected `r01`: WebSocket bypassed collision/restoration enforcement and custom stream events retained stale function types and `fc_*` identities;
- rejected `r02`: frozen baseline metadata did not match the prepared Phase 3 tree;
- rejected `r03`: patch `481141374bc47c88659c499909c0818f84c6bbe0c2ec1f9e6ae79885d89b064c`, result tree `acbe9976fd3fc8d77fc45c2c4c7a0a18f32e357e`; independent review reproduced a high-severity stream/terminal mismatch for brace-prefixed raw input, incomplete wrappers, and raw JSON objects containing an `input` member;
- retained boundaries: structured pre-network collision errors, exact HTTP/WebSocket namespace restoration, consistent `ctc_*` lifecycle identity, arbitrary valid-wrapper fragmentation, `mcp__` stability, and close-1009 behavior must remain passing;
- rejected `r04`: patch `c1bddd3e590a02b1249a6ab7f7b57655a9369aa69ac6881487930eb650066300`, result tree `a04532ab00573a16269a15b25343aaa0901a1853`; integrity and frozen suites pass, but independent review found three blocking defects: shared normalization regressed approved xAI namespace-child request shape, WebSocket nonstream dropped prior output-item completions when terminal output was empty, and same-sequence custom-delta replay duplicated content;
- corrected `r05`: patch `b2ca393ea5ec36e0f8e17987c325a2a4496b09c1e98b6f1cd170f72043209b42`, result tree `7b754f67c75c8932ee7b3552ef789ccf6842e31f`; consumer-neutral shared translation, Codex-owned namespace normalization, request-local WebSocket item merging, and stable event replay suppression pass focused/full/race/vet/build/diff and combined approved P4-S2 regressions;
- independent `r05` review: active;
- diagnostic note: package-wide executor race still reproduces the pre-existing out-of-scope Antigravity cleanup race; all required P4-S1 and combined P4-S2 focused race suites pass.

#### P4-S2 — xAI tools and internal-search lifecycle

- status: DONE;
- prior blocker: `BLOCKED_VERIFICATION` after r01 and the permitted first targeted correction r02 each failed independent adversarial review;
- authorization: explicit approval received to attempt one additional isolated regression-first revision;
- immutable Phase 3 base: `.kit/evidence/cliproxyapi-v7.2.93-backport/phases/P4-BASE/r01/` (`allowlist_tree c3e3d40efc9d0144b247706af7d45b287b5c294b`);
- rejected revision r01: `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r01/`, patch `7785dd387496188785c5058fb4ab8e3fab25508c0b571264e1ffb803473bc3cd`, result tree `889a4bfa55d54d75a5fe00343bc330871184bef2`; same-name declaration leaked an internal `xs_call` lifecycle;
- rejected revision r02: `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r02/`, patch `80f3484c8d12b5337792e87f45628ec245389ab4d3f533ed21ce2d5d353d95ae`, result tree `7fa63d91f072c7d324f9ea15bf2a3bf6fbabcb1c`; prefix classification suppressed an exact declared namespaced client identity;
- frozen revision r03: `.kit/evidence/cliproxyapi-v7.2.93-backport/slices/P4-S2-xai-tool-lifecycle/r03/`, patch `869464e623c79acc03a143c3a1bbcca460c5fe127bc9a1654bbe61908861ba93`, result tree `3490ead86db86c3a38e4d70173f012d5d948fd98`;
- r03 verification: combined three-boundary regression, focused identity/search suite, full focused executor suite, full executor package, race-focused suite, exact scope, and forward/reverse proof passed;
- independent r03 review: APPROVED with no findings; exact two-file scope, both rejected boundaries, full identity matrix, focused/full/race suites, and forward/reverse proofs reproduced.

### Wave 2 — Apply, gate, and close

- status: pending;
- apply order: P4-S1, then P4-S2;
- gate: focused translator/executor tests, full Go tests, vet, build, and whitespace validation.

## Summary

- passed tasks: none yet;
- blocked tasks: none;
- unresolved concerns: none.

## Next Recommended Action

- Freeze the immutable Phase 3 base, dispatch P4-S1 and P4-S2 in isolated worktrees, and independently review each result before application.
