# IDEA: LLMHub Phase 2 — Web UI Rebuild with shadcn + Design Tokens

## Origin
Phase 1 (rebrand + embedded panel) is complete. The management panel (`web/`) is a ~45k-line React SPA using hand-rolled UI components styled with SCSS Modules (~9.8k lines). The styling layer is the heaviest maintenance burden. The visual language (warm gray, rounded corners) needs a rebrand to match the AI Engineering from Scratch design system.

## Core Idea
Big-bang rewrite of the entire styling layer:
1. Replace all SCSS with **Tailwind CSS v4** utility classes
2. Replace all 16 hand-rolled UI primitives with **shadcn/ui** components
3. Apply **design-token.json** visual language (VT323 headings, Source Serif 4 body, blueprint blue accent, sharp 0-radius corners)
4. Simplify themes from 4 (auto/white/light/dark) to 2 (light + dark)

## What Stays
- React 19, Vite 8, TypeScript 6 — no framework change
- Zustand stores (7 stores) — untouched
- API layer (Axios singleton + typed callers) — untouched
- react-router-dom 7 (HashRouter) — untouched
- i18n (react-i18next, 4 locales) — untouched
- CodeMirror (YAML editor + merge view) — stays, restyled
- motion (framer-motion v12) — stays for page transitions
- vite-plugin-singlefile — stays (single-file build constraint)
- Bun as package manager — stays

## What Changes
- SCSS Modules → Tailwind v4 utility classes (delete all `.module.scss` and global SCSS)
- Hand-rolled Button, Card, Input, Select, Modal, Sheet, Table, Skeleton, Collapsible, ToggleSwitch, EmptyState, AutocompleteInput, LoadingSpinner, SelectionCheckbox → shadcn equivalents
- Hand-rolled notification system → Sonner (shadcn toast)
- Custom MainLayout sidebar → shadcn Sidebar component
- CSS variable theming (3 themes) → Tailwind v4 `@theme` with design tokens (2 themes: light + dark)
- Fonts: system fonts → Google Fonts CDN (VT323, Source Serif 4, JetBrains Mono)

## Key Constraints
- Single-file build must keep working (all assets inlined into one HTML)
- Google Fonts loaded via CDN `<link>` in `index.html`
- No new runtime dependencies beyond shadcn's peer deps (Radix UI primitives)
- All 10 routes/pages must be rebuilt
- 4 i18n locale files unchanged (keys stay the same)

## Design Token Summary (from design-token.json)
- **BG**: `#fafaf5` (light), dark TBD
- **Text**: `#1a1a1a` primary, `#4a4a4a` secondary, `#7a7a78` muted
- **Accent**: `#3553ff` (blueprint blue)
- **Border**: `rgba(26, 26, 26, 0.16)` (soft rule)
- **Border radius**: 0 (sharp corners everywhere)
- **Heading font**: VT323, monospace fallback
- **Body font**: Source Serif 4, serif fallback
- **Code font**: JetBrains Mono, monospace fallback
- **Shadows**: `3px 3px 0 var(--ink)` (hard shadow), `5px 5px 0 var(--ink)` (hard shadow lg)
- **Buttons**: filled (`#3553ff` bg, white text, 0 radius) + outline + ghost variants

## Rejected Alternatives
1. **Option B: Hybrid (shadcn + keep SCSS for complex pages)** — rejected because the user chose big-bang. A hybrid approach defeats the purpose and leaves two styling systems.
2. **Option C: CSS Modules + design tokens, no Tailwind** — rejected because user explicitly wants shadcn + Tailwind. The SCSS module layer is the heaviest burden and wouldn't shrink.
3. **Inline font subsets** — rejected in favor of Google Fonts CDN. Offline use is not a requirement, and inlining adds ~200-400KB.
4. **Replace motion with pure CSS** — rejected. Page transitions are complex; rewriting animation logic adds risk for no clear benefit.
5. **Keep hand-rolled notifications** — rejected in favor of Sonner. Less custom code, better UX, shadcn-recommended.
6. **Restyle current sidebar** — rejected in favor of shadcn Sidebar component. Gets collapsible, keyboard, mobile sheet for free.
