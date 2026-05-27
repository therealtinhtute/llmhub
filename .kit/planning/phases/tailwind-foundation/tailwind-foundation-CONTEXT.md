# Context: Tailwind Foundation

Phase: tailwind-foundation
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: build, visual inspection

## Goal
Install Tailwind CSS v4 and shadcn/ui into the existing Vite + React 19 project. Map all design tokens from `design-token.json` to Tailwind v4 `@theme` CSS custom properties. Create light and dark theme definitions. Add Google Fonts. Prove the single-file build still works.

## Scope Boundary
### Allowed Surfaces
- `web/package.json` — add tailwind, shadcn deps
- `web/src/index.css` — new Tailwind entry with `@theme` block
- `web/index.html` — add Google Fonts CDN links
- `web/components.json` — shadcn config
- `web/vite.config.ts` — add Tailwind plugin if needed (v4 may auto-detect)
- `web/tsconfig.json` — path alias verification
- `web/src/styles/` — may add a `globals.css` or similar; do NOT delete existing SCSS yet

### Forbidden Surfaces
- Any page or component TSX files (no component changes this phase)
- Zustand stores
- API layer
- Router
- i18n files
- Existing SCSS files (do not delete yet — that's Phase 5)

## Spec Hooks
- Req 1: Install and configure Tailwind CSS v4 in Vite
- Req 2: Initialize shadcn/ui with `components.json`
- Req 3: Map design tokens to `@theme` CSS custom properties
- Req 4: Add Google Fonts CDN links
- Req 5: Create light and dark themes
- Constraint: single-file build must keep working
- Constraint: `@/` path alias preserved

## Locked Decisions
- Tailwind CSS v4 (CSS-first config via `@theme`, no `tailwind.config.js`)
- shadcn/ui with Vite installation path
- Google Fonts CDN (not inline subsets): VT323, Source Serif 4, JetBrains Mono
- Two themes only: light + dark
- Design token source: `design-token.json` at repo root
- Dark theme colors derived from existing `themes.scss` `[data-theme='dark']` block

## Assumptions
- Tailwind v4 works with Vite 8 and `vite-plugin-singlefile` without special config
- shadcn CLI (`bunx shadcn@latest init`) supports Vite + React 19 + Tailwind v4
- Google Fonts CDN links in `<head>` survive `vite-plugin-singlefile` inlining (they should — external links aren't inlined)

## Canonical Refs
- `design-token.json` — all color, typography, spacing, shadow, radius values
- `web/src/styles/themes.scss` — existing dark theme color values to preserve
- shadcn Vite installation: https://ui.shadcn.com/docs/installation/vite
- shadcn theming: https://ui.shadcn.com/docs/theming
- Tailwind v4 docs: https://ui.shadcn.com/docs/tailwind-v4

## Rejected Options
- Tailwind v3 with `tailwind.config.js` — v4 is current, CSS-first config is cleaner, shadcn supports it
- CSS-only theming without Tailwind — doesn't get us shadcn components
- Inline font subsets — adds 200-400KB to single-file output for no clear benefit

## Deferred Ideas
- Custom Tailwind plugin for design token utilities
- Font subsetting optimization
- Tailwind `@source` annotation for tree-shaking (explore if build size is large)

## Escalate If
- `vite-plugin-singlefile` cannot inline Tailwind CSS output → investigate alternative inlining strategies
- shadcn CLI doesn't support Tailwind v4 + Vite → fall back to manual component installation
- Google Fonts CDN links break single-file serving → consider inline subsets as fallback
