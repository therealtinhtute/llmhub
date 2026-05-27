# Context: shadcn Primitives

Phase: shadcn-primitives
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: high
Expected Proof: type-check, build, visual inspection

## Goal
Install all required shadcn components. Customize them to match design tokens (0 radius, hard shadows, blueprint blue accent). Replace all 16 hand-rolled UI primitives with shadcn equivalents. Migrate notification system from hand-rolled zustand store to Sonner.

## Scope Boundary
### Allowed Surfaces
- `web/src/components/ui/` — replace all files with shadcn components
- `web/src/components/common/NotificationContainer.tsx` — remove
- `web/src/components/common/SplashScreen.tsx` — restyle (uses spinner)
- `web/src/components/common/ConfirmationModal.tsx` — migrate to Dialog/AlertDialog
- `web/src/stores/useNotificationStore.ts` — remove
- `web/src/stores/index.ts` — remove useNotificationStore export
- All files that import from `components/ui/` or call `showNotification()` — update imports
- `web/package.json` — new shadcn/Radix dependencies + Sonner

### Forbidden Surfaces
- Page-level layout or structure (no page rebuilds yet)
- Zustand stores other than useNotificationStore
- API layer
- Router
- i18n locale files
- SCSS files (do not delete yet)

## Spec Hooks
- Req 6–19: All component replacements
- Req 20: NotificationContainer → Sonner
- Constraint: i18n keys unchanged (toast messages use same keys)

## Locked Decisions
- Use `bunx shadcn@latest add <component>` for each component
- Customize shadcn components inline by editing the generated source (shadcn is "open code" — you own the files)
- Override default border-radius to 0 in all components
- Map button variants: `primary` → `default`, `secondary` → `secondary`, `ghost` → `ghost`, `danger` → `destructive`
- Sonner replaces the entire notification stack (store + component)
- Keep `loading` prop on Button by extending shadcn Button

## Assumptions
- shadcn component files land in `web/src/components/ui/` (configured in `components.json`)
- Radix UI primitives don't conflict with existing React 19 features
- Sonner works with Vite + single-file build
- Existing consumers import from `@/components/ui/Button` etc. — shadcn components will live at the same paths

## Canonical Refs
- Current hand-rolled components: `web/src/components/ui/*.tsx`
- Current notification store: `web/src/stores/useNotificationStore.ts`
- shadcn component docs: https://ui.shadcn.com/docs/components/
- Sonner docs: https://ui.shadcn.com/docs/components/sonner

## Rejected Options
- Keep hand-rolled components and just restyle — more maintenance, no accessibility improvement
- Use Radix directly without shadcn — shadcn provides styled defaults that match our needs

## Deferred Ideas
- Form validation with React Hook Form + shadcn Field
- Data Table with sorting/filtering
- Command palette

## Escalate If
- shadcn component API is incompatible with >5 consumer sites for a single component → may need adapter wrappers
- Sonner toast positioning conflicts with existing layout → may need z-index or portal adjustments
- Type errors exceed 20 after component swap → stop and reassess import mapping strategy
