# Plan: Cleanup & Verify

Phase: cleanup-verify
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-27

## Goal
Delete all SCSS files and the sass dependency. Remove SCSS config from Vite. Run the full verification suite.

## Inputs
- All pages rebuilt from Phase 4 (zero SCSS imports in TSX)
- List of all `.module.scss` files (20 files)
- List of all global SCSS files (7 files)
- `web/package.json` (sass dependency)
- `web/vite.config.ts` (SCSS preprocessor config)

---

## Wave 1: Delete SCSS files and remove sass
### T1 — Delete all SCSS module files
- type: refactor
- inputs:
  - All `.module.scss` files
- touches:
  - `web/src/pages/*.module.scss` — delete 10 files
  - `web/src/features/providers/**/*.module.scss` — delete 4 files
  - `web/src/components/**/*.module.scss` — delete 6 files
- avoid:
  - deleting any non-SCSS file
  - using `rm` (use `trash` per global rule)
- steps:
  1. Run `grep -r "module.scss" web/src/ --include="*.tsx" --include="*.ts"` — must return zero results. If not, fix those files first.
  2. Collect all `.module.scss` files: `find web/src -name "*.module.scss"`
  3. Delete each with `trash`:
     ```
     find web/src -name "*.module.scss" -exec trash {} +
     ```
  4. Also delete non-module SCSS in components:
     ```
     trash web/src/components/common/PageTransition.scss
     trash web/src/components/common/SplashScreen.scss
     ```
- expected outputs:
  - Zero `.module.scss` files in `web/src/`
  - Zero `.scss` files in `web/src/components/`
- verification:
  - `find web/src -name "*.module.scss"` — zero results
  - `find web/src/components -name "*.scss"` — zero results
- stop if:
  - grep found remaining SCSS imports → fix before deleting
- escalate to:
  - page-rebuild phase (go back and fix)

### T2 — Delete global SCSS files
- type: refactor
- inputs:
  - `web/src/styles/` directory (7 files)
- touches:
  - `web/src/styles/variables.scss` — delete
  - `web/src/styles/themes.scss` — delete
  - `web/src/styles/reset.scss` — delete
  - `web/src/styles/layout.scss` — delete
  - `web/src/styles/global.scss` — delete
  - `web/src/styles/components.scss` — delete
  - `web/src/styles/mixins.scss` — delete
- avoid:
  - deleting `web/src/index.css` (Tailwind entry file)
- steps:
  1. Check `web/src/main.tsx` no longer imports `./styles/global.scss` — if it does, remove that import line
  2. Delete all 7 files: `trash web/src/styles/*.scss`
  3. If `web/src/styles/` directory is now empty, leave it (index.css may be elsewhere) or remove if truly empty
- expected outputs:
  - Zero SCSS files in `web/src/styles/`
- verification:
  - `find web/src/styles -name "*.scss"` — zero results
  - `grep -r "\.scss" web/src/main.tsx` — zero matches
- stop if:
  - main.tsx still imports global.scss → fix the import first
- escalate to:
  - page-rebuild phase

### T3 — Remove sass dependency and Vite SCSS config
- type: refactor
- inputs:
  - `web/package.json`
  - `web/vite.config.ts`
- touches:
  - `web/package.json` — remove `sass` from devDependencies
  - `web/vite.config.ts` — remove `css.preprocessorOptions.scss` block
  - `web/src/types/style.d.ts` — remove SCSS module type declaration if present
- avoid:
  - removing any other dependency
  - changing any other Vite config
- steps:
  1. Remove `sass` from devDependencies: `cd web && bun remove sass`
  2. Edit `vite.config.ts`:
     - Remove the `css.preprocessorOptions.scss` block (the `additionalData` line)
     - If `css` key is now empty, remove it entirely
  3. Check `web/src/types/style.d.ts` — if it declares `*.module.scss`, delete the file or remove those declarations
  4. Run `bun install` to update lockfile
- expected outputs:
  - No sass in package.json
  - No SCSS config in vite.config.ts
- verification:
  - `grep "sass" web/package.json` — zero matches (check devDependencies section)
  - `grep "scss" web/vite.config.ts` — zero matches
  - `cd web && bun install` — succeeds
- stop if:
  - N/A
- escalate to:
  - N/A

---

## Wave 2: Full verification suite
### T4 — Type-check and build
- type: test
- inputs:
  - All changes from Wave 1
- touches:
  - nothing
- avoid:
  - any code changes
- steps:
  1. `cd web && bun run type-check` — must exit 0
  2. `cd web && bun run build` — must exit 0
  3. Verify `dist/index.html` is a single file
  4. Check file size: `ls -la web/dist/index.html`
  5. Verify no SCSS references in built output: `grep -c "scss" web/dist/index.html` — zero or near-zero
- expected outputs:
  - Clean type-check, successful build, single HTML file
- verification:
  - Both commands exit 0
  - `ls -la web/dist/index.html` — file exists
- stop if:
  - build fails → diagnose and fix
- escalate to:
  - page-rebuild or shadcn-primitives phase depending on error type

### T5 — Visual and functional verification
- type: test
- inputs:
  - Built application
- touches:
  - nothing
- avoid:
  - any code changes
- steps:
  1. Run `cd web && bun run dev`
  2. Check all 10 routes render:
     - `/` (Dashboard)
     - `/ai-providers` (Providers Workbench)
     - `/auth-files` (Auth Files)
     - `/auth-files/oauth-excluded`
     - `/auth-files/oauth-model-alias`
     - `/oauth` (OAuth)
     - `/quota` (Quota)
     - `/config` (Config)
     - `/logs` (Logs)
     - `/system` (System)
  3. For each route, verify:
     - Design token aesthetic: VT323 headings, sharp corners, blueprint blue accent
     - Correct data rendering (if backend is running)
     - Mobile responsive at ≤768px viewport
  4. Theme toggle: switch between light and dark — both render correctly
  5. Language toggle: switch between en, zh-CN, zh-TW, ru — all strings update
  6. Page transitions: navigate between routes — motion animations work
  7. CodeMirror: open Config page — YAML editor renders, editing works, diff modal opens
  8. Login page: visit `/login` — form renders with design tokens
- expected outputs:
  - All routes render correctly in both themes
  - All functional behaviors preserved
- verification:
  - Manual visual inspection of all 10 routes
  - Theme toggle works
  - Language toggle works
  - Page transitions animate
  - CodeMirror editable
- stop if:
  - any route fails to render → route back to page-rebuild
  - any functional regression → investigate and fix
- escalate to:
  - page-rebuild phase for visual issues; shadcn-primitives phase for component issues

---

## Wave 3: Go build verification
### T6 — Full Go binary build
- type: test
- inputs:
  - Verified frontend build from T4
- touches:
  - nothing (builds from repo root)
- avoid:
  - any backend code changes
- steps:
  1. From repo root: `make build`
  2. Verify binary builds successfully
  3. Verify binary size is reasonable (compare to pre-rebuild if possible)
  4. Optionally run the binary and access management panel at `/management.html`
- expected outputs:
  - Go binary builds with embedded frontend
  - Management panel accessible via binary
- verification:
  - `make build` exits 0
  - Binary exists at expected output path
- stop if:
  - `make build` fails → check if `make embed` step works
- escalate to:
  - user clarification (build pipeline issue)

## Risks / Watch-fors
- Hidden SCSS imports in barrel files (`index.ts`) that aren't direct imports in TSX
- `PageTransition.scss` and `SplashScreen.scss` are non-module imports — easy to miss in grep
- `style.d.ts` SCSS module type declarations may cause phantom type errors after SCSS removal
- `bun.lock` may need updating after sass removal
- `make build` depends on `make embed` which depends on `bun run build` — full chain must work
