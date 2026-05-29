# Plan: Strip Page Transition Animations

Phase: phase-2-kill-animations
Status: ready
Wave Count: 1
Execution Owner: work
Updated At: 2026-05-29

## Goal
Remove all `motion` animation calls from `PageTransition.tsx`; keep layer stack management, scroll position preservation, and `PageTransitionLayerContext`. Transitions become instant.

---

## Inputs
- `src/components/common/PageTransition.tsx` — current implementation
- `src/components/common/PageTransitionLayer.ts` — context provider (read-only inventory)

---

## Wave 1 — Strip animate() calls, keep structural code

### T1 — Verify motion usage across codebase
- type: verification
- inputs:
  - `package.json`
  - all source files
- touches:
  - none (read-only grep)
- steps:
  1. `grep -r "motion" web/src/` — find all files importing or using motion
  2. `grep -r "from 'motion'" web/src/`
  3. Confirm that `PageTransition.tsx` is the only consumer of `motion/mini` or the animation-specific imports
- expected outputs:
  - List of all files importing `motion`
- verification:
  - grep output shows `Motion` or `motion/mini` usage across files
- stop if:
  - other components use `motion` for non-trivial animations — those must be left alone; package remains in `package.json`
- escalate to:
  - user clarification

### T2 — Read PageTransitionLayer.ts
- type: refactor
- inputs:
  - `src/components/common/PageTransitionLayer.ts`
- touches:
  - none (read-only)
- steps:
  1. Read the file fully — understand what `PageTransitionLayerContext` exposes
  2. Note which fields are used downstream (likely `status`, `isCurrentLayer`, `isAnimating`)
  3. Confirm `isAnimating` is only a context flag, not tied to actual animation state
- expected outputs:
  - Complete understanding of layer context contract
- verification:
  - Read complete
- stop if:
  - downstream consumers depend on actual animation timing (unlikely — context only)
- escalate to:
  - user clarification

### T3 — Strip motion/animate() from PageTransition.tsx
- type: implementation
- inputs:
  - `PageTransition.tsx` (current)
  - Motion usage inventory
- touches:
  - `src/components/common/PageTransition.tsx`
- avoid:
  - Any other file
  - Removing the `motion` package from `package.json`
- steps:
  1. Identify the entire `useLayoutEffect` block that runs animations (lines ~223–406)
  2. Replace the body with a no-op that calls `completeTransition()` immediately (or removes the effect entirely)
  3. Remove the `animate` import from `motion/mini` (may leave `import { ... } from 'motion'` if other motion symbols are used, otherwise remove entirely)
  4. Remove constants: `VERTICAL_ENTER_DURATION`, `VERTICAL_EXIT_DURATION`, `VERTICAL_ENTER_DISTANCE`, `VERTICAL_EXIT_DISTANCE`, `REDUCED_MOTION_DURATION`, `IOS_TRANSITION_DURATION`, etc.
  5. Remove `prefersReducedMotion`, `buildVerticalTransform`, `buildIosTransform`, `clearLayerStyles` — all animation-specific utils
  6. Keep layer stack management (`setLayers` logic), scroll position map, `resolveScrollContainer` — these have no animation dependency
  7. Keep `PageTransitionLayerContext.Provider` — layer status still works, just transitions are instant
  8. Set `isAnimating` to always `false` in context value — no animation-driven state changes
  9. The exit layer logic (`exitingLayerRef`, `currentLayerRef`) can be simplified since there is no animation — the layer enters instantly and the old layer is immediately popped. Consider simplifying `setLayers` call to skip the `exiting` state entirely and just swap directly to `nextCurrent`.
- expected outputs:
  - `PageTransition.tsx` — layer stack + scroll preservation intact; no animation code
- verification:
  - `bun run dev` → navigate between pages → instant transitions, no slide/fade
  - Scroll position preserved on back navigation
  - `process.exit` not called; no runtime errors in console
- stop if:
  - removing the exiting-layer state causes a flash or double-render — may need to keep one render cycle of "old layer exits"
- escalate to:
  - user clarification

### T4 — Verify no motion runtime errors
- type: verification
- inputs:
  - Edited `PageTransition.tsx`
- touches:
  - none
- steps:
  1. Run `bun run dev`, open browser console
  2. Navigate: Dashboard → Config → Logs → System → Dashboard
  3. Check console for any motion-related errors (e.g., ` motion: animate target not found`)
  4. Verify scroll position is preserved when navigating back
- expected outputs:
  - Clean console on all route navigations
- verification:
  - Manual inspection complete
- stop if:
  - errors appear — revert to previous step or investigate
- escalate to:
  - user clarification

---

## Risks / Watch-fors
- **Instant exit layer flash** — stripping animation may cause the previous layer to disappear instantly. The layer management may need a minimal "mark as exiting, then immediate remove" cycle to avoid this. Test carefully.
- **isAnimating context always false** — downstream code that reads `isAnimating` from `PageTransitionLayerContext` will always get `false`. This should be fine based on the assumption scan, but verify with a grep.
- **motion package removal temptation** — after seeing the stripped code, there's a strong urge to remove `motion` from `package.json`. Resist; it's out of scope and other UI may use it.
