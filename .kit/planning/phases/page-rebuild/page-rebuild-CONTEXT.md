# Context: Page Rebuild

Phase: page-rebuild
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: visual inspection, functional check, responsive check, type-check

## Goal
Rebuild all 11 page/feature components with Tailwind utilities and shadcn components. Remove all SCSS module imports from TSX files. Preserve all business logic, API calls, state subscriptions, and i18n keys.

## Scope Boundary
### Allowed Surfaces
- `web/src/pages/*.tsx` — all 11 pages
- `web/src/features/providers/` — all provider feature components and sub-components
- `web/src/components/config/` — ConfigSection, ConfigSourceEditor, VisualConfigEditor, VisualConfigEditorBlocks, DiffModal
- `web/src/components/modelAlias/` — ModelMappingDiagram and sub-components
- `web/src/components/quota/` — QuotaCard, QuotaSection, quotaConfigs, useGridColumns, useQuotaLoader
- `web/src/components/common/SecondaryScreenShell.tsx` — restyle
- `web/src/components/common/SplashScreen.tsx` — restyle (convert from `.scss` to Tailwind)
- `web/src/components/common/PageTransition.tsx` — convert animation CSS from `.scss` to Tailwind/inline
- `web/src/components/providers/` — ProviderStatusBar and utilities

### Forbidden Surfaces
- Zustand stores (except removing SCSS class references)
- API layer / typed callers
- i18n locale files
- Router structure
- motion library internals
- CodeMirror library code (only restyle its container/wrapper)

## Spec Hooks
- Req 22–32: All page rebuilds
- Constraint: i18n keys unchanged
- Constraint: keep motion library
- Constraint: preserve CodeMirror integration

## Locked Decisions
- Each page is rebuilt in-place: same file path, same export, same props
- Business logic (hooks, API calls, state subscriptions) stays untouched — only JSX markup and className changes
- SCSS module `styles.xyz` references replaced with Tailwind utility strings
- Complex CSS grid/flex layouts translated to Tailwind grid/flex utilities
- CodeMirror wrapper gets Tailwind classes; CodeMirror's own theme CSS untouched
- Provider adapters/descriptors logic preserved exactly
- Model mapping diagram layout rebuilt with Tailwind grid

## Assumptions
- All shadcn primitives are available from Phase 2
- Layout shell is stable from Phase 3
- Tailwind utility classes can express all current SCSS module layouts
- CodeMirror's built-in CSS doesn't conflict with Tailwind's preflight reset (may need `@layer` adjustments)

## Canonical Refs
- Page files: `web/src/pages/*.tsx`
- Provider features: `web/src/features/providers/`
- Config components: `web/src/components/config/`
- SCSS modules (reference for what to translate): all `.module.scss` files

## Rejected Options
- Gradual page-by-page migration with SCSS coexistence — spec mandates big-bang; no hybrid period

## Deferred Ideas
- Dashboard charts (shadcn Chart)
- Data table with sorting/filtering on providers
- Form validation library

## Escalate If
- A page's CSS is too complex for Tailwind utilities (>100 lines of custom CSS needed) → consider extracting to a Tailwind `@apply` block or custom component class
- CodeMirror styling breaks after Tailwind preflight → may need to scope preflight exclusion
- ProvidersWorkbenchPage type errors exceed 30 → stop and map the component tree before continuing
