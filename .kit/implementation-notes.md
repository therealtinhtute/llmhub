# Implementation Notes

Phase: module-rebrand
Started: 2026-05-27

## Wave 1 — T2 extended
- Found 2 non-import `router-for-me/CLIProxyAPI` string references beyond bulk sed scope: release URL in `config_basic.go` and comment URL in `openai_responses_websocket.go`. Updated both to `therealtinhtute/llmhub`.
- Also caught and renamed `latestReleaseUserAgent` from `CLIProxyAPI` to `LLMHub` in `config_basic.go`.

## Wave 2 — T3/T4 extended
- `PanelGitHubRepository` had references in 5 additional files beyond the plan's list: `parse.go`, `config_diff.go`, `config_diff_test.go`, `server.go`, `sdk/config/config.go`. All cleaned up.
- Identity renames extended beyond plan's 3 files: also updated TUI i18n strings, gitstore git author, home/certificate.go auth dir path, and package doc comments.
- **Deliberately kept as-is** (7 references): auth header values (`xai.go` referrer, `kimi.go` X-Msh-Platform), cache key prefix (`codex_executor.go`), device identifiers (`kimi_executor.go`). These are sent to external APIs — changing them risks breaking authentication flows or invalidating caches.
- `sdk/config/config.go` re-export of `DefaultPanelGitHubRepository` removed.

---

Phase: embed-panel
Started: 2026-05-27

## Wave 2 — T3 embed strategy
- `internal/managementasset/static/` added to `.gitignore` as a generated artifact (rebuilt from `web/` source).
- The `static/management.html` file must exist at `go build` time — CI and goreleaser hooks handle this.

## Wave 3 — T4 updater.go gutting
- Gutted entire `updater.go` to just 2 constant declarations (`managementAssetName`, `ManagementFileName`).
- Removed `SetCurrentConfig` from `server.go` (2 call sites) and `main.go` (1 call site) — no longer needed since handler reads `s.cfg` directly and doesn't check config for download decisions.
- Removed `managementasset` import from `main.go` entirely.

## Wave 4 — Dockerfile multi-stage
- Added `oven/bun:1` as a frontend build stage to Dockerfile. The `COPY --from=frontend` step copies the built HTML into the Go build context before `go build`.
- Also updated `.gitignore` binary name from `cli-proxy-api` to `llmhub` (missed in module-rebrand).

---

Phase: doc-cleanup
Started: 2026-05-27

## Wave 1 — README rebrand approach
- Full rewrite of all three READMEs rather than surgical edits. The sponsor sections were deeply interleaved with branding, making surgical edits error-prone.
- "Getting Started" links to `help.router-for.me` replaced with local config reference note ("Documentation is being migrated"). The external doc site is tied to the old project.
- Third-party project names containing "CLIProxyAPI" (Tray, Dashboard, Quota Inspector) deliberately kept — these are external GitHub repo names we don't control.
- Telegram group URL `t.me/CLIProxyAPI` kept in CN README — it's an external resource.
- Sections renamed: "Who is with us?" → "Community Projects", "谁与我们在一起？" → "社区项目", "関連プロジェクト" → "コミュニティプロジェクト".

## Wave 2 — config.example.yaml
- Removed `panel-github-repository` and `disable-auto-update-panel` entries entirely since the panel is now embedded (no GitHub download needed).
- Updated `disable-control-panel` comment from "asset download" to "bundled management control panel".

## Wave 2 — .github/ CI files (extended scope)
- Extended T4 beyond the plan's `assets/` cleanup to also fix `.github/` CI files. These had `router-for-me/models.git` references in 3 workflow files and old branding in FUNDING.yml and docker-image.yml.
- DOCKERHUB_REPO changed from `eceasy/cli-proxy-api` to `eceasy/llmhub` — matches the docker-compose.yml change from embed-panel phase.

---

Phase: tailwind-foundation
Started: 2026-05-28

## T2 — shadcn init: manual instead of CLI
- Created `components.json` and `src/lib/utils.ts` manually instead of `bunx shadcn@latest init`.
- **Why**: CLI is interactive and may scaffold files conflicting with existing `src/` structure. Manual creation is deterministic.
- **Tradeoff**: If shadcn CLI adds new required files in future versions, they won't be present. Low risk — schema is stable.

## T1 — index.css: replaced Vite scaffold default
- Overwrote existing `src/index.css` (Vite boilerplate with default styles) with Tailwind import.
- **Why**: File was unused — `main.tsx` imported `global.scss` not `index.css`. Boilerplate styles would conflict with design tokens.

## T3 — Dark theme selector: `.dark` class vs `[data-theme='dark']`
- Used `.dark` class selector (shadcn convention) for dark theme in `index.css`.
- **Why**: shadcn components expect `.dark` on `<html>`. Existing SCSS uses `[data-theme='dark']`.
- **Tradeoff**: `useThemeStore` will need to apply `.dark` class in addition to or instead of `data-theme` — addressed in Phase 3 (layout-shell).

---

Phase: shadcn-primitives
Started: 2026-05-28

## T4 — Compat wrapper approach for consumer migration
- Created `Legacy*.tsx` thin wrappers that preserve old component APIs while delegating to shadcn internals.
- 11 wrappers: LegacyModal, LegacySelect, LegacySheet, LegacyInput, LegacyTable, LegacySkeleton, LegacyCollapsible, LegacyToggleSwitch, LegacySelectionCheckbox, LegacyEmptyState, LegacyLoadingSpinner.
- **Why**: User chose this over direct consumer updates. Avoids touching 20+ consumer files' internal logic during component installation phase. Consumers keep their existing API contracts.
- **Tradeoff**: Extra indirection layer; Legacy files must be removed in Phase 4 when consumers adopt shadcn APIs directly.

## T4 — LegacySheet: confirmClose is async
- `LegacySheet.confirmClose` returns `Promise<boolean>`. The shadcn `Sheet` `onOpenChange` is synchronous.
- Handled by making `handleOpenChange` async — Radix Dialog fires `onOpenChange` but doesn't gate on its return value, so we prevent close via `onPointerDownOutside`/`onEscapeKeyDown` prevention when `closeDisabled` is true, and run the async guard ourselves.

## T5 — Notification migration: kept ConfirmationModal as-is
- Replaced `showNotification()` → Sonner `toast()` across all consumers.
- Kept `showConfirmation()` + `ConfirmationModal` + `useNotificationStore` confirmation state as-is.
- **Why**: `showConfirmation` is a complex pattern (Zustand state → global modal with async onConfirm/onCancel callbacks). Migrating to AlertDialog API would require changing 8+ call sites' control flow. Deferred to Phase 4.
- **Tradeoff**: `useNotificationStore` still exists but only for confirmation state. `NotificationContainer` replaced by Sonner `<Toaster />`.

---

Phase: layout-shell
Started: 2026-05-28

## T1 — Theme class strategy: `.dark` class not `data-theme`
- CONTEXT.md stated `data-theme` attribute; PLAN.md step 4 explicitly said to use `.dark` class.
- Followed PLAN (more specific, aligns with `index.css` already using `.dark` selector from Phase 1).
- `useThemeStore.applyTheme` now toggles `document.documentElement.classList` instead of `setAttribute`.

## T1 — resolvedTheme kept in useThemeStore
- Phase 4-scope pages (SystemPage, OAuthPage, AuthFilesPage, ConfigPage, QuotaSection, ModelMappingDiagram) read `state.resolvedTheme` from the store.
- Kept `resolvedTheme` as a direct alias for `theme` (they're identical now that auto/white are removed).
- **Why**: Touching those files is out of scope for Phase 3. No functional difference since `Theme` is now `'light' | 'dark'` = `ResolvedTheme`.

## T2 — Lowercase/uppercase button.tsx + input.tsx conflict
- shadcn sidebar install created `button.tsx` and `input.tsx` (lowercase) despite customized `Button.tsx` and `Input.tsx` (uppercase) already existing.
- Deleted lowercase duplicates and updated `sidebar.tsx` imports to uppercase.
- **Why**: TypeScript TS1149 casing conflict on Linux case-sensitive filesystem. Customized uppercase versions are the correct canonical files.

## T2 — LegacySelect.tsx casing pre-existing bug
- `LegacySelect.tsx` imported from `'./select'` (lowercase) but only `Select.tsx` (uppercase) exists.
- Fixed inline (out of strict Phase 2 scope, but required for type-check to pass).

## T2 — shadcn Sidebar internal Math.random lint error
- `sidebar.tsx` `SidebarMenuSkeleton` uses `Math.random()` in `useMemo` — triggers `react-hooks/purity` lint rule.
- Suppressed with `// eslint-disable-next-line` on the specific line. Shadcn-generated code, not changed further.

## T2 — MainLayout content padding: removed 70px top offset
- Original `.main-content` had `padding-top: 70px` to clear a `position: fixed` header.
- New layout: header is part of normal flow inside SidebarInset (not fixed), so 70px top offset no longer needed.
- Replaced with `pt-6` (24px). Pages use their own SCSS modules for internal padding — not affected.
