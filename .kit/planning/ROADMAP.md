# ROADMAP: LLMHub Phase 2 — Web UI Rebuild with shadcn + Design Tokens

## Planning Basis
- source spec: `.kit/planning/SPEC.md`
- planning mode: `full`
- entry phase: `tailwind-foundation`
- execution mode: sequential (each phase depends on prior)

---

## Phase 1: tailwind-foundation
**Goal:** Install Tailwind CSS v4 + shadcn/ui into the Vite build pipeline. Map design tokens to CSS custom properties. Set up light/dark theming. Add Google Fonts. Verify single-file build still works.

**Deliverables:**
- Tailwind CSS v4 installed and configured in Vite
- shadcn/ui initialized with `components.json`
- `src/index.css` with `@theme` block mapping all design tokens (colors, typography, spacing, shadows, radius)
- Light + dark theme CSS custom properties
- Google Fonts CDN links in `index.html`
- `bun run build` produces single HTML with Tailwind CSS inlined

**Dependencies:**
- None (entry phase)

**Risks / Watch-fors:**
- Tailwind v4 uses CSS-first config (`@theme` in CSS, no `tailwind.config.js`) — different from v3 docs
- `vite-plugin-singlefile` must inline Tailwind's generated CSS; verify early
- shadcn CLI `init` may scaffold files that conflict with existing `src/` structure

---

## Phase 2: shadcn-primitives
**Goal:** Install all needed shadcn components via CLI. Customize each component to match design tokens (0 radius, hard shadows, blueprint blue). Replace all 16 hand-rolled UI primitives with shadcn equivalents. Migrate notifications to Sonner.

**Deliverables:**
- 16 shadcn components installed and customized: Button, Card, Input, Select, Dialog, AlertDialog, Sheet, Table, Skeleton, Collapsible, Switch, Empty, Combobox, Spinner, Checkbox, Sonner
- All hand-rolled `components/ui/` files replaced
- `useNotificationStore` replaced with Sonner toast calls
- `NotificationContainer` component removed
- All existing imports updated to new components

**Dependencies:**
- Phase 1 complete (Tailwind + shadcn initialized, design tokens mapped)

**Risks / Watch-fors:**
- shadcn `Select` uses Radix, which has different API than current custom Select — consumer components need prop changes
- `Modal` → `Dialog` migration: current modal uses motion animations — need to verify Dialog plays well with motion or use Dialog's built-in animations
- Sonner migration touches every component that calls `showNotification()` — search and replace carefully
- `AutocompleteInput` → `Combobox` may have different selection/filtering behavior

---

## Phase 3: layout-shell
**Goal:** Replace the custom `MainLayout` sidebar with shadcn Sidebar. Rebuild the app shell (header, sidebar, content area) with Tailwind utilities. Simplify theme picker to light/dark only. Update `useThemeStore`.

**Deliverables:**
- shadcn `Sidebar` component installed and wired as the main navigation
- `MainLayout.tsx` rebuilt with Tailwind classes (no SCSS)
- Theme picker shows only light/dark options
- `useThemeStore` updated: only `light` | `dark` types
- `THEME_CARDS` array reduced to 2 entries
- Mobile sidebar (sheet mode) working
- Sidebar collapse/expand working

**Dependencies:**
- Phase 2 complete (shadcn Button, Sheet, and other primitives available)

**Risks / Watch-fors:**
- shadcn Sidebar has its own state management (SidebarProvider) — must coexist with existing Zustand stores
- Current sidebar uses `sidebarCollapsed` state with ResizeObserver for `--content-center-x` — need to verify shadcn Sidebar exposes similar hooks
- `PageTransition` component depends on content ref from MainLayout — ensure ref forwarding works with new layout
- Navigation group labels and icons need remapping to shadcn Sidebar's `SidebarGroup` / `SidebarMenuItem` structure

---

## Phase 4: page-rebuild
**Goal:** Rebuild all 11 page/feature components with Tailwind utilities and shadcn components. Remove all SCSS module imports. Preserve all business logic, API calls, and i18n keys.

**Deliverables:**
- All 11 pages rebuilt: LoginPage, DashboardPage, ProvidersWorkbenchPage, AuthFilesPage, AuthFilesOAuthExcludedEditPage, AuthFilesOAuthModelAliasEditPage, OAuthPage, QuotaPage, ConfigPage, LogsPage, SystemPage
- All provider feature sub-components rebuilt (ProviderSheet, ProviderHeaderCard, ProviderResourceTable, ProviderCategoryList, ProviderResourcePanel, ProviderStatusBar, OpenAIBrandToolbar)
- Config components rebuilt (ConfigSection, ConfigSourceEditor, VisualConfigEditor, DiffModal)
- ModelMappingDiagram components rebuilt
- QuotaCard/QuotaSection rebuilt
- SecondaryScreenShell rebuilt
- CodeMirror integration preserved (restyled with Tailwind)
- Zero SCSS module imports remaining in any TSX file

**Dependencies:**
- Phase 3 complete (layout shell provides the app frame)

**Risks / Watch-fors:**
- ProvidersWorkbenchPage is the most complex page (~2k SCSS lines) — highest risk of regression
- AuthFilesPage has the largest SCSS module (2045 lines) — many edge-case styles
- ConfigPage CodeMirror styling: CodeMirror has its own CSS that may conflict with Tailwind's reset
- LogsPage has scroll-locked auto-scroll behavior that depends on specific CSS
- ModelMappingDiagram has complex CSS grid/flex layouts

---

## Phase 5: cleanup-verify
**Goal:** Delete all SCSS files and sass dependency. Remove SCSS config from Vite. Run full verification suite: build, type-check, visual check, responsive check, i18n, theme toggle, animations, Go build.

**Deliverables:**
- All 20 `.module.scss` files deleted
- All 7 global SCSS files deleted
- `sass` removed from `package.json` devDependencies
- SCSS preprocessor config removed from `vite.config.ts`
- `bun run build` succeeds (single HTML)
- `bun run type-check` passes
- All 10 routes render in both themes
- Mobile responsive on all pages
- i18n works for all 4 locales
- Page transitions work
- CodeMirror works
- `make build` (Go binary) succeeds

**Dependencies:**
- Phase 4 complete (all pages rebuilt, no SCSS imports remain)

**Risks / Watch-fors:**
- Hidden SCSS imports that weren't caught during page rebuild
- Global CSS classes from `components.scss` used by components not in the explicit rebuild list
- CodeMirror CSS may have been loaded via SCSS — verify it has standalone CSS loading
- `PageTransition.scss` and `SplashScreen.scss` are non-module SCSS files — need Tailwind equivalents
