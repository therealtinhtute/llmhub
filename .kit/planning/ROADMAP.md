# ROADMAP: Design System Migration — Supermemory Console Aesthetic

## Planning Basis
- source spec: `.kit/planning/SPEC.md`
- planning mode: `full`
- entry phase: `token-foundation`
- execution mode: sequential phases, parallel waves within each

## Phase 1: token-foundation
**Goal:** Replace all CSS design tokens, font stacks, shadow scale, radius, and dependencies so the visual foundation matches supermemory console. No component code changes yet.

**Deliverables:**
- `index.css` rewritten: oklch color tokens, Space Grotesk/Mono/Inter font stacks, 0.75rem radius, subtle shadow scale, `@layer base` body rule, `tw-animate-css` import
- `index.html` Google Fonts links swapped (VT323/Source Serif 4/JetBrains Mono → Space Grotesk/Space Mono/Inter)
- `components.json` base color changed to zinc
- `tw-animate-css` package installed
- Custom `@keyframes` reviewed and kept/removed based on usage

**Dependencies:**
- None (entry phase)

**Risks / Watch-fors:**
- `shadow-[var(--shadow-hard)]` in Card.tsx will break visually — consumers fixed in Phase 3
- `font-display` and `font-body` references in TSX won't resolve — consumers fixed in Phase 3
- Old `--text-primary`, `--glass-*` etc. aliases still referenced — kept temporarily, cleaned in Phase 4

## Phase 2: legacy-removal
**Goal:** Delete all 13 Legacy* wrapper components and migrate their 71 consumer files to shadcn/ui primitives directly.

**Deliverables:**
- 13 Legacy* files deleted from `src/components/ui/`
- 71 consumer files migrated to shadcn primitives
- Small `EmptyState` utility component created (replaces LegacyEmptyState, 8 consumers)
- Combobox pattern for autocomplete (replaces LegacyAutocompleteInput, 2 consumers)

**Dependencies:**
- Phase 1 complete (tokens in place so migrated components render correctly)

**Risks / Watch-fors:**
- LegacyInput adds label/hint/error props — ensure consistent pattern when migrating to raw Input + Label
- LegacyToggleSwitch has 9 consumers — largest blast radius
- LegacyAutocompleteInput has custom filtering — preserve in combobox migration

## Phase 3: style-alignment
**Goal:** Update all component TSX to use new design system classes. Remove retro-specific styling (rounded-none, shadow-hard, font-display/font-body). Migrate global utility class consumers to inline Tailwind.

**Deliverables:**
- All `rounded-none` in shadcn UI components updated to use default radius
- `shadow-[var(--shadow-hard)]` in Card.tsx → `shadow-sm`
- Global utility class consumers (`.status-badge`, `.error-box`, `.item-list`, `.form-group`, `.pill`, `.hint`) migrated to Tailwind utilities
- SCSS compatibility alias consumers (`--glass-*`, `--text-*`, `--bg-*`, `--*-badge-*`, `--amber-*`, etc.) migrated to semantic Tailwind classes
- Custom scrollbar utility class added

**Dependencies:**
- Phase 2 complete (legacy components gone, no conflicting wrapper styles)

**Risks / Watch-fors:**
- VisualConfigEditor.tsx has heavy use of global utility classes via parent selector overrides — complex migration
- ProviderStatusBar.tsx, AuthFilesPage.tsx, and quotaStyles.ts reference SCSS aliases extensively
- Some `rounded-none` in shadcn primitives (Button, Dialog, etc.) are intentional for the old design — remove all

## Phase 4: cleanup-verify
**Goal:** Delete all dead CSS (compatibility aliases, global utility classes, App.css). Run full verification suite.

**Deliverables:**
- SCSS compatibility aliases deleted from index.css (lines 230-273)
- Global utility classes deleted from index.css (lines 275-422)
- `App.css` deleted (Vite scaffold leftover, unused)
- All verification checks pass (build, type-check, grep for legacy references, functional test)

**Dependencies:**
- Phase 3 complete (all consumers migrated before deletion)

**Risks / Watch-fors:**
- Must grep for every global class name before deletion to confirm zero consumers remain
- `status-bar-tooltip` CSS may still be needed — verify before removing
