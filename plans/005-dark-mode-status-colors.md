# Plan 005: Replace hardcoded emerald/amber color tokens with dark-mode-aware equivalents (Phase 1)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command before moving on. If anything in "STOP conditions" occurs,
> stop and report — do not improvise. Update the status row in `plans/README.md`
> when done.
>
> **Drift check (run first)**:
> ```bash
> git diff --stat 5f81cee..HEAD -- \
>   web/src/pages/ModelsPage.tsx \
>   web/src/components/providers/ProviderStatusBar.tsx \
>   web/src/features/providers/components/ProviderResourceTable.tsx \
>   web/src/features/authFiles/components/AuthFileCard.tsx
> ```
> If any file changed since this plan was written, compare the "Current state"
> excerpts below against live code before proceeding.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (styling only, no logic)
- **Depends on**: none
- **Category**: accessibility / dark mode
- **Planned at**: commit `5f81cee`, 2026-06-12

## Why this matters

The app supports a dark theme (toggled via `useThemeStore`). Status and badge
components use hardcoded Tailwind color tokens — `emerald-100/700` for success,
`amber-100/700` for warning — with no `dark:` variants. In dark mode:

- `bg-emerald-100` (a very light green) on a dark background creates a jarring
  bright patch instead of a muted "success" feel.
- `text-emerald-700` (a dark green) on a dark background has poor contrast — the
  WCAG AA minimum requires 4.5:1 for normal text; dark green on dark gray fails this.
- Same problem for all `amber-100/700` warning variants.

There are 40+ hardcoded instances across 15+ files. This plan fixes the 4 highest-
traffic files (Phase 1). The remaining files are documented in the "Phase 2 backlog"
section below but are NOT in scope here.

## Dark mode replacement patterns

These are the canonical substitutions to use everywhere in this plan. Apply them
consistently — do not invent new values.

| Light class | Dark class to add | Context |
|---|---|---|
| `text-emerald-700` | `dark:text-emerald-400` | success text |
| `text-emerald-600` | `dark:text-emerald-400` | success text (lighter source) |
| `bg-emerald-100` | `dark:bg-emerald-950` | success badge bg |
| `bg-emerald-100/10` | `dark:bg-emerald-950/10` | success bg (faint) |
| `bg-emerald-100/50` | `dark:bg-emerald-950/50` | success bg (medium) |
| `bg-emerald-100/60` | `dark:bg-emerald-950/60` | success bg (medium-high) |
| `border-emerald-400/40` | `dark:border-emerald-500/40` | success border |
| `border-emerald-500/34` | `dark:border-emerald-500/40` | success border (light) |
| `text-amber-700` | `dark:text-amber-400` | warning text |
| `text-amber-600` | `dark:text-amber-400` | warning text (lighter source) |
| `bg-amber-100` | `dark:bg-amber-950` | warning badge bg |
| `bg-amber-100/60` | `dark:bg-amber-950/60` | warning bg (medium-high) |
| `border-amber-400/30` | `dark:border-amber-500/30` | warning border |
| `bg-amber-500` | `dark:bg-amber-400` | amber progress bar fill |
| `bg-amber-500/15` | `dark:bg-amber-500/20` | amber badge bg (opacity-based) |
| `bg-[rgba(245,158,11,0.10)]` | (no dark change — already < 0.15 opacity, readable) | inline amber bg |
| `border-[rgba(245,158,11,0.30)]` | (no dark change — border alpha already low) | inline amber border |

**Do NOT** change:
- `bg-emerald-500` at `SystemPage.tsx:19` — this is a brand icon background, not a status badge.
- Any `amber-400` dot indicator (solid color, not a badge) — keep as-is.
- Any `dark:` variant already present — do not double-add.

## Scope — Phase 1 (this plan)

**In scope** (4 files, each change listed below):
1. `web/src/pages/ModelsPage.tsx` — `getStatusBadgeClasses()` function
2. `web/src/components/providers/ProviderStatusBar.tsx` — provider status pill
3. `web/src/features/providers/components/ProviderResourceTable.tsx` — resource status badges
4. `web/src/features/authFiles/components/AuthFileCard.tsx` — auth file status chip

**Out of scope** (deferred to Phase 2, listed at the end):
- All other files. Do NOT touch them in this plan even if they have the same pattern.

## Git workflow

- Branch: `advisor/005-dark-mode-status-colors-phase1`
- Commit: `fix(web): add dark-mode variants to emerald/amber status tokens (phase 1)`
- Do NOT push or open a PR unless instructed.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Type-check | `cd web && bun run type-check` | exit 0 |
| Tests | `cd web && bun run test:run` | 60 tests pass |
| Lint | `cd web && bun run lint` | exit 0 errors |

---

## File 1: `web/src/pages/ModelsPage.tsx`

### Current state (lines 73–75)

```ts
function getStatusBadgeClasses(type: ModelStatus): string {
  if (type === 'success') return 'text-emerald-700 bg-emerald-100 border-emerald-400/40';
  if (type === 'info') return 'text-blue-700 bg-blue-100 border-blue-400/40';
  if (type === 'warning') return 'text-amber-700 bg-amber-100 border-amber-400/30';
  return 'text-muted-foreground bg-muted border-border';
}
```

### Target state

```ts
function getStatusBadgeClasses(type: ModelStatus): string {
  if (type === 'success') return 'text-emerald-700 dark:text-emerald-400 bg-emerald-100 dark:bg-emerald-950 border-emerald-400/40 dark:border-emerald-500/40';
  if (type === 'info') return 'text-blue-700 bg-blue-100 border-blue-400/40';
  if (type === 'warning') return 'text-amber-700 dark:text-amber-400 bg-amber-100 dark:bg-amber-950 border-amber-400/30 dark:border-amber-500/30';
  return 'text-muted-foreground bg-muted border-border';
}
```

**Note**: The `'info'` line is not changed (blue is not in scope). The `return`
fallback is not changed (uses semantic design tokens, already dark-mode-aware).

**Verify**:
```bash
grep -n "dark:text-emerald-400\|dark:bg-emerald-950" web/src/pages/ModelsPage.tsx
```
Expected: 2 lines.

---

## File 2: `web/src/components/providers/ProviderStatusBar.tsx`

### Current state (lines 47–53)

```ts
const ratePillClass = successRate === 100
  ? 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-emerald-700 bg-emerald-100'
  : successRate >= 80
    ? 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-amber-700 bg-amber-100'
    : 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-destructive bg-destructive/10';
```

(Line numbers are approximate; use the strings as the anchor, not line numbers.)

### Target state

```ts
const ratePillClass = successRate === 100
  ? 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-emerald-700 dark:text-emerald-400 bg-emerald-100 dark:bg-emerald-950'
  : successRate >= 80
    ? 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-amber-700 dark:text-amber-400 bg-amber-100 dark:bg-amber-950'
    : 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-destructive bg-destructive/10';
```

The `text-destructive / bg-destructive/10` line uses semantic tokens — do not change it.

**Verify**:
```bash
grep -n "dark:text-emerald-400\|dark:text-amber-400" web/src/components/providers/ProviderStatusBar.tsx
```
Expected: 2 lines.

---

## File 3: `web/src/features/providers/components/ProviderResourceTable.tsx`

This file has 4 badge instances.

### Instance A — "pending" (lines ~136 and ~144, two separate badges using amber)

**Current state** (line ~136):
```tsx
<span className="inline-flex items-center gap-1 px-2 py-0.5 border border-[rgba(245,158,11,0.30)] bg-[rgba(245,158,11,0.10)] text-amber-700 text-[11px] font-medium whitespace-nowrap">
```

**Current state** (line ~144, different badge, same pattern):
```tsx
<span className="inline-flex items-center gap-1 px-2 py-0.5 border border-[rgba(245,158,11,0.30)] bg-[rgba(245,158,11,0.10)] text-amber-700 text-[11px] font-medium whitespace-nowrap">
```

Per the replacement table, `rgba(245,158,11,0.10)` and `rgba(245,158,11,0.30)` do
**not** need dark: variants (their opacity is already very low). Only `text-amber-700`
needs a dark: variant.

**Target state** (both lines — same change applied to both):
```tsx
<span className="inline-flex items-center gap-1 px-2 py-0.5 border border-[rgba(245,158,11,0.30)] bg-[rgba(245,158,11,0.10)] text-amber-700 dark:text-amber-400 text-[11px] font-medium whitespace-nowrap">
```

### Instance B — "connected" pill (line ~151)

**Current state**:
```tsx
<span className="inline-flex items-center gap-1 px-2 py-0.5 border border-emerald-400/40 bg-emerald-100 text-emerald-700 text-[11px] font-medium whitespace-nowrap">
```

**Target state**:
```tsx
<span className="inline-flex items-center gap-1 px-2 py-0.5 border border-emerald-400/40 dark:border-emerald-500/40 bg-emerald-100 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-400 text-[11px] font-medium whitespace-nowrap">
```

### Instance C — token usage pill (line ~250)

**Current state**:
```tsx
<span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold leading-[1.4] border whitespace-nowrap tabular-nums bg-emerald-100 text-emerald-700 border-emerald-400/40">
```

**Target state**:
```tsx
<span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold leading-[1.4] border whitespace-nowrap tabular-nums bg-emerald-100 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-400 border-emerald-400/40 dark:border-emerald-500/40">
```

**Verify**:
```bash
grep -c "dark:text-emerald-400\|dark:text-amber-400" web/src/features/providers/components/ProviderResourceTable.tsx
```
Expected: at least `3` (one per instance that needed it).

---

## File 4: `web/src/features/authFiles/components/AuthFileCard.tsx`

This file has 3 badge instances.

### Instance A — status chip ternary (lines ~120–121)

**Current state**:
```ts
const statusChipClass = hasError
  ? 'text-amber-700 bg-amber-100 border border-amber-400/30'
  : 'text-emerald-700 bg-emerald-100 border border-emerald-400/40';
```

**Target state**:
```ts
const statusChipClass = hasError
  ? 'text-amber-700 dark:text-amber-400 bg-amber-100 dark:bg-amber-950 border border-amber-400/30 dark:border-amber-500/30'
  : 'text-emerald-700 dark:text-emerald-400 bg-emerald-100 dark:bg-emerald-950 border border-emerald-400/40 dark:border-emerald-500/40';
```

### Instance B — warning/notice banner (line ~249)

**Current state**:
```tsx
className={`flex items-start gap-[6px] text-[11px] text-amber-700 bg-amber-100/60 border border-amber-400/30 p-[8px_10px] break-words ${compact ? 'text-[11px] p-[6px_8px]' : ''}`}
```

**Target state**:
```tsx
className={`flex items-start gap-[6px] text-[11px] text-amber-700 dark:text-amber-400 bg-amber-100/60 dark:bg-amber-950/60 border border-amber-400/30 dark:border-amber-500/30 p-[8px_10px] break-words ${compact ? 'text-[11px] p-[6px_8px]' : ''}`}
```

### Instance C — session pill (line ~260)

**Current state**:
```tsx
className={`inline-flex items-baseline gap-[5px] min-w-0 bg-emerald-100/50 text-emerald-700 ${compact ? 'py-[3px] px-2' : 'py-1 px-[10px]'}`}
```

**Target state**:
```tsx
className={`inline-flex items-baseline gap-[5px] min-w-0 bg-emerald-100/50 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400 ${compact ? 'py-[3px] px-2' : 'py-1 px-[10px]'}`}
```

**Verify**:
```bash
grep -c "dark:text-emerald-400\|dark:text-amber-400" web/src/features/authFiles/components/AuthFileCard.tsx
```
Expected: at least `3`.

---

## Final verification

```bash
cd web && bun run type-check
cd web && bun run test:run
cd web && bun run lint
```

All three must exit 0.

Also verify no dark: variants were accidentally doubled:
```bash
grep -n "dark:.*dark:" web/src/pages/ModelsPage.tsx web/src/components/providers/ProviderStatusBar.tsx web/src/features/providers/components/ProviderResourceTable.tsx web/src/features/authFiles/components/AuthFileCard.tsx
```
Expected: no output (no doubled dark: in a single string).

## Done criteria

- [ ] `grep -c "dark:text-emerald-400\|dark:text-amber-400" web/src/pages/ModelsPage.tsx` → `2`
- [ ] `grep -c "dark:text-emerald-400\|dark:text-amber-400" web/src/components/providers/ProviderStatusBar.tsx` → `2`
- [ ] `grep -c "dark:text-emerald-400\|dark:text-amber-400" web/src/features/providers/components/ProviderResourceTable.tsx` → `3`
- [ ] `grep -c "dark:text-emerald-400\|dark:text-amber-400" web/src/features/authFiles/components/AuthFileCard.tsx` → `3`
- [ ] `cd web && bun run type-check` exits 0
- [ ] `cd web && bun run test:run` exits 0; 60 tests pass
- [ ] `cd web && bun run lint` exits 0 errors
- [ ] `git status` shows only in-scope files modified
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

- A file's current state doesn't match the "Current state" excerpts above — drift
  check failed, stop and report the actual current lines.
- `bun run lint` reports class order violations — if the linter enforces Tailwind
  class ordering, sort new `dark:` variants to follow their base class immediately
  (e.g. `bg-emerald-100 dark:bg-emerald-950` as an adjacent pair).

## Phase 2 backlog (not in this plan — create a new plan or extend this one)

These files also have hardcoded emerald/amber without dark: variants. Apply the same
substitution table from the "Dark mode replacement patterns" section above:

| File | Lines | Pattern |
|------|-------|---------|
| `web/src/features/authFiles/components/AuthFileQuotaSection.tsx` | 35, 53 | amber warning label + status pill |
| `web/src/features/authFiles/components/QuotaProgressBar.tsx` | 10, 16 | `bg-amber-500` progress bar fill |
| `web/src/components/quota/QuotaCard.tsx` | 30, 36 | `bg-amber-500` progress bar fill |
| `web/src/components/quota/quotaStyles.ts` | 70 | amber warning label string |
| `web/src/components/quota/quotaConfigs.ts` | 1663, 1668 | `text-emerald-700` / `text-amber-700` in template strings |
| `web/src/features/providers/components/ProviderHeaderCard.tsx` | 88 | amber warning banner |
| `web/src/features/providers/components/ProviderResourcePanel.tsx` | 123 | amber warning block |
| `web/src/features/providers/components/ProviderCategoryList.tsx` | 98, 105 | amber icon + amber badge |
| `web/src/features/providers/sheets/ResourceDetailView.tsx` | 90 | emerald success badge |
| `web/src/features/providers/sheets/forms/BaseProviderForm.tsx` | 179 | `text-emerald-600` check icon |
| `web/src/pages/SystemPage.tsx` | 345 | amber warning badge (skip line 19 — brand icon bg) |
| `web/src/components/config/VisualConfigEditor.tsx` | 328 | amber number badge |
| `web/src/pages/ConfigPage.tsx` | 425–427 | amber/emerald status bar classes |
| `web/src/pages/ApiKeysPage.tsx` | 135 | amber warning banner |
| `web/src/pages/OAuthPage.tsx` | 378, 495 | emerald success badges |
| `web/src/pages/LogsPage.tsx` | 842, 881, 885, 961 | amber/emerald status badges + warning banner |

## Maintenance notes

- Tailwind v4 is configured in this project. Tailwind's JIT scanner does NOT pick
  up `dark:` variants added to template literal strings at runtime — it only scans
  static class strings. All class names in this plan appear as static string literals,
  so they will be included in the production build. If a future change moves them to
  runtime-computed strings, the `dark:` variants must be listed in a
  `safelist` in `tailwind.config`.
- If the design system ever introduces a semantic `--warning-foreground` CSS variable
  (like `--destructive-foreground` already exists), replace all `text-amber-700
  dark:text-amber-400` with the semantic token and remove the hardcoded amber.
  Same for `--success-foreground`. This plan's replacements are a pragmatic
  intermediate step, not the long-term target.
