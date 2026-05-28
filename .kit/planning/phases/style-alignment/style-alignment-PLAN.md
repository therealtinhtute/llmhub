# Plan: Style Alignment

Phase: style-alignment
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-28

## Goal
Update all component TSX to use new design system classes. Migrate global utility class consumers and SCSS alias consumers to inline Tailwind.

## Inputs
- All TSX files with retro-specific classes
- Audit results: rounded-none (15+ files), shadow-hard (1 file), font-display (0 files — already gone from Phase 1), global utility classes (8+ files), SCSS aliases (7+ files)

## Wave 1 — shadcn UI component updates
### T1 — Remove rounded-none from shadcn UI components
- type: refactor
- touches:
  - `web/src/components/ui/Button.tsx` (1 instance)
  - `web/src/components/ui/Card.tsx` (1 instance)
  - `web/src/components/ui/Input.tsx` (1 instance)
  - `web/src/components/ui/Select.tsx` (3 instances)
  - `web/src/components/ui/dialog.tsx` (2 instances)
  - `web/src/components/ui/sheet.tsx` (1 instance)
  - `web/src/components/ui/checkbox.tsx` (1 instance)
  - `web/src/components/ui/alert-dialog.tsx` (2 instances)
  - `web/src/components/ui/badge.tsx` (if present)
  - `web/src/components/common/SplashScreen.tsx` (1 instance)
- avoid:
  - changing component behavior or props
- steps:
  1. For each file: read, find `rounded-none`, remove it (the default radius from CSS vars will apply)
  2. In Card.tsx: also replace `shadow-[var(--shadow-hard)]` with `shadow-sm`
- expected outputs:
  - No `rounded-none` in any UI component
  - No `shadow-hard` references
- verification:
  - `grep -r "rounded-none" web/src/components/ui/` returns no matches
  - `grep -r "shadow-hard" web/src/` returns no matches
- stop if:
  - a component structurally requires no border-radius
- escalate to:
  - user clarification

### T2 — Add custom scrollbar class to index.css
- type: implementation
- touches:
  - `web/src/index.css` (add after `@layer base` block)
- avoid:
  - modifying tokens or other sections
- steps:
  1. Add `.custom-scrollbar` utility class with thin scrollbar styling
- expected outputs:
  - `.custom-scrollbar` class available in CSS
- verification:
  - `grep "custom-scrollbar" web/src/index.css` returns a match

## Wave 2 — Global utility class consumer migration
### T3 — Migrate .error-box consumers to inline Tailwind
- type: migration
- touches:
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/SystemPage.tsx`
  - `web/src/pages/ConfigPage.tsx`
  - `web/src/components/config/VisualConfigEditor.tsx`
  - `web/src/components/config/VisualConfigEditorBlocks.tsx`
- avoid:
  - changing error display logic
- steps:
  1. Replace `className="error-box"` with inline Tailwind: `className="p-3 mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm"`
  2. Handle variants (`.error-box.m-0` etc.)
- expected outputs:
  - No `.error-box` class references in TSX
- verification:
  - `grep -r "error-box" web/src/ --include="*.tsx"` returns no matches

### T4 — Migrate .status-badge consumers to inline Tailwind
- type: migration
- touches:
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/SystemPage.tsx`
  - `web/src/pages/OAuthPage.tsx`
- avoid:
  - changing status logic
- steps:
  1. Replace `.status-badge` + variants (`.success`, `.error`, `.warning`) with Badge component or inline Tailwind
- expected outputs:
  - No `.status-badge` references in TSX
- verification:
  - `grep -r "status-badge" web/src/ --include="*.tsx"` returns no matches

### T5 — Migrate .item-list/.item-row/.item-meta/.item-title/.item-subtitle/.item-actions consumers
- type: migration
- touches:
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/SystemPage.tsx`
  - `web/src/components/config/VisualConfigEditorBlocks.tsx`
  - `web/src/components/config/VisualConfigEditor.tsx` (parent selector overrides referencing these classes)
- avoid:
  - changing list data rendering logic
- steps:
  1. Replace `.item-list` with `flex flex-col`
  2. Replace `.item-row` with `flex items-center justify-between gap-2 py-2.5 border-b border-border last:border-b-0`
  3. Replace `.item-meta` with `flex flex-col gap-0.5 min-w-0 flex-1`
  4. Replace `.item-title` with `text-sm font-medium text-foreground`
  5. Replace `.item-subtitle` with `text-xs text-muted-foreground`
  6. Replace `.item-actions` with `flex items-center gap-1.5 shrink-0`
  7. Update VisualConfigEditor parent selector overrides that target these classes
- expected outputs:
  - No `.item-list/.item-row/.item-meta` etc. references
- verification:
  - `grep -rE "item-list|item-row|item-meta|item-title|item-subtitle|item-actions" web/src/ --include="*.tsx"` returns no matches

### T6 — Migrate .form-group, .hint, .pill, .input consumers
- type: migration
- touches:
  - `web/src/components/config/VisualConfigEditorBlocks.tsx`
  - `web/src/components/config/VisualConfigEditor.tsx`
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/ConfigPage.tsx`
  - `web/src/features/authFiles/components/AuthFilesPrefixProxyEditorModal.tsx`
- avoid:
  - changing form behavior
- steps:
  1. Replace `.form-group` with `flex flex-col gap-1.5 mb-3`
  2. Replace `.hint` with `text-[13px] text-muted-foreground`
  3. Replace `.pill` with inline badge styles or Badge component
  4. Replace `.input` (native input class) with shadcn Input component or inline Tailwind
  5. Update VisualConfigEditor parent selector overrides
- expected outputs:
  - No `.form-group/.hint/.pill/.input` references
- verification:
  - `grep -rE "\"form-group|\"hint\"|\"pill\"|className=\"input\"" web/src/ --include="*.tsx"` returns no matches

## Wave 3 — SCSS compatibility alias consumer migration
### T7 — Migrate SCSS alias consumers
- type: migration
- touches:
  - `web/src/components/providers/ProviderStatusBar.tsx` (`--success-badge-*`, `--warning-*`, `--failure-badge-*`)
  - `web/src/pages/AuthFilesPage.tsx` (`--bg-secondary`, `--bg-primary`, `--primary-color`, `--border-color`, `--filter-color`, `--filter-surface`, `--count-badge-*`, `--bg-tertiary`, `--bg-hover`)
  - `web/src/features/providers/components/ProviderCategoryList.tsx` (`--primary-30`, `--primary-10`, `--amber-*`)
  - `web/src/components/quota/quotaStyles.ts` (`--count-badge-*`, `--warning-*`)
  - `web/src/components/config/ConfigSection.tsx` (`--bg-primary`)
  - `web/src/components/common/SecondaryScreenShell.tsx` (`--glass-bg`, `--glass-border`)
  - `web/src/pages/PlaceholderPage.tsx` (`--text-secondary`)
  - `web/src/pages/AuthFilesOAuthModelAliasEditPage.tsx` (`--bg-tertiary`)
  - `web/src/pages/AuthFilesOAuthExcludedEditPage.tsx` (`--bg-tertiary`, `--bg-hover`)
- avoid:
  - changing component behavior
- steps:
  1. For each file, replace SCSS alias references with semantic Tailwind equivalents:
     - `var(--text-secondary)` → `text-muted-foreground`
     - `var(--bg-primary)` → `bg-background`
     - `var(--bg-secondary)` → `bg-muted`
     - `var(--bg-tertiary)` → `bg-secondary`
     - `var(--bg-hover)` → `hover:bg-accent`
     - `var(--primary-color)` → semantic primary classes
     - `var(--glass-bg)` → `bg-background/80 backdrop-blur-md`
     - `var(--glass-border)` → `border-border/50`
     - `var(--success-badge-bg)` → `bg-emerald-100 dark:bg-emerald-900`
     - `var(--success-badge-text)` → `text-emerald-700 dark:text-emerald-300`
     - `var(--warning-bg)` → `bg-amber-100 dark:bg-amber-900`
     - `var(--warning-text)` → `text-amber-700 dark:text-amber-300`
     - `var(--failure-badge-bg)` → `bg-destructive/10`
     - `var(--failure-badge-text)` → `text-destructive`
     - `var(--count-badge-bg)` → `bg-primary/10`
     - `var(--count-badge-text)` → `text-primary`
     - `var(--amber-text)` → amber color classes
     - `var(--filter-color)`, `var(--filter-surface)` → evaluate if dynamic CSS vars still needed or can use data attributes
     - `var(--border-color)` → `border-border`
- expected outputs:
  - No SCSS alias references in any TSX file
- verification:
  - `grep -rE "var\(--(text-primary|text-secondary|bg-primary|bg-secondary|bg-tertiary|bg-hover|glass-|success-badge|failure-badge|count-badge|amber-|filter-color|filter-surface|warning-bg|warning-text|primary-color|border-color)" web/src/ --include="*.tsx" --include="*.ts"` returns no matches
- stop if:
  - dynamic CSS variable usage (e.g., `--filter-color` set via JS) can't be replaced with static Tailwind classes
- escalate to:
  - user clarification (may need to keep a few CSS vars for dynamic styling)

### T8 — Build verification
- type: test
- steps:
  1. `cd web && bun run build` — must succeed
  2. `grep -r "rounded-none" web/src/components/ui/` — zero matches
  3. `grep -r "shadow-hard" web/src/` — zero matches
  4. `grep -rE "error-box|status-badge|item-list|item-row|form-group" web/src/ --include="*.tsx"` — zero matches
- expected outputs:
  - Clean build, no legacy class references
- verification:
  - All commands produce expected results

## Risks / Watch-fors
- VisualConfigEditor.tsx has complex parent-selector overrides targeting global utility classes — most complex migration in this phase
- AuthFilesPage.tsx has heavy SCSS alias usage with dynamic CSS var composition (color-mix) — may need to preserve some dynamic vars
- `--filter-color` and `--filter-surface` are set dynamically via style props in AuthFilesPage — may need a different approach than static Tailwind
