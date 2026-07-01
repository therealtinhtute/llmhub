# Plan 003: Wire useUnsavedChangesGuard to ConfigPage so sidebar navigation can't silently discard edits

> **Executor instructions**: Follow this plan step by step. Run every
> verification command before moving on. If anything in "STOP conditions" occurs,
> stop and report — do not improvise. Update the status row in `plans/README.md`
> when done.
>
> **Drift check (run first)**:
> `git diff --stat 5f81cee..HEAD -- web/src/pages/ConfigPage.tsx web/src/hooks/useUnsavedChangesGuard.ts`
> If either file changed since this plan was written, compare the
> "Current state" excerpts against live code before proceeding.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug / interaction
- **Planned at**: commit `5f81cee`, 2026-06-12

## Why this matters

`useUnsavedChangesGuard` is a complete, production-ready React Router blocker hook
(`hooks/useUnsavedChangesGuard.ts`) that intercepts sidebar/back navigation and shows
a confirmation dialog when there are unsaved changes. It is imported by **zero** pages
today.

`ConfigPage` has an explicit `isDirty` flag that tracks whether the YAML editor has
unsaved edits. If the user has unsaved edits and clicks any sidebar link, the browser
navigates immediately and the edits are silently discarded. The hook was built
specifically for this case and just needs to be called.

Scope: **ConfigPage only**. `ProviderSheet` also has a dirty guard but it lives
inside a sheet overlay managed via a ref — wiring it requires exposing `isDirty`
through `ProviderSheetHandle`, which is a separate, larger change. Do not touch
`ProviderSheet` in this plan.

## Current state

**`web/src/hooks/useUnsavedChangesGuard.ts:22`** — the full hook (already written,
uses `useBlocker` from React Router, shows a ConfirmationModal via
`useNotificationStore`, returns `allowNextNavigation()`):

```ts
export function useUnsavedChangesGuard(options: UseUnsavedChangesGuardOptions) {
  // ...full implementation, zero callers...
}
// returns: { allowNextNavigation }
```

`UseUnsavedChangesGuardOptions`:
```ts
type UseUnsavedChangesGuardOptions = {
  enabled?: boolean;          // default true
  shouldBlock: boolean | BlockerFunction;
  dialog: UnsavedChangesDialog;  // { title, message, confirmText, cancelText, variant? }
};
```

**`web/src/pages/ConfigPage.tsx:91`** — `isDirty` is derived state:
```ts
const isDirty = dirty || visualDirty;
```

**`web/src/pages/ConfigPage.tsx:130–164`** — `handleConfirmSave` is the success path
after the user confirms the diff modal. On success it calls `setDirty(false)` and
`toast.success(...)`. The `allowNextNavigation()` call from the guard must fire here:

```ts
const handleConfirmSave = async () => {
  setSaving(true);
  try {
    // ... saves YAML, reloads config store ...
    toast.success(t('config_management.save_success'));  // ← add allowNextNavigation() after this
    if (commercialModeChanged) {
      toast.warning(t('notification.commercial_mode_restart_required'));
    }
  } catch (err: unknown) {
    // ...
  } finally {
    setSaving(false);
  }
};
```

**i18n keys** — add to `web/src/i18n/locales/en.json` and `web/src/i18n/locales/vi.json`
under the `config_management` namespace. Check if these keys already exist before
adding; do not duplicate.

Repo convention for hooks: import at component top level, call inside the function body.
Example from existing usage: `useHeaderRefresh(handleHeaderRefresh)` at `ConfigPage.tsx:406`.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Type-check | `cd web && bun run type-check` | exit 0 |
| Tests | `cd web && bun run test:run` | 60 tests pass |
| Lint | `cd web && bun run lint` | exit 0 errors |

## Scope

**In scope** (the only files you should modify):
- `web/src/pages/ConfigPage.tsx`
- `web/src/i18n/locales/en.json`
- `web/src/i18n/locales/vi.json`

**Out of scope** (do NOT touch):
- `web/src/hooks/useUnsavedChangesGuard.ts` — the hook is complete; do not modify it.
- `web/src/features/providers/sheets/ProviderSheet.tsx` — separate task, different pattern.
- `web/src/features/providers/ProvidersWorkbenchPage.tsx` — same, out of scope.
- Any other page or component.

## Git workflow

- Branch: `advisor/003-config-unsaved-guard`
- Commit: `fix(web): wire unsaved-changes guard to ConfigPage`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Check whether the i18n keys already exist

Run:
```bash
grep -n "unsaved_dialog\|leave_without_saving\|discard_changes" web/src/i18n/locales/en.json
```

If any key covering "unsaved changes / discard?" already exists, use it. If not,
add the following under the `"config_management"` object in both locale files:

**en.json** — add inside the `"config_management": { ... }` block:
```json
"unsaved_dialog_title": "Unsaved changes",
"unsaved_dialog_message": "You have unsaved changes to the config. Leave without saving?",
"unsaved_dialog_confirm": "Leave without saving",
"unsaved_dialog_cancel": "Stay and save"
```

**vi.json** — add inside the `"config_management": { ... }` block:
```json
"unsaved_dialog_title": "Thay đổi chưa lưu",
"unsaved_dialog_message": "Bạn có thay đổi cấu hình chưa được lưu. Rời đi mà không lưu?",
"unsaved_dialog_confirm": "Rời đi",
"unsaved_dialog_cancel": "Ở lại và lưu"
```

**Verify**: `grep -n "unsaved_dialog_title" web/src/i18n/locales/en.json web/src/i18n/locales/vi.json` → prints 2 lines.

### Step 2: Import and call the hook in ConfigPage

At the top of `web/src/pages/ConfigPage.tsx`, add to the existing imports:
```ts
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard';
```

Inside the `ConfigPage` function body, after the existing `useHeaderRefresh` call
(around line 406), add:

```ts
const { allowNextNavigation } = useUnsavedChangesGuard({
  shouldBlock: isDirty,
  dialog: {
    title: t('config_management.unsaved_dialog_title'),
    message: t('config_management.unsaved_dialog_message'),
    confirmText: t('config_management.unsaved_dialog_confirm'),
    cancelText: t('config_management.unsaved_dialog_cancel'),
    variant: 'danger',
  },
});
```

**Verify**: `grep -n "useUnsavedChangesGuard\|allowNextNavigation" web/src/pages/ConfigPage.tsx` → prints 2 lines (import + call).

### Step 3: Call allowNextNavigation after successful save

In `handleConfirmSave` (around line 148, after `toast.success(...)`), add a call to
`allowNextNavigation()` so the post-save navigation (e.g. closing the diff modal and
the app immediately navigating somewhere) is not blocked:

```ts
toast.success(t('config_management.save_success'));
allowNextNavigation();  // ← add this line
```

**Verify**: `grep -n "allowNextNavigation" web/src/pages/ConfigPage.tsx` → prints the call inside `handleConfirmSave`.

### Step 4: Run all gates

```bash
cd web && bun run type-check
cd web && bun run test:run
cd web && bun run lint
```

**Verify**: all exit 0.

## Test plan

No automated test for the guard hook exists yet (it relies on `useBlocker` which is
a React Router integration, not a unit-testable pure function). Manual verification:

1. Start the dev server (`make dev` + `make dev-web`).
2. Log in, navigate to Config page, make any edit (add a space).
3. Click a sidebar link — the confirmation dialog should appear.
4. Click "Stay and save" — stays on Config page.
5. Click a sidebar link again — dialog appears.
6. Click "Leave without saving" — navigates away.
7. Navigate back to Config, make an edit, click "Save" → confirm save → navigate to
   another page. Navigation should proceed without a dialog (allowNextNavigation fired).

## Done criteria

- [ ] `grep -n "useUnsavedChangesGuard" web/src/pages/ConfigPage.tsx` prints 2 lines (import + call)
- [ ] `grep -n "allowNextNavigation" web/src/pages/ConfigPage.tsx` prints the call in `handleConfirmSave`
- [ ] `grep -n "unsaved_dialog_title" web/src/i18n/locales/en.json` exits 0
- [ ] `cd web && bun run type-check` exits 0
- [ ] `cd web && bun run test:run` exits 0; 60 tests pass
- [ ] `git status` shows only in-scope files modified
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

- Code at `ConfigPage.tsx:91` (`isDirty`) or `handleConfirmSave` doesn't match the
  excerpts above — drift check failed, stop and report.
- `bun run type-check` introduces new errors not caused by this change.
- The `shouldBlock` wiring requires accessing `isDirty` from outside the component
  scope — it does not; `isDirty` is defined at line 91 inside the component, so
  the hook call and `isDirty` are in the same scope.

## Maintenance notes

- If a second "save" path is added to ConfigPage (e.g., an auto-save feature), call
  `allowNextNavigation()` from that path too.
- If `ProviderSheet` dirty guard is wired later, it will need `ProviderSheetHandle`
  to expose `isDirty` (currently only `confirmDiscardIfDirty` is exposed). Add
  `isDirty: boolean` to the handle interface, then call `useUnsavedChangesGuard` in
  `ProvidersWorkbenchPage` with `shouldBlock: sheetState.open && (sheetRef.current?.isDirty ?? false)`.
- `useUnsavedChangesGuard` uses React Router's `useBlocker` — it only works inside
  a Router context. If ConfigPage is ever rendered outside a Router (e.g., in a test),
  the hook will throw. Wrap it in `enabled={isInRouter}` if needed.
