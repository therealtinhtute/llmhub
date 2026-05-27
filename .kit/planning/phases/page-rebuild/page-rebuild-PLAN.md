# Plan: Page Rebuild

Phase: page-rebuild
Status: ready
Wave Count: 4
Execution Owner: work
Updated At: 2026-05-27

## Goal
Rebuild all 11 page/feature components with Tailwind utilities and shadcn components. Remove all SCSS module imports from TSX files. Preserve all business logic, API calls, and i18n keys.

## Inputs
- All shadcn components from Phase 2
- Layout shell from Phase 3
- Current page source files and their `.module.scss` companions (for reference)
- Current provider feature sub-components

---

## Wave 1: Simple pages (low complexity, few SCSS lines)
### T1 — Rebuild LoginPage
- type: implementation
- inputs:
  - `web/src/pages/LoginPage.tsx` (current)
  - `web/src/pages/LoginPage.module.scss` (323 lines, reference)
- touches:
  - `web/src/pages/LoginPage.tsx` — replace all className with Tailwind
- avoid:
  - changing login logic, API calls, auth store usage
- steps:
  1. Read current LoginPage.tsx and LoginPage.module.scss
  2. Remove `import styles from './LoginPage.module.scss'`
  3. Replace all `styles.xxx` with equivalent Tailwind utility strings
  4. Use shadcn Card for the login card container
  5. Use shadcn Input for form fields
  6. Use shadcn Button for submit
  7. Apply design token fonts: VT323 for heading, Source Serif 4 for body
  8. Use hard shadow on the login card
- expected outputs:
  - LoginPage renders with design token aesthetic, no SCSS imports
- verification:
  - `cd web && bun run type-check` — no errors in LoginPage
  - `grep "module.scss" web/src/pages/LoginPage.tsx` — zero matches
- stop if:
  - N/A
- escalate to:
  - N/A

### T2 — Rebuild SystemPage
- type: implementation
- inputs:
  - `web/src/pages/SystemPage.tsx`
  - `web/src/pages/SystemPage.module.scss` (374 lines)
- touches:
  - `web/src/pages/SystemPage.tsx`
- avoid:
  - changing system info API calls or data display logic
- steps:
  1. Read current files
  2. Remove SCSS import, replace with Tailwind utilities
  3. Use shadcn Card, Table, Badge for layout
- expected outputs:
  - SystemPage styled with Tailwind, no SCSS
- verification:
  - `grep "module.scss" web/src/pages/SystemPage.tsx` — zero
  - type-check passes
- stop if: N/A
- escalate to: N/A

### T3 — Rebuild OAuthPage
- type: implementation
- inputs:
  - `web/src/pages/OAuthPage.tsx`
  - `web/src/pages/OAuthPage.module.scss` (247 lines)
- touches:
  - `web/src/pages/OAuthPage.tsx`
- avoid:
  - changing OAuth API calls or state
- steps:
  1. Read, remove SCSS, replace with Tailwind + shadcn
- expected outputs:
  - OAuthPage styled with Tailwind
- verification:
  - `grep "module.scss" web/src/pages/OAuthPage.tsx` — zero
  - type-check passes
- stop if: N/A
- escalate to: N/A

### T4 — Rebuild PlaceholderPage
- type: implementation
- inputs:
  - `web/src/pages/PlaceholderPage.tsx`
  - `web/src/pages/PlaceholderPage.module.scss`
- touches:
  - `web/src/pages/PlaceholderPage.tsx`
- avoid: N/A
- steps:
  1. Read, remove SCSS, replace with Tailwind
- expected outputs:
  - PlaceholderPage styled with Tailwind
- verification:
  - `grep "module.scss" web/src/pages/PlaceholderPage.tsx` — zero
- stop if: N/A
- escalate to: N/A

---

## Wave 2: Medium pages + shared components
### T5 — Rebuild DashboardPage
- type: implementation
- inputs:
  - `web/src/pages/DashboardPage.tsx`
  - `web/src/pages/DashboardPage.module.scss` (567 lines)
- touches:
  - `web/src/pages/DashboardPage.tsx`
- avoid:
  - changing dashboard data logic
- steps:
  1. Read both files
  2. Remove SCSS import
  3. Rebuild stat cards with shadcn Card
  4. Rebuild grid layout with Tailwind grid utilities
  5. Ensure responsive breakpoints preserved
- expected outputs:
  - DashboardPage with Tailwind grid, shadcn Cards
- verification:
  - `grep "module.scss" web/src/pages/DashboardPage.tsx` — zero
  - type-check passes
- stop if: N/A
- escalate to: N/A

### T6 — Rebuild QuotaPage + quota components
- type: implementation
- inputs:
  - `web/src/pages/QuotaPage.tsx`
  - `web/src/pages/QuotaPage.module.scss` (717 lines)
  - `web/src/components/quota/` — QuotaCard, QuotaSection, useGridColumns, useQuotaLoader, quotaConfigs
- touches:
  - `web/src/pages/QuotaPage.tsx`
  - `web/src/components/quota/*.tsx`
- avoid:
  - changing quota API calls or calculation logic
- steps:
  1. Read all files
  2. Remove SCSS, replace with Tailwind + shadcn Card, Badge, Progress
  3. Rebuild grid layout with Tailwind responsive grid
- expected outputs:
  - QuotaPage and components with Tailwind
- verification:
  - `grep -r "module.scss" web/src/pages/QuotaPage.tsx web/src/components/quota/` — zero
  - type-check passes
- stop if: N/A
- escalate to: N/A

### T7 — Rebuild ConfigPage + config components
- type: implementation
- inputs:
  - `web/src/pages/ConfigPage.tsx`
  - `web/src/pages/ConfigPage.module.scss` (395 lines)
  - `web/src/components/config/` — ConfigSection, ConfigSourceEditor, VisualConfigEditor, VisualConfigEditorBlocks, DiffModal
  - `web/src/components/config/*.module.scss`
- touches:
  - `web/src/pages/ConfigPage.tsx`
  - `web/src/components/config/*.tsx`
- avoid:
  - changing CodeMirror integration or YAML parsing logic
  - changing config API calls
- steps:
  1. Read all files
  2. Remove SCSS imports from all config components
  3. Replace with Tailwind utilities
  4. DiffModal: migrate from hand-rolled Modal to shadcn Dialog
  5. ConfigSection: use shadcn Card + Collapsible
  6. VisualConfigEditor: Tailwind form layout
  7. CodeMirror wrapper: add Tailwind classes for container; leave CodeMirror's internal CSS alone
  8. Add `[&_.cm-editor]:rounded-none [&_.cm-editor]:border-border` for CodeMirror scoping
- expected outputs:
  - Config page and components with Tailwind, CodeMirror still functional
- verification:
  - `grep -r "module.scss" web/src/pages/ConfigPage.tsx web/src/components/config/` — zero
  - type-check passes
  - CodeMirror editor renders and YAML editing works
- stop if:
  - CodeMirror CSS breaks with Tailwind preflight
- escalate to:
  - plan phase (may need `@layer` exclusion for CodeMirror)

### T8 — Rebuild LogsPage
- type: implementation
- inputs:
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/LogsPage.module.scss` (724 lines)
  - `web/src/pages/hooks/` — log parsing, types, filters, scroller
- touches:
  - `web/src/pages/LogsPage.tsx`
- avoid:
  - changing log parsing, filtering, or auto-scroll logic
- steps:
  1. Read files
  2. Remove SCSS, replace with Tailwind
  3. Use shadcn Table for log entries if applicable, or Tailwind flex for log viewer
  4. Preserve auto-scroll behavior (scroll-locked)
  5. Preserve filter controls with shadcn Select/Input
- expected outputs:
  - LogsPage with Tailwind, auto-scroll preserved
- verification:
  - `grep "module.scss" web/src/pages/LogsPage.tsx` — zero
  - type-check passes
- stop if: N/A
- escalate to: N/A

### T9 — Rebuild common components (SecondaryScreenShell, SplashScreen, PageTransition)
- type: implementation
- inputs:
  - `web/src/components/common/SecondaryScreenShell.tsx` + `.module.scss`
  - `web/src/components/common/SplashScreen.tsx` + `.scss`
  - `web/src/components/common/PageTransition.tsx` + `.scss`
- touches:
  - All 3 component files
- avoid:
  - changing animation logic in PageTransition
  - changing motion library usage
- steps:
  1. SecondaryScreenShell: remove SCSS, use Tailwind for container layout
  2. SplashScreen: convert CSS animation to Tailwind `animate-*` or inline keyframes in `index.css`
  3. PageTransition: convert transition CSS to inline styles or Tailwind; motion handles the actual animation, CSS just sets initial states
- expected outputs:
  - All 3 components SCSS-free
- verification:
  - `grep -r "\.scss" web/src/components/common/` — zero (from imports)
  - type-check passes
  - Page transitions still animate
- stop if:
  - PageTransition animation breaks
- escalate to:
  - user clarification (animation approach)

---

## Wave 3: Complex pages
### T10 — Rebuild AuthFilesPage + sub-pages
- type: implementation
- inputs:
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/AuthFilesPage.module.scss` (2045 lines — largest SCSS module)
  - `web/src/pages/AuthFilesOAuthExcludedEditPage.tsx` + `.module.scss` (220 lines)
  - `web/src/pages/AuthFilesOAuthModelAliasEditPage.tsx` + `.module.scss` (225 lines)
  - `web/src/features/authFiles/` — uiState, constants
  - `web/src/components/modelAlias/` — ModelMappingDiagram + sub-components + `.module.scss` (359 lines)
- touches:
  - All page files listed above
  - `web/src/components/modelAlias/*.tsx`
- avoid:
  - changing auth file API calls or data logic
  - changing model mapping business logic
- steps:
  1. Read all files carefully — this is the most SCSS-heavy page (2045 lines)
  2. AuthFilesPage: remove SCSS, rebuild with Tailwind + shadcn Table, Card, Badge, Button, Dialog
  3. AuthFilesOAuthExcludedEditPage: remove SCSS, rebuild with Tailwind + shadcn form components
  4. AuthFilesOAuthModelAliasEditPage: remove SCSS, rebuild with Tailwind + shadcn
  5. ModelMappingDiagram: rebuild CSS grid layout with Tailwind grid; preserve context menu with shadcn DropdownMenu
  6. ModelMappingDiagramModals: migrate to shadcn Dialog
- expected outputs:
  - All auth file pages and model mapping components SCSS-free
- verification:
  - `grep -r "module.scss" web/src/pages/AuthFiles* web/src/components/modelAlias/` — zero
  - type-check passes
- stop if:
  - AuthFilesPage layout too complex for Tailwind utilities (>100 lines of custom CSS needed)
- escalate to:
  - plan phase (may need `@apply` blocks for complex layouts)

### T11 — Rebuild ProvidersWorkbenchPage + all sub-components
- type: implementation
- inputs:
  - `web/src/features/providers/ProvidersWorkbenchPage.tsx` + `.module.scss`
  - `web/src/features/providers/components/` — ProviderHeaderCard, ProviderCategoryList, ProviderResourceTable, ProviderResourcePanel, OpenAIBrandToolbar + their `.module.scss` files
  - `web/src/features/providers/sheets/` — ProviderSheet and forms + `.module.scss` (690 lines)
  - `web/src/components/providers/ProviderStatusBar.tsx` + `.module.scss` (157 lines)
- touches:
  - All provider feature component files
  - `web/src/components/providers/ProviderStatusBar.tsx`
- avoid:
  - changing provider adapters/descriptors logic
  - changing useProviderWorkbench hook
  - changing provider API calls
- steps:
  1. Read all provider component files and their SCSS modules
  2. ProvidersWorkbenchPage: remove SCSS, rebuild with Tailwind grid + shadcn components
  3. ProviderHeaderCard: shadcn Card + Badge
  4. ProviderCategoryList: Tailwind flex/grid
  5. ProviderResourceTable: shadcn Table
  6. ProviderResourcePanel: Tailwind layout + shadcn components
  7. OpenAIBrandToolbar: Tailwind flex
  8. ProviderSheet (slide-out panel): shadcn Sheet with form content
  9. Sheet forms (sharedForm.module.scss — 690 lines): rebuild with Tailwind form utilities + shadcn Input, Select, Switch
  10. ProviderStatusBar: Tailwind flex + shadcn Badge
- expected outputs:
  - All provider components SCSS-free
  - Provider workbench fully functional: list, filter, select, sheet open/close, form editing
- verification:
  - `grep -r "module.scss" web/src/features/providers/ web/src/components/providers/` — zero
  - type-check passes
- stop if:
  - type errors exceed 30 in provider components
- escalate to:
  - plan phase (may need to map provider component tree before continuing)

---

## Wave 4: Full page verification
### T12 — Cross-page type-check and build
- type: test
- inputs:
  - All changes from T1–T11
- touches:
  - nothing (verification only)
- avoid:
  - any code changes
- steps:
  1. `cd web && bun run type-check` — zero errors
  2. `cd web && bun run build` — succeeds
  3. `grep -r "module.scss" web/src/` — zero matches in any TSX/TS file
  4. `grep -r "from.*styles/" web/src/` — check if any global SCSS imports remain in TSX files
  5. `cd web && bun run dev` — visual spot check all 10 routes in browser
- expected outputs:
  - Zero SCSS imports in any TSX file
  - Clean build and type-check
  - All routes render
- verification:
  - All commands above pass
- stop if:
  - any route doesn't render
- escalate to:
  - plan phase (revisit the broken page)

## Risks / Watch-fors
- AuthFilesPage (2045 SCSS lines) is the highest-risk page — plan extra time
- ProvidersWorkbenchPage provider sheet forms (690 SCSS lines) have many form variants per provider — each needs testing
- CodeMirror container in ConfigPage needs careful scoping to avoid Tailwind preflight conflicts
- ModelMappingDiagram uses complex CSS grid with named areas — translate carefully
- PageTransition CSS must work with motion's AnimatePresence — test transitions between routes after rebuild
