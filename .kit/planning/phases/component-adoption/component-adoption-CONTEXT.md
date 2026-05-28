# Context: Component Token Adoption

Phase: component-adoption
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: build, visual inspection

## Goal
Update components with hardcoded colors to use new semantic tokens. Update layout components to consume layout/dimension tokens. This phase makes components token-aware rather than hardcoded.

## Scope Boundary
### Allowed Surfaces
- `web/src/components/config/DiffModal.tsx` — replace hardcoded hex with semantic tokens
- `web/src/components/quota/quotaStyles.ts` — replace hardcoded hex with semantic tokens
- `web/src/components/modelAlias/ModelMappingDiagram.tsx` — replace hardcoded color array with graph tokens
- `web/src/components/ui/sidebar.tsx` — use `--sidebar-width`, `--sidebar-collapsed-width`
- `web/src/components/layout/MainLayout.tsx` — use layout tokens if applicable
- Page components — use `--page-header-*`, `--page-title-*` tokens
- Form-related UI components — use `--input-text-size`, `--height-md`, `--label-text-size`

### Forbidden Surfaces
- `web/src/index.css` — already complete from Phase 1
- API layer, stores, routing
- Adding new components or features

## Spec Hooks
- R13: Component token adoption (items 35-40)
- Key Decision 5: Dual variable mapping — components can use either shadcn standard names or new semantic names

## Locked Decisions
- DiffModal: git-diff colors (`#3fb950` green, `#f85149` red) map to `var(--success)` and `var(--error)`; blue accent `#388bfd` maps to `var(--accent)`
- quotaStyles: gradient/badge colors map to semantic tokens where possible; complex gradient strings may keep inline colors if no semantic equivalent exists
- ModelMappingDiagram: hardcoded color array replaced with CSS custom property references to `--graph-*` tokens
- Sidebar: width values replaced with `var(--sidebar-width)` and `var(--sidebar-collapsed-width)`
- Components use `var()` references to CSS custom properties, not Tailwind utility classes for these specific tokens

## Assumptions
- Phase 1 tokens are all defined and build-verified
- Phase 2 dark mode removal is complete (no `dark:` interference)
- DiffModal's `color-mix()` expressions can be replaced with simpler `var()` references + oklch alpha variants
- Graph tokens defined in Phase 1 match the color needs of ModelMappingDiagram

## Canonical Refs
- `web/src/components/config/DiffModal.tsx` — 15 hardcoded hex references
- `web/src/components/quota/quotaStyles.ts` — 5 hardcoded hex references
- `web/src/components/modelAlias/ModelMappingDiagram.tsx` — 8-color array
- `web/src/components/ui/sidebar.tsx` — sidebar width values

## Rejected Options
- Converting ALL Tailwind utility colors to var() references — rejected; only converting hardcoded hex values. Tailwind utilities like `bg-primary` already resolve through the token system.
- Creating wrapper utility classes for graph colors — rejected; inline var() references are simpler for 3 files

## Deferred Ideas
- Component-level token adoption beyond hardcoded colors (e.g., using --height-md for all button heights across all components)
- Graph visualization component consuming graph tokens

## Escalate If
- DiffModal color-mix() replacement produces visual regression
- quotaStyles gradient cannot be expressed with semantic tokens
- ModelMappingDiagram uses colors for semantic meaning beyond decoration
