# Plan: Layout Shell

Phase: layout-shell
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-27

## Goal
Replace the custom MainLayout sidebar with shadcn Sidebar. Rebuild the app shell with Tailwind. Simplify theming to light/dark only. Update useThemeStore.

## Inputs
- shadcn Sidebar component (installed in Phase 2)
- Current `MainLayout.tsx` (765 lines)
- Current `useThemeStore.ts`
- Current `types/index.ts` (Theme type)
- Navigation structure (4 groups, 8 routes, icons)

---

## Wave 1: Theme system simplification
### T1 — Simplify useThemeStore to light/dark
- type: refactor
- inputs:
  - `web/src/stores/useThemeStore.ts`
  - `web/src/types/index.ts` or wherever `Theme` type is defined
- touches:
  - `web/src/stores/useThemeStore.ts` — reduce to `light` | `dark`
  - `web/src/types/` — update Theme type
- avoid:
  - MainLayout (handled in T3)
  - SCSS theme files (deleted in Phase 5)
- steps:
  1. Update `Theme` type to `'light' | 'dark'`
  2. Update `useThemeStore`:
     - Default to `light`
     - Remove `white` and `auto` handling
     - `setTheme` only accepts `light` | `dark`
     - Persist logic unchanged (localStorage)
  3. Update theme application: `data-theme="light"` or `data-theme="dark"` on `<html>`, or use class-based `.dark` for Tailwind dark mode
  4. Decide: use `data-theme` attribute or Tailwind's `class` strategy (`.dark` on `<html>`). shadcn recommends class strategy. Use `.dark` class on `<html>`.
  5. Update `index.css`: change dark theme selector from `[data-theme='dark']` to `.dark`
- expected outputs:
  - Theme store accepts only `light` | `dark`
  - Dark mode toggles via `.dark` class on `<html>`
- verification:
  - `cd web && bun run type-check` — no type errors
  - `grep -r "data-theme.*white" web/src/` — zero matches
  - `grep -r "'auto'" web/src/stores/useThemeStore` — zero matches
- stop if:
  - existing code depends on `data-theme` attribute in ways that can't switch to class
- escalate to:
  - plan phase (may need to keep `data-theme` alongside class)

---

## Wave 2: Rebuild MainLayout with shadcn Sidebar
### T2 — Install and configure shadcn Sidebar
- type: implementation
- inputs:
  - shadcn Sidebar component
  - Navigation structure from current `MainLayout.tsx`
- touches:
  - `web/src/components/layout/MainLayout.tsx` — full rewrite
  - `web/src/App.tsx` — wrap with `SidebarProvider`
- avoid:
  - page components
  - API layer
  - router structure
- steps:
  1. Read shadcn Sidebar docs and component source
  2. Wrap app root (in `App.tsx` or `ProtectedRoute`) with `<SidebarProvider>`
  3. Rewrite `MainLayout.tsx`:
     - **Sidebar**: Use `Sidebar`, `SidebarContent`, `SidebarGroup`, `SidebarGroupLabel`, `SidebarMenu`, `SidebarMenuItem`, `SidebarMenuButton`
     - Map 4 nav groups (operate, gateway, observe, control) to `SidebarGroup`
     - Map nav items to `SidebarMenuItem` with `SidebarMenuButton` containing `NavLink`
     - Preserve sidebar icons (existing SVG icons)
     - Add `SidebarHeader` with brand logo + title
     - Add `SidebarFooter` if needed for version info
     - Use `SidebarTrigger` for collapse toggle
     - Use `SidebarRail` for hover-to-expand on collapsed state
  4. **Header**: Rebuild with Tailwind:
     - Refresh button, theme toggle, language toggle, logout button
     - Use shadcn `DropdownMenu` for language picker (replaces hand-rolled popover)
     - Use shadcn `DropdownMenu` for theme picker
  5. **Content area**: `<main>` with Tailwind classes
     - Preserve `contentRef` for ResizeObserver (`--content-center-x`)
     - Preserve `headerRef` for ResizeObserver (`--header-height`)
     - Preserve `PageTransition` render prop
  6. **Theme picker**: Reduce `THEME_CARDS` to 2 entries (light + dark) with design-token preview colors
  7. Remove all SCSS class references (no more `.app-shell`, `.main-header`, `.sidebar`, etc.)
- expected outputs:
  - `MainLayout.tsx` rebuilt entirely with Tailwind + shadcn components
  - Sidebar collapses/expands, works on mobile as sheet
  - Theme toggle switches between light and dark
  - Language picker works via DropdownMenu
- verification:
  - `cd web && bun run type-check` — zero errors
  - `cd web && bun run dev` — visual check: sidebar renders, navigation works, collapse works
  - Mobile viewport: sidebar slides as sheet
  - Theme toggle: switches between light/dark
  - Language toggle: switches between 4 locales
- stop if:
  - shadcn Sidebar's SidebarProvider conflicts with router state
  - PageTransition breaks with new layout structure
- escalate to:
  - user clarification (Sidebar alternative)

### T3 — Preserve dynamic layout measurements
- type: implementation
- inputs:
  - ResizeObserver logic from current MainLayout
  - `--header-height` and `--content-center-x` CSS variables
- touches:
  - `web/src/components/layout/MainLayout.tsx` — ensure refs + ResizeObserver preserved
- avoid:
  - changing what the CSS variables are used for (downstream consumers)
- steps:
  1. Verify `headerRef` is attached to the header element in the new layout
  2. Verify `contentRef` is attached to the main content area
  3. Verify both ResizeObserver callbacks still set CSS variables correctly
  4. Verify `--header-height` is used by `useEdgeSwipeBack` or other hooks — if so, ensure the value is correct
- expected outputs:
  - `--header-height` and `--content-center-x` CSS variables set correctly at runtime
- verification:
  - Open dev tools → inspect `<html>` element → verify CSS variables are set
  - Resize window → variables update
- stop if:
  - CSS variables not being used by anything → can remove (simplify)
- escalate to:
  - N/A

---

## Wave 3: Integration verification
### T4 — Full layout integration test
- type: test
- inputs:
  - All changes from T1–T3
- touches:
  - nothing (verification only)
- avoid:
  - any code changes
- steps:
  1. `cd web && bun run type-check` — zero errors
  2. `cd web && bun run build` — succeeds
  3. `cd web && bun run dev` — visual checks:
     - All 8 navigation items render and link to correct routes
     - Sidebar collapse/expand works on desktop
     - Sidebar sheet works on mobile
     - Theme toggle switches light/dark
     - Language toggle works for all 4 locales
     - Logout button works
     - Refresh button works
     - Page transitions animate between routes
  4. Check that pages still render their content (even if unstyled — page rebuild is next phase)
- expected outputs:
  - Layout shell fully functional
- verification:
  - All checks above pass
- stop if:
  - navigation is broken or routes don't resolve
- escalate to:
  - plan phase (layout-shell may need re-approach)

## Risks / Watch-fors
- shadcn Sidebar uses `useSidebar()` context hook — all sidebar consumers must be inside `SidebarProvider`
- The current `getRouteOrder` and `getTransitionVariant` functions are used by PageTransition — preserve them in the new layout
- Language picker currently uses `LANGUAGE_ORDER` and `LANGUAGE_LABEL_KEYS` constants — keep using them
- The `config?.loggingToFile` conditional for showing the Logs nav item must be preserved
