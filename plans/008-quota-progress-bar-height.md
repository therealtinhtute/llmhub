# Plan 008: Unify quota progress bar height across all surfaces

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat c0fff0f..HEAD -- web/src/components/quota/quotaStyles.ts web/src/features/authFiles/components/AuthFileQuotaSection.tsx`
> If either file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `c0fff0f`, 2026-06-18

## Why this matters

The quota progress bar is rendered in two places with different heights:

- **Quota Management page** (`QuotaCard` via `quotaStyles`): `h-1` — 4 px tall, very thin.
- **Auth File detail panel** (`AuthFileQuotaSection`): `h-2` — 8 px tall, 2× taller.

Both show the same data. The inconsistency is immediately visible when a user
switches between views. The `quotaStyles` version also adds `rounded-full` that
the `AuthFileQuotaSection` version omits, so the bars don't even share the same
shape. Settling on `h-1.5` (6 px) with `rounded-full` on both surfaces removes
the discrepancy without making the bar so tall it crowds the row.

## Current state

**Files in scope:**

1. `web/src/components/quota/quotaStyles.ts` — Tailwind class map used by
   `QuotaCard` (Quota Management page). The `quotaBar` key is at line 56.

2. `web/src/features/authFiles/components/AuthFileQuotaSection.tsx` — per-file
   auth detail component. Has a local `quotaStyleMap` object that is **not**
   exported and is only used inside this file. The `quotaBar` key is at line 40.

**Excerpt A — `web/src/components/quota/quotaStyles.ts:54–58`:**
```ts
  quotaBar: 'h-1 bg-secondary rounded-full overflow-hidden',
  quotaBarFill: 'h-full',
  quotaMeta:
    'flex items-center gap-2 text-[12px] text-muted-foreground whitespace-nowrap max-md:justify-start',
```

**Excerpt B — `web/src/features/authFiles/components/AuthFileQuotaSection.tsx:40–41`:**
```ts
  quotaBar: 'h-2 bg-secondary overflow-hidden',
  quotaBarFill: 'h-full',
```

Note: the `AuthFileQuotaSection` version is missing `rounded-full` and uses
`h-2` instead of `h-1`. Both need to become `h-1.5 rounded-full`.

## Commands you will need

All commands run from the `web/` directory.

| Purpose    | Command              | Expected on success                   |
|------------|----------------------|---------------------------------------|
| Type-check | `bun run type-check` | exit 0, no errors                     |
| Lint       | `bun run lint`       | exit 0, or pre-existing warnings only |

## Scope

**In scope** (the only files you should modify):
- `web/src/components/quota/quotaStyles.ts`
- `web/src/features/authFiles/components/AuthFileQuotaSection.tsx`

**Out of scope** (do NOT touch):
- `web/src/features/authFiles/components/QuotaProgressBar.tsx` — this is a
  separate component used inside `AuthFileQuotaSection` for the bar itself; it
  renders `quotaBar` / `quotaBarFill` via the passed-in `styleMap`, so changing
  `AuthFileQuotaSection.tsx`'s `quotaStyleMap.quotaBar` is sufficient.
- `web/src/components/quota/QuotaCard.tsx` — consumes `styles.quotaBar` from
  `quotaStyles`; changing the style map is sufficient.
- Any other file.

## Git workflow

- Branch: `advisor/008-quota-bar-height`
- Single commit: `fix(web): unify quota progress bar height to h-1.5`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Update quotaStyles.ts — change h-1 to h-1.5

Open `web/src/components/quota/quotaStyles.ts`.

Locate line 56 (`quotaBar`):
```ts
  quotaBar: 'h-1 bg-secondary rounded-full overflow-hidden',
```

Change `h-1` to `h-1.5`:
```ts
  quotaBar: 'h-1.5 bg-secondary rounded-full overflow-hidden',
```

(`rounded-full` is already present — do not remove it.)

**Verify**: `grep -n "quotaBar" web/src/components/quota/quotaStyles.ts`
Expected:
```
56:  quotaBar: 'h-1.5 bg-secondary rounded-full overflow-hidden',
```

### Step 2: Update AuthFileQuotaSection.tsx — change h-2 to h-1.5 and add rounded-full

Open `web/src/features/authFiles/components/AuthFileQuotaSection.tsx`.

Locate the `quotaStyleMap` object (around line 29–54). Find the `quotaBar` key
at approximately line 40:
```ts
  quotaBar: 'h-2 bg-secondary overflow-hidden',
```

Change it to:
```ts
  quotaBar: 'h-1.5 bg-secondary rounded-full overflow-hidden',
```

**Verify**: `grep -n "quotaBar" web/src/features/authFiles/components/AuthFileQuotaSection.tsx`
Expected:
```
40:  quotaBar: 'h-1.5 bg-secondary rounded-full overflow-hidden',
```

### Step 3: Type-check

From `web/`:
```
bun run type-check
```
Expected: exit 0. (String-only change; no type errors possible from these edits.)

### Step 4: Lint

From `web/`:
```
bun run lint
```
Expected: exit 0 or pre-existing warnings unchanged.

## Test plan

No automated tests cover Tailwind class strings. Manual visual gate:

1. Run `bun dev` from `web/`.
2. Navigate to Quota Management page → load quota for any card. Confirm the
   progress bar is visibly taller than before (6 px vs 4 px) and has rounded
   ends.
3. Navigate to Auth Files page → open a file detail panel that shows quota.
   Confirm the progress bar height matches the Quota Management page bars.

## Done criteria

- [ ] `bun run type-check` exits 0 (from `web/`)
- [ ] `bun run lint` exits 0 (from `web/`)
- [ ] `grep "quotaBar" web/src/components/quota/quotaStyles.ts` shows `h-1.5 ... rounded-full`
- [ ] `grep "quotaBar" web/src/features/authFiles/components/AuthFileQuotaSection.tsx` shows `h-1.5 ... rounded-full`
- [ ] Only the two in-scope files are modified (`git diff --name-only`)
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

Stop and report back if:

- The `quotaBar` values in either file do not match the excerpts above.
- Type-check or lint produce errors new to this change.
- The fix appears to require touching a file outside the in-scope list.

## Maintenance notes

- `AuthFileQuotaSection` maintains its own `quotaStyleMap` object (not imported
  from `quotaStyles`). If the two style maps diverge again in the future, a
  follow-up should extract a shared constant. That refactor is out of scope here.
- The `QuotaProgressBar` component at
  `web/src/features/authFiles/components/QuotaProgressBar.tsx` receives
  `quotaBar` and `quotaBarFill` class strings via props — it does not hardcode
  heights itself. Any future change to bar height only requires updating the
  two style maps in this plan.
