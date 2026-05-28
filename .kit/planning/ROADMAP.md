# ROADMAP: Design Token v2 — Warm Palette + Extended Semantic System

## Planning Basis
- source spec: `.kit/planning/SPEC.md`
- planning mode: `full`
- entry phase: `token-foundation`
- execution mode: sequential, 4 phases

---

## Phase 1: token-foundation
**Goal:** Rewrite `index.css` with the complete new token system — warm cream/beige oklch palette, extended semantic variables (~90 properties), rewritten `@theme` block. This is the single-file foundation that everything else builds on.

**Deliverables:**
- New `:root` block with all color, accent, status, typography, spacing, dimension, layout, radius, shadow, graph, and safe-area tokens in oklch
- Dual mapping: shadcn standard names (`--background`, `--primary`, etc.) AND new semantic names (`--bg-primary`, `--accent`, etc.)
- Rewritten `@theme` block exposing all new tokens to Tailwind
- Simplified radius (direct values) and shadow (3-level) scales
- Updated `--font-brand`, `--font-display`, `--f-serif` aliases

**Dependencies:**
- `design-token-new.json` as source of truth
- Hex-to-oklch conversion for all values

**Risks / Watch-fors:**
- Tailwind v4 `@theme` block may not support all custom property patterns — validate with `bun run build`
- oklch conversion accuracy — verify visual match against hex originals
- Naming collisions between new `--text-*` tokens and Tailwind's built-in `--text-*` utilities

---

## Phase 2: dark-mode-removal
**Goal:** Strip all dark mode artifacts: `.dark` CSS block, `@custom-variant dark`, `dark:` utility classes from all components, theme store simplification.

**Deliverables:**
- No `.dark {}` block in `index.css`
- No `@custom-variant dark` directive
- No `dark:` prefixed classes in any component/page/feature file
- Simplified `useThemeStore.ts` (always light)
- Theme toggle removed from UI

**Dependencies:**
- Phase 1 complete (new tokens in place so light theme renders correctly)

**Risks / Watch-fors:**
- 31 files reference "dark" in some form — distinguish `dark:` utility classes from string content
- Theme store consumers may break if store API changes

---

## Phase 3: component-adoption
**Goal:** Update components with hardcoded colors to use new semantic tokens. Update layout components to use layout/dimension tokens.

**Deliverables:**
- `DiffModal.tsx` using `--success`, `--error`, `--accent` tokens
- `quotaStyles.ts` using semantic tokens
- `ModelMappingDiagram.tsx` using graph tokens
- `sidebar.tsx` using `--sidebar-width`, `--sidebar-collapsed-width`
- Page headers using `--page-header-*`, `--page-title-*` tokens
- Form components using `--input-text-size`, `--height-md`, `--label-text-size`

**Dependencies:**
- Phase 1 complete (tokens defined)
- Phase 2 complete (no dark: classes interfering)

**Risks / Watch-fors:**
- DiffModal has complex color-mix expressions — careful replacement needed
- quotaStyles uses elaborate gradient strings — replacement must preserve visual intent

---

## Phase 4: verification
**Goal:** Full build validation, visual regression check, cleanup proof, Go binary build.

**Deliverables:**
- `bun run build` passes
- `make build` passes
- Grep proofs: no `.dark` blocks, no `dark:` classes, no stale token names
- Token count proof: ~90+ custom properties in index.css
- Visual confirmation of warm cream palette across all routes

**Dependencies:**
- Phases 1-3 complete

**Risks / Watch-fors:**
- Single-file build plugin may interact poorly with new CSS variable count
- Go embed step may cache stale assets
