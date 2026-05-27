# Plan: shadcn Primitives

Phase: shadcn-primitives
Status: ready
Wave Count: 4
Execution Owner: work
Updated At: 2026-05-27

## Goal
Install all needed shadcn components. Customize to match design tokens. Replace all 16 hand-rolled UI primitives. Migrate notifications to Sonner.

## Inputs
- Tailwind foundation from Phase 1 (index.css with @theme, components.json, cn() utility)
- Current hand-rolled components: `web/src/components/ui/*.tsx`
- Current notification store: `web/src/stores/useNotificationStore.ts`
- All consumer files that import from `@/components/ui/`

---

## Wave 1: Install all shadcn components + Sonner
### T1 — Batch install shadcn components
- type: implementation
- inputs:
  - `web/components.json`
- touches:
  - `web/src/components/ui/` — new component files
  - `web/package.json` — Radix UI dependencies added automatically
- avoid:
  - modifying any page or feature code yet
  - deleting existing hand-rolled components yet
- steps:
  1. Back up existing `web/src/components/ui/` to a temp reference (or rely on git)
  2. Install components via CLI (run each from `web/`):
     ```
     bunx shadcn@latest add button card input select dialog alert-dialog sheet table skeleton collapsible switch empty combobox spinner checkbox sonner separator badge dropdown-menu label tooltip scroll-area
     ```
  3. Verify all component files appear in `web/src/components/ui/`
  4. Install Sonner: `bun add sonner` in `web/`
- expected outputs:
  - All shadcn component source files in `web/src/components/ui/`
  - Sonner installed
- verification:
  - `ls web/src/components/ui/` — lists all installed components
  - `cd web && bun run type-check` — may have conflicts with old files; expected at this stage
- stop if:
  - shadcn CLI fails to install >3 components
- escalate to:
  - plan phase (manual component copy)

---

## Wave 2: Customize shadcn components for design tokens
### T2 — Customize Button component
- type: implementation
- inputs:
  - shadcn `button.tsx` from T1
  - Current hand-rolled `Button.tsx` (variant API reference)
- touches:
  - `web/src/components/ui/button.tsx`
- avoid:
  - changing consumer code yet
- steps:
  1. Edit shadcn `button.tsx` variants:
     - `default` → maps to old `primary`: `bg-primary text-primary-foreground`
     - `secondary` → `bg-secondary text-secondary-foreground border border-border`
     - `ghost` → `bg-transparent text-muted-foreground hover:bg-accent hover:text-accent-foreground`
     - `destructive` → `bg-destructive text-destructive-foreground`
  2. Ensure `rounded-none` (0 radius) on all variants
  3. Add `loading` prop: when true, show Spinner and disable button
  4. Ensure `fullWidth` prop or `w-full` variant works
  5. Export variant types that match old API for smoother migration
- expected outputs:
  - `button.tsx` with design-token styling, loading prop, 0 radius
- verification:
  - `cd web && bun run type-check` — button.tsx compiles
- stop if:
  - CVA variant API incompatible with loading prop extension
- escalate to:
  - user clarification

### T3 — Customize all other shadcn components
- type: implementation
- inputs:
  - All shadcn component files from T1
  - `design-token.json` (0 radius, hard shadows, colors)
- touches:
  - `web/src/components/ui/*.tsx` — all shadcn component files
- avoid:
  - changing consumer code yet
- steps:
  1. For each component, apply design token overrides:
     - Replace all `rounded-*` with `rounded-none`
     - Replace soft shadows with `shadow-[var(--shadow-hard)]` where applicable
     - Ensure colors use `bg-background`, `text-foreground`, `border-border`, etc.
  2. Card: add `shadow-[var(--shadow-hard)]` and `border border-border`
  3. Dialog: ensure overlay uses `bg-black/35`, dialog panel uses `bg-background border border-border rounded-none`
  4. Sheet: `rounded-none`, design token borders
  5. Table: ensure `border-border` separators, `bg-muted` header
  6. Input: `rounded-none`, `border-border`, `focus:ring-ring`
  7. Select: `rounded-none`, trigger and content styled with tokens
  8. Switch: style track with `bg-border` off state, `bg-primary` on state
  9. Skeleton: `bg-muted` with 0 radius
  10. Empty: use `border-dashed border-border rounded-none`
- expected outputs:
  - All components styled with design tokens
- verification:
  - `cd web && bun run type-check` — compiles
- stop if:
  - more than 5 components need fundamentally different markup than shadcn provides
- escalate to:
  - plan phase (may need custom wrappers)

---

## Wave 3: Replace hand-rolled components + migrate notifications
### T4 — Remove old UI components, update imports
- type: migration
- inputs:
  - Old hand-rolled components in `web/src/components/ui/`
  - All consumer files importing from `@/components/ui/`
- touches:
  - `web/src/components/ui/` — delete old files that shadcn replaced (Button.tsx, Card.tsx, Input.tsx, Select.tsx, Modal.tsx, Sheet/, Table/, Skeleton/, Collapsible/, ToggleSwitch.tsx, EmptyState.tsx, AutocompleteInput.tsx, LoadingSpinner.tsx, SelectionCheckbox.tsx, icons.tsx, scrollLock.ts)
  - All TSX files that import these components — update import paths and prop APIs
- avoid:
  - changing page layout or structure
  - changing business logic
  - changing API calls
- steps:
  1. Grep for all imports of each old component: `grep -r "from.*components/ui/Button" web/src/`
  2. For each old component, map to new shadcn equivalent:
     - `Button` → `button` (same path, different export style)
     - `Card` → `card` (Card, CardHeader, CardContent, CardFooter, CardTitle)
     - `Input` → `input`
     - `Select` → `select` (Select, SelectTrigger, SelectContent, SelectItem, SelectValue)
     - `Modal` → `dialog` (Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogFooter)
     - `ConfirmationModal` → AlertDialog
     - `Sheet/` → `sheet`
     - `Table/` → `table`
     - `Skeleton/` → `skeleton`
     - `Collapsible/` → `collapsible`
     - `ToggleSwitch` → `switch`
     - `EmptyState` → `empty`
     - `AutocompleteInput` → `combobox`
     - `LoadingSpinner` → `spinner`
     - `SelectionCheckbox` → `checkbox`
  3. Update each consumer file:
     - Change import paths
     - Adapt prop names (e.g., `variant="danger"` → `variant="destructive"`)
     - Adapt component structure (e.g., Modal → Dialog with DialogContent children)
  4. Delete old component files after all consumers are updated
  5. Keep `icons.tsx` if it's still used for custom SVG icons
  6. Move `scrollLock.ts` to utils if still needed, or replace with Radix's built-in scroll lock
- expected outputs:
  - All old UI component files removed (except icons.tsx if needed)
  - All consumers use shadcn imports
  - No TypeScript errors from import mismatches
- verification:
  - `cd web && bun run type-check` — zero errors
  - `grep -r "module.scss" web/src/components/ui/` — zero matches (shadcn components have no SCSS)
- stop if:
  - type errors exceed 20 after bulk replacement
- escalate to:
  - plan phase (may need phased component migration instead of bulk)

### T5 — Migrate notifications to Sonner
- type: migration
- inputs:
  - `web/src/stores/useNotificationStore.ts`
  - `web/src/components/common/NotificationContainer.tsx`
  - All files calling `showNotification()`
- touches:
  - `web/src/App.tsx` — add `<Toaster />` from Sonner
  - All files calling `useNotificationStore` or `showNotification()` — replace with Sonner's `toast()` API
  - `web/src/stores/useNotificationStore.ts` — delete
  - `web/src/stores/index.ts` — remove useNotificationStore export
  - `web/src/components/common/NotificationContainer.tsx` — delete
- avoid:
  - changing notification message strings (i18n keys stay same)
  - changing when/where notifications are triggered
- steps:
  1. Add `<Toaster />` to `App.tsx` (or layout root)
  2. Grep for all `showNotification(` calls: `grep -rn "showNotification" web/src/`
  3. Replace each call:
     - `showNotification(message, 'success')` → `toast.success(message)`
     - `showNotification(message, 'error')` → `toast.error(message)`
     - `showNotification(message, 'warning')` → `toast.warning(message)`
     - `showNotification(message)` → `toast(message)`
  4. Remove `useNotificationStore` import from each consumer
  5. Delete `useNotificationStore.ts` and `NotificationContainer.tsx`
  6. Update `stores/index.ts` to remove the export
  7. Style Sonner toaster with design tokens: `rounded-none`, border colors, font
- expected outputs:
  - Sonner toasts replace all hand-rolled notifications
  - No references to old notification store
  - Toast styling matches design tokens
- verification:
  - `grep -r "useNotificationStore" web/src/` — zero matches
  - `grep -r "NotificationContainer" web/src/` — zero matches
  - `cd web && bun run type-check` — zero errors
- stop if:
  - Sonner z-index conflicts with existing overlays
- escalate to:
  - user clarification (toast positioning)

---

## Wave 4: Build check
### T6 — Type-check and build verification
- type: test
- inputs:
  - All changes from T1–T5
- touches:
  - nothing (verification only)
- avoid:
  - any code changes
- steps:
  1. `cd web && bun run type-check` — must pass
  2. `cd web && bun run build` — must succeed
  3. Inspect `dist/index.html` — still single file
- expected outputs:
  - Clean type-check
  - Successful build
- verification:
  - Both commands exit 0
- stop if:
  - type-check has errors → fix before proceeding
- escalate to:
  - plan phase if >10 errors remain after reasonable fixes

## Risks / Watch-fors
- shadcn's `Select` is Radix-based and has a fundamentally different API from the custom Select (value/onValueChange vs onChange) — may require the most consumer-side changes
- shadcn's `Dialog` uses portal by default — existing modal positioning logic may be obsolete (good — less code)
- `AutocompleteInput` → `Combobox` migration is the least 1:1 — combobox has different filtering/selection UX
- Some old component files may have re-exports or barrel files (like `Sheet/index.ts`) — ensure all are updated
