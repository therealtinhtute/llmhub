# Implementation Notes

Running log of decisions, surprises, and tradeoffs made during plan execution.
Written for the repo owner — things the plan spec didn't cover or that turned out differently.

---

## Plan 001 — Fix restoreSession failure cache
**Executed**: 2026-06-12  
**File changed**: `web/src/stores/useAuthStore.ts` (+1 line)

### What the plan said vs. what I found

The plan's "Current state" excerpt matched the live code exactly — no drift.

### Decisions not in the spec

**Where exactly to clear the cache.**  
The catch block has two concerns: (1) clear the singleton so the next call can retry, and (2) return `false` so the current caller knows login failed. I placed `restoreSessionPromise = null` *before* `return false` so that if for any reason the return were ever moved or wrapped, the clear still happens. There was an alternative: clear it *after* the IIFE resolves (i.e. in the outer `restoreSession` function). I didn't do that because clearing inside the catch is the right place — it means "this attempt is abandoned, a new attempt is allowed." Clearing after the outer `.then()` would require chaining the promise, which is a bigger change.

**Whether to also clear on the `return false` at line 83 (the non-login path).**  
The IIFE has a second `return false` at line 83 that runs when `wasLoggedIn` is false, or when `resolvedBase`/`resolvedKey` are empty. The plan only asked to fix the catch block. I deliberately left line 83 alone because:
- That path isn't an error — it's a legitimate "no stored session" state.
- Clearing the promise there would mean re-running the migration logic (`migratePlaintextKeys`) on every call, which is wasteful.
- The original intent of the singleton is to deduplicate concurrent calls on mount, not to allow indefinite retries of non-error paths.

**No change to the `return false` at line 83 is the right call.** The singleton should only be cleared when we want the caller to retry, not when there's simply nothing to restore.

### Lint warnings (pre-existing, not introduced)

`bun run lint` exits with 8 warnings, all in `web/src/components/ui/sidebar.tsx` (fast-refresh export warning). None are related to this change. Zero errors.

### Tradeoffs

**One-line fix vs. richer retry logic.**  
A more thorough fix would add exponential backoff or a retry count. I didn't do that — the plan asked for minimum safe change, and the UX is already acceptable: the next page load (or any future `restoreSession()` call) will retry cleanly. Adding retry logic here would require either a loop inside the IIFE or a new exported function, which is scope creep.

**Module-level singleton remains.**  
The singleton pattern itself is fine — `restoreSession` is called from one place (`LoginPage.tsx:135`). The fix just makes failures non-sticky. If a second caller is ever added, the singleton should be revisited (noted in the plan's maintenance notes).

---

## Plan 002 — Web test infrastructure (Vitest + utility test suite)
**Executed**: 2026-06-12  
**Files created**: `vitest.config.ts`, `package.json` (+2 scripts), `src/utils/__tests__/validation.test.ts`, `src/utils/__tests__/encryption.test.ts`, `src/utils/__tests__/recentRequests.test.ts`, `src/utils/__tests__/format.test.ts`

### Things the plan didn't cover

**Vitest version landed as 4.1.8, not v2/v3.**  
The plan was written against a theoretical "current Vitest" without pinning a version. Bun resolved `vitest@4.1.8`. The config API is the same across v2–v4, so no changes were needed.

**`vitest.config.ts` instead of extending `vite.config.ts`.**  
The plan recommended a standalone `vitest.config.ts` to avoid the `viteSingleFile` plugin interfering. This was confirmed correct — `viteSingleFile` would have tried to inline all test assets, which breaks vitest's module resolution. A separate config was the right call.

**`jsdom` environment is needed only for `encryption.ts`**, but the plan used it globally (all files). The tradeoff: slightly heavier per-file setup vs. per-file `@vitest-environment` annotations. For four utility files with no Node-specific concern, global jsdom is simpler and has negligible cost (~330ms for environment setup regardless).

### One test failure during implementation and why

**`formatFileSize(512)` returned `'512.00 B'`, not `'0.50 KB'`.**  
The plan's test expected sub-1024 byte counts to be rendered as fractional KB. The actual implementation uses `Math.floor(Math.log(bytes) / Math.log(1024))` which returns `0` for any value under 1024, keeping the unit as `B`. The test was wrong, not the implementation. Fixed by replacing `formatFileSize(512) → '0.50 KB'` with two correct cases: `formatFileSize(1536) → '1.50 KB'` and `formatFileSize(512) → '512.00 B'`.

This is a spec-documentation value: `formatFileSize` never shows fractional kilobytes for values under 1 KB — it stays in bytes. Worth knowing if you ever display file sizes near the KB boundary.

### `Array.prototype.at()` not available in the project's ES2020 lib

The plan's test drafts used `.at(-1)` for "last element of array". TypeScript rejected these because `tsconfig.json` targets `ES2020` and `lib: ["ES2020", ...]`, and `Array.at()` is ES2022.

**Decision: replace `.at(-1)` with `arr[arr.length - 1]` in the tests** rather than bumping the tsconfig lib target. Bumping the lib would affect the production build's type surface — tests shouldn't drive that. The workaround is two characters longer and equally clear.

**The plan template should note the ES lib constraint** for future test authors.

### What I didn't test (and why)

**`encryption.ts` fallback path** (when `window.location` throws):  
The `getKeyBytes()` function has a catch block that falls back to a simple key if `window.location.host` throws. Testing this requires either `vi.spyOn(window, 'location', 'get').mockImplementation(...)` or `vi.resetModules()` to clear the module-level `cachedKeyBytes`. Either adds complexity for a branch that only fires in non-browser environments — not worth it at this stage.

**`formatUnixTimestamp` exact output**:  
The function calls `.toLocaleString()` without a fixed locale, so output varies by OS locale settings. Tests assert structure (non-empty string) rather than exact text. If locale-stable output ever matters, the function should accept a `locale` parameter and tests should pass `'en-US'`.

**`normalizeTimestampForDateParse` from `timestamp.ts`**:  
Not in scope for this plan (scope was the four main utility files). Worth adding — it has clear input/output contracts and handles a subtle RFC3339 precision edge case.

### Verification summary (plan 001)

| Check | Result |
|-------|--------|
| `grep "restoreSessionPromise = null"` prints 2 lines | ✅ lines 78 and 140 |
| `bun run type-check` exits 0 | ✅ |
| `bun run lint` exits 0 errors (8 pre-existing warnings) | ✅ |
| Only `useAuthStore.ts` modified (+1 line) | ✅ |

### Verification summary (plan 002)

| Check | Result |
|-------|--------|
| `grep '"vitest"' package.json` prints version line | ✅ `"vitest": "^4.1.8"` |
| `ls vitest.config.ts` exits 0 | ✅ |
| `grep '"test:run"' package.json` prints new script | ✅ |
| `bun run test:run` exits 0 | ✅ 60 tests, 4 files |
| `bun run type-check` exits 0 | ✅ (after removing `.at()` usage) |
| `bun run lint` exits 0 errors | ✅ (8 pre-existing warnings) |
| Only in-scope files modified/created | ✅ |
