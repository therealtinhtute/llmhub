# Plan 004: Fix LoginPage double error — remove redundant toast on submission failure

> **Executor instructions**: Follow this plan step by step. Run every
> verification command before moving on. If anything in "STOP conditions" occurs,
> stop and report — do not improvise. Update the status row in `plans/README.md`
> when done.
>
> **Drift check (run first)**:
> `git diff --stat 5f81cee..HEAD -- web/src/pages/LoginPage.tsx`
> If the file changed since this plan was written, compare the
> "Current state" excerpt below against the live code before proceeding.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: VERY LOW
- **Depends on**: none
- **Category**: UX / interaction
- **Planned at**: commit `5f81cee`, 2026-06-12

## Why this matters

When login fails, the page currently emits the same error message **twice in two
different UI layers**:

1. `setError(message)` — sets inline form error text that appears directly under the
   form (appropriate for form validation feedback, clearly scoped to this action).
2. `toast.error(...)` — also fires a pop-up toast notification at the same moment.

Users get a toast pop-up AND the inline error simultaneously. The toast is redundant
because the inline error is already visible and stays visible (it doesn't disappear
after a few seconds the way a toast does). The toast adds noise without adding
information.

On success (line 174), a `toast.success(...)` IS appropriate because there is no
persistent inline success state — the page navigates away, and the toast confirms the
action completed. That line must not be touched.

## Current state

**`web/src/pages/LoginPage.tsx:176–183`**:

```ts
    } catch (err: unknown) {
      const message = getLocalizedErrorMessage(err, t);
      setError(message);                                                   // line 178 — KEEP
      toast.error(`${t('notification.login_failed')}: ${message}`);       // line 179 — REMOVE
    } finally {
      setLoading(false);
    }
```

## Scope

**In scope**: one line deleted in `web/src/pages/LoginPage.tsx`.
**Out of scope**: every other file. Do not touch the success toast at line 174.

## Git workflow

- Branch: `advisor/004-login-double-error`
- Commit: `fix(web): remove redundant toast on login failure`
- Do NOT push or open a PR unless instructed.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Type-check | `cd web && bun run type-check` | exit 0 |
| Tests | `cd web && bun run test:run` | 60 tests pass |
| Lint | `cd web && bun run lint` | exit 0 errors |

## Steps

### Step 1: Delete the redundant toast line

In `web/src/pages/LoginPage.tsx`, remove **line 179 only**:
```ts
      toast.error(`${t('notification.login_failed')}: ${message}`);
```

The catch block after the change should look like:

```ts
    } catch (err: unknown) {
      const message = getLocalizedErrorMessage(err, t);
      setError(message);
    } finally {
      setLoading(false);
    }
```

**Verify**:
```bash
grep -n "toast.error" web/src/pages/LoginPage.tsx
```
Expected output: **no lines** (the `toast.error` call is gone). The `toast.success`
call at line 174 is NOT a `toast.error` — it will not be flagged by this grep.

### Step 2: Check whether `toast` import is now unused

After removing the only `toast.error` call, check if `toast` is still referenced
elsewhere in the file:
```bash
grep -n "\btoast\b" web/src/pages/LoginPage.tsx
```
If the only remaining reference is the import (i.e. `toast.success` is also gone),
remove the import. If `toast.success` is still present (line 174), the import stays.
The expected result: at least one `toast` usage remains (the success toast), so the
import does not need to change.

### Step 3: Run all gates

```bash
cd web && bun run type-check
cd web && bun run test:run
cd web && bun run lint
```

**Verify**: all exit 0.

## Test plan

No automated tests exist for LoginPage (it requires full router and store context).
Manual verification:

1. Start `make dev` + `make dev-web`.
2. On the login page, enter incorrect credentials and submit.
3. Verify: inline error message appears under the form — **no toast appears**.
4. Enter correct credentials and submit.
5. Verify: `toast.success(...)` still fires ("Connected" confirmation).

## Done criteria

- [ ] `grep -n "toast.error" web/src/pages/LoginPage.tsx` → no output
- [ ] `grep -n "toast.success" web/src/pages/LoginPage.tsx` → still present (line ~174)
- [ ] `cd web && bun run type-check` exits 0
- [ ] `cd web && bun run test:run` exits 0; 60 tests pass
- [ ] `cd web && bun run lint` exits 0 errors
- [ ] `git status` shows only `LoginPage.tsx` modified
- [ ] `plans/README.md` status row updated to DONE

## STOP conditions

- The catch block at lines 176–183 does not match the excerpt above — drift, stop
  and report the current code.
- After removal, `bun run type-check` reports `toast` is now an unused import — in
  that case also remove the `toast` import from the file's import list.

## Maintenance notes

- If a `notification.login_failed` key is now unreferenced in the whole codebase,
  consider removing it from the locale files (cosmetic, not required for this plan).
  `grep -rn "login_failed" web/src/` first to confirm whether it's still used
  elsewhere before removing.
