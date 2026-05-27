# Context: Cleanup & Verify

Phase: cleanup-verify
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: build, type-check, visual, responsive, i18n, theme, animation, Go build

## Goal
Delete all SCSS files and the sass dependency. Remove SCSS config from Vite. Run the full verification suite to prove the rebuild is complete and nothing is broken.

## Scope Boundary
### Allowed Surfaces
- `web/src/styles/` — delete all 7 SCSS files
- `web/src/pages/*.module.scss` — delete all page SCSS modules
- `web/src/features/**/*.module.scss` — delete all feature SCSS modules
- `web/src/components/**/*.module.scss` — delete all component SCSS modules
- `web/src/components/common/PageTransition.scss` — delete (already converted)
- `web/src/components/common/SplashScreen.scss` — delete (already converted)
- `web/package.json` — remove `sass` devDependency
- `web/vite.config.ts` — remove `css.preprocessorOptions.scss` block
- `web/src/types/style.d.ts` — remove SCSS module type declaration if present

### Forbidden Surfaces
- Business logic in any TSX file
- Zustand stores
- API layer
- i18n files
- Router

## Spec Hooks
- Req 33: Delete all `.module.scss` files
- Req 34: Delete global SCSS files
- Req 35: Remove sass dep and Vite SCSS config
- Req 38–46: All verification requirements

## Locked Decisions
- Use `trash` (not `rm`) for all file deletions per global rule
- Verification order: type-check → build → visual → responsive → i18n → theme → animation → Go build
- Any remaining SCSS import in a TSX file is a blocking error — fix before proceeding

## Assumptions
- All pages and components have been rebuilt in Phase 4 with zero SCSS imports
- No TSX file references any `.module.scss` file
- No global SCSS class is still needed by any component
- CodeMirror has its own CSS loading independent of SCSS pipeline

## Canonical Refs
- All `.module.scss` files (to be deleted)
- All global SCSS files in `web/src/styles/`
- `web/vite.config.ts` SCSS config block
- `web/package.json` sass dependency

## Rejected Options
- Keep SCSS as a backup — contradicts big-bang constraint; clean break required

## Deferred Ideas
- Tailwind build size optimization (purge tuning, `@source` annotations)
- PostCSS plugin exploration
- CSS bundle analysis

## Escalate If
- `bun run build` fails after SCSS removal → hidden SCSS import somewhere; grep and fix
- `make build` (Go binary) fails → frontend build output may have changed structure
- Visual regression on any page → page rebuild incomplete; route back to page-rebuild phase
