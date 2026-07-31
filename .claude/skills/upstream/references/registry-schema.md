# Registry and Checkpoint Schema

Two file kinds. The registry indexes upstreams; one checkpoint file per upstream holds its state.

## `docs/upstream/registry.json`

```json
{
  "schema_version": 1,
  "upstreams": [
    {
      "slug": "cliproxyapi",
      "owner_repo": "router-for-me/CLIProxyAPI",
      "repository": "https://github.com/router-for-me/CLIProxyAPI",
      "remote": "cliproxyapi",
      "checkpoint": "docs/upstream/cliproxyapi-checkpoint.json",
      "path_map": {}
    }
  ]
}
```

| Field | Meaning |
| --- | --- |
| `slug` | Short identifier; also the ref namespace segment and file prefix |
| `owner_repo` | GitHub `owner/name`, used for `gh api` release queries |
| `remote` | Local git remote name; created by `upstream_sync.py add` |
| `checkpoint` | Repo-relative path to this upstream's checkpoint file |
| `path_map` | Longest-prefix rewrites from upstream paths to local paths |

### `path_map`

Use when the local tree renamed or relocated an upstream package. Keys are upstream
path prefixes, values are local prefixes. Longest matching prefix wins.

```json
"path_map": {
  "internal/api/handlers": "internal/api/modules",
  "sdk/cliproxy": "sdk/cliproxy"
}
```

An unmapped path compares against the identical local path. A wrong `path_map` silently
inflates `diverged-absent` — verify a sample by hand after adding entries.

## `docs/upstream/{slug}-checkpoint.json`

```json
{
  "schema_version": 1,
  "upstream": { "repository": "...", "remote": "cliproxyapi" },
  "scope_policy": {
    "strategy": "targeted-semantic-ports",
    "include": ["full-codex-live", "postgres-backed-controls"],
    "exclude": ["plugin-platform", "wholesale-source-merge"]
  },
  "checkpoint": {
    "tag": "v7.2.112",
    "commit": "a63da8ae...",
    "ref": "refs/upstream-checkpoints/cliproxyapi/v7.2.112",
    "published_at": "2026-07-31T08:39:29Z",
    "url": "...",
    "target_commitish": "main",
    "checked_at": "2026-07-31T09:30:29Z"
  },
  "local_baseline": { "commit": "234daa3f...", "branch": "master" },
  "prior_checkpoints": [
    { "tag": "v7.2.93", "commit": "...", "ref": "...", "role": "prior-approved-targeted-parity" }
  ],
  "source_range": { "from_exclusive": "v7.2.93", "to_inclusive": "v7.2.112" },
  "releases": [{ "tag": "v7.2.94", "commit": "...", "published_at": "...", "url": "..." }]
}
```

| Field | Rule |
| --- | --- |
| `checkpoint` | Always a published non-prerelease release, never a branch head |
| `checkpoint.ref` | Local immutable ref; the only thing analysis is allowed to diff against |
| `local_baseline` | Local commit at sync time; makes every gap count reproducible |
| `prior_checkpoints` | Append-only history; `role` records why that checkpoint mattered |
| `source_range` | The exclusive/inclusive tag window all gap and ledger output covers |
| `releases` | Every stable release inside the window, oldest first |
| `scope_policy` | Written only by `upstream_sync.py policy` after user approval |

`scope_policy` survives every sync — the script preserves it. Never hand-edit it to widen
scope; rerun `policy` so the change is attributable.

## Generated analysis files

| Path | Producer | Lifetime |
| --- | --- | --- |
| `docs/upstream/{slug}-gap-{from}..{to}.json` | `upstream_gap.py` | Regenerate freely; derived |
| `docs/upstream/{slug}-ledger-{from}..{to}.md` | `upstream_ledger.py` | Hand-edited during triage; keep |

The ledger is the only generated file that accumulates human judgment. Regenerating it
overwrites the Disposition column — copy it out first if the range has not changed.
