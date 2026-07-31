# Triage Rubric

Turns a ledger of commits into a scope decision. Triage groups by capability, never by
commit — a plan built per commit produces incoherent phases.

## Step 1 — group into capability slices

Merge ledger rows into slices that share one observable outcome. A slice is portable on
its own and testable on its own.

Good slices: "Codex Live sideband relay", "weighted credential scheduling", "Claude tool
schema normalization".
Bad slices: "v7.2.104 changes", "translator fixes", "misc".

Record for each slice: source commits, touched local surfaces, and the gap classes it
covers.

## Step 2 — score each slice

| Axis | 1 | 2 | 3 |
| --- | --- | --- | --- |
| **Value** | Cosmetic, unused path, upstream-only concern | Fixes a real defect or unblocks a client | New capability users asked for, or fixes a live failure |
| **Blast radius** | One file behind an existing interface | One subsystem, additive contract | Cross-cutting: auth, routing, storage, or a public contract |
| **Architectural fit** | Inverts a local invariant | Needs re-expression behind local interfaces | Drops in behind existing interfaces |

Priority = Value + Fit − Blast radius. Rank slices; do not treat the number as a verdict.

## Step 3 — apply the invariant gate

A slice that inverts any local invariant is `exclude`, regardless of score. For llmhub the
standing invariants are:

1. Postgres is the authoritative runtime store for config, credentials, and state — no
   feature may read authoritative state from local YAML or files.
2. Amp routing, Kiro support, Gemini CLI paths, and provider-specific route semantics stay
   behavior-compatible.
3. The embedded management web keeps llmhub's own layout, tokens, components, and i18n —
   upstream visual design never replaces it.
4. Public SDK boundaries under `sdk/` stay backward compatible unless a versioned additive
   contract is documented.
5. llmhub branding and release contracts are not upstream's.

Verify the list against `CLAUDE.md` and `docs/ARCHITECTURE.md` each cycle rather than
trusting this copy — invariants change.

## Step 4 — assign lanes

| Lane | Use when |
| --- | --- |
| `tiny` | Docs, model metadata, a narrow single-file fix with no contract change |
| `normal` | One subsystem, additive, existing test surface covers it |
| `high-risk` | Public contract, auth, storage, cross-cutting runtime, or a new subsystem |

Upstream parity work spanning more than one subsystem is `high-risk` by default. See
`docs/FEATURE_INTAKE.md` for the authoritative lane definitions.

## Step 5 — present the decision

Use `AskUserQuestion`. Scope is always the user's call.

- Present slices grouped as **include** / **exclude** / **defer** with the recommendation
  first.
- Name the invariant behind every `exclude`. An exclusion without a stated reason is a
  silent drop.
- State the cost of each `include` in surfaces touched, not in hours.
- Never bundle an unrelated slice into an approved group to "save a round trip".

## Step 6 — record it

```bash
python3 .claude/skills/upstream/scripts/upstream_sync.py policy \
  --slug {slug} \
  --strategy targeted-semantic-ports \
  --include codex-live-full postgres-backed-controls \
  --exclude plugin-platform file-authoritative-runtime wholesale-source-merge
```

The recorded policy is the authority the plan cites. If the plan and the policy disagree,
the policy is wrong — rerun `policy`, do not edit the plan to match.

## Anti-patterns

- **Porting everything because it is there.** A fork that tracks upstream commit-for-commit
  is a fork with no reason to exist. Default to `reject` and let value argue upward.
- **Deferring without a trigger.** `defer` needs the condition that pulls it back in.
- **Scoring after the decision.** If the slice list was written to justify a conclusion, the
  rubric added nothing.
- **Treating an upstream refactor as a feature.** Refactors carry no user-visible outcome;
  port the behavior, not the shape.
