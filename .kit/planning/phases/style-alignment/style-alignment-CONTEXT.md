# Context: Style Alignment

Phase: style-alignment
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: build, visual

## Goal
Update all component TSX to use new design system classes. Remove retro-specific styling (rounded-none, shadow-hard, font-display/font-body). Migrate global utility class consumers and SCSS alias consumers to inline Tailwind.

## Scope Boundary
### Allowed Surfaces
- `web/src/components/ui/*.tsx` — update shadcn component classes
- All TSX files using `font-display`, `font-body`, `rounded-none`, `shadow-hard`
- All TSX files referencing global utility classes (`.status-badge`, `.error-box`, `.item-list/.item-row`, `.form-group`, `.pill`, `.hint`)
- All TSX files referencing SCSS compatibility aliases (`--text-primary`, `--glass-*`, `--bg-*`, `--*-badge-*`, `--amber-*`, `--filter-*`, `--warning-*`, `--primary-color`, `--border-color`)
- `web/src/index.css` — add `.custom-scrollbar` utility class

### Forbidden Surfaces
- Design tokens section of index.css (already done in Phase 1)
- API layer, routing, i18n, Zustand stores
- Adding new components or features
- Deleting SCSS aliases or global classes from index.css (Phase 4 — delete after consumers are migrated)

## Spec Hooks
- R8.39: Migrate consumers of global utility classes to Tailwind inline
- R9.41-R9.44: font-display→font-sans, font-body→font-sans, rounded-none removal, shadow-hard→shadow-sm
- R10.46-47: Custom scrollbar styles

## Locked Decisions
- Remove ALL `rounded-none` from shadcn UI components — the new 0.75rem radius should apply everywhere
- `shadow-[var(--shadow-hard)]` in Card.tsx → `shadow-sm` (supermemory's subtle shadow approach)
- `font-display` → `font-sans` everywhere (Space Grotesk replaces VT323 for headings)
- `font-body` → `font-sans` everywhere (Space Grotesk replaces Source Serif for body)
- Global utility class consumers get inline Tailwind — no new utility classes
- SCSS alias consumers → use semantic Tailwind classes (e.g., `--success-badge-bg` → `bg-emerald-100 dark:bg-emerald-900`)

## Assumptions
- All `rounded-none` instances in shadcn components were added for the retro look — none are structurally required
- Global utility class consumers are limited to the files found in audit (LogsPage, SystemPage, OAuthPage, VisualConfigEditor, VisualConfigEditorBlocks, ConfigPage, AuthFilesPage, AuthFilesPrefixProxyEditorModal)
- SCSS alias consumers are in ProviderStatusBar, AuthFilesPage, ProviderCategoryList, quotaStyles, ConfigSection, SecondaryScreenShell, PlaceholderPage

## Canonical Refs
- Grep audit results from brainstorm session
- `.kit/planning/SPEC.md` — R8, R9, R10

## Rejected Options
- Creating new utility classes to replace old ones — adds maintenance, prefer inline Tailwind
- Keeping `rounded-none` on some components for "sharp accent" — defeats full aesthetic migration

## Deferred Ideas
- Refactoring VisualConfigEditor's complex parent-selector overrides into proper component composition

## Escalate If
- A `rounded-none` is structurally required (e.g., for edge-to-edge rendering in a parent)
- Global utility class has more consumers than audited
- SCSS alias consumer migration produces visual regressions
