# Plan: Tailwind Foundation

Phase: tailwind-foundation
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-27

## Goal
Install Tailwind CSS v4 + shadcn/ui into the Vite project. Map design tokens to CSS custom properties. Create light/dark themes. Add Google Fonts. Verify single-file build works.

## Inputs
- `design-token.json` (repo root) — all color, typography, spacing, shadow, radius values
- `web/src/styles/themes.scss` — existing dark theme color values to preserve
- `web/package.json` — current dependencies
- `web/vite.config.ts` — current Vite config
- `web/index.html` — add font links
- `web/tsconfig.json` — path alias verification

---

## Wave 1: Install Tailwind + shadcn
### T1 — Install Tailwind CSS v4
- type: implementation
- inputs:
  - `web/package.json`
- touches:
  - `web/package.json` (add `tailwindcss`, `@tailwindcss/vite`)
  - `web/vite.config.ts` (add Tailwind Vite plugin)
  - `web/src/index.css` (create with `@import "tailwindcss"`)
  - `web/src/main.tsx` (import `index.css`)
- avoid:
  - deleting any existing SCSS imports or files
  - changing any component code
- steps:
  1. Run `bun add -D tailwindcss @tailwindcss/vite` in `web/`
  2. Add `tailwindcss` plugin to `vite.config.ts` plugins array (before `react()`)
  3. Create `web/src/index.css` with `@import "tailwindcss";`
  4. Add `import './index.css'` to `web/src/main.tsx` (before existing style imports)
- expected outputs:
  - Tailwind CSS v4 installed and configured
  - `bun run build` still succeeds
- verification:
  - `cd web && bun run build` — succeeds without errors
  - Inspect `dist/index.html` — contains Tailwind base styles
- stop if:
  - `vite-plugin-singlefile` fails to inline Tailwind output
- escalate to:
  - user clarification (alternative inlining strategy)

### T2 — Initialize shadcn/ui
- type: implementation
- inputs:
  - `web/package.json`
  - `web/tsconfig.json` (path alias `@/`)
- touches:
  - `web/components.json` (new file)
  - `web/package.json` (shadcn peer deps: `class-variance-authority`, `clsx`, `tailwind-merge`, `lucide-react`)
  - `web/src/lib/utils.ts` (new file — `cn()` helper)
- avoid:
  - modifying any existing component files
- steps:
  1. Run `bunx shadcn@latest init` in `web/` — select: Vite, TypeScript, `src/components/ui`, `@/`, CSS variables
  2. If CLI conflicts with existing files, manually create `components.json` and `src/lib/utils.ts`
  3. Verify `components.json` has correct `aliases.components`: `@/components/ui` and `aliases.utils`: `@/lib/utils`
- expected outputs:
  - `web/components.json` configured
  - `web/src/lib/utils.ts` with `cn()` function
- verification:
  - `cat web/components.json` — confirms path aliases
  - `cd web && bun run type-check` — no new errors from utils.ts
- stop if:
  - shadcn CLI doesn't support Tailwind v4 + Vite
- escalate to:
  - plan phase (manual installation path)

---

## Wave 2: Design token mapping + themes
### T3 — Map design tokens to Tailwind @theme
- type: implementation
- inputs:
  - `design-token.json`
  - `web/src/styles/themes.scss` (dark theme values)
  - `web/src/index.css` (from T1)
- touches:
  - `web/src/index.css` — add `@theme` block and CSS custom properties
- avoid:
  - deleting existing SCSS theme file
  - changing any component
- steps:
  1. Read `design-token.json` for all color, typography, spacing, shadow, radius values
  2. Read `themes.scss` for dark theme color values
  3. Add `@theme` block to `index.css` with font families:
     - `--font-display: 'VT323', ui-monospace, 'JetBrains Mono', monospace`
     - `--font-body: 'Source Serif 4', 'Source Serif Pro', Georgia, serif`
     - `--font-mono: 'JetBrains Mono', ui-monospace, 'Consolas', monospace`
  4. Add `:root` (light theme) CSS custom properties:
     - `--background: #fafaf5`
     - `--foreground: #1a1a1a`
     - `--card: #f3f1e8`
     - `--card-foreground: #1a1a1a`
     - `--primary: #3553ff`
     - `--primary-foreground: #ffffff`
     - `--secondary: #f3f1e8`
     - `--secondary-foreground: #1a1a1a`
     - `--muted: #f3f1e8`
     - `--muted-foreground: #4a4a4a`
     - `--accent: #f3f1e8`
     - `--accent-foreground: #1a1a1a`
     - `--destructive: #c65746`
     - `--destructive-foreground: #ffffff`
     - `--border: rgba(26,26,26,0.16)`
     - `--input: rgba(26,26,26,0.16)`
     - `--ring: #3553ff`
     - `--radius: 0px`
     - `--shadow-hard: 3px 3px 0 #1a1a1a`
     - `--shadow-hard-lg: 5px 5px 0 #1a1a1a`
     - Plus success/warning/info semantic colors from design tokens
  5. Add `.dark` (dark theme) CSS custom properties:
     - `--background: #151412`
     - `--foreground: #f6f4f1`
     - `--card: #1d1b18`
     - `--card-foreground: #f6f4f1`
     - `--primary: #3553ff`
     - `--primary-foreground: #ffffff`
     - `--border: #3a3530`
     - `--muted: #262320`
     - `--muted-foreground: #c9c3bb`
     - etc. (derived from existing `themes.scss [data-theme='dark']`)
     - `--shadow-hard: 3px 3px 0 rgba(0,0,0,0.5)`
     - `--shadow-hard-lg: 5px 5px 0 rgba(0,0,0,0.5)`
- expected outputs:
  - `index.css` with complete token mapping for both themes
  - Tailwind utilities like `bg-background`, `text-foreground`, `bg-primary` available
- verification:
  - `cd web && bun run build` — succeeds
  - Add a test `<div className="bg-background text-foreground">test</div>` in App.tsx temporarily; inspect in dev server to confirm colors apply
  - Remove test div after verification
- stop if:
  - Tailwind v4 `@theme` syntax doesn't recognize custom properties for utility generation
- escalate to:
  - plan phase (may need `@theme inline` or alternative mapping approach)

### T4 — Add Google Fonts CDN
- type: implementation
- inputs:
  - `web/index.html`
  - `design-token.json` (font families)
- touches:
  - `web/index.html` — add `<link>` tags in `<head>`
- avoid:
  - any changes to component code
- steps:
  1. Add Google Fonts `<link>` tags to `web/index.html` `<head>`:
     - `<link rel="preconnect" href="https://fonts.googleapis.com">`
     - `<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>`
     - `<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&family=Source+Serif+4:wght@400;600;700&family=VT323&display=swap" rel="stylesheet">`
- expected outputs:
  - Fonts load when dev server runs
- verification:
  - `cd web && bun run dev` — open browser, inspect computed font-family on body (should show Source Serif 4 if `font-body` is applied)
- stop if:
  - none expected
- escalate to:
  - N/A

---

## Wave 3: Build verification
### T5 — Verify single-file build with Tailwind
- type: test
- inputs:
  - All changes from T1–T4
- touches:
  - nothing new (verification only)
- avoid:
  - any code changes
- steps:
  1. Run `cd web && bun run build`
  2. Check `dist/index.html` exists and is a single file
  3. Check that Tailwind CSS is present in the file (search for `--background` or Tailwind utility classes)
  4. Check that Google Fonts `<link>` tags are present
  5. Check file size — should be within ±30% of current build
  6. Run `cd web && bun run type-check` — zero errors
- expected outputs:
  - `dist/index.html` — single file with Tailwind CSS inlined
  - No type errors
- verification:
  - `ls -la web/dist/index.html` — file exists
  - `grep -c 'tailwindcss' web/dist/index.html || grep -c '\-\-background' web/dist/index.html` — >0 matches
  - `grep -c 'fonts.googleapis.com' web/dist/index.html` — >0 matches
  - `cd web && bun run type-check` — exit 0
- stop if:
  - single-file build fails or Tailwind CSS is missing from output
- escalate to:
  - user clarification (vite-plugin-singlefile compatibility)

## Risks / Watch-fors
- Tailwind v4 is relatively new — shadcn docs may lag behind on v4-specific patterns
- `@import "tailwindcss"` must come before custom CSS in `index.css`
- The existing `main.tsx` imports `./styles/global.scss` — the new `index.css` import must coexist with it until Phase 5
