# Gap Classification

Two layers. `upstream_gap.py` produces the structural class from blob identity — a fact.
The semantic disposition is the analyst's judgment, recorded in the ledger.

## Structural classes

Computed by comparing blob SHAs across three trees: upstream `from`, upstream `to`, local
`HEAD` (after `path_map` rewriting).

| Class | Condition | Meaning | Action |
| --- | --- | --- | --- |
| `match` | local blob == upstream `to` blob | Already identical to the target | None; do not re-port |
| `baseline` | local blob == upstream `from` blob | Untouched locally, upstream moved | Prime port candidate — a clean apply is usually possible |
| `upstream-add-absent` | upstream added it, missing locally | New upstream capability | Read it; likely the real feature work |
| `diverged-absent` | upstream changed it, missing locally | Local architecture never had this file | Map to the local equivalent before judging |
| `upstream-delete-present-local` | upstream deleted it, present locally | Upstream removed or relocated | Check for a rename before deleting anything locally |
| `semantic-review` | differs from both `from` and `to` | Both sides evolved | Requires reading; the expensive bucket |

`match` and `baseline` are cheap and trustworthy. `semantic-review` is where the real work
is — expect it to dominate on a mature fork.

## Semantic dispositions

Every ledger row ends at exactly one of these. No row may stay blank at the end of triage.

| Disposition | Use when | Required evidence |
| --- | --- | --- |
| `already-present` | Local code implements the same behavior differently | Local `path:line` of the implementing symbol **and** the upstream `path:line` |
| `adapt` | Behavior is wanted and must be re-expressed behind local interfaces | Upstream commit sha + the local interface it lands behind |
| `reject` | Behavior conflicts with a local invariant or is unwanted | The invariant it would invert, named concretely |
| `superseded-locally` | Local code already solved it in a stronger way | Local `path:line` plus why it is stronger |
| `defer` | Wanted but out of this cycle's scope | The condition that would pull it in |

## Evidence discipline

1. Never infer `already-present` from a path name, a file existing, or a similar function
   name. Name the symbol that implements the behavior.
2. Never mark `adapt` without reading the upstream diff. Use `librarian` to cite exact
   upstream lines rather than cloning or guessing.
3. A `reject` must name the invariant — "Postgres is the authoritative runtime store", not
   "does not fit our architecture".
4. When upstream and local disagree on behavior and both look defensible, that is a product
   decision. Surface it with `AskUserQuestion`; do not silently pick one.
5. Report counts exactly as the script printed them. Do not round, re-derive, or estimate.

## Reading a `semantic-review` path efficiently

1. `git diff {from_ref} {to_ref} -- {upstream_path}` — what upstream actually changed.
2. `git diff {from_ref} HEAD -- {upstream_path}` — how far the local file already drifted.
3. If step 1 is a bug fix and step 2 shows the local file never had the bug, the answer is
   `superseded-locally` or `already-present` — stop, do not read further.
4. Only when both diffs touch the same behavior does the path need full reading.

## Known false signals

- **Vendored or generated files** (lockfiles, `models.json`, embedded assets) inflate
  `semantic-review`. Classify them as a group, not per file.
- **Module path rewrites** across the whole tree turn every file into `semantic-review`.
  If the count is implausibly high, `path_map` or the import path is the cause — fix that
  before analyzing.
- **Branding and release churn** appear as real diffs but carry no behavior. Reject as a
  group once, then cite that group decision for each row.
