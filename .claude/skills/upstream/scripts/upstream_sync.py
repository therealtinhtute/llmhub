#!/usr/bin/env python3
"""Register upstream repositories and advance their immutable checkpoints.

Subcommands:
  list                                  show registered upstreams and current checkpoints
  add    --slug S --repo owner/name     register an upstream and its git remote
  sync   --slug S [--tag vX.Y.Z]        resolve the latest release, fetch it, advance checkpoint
  policy --slug S [--include ...]       record the approved include/exclude scope policy
"""

from __future__ import annotations

import argparse
import json
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
    read_json,
    ref_exists,
    ref_for,
    registry_path,
    repo_root,
    run,
    write_json,
)


def gh_releases(owner_repo: str) -> list[dict[str, Any]]:
    raw = run(
        [
            "gh",
            "api",
            "--paginate",
            f"repos/{owner_repo}/releases?per_page=100",
        ]
    )
    releases: list[dict[str, Any]] = []
    decoder = json.JSONDecoder()
    idx = 0
    # --paginate concatenates JSON arrays; decode them back to back.
    while idx < len(raw):
        while idx < len(raw) and raw[idx].isspace():
            idx += 1
        if idx >= len(raw):
            break
        chunk, offset = decoder.raw_decode(raw, idx)
        releases.extend(chunk)
        idx = offset
    stable = [
        r for r in releases if not r.get("prerelease") and not r.get("draft") and r.get("tag_name")
    ]
    stable.sort(key=lambda r: r.get("published_at") or "", reverse=True)
    return stable


def release_commit(owner_repo: str, tag: str) -> str:
    out = run(["gh", "api", f"repos/{owner_repo}/git/ref/tags/{tag}"])
    obj = json.loads(out)["object"]
    if obj["type"] == "commit":
        return obj["sha"]
    tag_obj = json.loads(run(["gh", "api", f"repos/{owner_repo}/git/tags/{obj['sha']}"]))
    return tag_obj["object"]["sha"]


def ensure_remote(remote: str, repository: str) -> None:
    existing = run(["git", "remote"], check=False).split()
    if remote not in existing:
        run(["git", "remote", "add", remote, repository])


def fetch_checkpoint(remote: str, slug: str, tag: str) -> None:
    ref = ref_for(slug, tag)
    if ref_exists(ref):
        return
    run(["git", "fetch", "--no-tags", remote, f"+refs/tags/{tag}:{ref}"])


def local_baseline() -> dict[str, str]:
    return {
        "commit": run(["git", "rev-parse", "HEAD"]).strip(),
        "branch": run(["git", "rev-parse", "--abbrev-ref", "HEAD"]).strip(),
    }


def cmd_list(root: Path, _args: argparse.Namespace) -> int:
    registry = load_registry(root)
    entries = registry.get("upstreams", [])
    if not entries:
        print("no upstream registered")
        return 0
    for entry in entries:
        checkpoint = read_json(checkpoint_path(root, entry)).get("checkpoint", {})
        tag = checkpoint.get("tag", "-")
        ref = ref_for(entry["slug"], tag) if tag != "-" else "-"
        state = "present" if tag != "-" and ref_exists(ref) else "missing"
        print(f"{entry['slug']:<16} {entry['owner_repo']:<40} checkpoint={tag} ref={state}")
    return 0


def cmd_add(root: Path, args: argparse.Namespace) -> int:
    if "/" not in args.repo:
        raise UpstreamError("--repo must be owner/name")
    registry = load_registry(root)
    if any(e.get("slug") == args.slug for e in registry["upstreams"]):
        raise UpstreamError(f"slug already registered: {args.slug}")
    remote = args.remote or args.slug
    repository = f"https://github.com/{args.repo}"
    ensure_remote(remote, repository)
    registry["upstreams"].append(
        {
            "slug": args.slug,
            "owner_repo": args.repo,
            "repository": repository,
            "remote": remote,
            "checkpoint": f"docs/upstream/{args.slug}-checkpoint.json",
            "path_map": {},
        }
    )
    write_json(registry_path(root), registry)
    print(f"registered {args.slug} -> {repository} (remote: {remote})")
    print(f"next: upstream_sync.py sync --slug {args.slug}")
    return 0


def cmd_sync(root: Path, args: argparse.Namespace) -> int:
    registry = load_registry(root)
    entry = find_upstream(registry, args.slug)
    slug = entry["slug"]
    owner_repo = entry["owner_repo"]
    remote = entry.get("remote", slug)
    ensure_remote(remote, entry["repository"])

    releases = gh_releases(owner_repo)
    if not releases:
        raise UpstreamError(f"no stable releases found for {owner_repo}")

    if args.tag:
        target = next((r for r in releases if r["tag_name"] == args.tag), None)
        if target is None:
            raise UpstreamError(f"{args.tag} is not a stable release of {owner_repo}")
    else:
        target = releases[0]

    path = checkpoint_path(root, entry)
    previous = read_json(path)
    prior_checkpoint = previous.get("checkpoint") or {}
    prior_tag = prior_checkpoint.get("tag")

    # Repair refs before any early return so a fresh clone can still analyze.
    fetch_checkpoint(remote, slug, target["tag_name"])
    range_from = previous.get("source_range", {}).get("from_exclusive")
    for tag in {prior_tag, range_from} - {None, ""}:
        fetch_checkpoint(remote, slug, tag)

    if prior_tag == target["tag_name"] and not args.force:
        print(f"checkpoint already at {prior_tag} — no upstream delta")
        print(f"refs present: {', '.join(sorted({t for t in (prior_tag, range_from) if t}))}")
        return 0

    target_commit = release_commit(owner_repo, target["tag_name"])

    published = {r["tag_name"]: r for r in releases}
    in_range = []
    if prior_tag and prior_tag in published:
        floor = published[prior_tag]["published_at"]
        in_range = [
            r
            for r in releases
            if floor < (r.get("published_at") or "") <= (target.get("published_at") or "")
        ]
    else:
        in_range = [target]
    in_range.sort(key=lambda r: r.get("published_at") or "")

    priors = list(previous.get("prior_checkpoints", []))
    if prior_checkpoint and prior_tag != target["tag_name"]:
        priors.append(
            {
                "tag": prior_tag,
                "commit": prior_checkpoint.get("commit"),
                "ref": prior_checkpoint.get("ref"),
                "role": "prior-checkpoint",
            }
        )

    payload = {
        "schema_version": previous.get("schema_version", 1),
        "upstream": {"repository": entry["repository"], "remote": remote},
        "scope_policy": previous.get(
            "scope_policy",
            {"strategy": "targeted-semantic-ports", "include": [], "exclude": []},
        ),
        "checkpoint": {
            "tag": target["tag_name"],
            "commit": target_commit,
            "ref": ref_for(slug, target["tag_name"]),
            "published_at": target.get("published_at"),
            "url": target.get("html_url"),
            "target_commitish": target.get("target_commitish"),
            "checked_at": now_iso(),
        },
        "local_baseline": local_baseline(),
        "prior_checkpoints": priors,
        "source_range": {
            "from_exclusive": prior_tag,
            "to_inclusive": target["tag_name"],
        },
        "releases": [
            {
                "tag": r["tag_name"],
                "commit": release_commit(owner_repo, r["tag_name"]),
                "published_at": r.get("published_at"),
                "url": r.get("html_url"),
            }
            for r in in_range
        ],
        "generated_at": now_iso(),
    }
    write_json(path, payload)

    print(f"slug           : {slug}")
    print(f"previous       : {prior_tag or '(none)'}")
    print(f"checkpoint     : {target['tag_name']} {target_commit[:12]}")
    print(f"ref            : {payload['checkpoint']['ref']}")
    print(f"releases in range: {len(payload['releases'])}")
    print(f"local baseline : {payload['local_baseline']['commit'][:12]} ({payload['local_baseline']['branch']})")
    print(f"written        : {path.relative_to(root)}")
    if prior_tag:
        print(f"next: upstream_gap.py --slug {slug}")
    else:
        print("first checkpoint recorded — no prior range to analyze")
    return 0


def cmd_policy(root: Path, args: argparse.Namespace) -> int:
    registry = load_registry(root)
    entry = find_upstream(registry, args.slug)
    path = checkpoint_path(root, entry)
    payload = read_json(path)
    if not payload:
        raise UpstreamError(f"no checkpoint at {path} — run sync first")
    policy = payload.setdefault(
        "scope_policy", {"strategy": "targeted-semantic-ports", "include": [], "exclude": []}
    )
    if args.strategy:
        policy["strategy"] = args.strategy
    if args.include:
        policy["include"] = args.include
    if args.exclude:
        policy["exclude"] = args.exclude
    payload["generated_at"] = now_iso()
    write_json(path, payload)
    print(json.dumps(policy, indent=1))
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("list", help="show registered upstreams")

    p_add = sub.add_parser("add", help="register an upstream repository")
    p_add.add_argument("--slug", required=True)
    p_add.add_argument("--repo", required=True, help="owner/name on GitHub")
    p_add.add_argument("--remote", help="git remote name (default: slug)")

    p_sync = sub.add_parser("sync", help="advance the checkpoint to a release")
    p_sync.add_argument("--slug")
    p_sync.add_argument("--tag", help="pin a specific release instead of the latest")
    p_sync.add_argument("--force", action="store_true", help="rewrite even if the tag is unchanged")

    p_policy = sub.add_parser("policy", help="record approved scope policy")
    p_policy.add_argument("--slug")
    p_policy.add_argument("--strategy")
    p_policy.add_argument("--include", nargs="*")
    p_policy.add_argument("--exclude", nargs="*")

    args = parser.parse_args()
    root = repo_root()
    handlers = {"list": cmd_list, "add": cmd_add, "sync": cmd_sync, "policy": cmd_policy}
    try:
        return handlers[args.command](root, args)
    except UpstreamError as exc:
        fail(str(exc))


if __name__ == "__main__":
    sys.exit(main())
