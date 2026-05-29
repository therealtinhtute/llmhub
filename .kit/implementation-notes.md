# Implementation Notes

## 2026-05-29 — release-pipeline

### T1: .goreleaser.yml — binary format
- GoReleaser v2.16.0 requires Go >= 1.26.3 but go.mod has 1.26.0; GoReleaser auto-switches toolchain. No code change needed — the workflow's `setup-go` will install whatever go.mod declares and GoReleaser will upgrade the toolchain at runtime.
- `formats: [binary]` with `name_template: 'llmhub-{{ .Os }}-{{ .Arch }}'` produces bare executables named `llmhub-linux-amd64`, `llmhub-linux-arm64`, etc. Windows gets `.exe` appended automatically by GoReleaser.
- Removed `files:` (can't bundle with binary format). `config.example.yaml` is now fetched separately by the VPS installer from raw master.

### T2: .github/workflows/release.yml
- Used `goreleaser-action@v6` (latest stable v6 supports GoReleaser v2).
- `fetch-depth: 0` required so GoReleaser can read full git history for changelog generation.
- Bun setup step placed BEFORE goreleaser-action so the `before` hooks (web build + embed) succeed in CI.

## 2026-05-29 — vps-installer

### scripts/install.sh
- Combined Wave 1 + Wave 2 into one coherent script (no artificial split — the binary install and service setup are a single logical flow).
- `sed -i` used to rewrite `auth-dir` in the fetched config.example.yaml before installing it. POSIX `sed -i` requires no extension arg on Linux (GNU sed), but macOS would need `sed -i ''`. Since the installer is Linux-only (per spec), this is safe.
- Idempotency: `id llmhub` check before `useradd`; config install only if absent; systemd unit overwrite is safe (declarative).
- `hostname -I` fallback to "SERVER_IP" string if unavailable (e.g., container without network).

## 2026-05-29 — docs-consistency

### README.md
- One-liner placed first as the recommended path; manual steps follow for users who want control.
- Removed all tar.gz references; `aarch64` remains only in the arch-detection case statement (maps input → `arm64` variable), which is correct.
- Management panel URL and config-editing guidance preserved inline with the one-liner section.

### Makefile
- Dropped `version` variable extraction (raw binaries don't encode version in the filename).
- Dropped `aarch64` mapping (now uses literal `arm64` from `go env GOARCH`).
- `install-latest` no longer creates a tmpdir or extracts archives — raw binary is chmod'd and installed directly.
