# Plan: Dark Mode Removal

Phase: dark-mode-removal
Status: ready
Wave Count: 3
Execution Owner: work
Updated At: 2026-05-28

## Goal
Remove all dark mode artifacts from the codebase. Strip `.dark` CSS block, `@custom-variant dark`, all `dark:` utility classes from components/pages/features, simplify theme store.

## Inputs
- `web/src/index.css` — `.dark {}` block and `@custom-variant dark` directive
- `web/src/stores/useThemeStore.ts` — dark mode toggle logic
- 23 files with `dark:` utility classes (from audit)
- `.kit/planning/SPEC.md` — requirement R12

## Wave 1
### T1 — Remove `.dark` block and `@custom-variant dark` from index.css
- type: refactor
- inputs:
  - `web/src/index.css` (Phase 1 output)
- touches:
  - `web/src/index.css`
- avoid:
  - `:root` block (Phase 1 work)
  - `@theme` block
  - Component files
- steps:
  1. Delete the entire `.dark { ... }` block (~45 lines)
  2. Delete `@custom-variant dark (&:where(.dark, .dark *));` line
- expected outputs:
  - `index.css` with no `.dark` block and no dark custom variant
- verification:
  - `grep -n "\.dark" web/src/index.css` returns zero matches
  - `grep -n "@custom-variant dark" web/src/index.css` returns zero matches
- stop if:
  - N/A (straightforward deletion)
- escalate to:
  - N/A

### T2 — Simplify useThemeStore.ts
- type: refactor
- inputs:
  - `web/src/stores/useThemeStore.ts`
- touches:
  - `web/src/stores/useThemeStore.ts`
- avoid:
  - Other stores
  - API layer
- steps:
  1. Simplify `applyTheme` to always apply light (remove `classList.add('dark')` branch)
  2. Simplify `cycleTheme` to no-op or remove
  3. Keep store shape (theme, resolvedTheme, setTheme) to avoid breaking consumers
  4. Default `theme` and `resolvedTheme` to `'light'` always
- expected outputs:
  - Theme store that always resolves to light
- verification:
  - `grep -n "dark" web/src/stores/useThemeStore.ts` returns zero matches (except possibly type imports)
- stop if:
  - Store consumers use `resolvedTheme` for non-styling logic
- escalate to:
  - user clarification if theme-dependent logic found

## Wave 2
### T3 — Remove `dark:` utility classes from all component, page, and feature files
- type: refactor
- inputs:
  - 23 files with `dark:` utility classes (from audit)
- touches:
  - `web/src/components/common/SecondaryScreenShell.tsx`
  - `web/src/components/config/VisualConfigEditor.tsx`
  - `web/src/components/providers/ProviderStatusBar.tsx`
  - `web/src/components/quota/quotaStyles.ts`
  - `web/src/components/ui/badge.tsx`
  - `web/src/components/ui/checkbox.tsx`
  - `web/src/components/ui/dropdown-menu.tsx`
  - `web/src/components/ui/Input.tsx`
  - `web/src/components/ui/switch.tsx`
  - `web/src/features/authFiles/constants.ts`
  - `web/src/features/authFiles/components/AuthFileCard.tsx`
  - `web/src/features/authFiles/components/AuthFileQuotaSection.tsx`
  - `web/src/features/providers/components/ProviderCategoryList.tsx`
  - `web/src/features/providers/components/ProviderHeaderCard.tsx`
  - `web/src/features/providers/components/ProviderResourcePanel.tsx`
  - `web/src/features/providers/components/ProviderResourceTable.tsx`
  - `web/src/features/providers/sheets/forms/BaseProviderForm.tsx`
  - `web/src/features/providers/sheets/ResourceDetailView.tsx`
  - `web/src/pages/SystemPage.tsx`
  - `web/src/pages/OAuthPage.tsx`
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/ConfigPage.tsx`
  - `web/src/utils/quota/constants.ts`
- avoid:
  - Changing component logic or structure
  - Changing i18n keys
  - Modifying API calls
- steps:
  1. For each file, find all `dark:` prefixed class names in className strings
  2. Remove the `dark:` class (keep the light variant if it exists)
  3. For constant files (constants.ts), remove dark-specific color entries from maps/objects
  4. Distinguish string content "dark" (labels, descriptions) from `dark:` CSS utilities — only remove the latter
- expected outputs:
  - Zero `dark:` utility classes in any `.tsx`/`.ts` file
- verification:
  - `grep -rn "dark:" web/src/components/ web/src/features/ web/src/pages/ --include="*.tsx" --include="*.ts" | grep -v "//.*dark:" | grep -v "'dark'" | grep -v '"dark"'` returns zero matches
- stop if:
  - A component's visual appearance depends critically on `dark:` overrides that have no light equivalent
- escalate to:
  - user clarification for visual regression decisions

### T4 — Remove theme toggle from UI
- type: refactor
- inputs:
  - Theme toggle location (likely MainLayout.tsx or a header component)
- touches:
  - Component containing theme toggle button
- avoid:
  - Changing layout structure beyond removing the toggle
- steps:
  1. Find the theme toggle button/icon (search for `cycleTheme` or theme toggle imports)
  2. Remove the toggle element from the JSX
  3. Remove unused imports
- expected outputs:
  - No theme toggle visible in the UI
- verification:
  - `grep -rn "cycleTheme\|toggleTheme\|themeToggle" web/src/ --include="*.tsx"` returns zero matches (or only the store definition)
- stop if:
  - N/A
- escalate to:
  - N/A

## Wave 3
### T5 — Clean up Theme type
- type: refactor
- inputs:
  - `web/src/types/common.ts` (Theme type)
  - `web/src/stores/useThemeStore.ts`
- touches:
  - `web/src/types/common.ts`
  - Any file importing `Theme` type
- avoid:
  - Removing the type entirely (keep for future dark mode restoration)
- steps:
  1. Simplify `Theme` type to just `'light'` (or keep as `'light' | 'dark'` for future restoration — prefer keeping)
  2. Verify no runtime logic branches on theme value for non-styling purposes
- expected outputs:
  - Clean theme type, no dark-specific logic
- verification:
  - `cd web && bun run build` succeeds
  - `grep -rn "=== .dark.\|=== \"dark\"\|== .dark." web/src/ --include="*.tsx" --include="*.ts"` returns zero matches (excluding store)
- stop if:
  - Theme type is used in API calls or backend communication
- escalate to:
  - user clarification

## Risks / Watch-fors
- `constants.ts` files may have color maps keyed by theme — need to flatten to light-only
- Some `dark:` classes may be the ONLY styling (no light counterpart) — these need replacement, not just removal
- `useThemeStore` consumers that destructure `cycleTheme` will get a no-op, which is fine
