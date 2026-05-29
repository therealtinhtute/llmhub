# Plan: release-pipeline

Phase: release-pipeline
Status: ready
Wave Count: 2
Execution Owner: work
Updated At: 2026-05-29

## Goal
Publish raw per-arch binaries (`llmhub-{os}-{arch}`) on tag push; keep web-embed hooks and checksums.

## Inputs
- `.goreleaser.yml` (current tar.gz config)
- `go.mod` (Go version for CI setup)
- `Makefile` `release-check` / `release-snapshot` targets

## Wave 1
### T1 — Convert `.goreleaser.yml` archives to raw binaries
- type: implementation
- inputs:
  - `.goreleaser.yml`
- touches:
  - `.goreleaser.yml` `archives` block + `name_template`
- avoid:
  - `builds` block, `before` hooks, platform matrix, `checksum` block (keep all unchanged)
- steps:
  1. In `archives:`, replace `formats: [tar.gz]` with `formats: [binary]`.
  2. Set `name_template` to `llmhub-{{ .Os }}-{{ .Arch }}` (drop the `aarch64` conditional so arm64 stays literal).
  3. Add `format_overrides` only if needed for Windows `.exe`; confirm GoReleaser appends `.exe` to the binary automatically — if so, no extra naming needed; the binary's own `.exe` is preserved.
  4. Remove the `files:` list (LICENSE/README/config) — raw binaries cannot bundle files.
  5. Leave `checksum.name_template: checksums.txt` as-is.
- expected outputs:
  - `.goreleaser.yml` producing raw binaries named `llmhub-linux-amd64`, `llmhub-linux-arm64`, `llmhub-darwin-{amd64,arm64}`, `llmhub-freebsd-{amd64,arm64}`, `llmhub-windows-{amd64,arm64}.exe`
- verification:
  - `make release-check` exits 0
  - `make release-snapshot` then `ls dist/` shows raw `llmhub-*` files (no `.tar.gz`); run `./dist/.../llmhub -h` on the linux build
- stop if:
  - `formats: [binary]` is rejected by GoReleaser v2.16.0 (escalate — do not silently keep tar.gz)
- escalate to:
  - plan phase / brainstorm refine

## Wave 2
### T2 — Add `.github/workflows/release.yml`
- type: implementation
- inputs:
  - finalized `.goreleaser.yml` from T1
  - `go.mod` Go version
- touches:
  - `.github/workflows/release.yml` (new)
- avoid:
  - any other workflow file (no docker/PR-guard/agents-md/retarget)
- steps:
  1. Trigger: `on: push: tags: ['v*']`.
  2. Single job on `ubuntu-latest` with `permissions: contents: write`.
  3. Steps: `actions/checkout` with `fetch-depth: 0`; `actions/setup-go` reading version from `go.mod`; `oven-sh/setup-bun`; `goreleaser/goreleaser-action` with `version: v2.16.0`, `args: release --clean`, env `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`.
  4. Do not add lint/test/docker steps — build+release only.
- expected outputs:
  - `.github/workflows/release.yml` that, on a `v*` tag, builds and attaches raw binaries + checksums to a GitHub Release
- verification:
  - `actionlint .github/workflows/release.yml` (if available) or YAML parse check; manual review that step versions exist and `GITHUB_TOKEN` is wired
  - confirm GoReleaser version matches Makefile's `GORELEASER_VERSION` (v2.16.0)
- stop if:
  - go.mod has no resolvable version for setup-go, or Bun setup conflicts with embed hooks
- escalate to:
  - user clarification (CI secrets/permissions) | plan phase

## Risks / Watch-fors
- arm64 must serialize as `arm64`, not `aarch64` — Phases 2 & 3 hard-code this contract.
- Bun setup must come before GoReleaser so the `before` embed hook succeeds in CI.
- `release --clean` wipes `dist/`; ensure no needed artifacts live there in CI.
