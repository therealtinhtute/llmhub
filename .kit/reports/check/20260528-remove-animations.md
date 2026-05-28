# Check Report — Remove All Transitions, Animations & Glow Borders

**Date**: 2026-05-28  
**Scope**: 47 files in `web/src/` — strip all CSS motion and glow effects  
**Depth**: Standard (247 lines removed, 47 files)

## Gate

| Check | Result |
|-------|--------|
| TypeScript | ✅ `tsc --noEmit` — zero errors |
| ESLint | ✅ 0 errors, 3 pre-existing warnings (unchanged) |
| Build | ✅ `vite build` — 2,117 kB, 514ms |

## Scope Drift

**On target** — all 47 changed files are under `web/src/`, all changes trace to animation/transition/glow removal.

## Findings

### 🟠 Major — `bg-muted` removed from PageTransition layers (FIXED)
Agent removed `bg-muted` from 3 layer className strings in `PageTransition.tsx`. This is a background color, not an animation — layers would become transparent. **Fixed during review** by restoring `bg-muted` on all 3 variants.

### 🟠 Major — `tw-animate-css` dead dependency (FIXED)
Import removed from `index.css` but package remained in `package.json`. **Fixed during review**: removed from `package.json`, ran `bun install`, verified build still passes.

### 🟡 Minor — DashboardPage cosmetic reformatting
Agent reformatted ~30 lines (trailing commas, import collapsing, `ref` ternary flattening) beyond the animation removal scope. No semantic changes — purely cosmetic. Not blocking.

### 🟡 Minor — `motion` package still in dependencies
`motion/mini` is still imported by `PageTransition.tsx` and `AuthFilesPage.tsx` for WAAPI-based programmatic animations. These were not in scope to remove. Dependency is correctly retained.

### 💡 Suggestion — `focus:outline-hidden` on Sheet/Dialog close buttons
Now that `focus:ring-*` was removed, the only focus indicator on these close buttons is `hover:opacity-100`. Keyboard users see no focus ring at all. Consider adding `focus-visible:border-ring` as a future improvement if accessibility matters.

## Verification

```
tsc --noEmit              → 0 errors
eslint src/               → 0 errors (3 pre-existing warnings)
vite build                → ✅ 2,117 kB, 514ms
bun install               → 1 package removed (tw-animate-css)
grep transition/animation → 0 remaining (excluding animate-spin/pulse)
grep focus:shadow/ring    → 0 remaining glow effects
```

## Verdict

**APPROVE** — 2 issues caught and fixed during review. No blockers remain.
