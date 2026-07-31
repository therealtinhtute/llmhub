#!/usr/bin/env python3
"""Emit the upstream commit ledger for a checkpoint range, grouped by release.

One row per non-merge upstream commit with its touched surfaces and an empty
Disposition column for the triage phase to fill. Writes
docs/upstream/{slug}-ledger-{from}..{to}.md.
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
    now_iso,
    object_exists,
    read_json,
    ref_exists,
    ref_for,
    repo_root,
    run,
)

RECORD_SEP = "\x1e"
FIELD_SEP = "\x1f"


def commits_between(base: str, head: str) -> list[dict[str, Any]]:
    out = run(
        [
            "git",
            "log",
            "--no-merges",
            "--date=short",
            f"--pretty=format:{RECORD_SEP}%H{FIELD_SEP}%ad{FIELD_SEP}%an{FIELD_SEP}%s",
            "--name-only",
            f"{base}..{head}",
        ]
    )
    commits: list[dict[str, Any]] = []
    for block in out.split(RECORD_SEP):
        block = block.strip("\n")
        if not block:
            continue
        header, _, body = block.partition("\n")
        sha, date, author, subject = header.split(FIELD_SEP, 3)
        files = [line for line in body.splitlines() if line.strip()]
        commits.append(
            {
                "sha": sha,
                "date": date,
                "author": author,
                "subject": subject,
                "files": files,
            }
        )
    return commits


def surfaces(files: list[str], depth: int = 2, limit: int = 3) -> str:
    seen: list[str] = []
    for path in files:
        parts = path.split("/")
        prefix = "/".join(parts[:depth]) if len(parts) > depth else path
        if prefix not in seen:
            seen.append(prefix)
    if not seen:
        return "-"
    shown = ", ".join(f"`{s}`" for s in seen[:limit])
    return shown + (f" +{len(seen) - limit}" if len(seen) > limit else "")


def escape(text: str) -> str:
    return text.replace("|", "\\|")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--slug")
    parser.add_argument("--from", dest="from_tag")
    parser.add_argument("--to", dest="to_tag")
    args = parser.parse_args()

    root = repo_root()
    registry = load_registry(root)
    entry = find_upstream(registry, args.slug)
    slug = entry["slug"]
    checkpoint = read_json(checkpoint_path(root, entry))
    if not checkpoint:
        raise UpstreamError(f"no checkpoint for {slug} — run: upstream_sync.py sync --slug {slug}")

    source_range = checkpoint.get("source_range", {})
    from_tag = args.from_tag or source_range.get("from_exclusive")
    to_tag = args.to_tag or source_range.get("to_inclusive")
    if not from_tag or not to_tag:
        raise UpstreamError(f"{slug} has no complete source_range; pass --from and --to")

    from_ref, to_ref = ref_for(slug, from_tag), ref_for(slug, to_tag)
    for tag, ref in ((from_tag, from_ref), (to_tag, to_ref)):
        if not ref_exists(ref):
            raise UpstreamError(f"missing ref {ref} for {tag} — run upstream_sync.py sync first")

    all_commits = commits_between(from_ref, to_ref)
    by_sha = {c["sha"]: c for c in all_commits}

    # Assign each commit to the earliest release that contains it.
    releases = [r for r in checkpoint.get("releases", []) if r.get("commit")]
    releases.sort(key=lambda r: r.get("published_at") or "")
    assigned: dict[str, str] = {}
    cursor = from_ref
    for release in releases:
        if not object_exists(release["commit"]):
            continue
        out = run(["git", "rev-list", "--no-merges", f"{cursor}..{release['commit']}"], check=False)
        for sha in out.split():
            if sha in by_sha and sha not in assigned:
                assigned[sha] = release["tag"]
        cursor = release["commit"]

    lines = [
        f"# Upstream ledger — {slug} {from_tag}..{to_tag}",
        "",
        f"- generated: {now_iso()}",
        f"- upstream: {checkpoint.get('upstream', {}).get('repository', '-')}",
        f"- local baseline: `{checkpoint.get('local_baseline', {}).get('commit', '-')[:12]}`",
        f"- non-merge commits: {len(all_commits)}",
        "",
        "Disposition values: `already-present`, `adapt`, `reject`, `superseded-locally`, `defer`.",
        "Every non-empty disposition needs a citation on both sides.",
        "",
        "| Release | Commit | Date | Subject | Surfaces | Disposition | Evidence |",
        "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    ordered = sorted(all_commits, key=lambda c: (assigned.get(c["sha"], "~unassigned"), c["date"]))
    for commit in ordered:
        lines.append(
            "| {rel} | `{sha}` | {date} | {subject} | {surf} |  |  |".format(
                rel=assigned.get(commit["sha"], "unassigned"),
                sha=commit["sha"][:12],
                date=commit["date"],
                subject=escape(commit["subject"])[:110],
                surf=surfaces(commit["files"]),
            )
        )

    unassigned = sum(1 for c in all_commits if c["sha"] not in assigned)
    if unassigned:
        lines += ["", f"> {unassigned} commit(s) are not reachable from any recorded release commit."]

    out_path = root / "docs" / "upstream" / f"{slug}-ledger-{from_tag}..{to_tag}.md"
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(lines) + "\n", encoding="utf-8")

    print(f"{slug}: {from_tag}..{to_tag}")
    print(f"non-merge commits: {len(all_commits)}  releases: {len(releases)}  unassigned: {unassigned}")
    print(f"written: {out_path.relative_to(root)}")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except UpstreamError as exc:
        fail(str(exc))
