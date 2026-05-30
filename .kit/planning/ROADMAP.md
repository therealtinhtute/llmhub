# ROADMAP: Web UI Cleanup — Config Tabs Layout Fix + PageTransition Removal

## Planning Basis
- source spec: `.kit/planning/SPEC.md`
- planning mode: `full`
- recommended entry phase: `config-tabs-layout`
- execution mode: sequential (independent file surfaces; each phase verifiable separately)

## Entry Phase
`config-tabs-layout`

## Phase 1: config-tabs-layout
**Goal:** Replace the 3-col/2-col grid TabsList with a single horizontal scrollable row and remove the redundant "快速跳转" info bar.

**Deliverables:**
- `VisualConfigEditor.tsx` with flex-row TabsList and no info bar
- Compact tab triggers: icon + label + optional error badge (no index prefix)

**Dependencies:** none — first phase, single file

**Risks / Watch-fors:**
- Tab triggers may overflow or compress too much on narrow screens — verify with browser dev tools at 320px width
- Removing the index prefix (`01`, `02`, etc.) changes the visual identity — confirm this is acceptable during verification

---

## Phase 2: remove-page-transition
**Goal:** Delete PageTransition component and its context entirely. Clean up MainLayout and 4 consumer pages that reference the now-always-true `isCurrentLayer` guard.

**Deliverables:**
- `PageTransition.tsx` deleted
- `PageTransitionLayer.ts` deleted
- `MainLayout.tsx` renders `<MainRoutes />` directly; `getRouteOrder` and `getTransitionVariant` removed
- 4 consumer pages cleaned of `usePageTransitionLayer` imports and `isCurrentLayer` guards

**Dependencies:** none — independent of Phase 1 (no file overlap)

**Risks / Watch-fors:**
- `AuthFilesPage` uses `isCurrentLayer` to gate data loading AND polling interval — removing the guard means data always loads and polls; verify no double-fetch or performance regression
- `SecondaryScreenShell` conditionally renders floating actions — removing guard means floating actions always render; confirm no visual overlap issues
- Ensure no other files import from `PageTransition.tsx` or `PageTransitionLayer.ts` beyond the known 6
