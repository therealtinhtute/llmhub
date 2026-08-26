# Release Workflow

Workflow `.github/workflows/release.yml` triggers on `push` tags `v*` and `workflow_dispatch` with `tag` input.

## Push-tag path (normal)

1. `git tag -a v0.0.N -m "v0.0.N: ..." <commit> && git push origin v0.0.N`
2. Workflow checks out tag, `goreleaser` builds 8 binaries (linux/darwin/windows/freebsd × amd64/arm64) with `ldflags -X 'main.Version={{.Version}}'`.
3. Creates draft `gh release create v0.0.N --draft`, uploads 8 binaries + `checksums.txt`, verifies `gh release view --json assets | length == 9`, downloads and re-verifies, then `gh release edit --draft=false`.

Typical duration 11-12m. Failure modes: `gh release create --draft` fails if a release with same tag already exists (often because `gh release create` was run manually and created 0-asset release). Delete the 0-asset release first: `gh release delete v0.0.N --repo therealtinhtute/llmhub --yes` (keep tag), then dispatch.

## Manual dispatch path (existing tag)

`gh workflow run "Release" --repo therealtinhtute/llmhub --ref master -f tag=v0.0.N`

Poll: `gh run list --repo therealtinhtute/llmhub --workflow "Release" --limit 5` and `gh run view <id> --json status,conclusion`.

## Never do

- `gh release create v0.0.N` manually — creates 0-asset release, bypasses `goreleaser`, and blocks the workflow's draft creation.
- `git tag --sort=-v:refname` for version — picks stale `v6`.

## Verify

`gh release view v0.0.N --repo therealtinhtute/llmhub --json tagName,assets --jq '.assets[].name'` must list `llmhub-linux-amd64`, `llmhub-linux-arm64`, `llmhub-darwin-amd64`, `llmhub-darwin-arm64`, `llmhub-freebsd-amd64`, `llmhub-freebsd-arm64`, `llmhub-windows-amd64.exe`, `llmhub-windows-arm64.exe`, `checksums.txt`.
`git ls-remote --tags origin | grep v0.0.N`
