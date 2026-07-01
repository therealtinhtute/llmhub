# Plan 001: Fix restoreSession so transient failures don't permanently block auto-login

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat 5f81cee..HEAD -- web/src/stores/useAuthStore.ts`
> If the file changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, treat it as
> a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `5f81cee`, 2026-06-11

## Why this matters

`restoreSession()` is called once on app start to auto-login users who checked
"Remember me". It uses a module-level promise singleton to prevent duplicate
calls. However, when the call fails (network error, server temporarily down),
the resolved-`false` promise is cached permanently. Subsequent calls return the
cached failure without retrying — so the user is stuck entering credentials
manually every page load even though `rememberPassword=true` and their stored
key is valid. The fix is to clear the cache on failure so the next mount can try
again.

## Current state

- `web/src/stores/useAuthStore.ts` — Zustand auth store; contains the
  `restoreSession` implementation and the module-level singleton.

Current code at `useAuthStore.ts:29` and `45–87`:

```ts
// line 29
let restoreSessionPromise: Promise<boolean> | null = null;

// lines 45–87
restoreSession: () => {
  if (restoreSessionPromise) return restoreSessionPromise;

  restoreSessionPromise = (async () => {
    // ... migration + resolve credentials ...

    if (wasLoggedIn && resolvedBase && resolvedKey) {
      try {
        await get().login({ ... });
        return true;
      } catch (error) {
        console.warn('Auto login failed:', error);
        return false;     // <-- promise resolves false AND is still cached
      }
    }

    return false;
  })();

  return restoreSessionPromise;
},
```

The catch block at ~line 77 returns `false` but does **not** clear
`restoreSessionPromise`. Every subsequent call hits `if (restoreSessionPromise)`
and returns the same cached `false` without retrying.

The `logout()` at line 139 correctly resets the promise:
```ts
logout: () => {
  restoreSessionPromise = null;  // line 139
  ...
}
```

**Repo conventions**: this is a TypeScript/Zustand store; there are no separate
error-handling utilities — errors are caught inline in store actions and
surfaced via `set({ connectionError: ... })`. Match the existing pattern.
The project uses `bun` (see `web/package.json` `"packageManager": "bun@1.3.14"`).

## Commands you will need

| Purpose     | Command                              | Expected on success        |
|-------------|--------------------------------------|----------------------------|
| Type-check  | `cd web && bun run type-check`       | exit 0, no errors          |
| Lint        | `cd web && bun run lint`             | exit 0                     |

(No tests exist yet for this store; test infrastructure is added in plan 002.)

## Scope

**In scope** (the only file you should modify):
- `web/src/stores/useAuthStore.ts`

**Out of scope** (do NOT touch):
- `web/src/pages/LoginPage.tsx` — the caller; no changes needed there.
- `web/src/router/ProtectedRoute.tsx` — uses `checkAuth`, not `restoreSession`.
- Any other store or page file.

## Git workflow

- Branch: `advisor/001-restore-session-cache`
- Commit message style matches the repo: `fix(web): clear restoreSession cache on auto-login failure`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Clear the module-level promise on auto-login failure

In `web/src/stores/useAuthStore.ts`, locate the `catch` block inside the IIFE
assigned to `restoreSessionPromise` (~line 77). Add one line to clear the
cache before returning `false`:

```ts
} catch (error) {
  restoreSessionPromise = null;   // <-- add this line
  console.warn('Auto login failed:', error);
  return false;
}
```

This mirrors the `logout()` reset at line 139 and allows the next call to
`restoreSession()` to try again from scratch.

**Verify**: `grep -n "restoreSessionPromise = null" web/src/stores/useAuthStore.ts`
→ should print **two** lines: one inside the catch block (new) and one in `logout()`.

### Step 2: Type-check and lint

**Verify**: `cd web && bun run type-check` → exits 0, no errors.
**Verify**: `cd web && bun run lint` → exits 0.

## Test plan

No automated tests exist for this store yet (plan 002 creates the test
infrastructure). Manual verification:

1. Run `make dev` from the repo root (starts the server on `:9090`).
2. Run `make dev-web` in the `web/` directory.
3. Open the browser, log in with a valid key and check "Remember me".
4. Stop the backend server.
5. Reload the browser — auto-login should fail (server unreachable), and
   `restoreSessionPromise` is now `null`.
6. Restart the backend server.
7. Reload again — auto-login should succeed (no page reload required beyond
   this one).

Without this fix, step 7 would fail silently and show the login form even
though credentials are stored.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `cd web && bun run type-check` exits 0
- [ ] `cd web && bun run lint` exits 0
- [ ] `grep -n "restoreSessionPromise = null" web/src/stores/useAuthStore.ts` prints exactly 2 lines
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

Stop and report back (do not improvise) if:

- The code at `useAuthStore.ts:29` or `useAuthStore.ts:45–87` does not match
  the excerpts above (codebase has drifted since this plan was written).
- `bun run type-check` or `bun run lint` fails after the change and the failure
  is not obviously caused by this edit.
- The fix requires touching any file outside `web/src/stores/useAuthStore.ts`.

## Maintenance notes

- If the `restoreSession` function is ever refactored to accept retry options or
  a signal, the cache-clearing logic here should move to the caller or become
  explicit retry state.
- The module-level singleton pattern works because `restoreSession()` is
  currently only called once (from `LoginPage.tsx:135`). If additional callers
  are added, revisit whether the singleton is still appropriate.
- PR reviewers: confirm the two `restoreSessionPromise = null` lines are
  symmetrically placed (catch block + logout) and that no new `return false`
  paths were introduced without clearing the cache.
