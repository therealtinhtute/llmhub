# Check Report: GitHub binary release pipeline + VPS installer

Date: 2026-05-29 06:47
Spec: .kit/planning/SPEC.md
Run: .kit/runs/work/20260529-0000-release-pipeline.md
Depth: Standard (182 lines, 5 files)

## Scope Classification
**Label:** ✅ on target

Changed files:
- `.goreleaser.yml` — raw binary format (R3)
- `.github/workflows/release.yml` — new CI workflow (R1, R2)
- `scripts/install.sh` — new VPS installer (R7–R13)
- `README.md` — one-liner + raw-binary docs (R14)
- `Makefile` — download/install targets updated (R15)

All files trace to SPEC requirements. No drift.

## Artifact Alignment
**Label:** ✅ aligned

- Phase 1 (release-pipeline): `.goreleaser.yml`, `.github/workflows/release.yml` — both delivered, verified via `make release-check` + YAML parse
- Phase 2 (vps-installer): `scripts/install.sh` — delivered, verified via `sh -n`
- Phase 3 (docs-consistency): `README.md`, `Makefile` — both updated, verified via grep (no tar.gz/aarch64 in wrong places) + `make -n` dry run

Run log shows all tasks DONE with verification evidence. No phase boundary violations.

## Gate Results

| Check | Result | Evidence |
|-------|--------|----------|
| Tests | ✅ PASS | `go test ./test/... ./sdk/... ./internal/...` — all existing suites pass; no tests cover changed surfaces (CI/docs/scripts) |
| Build | ✅ PASS | `go build ./cmd/server/` → BUILD: OK |
| Lint | ✅ PASS | `make release-check` → "1 configuration file(s) validated"; `sh -n scripts/install.sh` → POSIX: OK |

**Gate verdict:** ✅ PASS

## Code Review

### Security
- ✅ No hardcoded secrets (only `${{ secrets.GITHUB_TOKEN }}` reference in workflow)
- ✅ No command injection vectors in installer (all curl/sed use static URLs or quoted variables)
- ✅ Installer requires root check (line 15–18)
- ✅ Config installed with restrictive perms (0640, group llmhub)

### Correctness
- ✅ Asset naming consistent across all 5 files: `llmhub-{os}-{arch}` (`.exe` auto-appended by GoReleaser on Windows per implementation notes line 7)
- ✅ Installer idempotency: `id llmhub` check before useradd; config install only if absent; systemd unit overwrite is safe (declarative)
- ✅ `sed -i` Linux-only (GNU sed) — safe per spec (installer is Linux-only, R7)
- ✅ Workflow step order: Bun setup BEFORE goreleaser so embed hooks succeed (implementation notes line 13)
- ✅ `fetch-depth: 0` present for changelog generation

### Architecture
- ✅ Minimal workflow scope (build+release only, no docker/PR-guard revival per R6)
- ✅ Raw binary format drops file bundling; installer fetches config separately from raw master (implementation notes line 8)
- ✅ README structure: one-liner primary, manual fallback, systemd reframed as "installer does this"

### Code Quality
- ✅ POSIX sh compliance (no bash-isms)
- ✅ Error handling: `set -eu`, unsupported arch exits with clear message
- ✅ User feedback: progress messages, final status + URLs printed

## Pattern Completeness
Checked for sibling instances of the asset-naming pattern across the repo:
- ✅ All references to old tar.gz naming removed (verified via `grep 'tar.gz' README.md` → 0 matches in Installation section)
- ✅ `aarch64` mapping removed from Makefile; only remains in README arch-detection case (correct — maps user input `aarch64` → variable `arm64`)

## Knowledge Sync
**Doc debt:** none

No new invariants introduced. The asset-naming contract (`llmhub-{os}-{arch}`) is already captured in:
- `.goreleaser.yml` (source of truth)
- Implementation notes (line 7)
- SPEC R3

## Findings
**Blockers:** 0 critical, 0 major

**Minor observations:**
- 🟡 Go toolchain version skew: `go.mod` declares 1.26.0, but GoReleaser v2.16.0 requires >= 1.26.3. GoReleaser auto-switches at runtime (implementation notes line 6). Not a blocker — workflow will succeed — but consider bumping `go.mod` to 1.26.3+ for clarity.

## Autofix
- **safe_auto:** 0 applied
- **gated_auto:** 0 awaiting confirmation
- **manual:** 0
- **advisory:** 1 (go.mod version bump — optional)

## Recommendation
**APPROVE** — all requirements met, gate clean, no security/correctness issues, artifact alignment verified.

Optional follow-up: bump `go.mod` to `go 1.26.3` to match GoReleaser's requirement and avoid the auto-switch message in CI logs.
