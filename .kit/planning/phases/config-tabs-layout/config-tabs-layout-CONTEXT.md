# Context: Config Tabs Layout Fix

Phase: config-tabs-layout
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: low
Expected Proof: visual (browser inspection at desktop + mobile widths)

## Goal
Replace the 3-col/2-col grid on `Tabs.List` with a single horizontal flex row. Remove the redundant "快速跳转" info bar. Make tab triggers compact.

## Scope Boundary
### Allowed Surfaces
- `web/src/components/config/VisualConfigEditor.tsx` — TabsList layout, info bar removal, trigger compaction

### Forbidden Surfaces
- `ConfigSection.tsx` — no changes needed for this phase
- Tab content rendering behavior — stays as default Radix unmount-inactive
- Any backend files
- `PageTransition.tsx` or `MainLayout.tsx` — belongs to Phase 2

## Spec Hooks
- Task 1: "Tabs.List uses flex flex-row overflow-x-auto"
- Task 1: "Remove the 快速跳转 info bar entirely"
- Task 1: "Tab triggers become compact: icon + label + optional error badge, no index number prefix needed"

## Locked Decisions
- Flex row with `overflow-x-auto` — not a grid, not wrapping
- Remove the info bar div entirely (lines 323-339 in current file)
- Remove the `indexLabel` concept from tab triggers (the `01`, `02` numbering)
- Keep `ConfigSection` index labels inside tab content unchanged — only tab triggers lose the prefix

## Assumptions
- Radix `Tabs.List` accepts arbitrary className for layout — confirmed by current usage
- 6 tab triggers fit comfortably in a single row at ≥768px — triggers are icon + short label (~80-100px each)
- Horizontal scroll at <768px is acceptable UX

## Canonical Refs
- `.kit/planning/SPEC.md` — Task 1
- `web/src/components/config/VisualConfigEditor.tsx` — lines 321-377 (current TabsList + triggers)

## Rejected Options
- Keep 3-col/2-col grid: was a misimplementation of the original spec intent
- 6-col grid: tabs too narrow, doesn't scroll on mobile
- Flex wrap: multi-row tabs are ugly and waste vertical space

## Deferred Ideas
- Keyboard arrow-key tab navigation (Radix supports natively, just not styled)
- Subtle fade animation between tab contents

## Escalate If
- Radix `Tabs.List` doesn't support `overflow-x-auto` properly (scroll clipping on triggers)
- Tab triggers overflow even at desktop widths — revisit trigger sizing in `brainstorm`
