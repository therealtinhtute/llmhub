# Agent Instructions

Add project-specific agent instructions here.

<!-- ZHARNESS:BEGIN -->
## Harness

Run `zharness --version`, then `zharness preflight <stage> [--mode <mode>] --json` for every workflow skill invocation. Follow a returned stop and recovery exactly.

Read `docs/WORKFLOW.md`, then only the returned stage playbook and the repository material relevant to the requested outcome — start that search at `docs/README.md`, this repository's authored documentation map; if it is absent, proceed without it, which is not an error. Repository docs, code, tests, and observable behavior are authoritative; the database is a lifecycle ledger and recovery index.

Read-only and bounded work may use reduced mode and must not mutate harness state. Durable planning, full execution, full checks, and durable handoffs require an initialized database. Claim completion only with executable or observable evidence.
<!-- ZHARNESS:END -->

## Release

- llmhub release series is `v0.0.x` (see `gh release list --repo therealtinhtute/llmhub`). The `v6.10.x` tags are stale local tags from an old series — never use them.
- Next version = `incpatch` of latest GitHub release: `gh api repos/therealtinhtute/llmhub/releases/latest --jq .tag_name` or `gh release list --limit 1` or `git ls-remote --tags origin | grep -E 'v0\.0\.[0-9]+' | sort -V | tail -1`. Never use `git tag --sort=-v:refname` alone — it picks the stale `v6` series (`6 > 0`).
- Tag must be annotated and pushed before release: `git tag -a v0.0.N -m "v0.0.N: ..." <commit> && git push origin v0.0.N`; workflow `.github/workflows/release.yml` triggers on `v*` and runs `goreleaser` with `ldflags -X 'main.Version={{.Version}}'`.
- Verify with `gh release view v0.0.N --repo therealtinhtute/llmhub --json tagName` and `git ls-remote --tags origin | grep v0.0.N`.
