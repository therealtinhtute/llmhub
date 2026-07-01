# Plan 007: Add rounded corners to quota cards

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat c0fff0f..HEAD -- web/src/components/quota/quotaStyles.ts`
> If the file changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, treat it as
> a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `c0fff0f`, 2026-06-18

## Why this matters

Every quota card on the Quota Management page has sharp square corners
(`bg-background border border-border p-3 flex flex-col gap-2` — no `rounded-*`
class). Every other surface in the design system (AppCard, modals, badges,
buttons) uses `rounded-md` or larger. The mismatch makes quota cards feel like
a legacy table row dropped into a card layout. One class addition removes the
visual inconsistency at zero functional cost.

## Current state

**File in scope:**
- `web/src/components/quota/quotaStyles.ts` — the central Tailwind class map
  consumed by every quota component (`QuotaCard`, `QuotaSection`,
  `AllQuotaSection`). The `fileCard` key at line 97–98 defines the card shell.

**Relevant excerpt (`web/src/components/quota/quotaStyles.ts:96–101`):**
```ts
  // File card
  fileCard:
    'bg-background border border-border p-3 flex flex-col gap-2',
  cardHeader: 'flex items-center gap-2 min-h-7',
  typeBadge: 'px-2.5 py-1 rounded-xl text-[12px] font-semibold whitespace-nowrap shrink-0',
  fileName: 'text-sm font-semibold text-foreground break-all leading-[1.4]',
```

The `typeBadge` already has `rounded-xl` — a rounded-md card shell won't
conflict; the badge sits inside the card.

**Comparable UI for convention reference:**
`web/src/components/ui/AppCard.tsx` — uses `rounded-md` for the card shell.
Match that value.

## Commands you will need

All commands run from the `web/` directory.

| Purpose    | Command                   | Expected on success                      |
|------------|---------------------------|------------------------------------------|
| Type-check | `bun run type-check`      | exit 0, no errors                        |
| Lint       | `bun run lint`            | exit 0, or only pre-existing warnings    |

## Scope

**In scope** (the only file you should modify):
- `web/src/components/quota/quotaStyles.ts`

**Out of scope** (do NOT touch):
- `web/src/components/quota/QuotaCard.tsx` — consumes `styles.fileCard`; no change needed there.
- `web/src/features/authFiles/components/AuthFileQuotaSection.tsx` — has its own local `quotaStyleMap` that does not use `fileCard`; that component is handled by a separate plan (Plan 008).
- Any other file.

## Git workflow

- Branch: `advisor/007-quota-card-rounded-corners`
- Single commit: `fix(web): add rounded-md to quota cards`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add `rounded-md` to the `fileCard` class

Open `web/src/components/quota/quotaStyles.ts`.

Locate line 97–98:
```ts
  fileCard:
    'bg-background border border-border p-3 flex flex-col gap-2',
```

Change it to:
```ts
  fileCard:
    'bg-background border border-border rounded-md p-3 flex flex-col gap-2',
```

**Verify**: `grep -n "fileCard" web/src/components/quota/quotaStyles.ts`
Expected output includes `rounded-md`:
```
98:    'bg-background border border-border rounded-md p-3 flex flex-col gap-2',
```

### Step 2: Type-check

From `web/`:
```
bun run type-check
```
Expected: exit 0, zero errors. (This is a string-only change; no type errors possible. If any appear, they are pre-existing and must be reported, not fixed here.)

### Step 3: Lint

From `web/`:
```
bun run lint
```
Expected: exit 0, or the same warnings that existed before this change.

## Test plan

No automated tests exist for Tailwind class strings in this repo (the test
suite under `web/src/utils/__tests__/` covers pure utility functions — not
component rendering). Manual visual verification is the appropriate gate here:

1. Run `bun dev` from `web/` (proxies API to `localhost:9090`).
2. Navigate to the Quota Management page.
3. Confirm quota cards have visibly rounded corners.
4. Confirm the type badge, progress bars, and filename text inside each card
   are unaffected.

## Done criteria

- [ ] `bun run type-check` exits 0 (from `web/`)
- [ ] `bun run lint` exits 0 (from `web/`)
- [ ] `grep "fileCard" web/src/components/quota/quotaStyles.ts` shows `rounded-md` in the value
- [ ] Only `web/src/components/quota/quotaStyles.ts` is modified (`git diff --name-only`)
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

Stop and report back if:

- The `fileCard` key at line 97–98 does not match the excerpt above (file has drifted).
- Type-check or lint produce errors that were not present before this change.
- The fix appears to require modifying any file outside the in-scope list.

## Maintenance notes

- The `fileCard` class is also used in per-provider card classes (they are
  passed as `cardClassName` alongside `fileCard` in `QuotaCard`). The per-
  provider classes add gradient background-image overrides but do not set
  `border-radius` — so `rounded-md` from `fileCard` will apply to all of them.
- If a provider ever needs a different corner radius (e.g., flush-top cards in
  a list), override with `rounded-t-none` on the per-provider class, not by
  removing the base `rounded-md`.
- The companion visual polish plans (008–011) are independent and can run in
  any order alongside this one.
