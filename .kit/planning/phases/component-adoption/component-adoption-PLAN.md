# Plan: Component Token Adoption

Phase: component-adoption
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-05-28

## Goal
Update components with hardcoded colors to use new semantic tokens. Update layout components to consume layout/dimension tokens.

## Inputs
- Phase 1 output: `web/src/index.css` with all tokens defined
- Phase 2 output: no dark mode artifacts
- `web/src/components/config/DiffModal.tsx` — 15 hardcoded hex references
- `web/src/components/quota/quotaStyles.ts` — hardcoded hex in gradients/badges
- `web/src/components/modelAlias/ModelMappingDiagram.tsx` — 8-color hardcoded array
- `web/src/components/ui/sidebar.tsx` — sidebar width values
- `.kit/planning/SPEC.md` — requirement R13

## Wave 1
### T1 — Update DiffModal.tsx to use semantic tokens
- type: refactor
- inputs:
  - `web/src/components/config/DiffModal.tsx`
  - New tokens: `--success`, `--error`, `--accent`
- touches:
  - `web/src/components/config/DiffModal.tsx`
- avoid:
  - Changing diff logic or component structure
  - Other config components
- steps:
  1. Replace `#3fb950` (git green) with `var(--success)` or oklch equivalent
  2. Replace `#f85149` (git red) with `var(--error)` or oklch equivalent
  3. Replace `#388bfd` (git blue) with `var(--accent)` or oklch equivalent
  4. Simplify `color-mix(in_srgb, #3fb950_8%, hsl(...))` expressions to use `var(--success-muted)` or equivalent oklch with alpha
  5. Simplify `color-mix(in_srgb, #f85149_8%, hsl(...))` to use `var(--error-muted)` or equivalent
  6. Simplify `color-mix(in_srgb, #388bfd_8%, hsl(...))` to use `var(--accent-muted)` or equivalent with alpha
- expected outputs:
  - DiffModal with zero hardcoded hex colors (using semantic tokens)
- verification:
  - `grep -c "#[0-9a-fA-F]\{6\}" web/src/components/config/DiffModal.tsx` returns 0
  - `cd web && bun run build` succeeds
- stop if:
  - color-mix expressions can't be cleanly replaced (visual regression risk)
- escalate to:
  - user clarification for visual fidelity vs token purity tradeoff

### T2 — Update quotaStyles.ts to use semantic tokens
- type: refactor
- inputs:
  - `web/src/components/quota/quotaStyles.ts`
  - New tokens: `--warning`, `--success`, `--error`
- touches:
  - `web/src/components/quota/quotaStyles.ts`
- avoid:
  - Changing quota logic or component structure
- steps:
  1. Replace `#e0aa14` quota medium color with `var(--warning)` or keep as fallback value referencing token
  2. Replace complex gradient strings with token-based equivalents where possible
  3. Replace hardcoded badge colors (`#6b4b00`, gold gradients) with semantic tokens
  4. For complex gradients that can't map to tokens, keep inline but note in comments
- expected outputs:
  - quotaStyles with semantic token usage where feasible
- verification:
  - `cd web && bun run build` succeeds
  - Visual inspection of quota badges still look correct
- stop if:
  - Complex gradient strings break when tokenized
- escalate to:
  - user clarification — some gradients may intentionally stay hardcoded

### T3 — Update ModelMappingDiagram.tsx to use graph tokens
- type: refactor
- inputs:
  - `web/src/components/modelAlias/ModelMappingDiagram.tsx`
  - New tokens: `--graph-*`
- touches:
  - `web/src/components/modelAlias/ModelMappingDiagram.tsx`
- avoid:
  - Changing diagram logic or layout
- steps:
  1. Replace hardcoded color array `['#8b8680', '#10b981', '#f59e0b', '#c65746', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16']` with references to graph tokens or a CSS custom property array
  2. If graph tokens don't map 1:1 to the 8-color array, use the closest semantic matches or keep a curated array using `var()` references
- expected outputs:
  - ModelMappingDiagram using token-based colors
- verification:
  - `cd web && bun run build` succeeds
  - Diagram colors render correctly
- stop if:
  - The 8-color array serves a specific semantic purpose (provider-to-color mapping) that tokens don't cover
- escalate to:
  - user clarification on whether diagram colors should be tokenized or remain decorative

## Wave 2
### T4 — Update sidebar.tsx with layout tokens
- type: refactor
- inputs:
  - `web/src/components/ui/sidebar.tsx`
  - New tokens: `--sidebar-width`, `--sidebar-collapsed-width`
- touches:
  - `web/src/components/ui/sidebar.tsx`
- avoid:
  - Changing sidebar behavior or structure
- steps:
  1. Find hardcoded width values (likely `260px` or `16rem` and `60px` or similar)
  2. Replace with `var(--sidebar-width)` and `var(--sidebar-collapsed-width)`
- expected outputs:
  - Sidebar using layout tokens for width
- verification:
  - `cd web && bun run build` succeeds
  - Sidebar expands/collapses correctly
- stop if:
  - Sidebar uses CSS variables already from shadcn's sidebar primitive
- escalate to:
  - N/A

### T5 — Update page headers and form components with layout/typography tokens
- type: refactor
- inputs:
  - Page components in `web/src/pages/`
  - Form components in `web/src/components/ui/`
  - New tokens: `--page-header-*`, `--page-title-*`, `--input-text-size`, `--height-md`, `--label-text-size`
- touches:
  - Page components with header sections
  - `web/src/components/ui/Input.tsx`
  - `web/src/components/ui/FormInput.tsx`
  - `web/src/components/ui/FormSelect.tsx`
  - `web/src/components/ui/label.tsx`
- avoid:
  - Changing page logic, routing, or API calls
  - Adding new components
- steps:
  1. Find page header patterns (title text size, padding) and replace hardcoded values with `var(--page-title-size)`, `var(--page-header-py)`, `var(--page-header-px)`
  2. Update Input component font-size to `var(--input-text-size)` if hardcoded
  3. Update label font-size to `var(--label-text-size)` if hardcoded
  4. Only update values that are currently hardcoded — don't touch Tailwind utility classes that already resolve through the token system
- expected outputs:
  - Page headers and form components using layout/typography tokens
- verification:
  - `cd web && bun run build` succeeds
  - Pages render with correct header sizing
  - Form inputs render with correct text size
- stop if:
  - Components already use Tailwind utilities that resolve through the token system (no change needed)
- escalate to:
  - N/A

## Risks / Watch-fors
- DiffModal's `color-mix()` is the most complex replacement — test carefully
- quotaStyles gradient strings are long and fragile — prefer minimal changes
- ModelMappingDiagram's 8-color array may be intentionally decorative (provider colors) — tokenizing may over-constrain
- Sidebar might already use CSS variables from shadcn — check before changing
