---
id: 01KZB88NS7DXSHCAVWR3ZM77JQ
type: plan
intake_id: 01KZB88KH357MGHT267KSNHBB3
lane: normal
status: active
created: 2026-08-06
updated: 2026-08-06
---

# Unified provider console — merge AI Providers + OAuth Login

Status: **locked** · Created 2026-08-06 · Ships independently
Source of the shape: `decolua/9router` — single provider registry with a `category` field
(`oauth | apikey | freeTier | free`) driving UI grouping
Reference skill: `.claude/skills/9router-port/references/providers.md` §"Categories 9router uses"

## Outcome

Collapse `/ai-providers` and `/oauth` into one screen at `/ai-providers`: a 240px left rail whose
rows are grouped into 9router-style category sections, and a right panel that renders different
content per entry kind. OAuth providers, preset providers (OpenRouter, NVIDIA, OpenCode) and
API-key provider brands all sit as **peer rows in the same rail**. Frontend only — no Go change.

Success signals:
- `/oauth` is gone from the sidebar and redirects to `/ai-providers?entry=oauth:anthropic`; the
  sidebar `gateway` group drops from 5 items to 4.
- The rail renders 15 rows across 5 sections: `OAUTH LOGIN` (6), `FREE` (1), `FREE TIER` (2),
  `API KEY` (4), `CUSTOM` (2).
- Selecting an OAuth row shows the login button plus a table of that provider's auth files;
  starting a login, copying the URL, pasting a callback, and the success/error badges all behave
  exactly as they do on today's `/oauth` page.
- Selecting `OpenRouter` shows only the `openai-compatibility` entries whose `base_url` matches the
  preset, with a prefilled Add button; entries whose `base_url` matches no preset appear under
  `CUSTOM`, and the two sets partition the group with no duplicates and no drops.
- `cd web && bun run type-check && bun run lint && bun run build` all clean.
- `make build-web && make build` succeeds.
- No new test files created anywhere under `web/` (project rule, `CLAUDE.md`).

## Authority and Requirements

Authority:
- Owner decisions taken 2026-08-06 in the `/think` session that locked this initiative — four
  explicit answers, recorded verbatim in `## Decisions`. These are the binding scope authority for
  D1-D4 below.
- `.claude/skills/9router-port/references/providers.md` §"Categories 9router uses" — external
  reference project, cited for the category taxonomy only, not as a spec to copy verbatim (the
  card-grid presentation is explicitly rejected, see `NG3`).
- Direct reads of the current panel source, 2026-08-06 — authority for "frontend only, no Go
  change": `providersApi.getProviderPresets()` already exists
  (`web/src/services/api/providers.ts:592`) and is already served by
  `GET /v0/management/provider-presets`
  (`internal/api/handlers/management/provider_presets.go:13`); `useAuthFilesData()` already exposes
  `files`, `loadFiles`, `handleStatusToggle`, `handleDelete`, `handleDownload`
  (`web/src/features/authFiles/hooks/useAuthFilesData.ts:27-53`); `AuthFileItem.type` already
  carries the provider discriminator (`web/src/types/authFile.ts:8-24`).
- `CLAUDE.md` project rule "Do not create new frontend test files anywhere under `web/`" —
  authority for the verification style in `## Verify`.

Owner decisions:
- `D1` — Merge two screens only. The OAuth panel embeds the account list; deep management
  (batch upload, model alias, excluded models, kiro quota) keeps linking out to `/auth-files`.
- `D2` — Preset entries follow the catalog model: always visible in the rail even with zero
  configured entries, exactly like today's OAuth cards.
- `D3` — OpenRouter and NVIDIA are separate top-level rows, 9router-style.
- `D4` — Keep the 240px left rail + right panel layout, grouped by category.

Requirements:
1. `R1` — New `web/src/features/providers/entries.ts` exporting a `ProviderEntry` discriminated
   union (`kind: 'oauth' | 'config' | 'preset'`), a `ProviderEntryCategory` type
   (`'oauth' | 'free' | 'freeTier' | 'apikey' | 'custom'`), and `buildEntries()` producing the
   ordered, sectioned rail model from workbench groups + the OAuth provider list + presets + auth
   files. | source: `D4`, `## Changes` §1
2. `R2` — `ProviderCategoryList` renders section headers and rows from `buildEntries()` output
   instead of iterating `ProviderGroup[]`; the rail is height-capped with vertical overflow so 15
   rows do not push the panel off-screen. | source: `D4`, `## Changes` §2
3. `R3` — New `web/src/features/providers/panels/OAuthLoginPanel.tsx` carrying the full state logic
   lifted from `web/src/pages/OAuthPage.tsx` verbatim — `startAuth`, `startPolling`,
   `completeProviderAuth`, `resetProviderAttempt`, `submitCallback`, the gemini-cli project-id
   field, and the xai callback parsing helpers (`buildXaiCallbackUrl`,
   `readQueryLikeCallbackInput`, `extractDisplayedXaiCode`, `resolveCallbackUrl`). No behavior
   change to the flow. | source: `D1`, `## Changes` §3
4. `R4` — New `web/src/features/providers/panels/AuthFileMiniTable.tsx`: the auth files belonging to
   the selected OAuth provider, reusing `useAuthFilesData()`, with enable/disable, download, delete,
   and a link to `/auth-files` for everything else. Mapping is
   `anthropic→claude`, `gemini-cli→gemini-cli`, all others identity, applied against
   `AuthFileItem.type`. | source: `D1`, `## Changes` §4
5. `R5` — `web/src/pages/OAuthPage.tsx` is removed; `/oauth` becomes
   `<Navigate to="/ai-providers?entry=oauth:anthropic" replace />`; the `/oauth` sidebar item is
   removed from the `gateway` nav group (`web/src/components/layout/MainLayout.tsx:222`);
   `ProvidersWorkbenchPage` reads and writes the selected entry as an `?entry=` search param so the
   redirect lands on the right row and the selection is linkable. | source: `D1`, `## Changes` §5
6. `R6` — Preset rows in the rail, one per catalog entry, always rendered regardless of whether any
   matching config exists. Category is derived from existing preset JSON fields with no schema
   change: `defaultApiKey` non-empty and `signupUrl` empty → `free`; `signupUrl` and `freeTierNote`
   both non-empty → `freeTier`; otherwise `apikey`. | source: `D2`, `D3`, `## Changes` §6
7. `R7` — `openaiCompatibility` resources partition by normalized base URL — lowercase, trailing
   slash stripped. A resource matching a preset's `baseUrl` belongs to that preset's row; every
   other resource belongs to the `CUSTOM` → `OpenAI Compatible` row. The partition is total and
   disjoint. | source: `D3`, `## Changes` §7
8. `R8` — New `web/src/features/providers/panels/PresetProviderPanel.tsx`: preset header (base URL,
   free-tier note, verified/unverified badge, signup link), the existing `ProviderResourceTable`
   over the matched resources, and an Add button that opens the create sheet with that preset
   preselected. `ProviderSheet` and `BaseProviderForm` accept a new optional `initialPresetId`
   prop consumed by the preset picker landed in `1abe73c9`. | source: `D2`, `D3`, `## Changes` §8
9. `R9` — Every new user-visible string added to both `web/src/i18n/locales/en.json` and
   `web/src/i18n/locales/vi.json`. No hardcoded English in JSX. | source: `## Changes` §9

## Non-goals

- `NG1` — `/auth-files` and its two sub-routes (`/auth-files/oauth-excluded`,
  `/auth-files/oauth-model-alias`) are not merged, not moved, and not modified. Owner chose the
  two-screen merge; absorbing the 890-line page was the rejected option. | source: `D1`
- `NG2` — Auth-file-only providers (`kiro`, `qwen`, `iflow`, `aistudio`, and file-based `vertex`)
  get no rail row. They have no OAuth start endpoint on `/oauth` today and no
  `openai-compatibility` config; they stay visible on `/auth-files`. | source: `D1`
- `NG3` — No flat card-grid layout. 9router's card grid would drop the resource table's sort,
  model filter, usage stats, and row actions — a real feature loss against the owner's stated
  "vẫn đảm bảo full tính năng". | source: `D4`
- `NG4` — No Go change, no new backend endpoint, no config schema change. Both APIs this plan needs
  already exist and are already wired. | source: `## Authority`
- `NG5` — No new entries added to `internal/config/presets/providers.json`. The catalog stays at
  the three seeded providers. | source: `D3`
- `NG6` — No `preset_id` field on `OpenAICompatibility`. Matching is by base URL only. This keeps
  the change frontend-only and avoids adding a feature field to the config YAML. The known
  consequence is accepted and documented: editing an entry's `base_url` by hand moves it from its
  preset row to `CUSTOM`. | source: `D3`, `## Risk`
- `NG7` — `model-combos` is untouched. The owner explicitly left that phase pending. | source:
  owner instruction 2026-08-06 ("pending cái model-combos")

## Building

A rail whose rows come from one view model instead of one data source:

```
buildEntries()
   ├── OAUTH LOGIN   codex · anthropic · antigravity · gemini-cli · kimi · xai
   │                 from OAuthPage's PROVIDERS list + auth files grouped by AuthFileItem.type
   ├── FREE          OpenCode Free
   ├── FREE TIER     OpenRouter · NVIDIA NIM
   │                 from GET /v0/management/provider-presets, category derived from JSON fields
   ├── API KEY       Gemini · Codex · Claude · Vertex
   │                 from the existing workbench ProviderGroup[]
   └── CUSTOM        OpenAI Compatible (unmatched entries) · Ampcode
```

`codex` appears twice on purpose — once as an OAuth login, once as an API-key brand. That duality
already exists today across the two screens and matches 9router, which also lists a vendor under
more than one category when it supports more than one auth method.

Panel content by entry kind:

| kind | panel |
|---|---|
| `oauth` | `OAuthLoginPanel` — login button, auth URL box, callback input, status badge, plus `AuthFileMiniTable` for that provider |
| `preset` | `PresetProviderPanel` — preset header + `ProviderResourceTable` over matched entries + prefilled Add |
| `config` | today's `ProviderResourcePanel`, unchanged |

## Not building

| Excluded | Why |
|---|---|
| Absorbing `/auth-files` | `NG1` — owner picked the two-screen merge |
| Rail rows for kiro/qwen/iflow/aistudio | `NG2` — no OAuth start, no config block |
| Card grid | `NG3` — loses table affordances |
| `preset_id` config field | `NG6` — would require a Go + YAML change for a matching edge case |
| New presets | `NG5` |

## Changes

### 1. Entry view model — `web/src/features/providers/entries.ts` (new)

```ts
export type ProviderEntryCategory = 'oauth' | 'free' | 'freeTier' | 'apikey' | 'custom';

export type ProviderEntry =
  | { kind: 'oauth';  key: string; category: 'oauth'; oauthId: OAuthProvider;
      titleKey: string; hintKey: string; urlLabelKey: string;
      icon: string | { light: string; dark: string }; accountCount: number }
  | { kind: 'preset'; key: string; category: 'free' | 'freeTier' | 'apikey';
      preset: ProviderPreset; resources: ProviderResource[] }
  | { kind: 'config'; key: string; category: 'apikey' | 'custom';
      group: ProviderGroup; resources: ProviderResource[] };
```

`key` is the stable `?entry=` value: `oauth:anthropic`, `preset:openrouter`, `config:gemini`.
`buildEntries({ groups, presets, authFiles })` returns `ProviderEntry[]` in rail order. Section
order is fixed: `oauth`, `free`, `freeTier`, `apikey`, `custom`.

Category derivation for presets (no schema change, `R6`):

```ts
const presetCategory = (p: ProviderPreset): 'free' | 'freeTier' | 'apikey' =>
  p.defaultApiKey && !p.signupUrl ? 'free'
  : p.signupUrl && p.freeTierNote ? 'freeTier'
  : 'apikey';
```

Against the current catalog this yields `opencode → free`, `openrouter → freeTier`,
`nvidia → freeTier`.

Base-URL normalization and partition (`R7`):

```ts
const normalizeBaseUrl = (u: string | null | undefined) =>
  (u ?? '').trim().toLowerCase().replace(/\/+$/, '');
```

### 2. Rail — `components/ProviderCategoryList.tsx`

Takes `entries: ProviderEntry[]` and `activeKey: string` instead of `groups`/`activeBrand`. Groups
consecutive entries by `category`, emits an uppercase section label per category, keeps the exact
row markup, logo treatment, count badge, and issue-triangle behavior in place today. Adds
`max-h-[calc(100vh-220px)] overflow-y-auto` to the `<aside>`.

Logos: OAuth rows reuse the SVGs already imported by `OAuthPage.tsx:12-19`; preset rows fall back
to the existing `openai-light.svg` used for `openaiCompatibility`.

### 3. OAuth panel — `panels/OAuthLoginPanel.tsx` (new)

The whole of `OAuthPage.tsx` minus its page chrome and its 4-column grid: `ProviderState`,
`PROVIDERS`, `CALLBACK_SUPPORTED`, `XAI_CALLBACK_URL`, `SUCCESS_RESET_DELAY_MS`,
`getProviderI18nPrefix`, `getAuthKey`, `getIcon`, `isAbsoluteUrl`, `readQueryLikeCallbackInput`,
`extractDisplayedXaiCode`, `buildXaiCallbackUrl`, `resolveCallbackUrl`, and every handler. The
component renders a single provider instead of six; timer refs stay keyed by provider so a login
started on one row survives switching rows and back. `PROVIDERS` and `CALLBACK_SUPPORTED` move out
into `entries.ts` so `buildEntries` and the panel share one list.

### 4. Account table — `panels/AuthFileMiniTable.tsx` (new)

Consumes `useAuthFilesData()`. Filters by

```ts
const OAUTH_TO_AUTH_FILE_TYPE: Record<OAuthProvider, string> = {
  codex: 'codex', anthropic: 'claude', antigravity: 'antigravity',
  'gemini-cli': 'gemini-cli', kimi: 'kimi', xai: 'xai',
};
```

Columns: name, status, last refresh, actions (enable/disable via `handleStatusToggle`, download via
`handleDownload`, delete via `handleDelete`). Footer links to `/auth-files` for batch upload, model
alias, excluded models, and kiro quota. Files whose `type` is absent or `unknown` are not shown here
and remain reachable on `/auth-files` — the footer states this.

### 5. Route and nav consolidation

- `web/src/router/MainRoutes.tsx:26` — `/oauth` element becomes
  `<Navigate to="/ai-providers?entry=oauth:anthropic" replace />`.
- `web/src/components/layout/MainLayout.tsx:222` — drop the `/oauth` nav item.
- `web/src/pages/OAuthPage.tsx` — removed with `trash` after extraction.
- `ProvidersWorkbenchPage` — `?entry=` search param is the source of truth for selection, defaulting
  to the first `config` entry (`config:gemini`) when absent or unknown. The existing
  `confirmDiscardIfDirty` guard on row switching is preserved.

### 6. Preset rows

`ProvidersWorkbenchPage` fetches `providersApi.getProviderPresets()` alongside the workbench
snapshot and feeds it to `buildEntries`. A failed preset fetch degrades to zero preset rows and
leaves every `openai-compatibility` entry under `CUSTOM` — the rail still renders.

### 7. Resource partition

`buildEntries` walks the `openaiCompatibility` group once, bucketing each resource by
`normalizeBaseUrl(resource.baseUrl)` against a `Map<string, presetId>` built from the catalog.
Unmatched resources form the `CUSTOM` → `OpenAI Compatible` row's resources.

### 8. Preset panel — `panels/PresetProviderPanel.tsx` (new)

Header: display name, base URL, free-tier note, signup link with `IconExternalLink`, and the
unverified warning already styled in `BaseProviderForm` (`87f5c20d`). Body: the existing
`ProviderResourceTable` with the same props the OpenAI-compatibility panel passes today, so sort,
model filter, usage stats, and row actions carry over unchanged. Add button opens the create sheet
with `initialPresetId` set.

### 9. i18n

New keys under `providersPage.categories.sections.*` (5 section labels),
`providersPage.presets.*` (header labels, verified/unverified, signup, empty state), and
`providersPage.oauthAccounts.*` (mini-table headers, footer link). Existing `auth_login.*` keys are
reused verbatim by `OAuthLoginPanel` — no renames.

## Verify

Per phase, in order:

- `cd web && bun run type-check` — clean.
- `cd web && bun run lint` — clean.
- `cd web && bun run build` — `tsc && vite build` succeeds.
- `make build-web && make build` — the embedded panel builds and the Go binary compiles.
- Browser runtime check against the already-running server via `make dev-web` (proxies to
  `DEV_WEB_API_BASE`, default `http://localhost:9090`; does not restart the backend):
  - rail shows 5 sections and 15 rows;
  - `/oauth` redirects to `/ai-providers?entry=oauth:anthropic` and that row is selected;
  - selecting each of the 6 OAuth rows renders the login button and the account table without a
    console error;
  - selecting `OpenRouter` shows only base-URL-matching entries; the count of preset-row resources
    plus the `CUSTOM` row's resources equals the total `openai-compatibility` resource count;
  - browser console is free of React errors and warnings across the sweep.
- No new files under `web/` matching `*.test.*` or `*.spec.*` (`CLAUDE.md` rule).

## Rollback

Pure frontend plus two route/nav edits. `git revert` of the phase commit restores both screens
byte-for-byte; `OAuthPage.tsx` comes back with the revert. No database change, no config change, no
migration, no external state touched. Phase 2 reverts independently of Phase 1 — removing preset
rows returns every `openai-compatibility` entry to the single `CUSTOM` row.

## Risk

- **Base-URL matching is brittle to hand edits** (`NG6`). An owner who edits an entry's `base_url`
  away from the preset value sees it move to `CUSTOM`. Accepted and self-explanatory; the escape
  hatch is documented in the preset panel's empty state. Escalation: add `preset_id` in a follow-up
  if this bites in practice.
- **Auth files with missing or `unknown` `type` are invisible in the OAuth panel.** Mitigated by
  the mini-table footer stating that `/auth-files` is the complete list, and by never making the
  panel the only path to a credential.
- **A 15-row rail overflows short viewports.** Mitigated by the height cap and scroll in `R2`.
- **`OAuthPage` logic drifting during extraction.** Mitigated by lifting the module wholesale rather
  than rewriting it, and by the per-row runtime sweep in `## Verify`.

## Approach and Risks

Chosen approach: exactly `## Changes` §1-9, split into two phases. Phase 1 replaces the rail's data
source with a `ProviderEntry` view model and folds `/oauth` in; Phase 2 adds preset rows on top of
that same view model. Both are frontend-only and ride APIs that already exist and are already
wired — `providersApi.getProviderPresets()` and `useAuthFilesData()` — so no Go, no endpoint, and
no config schema work appears anywhere in this plan.

The ordering is forced by a real dependency, not by size: preset rows are entries in the same rail
model that Phase 1 introduces. Building presets first would mean inventing that model twice.

Rejected alternative: port 9router's flat card grid instead of keeping the rail. Rejected because
the owner's requirement was "vẫn đảm bảo full tính năng" and the grid has nowhere to put the
resource table's sort control, model filter, usage stats, or per-row actions — it would ship the
look and drop the function. The rail keeps `ProviderResourceTable` untouched, which is why this
plan can claim zero feature loss rather than merely intending it.

Also rejected: absorbing `/auth-files` into the same screen (the truest 9router shape). Rejected by
owner decision `D1` — the 890-line page carries batch upload, model alias, excluded models, and
kiro quota across two sub-routes, and re-homing all of it is where feature loss would actually
come from. The mini-table plus a link gives the OAuth panel real content at a fraction of the risk.

Primary risks:
- **Extraction drift in the OAuth flow** — the login/poll/callback state machine is the only
  genuinely stateful thing moving. Mitigated by lifting `OAuthPage.tsx` wholesale (no rewrite) and
  by a per-provider runtime sweep in Wave 3's check rather than a compile-only check.
- **Silent resource loss in the partition** — a bug in `normalizeBaseUrl` could drop entries from
  both the preset row and `CUSTOM`. Mitigated by making the count identity
  (`sum(preset rows) + custom row == total`) an explicit check in Phase 2, not an assumption.
- **Preset fetch failure breaking the rail** — mitigated by the documented degradation in
  `## Changes` §6: zero preset rows, everything under `CUSTOM`, rail still renders.

Recovery: each phase is a single frontend commit revertible on its own (`## Rollback`). If Phase 1's
runtime sweep finds the OAuth flow broken in a way the extraction cannot explain, stop and restore
`/oauth` as a standalone route — the rail work is still shippable without the OAuth section, since
`buildEntries` degrades to exactly today's `config`-only rail.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned

### Phase: provider-console-merge
- story_id: `01KZB8F85NSXWGJNK87BTGWQ98`
- goal: Replace the provider rail's data source with a sectioned `ProviderEntry` view model and fold
  `/oauth` into `/ai-providers` as an `OAUTH LOGIN` section, with each OAuth row showing its login
  flow plus its auth files.
- depends_on: none
- touched_surfaces: `web/src/features/providers/entries.ts` (new),
  `web/src/features/providers/panels/OAuthLoginPanel.tsx` (new),
  `web/src/features/providers/panels/AuthFileMiniTable.tsx` (new),
  `web/src/features/providers/components/ProviderCategoryList.tsx`,
  `web/src/features/providers/ProvidersWorkbenchPage.tsx`,
  `web/src/features/providers/types.ts`, `web/src/router/MainRoutes.tsx`,
  `web/src/components/layout/MainLayout.tsx`, `web/src/i18n/locales/en.json`,
  `web/src/i18n/locales/vi.json`, `web/src/pages/OAuthPage.tsx` (removed)
- avoided_surfaces: no Go change (`NG4`); no change to `ProviderResourceTable`,
  `ProviderResourcePanel`, `OpenAIBrandToolbar`, `ProviderSheet`, or `useProviderWorkbench`; no
  change to `/auth-files` or its sub-routes (`NG1`); no new test files under `web/`
- lifecycle status: planned

#### Wave 1 — Entry view model (R1, depends on none)
- task 1.1: Create `web/src/features/providers/entries.ts` with `ProviderEntryCategory`, the
  `ProviderEntry` union, `PROVIDERS` and `CALLBACK_SUPPORTED` moved out of `OAuthPage.tsx`,
  `OAUTH_TO_AUTH_FILE_TYPE`, `normalizeBaseUrl`, and `buildEntries({ groups, authFiles })`
  returning `config` and `oauth` entries in fixed section order. Preset entries are not built yet.
  - touched: `web/src/features/providers/entries.ts` (new),
    `web/src/features/providers/types.ts`
  - check: `cd web && bun run type-check` clean
- task 1.2: Section ordering and `key` stability — `buildEntries` emits categories in the order
  `oauth, free, freeTier, apikey, custom`; `key` values are `oauth:{id}` and `config:{brand}`;
  `ampcode` lands in `custom` after `openaiCompatibility`.
  - touched: `web/src/features/providers/entries.ts`
  - check: `cd web && bun run type-check && bun run lint` clean; confirmed visually in Wave 4's
    runtime sweep (rail order matches `## Building`)

#### Wave 2 — Rail rendering (R2, depends on Wave 1)
- task 2.1: `ProviderCategoryList` switches its props to `entries: ProviderEntry[]` /
  `activeKey: string` / `onSelect(key)`, renders an uppercase section label per category, and keeps
  the existing row markup, logos, count badge, and issue triangle unchanged.
  - touched: `web/src/features/providers/components/ProviderCategoryList.tsx`
  - check: `cd web && bun run type-check && bun run lint` clean
- task 2.2: Height cap — `max-h-[calc(100vh-220px)] overflow-y-auto` on the rail `<aside>`.
  - touched: `web/src/features/providers/components/ProviderCategoryList.tsx`
  - check: runtime sweep in Wave 4 — all 5 section labels reachable by scrolling at a 900px-tall
    viewport, panel not pushed off-screen

#### Wave 3 — OAuth panel and account table (R3, R4, depends on Wave 1)
- task 3.1: Create `panels/OAuthLoginPanel.tsx` by lifting the state logic of `OAuthPage.tsx`
  wholesale (`ProviderState`, `startAuth`, `startPolling`, `completeProviderAuth`,
  `resetProviderAttempt`, `clearProviderTimers`, `submitCallback`, `copyLink`, the gemini-cli
  project-id field, and the xai helpers `isAbsoluteUrl`, `readQueryLikeCallbackInput`,
  `extractDisplayedXaiCode`, `buildXaiCallbackUrl`, `resolveCallbackUrl`), rendering one provider
  instead of a six-card grid. Timer refs stay keyed by provider id.
  - touched: `web/src/features/providers/panels/OAuthLoginPanel.tsx` (new)
  - check: `cd web && bun run type-check && bun run lint` clean
- task 3.2: Create `panels/AuthFileMiniTable.tsx` consuming `useAuthFilesData()`, filtered by
  `OAUTH_TO_AUTH_FILE_TYPE`, with enable/disable, download, delete, and a footer link to
  `/auth-files` stating that files with absent or `unknown` type are listed only there.
  - touched: `web/src/features/providers/panels/AuthFileMiniTable.tsx` (new)
  - check: `cd web && bun run type-check && bun run lint` clean
- task 3.3: `ProvidersWorkbenchPage` renders `OAuthLoginPanel` + `AuthFileMiniTable` for
  `kind === 'oauth'` and today's `ProviderResourcePanel` for `kind === 'config'`, preserving the
  `confirmDiscardIfDirty` guard when switching rows.
  - touched: `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - check: `cd web && bun run build` succeeds

#### Wave 4 — Route/nav consolidation and runtime sweep (R5, R9, depends on Waves 2 and 3)
- task 4.1: `?entry=` search param drives selection in `ProvidersWorkbenchPage`, defaulting to
  `config:gemini` when absent or unknown; `/oauth` route becomes
  `<Navigate to="/ai-providers?entry=oauth:anthropic" replace />`; the `/oauth` sidebar item is
  removed from the `gateway` nav group.
  - touched: `web/src/features/providers/ProvidersWorkbenchPage.tsx`,
    `web/src/router/MainRoutes.tsx`, `web/src/components/layout/MainLayout.tsx`
  - check: `cd web && bun run build` succeeds
- task 4.2: Remove `web/src/pages/OAuthPage.tsx` with `trash` and confirm no dangling import.
  - touched: `web/src/pages/OAuthPage.tsx` (removed)
  - check: `cd web && bun run type-check` clean and
    `grep -rn "OAuthPage" web/src` returns no results
- task 4.3: i18n — add `providersPage.categories.sections.*` and `providersPage.oauthAccounts.*` to
  both locale files; reuse existing `auth_login.*` keys unchanged.
  - touched: `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json`
  - check: `grep -c` key parity between the two locale files for both new namespaces; no literal
    English string in the new panel JSX
- task 4.4: Runtime sweep against the running server via `make dev-web`: rail shows 5 sections /
  15 rows in `## Building` order; `/oauth` redirects and selects `oauth:anthropic`; each of the 6
  OAuth rows renders login button + account table; the browser console is free of React errors and
  warnings across the sweep.
  - touched: none (verification only)
  - check: `make build-web && make build` succeeds, then the browser sweep above passes with a
    clean console. **Stop and restore `/oauth` as a standalone route if the OAuth flow is broken in
    a way the extraction cannot explain (see `## Approach and Risks` recovery).**

### Phase: provider-preset-rows
- story_id: `01KZB8FFGC7J2NVD3QP72S8ERV`
- goal: Add OpenRouter, NVIDIA, and OpenCode as always-visible peer rows in the rail, each owning
  the `openai-compatibility` entries whose base URL matches its preset, with the rest falling to a
  `CUSTOM` row.
- depends_on: provider-console-merge
- touched_surfaces: `web/src/features/providers/entries.ts`,
  `web/src/features/providers/panels/PresetProviderPanel.tsx` (new),
  `web/src/features/providers/ProvidersWorkbenchPage.tsx`,
  `web/src/features/providers/sheets/ProviderSheet.tsx`,
  `web/src/features/providers/sheets/forms/BaseProviderForm.tsx`,
  `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json`
- avoided_surfaces: no Go change and no new preset entries (`NG4`, `NG5`); no `preset_id` config
  field (`NG6`); no change to `ProviderResourceTable`; no new test files under `web/`
- lifecycle status: planned

#### Wave 1 — Preset entries and partition (R6, R7, depends on provider-console-merge)
- task 1.1: `buildEntries` accepts `presets: ProviderPreset[]`, emits one `preset` entry per catalog
  entry regardless of matched-resource count, and assigns category via the `presetCategory`
  derivation in `## Changes` §1.
  - touched: `web/src/features/providers/entries.ts`
  - check: `cd web && bun run type-check` clean; runtime sweep in Wave 3 confirms
    `opencode → FREE`, `openrouter → FREE TIER`, `nvidia → FREE TIER`
- task 1.2: Partition the `openaiCompatibility` group by `normalizeBaseUrl` against a
  `Map<normalizedBaseUrl, presetId>`; unmatched resources become the `CUSTOM` → `OpenAI Compatible`
  row's resources.
  - touched: `web/src/features/providers/entries.ts`
  - check: runtime sweep in Wave 3 asserts the count identity — sum of preset-row resource counts
    plus the `CUSTOM` row's count equals the total `openai-compatibility` resource count
- task 1.3: `ProvidersWorkbenchPage` fetches `providersApi.getProviderPresets()` alongside the
  workbench snapshot; a rejected fetch yields zero preset rows and leaves every entry under
  `CUSTOM` without breaking the rail.
  - touched: `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - check: `cd web && bun run build` succeeds; runtime sweep in Wave 3 confirms the degraded path by
    blocking `/provider-presets` in devtools and reloading

#### Wave 2 — Preset panel and prefilled create (R8, R9, depends on Wave 1)
- task 2.1: Create `panels/PresetProviderPanel.tsx` — preset header (display name, base URL,
  free-tier note, `IconExternalLink` signup link, unverified warning matching the styling landed in
  `87f5c20d`) above the existing `ProviderResourceTable` with the same props the
  openai-compatibility panel passes today, plus an empty state naming base-URL matching as the
  grouping rule.
  - touched: `web/src/features/providers/panels/PresetProviderPanel.tsx` (new),
    `web/src/features/providers/ProvidersWorkbenchPage.tsx`
  - check: `cd web && bun run type-check && bun run lint` clean
- task 2.2: `ProviderSheet` and `BaseProviderForm` accept an optional `initialPresetId` that
  preselects the existing preset picker; the preset panel's Add button passes its own preset id.
  - touched: `web/src/features/providers/sheets/ProviderSheet.tsx`,
    `web/src/features/providers/sheets/forms/BaseProviderForm.tsx`
  - check: `cd web && bun run build` succeeds; runtime sweep in Wave 3 confirms Add from the
    OpenRouter row opens the sheet with OpenRouter already selected and base URL/headers prefilled
- task 2.3: i18n — add `providersPage.presets.*` to both locale files.
  - touched: `web/src/i18n/locales/en.json`, `web/src/i18n/locales/vi.json`
  - check: `grep -c` key parity between the two locale files for the new namespace; no literal
    English string in the new panel JSX

#### Wave 3 — Full verification (depends on Wave 2)
- task 3.1: Full gate — type check, lint, production build, embedded build, then the browser sweep
  covering every check deferred from Waves 1 and 2 (category assignment, count identity, degraded
  preset fetch, prefilled Add), with a clean console throughout.
  - touched: none (verification only)
  - check: `cd web && bun run type-check && bun run lint && bun run build` all clean;
    `make build-web && make build` succeeds; browser sweep passes; no file under `web/` matching
    `*.test.*` or `*.spec.*` was added

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- none

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- 2026-08-06 · planning · `D1` merge two screens only, OAuth panel embeds the account list ·
  owner answer: "Gộp 2 màn, panel OAuth nhúng danh sách account". Rationale: absorbing the 890-line
  `/auth-files` page is where feature loss would come from; a mini-table plus a link gives the OAuth
  panel real content at a fraction of the risk.
- 2026-08-06 · planning · `D2` preset rows are always visible, catalog-style · owner answer:
  "Catalog — luôn hiện, kể cả chưa cấu hình". Rationale: OAuth cards are visible before login, so
  preset rows must be too or they are not peers.
- 2026-08-06 · planning · `D3` OpenRouter and NVIDIA are separate top-level rows · owner answer:
  "openrouter, nvidia riêng như 9router". The owner did not pick a matching mechanism, so planning
  chose base-URL matching (`NG6`) as the reversible zero-schema-change default and stated it back
  before this plan was written.
- 2026-08-06 · planning · `D4` keep the 240px rail + panel layout grouped by category · owner
  answer: "Giữ rail trái 240px + panel phải, group theo category". Rationale: preserves the resource
  table's sort, model filter, usage stats, and row actions, which a card grid would drop.
- 2026-08-06 · planning · plan authored without a prior `brainstorm` artifact · the `/think`
  session that preceded this stage locked outcome, authority, requirements, and non-goals with four
  explicit owner answers, so the initiative was not ambiguous and routing back to `brainstorm`
  would have re-derived settled scope.

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- none

## Current State and Next Action
- active_phase: none
- lifecycle_status: planned
- latest_run_id: none
- latest_trace_ids: []
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- open_items: []
- exact_next_action: work full provider-console-merge
