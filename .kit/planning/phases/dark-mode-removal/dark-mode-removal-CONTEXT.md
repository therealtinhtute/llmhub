# Context: Dark Mode Removal

Phase: dark-mode-removal
Status: ready
Spec Link: ../../SPEC.md
Roadmap Link: ../../ROADMAP.md
Blast Radius: medium
Expected Proof: build, grep

## Goal
Remove all dark mode artifacts from the codebase: CSS `.dark` block, `@custom-variant dark` directive, all `dark:` utility classes from components/pages/features, and simplify or remove the theme store.

## Scope Boundary
### Allowed Surfaces
- `web/src/index.css` — remove `.dark {}` block and `@custom-variant dark`
- `web/src/stores/useThemeStore.ts` — simplify (always light) or delete
- `web/src/components/**/*.tsx` — remove `dark:` prefixed classes
- `web/src/features/**/*.tsx` — remove `dark:` prefixed classes
- `web/src/pages/**/*.tsx` — remove `dark:` prefixed classes
- `web/src/features/**/constants.ts` — remove dark-specific color mappings
- `web/src/utils/quota/constants.ts` — remove dark-specific entries
- `web/src/types/common.ts` — simplify `Theme` type if needed

### Forbidden Surfaces
- Token definitions in `:root` (Phase 1's work)
- API layer, stores (other than theme store)
- Routing, i18n
- Animation/motion logic

## Spec Hooks
- R12: Dark mode removal (items 30-34)
- Key Decision 3: Light-only — new token source only defines light palette

## Locked Decisions
- Theme store simplified to always-light (not deleted — preserves the store API shape for future dark mode restoration)
- `dark:` classes removed entirely, not replaced with light-specific alternatives (the base styles already handle light)
- `@custom-variant dark` removed (no dark variant needed in Tailwind)
- String content containing "dark" (labels, descriptions) is NOT touched — only `dark:` CSS utility classes

## Assumptions
- Phase 1 tokens produce a correct light theme, so removing dark has no visual regression
- No component logic depends on `resolvedTheme === 'dark'` for non-styling behavior (need to verify)
- Theme toggle UI is in MainLayout or a header component

## Canonical Refs
- `web/src/index.css` lines 115-160 — `.dark {}` block
- `web/src/index.css` line 4 — `@custom-variant dark`
- `web/src/stores/useThemeStore.ts` — theme toggle logic
- 23 files with `dark:` utility classes (from grep audit)

## Rejected Options
- Keeping `.dark` block "just in case" — rejected; dead code that confuses future maintainers. Deferred Ideas captures future dark mode as a separate token definition effort.
- Deleting theme store entirely — rejected; simpler to keep as always-light than to find/remove all consumers

## Deferred Ideas
- Dark mode restoration (needs separate dark token definition)
- System preference detection (prefers-color-scheme)

## Escalate If
- Component logic branches on `resolvedTheme === 'dark'` for non-styling behavior
- Theme store has more consumers than expected
