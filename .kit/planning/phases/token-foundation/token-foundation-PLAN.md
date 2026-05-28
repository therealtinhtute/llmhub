# Plan: Token Foundation

Phase: token-foundation
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-28

## Goal
Rewrite `web/src/index.css` with the complete new design token system from `design-token-new.json`. All colors in oklch. Dual-map shadcn standard variables to new semantic names. Rewrite `@theme` block.

## Inputs
- `design-token-new.json` — all token values
- `web/src/index.css` — current file to rewrite
- `.kit/planning/SPEC.md` — requirements R1-R11

## Wave 1
### T1 — Rewrite `:root` core color tokens + accent system
- type: implementation
- inputs:
  - `design-token-new.json` colors, cssCustomProperties sections
  - Current `:root` block in `web/src/index.css`
- touches:
  - `web/src/index.css` `:root` block
- avoid:
  - `.dark` block (Phase 2)
  - Component files
  - `@theme` block (T3)
- steps:
  1. Convert all hex colors from design-token-new.json to oklch values
  2. Replace existing `:root` color tokens with new warm palette oklch values
  3. Add dual mapping: shadcn standard names (`--background`, `--foreground`, `--card`, `--popover`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--border`, `--input`, `--ring` + foreground variants) point to new warm values
  4. Add new semantic background tokens: `--bg-primary`, `--bg-secondary`, `--bg-muted`, `--bg-elevated`, `--bg-overlay`
  5. Add new semantic text tokens: `--text-strong`, `--text-primary`, `--text-secondary`, `--text-muted`, `--text-inverse`
  6. Add new border tokens: `--border-muted`, `--border-accent`
  7. Add accent system: `--accent`, `--accent-foreground`, `--accent-start`, `--accent-mid`, `--accent-end`, `--accent-gradient`, `--accent-hover`, `--accent-muted`
  8. Map `--primary` → `--accent` value, `--primary-hover` → `--accent-hover`, add `--primary-active`
  9. Add status colors in oklch: `--success`, `--success-muted`, `--error`, `--error-muted`, `--warning`, `--warning-muted`, `--info`, `--info-muted`, `--dreaming`, `--dreaming-muted`, `--danger`
  10. Map `--destructive` → `--danger` oklch value
- expected outputs:
  - `:root` block with ~50 color/accent/status custom properties in oklch
- verification:
  - `grep -c "^  --" web/src/index.css` shows increased property count
  - `grep "oklch" web/src/index.css | head -20` confirms oklch format
- stop if:
  - oklch conversion produces unexpected values (check a few key conversions manually)
- escalate to:
  - user clarification if hex → oklch mapping is ambiguous

### T2 — Add typography, spacing, dimension, layout, radius, shadow, graph, and safe-area tokens
- type: implementation
- inputs:
  - `design-token-new.json` cssCustomProperties section
  - T1 output (`:root` with color tokens)
- touches:
  - `web/src/index.css` `:root` block (appending after color tokens)
- avoid:
  - Modifying color tokens from T1
  - `.dark` block
  - `@theme` block
  - Component files
- steps:
  1. Add typography scale tokens: `--text-xs` through `--text-3xl`, `--font-normal/medium/semibold/bold`, `--leading-tight/normal/relaxed`
  2. Add font aliases: `--font-brand`, `--font-display` → Space Grotesk stack; `--f-serif` → Georgia stack
  3. Add spacing scale: `--space-1` through `--space-16`
  4. Add component dimension tokens: `--height-xs/sm/md/lg`, `--icon-xs/sm/md/lg`
  5. Add layout tokens: `--sidebar-width`, `--sidebar-collapsed-width`, `--container-narrow/medium/wide/xwide`, `--page-header-py/px`, `--page-title-size/weight/mt`, `--input-text-size`, `--body-text-size`, `--caption-text-size`, `--label-text-size`
  6. Replace radius tokens: `--radius-sm: 3px`, `--radius-md: 4px`, `--radius-lg: 6px`, `--radius-xl: 16px`, `--radius-full: 9999px`, `--border-width: 1px`. Update `--radius` base to `0.375rem` (6px, so `--radius-lg` = `var(--radius)`)
  7. Replace shadow scale: `--shadow-sm`, `--shadow-md`, `--shadow-lg` (3 levels). Remove `--shadow-2xs`, `--shadow-xs`, `--shadow-xl`, `--shadow-2xl`
  8. Add graph tokens in oklch: `--graph-bg`, `--graph-doc-fill/stroke/inner`, `--graph-mem-fill/fill-hover/stroke`, `--graph-accent`, `--graph-text-primary/secondary/muted`, `--graph-edge-derives/updates/extends`, `--graph-mem-border-forgotten/expiring/recent`, `--graph-glow`, `--graph-icon`, `--graph-popover-bg/border/text-primary/text-secondary/text-muted`, `--graph-control-bg/border`
  9. Add safe area insets: `--safe-top/bottom/left/right`
  10. Keep existing sidebar tokens (`--sidebar`, `--sidebar-foreground`, etc.) updated with new warm palette values
  11. Keep existing chart tokens (`--chart-1` through `--chart-5`) — update values if needed to match warm palette
- expected outputs:
  - `:root` block with ~90+ total custom properties
- verification:
  - `grep -c "^  --" web/src/index.css` shows ~90+ properties
  - `grep "graph" web/src/index.css | wc -l` shows ~20+ graph tokens
- stop if:
  - Token naming collides with existing Tailwind utilities
- escalate to:
  - plan phase if naming collision found

## Wave 2
### T3 — Rewrite `@theme` block to expose new tokens to Tailwind
- type: implementation
- inputs:
  - T1 + T2 output (complete `:root` block)
  - Current `@theme` block in `web/src/index.css`
- touches:
  - `web/src/index.css` `@theme` block(s)
- avoid:
  - `:root` block (already written)
  - `.dark` block
  - Component files
- steps:
  1. Keep existing shadcn color mappings: `--color-background`, `--color-foreground`, `--color-card`, `--color-primary`, etc. → `var(--background)`, etc.
  2. Add new semantic color mappings: `--color-bg-primary: var(--bg-primary)`, `--color-bg-secondary`, `--color-bg-muted`, `--color-bg-elevated`
  3. Add text color mappings: `--color-text-strong`, `--color-text-primary`, `--color-text-secondary`, `--color-text-muted`
  4. Add accent color mappings: `--color-accent: var(--accent)`, `--color-accent-hover`, `--color-accent-muted`
  5. Add status color mappings: `--color-success`, `--color-error`, `--color-warning`, `--color-info`, `--color-dreaming`, `--color-danger`
  6. Update font stacks: keep `--font-sans`, `--font-mono`; add `--font-serif` if not present
  7. Update radius: `--radius-sm: 3px`, `--radius-md: 4px`, `--radius-lg: 6px`, `--radius-xl: 16px`
  8. Keep sidebar and chart color mappings
  9. Remove any stale theme entries that no longer exist in `:root`
- expected outputs:
  - `@theme` block with all new color tokens exposed as Tailwind utilities
  - Tailwind classes like `bg-bg-primary`, `text-text-secondary`, `text-success` become available
- verification:
  - `cd web && bun run build` succeeds (proves Tailwind processes the theme correctly)
- stop if:
  - Build fails due to `@theme` block syntax
  - Tailwind utility naming collision
- escalate to:
  - user clarification if Tailwind v4 rejects token patterns

## Wave 3
### T4 — Build verification
- type: test
- inputs:
  - T1-T3 complete `web/src/index.css`
- touches:
  - None (read-only)
- avoid:
  - File modifications
- steps:
  1. Run `cd web && bun run build` — must succeed
  2. Verify output is a single HTML file in `dist/`
  3. Count custom properties: `grep -c "^  --" web/src/index.css` — expect ~90+
  4. Verify oklch format: `grep -c "oklch" web/src/index.css` — expect many
  5. Verify no hex colors in `:root` (except shadow values which use hex alpha shorthand): `grep "^  --.*#[0-9a-fA-F]" web/src/index.css` — only shadow values allowed
- expected outputs:
  - Build succeeds
  - Token count verified
- verification:
  - All checks above pass
- stop if:
  - Build fails
- escalate to:
  - plan phase token-foundation if build errors need token restructuring

## Risks / Watch-fors
- Tailwind v4's `@theme` has specific rules about what can go inside — test build early
- `--text-xs` etc. may collide with Tailwind's built-in font-size utilities — if so, prefix with `--ds-text-xs` instead
- Shadow hex alpha shorthand (`#0000000d`) is fine in shadow values — don't over-convert to oklch
- Existing animations, scrollbar styles, tooltip pseudo-elements must be preserved exactly
