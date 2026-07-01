# Plan 010: Replace quota card loading text with skeleton shimmer

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat c0fff0f..HEAD -- web/src/components/quota/QuotaCard.tsx`
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

While quota data is loading, the `QuotaCard` shows a plain text string
(`t('${i18nPrefix}.loading')`) inside the quota section. There is no visual
indication of what shape of content is about to appear. The `Skeleton` component
(`web/src/components/ui/skeleton.tsx`) already exists in the project and is
imported nowhere in the quota components — it's unused. Replacing the loading
text with a 3-row skeleton (label / percent line + bar, repeated) gives the page
a polished "content is coming" feel and avoids jarring layout shifts when real
data arrives.

## Current state

**Files in scope:**

1. `web/src/components/quota/QuotaCard.tsx` — the card shell component. The
   loading branch is at line 128–129.

2. `web/src/components/ui/skeleton.tsx` — the Skeleton primitive. Read-only
   reference; do not modify.

**Excerpt A — `web/src/components/quota/QuotaCard.tsx:111–163` (abbreviated):**
```tsx
  return (
    <div className={`${styles.fileCard} ${cardClassName}`}>
      <div className={styles.cardHeader}>
        ...
      </div>

      <div className={styles.quotaSection}>
        {quotaStatus === 'loading' ? (
          <div className={styles.quotaMessage}>{t(`${i18nPrefix}.loading`)}</div>   // ← line 128-129, REPLACE THIS
        ) : quotaStatus === 'idle' ? (
          ...
        ) : ...}
      </div>
    </div>
  );
```

**Excerpt B — `web/src/components/ui/skeleton.tsx` (unchanged reference):**
```tsx
import { cn } from "@/lib/utils"

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("animate-pulse rounded-md bg-accent", className)}
      {...props}
    />
  )
}

export { Skeleton }
```

**Existing imports in `QuotaCard.tsx` (from the top of the file):**
```tsx
import { useTranslation } from 'react-i18next';
import type { ReactElement, ReactNode } from 'react';
import type { TFunction } from 'i18next';
import type { AuthFileItem, ResolvedTheme, ThemeColors } from '@/types';
import { TYPE_COLORS } from '@/utils/quota';
import { quotaStyles as styles } from './quotaStyles';
import type { QuotaStyleMap } from './quotaStyles';
```
`Skeleton` is not yet imported. Add it.

**Convention reference:** `web/src/components/ui/AppCard.tsx` or any other
component using `Skeleton` — look for `import { Skeleton } from '@/components/ui/skeleton'`.
(As of the commit this plan was written, no component imports it — this will be
the first usage. The import path alias is `@/components/ui/skeleton`.)

## Commands you will need

All commands run from the `web/` directory.

| Purpose    | Command              | Expected on success                   |
|------------|----------------------|---------------------------------------|
| Type-check | `bun run type-check` | exit 0, no errors                     |
| Lint       | `bun run lint`       | exit 0, or pre-existing warnings only |

## Scope

**In scope** (the only file you should modify):
- `web/src/components/quota/QuotaCard.tsx`

**Out of scope** (do NOT touch):
- `web/src/components/ui/skeleton.tsx` — only import it; do not modify.
- `web/src/components/quota/quotaStyles.ts` — the loading state does not need a new style key; use inline Tailwind classes for the skeleton layout.
- `web/src/features/authFiles/components/AuthFileQuotaSection.tsx` — has its own loading state; that surface is a separate concern not in scope.
- Any other file.

## Git workflow

- Branch: `advisor/010-quota-skeleton-loading`
- Single commit: `fix(web): replace quota card loading text with skeleton shimmer`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add Skeleton import to QuotaCard.tsx

Open `web/src/components/quota/QuotaCard.tsx`.

After the existing import block (around line 11), add:
```tsx
import { Skeleton } from '@/components/ui/skeleton';
```

Place it after the relative imports, before or after the `quotaStyles` import —
match the existing import ordering style in the file (relative imports last).

**Verify**: `grep -n "Skeleton" web/src/components/quota/QuotaCard.tsx`
Expected: one line showing the import.

### Step 2: Replace the loading text with a skeleton block

Still in `web/src/components/quota/QuotaCard.tsx`.

Locate the loading branch inside the `quotaSection` div (around line 128–129):
```tsx
        {quotaStatus === 'loading' ? (
          <div className={styles.quotaMessage}>{t(`${i18nPrefix}.loading`)}</div>
```

Replace `<div className={styles.quotaMessage}>{t(`${i18nPrefix}.loading`)}</div>`
with the following skeleton layout (three rows mimicking a quota entry each):

```tsx
          <div className="flex flex-col gap-2" aria-label={t(`${i18nPrefix}.loading`)} role="status">
            {[0, 1, 2].map((i) => (
              <div key={i} className="flex flex-col gap-1">
                <div className="flex items-center justify-between gap-2">
                  <Skeleton className="h-3 w-2/3" />
                  <Skeleton className="h-3 w-10" />
                </div>
                <Skeleton className="h-1.5 w-full rounded-full" />
              </div>
            ))}
          </div>
```

The result for the loading branch should read:
```tsx
        {quotaStatus === 'loading' ? (
          <div className="flex flex-col gap-2" aria-label={t(`${i18nPrefix}.loading`)} role="status">
            {[0, 1, 2].map((i) => (
              <div key={i} className="flex flex-col gap-1">
                <div className="flex items-center justify-between gap-2">
                  <Skeleton className="h-3 w-2/3" />
                  <Skeleton className="h-3 w-10" />
                </div>
                <Skeleton className="h-1.5 w-full rounded-full" />
              </div>
            ))}
          </div>
```

**Verify**: `grep -n "animate-pulse\|aria-label\|Skeleton" web/src/components/quota/QuotaCard.tsx`
Expected: the import line and lines inside the loading branch.

### Step 3: Type-check

From `web/`:
```
bun run type-check
```
Expected: exit 0. If you see a type error about `Skeleton` props, confirm the
import path is `@/components/ui/skeleton` (lowercase) and the exported name is
`Skeleton` (capital S) — both as shown in `skeleton.tsx`.

### Step 4: Lint

From `web/`:
```
bun run lint
```
Expected: exit 0 or pre-existing warnings only.

## Test plan

No automated tests cover component rendering in this codebase at the time of
this plan. Manual visual gate:

1. Run `bun dev` from `web/`.
2. Navigate to the Quota Management page.
3. On a credential card in idle state, click "Load quota" (or the equivalent
   idle CTA for that provider).
4. While the request is in flight, confirm the card body shows 3 skeleton rows
   (pulsing grey bars) instead of the loading text.
5. After data loads, confirm the skeleton disappears and real quota rows appear.
6. For providers where quota loads very fast: throttle the network to "Slow 3G"
   in DevTools to make the skeleton visible.

## Done criteria

- [ ] `bun run type-check` exits 0 (from `web/`)
- [ ] `bun run lint` exits 0 (from `web/`)
- [ ] `grep "Skeleton" web/src/components/quota/QuotaCard.tsx` shows both the import and usage
- [ ] `grep "quotaMessage.*loading\|i18nPrefix.*loading" web/src/components/quota/QuotaCard.tsx` returns NO match (the old text div is gone)
- [ ] Only `web/src/components/quota/QuotaCard.tsx` is modified (`git diff --name-only`)
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

Stop and report back if:

- The loading branch at line 128–129 does not match the excerpt above.
- The `Skeleton` component at `web/src/components/ui/skeleton.tsx` does not
  export a named export `Skeleton`.
- Type-check fails because the `cn` import or `animate-pulse` class is not
  recognized — that would indicate a build configuration issue, not a code issue.
- The fix appears to require touching a file outside the in-scope list.

## Maintenance notes

- The skeleton uses `h-1.5` for the bar, which matches the target height from
  Plan 008 (quota bar unification). If Plan 008 has not yet landed and the bar
  height is still `h-1`, consider using `h-1` in the skeleton bar too for
  consistency until Plan 008 lands — update it afterwards.
- The `role="status"` + `aria-label` ensures screen readers announce that
  loading is in progress. The `aria-label` reuses the existing i18n key
  `${i18nPrefix}.loading` (already translated in `en.json` and `vi.json`).
- The 3-row skeleton is an approximation. Some providers show 1 row (Kimi),
  others show 4+ (Codex with multiple windows). The skeleton does not need to
  exactly match — its purpose is to signal "content is loading" with the right
  general shape.
