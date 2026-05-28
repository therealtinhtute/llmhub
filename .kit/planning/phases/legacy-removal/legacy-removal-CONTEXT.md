# Context: Legacy Removal

Phase: legacy-removal
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: build, type-check

## Goal
Delete all 13 Legacy* wrapper components and migrate their 71 consumer files to use shadcn/ui primitives directly.

## Scope Boundary
### Allowed Surfaces
- `web/src/components/ui/Legacy*.tsx` — delete these files
- All 71 consumer files that import Legacy* components — update imports and JSX
- `web/src/components/ui/` — new EmptyState.tsx component (small utility)

### Forbidden Surfaces
- Design tokens / index.css (Phase 1)
- shadcn UI component internals (Button.tsx, Card.tsx, etc.) — use as-is
- API layer, routing, i18n
- Global utility classes / SCSS aliases (Phase 3-4)
- Adding new shadcn components beyond what's needed for direct replacement

## Spec Hooks
- R7.24-R7.36: Legacy component deletion and consumer migration

## Locked Decisions
- Each Legacy wrapper maps to a specific shadcn equivalent (see migration table below)
- LegacyEmptyState → new small `EmptyState` component (no shadcn equivalent exists)
- LegacyAutocompleteInput → `Input` + `Popover` combobox pattern
- LegacyLoadingSpinner → inline `Loader2` icon from lucide-react (no wrapper needed)
- Preserve any label/hint/error patterns by composing shadcn `Label` + `Input` inline

## Migration Table
| Legacy Component | Shadcn Replacement | Consumer Count |
|---|---|---|
| LegacyCard | Card, CardHeader, CardContent | 8 |
| LegacyModal | Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter | 8 |
| LegacyInput | Input + Label | 8 |
| LegacyToggleSwitch | Switch + Label | 9 |
| LegacySelect | Select, SelectTrigger, SelectContent, SelectItem | 6 |
| LegacyEmptyState | new EmptyState utility | 8 |
| LegacySelectionCheckbox | Checkbox + Label | 5 |
| LegacyLoadingSpinner | Loader2 icon inline | 5 |
| LegacyCollapsible | Collapsible, CollapsibleTrigger, CollapsibleContent | 3 |
| LegacyAutocompleteInput | Input + Popover (combobox) | 2 |
| LegacySheet | Sheet, SheetContent, SheetHeader, SheetTitle | 1 |
| LegacySkeleton | Skeleton | 1 |
| LegacyTable | Table, TableHeader, TableBody, TableRow, TableHead, TableCell | 1 |

## Assumptions
- Legacy wrappers are thin convenience layers around shadcn — migration is mostly import + JSX restructuring
- No business logic lives in Legacy* components (they're pure UI wrappers)
- Consumer files may import multiple Legacy components — handle all in one pass per file

## Canonical Refs
- Legacy component audit from brainstorm session
- shadcn/ui component API docs

## Rejected Options
- Keeping Legacy wrappers and just restyling them — adds maintenance burden, they're thin wrappers anyway
- Auto-generating migration with codemod — 71 files is manageable manually, codemods risk subtle breakage

## Deferred Ideas
- Adding shadcn Combobox as a proper component (currently using Input + Popover pattern)
- Adding shadcn Form wrapper for consistent label/error handling

## Escalate If
- A Legacy component contains business logic beyond UI wrapping
- Consumer count is higher than audited (indicates missed imports)
- Type errors after migration suggest API incompatibility
