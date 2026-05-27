# Run: shadcn-primitives

Phase: shadcn-primitives
Mode: full
Notes: enabled
Started: 2026-05-28 00:20
Status: COMPLETE

## Preflight
- SPEC: locked (user-confirmed)
- Phase CONTEXT: present, ready
- Phase PLAN: present, ready, 4 waves / 6 tasks
- Drift check: clean — no changes in allowed surfaces beyond Phase 1
- Verdict: **READY**

## Wave 1: Install shadcn components + Sonner

### T1 — Batch install shadcn components
- Status: DONE
- 16 shadcn components installed via `bunx shadcn@latest add`
- Sonner installed alongside
- Fixed: `sonner.tsx` removed `next-themes` dependency, `dialog.tsx`/`alert-dialog.tsx` fixed Button import casing
- Removed `"use client"` directives from 6 files

## Wave 2: Customize components for design tokens

### T2 — Customize Button component
- Status: DONE
- Added `VARIANT_ALIAS` map (`primary`→`default`, `danger`→`destructive`) with TypeScript support via `VariantAlias` type
- Added `loading` prop (Loader2 spinner + disabled)
- Added `fullWidth` prop
- `rounded-none` in base CVA class

### T3 — Customize all other components
- Status: DONE
- Card: `rounded-none border border-border shadow-[var(--shadow-hard)]`
- Input: `rounded-none`
- Dialog/AlertDialog: `rounded-none border border-border`
- Sheet: `rounded-none` on close button
- Sonner: hardcoded `--border-radius: 0px`, removed next-themes
- Checkbox: `rounded-none`
- All components use `cn()` from `@/lib/utils`

## Wave 3: Replace hand-rolled components + migrate notifications

### T4 — Remove old UI components, update imports
- Status: DONE
- Trashed old components: Modal, Select, Sheet/, Table/, Skeleton/, Collapsible/, ToggleSwitch, SelectionCheckbox, AutocompleteInput, EmptyState, LoadingSpinner
- Created 14 Legacy wrappers: LegacyModal, LegacySelect, LegacySheet, LegacyInput, LegacyTable, LegacySkeleton, LegacyCollapsible, LegacyToggleSwitch, LegacySelectionCheckbox, LegacyEmptyState, LegacyLoadingSpinner, LegacyAutocompleteInput, LegacyCard
- Updated 35+ consumer imports to use Legacy wrappers
- Button variant alias type extended to accept `"primary"` and `"danger"` at type level

### T5 — Migrate notifications to Sonner
- Status: DONE
- Replaced `<NotificationContainer />` with `<Toaster position="top-right" richColors closeButton />` in App.tsx
- Converted `showNotification(msg, type)` → `toast[type](msg)` across 17 files
- `showConfirmation` + `ConfirmationModal` kept as-is (deferred to Phase 4)
- `useNotificationStore` kept for confirmation state only

## Wave 4: Build check

### T6 — Type-check and build verification
- Status: DONE
- `bun run build` → tsc clean + vite build → dist/index.html 2,192.43 kB ✓
- Zero type errors
- 2501 modules transformed
