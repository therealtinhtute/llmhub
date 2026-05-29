# Context: Strip Page Transition Animations

Phase: phase-2-kill-animations
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: low
Expected Proof: unit (code inspection) + e2e (manual) — instant route changes, scroll preserved

## Goal
Remove all `motion` animation calls from `PageTransition.tsx` while keeping layer stack management, scroll position preservation per route key, and `PageTransitionLayerContext`. Transitions become instant (no spatial motion).

## Scope Boundary
### Allowed Surfaces
- `src/components/common/PageTransition.tsx` — strip `animate()` calls only
- `src/components/common/PageTransitionLayer.ts` — untouched
- Any import cleanup of `motion` that directly results from this edit

### Forbidden Surfaces
- Any component beyond PageTransition
- Config tabs (other phase)
- Removing `motion` from `package.json` (used by other UI; out of scope)
- Any route or layout changes

## Spec Hooks
- Task 2 from SPEC.md: strip animations, keep structure
- `PageTransitionLayerContext.isAnimating` always `false` after edit

## Locked Decisions
- Keep `PageTransition` component — removed component would lose scroll preservation and layer context
- Strip `animate()` calls but keep layer stack (`current` / `stacked` / `exiting`) — enables future opt-in transitions
- Keep scroll position Map (`scrollPositionsRef`) — needed per spec
- `motion` package stays in `package.json` — scope creep to remove it

## Assumptions
- `motion` library imported only by `PageTransition` — need to verify via grep before committing
- No downstream consumer depends on `isAnimating === true` for visual behavior (likely just a context flag)
- Reduced motion media query handling can be removed as part of the animation strip

## Canonical Refs
- SPEC.md (`.kit/planning/SPEC.md`)
- `src/components/common/PageTransition.tsx` (current implementation)
- `src/components/common/PageTransitionLayer.ts` (context provider)

## Rejected Options
- **Removing entire `PageTransition`** — lost scroll position preservation + layer context; no easy rollback; explicitly rejected in brainstorm
- **Keeping motion but making it instant** — complexity for no benefit

## Deferred Ideas
- Re-add transitions via a config flag or `prefers-reduced-motion` override
- Adding a `transitionEnabled` flag to re-enable spatial animations

## Escalate If
- Any non-PageTransition consumer depends on `isAnimating` context flag for non-trivial behavior
- `motion` is used by other components — means the package cannot be removed and should remain as-is
