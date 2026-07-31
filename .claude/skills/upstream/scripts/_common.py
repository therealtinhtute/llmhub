"""Shared helpers for the upstream parity skill scripts."""

from __future__ import annotations

import json
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

REGISTRY_RELPATH = "docs/upstream/registry.json"
REF_NAMESPACE = "refs/upstream-checkpoints"


class UpstreamError(RuntimeError):
    """Recoverable error with an operator-facing message."""


def fail(message: str) -> "NoReturn":  # type: ignore[valid-type]
    print(f"error: {message}", file=sys.stderr)
    raise SystemExit(1)


def run(args: list[str], *, check: bool = True, cwd: Path | None = None) -> str:
    proc = subprocess.run(
        args,
        cwd=str(cwd) if cwd else None,
        capture_output=True,
        text=True,
    )
    if check and proc.returncode != 0:
        raise UpstreamError(
            f"command failed ({proc.returncode}): {' '.join(args)}\n{proc.stderr.strip()}"
        )
    return proc.stdout


def repo_root() -> Path:
    try:
        out = run(["git", "rev-parse", "--show-toplevel"]).strip()
    except UpstreamError as exc:
        fail(f"not inside a git repository: {exc}")
    return Path(out)


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def read_json(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    try:
        with path.open(encoding="utf-8") as handle:
            return json.load(handle)
    except json.JSONDecodeError as exc:
        raise UpstreamError(f"{path} is not valid JSON: {exc}") from exc


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        json.dump(payload, handle, indent=1, ensure_ascii=False)
        handle.write("\n")


def registry_path(root: Path) -> Path:
    return root / REGISTRY_RELPATH


def load_registry(root: Path) -> dict[str, Any]:
    data = read_json(registry_path(root))
    if not data:
        return {"schema_version": 1, "upstreams": []}
    data.setdefault("upstreams", [])
    return data


def find_upstream(registry: dict[str, Any], slug: str | None) -> dict[str, Any]:
    entries = registry.get("upstreams", [])
    if not entries:
        raise UpstreamError(
            "no upstream registered — run: upstream_sync.py add --slug <slug> --repo <owner>/<name>"
        )
    if slug is None:
        if len(entries) > 1:
            names = ", ".join(e["slug"] for e in entries)
            raise UpstreamError(f"--slug is required; registered upstreams: {names}")
        return entries[0]
    for entry in entries:
        if entry.get("slug") == slug:
            return entry
    raise UpstreamError(f"unknown upstream slug: {slug}")


def checkpoint_path(root: Path, entry: dict[str, Any]) -> Path:
    rel = entry.get("checkpoint") or f"docs/upstream/{entry['slug']}-checkpoint.json"
    return root / rel


def ref_for(slug: str, tag: str) -> str:
    return f"{REF_NAMESPACE}/{slug}/{tag}"


def ref_exists(ref: str) -> bool:
    proc = subprocess.run(
        ["git", "rev-parse", "--verify", "--quiet", ref],
        capture_output=True,
        text=True,
    )
    return proc.returncode == 0


def object_exists(sha: str) -> bool:
    proc = subprocess.run(["git", "cat-file", "-e", f"{sha}^{{commit}}"], capture_output=True)
    return proc.returncode == 0


def map_path(path: str, path_map: dict[str, str]) -> str:
    """Rewrite an upstream path to its local equivalent using longest-prefix match."""
    best = ""
    for prefix in path_map:
        if path == prefix or path.startswith(prefix.rstrip("/") + "/"):
            if len(prefix) > len(best):
                best = prefix
    if not best:
        return path
    return path_map[best].rstrip("/") + path[len(best.rstrip("/")) :]
