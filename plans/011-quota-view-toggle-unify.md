# Plan 011: Unify view-mode toggle in QuotaPage to match QuotaSection

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat c0fff0f..HEAD -- web/src/pages/QuotaPage.tsx web/src/components/quota/quotaStyles.ts`
> If either file changed since this plan was written, compare the "Current state"
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

The Quota Management page has two different visual treatments for the same
"Paged / All" view-mode toggle control:

- **QuotaPage.tsx (All-tab header, lines 152–176)**: raw `<button>` elements
  inside a plain `div` with a rectangle border (`border border-border/70
  bg-muted/60 overflow-hidden`). No rounded corners, no design-system component.
- **QuotaSection.tsx (per-provider tab header, lines 283–310)**: `Button`
  components from `@/components/ui/Button` inside the `styles.viewModeToggle`
  wrapper (rounded pill, subtle inset shadow).

A user switching between the "All" tab and a provider-specific tab sees the
toggle change shape. Updating `QuotaPage` to use the same `styles.viewModeToggle`
class and `Button` component removes this inconsistency in about 15 lines of
change.

## Current state

**Files in scope:**

1. `web/src/pages/QuotaPage.tsx` — the top-level page component. The toggle
   block is at lines 152–176. The `quotaStyles` import is already present
   at line 21 (`import { quotaStyles as styles } from
   '@/components/quota/quotaStyles'`). `Button` is NOT yet imported.

2. `web/src/components/quota/quotaStyles.ts` — read-only reference for class
   strings. Do not modify.

**Excerpt A — `web/src/pages/QuotaPage.tsx:152–176` (REPLACE THIS):**
```tsx
          <div className="flex items-center gap-1 shrink-0 pb-px mb-1">
            <div className="inline-flex items-center border border-border/70 bg-muted/60 overflow-hidden">
              <button
                type="button"
                className={`px-2.5 py-1 text-[11px] font-semibold transition-colors ${
                  viewMode === 'paged'
                    ? 'bg-background text-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
                onClick={() => setViewMode('paged')}
              >
                {t('auth_files.view_mode_paged')}
              </button>
              <button
                type="button"
                className={`px-2.5 py-1 text-[11px] font-semibold transition-colors ${
                  viewMode === 'all'
                    ? 'bg-background text-foreground'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
                onClick={() => setViewMode('all')}
              >
                {t('auth_files.view_mode_all')}
              </button>
            </div>
```

**Excerpt B — `web/src/components/quota/quotaStyles.ts:43–47` (class strings to reuse):**
```ts
  viewModeToggle:
    'inline-flex gap-1 items-center p-[3px] rounded-full bg-muted/92 border border-border/88 shadow-[inset_0_1px_0_rgba(255,255,255,0.16)] max-md:flex-auto max-md:w-full',
  viewModeButton: 'rounded-full border-transparent bg-transparent text-muted-foreground shadow-none max-md:flex-1',
  viewModeButtonActive: 'bg-primary border-primary text-white shadow-[0_8px_18px_-14px_rgba(0,0,0,0.45)]',
```

**Excerpt C — `web/src/pages/QuotaPage.tsx:1–22` (existing imports, for reference):**
```tsx
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useAuthStore, useThemeStore } from '@/stores';
import { authFilesApi, configFileApi } from '@/services/api';
import {
  AllQuotaSection,
  QuotaSection,
  ANTIGRAVITY_CONFIG,
  CLAUDE_CONFIG,
  CODEX_CONFIG,
  GEMINI_CLI_CONFIG,
  KIRO_CONFIG,
  KIMI_CONFIG,
  XAI_CONFIG,
} from '@/components/quota';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { IconRefreshCw, IconFilterAll } from '@/components/ui/icons';
import { getAuthFileIcon } from '@/features/authFiles/constants';
import type { AuthFileItem, ResolvedTheme } from '@/types';
import { quotaStyles as styles } from '@/components/quota/quotaStyles';
```

`Button` is absent from this list. It must be added.

**Exemplar — matching pattern in `web/src/components/quota/QuotaSection.tsx:283–310`:**
```tsx
            <div className={styles.viewModeToggle}>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  effectiveViewMode === 'paged' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => setViewMode('paged')}
              >
                {t('auth_files.view_mode_paged')}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  effectiveViewMode === 'all' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => { ... }}
              >
                {t('auth_files.view_mode_all')}
              </Button>
            </div>
```

## Commands you will need

All commands run from the `web/` directory.

| Purpose    | Command              | Expected on success                   |
|------------|----------------------|---------------------------------------|
| Type-check | `bun run type-check` | exit 0, no errors                     |
| Lint       | `bun run lint`       | exit 0, or pre-existing warnings only |

## Scope

**In scope** (the only file you should modify):
- `web/src/pages/QuotaPage.tsx`

**Out of scope** (do NOT touch):
- `web/src/components/quota/quotaStyles.ts` — class strings are already correct; no change.
- `web/src/components/quota/QuotaSection.tsx` — already uses the correct pattern; no change.
- `web/src/components/quota/AllQuotaSection.tsx` — the view mode toggle for the All-tab grid; also already receives `viewMode`/`onViewModeChange` from QuotaPage; no change.
- Any other file.

## Git workflow

- Branch: `advisor/011-quota-view-toggle-unify`
- Single commit: `fix(web): unify quota view-mode toggle to pill style`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add Button import to QuotaPage.tsx

Open `web/src/pages/QuotaPage.tsx`.

Add `Button` to the import block. Place it with the other UI component imports,
after the `@/components/ui/tabs` import line. Add:
```tsx
import { Button } from '@/components/ui/Button';
```

**Verify**: `grep -n "import.*Button" web/src/pages/QuotaPage.tsx`
Expected: one line showing the Button import.

### Step 2: Replace the raw toggle block

Still in `web/src/pages/QuotaPage.tsx`.

Locate lines 152–176 (the `<div className="flex items-center gap-1 shrink-0 pb-px mb-1">` block containing the two raw `<button>` elements and the refresh button).

The two raw `<button>` elements and their container (the inner div with `border border-border/70`) must be replaced. The outer wrapper div and the refresh `<button>` at lines 177–186 are **kept as-is**.

Replace the inner div + two buttons (lines 153–176) with:
```tsx
            <div className={styles.viewModeToggle}>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  viewMode === 'paged' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => setViewMode('paged')}
              >
                {t('auth_files.view_mode_paged')}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  viewMode === 'all' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => setViewMode('all')}
              >
                {t('auth_files.view_mode_all')}
              </Button>
            </div>
```

After the change the full block at the former lines 152–187 should look like:
```tsx
          <div className="flex items-center gap-1 shrink-0 pb-px mb-1">
            <div className={styles.viewModeToggle}>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  viewMode === 'paged' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => setViewMode('paged')}
              >
                {t('auth_files.view_mode_paged')}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  viewMode === 'all' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => setViewMode('all')}
              >
                {t('auth_files.view_mode_all')}
              </Button>
            </div>
            <button
              type="button"
              className="p-1.5 text-muted-foreground hover:text-foreground hover:bg-muted rounded disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              onClick={() => void handleTabRefresh()}
              disabled={loading}
              title={t('quota_management.refresh_all_credentials')}
              aria-label={t('quota_management.refresh_all_credentials')}
            >
              <IconRefreshCw size={13} className={loading ? 'animate-spin' : ''} />
            </button>
          </div>
```

Note: The refresh `<button>` at lines 177–186 is a plain `<button>` (icon-only,
small). It does NOT need to change — this plan is only about the view-mode
toggle.

**Verify**: `grep -n "viewModeToggle\|inline-flex.*border.*border-border/70" web/src/pages/QuotaPage.tsx`
Expected: `viewModeToggle` appears (from `styles.viewModeToggle`); the old
`border border-border/70` string is GONE.

### Step 3: Type-check

From `web/`:
```
bun run type-check
```
Expected: exit 0.

Common failure: if `Button` import path is wrong. Confirm it is
`@/components/ui/Button` (capital B, matching the filename `Button.tsx`).

### Step 4: Lint

From `web/`:
```
bun run lint
```
Expected: exit 0 or pre-existing warnings unchanged.

## Test plan

No automated tests cover component rendering in this repo. Manual visual gate:

1. Run `bun dev` from `web/`.
2. Navigate to the Quota Management page.
3. On the "All" tab (default), confirm the toggle is now a **rounded pill**
   (matching the toggle on provider-specific tabs).
4. Switch to a provider tab (e.g. Claude). Confirm the toggle on that tab looks
   identical to the "All" tab toggle.
5. Click "All" mode on the "All" tab — confirm the AllQuotaSection expands.
6. Return to "Paged" — confirm it pages correctly.

## Done criteria

- [ ] `bun run type-check` exits 0 (from `web/`)
- [ ] `bun run lint` exits 0 (from `web/`)
- [ ] `grep "border border-border/70" web/src/pages/QuotaPage.tsx` returns NO match (old container gone)
- [ ] `grep "viewModeToggle" web/src/pages/QuotaPage.tsx` shows the new wrapper class
- [ ] `grep "import.*Button" web/src/pages/QuotaPage.tsx` shows the Button import
- [ ] Only `web/src/pages/QuotaPage.tsx` is modified (`git diff --name-only`)
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

Stop and report back if:

- The block at lines 152–176 does not match Excerpt A (codebase drifted — line
  numbers may shift if other plans landed first; find the `border border-border/70`
  div using `grep -n "border-border/70" web/src/pages/QuotaPage.tsx`).
- Type-check fails on `Button` — confirm the export in
  `web/src/components/ui/Button.tsx` is a named export (`export { Button }` or
  `export function Button`), not a default export.
- The fix appears to require touching a file outside the in-scope list.

## Maintenance notes

- `QuotaPage.tsx` passes `viewMode` and `onViewModeChange` to both
  `AllQuotaSection` and `QuotaSection`. The toggle in `QuotaPage` controls the
  shared state. The child sections render their own toggles only when
  `externalViewMode` is `undefined` — since the parent now always passes it, the
  child toggle in `QuotaSection` is effectively hidden on the per-provider tabs.
  This is correct behavior — the single toggle in the tab header controls both
  views.
- If a future design wants the refresh button (`<button>` at lines 177–186 in
  the original, kept as-is here) to also use the `Button` component for
  consistency, that is a separate follow-up. The change is intentionally minimal.
