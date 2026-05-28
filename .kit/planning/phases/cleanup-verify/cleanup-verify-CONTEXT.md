# Context: Cleanup & Verify

Phase: cleanup-verify
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: low
Expected Proof: build, type-check, grep

## Goal
Delete all dead CSS (SCSS compatibility aliases, global utility classes, App.css) and run full verification to prove migration is complete.

## Scope Boundary
### Allowed Surfaces
- `web/src/index.css` — delete SCSS aliases (lines 230-273) and global utility classes (lines 275-422)
- `web/src/App.css` — delete entirely
- Any straggler references found during verification

### Forbidden Surfaces
- Design tokens section of index.css (Phase 1 — complete)
- Component TSX logic (should be done by Phase 3)
- API layer, routing, i18n, Zustand stores
- Backend code

## Spec Hooks
- R8.37: SCSS compatibility aliases deletion
- R8.38: Global utility classes deletion
- R8.40: App.css deletion
- R11: Full verification suite

## Locked Decisions
- Delete App.css (Vite scaffold leftover — unused, verified by grep)
- Keep `status-bar-tooltip` CSS (still has consumers in ProviderStatusBar area)
- Keep `@keyframes` (all have consumers)
- Keep `@layer base` and `@theme` blocks

## Assumptions
- All consumers of SCSS aliases migrated in Phase 3
- All consumers of global utility classes migrated in Phase 3
- App.css has no active imports

## Canonical Refs
- `.kit/planning/SPEC.md` — R8, R11

## Rejected Options
- Keeping SCSS aliases "just in case" — they reference removed tokens, keeping them is confusing

## Deferred Ideas
- None — this is the final cleanup phase

## Escalate If
- Any consumer of deleted CSS is discovered during verification
- Build fails after deletion
