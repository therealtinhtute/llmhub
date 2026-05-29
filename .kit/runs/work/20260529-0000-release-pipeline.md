# Work Run: release-pipeline

Date: 2026-05-29
Phase: release-pipeline
Mode: full --notes
Spec: .kit/planning/SPEC.md
Plan: .kit/planning/phases/release-pipeline/release-pipeline-PLAN.md

## Preflight
- SPEC: locked ✓
- ROADMAP: present ✓
- Phase CONTEXT: present ✓
- Phase PLAN: present ✓
- Allowed surfaces: `.goreleaser.yml`, `.github/workflows/release.yml`
- Forbidden: all other workflows, builds/before blocks, installer, README, Makefile
- Drift: none (clean working tree)
- Status: READY

## Wave 1 — T1: Convert .goreleaser.yml archives to raw binaries
- Status: DONE
- Changed: `formats: [tar.gz]` → `formats: [binary]`; `name_template` → `llmhub-{{ .Os }}-{{ .Arch }}`; removed `format_overrides`, `files`
- Verification: `make release-check` → "1 configuration file(s) validated"

## Wave 2 — T2: Add .github/workflows/release.yml
- Status: DONE
- Created: trigger `v*` tags, ubuntu-latest, checkout+go+bun+goreleaser-action v6, `GITHUB_TOKEN`
- Verification: YAML parsed — 4 steps confirmed, trigger/permissions/job correct

## Phase 1: COMPLETE

---

# Phase 2: vps-installer

## Wave 1+2 — T1+T2: scripts/install.sh (combined)
- Status: DONE
- Created: POSIX sh, idempotent, full-deploy (binary + user + dirs + config + systemd)
- Verification: `sh -n scripts/install.sh` → POSIX parse OK

---

# Phase 3: docs-consistency

## Wave 1 — T1: Rewrite README Installation
- Status: DONE
- Changed: one-liner primary, raw-binary manual steps (`llmhub-linux-{arch}`), systemd reframed as "installer does this"
- Verification: `grep 'tar.gz\|aarch64' README.md` → only arch detection case (correct), no tar.gz

## Wave 2 — T2: Fix Makefile download/install targets
- Status: DONE
- Changed: `download-latest` / `install-latest` → raw `llmhub-{os}-{arch}` naming, no tar/zip extraction
- Verification: `make -n download-latest` / `make -n install-latest` → asset naming correct, no tar logic

---

# All Phases: COMPLETE
