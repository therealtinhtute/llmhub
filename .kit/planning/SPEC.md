# SPEC: LLMHub Phase 2 — Web UI Rebuild with shadcn + Design Tokens

Status: draft
Input Type: change-request
Lane: high-risk
Risk Flags: existing-behavior, multi-domain, public-contract
Affected Surfaces: browser, desktop
Downstream: plan full
Updated At: 2026-05-27

## Source Mode
lock-from-idea

## Source Inputs
- Existing `web/` codebase (~45k lines: TSX/TS/SCSS)
- `design-token.json` (scraped from aiengineeringfromscratch.com)
- shadcn/ui documentation (https://ui.shadcn.com/llms.txt)
- User decisions from brainstorm session

## Scenario
Phase 2 of the LLMHub monorepo initiative. Phase 1 (rebrand + embedded panel) is complete and stable. This phase rebuilds the web UI's styling layer and component system.

## Goal
Replace the entire SCSS Modules + hand-rolled UI component stack with Tailwind CSS v4 + shadcn/ui, applying the AI Engineering from Scratch design token visual language. Simplify theming from 4 variants to 2 (light + dark). Preserve all existing functionality, routes, API integration, state management, and i18n.

## Users / Actors
- Primary maintainer: implementing the rebuild
- Operators: using the management panel to manage LLMHub instances
- Management UI users: managing auths, providers, config, logs, quota, OAuth, system info

## Requirements

### Foundation
1. Install and configure Tailwind CSS v4 in the Vite build pipeline. Tailwind output must inline correctly via `vite-plugin-singlefile`.
2. Initialize shadcn/ui for Vite + React 19. Configure `components.json` with the project's path alias (`@/`).
3. Map design tokens from `design-token.json` to Tailwind v4 `@theme` CSS custom properties (colors, typography, spacing, shadows, border-radius).
4. Add Google Fonts CDN links to `index.html` for VT323, Source Serif 4, and JetBrains Mono.
5. Create a light theme and dark theme using Tailwind v4 CSS custom properties. Remove the `white` and `auto` theme variants.

### Component Replacement
6. Replace hand-rolled `Button` with shadcn `Button`. Map existing variants (primary, secondary, ghost, danger) to shadcn variants styled with design tokens.
7. Replace hand-rolled `Card` with shadcn `Card`.
8. Replace hand-rolled `Input` with shadcn `Input`.
9. Replace hand-rolled `Select` with shadcn `Select`.
10. Replace hand-rolled `Modal` with shadcn `Dialog` + `AlertDialog`.
11. Replace hand-rolled `Sheet` with shadcn `Sheet`.
12. Replace hand-rolled `Table` with shadcn `Table`.
13. Replace hand-rolled `Skeleton` with shadcn `Skeleton`.
14. Replace hand-rolled `Collapsible` with shadcn `Collapsible`.
15. Replace hand-rolled `ToggleSwitch` with shadcn `Switch`.
16. Replace hand-rolled `EmptyState` with shadcn `Empty`.
17. Replace hand-rolled `AutocompleteInput` with shadcn `Combobox`.
18. Replace hand-rolled `LoadingSpinner` with shadcn `Spinner`.
19. Replace hand-rolled `SelectionCheckbox` with shadcn `Checkbox`.
20. Replace hand-rolled `NotificationContainer` + `useNotificationStore` with Sonner toast.
21. Replace custom `MainLayout` sidebar with shadcn `Sidebar` component.

### Page Rebuild
22. Rebuild `LoginPage` with shadcn components + Tailwind utilities.
23. Rebuild `DashboardPage` with shadcn components + Tailwind utilities.
24. Rebuild `ProvidersWorkbenchPage` (most complex page) with shadcn components + Tailwind utilities. Preserve provider adapters/descriptors logic.
25. Rebuild `AuthFilesPage` with shadcn components + Tailwind utilities.
26. Rebuild `AuthFilesOAuthExcludedEditPage` with shadcn components + Tailwind utilities.
27. Rebuild `AuthFilesOAuthModelAliasEditPage` with shadcn components + Tailwind utilities.
28. Rebuild `OAuthPage` with shadcn components + Tailwind utilities.
29. Rebuild `QuotaPage` with shadcn components + Tailwind utilities.
30. Rebuild `ConfigPage` with shadcn components + Tailwind utilities. Preserve CodeMirror integration.
31. Rebuild `LogsPage` with shadcn components + Tailwind utilities.
32. Rebuild `SystemPage` with shadcn components + Tailwind utilities.

### Cleanup
33. Delete all `.module.scss` files (20 files, ~9.8k lines).
34. Delete global SCSS files (`variables.scss`, `themes.scss`, `reset.scss`, `layout.scss`, `global.scss`, `components.scss`, `mixins.scss`).
35. Remove `sass` from devDependencies. Remove SCSS preprocessor config from `vite.config.ts`.
36. Update `useThemeStore` to support only `light` and `dark` themes. Remove `white` and `auto` theme logic.
37. Update `MainLayout` theme picker to show only light/dark options.

### Verification
38. `bun run build` produces a single `dist/index.html` with all Tailwind CSS inlined.
39. `bun run type-check` passes with zero errors.
40. All 10 routes render correctly in both light and dark themes.
41. All existing API integrations (login, providers, auth files, config, logs, quota, OAuth, system) continue to work unchanged.
42. Mobile responsive behavior preserved on all pages.
43. i18n switching works for all 4 locales.
44. Page transitions (motion/framer-motion) work as before.
45. CodeMirror YAML editor and diff modal work correctly.
46. `make build` (full Go binary build) succeeds with the new frontend.

## Boundaries

### In Scope
- Full replacement of SCSS with Tailwind CSS v4
- Full replacement of hand-rolled UI primitives with shadcn/ui
- Application of design-token.json visual language
- Theme simplification (4 → 2)
- Google Fonts CDN integration
- Sonner toast migration
- shadcn Sidebar adoption
- All page rebuilds

### Out of Scope
- Changing Zustand stores (except useThemeStore theme list and useNotificationStore removal)
- Changing API layer or typed callers
- Changing routing structure or adding/removing routes
- Changing i18n keys or locale file structure
- Adding new features or pages
- Backend changes
- New provider support
- Changing CodeMirror version or functionality
- Changing motion/framer-motion version or animation logic (beyond restyling)

## Constraints
- Single-file build (`vite-plugin-singlefile`) must keep working
- Big-bang approach: no hybrid SCSS + Tailwind period; all SCSS deleted in one pass
- Google Fonts CDN (not inlined) for VT323, Source Serif 4, JetBrains Mono
- Keep motion library for animations
- Bun as package manager
- `@/` path alias preserved
- All i18n keys unchanged

## Design Token Mapping

### Colors (Light)
| Token | Value | Tailwind Usage |
|-------|-------|---------------|
| `--bg` | `#fafaf5` | `bg-background` |
| `--bg-surface` | `#f3f1e8` | `bg-card`, `bg-muted` |
| `--ink` | `#1a1a1a` | `text-foreground` |
| `--ink-soft` | `#4a4a4a` | `text-muted-foreground` |
| `--ink-mute` | `#7a7a78` | secondary text |
| `--blueprint` | `#3553ff` | `text-primary`, `bg-primary` |
| `--rule-soft` | `rgba(26,26,26,0.16)` | `border-border` |
| `--code-bg` | `#efece0` | code block backgrounds |

### Colors (Dark)
Derive from the existing dark theme in `themes.scss`, mapped to design token variable names. Keep the warm-dark palette (`#151412` bg, `#1d1b18` surface, `#f6f4f1` text).

### Typography
| Role | Font | Size | Weight | Tailwind Class |
|------|------|------|--------|---------------|
| H1 | VT323 | 48px | 400 | `font-display text-5xl` |
| H2 | VT323 | 30.4px | 400 | `font-display text-3xl` |
| H3 | JetBrains Mono | 15.2px | 600 | `font-mono text-sm font-semibold uppercase tracking-wider` |
| Body | Source Serif 4 | ~17px | 400 | `font-body text-base` |
| Code | JetBrains Mono | ~15px | 400 | `font-mono text-sm` |

### Spacing
Base unit: 11px from design tokens. Map to Tailwind's default spacing scale (closest: `2.75` = 11px). Pragmatic approach: use Tailwind's standard 4px grid where it fits naturally.

### Border Radius
All 0 — `rounded-none` everywhere. Override shadcn defaults.

### Shadows
| Token | Value |
|-------|-------|
| `--shadow-hard` | `3px 3px 0 var(--ink)` |
| `--shadow-hard-lg` | `5px 5px 0 var(--ink)` |

## Validation Expectations
- **Build proof**: `bun run build` succeeds; output is a single HTML file with inlined Tailwind CSS; file size comparable to current build (±30%).
- **Visual proof**: all 10 routes render with the design token aesthetic (VT323 headings, sharp corners, blueprint blue accent) in both light and dark themes.
- **Functional proof**: login flow, provider management, auth file editing, config editing (CodeMirror), log viewing, quota display, OAuth management, system info — all work identically to pre-rebuild.
- **Responsive proof**: all pages usable on mobile viewports (≤768px).
- **i18n proof**: language switching works across all 4 locales.
- **Theme proof**: light/dark toggle works; only 2 themes available; `data-theme` attribute applied correctly.
- **Animation proof**: page transitions animate between routes; no regression in motion behavior.
- **Go build proof**: `make build` produces a working binary with the new frontend embedded.

## Key Decisions
1. **Full Tailwind v4 + shadcn (Option A)** over hybrid approach (Option B).
   - Why: user chose big-bang. Hybrid defeats the purpose and leaves two styling systems as maintenance burden.
2. **Google Fonts CDN** over inline font subsets.
   - Why: keeps single-file output small. Offline panel use is not a requirement.
3. **Keep motion library** over pure CSS replacement.
   - Why: page transition logic is complex; rewriting adds risk for no clear benefit.
4. **Sonner** over keeping hand-rolled notifications.
   - Why: less custom code, better UX, shadcn-recommended integration.
5. **shadcn Sidebar** over restyling current sidebar.
   - Why: gets collapsible, keyboard shortcuts, mobile sheet mode for free.
6. **Light + dark only** over preserving all 4 themes.
   - Why: reduces theme maintenance; `white` theme was barely distinct from `light`; `auto` can be reimplemented as a preference that switches between the two.

## Deferred Ideas
- `auto` theme preference (detect system dark mode and switch between light/dark)
- New pages or routes
- Dashboard redesign with charts (shadcn Chart component)
- Form validation library integration (React Hook Form + shadcn Field)
- Data table with sorting/filtering (shadcn Data Table)
- Command palette (shadcn Command)
- RTL support

## Ambiguity Report
- Goal clarity: high
- Scope clarity: high
- Constraints clarity: high
- Acceptance clarity: high
