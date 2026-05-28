# Context: Token Foundation

Phase: token-foundation
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: build (bun run build), visual inspection

## Goal
Rewrite `web/src/index.css` to define the complete new design token system — ~90 CSS custom properties covering colors, accent, status, typography, spacing, dimensions, layout, radius, shadows, graph theming, and safe area insets. All colors in oklch. Dual-map shadcn standard variables to new semantic names.

## Scope Boundary
### Allowed Surfaces
- `web/src/index.css` — the only file modified in this phase

### Forbidden Surfaces
- Component files (`.tsx`, `.ts`) — no changes
- `components.json` — no changes
- `package.json` — no changes
- Pages, features, stores — no changes

## Spec Hooks
- R1: Core palette (items 1-3)
- R2: Accent system (items 4-7)
- R3: Status colors (items 8-10)
- R4: Typography scale (items 11-14)
- R5: Spacing scale (items 15-16)
- R6: Component dimensions (item 17)
- R7: Layout tokens (items 18-19, layout vars only — sidebar component update deferred to Phase 3)
- R8: Radius & shadow (items 20-21)
- R9: Graph tokens (items 22-23)
- R10: Safe area insets (item 24)
- R11: @theme block rewrite (items 25-29)
- Constraint: oklch format for all color values
- Constraint: Tailwind v4 inline theme configuration

## Locked Decisions
- All hex values converted to oklch using standard conversion
- Dual variable mapping: `--background` AND `--bg-primary` both exist, pointing to same oklch value
- `--primary` maps to `--accent` value (#117dff oklch equivalent)
- Radius uses direct values (3px/4px/6px/16px/9999px), not calc-from-base
- Shadow simplified to 3 levels (sm/md/lg)
- `.dark` block kept in this phase (removed in Phase 2) — this phase only rewrites `:root` and `@theme`
- Existing animations/keyframes preserved untouched
- Existing `.custom-scrollbar`, `.status-bar-tooltip` styles preserved

## Assumptions
- oklch conversion of hex values produces visually equivalent colors
- Tailwind v4 `@theme` block supports `--color-bg-primary: var(--bg-primary)` pattern
- No Tailwind utility name collisions with `--text-xs`, `--text-sm` etc. (CSS custom properties, not Tailwind theme keys)
- `--space-*` tokens don't collide with Tailwind's `--spacing` base

## Canonical Refs
- `design-token-new.json` — source of truth for all token values
- `web/src/index.css` — target file (current state: ~282 lines)
- `.kit/planning/SPEC.md` — requirements R1-R11

## Rejected Options
- Keeping hex values directly — rejected per user decision; oklch for perceptual uniformity
- Removing `.dark` block in this phase — rejected; sequencing risk too high, defer to Phase 2
- Using Tailwind `@theme` for ALL tokens — rejected; many tokens (graph-*, safe-area, layout) are CSS-only

## Deferred Ideas
- Component-level tokens (button bg/padding/radius from design-token-new.json components section)
- Font face declarations (@font-face rules)
- CSS container queries using --container-* tokens

## Escalate If
- Tailwind v4 `@theme` block rejects new color token naming pattern
- oklch conversion produces visibly different colors from hex source
- `bun run build` fails after token changes
