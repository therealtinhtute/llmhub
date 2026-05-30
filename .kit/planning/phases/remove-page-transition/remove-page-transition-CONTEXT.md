# Context: Remove PageTransition

Phase: remove-page-transition
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: type-check + visual (all routed pages render correctly)

## Goal
Delete the PageTransition component and its context system entirely. Update MainLayout to render routes directly. Remove `isCurrentLayer` guards from 4 consumer pages.

## Scope Boundary
### Allowed Surfaces
- `web/src/components/common/PageTransition.tsx` — delete
- `web/src/components/common/PageTransitionLayer.ts` — delete
- `web/src/components/layout/MainLayout.tsx` — remove PageTransition usage, `getRouteOrder`, `getTransitionVariant`
- `web/src/pages/AuthFilesPage.tsx` — remove `isCurrentLayer` guard
- `web/src/pages/ConfigPage.tsx` — remove `isCurrentLayer` guard
- `web/src/features/providers/ProvidersWorkbenchPage.tsx` — remove `isCurrentLayer` guard
- `web/src/components/common/SecondaryScreenShell.tsx` — remove `isCurrentLayer` guard

### Forbidden Surfaces
- `VisualConfigEditor.tsx` — belongs to Phase 1
- `motion` package or any other animation code outside PageTransition
- Router configuration or route definitions
- Any backend files

## Spec Hooks
- Task 2: "Delete PageTransition.tsx + PageTransitionLayer.ts"
- Task 2: "MainLayout.tsx renders <MainRoutes /> directly"
- Task 2: "4 consumers updated to remove usePageTransitionLayer imports and isCurrentLayer guards"

## Locked Decisions
- Full removal — not simplification or gutting internals
- `MainRoutes` called without `location` prop — uses React Router's current location via `useRoutes`
- `getRouteOrder` and `getTransitionVariant` are deleted, not kept as dead code
- Consumer guards removed: data loading, polling, and floating actions become unconditional

## Assumptions
- `MainRoutes` works correctly without explicit `location` prop — confirmed: `useRoutes(mainRoutes, location)` with `location` optional (line 31-32 of MainRoutes.tsx)
- No other files beyond the known 6 import from PageTransition or PageTransitionLayer — verified via grep
- Removing `isCurrentLayer` polling guard from `AuthFilesPage` won't cause performance issues — data was always loading anyway since the page is always "current"
- `contentRef` scroll container in MainLayout is still passed correctly without PageTransition — it wraps the routes directly

## Canonical Refs
- `.kit/planning/SPEC.md` — Task 2
- `web/src/components/common/PageTransition.tsx` — 224 lines, to be deleted
- `web/src/components/common/PageTransitionLayer.ts` — 22 lines, to be deleted
- `web/src/components/layout/MainLayout.tsx` — lines 11, 464-469 (import + usage), lines 244-293 (`getRouteOrder` + `getTransitionVariant`)

## Rejected Options
- Gut internals but keep shell: preserves dead abstractions, consumers still import a useless context
- Keep context for future re-enablement: YAGNI — if animations return, the component can be rebuilt from git history

## Deferred Ideas
- Scroll restoration via React Router `<ScrollRestoration />` for custom scroll containers
- Re-add page transitions with a simpler approach (CSS-only)

## Escalate If
- Grep reveals additional consumers of PageTransition or PageTransitionLayer beyond the known 6 files
- `MainRoutes` without `location` prop causes rendering issues (route mismatch, flash of wrong content)
- Removing `isCurrentLayer` guard causes visible performance regression (runaway polling, etc.)
