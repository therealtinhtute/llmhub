# Plan: Remove PageTransition

Phase: remove-page-transition
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-30

## Goal
Delete PageTransition and its context. Render routes directly in MainLayout. Remove all consumer guards on `isCurrentLayer`.

## Inputs
- `web/src/components/common/PageTransition.tsx` (to delete)
- `web/src/components/common/PageTransitionLayer.ts` (to delete)
- `web/src/components/layout/MainLayout.tsx`
- 4 consumer files (AuthFilesPage, ConfigPage, ProvidersWorkbenchPage, SecondaryScreenShell)
- `.kit/planning/SPEC.md` Task 2

## Wave 1
### T1 — Update MainLayout to render routes directly
- type: refactor
- inputs:
  - `web/src/components/layout/MainLayout.tsx`
- touches:
  - Import block (remove PageTransition import)
  - `getRouteOrder` function (lines 244-279, delete)
  - `getTransitionVariant` function (lines 281-293, delete)
  - JSX: replace `<PageTransition render={...} .../>` with `<MainRoutes />`
- avoid:
  - `contentRef` — keep as-is (used for scroll container styling)
  - Sidebar, nav, and layout structure
- steps:
  1. Read `MainLayout.tsx` fully
  2. Remove `import { PageTransition } from '@/components/common/PageTransition'`
  3. Delete the `getRouteOrder` function definition
  4. Delete the `getTransitionVariant` function definition
  5. Replace `<PageTransition render={(location) => <MainRoutes location={location} />} getRouteOrder={getRouteOrder} getTransitionVariant={getTransitionVariant} scrollContainerRef={contentRef} />` with `<MainRoutes />`
  6. Remove `navOrder` if it's only used by `getRouteOrder` (check first)
  7. Clean up any now-unused imports (`useCallback` if only used by `getTransitionVariant`)
- expected outputs:
  - MainLayout renders `<MainRoutes />` directly inside the content div
  - No PageTransition import or usage
  - No `getRouteOrder` or `getTransitionVariant` functions
- verification:
  - `cd web && npx tsc --noEmit` — no type errors
  - Visual: navigate between all pages — dashboard, auth files, config, providers, logs, system — all render correctly
- stop if:
  - `navOrder` is used for sidebar highlighting or other non-transition logic — keep it
  - Route rendering breaks without explicit `location` prop
- escalate to:
  - plan phase (if MainRoutes needs the location prop)

## Wave 2
### T2 — Remove isCurrentLayer guard from consumer pages (parallel — 4 files, no overlap)
- type: refactor
- inputs:
  - Wave 1 complete (MainLayout no longer provides PageTransitionLayerContext)
- touches:
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/ConfigPage.tsx`
  - `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - `web/src/components/common/SecondaryScreenShell.tsx`
- avoid:
  - Changing data fetching logic beyond removing the guard
  - Changing floating action rendering beyond removing the conditional
- steps:
  1. **AuthFilesPage.tsx**: Remove `import { usePageTransitionLayer }`, remove `pageTransitionLayer` and `isCurrentLayer` variables, remove the `if (!isCurrentLayer) return` guard in useEffect, change polling interval from `isCurrentLayer ? 240_000 : null` to `240_000`
  2. **ConfigPage.tsx**: Remove `import { usePageTransitionLayer }`, remove `pageTransitionLayer` and `isCurrentLayer` variables, set `shouldRenderFloatingActions = true` or inline it away
  3. **ProvidersWorkbenchPage.tsx**: Remove `import { usePageTransitionLayer }`, remove `pageTransitionLayer` and `isCurrentLayer` variables, pass `true` directly to `useHeaderRefresh` or remove the guard
  4. **SecondaryScreenShell.tsx**: Remove `import { usePageTransitionLayer }`, remove `pageTransitionLayer` and `isCurrentLayer` variables, set `shouldRenderFloatingAction = Boolean(floatingAction)` directly
- expected outputs:
  - No file imports from `PageTransitionLayer`
  - All guarded behaviors become unconditional
- verification:
  - `cd web && npx tsc --noEmit` — no type errors
  - `grep -rn "PageTransitionLayer\|usePageTransitionLayer\|isCurrentLayer" web/src/ --include="*.tsx" --include="*.ts"` — returns nothing
- stop if:
  - `isCurrentLayer` is used for logic beyond what the spec identified (check full usage before removing)
- escalate to:
  - user clarification

## Wave 3
### T3 — Delete PageTransition and PageTransitionLayer files
- type: refactor
- inputs:
  - Wave 1 + Wave 2 complete (no consumers remain)
- touches:
  - `web/src/components/common/PageTransition.tsx` (delete)
  - `web/src/components/common/PageTransitionLayer.ts` (delete)
- avoid:
  - Deleting any other files
- steps:
  1. Verify no remaining imports: `grep -rn "PageTransition" web/src/ --include="*.tsx" --include="*.ts"` — should return nothing
  2. Delete `web/src/components/common/PageTransition.tsx` via `trash`
  3. Delete `web/src/components/common/PageTransitionLayer.ts` via `trash`
- expected outputs:
  - Both files removed from the codebase
  - No broken imports
- verification:
  - `cd web && npx tsc --noEmit` — no type errors
  - `grep -rn "PageTransition" web/src/ --include="*.tsx" --include="*.ts"` — returns nothing
- stop if:
  - Grep in step 1 reveals unexpected consumers — escalate before deleting
- escalate to:
  - plan phase

## Risks / Watch-fors
- Wave 2 tasks are independent (4 different files) — can execute in parallel
- Wave 3 MUST wait for Wave 1 + 2 — deleting files with active consumers causes build failures
- `navOrder` in MainLayout may have dual use (sidebar highlighting + route ordering) — check before deleting
