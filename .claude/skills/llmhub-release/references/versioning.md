# Versioning

llmhub release series is `v0.0.x`. Do not use `v6.10.x` (stale local tags from old series) or `v7.2.x` (upstream CLIProxyAPI).

## Why v6 is wrong

`git tag --sort=-v:refname | head` sorts all local tags by semver, giving `v6.10.9` > `v0.0.27` (`6 > 0`). Those `v6` tags exist locally but never became GitHub Releases. `gh release list --repo therealtinhtute/llmhub` shows `v0.0.27` as `Latest`.

## Correct source of truth

1. `gh api repos/therealtinhtute/llmhub/releases/latest --jq .tag_name` (preferred, one call)
2. `gh release list --repo therealtinhtute/llmhub --limit 1 --json tagName --jq '.[0].tagName'`
3. `git ls-remote --tags origin | grep -E 'v0\.0\.[0-9]+' | sort -V | tail -1 | sed 's|.*refs/tags/||'`

Never `git tag --sort` alone.

## Next version

- patch (default): `v0.0.N` → `v0.0.N+1` via `{{ incpatch .Version }}` in `.goreleaser.yml:15`
- minor/major only on explicit user request.

Verify after push: `gh release view v0.0.N --repo therealtinhtute/llmhub --json tagName,assets` must be 9 assets; `git ls-remote --tags origin | grep v0.0.N`.
