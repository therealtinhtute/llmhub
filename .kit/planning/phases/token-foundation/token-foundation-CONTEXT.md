# Context: Token Foundation

Phase: token-foundation
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: build

## Goal
Replace all CSS design tokens (colors, fonts, shadows, radius, spacing) and dependencies to match supermemory console aesthetic. This is a CSS-only phase — no component TSX changes.

## Scope Boundary
### Allowed Surfaces
- `web/src/index.css` — token values, `@theme` block, `@layer base`, `@keyframes`, `@import`
- `web/index.html` — Google Fonts links
- `web/components.json` — base color
- `web/package.json` / `web/bun.lock` — add tw-animate-css

### Forbidden Surfaces
- Any `.tsx` / `.ts` component files
- Zustand stores
- API layer, routing, i18n files
- Legacy* component files (Phase 2)
- Global utility classes section of index.css (Phase 4)
- SCSS compatibility aliases section (Phase 4)

## Spec Hooks
- R1: Color tokens (oklch)
- R2: Typography (Space Grotesk, Space Mono, Inter)
- R3: Geometry (0.75rem radius, subtle shadows)
- R4: @theme block update
- R5: @layer base body rule
- R6: tw-animate-css + keyframes review
- R9.45: components.json base color → zinc

## Locked Decisions
- Use exact oklch values from supermemory globals.css — no custom adjustments
- Keep `--success`, `--warning`, `--info`, `--code-bg` tokens temporarily for backward compatibility
- Keep all `@keyframes` referenced in TSX (orbFloat, watermarkEnter, heroEnter, fadeSlideUp, cardEnter, dotPulse, brandFadeIn, splashEnter, splashLogoPulse, splashLoading)
- Remove `--shadow-hard` and `--shadow-hard-lg` from `:root` and `.dark`
- SCSS compatibility aliases left in place — cleaned up in Phase 3-4

## Assumptions
- All `@keyframes` referenced by DashboardPage, SplashScreen, LoginPage must stay
- `tw-animate-css` provides Tailwind v4 animation utilities without conflicting with motion library
- oklch is supported by all target browsers (modern evergreen)

## Canonical Refs
- `.kit/planning/SPEC.md` — "Design Token Mapping" section
- Supermemory `packages/ui/globals.css` — source of truth for oklch values
- Current `web/src/index.css` — file being rewritten

## Rejected Options
- Converting oklch to hex for browser compatibility — unnecessary, target is modern browsers
- Keeping VT323/Source Serif 4 as fallbacks — defeats purpose of full aesthetic swap

## Deferred Ideas
- Removing `--success`, `--warning`, `--info` custom tokens (evaluate after Phase 3)

## Escalate If
- `tw-animate-css` causes conflicts with motion library animations
- oklch rendering issues on any target browser
- Single-file build fails after token swap
