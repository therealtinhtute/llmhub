# Plan: Token Foundation

Phase: token-foundation
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-28

## Goal
Replace all CSS design tokens and dependencies to match supermemory console. CSS-only changes — no component TSX modifications.

## Inputs
- `web/src/index.css` (current token definitions)
- `web/index.html` (Google Fonts links)
- `web/components.json` (shadcn config)
- `web/package.json` (dependencies)
- Supermemory oklch values (captured in SPEC.md "Design Token Mapping")

## Wave 1
### T1 — Install tw-animate-css
- type: implementation
- inputs:
  - `web/package.json`
- touches:
  - `web/package.json`, `web/bun.lock`
- avoid:
  - any TSX files
- steps:
  1. Run `cd web && bun add tw-animate-css`
- expected outputs:
  - `tw-animate-css` in dependencies
- verification:
  - `grep tw-animate-css web/package.json` returns a match
- stop if:
  - package not found or incompatible with Tailwind v4
- escalate to:
  - plan phase

### T2 — Update Google Fonts in index.html
- type: implementation
- inputs:
  - `web/index.html`
- touches:
  - `web/index.html`
- avoid:
  - any other HTML content
- steps:
  1. Read `web/index.html`
  2. Replace Google Fonts links: remove VT323, Source Serif 4, JetBrains Mono; add Space Grotesk (400;500;600;700), Space Mono (400;700), Inter (400;500;600;700)
- expected outputs:
  - `index.html` with updated font links
- verification:
  - `grep "Space+Grotesk" web/index.html` returns a match
  - `grep "VT323" web/index.html` returns no match
- stop if:
  - index.html has no existing Google Fonts links
- escalate to:
  - user clarification

### T3 — Update components.json base color
- type: implementation
- inputs:
  - `web/components.json`
- touches:
  - `web/components.json`
- avoid:
  - aliases, style, icon config
- steps:
  1. Read `web/components.json`
  2. Change `"baseColor": "neutral"` to `"baseColor": "zinc"`
- expected outputs:
  - components.json with zinc base color
- verification:
  - `grep '"baseColor": "zinc"' web/components.json` returns a match
- stop if:
  - baseColor field not found
- escalate to:
  - user clarification

## Wave 2
### T4 — Rewrite index.css token section
- type: implementation
- inputs:
  - `web/src/index.css`
  - Supermemory oklch values from SPEC.md
- touches:
  - `web/src/index.css` (lines 1-143: imports, @theme, :root, .dark)
- avoid:
  - `@keyframes` section (lines 145-195) — keep all, they have TSX consumers
  - `status-bar-tooltip` CSS (lines 197-217) — keep
  - `@theme inline` sidebar block (lines 219-228) — keep
  - SCSS compatibility aliases (lines 230-273) — keep for now
  - Global utility classes (lines 275-422) — keep for now
- steps:
  1. Read `web/src/index.css`
  2. Add `@import "tw-animate-css";` after `@import "tailwindcss";`
  3. Update `@custom-variant dark` to `(&:where(.dark, .dark *))`
  4. Rewrite `@theme` block with supermemory font stacks and color mappings
  5. Rewrite `:root` with supermemory oklch light values, new radius, shadow scale, spacing, tracking
  6. Rewrite `.dark` with supermemory oklch dark values
  7. Add `@layer base` block with global border/outline and body styles
  8. Add iOS input zoom prevention media query
- expected outputs:
  - index.css with oklch tokens, Space Grotesk fonts, 0.75rem radius, subtle shadows
- verification:
  - `grep "oklch" web/src/index.css | wc -l` returns 40+ matches
  - `grep "Space Grotesk" web/src/index.css` returns a match
  - `grep "0.75rem" web/src/index.css` returns a match
  - `grep "shadow-hard" web/src/index.css` returns no match
  - `grep "VT323\|Source Serif\|JetBrains" web/src/index.css` returns no match
- stop if:
  - index.css structure differs significantly from expected
- escalate to:
  - user clarification

## Wave 3
### T5 — Build verification
- type: test
- inputs:
  - all changes from Wave 1-2
- touches:
  - none (read-only)
- avoid:
  - any file modifications
- steps:
  1. Run `cd web && bun run build`
  2. Verify single HTML output exists
  3. Check build output contains oklch values
- expected outputs:
  - Successful build, single dist/index.html
- verification:
  - `bun run build` exits 0
  - `ls web/dist/index.html` exists
  - `grep "oklch" web/dist/index.html` returns matches
- stop if:
  - Build fails — likely CSS syntax error
- escalate to:
  - plan phase

## Risks / Watch-fors
- Removing `--shadow-hard` will break Card.tsx shadow — expected, fixed in Phase 3
- `--font-display` and `--font-body` removal means those Tailwind classes won't resolve — fixed in Phase 3
- SCSS compat aliases may reference removed vars — kept for now, checked in Phase 3-4
