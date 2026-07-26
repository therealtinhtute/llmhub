---
id: 01KYER6XENTRZYAT2WGV713P36
type: check
phase: quota-usage-reset
lane: normal
mode: full
run_id: 01KYEN0YEGNYYPX1KCWETAD1AZ
proof_links: [{command: "cd web && bun run type-check", output_ref: "inline, this report", artifact_path: "web/src/components/quota/"}, {command: "cd web && bun run lint", output_ref: "inline, this report", artifact_path: "web/src/components/quota/"}, {command: "grep -rn 'reset-quota' web/src", output_ref: "inline, this report", artifact_path: "web/src/services/api/quota.ts"}, {command: "make build-web && make build", output_ref: "inline, this report", artifact_path: "internal/managementasset/static/management.html"}, {command: "git diff --check", output_ref: "inline, this report", artifact_path: "n/a"}, {command: "cd web && bun run test:run", output_ref: "inline, this report", artifact_path: "web/src/utils/__tests__/"}]
created: 2026-07-26
updated: 2026-07-26
---

# CHECK REPORT

Run ID: check-20260726-1445-quota-usage-reset
Scope: full
Artifact Alignment: aligned
Review Verdict: REQUEST CHANGES
Phase: quota-usage-reset
Spec: .kit/planning/SPEC.md
Plan: .kit/planning/phases/quota-usage-reset/quota-usage-reset-PLAN.md
Cook Run: .kit/runs/work/20260726-0725-quota-usage-reset.md
Created At: 2026-07-26 14:45

## Gate Evidence
- tests: `cd web && bun run test:run` → pass (4 files, 60 tests — pre-existing `utils/__tests__/*`; none cover the changed reset behavior, see Findings/Major)
- types: `cd web && bun run type-check` → pass (`tsc --noEmit`, no output)
- lint: `cd web && bun run lint` → pass (0 errors, 8 pre-existing warnings unrelated to this diff — verified via `git diff --stat` none fall on touched lines)
- build: `make build-web && make build` → pass (`dist/index.html` 1,918.91 kB / gzip 556.43 kB; `internal/managementasset/static/management.html` regenerated; `go build` exit 0)
- secrets scan: `git diff HEAD | grep '^+' | grep -iE "(password|secret|token|api_key|private_key)"` → no matches
- `git diff --check` → pass, no output

## Artifact Alignment
- status: aligned
- notes:
  - **Spec coverage**: diff implements exactly SPEC.md's W1 ("wire the existing `POST /v0/management/reset-quota` to a panel action, frontend only") — no backend files touched, matches the Amendment 2026-07-25 carve-out and Out-of-Scope exception.
  - **Boundary compliance**: changed files (`web/src/services/api/quota.ts` [new], `web/src/services/api/index.ts`, `web/src/components/quota/{QuotaCard,QuotaSection,AllQuotaSection}.tsx`, `web/src/i18n/locales/{en,vi}.json`) are a subset of T1's `touches` list in `quota-usage-reset-PLAN.md`. `useQuotaLoader.ts`, `QuotaPage.tsx` listed as allowed-but-unused — correctly untouched, no forced edits. No Go file, `sdk/cliproxy/auth/`, or the `ResetQuota` handler/route touched — respects `avoid`.
  - **Decision/context alignment**: confirm-before-reset (✅ `useConfirmationStore`, `variant: 'danger'`), refetch-not-patch on success (✅ `refreshQuotaForFile`/`refreshQuotaForItem`, response `models` array unused), all four documented failure codes surfaced distinctly (✅ 400/404/500/503 + generic fallback), `authIndex` never coerced — button disabled via `normalizeAuthIndex` check, not sent empty/coerced (✅). No rejected option reintroduced.
  - **Lane resolution note**: `SPEC.md` frontmatter `lane: high-risk` belongs to that SPEC's original phase (`phase: websocket-message-too-big`, id `01KY1T7GGY3DBEW2NAAV620VCK`) — the 2026-07-25 Amendment that added `quota-usage-reset` does not set a lane for it. `quota-usage-reset-CONTEXT.md:5` explicitly declares `Lane: normal` for this phase. Used `normal` (the phase-specific, most-recently-authored source) rather than the top-level SPEC frontmatter, which describes an unrelated phase's risk profile. Flagging this as a process gap: the Amendment should have stamped an explicit lane for the new phase to avoid this ambiguity next time.
  - **Execution proof alignment / Validation Matrix (lane: normal)**: `unit` = required, `integration` = optional, `e2e` = n-a, `manual-check` = optional, `command-output` = required.
    - `command-output`: satisfied — type-check, lint, build, grep all have real captured output above.
    - `unit`: **not satisfied**. No test covers the new reset behavior (`resetQuotaForFile`, `resetQuotaForItem`, `QuotaCard`'s reset button gating/disabled logic). `cd web && bun run test:run` passes but only exercises pre-existing `utils/__tests__/*` files, none of which touch `components/quota/`. This repo has no React component-test infra today (`jsdom` env is configured in `vitest.config.ts`, but no `@testing-library/react`/`@testing-library/jest-dom` dependency exists, and no prior component test exists anywhere in `web/src/components/`) — closing this gap is an infra decision (new dependency + first component-test pattern for the repo), not a same-wave targeted fix, so it was not done unilaterally here.
    - `e2e`: n-a for this lane — the missing browser/panel smoke noted in the Wave 2 (T2) run-artifact entry is **not** a gate blocker under this lane's matrix, independent of the `unit` gap above.

## Findings

### Critical
- none

### Major
- **Missing required `unit` proof for lane `normal`.** `internal/api/handlers/management/quota.go` calling contract is well understood, and the frontend logic (`resetQuotaForFile`/`resetQuotaForItem` in `web/src/components/quota/{QuotaSection,AllQuotaSection}.tsx`) is straightforward, but per the Harness Validation Matrix a `normal`-lane change requires `unit` evidence and none exists for this behavior. This is a hard-rule gate item (`.kit/docs/playbooks/check.md` Step 4.3): "A required cell with no matching evidence ⇒ gate FAIL, name the exact missing evidence class, and stop." **Remediation paths** (pick one): (a) add `@testing-library/react` + `@testing-library/jest-dom` as dev dependencies and write a focused unit test for the reset flow (happy path + one error-status path) — first component test in the repo, a real infra decision; (b) a human records `zharness intervention --verdict-id {this check's id, once recorded} --reason "..."` accepting the gap for this phase; this playbook does not self-override it.

### Minor / Suggestions
- `resolveResetErrorMessageKey` (status → i18n key switch) is duplicated verbatim between `QuotaSection.tsx` and `AllQuotaSection.tsx`. Consistent with this pair's existing convention of duplicating section-local helpers (e.g. `useQuotaPagination`) rather than sharing a module, so not flagged as a blocker — advisory only if a third consumer appears.
- 💡 doc debt: none — no new cross-file invariant beyond what `web/CLAUDE.md` (i18n keys in both locale files) and this phase's `CONTEXT.md` already capture.

## Next Action
- `zharness check record --verdict REQUEST_CHANGES --run-id 01KYEN0YEGNYYPX1KCWETAD1AZ` → recorded as check id `01KYER6XENTRZYAT2WGV713P36`.
- **Human intervention recorded**: `zharness intervention --verdict-id 01KYER6XENTRZYAT2WGV713P36` → intervention id `01KYER74YAX08RYM65EHKKA8FC`. User explicitly chose to ship W1 as-is rather than add component-test infra or self-test against the live credential (decision made via `AskUserQuestion`, 2026-07-26). This overrides the gate FAIL above by human judgment, per Step 5 — the playbook itself never waives it.
- W1 (`quota-usage-reset` Waves 1-2) is shippable. Ready for `git`/commit or `handoff` — not run automatically.
- Do not start Wave 3/4 (backend usage-reset endpoint) until independently gated — this intervention covers W1 only.
