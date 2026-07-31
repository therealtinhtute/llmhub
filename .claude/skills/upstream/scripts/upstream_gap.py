#!/usr/bin/env python3
"""Classify every path that changed upstream between two checkpoints against the local tree.

Compares blob identities across three trees — upstream `from`, upstream `to`, and local
HEAD — so each class is a fact, not a guess. Writes
docs/upstream/{slug}-gap-{from}..{to}.json and prints per-class counts.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path
from typing import Any

from _common import (
    UpstreamError,
    checkpoint_path,
    fail,
    find_upstream,
    load_registry,
    map_path,
    now_iso,
    read_json,
    ref_exists,
    ref_for,
    repo_root,
    run,
    write_json,
)

CLASSES = (
    "match",
    "baseline",
    "upstream-add-absent",
    "diverged-absent",
    "upstream-delete-present-local",
    "semantic-review",
)


def tree_blobs(ref: str) -> dict[str, str]:
    """Map path -> blob sha for every file in a tree."""
    out = run(["git", "ls-tree", "-r", ref])
    blobs: dict[str, str] = {}
    for line in out.splitlines():
        if not line:
            continue
        meta, _, path = line.partition("\t")
        parts = meta.split()
        if len(parts) < 3 or parts[1] != "blob":
            continue
        blobs[path] = parts[2]
    return blobs


def changed_paths(from_ref: str, to_ref: str) -> dict[str, str]:
    """Map path -> upstream change status (A/M/D/R...) between two upstream refs."""
    out = run(["git", "diff", "--name-status", "-M", from_ref, to_ref])
    changes: dict[str, str] = {}
    for line in out.splitlines():
        if not line:
            continue
        fields = line.split("\t")
        status = fields[0]
        if status.startswith("R") and len(fields) >= 3:
            changes[fields[1]] = "D"
            changes[fields[2]] = "A"
        elif len(fields) >= 2:
            changes[fields[1]] = status[0]
    return changes


def classify(
    upstream_status: str,
    from_blob: str | None,
    to_blob: str | None,
    local_blob: str | None,
) -> str:
    if to_blob is None:
        return "upstream-delete-present-local" if local_blob else "match"
    if local_blob == to_blob:
        return "match"
    if local_blob is None:
        return "upstream-add-absent" if upstream_status == "A" else "diverged-absent"
    if local_blob == from_blob:
        return "baseline"
    return "semantic-review"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--slug")
    parser.add_argument("--from", dest="from_tag", help="override source_range.from_exclusive")
    parser.add_argument("--to", dest="to_tag", help="override source_range.to_inclusive")
    parser.add_argument("--local", default="HEAD", help="local comparison ref (default: HEAD)")
    parser.add_argument("--class", dest="only", choices=CLASSES, help="print only this class")
    args = parser.parse_args()

    root = repo_root()
    registry = load_registry(root)
    entry = find_upstream(registry, args.slug)
    slug = entry["slug"]
    path_map: dict[str, str] = entry.get("path_map") or {}

    checkpoint = read_json(checkpoint_path(root, entry))
    if not checkpoint:
        raise UpstreamError(f"no checkpoint for {slug} — run: upstream_sync.py sync --slug {slug}")

    source_range = checkpoint.get("source_range", {})
    from_tag = args.from_tag or source_range.get("from_exclusive")
    to_tag = args.to_tag or source_range.get("to_inclusive")
    if not from_tag:
        raise UpstreamError(
            f"{slug} has no prior checkpoint; pass --from <tag> to pick a comparison base"
        )
    if not to_tag:
        raise UpstreamError(f"{slug} checkpoint has no target tag")

    from_ref, to_ref = ref_for(slug, from_tag), ref_for(slug, to_tag)
    for tag, ref in ((from_tag, from_ref), (to_tag, to_ref)):
        if not ref_exists(ref):
            raise UpstreamError(
                f"missing ref {ref} — run: git fetch --no-tags {entry.get('remote', slug)} "
                f"+refs/tags/{tag}:{ref}"
            )

    from_blobs = tree_blobs(from_ref)
    to_blobs = tree_blobs(to_ref)
    local_blobs = tree_blobs(args.local)
    changes = changed_paths(from_ref, to_ref)

    rows: list[dict[str, Any]] = []
    counts = {name: 0 for name in CLASSES}
    for upstream_path in sorted(changes):
        local_path = map_path(upstream_path, path_map)
        kind = classify(
            changes[upstream_path],
            from_blobs.get(upstream_path),
            to_blobs.get(upstream_path),
            local_blobs.get(local_path),
        )
        counts[kind] += 1
        rows.append(
            {
                "upstream_path": upstream_path,
                "local_path": local_path,
                "upstream_status": changes[upstream_path],
                "class": kind,
                "local_present": local_path in local_blobs,
            }
        )

    payload = {
        "schema_version": 1,
        "slug": slug,
        "generated_at": now_iso(),
        "from_exclusive": from_tag,
        "to_inclusive": to_tag,
        "local_ref": run(["git", "rev-parse", args.local]).strip(),
        "total_changed_paths": len(rows),
        "counts": counts,
        "paths": rows,
    }
    out_path = root / "docs" / "upstream" / f"{slug}-gap-{from_tag}..{to_tag}.json"
    write_json(out_path, payload)

    if args.only:
        for row in rows:
            if row["class"] == args.only:
                print(f"{row['class']:<30} {row['upstream_path']}")
        return 0

    print(f"{slug}: {from_tag} -> {to_tag} vs {args.local}")
    print(f"changed paths upstream: {len(rows)}")
    for name in CLASSES:
        print(f"  {name:<30} {counts[name]}")
    print(f"written: {out_path.relative_to(root)}")
    print(f"next: review each 'semantic-review' path — upstream_gap.py --slug {slug} --class semantic-review")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except UpstreamError as exc:
        fail(str(exc))
