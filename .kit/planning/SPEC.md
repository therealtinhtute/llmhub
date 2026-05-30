# SPEC — Web UI Cleanup: Config Tabs Layout Fix + PageTransition Removal

- **Status:** Locked
- **Input Type:** change-request
- **Lane:** normal
- **Risk Flags:** existing-behavior
- **Affected Surfaces:** browser
- **Downstream:** plan
- **Updated At:** 2026-05-30
- **Spec locked by:** brainstorm

## Context

The previous implementation (commit `82dc2ef`) applied Radix Tabs to the config editor but kept the 3-col / 2-col grid layout on the tab triggers — this doesn't match the spec's intent of a single horizontal row. Additionally, PageTransition was stripped of animations but left with a full layer-stack architecture (exit/stacked layers, route-order detection, transition-variant logic) that now does nothing. Four consumer pages still check `isCurrentLayer` from a context that always returns `true`.

This spec corrects both issues.

## In Scope

### Task 1 — Config Tabs: single horizontal row

**Goal:** Replace the 3-col / 2-col grid TabsList with a true single horizontal row of tab triggers.

**Current behavior:**
- `Tabs.List` uses `grid-cols-[repeat(3,minmax(0,1fr))] max-[1200px]:grid-cols-[repeat(2,minmax(0,1fr))]`
- A "快速跳转" info bar above the tabs shows the active section name and validation-blocked badge
- Inactive tab content is unmounted by default (correct — state is lifted to parent)

**New behavior:**
- `Tabs.List` uses `flex flex-row overflow-x-auto` — single scrollable row on all screen sizes
- Remove the "快速跳转" info bar entirely — tabs already convey active state and error badges
- Tab triggers become compact: icon + label + optional error badge, no index number prefix needed
- Inactive tab content stays unmounted (default Radix behavior, unchanged)

**Files touched:**
- `web/src/components/config/VisualConfigEditor.tsx`

---

### Task 2 — Remove PageTransition entirely

**Goal:** Delete the PageTransition component, its context, and all consumer guards. Routes render directly without a transition layer.

**Current behavior:**
- `PageTransition.tsx` maintains a multi-layer stack (current/stacked/exiting) with route-order detection, transition-variant logic, and scroll-position saving — all dead code since animations were stripped
- `PageTransitionLayer.ts` exports context with `status`, `isCurrentLayer`, `isAnimating`
- 4 consumers read `isCurrentLayer` to gate data loading, polling, or floating-action visibility — the value is always `true`
- `MainLayout.tsx` passes `getRouteOrder` and `getTransitionVariant` callbacks to `PageTransition`

**New behavior:**
- `MainLayout.tsx` renders `<MainRoutes />` directly (no `location` prop — uses router's current location)
- `getRouteOrder` and `getTransitionVariant` functions removed from `MainLayout.tsx`
- `PageTransition.tsx` deleted
- `PageTransitionLayer.ts` deleted
- 4 consumers updated to remove `usePageTransitionLayer` imports and `isCurrentLayer` guards:
  - `AuthFilesPage.tsx` — remove layer check; always load data and poll
  - `ConfigPage.tsx` — remove layer check; always show floating actions
  - `ProvidersWorkbenchPage.tsx` — remove layer check; always refresh
  - `SecondaryScreenShell.tsx` — remove layer check; always show floating action

**Files touched:**
- `web/src/components/layout/MainLayout.tsx`
- `web/src/components/common/PageTransition.tsx` (delete)
- `web/src/components/common/PageTransitionLayer.ts` (delete)
- `web/src/pages/AuthFilesPage.tsx`
- `web/src/pages/ConfigPage.tsx`
- `web/src/features/providers/ProvidersWorkbenchPage.tsx`
- `web/src/components/common/SecondaryScreenShell.tsx`

---

## Out of Scope

- Tab content lazy rendering or `forceMount`
- Re-adding animations or transition effects
- Scroll restoration (was broken — saved but never restored after animation strip)
- Removing `motion` package from `package.json`
- Release pipeline / installer changes (completed in prior spec)
- Any backend or API changes

---

## Validation Expectations

### Task 1
1. All 6 tabs visible in a single horizontal row on desktop
2. Tabs scroll horizontally on narrow screens — no wrapping, no multi-column grid
3. Error badges (amber count) still visible on tab triggers
4. No "快速跳转" bar rendered above the tabs
5. Clicking a tab shows its content; switching tabs preserves form state (parent-managed)

### Task 2
1. Route changes work correctly — all pages render
2. `AuthFilesPage` loads data and polls without `isCurrentLayer` guard
3. `ConfigPage` and `SecondaryScreenShell` show floating actions unconditionally
4. `ProvidersWorkbenchPage` refreshes unconditionally
5. No runtime errors from missing `PageTransitionLayerContext`
6. No `PageTransition` or `PageTransitionLayer` imports remain in the codebase
7. `getRouteOrder` and `getTransitionVariant` functions no longer in `MainLayout.tsx`

---

## Key Decisions (with rejected alternatives)

- **Flex row over grid** (chosen): matches shadcn Tabs convention and the original spec intent. *Rejected:* keeping 3-col/2-col grid — was a misimplementation; also rejected 6-col grid — tabs would be too narrow.
- **Remove 快速跳转 bar** (chosen): tabs already show active state and error badges. *Rejected:* keeping for validation-blocked warning — can be shown inline or in tab triggers.
- **Remove PageTransition + context entirely** (chosen): all layer logic is dead, all consumer guards are no-ops. *Rejected:* gutting internals but keeping shell — preserves dead abstractions; *rejected:* keeping context for future re-enablement — YAGNI.
- **Unmount inactive tabs** (chosen): parent manages form state, no data loss. Better performance for 6 content-heavy sections. *Rejected:* `forceMount` — no practical benefit when state is lifted.

---

## Deferred Ideas

- Add subtle fade animation between tab contents
- Scroll restoration via React Router `<ScrollRestoration />` for custom scroll containers
- Tab content lazy rendering with Suspense boundaries
- Keyboard arrow-key navigation between tabs (Radix supports this by default)
