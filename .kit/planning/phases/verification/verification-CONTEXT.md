# Context: Verification

Phase: verification
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: none (read-only phase)
Expected Proof: build, grep, visual

## Goal
Validate the complete migration: build passes, no stale artifacts, token count correct, visual appearance matches warm cream/beige palette, Go binary builds successfully.

## Scope Boundary
### Allowed Surfaces
- Read-only: all `web/src/` files
- Build commands: `bun run build`, `make build`
- Grep/search commands for verification

### Forbidden Surfaces
- No file modifications in this phase
- No new features or components

## Spec Hooks
- R14: Verification (items 41-48)
- Validation Expectations (all 5 proofs)

## Locked Decisions
- Build proof: `bun run build` must succeed
- Cleanup proof: grep-based validation for `.dark`, `dark:`, stale tokens
- Token proof: custom property count in index.css
- Go build proof: `make build` must succeed

## Assumptions
- Phases 1-3 are complete
- Dev server can be started for visual inspection

## Canonical Refs
- `.kit/planning/SPEC.md` Validation Expectations section

## Rejected Options
- Automated visual regression testing — rejected; no baseline screenshots exist, manual inspection sufficient for this scope

## Deferred Ideas
- Screenshot baseline for future regressions

## Escalate If
- Build fails after all phases
- Visual appearance doesn't match warm cream palette expectation
