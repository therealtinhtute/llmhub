# SPEC: Design System Migration — Supermemory Console Aesthetic

Status: draft
Input Type: change-request
Lane: normal
Risk Flags: existing-behavior, public-contract
Affected Surfaces: browser
Downstream: plan full
Updated At: 2026-05-28

## Source Mode
lock-from-idea

## Source Inputs
- Current `web/` codebase (Tailwind v4 + shadcn/ui "new-york" style, retro VT323 aesthetic)
- Supermemory `packages/ui/globals.css` (oklch tokens, Space Grotesk, zinc base)
- Supermemory `packages/ui/components.json` (shadcn new-york, zinc, CSS variables)
- Legacy component usage audit (13 Legacy* wrappers across 71 file references)

## Scenario
The web UI was rebuilt in Phase 2 with Tailwind v4 + shadcn/ui using a retro 90s aesthetic (VT323, Source Serif 4, hard shadows, 0px radius, warm neutral palette). The visual identity needs to shift entirely to the supermemory console style: modern, clean, cool-toned with oklch colors, Space Grotesk typography, subtle shadows, and rounded corners. All 13 Legacy* wrapper components should be removed in the same pass.

## Goal
1. Replace ALL design tokens (colors, typography, shadows, radius, spacing) with the supermemory console values
2. Remove all 13 Legacy* wrapper components and migrate their 71 consumer files to use shadcn/ui primitives directly
3. Delete all legacy CSS compatibility aliases and global utility classes
4. Add missing shadcn components that supermemory uses and the project could benefit from (tabs, textarea, progress, combobox, drawer)

## Users / Actors
- Primary maintainer: implementing the migration
- Operators: using the management panel to manage LLMHub instances

## Requirements

### R1: Token Replacement — Colors (oklch)
1. Replace all `:root` color values with supermemory oklch equivalents
2. Replace all `.dark` color values with supermemory oklch dark equivalents
3. Add chart color tokens (`--chart-1` through `--chart-5`) from supermemory
4. Remove custom tokens not in supermemory system: `--success`, `--warning`, `--info`, `--code-bg`, `--shadow-hard`, `--shadow-hard-lg`
5. Add supermemory shadow scale overrides (`--shadow-2xs` through `--shadow-2xl`)

### R2: Token Replacement — Typography
6. Replace font stacks: `--font-sans` → Space Grotesk + Inter; `--font-mono` → Space Mono
7. Remove `--font-display` (VT323) and `--font-body` (Source Serif 4) custom font families
8. Add `--font-serif` fallback stack from supermemory
9. Set base font size to 14px and letter-spacing to -0.01em via `@layer base` body rule
10. Update `index.html` Google Fonts links: remove VT323, Source Serif 4, JetBrains Mono; add Space Grotesk, Space Mono, Inter

### R3: Token Replacement — Geometry & Shadows
11. Change `--radius` from `0px` to `0.75rem`
12. Remove `--shadow-hard` and `--shadow-hard-lg` custom shadow tokens
13. Add supermemory's subtle shadow scale (`--shadow-2xs` through `--shadow-2xl`)
14. Update `--spacing` to `0.25rem`
15. Add `--tracking-normal: -0.01em`

### R4: @theme Block Update
16. Update `@theme` to match supermemory: map `--font-sans`, `--font-serif`, `--font-mono` to new stacks
17. Remove `--font-display` and `--font-body` from `@theme`
18. Add `--color-chart-*` and `--color-sidebar-*` tokens to `@theme`

### R5: Base Layer
19. Add `@layer base` rule: `* { @apply border-border outline-ring/50; }` and `body { @apply bg-background text-foreground; font-size: 14px; letter-spacing: var(--tracking-normal); }`
20. Add mobile input zoom prevention (16px font-size on inputs for iOS)

### R6: Animation & Dependencies
21. Add `tw-animate-css` package and `@import "tw-animate-css"` to index.css
22. Evaluate which custom `@keyframes` to keep (splash, page transitions used by motion library) vs remove (orbFloat, watermarkEnter, heroEnter — unused after token swap)
23. Keep motion library for page transitions

### R7: Legacy Component Removal
24. Delete `LegacyCard.tsx` → migrate 8 consumer files to shadcn `Card`/`CardHeader`/`CardContent`
25. Delete `LegacyModal.tsx` → migrate 8 consumer files to shadcn `Dialog`
26. Delete `LegacyInput.tsx` → migrate 8 consumer files to shadcn `Input` + `Label`
27. Delete `LegacyToggleSwitch.tsx` → migrate 9 consumer files to shadcn `Switch` + `Label`
28. Delete `LegacySelect.tsx` → migrate 6 consumer files to shadcn `Select`
29. Delete `LegacyEmptyState.tsx` → migrate 8 consumer files to a small `EmptyState` component using Tailwind utilities
30. Delete `LegacySelectionCheckbox.tsx` → migrate 5 consumer files to shadcn `Checkbox` + `Label`
31. Delete `LegacyLoadingSpinner.tsx` → migrate 5 consumer files to inline `Loader2` icon usage
32. Delete `LegacyCollapsible.tsx` → migrate 3 consumer files to shadcn `Collapsible`
33. Delete `LegacyAutocompleteInput.tsx` → migrate 2 consumer files to shadcn `Combobox` pattern (Input + Popover)
34. Delete `LegacySheet.tsx` → migrate 1 consumer file to shadcn `Sheet`
35. Delete `LegacySkeleton.tsx` → migrate 1 consumer file to shadcn `Skeleton`
36. Delete `LegacyTable.tsx` → migrate 1 consumer file to shadcn `Table`

### R8: Legacy CSS Cleanup
37. Delete all SCSS compatibility aliases (lines 230-273 of current index.css): `--text-primary`, `--text-secondary`, `--bg-*`, `--glass-*`, `--*-badge-*`, etc.
38. Delete all global utility classes (lines 275-422 of current index.css): `.status-badge`, `.error-box`, `.hint`, `.item-list`, `.item-row`, `.form-group`, `.input`, `.pill`
39. Migrate any consumers of deleted global classes to Tailwind utility classes inline
40. Delete `App.css` if fully unused after migration

### R9: Component Style Updates
41. Update all `font-display` references in TSX to `font-sans` (headings no longer use a separate display font)
42. Update all `font-body` references in TSX to `font-sans`
43. Replace `rounded-none` explicit overrides with default `rounded-*` classes (radius is now 0.75rem, not 0px)
44. Replace `shadow-hard` / `shadow-hard-lg` usages with standard `shadow-sm` / `shadow-md`
45. Update `components.json` base color from `neutral` to `zinc`

### R10: Custom Scrollbar Styles
46. Add `.custom-scrollbar` utility class matching supermemory's thin scrollbar styling
47. Apply to scroll areas as needed

### R11: Verification
48. `bun run build` produces a single `dist/index.html` with all CSS inlined
49. `bun run type-check` passes with zero errors
50. All routes render correctly in both light and dark themes with the new visual identity
51. No references to VT323, Source Serif 4, JetBrains Mono remain in CSS or HTML
52. No Legacy* component files exist in `src/components/ui/`
53. No Legacy* imports exist anywhere in the codebase
54. No SCSS compatibility variables (`--text-primary`, `--glass-*`, etc.) remain
55. No global utility classes (`.status-badge`, `.item-row`, etc.) remain in index.css
56. All API integrations continue working unchanged
57. Mobile responsive behavior preserved
58. i18n switching works
59. Page transitions (motion) work
60. `make build` (full Go binary) succeeds

## Boundaries

### In Scope
- Full replacement of color tokens (hex/rgba → oklch)
- Full replacement of typography (VT323/Source Serif 4/JetBrains Mono → Space Grotesk/Space Mono/Inter)
- Border radius change (0px → 0.75rem)
- Shadow system change (hard offset → subtle opacity)
- Removal of all 13 Legacy* wrapper components
- Migration of 71 consumer files
- Deletion of SCSS compatibility aliases
- Deletion of global utility classes
- Addition of tw-animate-css
- Custom scrollbar styles
- components.json base color update

### Out of Scope
- Changing Zustand stores (except removing Legacy-related code)
- Changing API layer or typed callers
- Changing routing structure
- Adding new pages or features
- Backend changes
- Adding supermemory-specific components (grid-plus, PixiJS graph, glassmorphism)
- Changing motion/framer-motion logic
- Adding recharts/chart component
- Adding cmdk command palette
- Adding vaul drawer
- Changing i18n keys

## Constraints
- Single-file build (`vite-plugin-singlefile`) must keep working
- Google Fonts CDN for Space Grotesk, Space Mono, Inter (not inlined)
- Bun as package manager
- `@/` path alias preserved
- All i18n keys unchanged
- Keep motion library for animations

## Design Token Mapping

### Colors (Light) — oklch
| Token | oklch Value | Role |
|-------|------------|------|
| `--background` | `oklch(0.9846 0.0017 247.8389)` | Page background (cool off-white with blue tint) |
| `--foreground` | `oklch(0.2744 0.0073 285.9081)` | Primary text |
| `--card` | `oklch(0.9851 0 0)` | Card surface |
| `--primary` | `oklch(0.614 0.2014 258.1073)` | Accent blue |
| `--secondary` | `oklch(0.9378 0.0296 262.5395)` | Secondary surface |
| `--muted` | `oklch(0.9846 0.0017 247.8389)` | Muted surface |
| `--muted-foreground` | `oklch(0.5693 0 0)` | Secondary text |
| `--accent` | `oklch(0.92 0.0054 286.2936)` | Accent surface |
| `--destructive` | `oklch(0.6368 0.2078 25.3313)` | Error/danger |
| `--border` | `oklch(0.9366 0.0017 247.8401)` | Borders |
| `--ring` | `oklch(0 0 0)` | Focus ring |
| `--radius` | `0.75rem` | Border radius |

### Colors (Dark) — oklch
| Token | oklch Value | Role |
|-------|------------|------|
| `--background` | `oklch(0.1487 0.0073 258.0408)` | Page background (cool blue-black) |
| `--foreground` | `oklch(0.967 0.0029 264.5419)` | Primary text |
| `--card` | `oklch(0.1487 0.0073 258.0408)` | Card surface (same as bg) |
| `--primary` | `oklch(0.614 0.2014 258.1073)` | Accent blue (same) |
| `--accent` | `oklch(0.3012 0 0)` | Accent surface |
| `--border` | `oklch(0.3715 0 0)` | Borders |
| `--ring` | `oklch(1 0 0)` | Focus ring (white) |

### Typography
| Role | Font | Tailwind Class |
|------|------|---------------|
| Headings | Space Grotesk | `font-sans text-xl font-semibold` |
| Body | Space Grotesk | `font-sans text-sm` (14px base) |
| Code | Space Mono | `font-mono text-sm` |

### Shadows
Subtle opacity shadows (0.02–0.1) replacing hard offset shadows. All shadow scale values from `--shadow-2xs` through `--shadow-2xl` redefined.

## Validation Expectations
- **Build proof**: `bun run build` succeeds; output is a single HTML file with inlined CSS
- **Visual proof**: all routes render with supermemory aesthetic (Space Grotesk, rounded corners, cool zinc palette, subtle shadows) in both light and dark
- **Cleanup proof**: `grep -r "Legacy" web/src/` returns zero results; `grep -r "VT323\|Source Serif\|JetBrains" web/src/` returns zero results; `grep -r "shadow-hard" web/src/` returns zero results
- **Functional proof**: all API integrations, login flow, provider management, config editing work identically
- **Go build proof**: `make build` produces a working binary with the new frontend embedded

## Key Decisions
1. **Full visual migration (Option A)** over cherry-pick (Option B) or fork-as-package (Option C).
   - Why: user wants complete aesthetic replacement. Cherry-picking creates visual inconsistency. Forking the package adds monorepo complexity this single-app project doesn't need.
2. **oklch color space** over keeping hex/rgba.
   - Why: supermemory uses oklch throughout; perceptually uniform; modern standard. Converting back to hex defeats the purpose.
3. **Remove Legacy* components now** over keeping them.
   - Why: they're wrappers around shadcn primitives. With the token swap, they'd need restyling anyway — cheaper to inline their shadcn equivalents directly.
4. **Space Grotesk for everything** (no separate display font) over keeping a display font.
   - Why: supermemory uses Space Grotesk for both headings and body. Simpler font stack, consistent visual identity.
5. **Keep custom EmptyState** as a small Tailwind utility component.
   - Why: no shadcn equivalent exists. 8 files use it. A 10-line component is simpler than duplicating the pattern 8 times.
6. **Don't adopt supermemory-specific components** (grid-plus, glassmorphism, PixiJS).
   - Why: those serve supermemory's memory graph use case, not a proxy management panel.

## Deferred Ideas
- Command palette (cmdk) for quick navigation
- Drawer component (vaul) for mobile provider editing
- Chart component (recharts) for dashboard metrics
- Auto theme preference (detect system dark mode)
- Glassmorphism effects for overlays
- Plus-pattern decorative backgrounds

## Ambiguity Report
- Goal clarity: high
- Scope clarity: high
- Constraints clarity: high
- Acceptance clarity: high
