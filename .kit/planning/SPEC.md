# SPEC: Design Token v2 — Warm Palette + Extended Semantic System

Status: draft
Input Type: change-request
Lane: normal
Risk Flags: existing-behavior
Affected Surfaces: browser
Downstream: plan full
Updated At: 2026-05-28

## Source Mode
lock-from-files

## Source Inputs
- `design-token-new.json` — new design token definition (warm cream/beige palette, extended semantic variables, Supermemory Console aesthetic)
- `web/src/index.css` — current design system (cool blue-purple oklch, standard shadcn variables, light + dark modes)

## Scenario
The web UI completed Phase 4 migration (shadcn, Tailwind v4, Space Grotesk/Inter typography). The current palette is cool blue-purple (oklch hue ~258). The new design tokens shift to a warm cream/beige palette (#faf9f4 base) with extended semantic variable layers covering backgrounds, text hierarchy, status colors, graph theming, spacing scale, typography scale, and layout dimensions. Dark mode is being dropped in favor of a single light theme.

## Goal
1. Replace ALL color tokens from cool blue-purple oklch to warm cream/beige oklch equivalents
2. Drop dark mode entirely (remove `.dark` block)
3. Add the full extended semantic variable system (~90 new CSS custom properties)
4. Rewrite `@theme` block to expose new tokens to Tailwind
5. Update all 34 UI components and page components that consume design tokens
6. Convert all hex values from new tokens to oklch for consistency

## Users / Actors
- Primary maintainer: implementing the migration
- Operators: using the management panel to manage LLMHub instances

## Requirements

### R1: Color Token Replacement — Core Palette (oklch conversion)

All hex values from `design-token-new.json` must be converted to oklch.

| New Token | Hex Source | Maps to shadcn var | Purpose |
|-----------|-----------|-------------------|---------|
| `--bg-primary` | #faf9f4 | `--background` | Page background (warm cream) |
| `--bg-secondary` | #f3f1ec | — | Secondary surfaces |
| `--bg-muted` | #e8e5e0 | `--muted` | Muted/badge surfaces |
| `--bg-elevated` | #ffffff | `--card`, `--popover` | Cards, popovers |
| `--bg-overlay` | #faf9f4b3 | — | Overlay backdrop |
| `--text-strong` | #000000 | — | Strongest emphasis |
| `--text-primary` | #0a0a0a | `--foreground`, `--card-foreground`, `--popover-foreground` | Primary text |
| `--text-secondary` | #525252 | `--muted-foreground` | Secondary text |
| `--text-muted` | #a3a3a3 | — | Placeholder/hint text |
| `--text-inverse` | #ffffff | `--primary-foreground`, `--destructive-foreground` | Text on dark bg |
| `--border` | #e3e0db | `--border`, `--input` | Default border |
| `--border-muted` | #f0ede8 | — | Subtle border |
| `--border-accent` | var(--accent) | — | Accent-colored border |

1. Convert all hex values to oklch and define in `:root`
2. Map to both new semantic names AND shadcn standard names (dual mapping)
3. Remove the `.dark` block entirely

### R2: Accent System

| Token | Hex | Purpose |
|-------|-----|---------|
| `--accent` | #117dff | Primary accent (replaces `--primary`) |
| `--accent-foreground` | #ffffff | Text on accent |
| `--accent-start` | #1148f7 | Gradient start |
| `--accent-mid` | #117dff | Gradient midpoint |
| `--accent-end` | #cdf4ff | Gradient end |
| `--accent-gradient` | linear-gradient(135deg, ...) | Accent gradient |
| `--accent-hover` | #1148f7 | Accent hover state |
| `--accent-muted` | #cdf4ff | Muted accent surface |

4. Define accent system in `:root`
5. Map `--primary` → `--accent` value, `--primary-foreground` → `--accent-foreground`
6. Map `--primary-hover` → `--accent-hover`, `--primary-active` → darker variant of `--accent-hover`
7. Define `--accent-gradient` using var references

### R3: Status Colors

| Token | Hex | Muted variant |
|-------|-----|---------------|
| `--success` | #4ade80 | #4ade802e |
| `--error` | #f87171 | #f871712e |
| `--warning` | #fbbf24 | #fbbf242e |
| `--info` | #60a5fa | #60a5fa2e |
| `--dreaming` | #a855f7 | #a855f72e |
| `--danger` | #ef4444 | — |

8. Define all status colors + muted variants in oklch
9. Map `--destructive` → `--danger` value
10. Expose status colors to Tailwind via `@theme` block

### R4: Typography Scale

| Token | Value |
|-------|-------|
| `--text-xs` | 12px |
| `--text-sm` | 14px |
| `--text-base` | 16px |
| `--text-lg` | 18px |
| `--text-xl` | 20px |
| `--text-2xl` | 24px |
| `--text-3xl` | 30px |
| `--font-normal` | 400 |
| `--font-medium` | 500 |
| `--font-semibold` | 600 |
| `--font-bold` | 700 |
| `--leading-tight` | 1.25 |
| `--leading-normal` | 1.5 |
| `--leading-relaxed` | 1.75 |

11. Define typography scale variables in `:root`
12. Font stacks unchanged (already Space Grotesk + Inter + Space Mono)
13. Add `--font-brand` and `--font-display` aliases pointing to Space Grotesk stack
14. Add `--f-serif` for Georgia/Times serif fallback

### R5: Spacing Scale

| Token | Value |
|-------|-------|
| `--space-1` | 4px |
| `--space-2` | 8px |
| `--space-3` | 12px |
| `--space-4` | 16px |
| `--space-5` | 20px |
| `--space-6` | 24px |
| `--space-8` | 32px |
| `--space-10` | 40px |
| `--space-12` | 48px |
| `--space-16` | 64px |

15. Define spacing scale in `:root`
16. Keep Tailwind's default spacing (--spacing: 0.25rem) for utility classes — these are for explicit use in component styles

### R6: Component Dimensions

| Token | Value | Purpose |
|-------|-------|---------|
| `--height-xs` | 28px | Compact buttons |
| `--height-sm` | 32px | Small inputs |
| `--height-md` | 36px | Default inputs/buttons |
| `--height-lg` | 44px | Large targets |
| `--icon-xs` | 14px | Tiny icons |
| `--icon-sm` | 16px | Small icons |
| `--icon-md` | 20px | Default icons |
| `--icon-lg` | 24px | Large icons |

17. Define height and icon size tokens in `:root`

### R7: Layout Tokens

| Token | Value |
|-------|-------|
| `--sidebar-width` | 260px |
| `--sidebar-collapsed-width` | 60px |
| `--container-narrow` | 26rem |
| `--container-medium` | 40rem |
| `--container-wide` | 53rem |
| `--container-xwide` | 67rem |
| `--page-header-py` | var(--space-4) |
| `--page-header-px` | var(--space-6) |
| `--page-title-size` | var(--text-xl) |
| `--page-title-weight` | var(--font-semibold) |
| `--page-title-mt` | var(--space-6) |
| `--input-text-size` | var(--text-sm) |
| `--body-text-size` | var(--text-sm) |
| `--caption-text-size` | var(--text-xs) |
| `--label-text-size` | var(--text-sm) |

18. Define layout tokens in `:root`
19. Update sidebar component to use `--sidebar-width` and `--sidebar-collapsed-width`

### R8: Radius & Shadow

| Token | Value |
|-------|-------|
| `--radius-sm` | 3px |
| `--radius-md` | 4px |
| `--radius-lg` | 6px |
| `--radius-xl` | 16px |
| `--radius-full` | 9999px |
| `--border-width` | 1px |
| `--shadow-sm` | 0 1px 2px 0 #0000000d |
| `--shadow-md` | 0 4px 6px -1px #0000001a, 0 2px 4px -2px #0000001a |
| `--shadow-lg` | 0 10px 15px -3px #0000001a, 0 4px 6px -4px #0000001a |

20. Replace current radius values: `--radius: 0.375rem` (6px base, derive sm/md/lg/xl from token values directly instead of calc)
21. Simplify shadow scale to 3 levels (sm/md/lg) — remove 2xs/xs/xl/2xl

### R9: Graph Tokens

| Token | Value | Purpose |
|-------|-------|---------|
| `--graph-bg` | var(--bg-secondary) | Canvas background |
| `--graph-doc-fill` | #e8e5e0 | Document node fill |
| `--graph-doc-stroke` | #d4d0cb | Document node border |
| `--graph-doc-inner` | #f3f1ec | Document inner area |
| `--graph-mem-fill` | #dbeafe | Memory node fill |
| `--graph-mem-fill-hover` | #bfdbfe | Memory node hover |
| `--graph-mem-stroke` | #3b82f6 | Memory node border |
| `--graph-accent` | var(--accent) | Graph accent |
| `--graph-text-*` | var(--text-*) | Text hierarchy |
| `--graph-edge-derives` | #b08a1a | Edge: derives |
| `--graph-edge-updates` | #7c3aed | Edge: updates |
| `--graph-edge-extends` | #7dd3fc | Edge: extends |
| `--graph-mem-border-forgotten` | #dc2626 | Forgotten state |
| `--graph-mem-border-expiring` | #d97706 | Expiring state |
| `--graph-mem-border-recent` | #059669 | Recent state |
| `--graph-glow` | #3b82f6 | Glow effect |
| `--graph-icon` | var(--text-muted) | Icons |
| `--graph-popover-*` | var(--*) | Popover theming |
| `--graph-control-*` | var(--*) | Control panel |

22. Define all graph tokens in `:root` — convert hex to oklch
23. Graph tokens that reference other vars use `var()` references

### R10: Safe Area Insets

24. Define `--safe-top/bottom/left/right` using `env(safe-area-inset-*, 0px)`

### R11: @theme Block Rewrite

25. Expose new color tokens to Tailwind: `--color-bg-primary`, `--color-bg-secondary`, etc.
26. Expose status colors: `--color-success`, `--color-error`, etc.
27. Expose accent system: `--color-accent`, `--color-accent-hover`, etc.
28. Keep existing shadcn-standard color mappings for compatibility
29. Update radius scale to use direct values instead of calc

### R12: Dark Mode Removal

30. Delete entire `.dark { }` block from index.css
31. Remove `@custom-variant dark (&:where(.dark, .dark *));` directive
32. Remove dark mode toggle from UI (if present)
33. Remove `dark:` prefixed classes from all components
34. Simplify sidebar colors — no dark variant needed

### R13: Component Token Adoption

35. Update `DiffModal.tsx` — replace hardcoded hex (`#3fb950`, `#f85149`, `#388bfd`) with `--success`, `--error`, `--accent` tokens
36. Update `quotaStyles.ts` — replace hardcoded hex with semantic tokens
37. Update `ModelMappingDiagram.tsx` — replace hardcoded color array with graph tokens
38. Update sidebar.tsx — use `--sidebar-width`, `--sidebar-collapsed-width` tokens
39. Update page headers — use `--page-header-*` and `--page-title-*` tokens
40. Update form components — use `--input-text-size`, `--height-md`, `--label-text-size`

### R14: Verification

41. `bun run build` succeeds
42. `bun run type-check` passes (if configured)
43. All routes render correctly in light theme with warm cream palette
44. No `.dark` class references remain in CSS or components
45. No hardcoded hex colors remain in component files (except DiffModal git-diff-specific colors if intentional)
46. All API integrations work unchanged
47. Mobile responsive behavior preserved
48. `make build` (full Go binary) succeeds

## Boundaries

### In Scope
- Full color palette swap (cool blue-purple → warm cream/beige) in oklch
- Dark mode removal
- Addition of ~90 new CSS custom properties (semantic backgrounds, text hierarchy, status, graph, spacing, typography, layout, dimensions)
- `@theme` block rewrite to expose new tokens to Tailwind utilities
- Component updates to consume new semantic tokens
- Radius and shadow simplification

### Out of Scope
- Adding new pages or features
- Backend/Go changes (except `make build` verification)
- Changing routing, stores, or API layer
- Adding new shadcn components
- i18n changes
- Motion/animation logic changes
- Graph visualization component (just defining the tokens — no graph UI exists yet)

## Constraints
- Single-file build (`vite-plugin-singlefile`) must keep working
- Google Fonts CDN for Space Grotesk, Inter (already configured)
- Bun as package manager
- `@/` path alias preserved
- All i18n keys unchanged
- oklch format for all color values
- Tailwind v4 inline theme configuration

## Validation Expectations
- **Build proof**: `bun run build` succeeds; output is a single HTML file
- **Visual proof**: all routes render with warm cream/beige palette, blue accent, no purple/cool tones
- **Cleanup proof**: `grep -r "\.dark" web/src/index.css` returns zero `.dark` blocks; no `dark:` utility classes in components
- **Token proof**: `grep -c "^  --" web/src/index.css` shows ~90+ custom properties defined
- **Go build proof**: `make build` produces a working binary

## Key Decisions
1. **Full rewrite (Option C)** over layered migration (Option B) or big-bang replace (Option A).
   - Why: user chose maximum thoroughness. New theme layer with all components rewritten to use new token names. Clean end state with no legacy variable names.
   - Rejected A: too risky without component updates. Rejected B: leaves temporary duplication and legacy names.
2. **oklch conversion** over keeping hex from tokens.
   - Why: user chose oklch for perceptual uniformity and consistency with existing codebase conventions.
3. **Light-only** over deriving dark mode or keeping current dark.
   - Why: new token source only defines light palette. Deriving dark mode adds scope without a reference design. Ship light-only, add dark later when tokens exist.
4. **Full token adoption** over core-only or shadcn-subset.
   - Why: user chose maximum adoption. All ~90 tokens including graph-*, layout, spacing scale, component dimensions. Future-proofs component development against the full design system.
5. **Dual variable mapping** — keep shadcn standard names AND add extended names.
   - Why: shadcn components depend on `--background`, `--foreground`, `--primary` etc. Breaking these breaks all shadcn primitives. Map both: `--background` = `--bg-primary` value, plus define `--bg-primary` explicitly.
6. **Direct radius values** instead of calc-from-base.
   - Why: new token file defines explicit values (3px, 4px, 6px, 16px) that don't follow a calc pattern. Simpler to define directly.

## Deferred Ideas
- Dark mode theme (needs separate token definition)
- Component-level tokens (button bg/padding/radius from `design-token-new.json` components section)
- Font face declarations (new tokens define empty fontFaces)
- CSS container queries using --container-* tokens
- Graph visualization UI (tokens ready, no component yet)

## Ambiguity Report
- Goal clarity: high
- Scope clarity: high
- Constraints clarity: high
- Acceptance clarity: high
