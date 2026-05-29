# ROADMAP: Web UI Update — Config Tabs + Animation Strip

## Planning Basis
- source spec: `.kit/planning/SPEC.md`
- planning mode: `full`
- recommended entry phase: `phase-1-config-tabs`
- execution mode: sequential (independent file changes; each phase can be verified separately)

## Entry Phase
`phase-1-config-tabs`

## Phase 1: phase-1-config-tabs
**Goal:** Replace the sticky 2/3-column aside navigation in VisualConfigEditor with a Radix UI horizontal Tabs structure. All 6 sections accessible via tab triggers; no more IntersectionObserver scroll-spy.

**Deliverables:**
- `VisualConfigEditor.tsx` refactored with Radix UI Tabs (`TabsRoot` / `TabsList` / `TabsTrigger` × 6 / `TabsContent` × 6)
- `ConfigSection.tsx` adjusted for non-scroll-snap layout (removal of horizontal scroll-snap parent)
- Mobile: horizontal scrollable TabsList
- Error badge (amber count) behavior preserved on tab triggers

**Dependencies:** none — first phase, no prior work needed

**Risks / Watch-fors:**
- Must check whether `@radix-ui/tabs` needs to be installed (project already uses `@radix-ui` v1.4.3 — confirm tabs package is included)
- `ConfigSection` horizontal scroll-snap removal may affect layout pacing — verify visually on both desktop and mobile

---

## Phase 2: phase-2-kill-animations
**Goal:** Strip all `motion` animation calls from `PageTransition.tsx`; keep layer stack, scroll position preservation, and context providers. Page transitions become instant.

**Deliverables:**
- `PageTransition.tsx` — `animate()` calls removed, no-op animation `useLayoutEffect`; layer stack and scroll preservation intact

**Dependencies:** Phase 1 complete (both are independent file changes but belong to the same spec)

**Risks / Watch-fors:**
- Verify `motion` is not used elsewhere (especially `motion/mini` imports) — if used elsewhere, package stays; leave comment if uncertain
- `PageTransitionLayerContext.isAnimating` always `false` after strip — ensure no downstream consumer depends on animation-state side effects

---

## Phase Ordering Rationale
- Phase 1 addresses the larger UI surface area (config page tabs) and benefits from being verified first
- Phase 2 is a targeted surgical edit on one component — low blast radius, quick to validate
- Independent files but share the same `SPEC.md` and can be verified separately
