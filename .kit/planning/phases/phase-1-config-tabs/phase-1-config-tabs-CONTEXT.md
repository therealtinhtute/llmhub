# Context: Config Tabs Refactor

Phase: phase-1-config-tabs
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: low
Expected Proof: e2e (manual) — navigate config page on desktop + mobile

## Goal
Replace the current sticky dual-column aside (desktop) + 2-col grid (mobile) section navigation in `VisualConfigEditor` with a single horizontal Radix UI Tabs component. Error badges preserve their current behavior (count on tab triggers).

## Scope Boundary
### Allowed Surfaces
- `src/components/config/VisualConfigEditor.tsx` — replace nav with Radix Tabs primitives
- `src/components/config/ConfigSection.tsx` — remove horizontal scroll-snap from parent if present
- `package.json` — add `@radix-ui/tabs` if not already present
- i18n keys related to tab labels (already exist as section titles)

### Forbidden Surfaces
- Any route or page beyond ConfigPage
- Visual | Source toggle behavior
- `PageTransition.tsx` (other phase)
- Any backend or Go code

## Spec Hooks
- Task 1 from SPEC.md: shadcn-style horizontal Tabs replacing aside header tabs
- Radix UI consistent with existing stack (Sheet, Dialog, AlertDialog already in use)
- Mobile horizontal scrollable tabs

## Locked Decisions
- Use `@radix-ui/tabs` — project's existing Radix UI v1.4.3; need to confirm `tabs` package is available or add it
- TabsList is a single horizontal row — not a multi-column grid
- All 6 TabsContent panels rendered in DOM (not lazy) — spec explicitly states this
- Mobile: horizontal scroll via Radix Tabs `ScrollArea` or native overflow-x
- IntersectionObserver scroll-spy removed — replaced by native Tabs active-state

## Assumptions
- `@radix-ui/tabs` is not yet in `package.json` — will need `bun add @radix-ui/tabs`
- `ConfigSection` relies on `scroll-snap-type: x_mandatory` — removing the parent scroll container may break section layout; need to verify
- Radix TabsTrigger supports error badge styling natively via `data-state` attributes

## Canonical Refs
- SPEC.md (`.kit/planning/SPEC.md`)
- `src/components/config/VisualConfigEditor.tsx` (current implementation)
- `src/components/config/ConfigSection.tsx` (section wrapper)
- Radix UI Tabs docs: `https://www.radix-ui.com/primitives/docs/components/tabs`

## Rejected Options
- **Keeping dual nav (desktop aside + mobile grid)** — duplicated state, no ARIA tab semantics; explicitly rejected in brainstorm
- **Using a different tabs library** — shadcn-ui or reach-ui would add a new package; Radix is already in use
- **Lazy TabsContent rendering** — spec says all sections always rendered; deferred idea

## Deferred Ideas
- Tab content lazy rendering (only active tab in DOM)
- Tab transition animation (fade between tabs)

## Escalate If
- `@radix-ui/tabs` cannot be added due to version conflict with existing Radix packages
- Removing scroll-snap breaks `ConfigSection` layout in a way that requires significant refactoring
- Error badge behavior cannot be preserved without IntersectionObserver
