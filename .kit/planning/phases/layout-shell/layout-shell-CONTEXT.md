# Context: Layout Shell

Phase: layout-shell
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: visual inspection, responsive check, theme toggle

## Goal
Replace the custom MainLayout sidebar with shadcn Sidebar component. Rebuild the entire app shell (header, sidebar, content area) with Tailwind utilities. Simplify theme system to light/dark only. Update useThemeStore.

## Scope Boundary
### Allowed Surfaces
- `web/src/components/layout/MainLayout.tsx` — full rebuild
- `web/src/stores/useThemeStore.ts` — simplify to light | dark
- `web/src/types/` — update Theme type if needed
- `web/src/App.tsx` — wrap with SidebarProvider if needed
- `web/src/components/common/PageTransition.tsx` — adjust ref forwarding if layout structure changes
- `web/src/router/ProtectedRoute.tsx` — minor adjustments if layout wrapper changes

### Forbidden Surfaces
- Individual page components (no page rebuilds yet)
- API layer
- i18n locale files (keys unchanged)
- SCSS files (do not delete yet)
- Zustand stores other than useThemeStore

## Spec Hooks
- Req 21: Replace custom sidebar with shadcn Sidebar
- Req 36: Update useThemeStore for light/dark only
- Req 37: Theme picker shows only light/dark
- Constraint: keep motion library for PageTransition
- Constraint: i18n keys unchanged

## Locked Decisions
- shadcn Sidebar with SidebarProvider wrapping the app
- Navigation groups map to SidebarGroup: operate, gateway, observe, control
- Sidebar icons preserved (existing SVG icons in `MainLayout`)
- Theme picker reduced to 2 cards (light + dark) with design-token colors
- `data-theme` attribute on `<html>` switches between `light` and `dark`
- `auto` theme removed from the UI; can be re-added later as deferred idea
- ResizeObserver for `--header-height` and `--content-center-x` preserved (used by floating elements)

## Assumptions
- shadcn Sidebar provides mobile sheet mode, keyboard shortcuts, and collapse without custom code
- SidebarProvider state doesn't conflict with existing sidebar state in MainLayout
- `PageTransition` render prop pattern works with the new layout structure
- `triggerHeaderRefresh` hook still works after layout rebuild

## Canonical Refs
- Current `MainLayout.tsx`: `web/src/components/layout/MainLayout.tsx` (765 lines)
- Current `useThemeStore`: `web/src/stores/useThemeStore.ts`
- Current `types/index.ts`: Theme type definition
- shadcn Sidebar: https://ui.shadcn.com/docs/components/sidebar

## Rejected Options
- Restyle current sidebar with Tailwind only — loses keyboard shortcuts, mobile sheet mode, and collapsible behavior that shadcn provides for free
- Remove ResizeObserver logic — still needed by floating config panels and provider navigation

## Deferred Ideas
- Command palette for quick navigation (shadcn Command)
- Breadcrumbs in header
- User avatar / profile menu

## Escalate If
- shadcn Sidebar's mobile behavior conflicts with existing edge-swipe-back hook
- PageTransition breaks with new content ref structure → may need to restructure transition layer
- Navigation route ordering logic (`getRouteOrder`, `getTransitionVariant`) doesn't fit shadcn Sidebar's active state API
