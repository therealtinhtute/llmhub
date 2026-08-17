---
id: 01KZZVZETP16SJH4Y5157WSP0Z
type: plan
intake_id: 01KZZVZVDSKW58XYPNR1NK44V7
lane: high-risk
status: active
created: 2026-08-14
updated: 2026-08-15
---

# Plan: Panel-triggered binary self-update

## Outcome
- result: An operator updates a running LLMHub VPS deployment by pressing a button in the management panel. The unprivileged server downloads and stages a checksum-verified release binary; a root-run `ExecStartPre` step independently re-verifies it against GitHub's checksum manifest and swaps it in at restart, retaining one previous binary for recovery.
- success_signals:
  - `llmhub version --short`, `llmhub update --check`, `llmhub update`, and `llmhub update rollback` dispatch before runtime configuration or Postgres loading and return deterministic exit codes.
  - A supported Linux or macOS release binary discovers the latest stable GitHub Release, selects the exact existing `llmhub-{os}-{arch}` asset, refuses malformed versions and downgrades, and reports an already-current version without modifying disk state.
  - Every candidate is downloaded through bounded HTTPS requests, matched against exactly one well-formed SHA-256 entry in the GoReleaser `checksums.txt` manifest, and rejected before staging when any check fails.
  - A verified candidate passes a bounded sanitized `version --short` probe, then is written to a staging directory inside `ReadWritePaths` without touching the installed target.
  - `llmhub apply-staged-update`, run as root by `ExecStartPre=+` before the service starts, re-fetches `checksums.txt` over HTTPS itself, re-hashes the staged binary, refuses a version not newer than the installed one, installs preserving mode 0755 root ownership, and retains `<target>.previous`.
  - An update whose new binary fails to reach a successful server start is reverted automatically to `<target>.previous` on the next start attempt, so a bad release cannot produce an unbounded restart loop.
  - The management panel exposes check and update actions behind the existing management-key auth; the update action stages, then triggers exactly one narrowly scoped `systemctl restart llmhub.service` through a wildcard-free sudoers rule.
  - The server process never writes `/usr/local/bin`, never runs migrations as part of an update, and never elevates privileges beyond that single restart invocation.
  - `make download-latest` and `make install-latest` no longer fetch or replace release binaries; both fail with an explicit deprecation result directing operators to the panel or `llmhub update`.
  - The release workflow creates a draft, verifies the complete binary and checksum asset matrix from the remote draft, and publishes only after verification.
  - Focused updater tests, `go test ./...`, `go vet ./...`, release configuration validation, snapshot release build, frontend embed build, backend build, and diff hygiene checks pass.

## Authority and Requirements
- authority:
  - Owner decision on 2026-08-15 selected checksum-only verification over minisign signing, one-click panel update with automatic restart, and systemd deployment through `scripts/install-local.sh`.
  - `scripts/install-local.sh:378` (`install -m 0755` into `/usr/local/bin`), `:430-442` (`User=`, `NoNewPrivileges=true`, `ProtectSystem=full`, `ReadWritePaths=${DATA_DIR} ${LOG_DIR} ${CONFIG_DIR}`): the running service cannot write its own binary, which forces the stage-then-privileged-apply split.
  - systemd `ExecStartPre=+` prefix: runs the listed command with full privileges, ignoring `User=`, and runs after the old process has stopped during `systemctl restart`, so the installed binary is not busy at swap time.
  - `cmd/server/main.go`, `cmd/server/db_runtime.go`, `internal/buildinfo/buildinfo.go`: current early positional command dispatch and linker-injected version metadata.
  - `internal/api/handlers/management/config_basic.go:21,39` and `internal/api/server.go:662`: existing `GET /v0/management/latest-version` handler and its GitHub latest-release URL, timeout, and user agent.
  - `web/src/pages/SystemPage.tsx:232` `handleVersionCheck` and `web/src/services/api/version.ts:8`: the existing panel version-check button that this plan extends rather than replaces.
  - `.goreleaser.yml`, `.github/workflows/release.yml`, `Makefile`: current release tag validation, bare-binary asset names, SHA-256 `checksums.txt`, build metadata injection, and snapshot gate.
  - Repository `CLAUDE.md` and `web/CLAUDE.md`: minimal surgical changes, Postgres-only runtime configuration, no frontend test files under `web/`, i18n keys in both `en.json` and `vi.json`.
- requirements:
  - R1 [accepted]: Add `version` and self-update positional commands dispatched before the startup banner, server flag parsing, Postgres loading, configuration access, or service startup. | source: `cmd/server/main.go`; `cmd/server/db_runtime.go`.
  - R2 [accepted]: Reuse `internal/buildinfo` and existing linker flags as the only version authority; `version --short` emits a normalized machine-readable release version with no banner or logging. | source: `internal/buildinfo/buildinfo.go`; `.goreleaser.yml`.
  - R3 [accepted]: Use the public GitHub Releases latest endpoint and existing `llmhub-{os}-{arch}[.exe]` naming, require an exact supported asset match, and treat all remote metadata, names, sizes, URLs, and bodies as untrusted input. | source: `internal/api/handlers/management/config_basic.go`; `.goreleaser.yml`.
  - R4 [accepted]: Validate normalized semantic versions, refuse malformed current/latest versions, refuse downgrades and same-version replacement, and exclude draft/prerelease from the stable channel. Development builds must not silently replace themselves. | source: owner-approved stable-channel behavior; existing SemVer tag gate in `Makefile`.
  - R5 [accepted]: All release metadata, checksum, and binary requests use HTTPS with explicit context/timeout handling, status checks, response-size limits, HTTPS-only redirect constraints, and deterministic cleanup. | source: owner-approved failure behavior; current bounded latest-version handler.
  - R6 [accepted]: Verify the downloaded binary against its raw SHA-256 digest from the release's `checksums.txt`, requiring exactly one well-formed entry for the selected asset. A mismatch leaves all installed and staged state unchanged. | source: existing `.goreleaser.yml` checksum asset.
  - R7 [accepted]: Write the verified candidate only into `${DATA_DIR}/update/` — inside the unit's `ReadWritePaths` — as `llmhub.staged` plus a `staged.json` recording version and digest. The unprivileged server never writes the installed target, `/usr/local/bin`, or any root-owned path. | source: `scripts/install-local.sh` unit boundary.
  - R8 [accepted]: Before marking a candidate staged, run it as a bounded subprocess with `version --short` in a sanitized environment and isolated temporary working directory, requiring exact normalized stdout and no runtime configuration, database, service startup, or writes outside the staging directory. Package initialization itself is not claimed pure; the contract covers observable command behavior. | source: `cmd/server/main.go` initialization order.
  - R9 [accepted]: `apply-staged-update` runs as root through `ExecStartPre=+`. It must not trust `staged.json` for authenticity: it independently re-fetches `checksums.txt` for the staged version over HTTPS, re-hashes the staged binary, and requires the staged version to be strictly newer than the installed one. Any failure — including network failure — skips the update and lets the service start on the current binary. | source: owner decision to drop signatures; `${DATA_DIR}` is service-writable, so an RCE in the server could otherwise stage an arbitrary binary for root installation.
  - R10 [accepted]: Before swapping, the root step writes a marker to a root-owned path outside `${DATA_DIR}`; the server clears it on successful start. A marker still present at the next apply means the previous swap never reached a healthy start, so the root step reverts to `<target>.previous` instead of applying anything new. | source: owner-approved boot-loop protection; `Restart=` in the generated unit.
  - R11 [accepted]: The updater never runs migrations, never mutates configuration, never elevates privileges except the single sudoers-permitted `systemctl restart llmhub.service`, and never updates from a background timer. | source: owner-approved trigger policy; `scripts/install-local.sh` service permissions.
  - R12 [accepted]: Enable self-update only for Linux and macOS on amd64 and arm64; retain existing Windows and FreeBSD release builds but return an explicit unsupported-platform result there. | source: owner-approved platform scope; current GoReleaser matrix.
  - R13 [accepted]: Expose `POST /v0/management/self-update` (discover, download, verify, probe, stage) and `POST /v0/management/self-update/apply` (trigger restart) behind the existing management-key middleware, returning typed outcomes the panel renders without exposing remote response bodies. | source: `internal/api/server.go` management route group.
  - R14 [accepted]: Install a sudoers drop-in permitting exactly `/usr/bin/systemctl restart llmhub.service` for the service user with `NOPASSWD:` and no wildcard, glob, or argument variability. | source: owner-approved one-click restart; sudoers argument-matching behavior.
  - R15 [accepted]: Add deterministic Go coverage for version normalization/comparison, release parsing and asset selection, bounded HTTP failures, checksum rejection, probe failure, staging paths, root-apply re-verification and downgrade refusal, boot-loop revert, and unsupported platforms, without touching the running test binary. | source: repository testing rules.
  - R16 [accepted]: Final verification includes focused updater tests, `go test ./...`, `go vet ./...`, `make release-check`, `make release-snapshot`, `make build-web`, `make build`, `bun run type-check`, `bun run lint`, `git diff --check`, and the expected nonzero/no-network/no-install behavior of the deprecated Makefile targets. | source: repository `CLAUDE.md`; existing Makefile gates.
  - R17 [accepted]: Remove unsigned mutation from `make download-latest` and `make install-latest`: no network request, no filesystem replacement, nonzero deprecation result directing operators to the panel or `llmhub update`. `make install-local` remains the local bootstrap path. | source: `Makefile`.
  - R18 [accepted]: The release workflow creates a draft release, verifies the complete remote binary and `checksums.txt` asset matrix, and publishes only after verification. It never overwrites assets under an existing tag and requires no signing secrets. | source: `.github/workflows/release.yml`; owner decision to drop signing.

## Non-goals
- NG1: Cryptographic signatures of any kind — minisign, cosign/Sigstore, GPG, TUF, or transparency logs. Authenticity rests on HTTPS to GitHub plus the checksum manifest, re-verified independently by the root apply step.
- NG2: Automatic update during server startup, periodic background checks, or unattended timer-driven updates. Update is always an explicit operator action.
- NG3: Automatic privilege elevation beyond the single sudoers-permitted restart; no package-manager integration, rolling fleet deployment, or zero-downtime process replacement.
- NG4: Windows or FreeBSD executable replacement; their release artifacts remain unchanged.
- NG5: A custom update service, private-release authentication, embedded GitHub credentials, update channels, prerelease opt-in, or arbitrary-version selection.
- NG6: Binary delta patching, archive extraction, multi-file application updates, or updates to package-managed and container deployments.
- NG7: Rollback of Postgres state, schema migrations, or any non-binary side effect. `<target>.previous` restores bytes only.
- NG8: Frontend test files under `web/`; panel changes are verified through type-check, lint, and production build.
- NG9: Refactoring unrelated CLI startup, release tooling, installer behavior, logging, or provider/runtime code, beyond containing the two deprecated Makefile targets.

## Approach and Risks
- approach: Four dependency-ordered phases, each independently mergeable. First strip the abandoned signing work from the release pipeline while keeping its draft-first publication and remote asset verification. Next build a testable `internal/updater` package that discovers releases, verifies checksums, probes the candidate, and stages it — exposed through early CLI commands. Then add the root-run `apply-staged-update` subcommand and the installer unit change that make a staged candidate actually take effect at restart, with boot-loop revert. Finally add the two management endpoints, the panel button, the sudoers drop-in, and the full verification gate.
- constraints:
  - The update source is the public GitHub Releases endpoint for `therealtinhtute/llmhub`. Runtime database configuration and the management proxy setting are unavailable to early CLI commands and to the root apply step, so both use a dedicated client with standard environment proxy behavior.
  - `internal/buildinfo` and current linker flags remain the only version source. No new config key, database row, or environment setting controls updater behavior.
  - Staging lives at `${DATA_DIR}/update/`; the boot-loop marker lives at a root-owned path outside `${DATA_DIR}` so the service user cannot forge or clear it. Both are created by `scripts/install-local.sh`.
  - The root apply step is a Go subcommand of the currently installed binary, which is root-owned and not service-writable. It is therefore trusted to perform its own verification.
  - `systemctl restart` stops the old process before `ExecStartPre` runs, so the installed binary is not busy and a plain rename suffices. No in-place replacement library is required.
  - Removal of superseded files uses `trash`, never `rm`.
  - Tests use temporary target files and injected seams and must never replace the running test or development executable.
  - Self-update support is Linux and macOS on amd64/arm64. Windows and FreeBSD binaries continue to build but reject updater entry points before mutation.
  - No frontend test files are added under `web/`; i18n keys are added to both `en.json` and `vi.json`.
- decisions:
  - Drop minisign entirely and delete the uncommitted signing work. Rationale: the signing key would have lived in GitHub Secrets, so it defended only against a partial GitHub compromise; the owner accepted that residual risk in exchange for removing key provisioning, rotation, and tooling from the project.
  - Keep the draft-first release workflow and the asset-matrix verifier already written for the signing effort, with signature checks removed. Rationale: they catch missing or renamed assets before publication and are independent of signing.
  - Split update into unprivileged stage and privileged apply rather than letting the server replace its own binary. Rationale: `ProtectSystem=full` and `ReadWritePaths` make direct replacement impossible, and preserving that boundary keeps a server RCE from becoming a root-installed binary.
  - Have the root step re-fetch `checksums.txt` itself rather than trusting `staged.json`. Rationale: `${DATA_DIR}` is service-writable, so a compromised server could otherwise stage an arbitrary binary and a matching digest; a fresh HTTPS fetch as root cannot be forged from inside the service.
  - Do not add `github.com/minio/selfupdate`. Rationale: it exists to replace a *running* executable; systemd has already stopped the process before the swap, so `os.Rename` is sufficient and the dependency is unnecessary.
  - Renumber requirements from the previous revision. Rationale: signing requirements were deleted and privilege-boundary requirements added; historical `Progress` entries referencing old numbers remain valid as a record of superseded work.
- rejected_alternatives:
  - Minisign, cosign/Sigstore, or GPG signing: equivalent protection while the key or workflow identity lives in GitHub, at the cost of key provisioning, rotation procedure, and release tooling the owner declined.
  - CI push over SSH from GitHub Actions: removes verification entirely and is simpler, but loses on-demand check/update from the panel and requires exposing SSH to runner networks.
  - Letting the service user own `/usr/local/bin/llmhub`: makes one-click trivial but turns any RCE in an internet-facing proxy holding OAuth tokens into persistent binary replacement.
  - Having the root step download the whole release itself: removes the staged-file trust problem but moves a multi-megabyte download into service startup and loses panel progress feedback.
  - Per-target advisory lock and fsync/pending-sibling power-loss ceremony from the previous revision: single-operator single-VPS deployment with a serialized apply at boot does not exhibit the concurrent-mutation or torn-transaction cases they defended against.
- risks:
  - risk: A GitHub account or token with release-write access publishes a malicious binary and matching `checksums.txt`; both the server and the root step verify it successfully.
    mitigation: None within this design — this is the risk the owner explicitly accepted by dropping signatures. Reduce exposure by keeping release publication to the tag workflow and enabling 2FA on the account.
    recovery: Restore `<target>.previous`, revoke the compromising credential, and delete the malicious release.
  - risk: A server compromise writes an arbitrary binary into `${DATA_DIR}/update/` and waits for a restart to have it installed as root.
    mitigation: The root step ignores `staged.json` for authenticity, re-fetches `checksums.txt` over HTTPS, re-hashes the staged file, and refuses any version not strictly newer than installed.
    recovery: Apply fails closed and the service starts on the current binary; the staged file is preserved for inspection rather than silently deleted.
  - risk: A genuine release starts far enough to pass `version --short` but crashes during real startup, and `Restart=` produces a boot loop on a broken binary.
    mitigation: The root step writes a root-owned marker before swapping and the server clears it after a successful start; a surviving marker at the next apply triggers automatic revert to `<target>.previous`.
    recovery: The service returns to the previous binary without operator action; the failed candidate remains staged for diagnosis.
  - risk: The sudoers rule is written with a wildcard or loose argument match, granting the service user broader root access than one restart.
    mitigation: Install an exact-command drop-in with no glob, validate with `visudo -c`, and assert in review that the rule contains no `*` and names the full unit.
    recovery: Remove the drop-in immediately; the panel degrades to "staged, restart manually" without further change.
  - risk: Remote metadata causes downgrade, duplicate-asset ambiguity, unbounded memory use, redirect abuse, or partial-body acceptance.
    mitigation: Normalize strict SemVer, reject development/malformed/same/older versions, require exactly one asset and one checksum entry, enforce HTTPS on redirects, set separate metadata and binary timeouts, validate declared size, and detect limit overflow.
    recovery: Fail closed before staging; report the rejected field or status without logging response bodies beyond a small diagnostic limit.
  - risk: `ExecStartPre=+` misbehaves — wrong ordering relative to the existing `init-db-from-env` step, or a nonzero exit blocking startup entirely.
    mitigation: Place `apply-staged-update` as the first `ExecStartPre`, ahead of `init-db-from-env`, so migrations run on the binary that will serve; make the subcommand exit 0 on every skip path so it can never prevent the service from starting.
    recovery: Revert the unit change with `install-local.sh`; the binary swap simply stops happening while the server keeps running.
  - risk: Anyone holding the management key can now trigger a root binary swap, not just read configuration.
    mitigation: Keep both endpoints behind the existing management-key middleware and document that the key is now equivalent to binary-deployment authority.
    recovery: Rotate the management key; the staged/apply path cannot be reached without it.
  - risk: Binary rollback occurs after a release already applied an incompatible schema change.
    mitigation: The update path never runs migrations; `init-db-from-env` runs as a separate unit step, and documentation states that `.previous` restores binary bytes only.
    recovery: Use the service's separate database recovery procedure.
- stop_conditions:
  - The root apply step would trust `staged.json`, a `${DATA_DIR}`-resident digest, or any other service-writable input as its authenticity source.
  - A candidate would be executed or installed before its checksum is verified.
  - The sudoers rule would contain a wildcard, a variable argument, or any command other than the single unit restart.
  - `apply-staged-update` could exit nonzero on a skip path and prevent the service from starting.
  - The server process would require write access to `/usr/local/bin` or any path outside `ReadWritePaths`.
  - A test would replace the currently running binary, require a live provider or database, or depend on timing and network services rather than deterministic fixtures.
  - `make download-latest` or `make install-latest` still performs a network download or executable replacement after the CLI exists.
  - Any verification command fails; retain the command and output and return to the owning phase rather than weakening the check.

## Phases and Verification
<!-- Phase and task definitions are immutable after this plan is approved and locked. Append-only Progress is the sole task execution-status source. Phase lifecycle status mirrors DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. -->
- planning_status: planned
- phases:
  - phase_slug: release-pipeline-cleanup
    story_id: 01M02G1B5KY663MPTSFH0KAS2P
    status: checked
    goal: Remove the abandoned signing work from the release pipeline while keeping draft-first publication and remote asset-matrix verification.
    depends_on: none
    allowed_surfaces:
      - .goreleaser.yml
      - .github/workflows/release.yml
      - Makefile release targets and help text
      - scripts/verify-release-assets.sh
      - scripts/sign-release-asset/, scripts/sign-release-asset.sh, scripts/verify-minisign-asset.sh (removal only)
      - docs/release-signing.md, docs/README.md (removal and link cleanup only)
    avoided_surfaces:
      - cmd/server, internal/, and web/ source
      - existing binary names, build matrix, checksum filename, and linker metadata contract
      - publishing a real tag or release
    waves:
      - wave: 1
        tasks:
          - task: Remove the uncommitted minisign work — `scripts/sign-release-asset/`, `scripts/sign-release-asset.sh`, `scripts/verify-minisign-asset.sh`, and `docs/release-signing.md` — with `trash`, and drop the `docs/README.md` link to the removed document.
            requirements: [R18]
            dependencies: none
            expected_output: No minisign tooling, nested signing module, or signing documentation remains in the worktree; `docs/README.md` has no dangling link.
            checks:
              - test ! -e scripts/sign-release-asset && test ! -e scripts/sign-release-asset.sh && test ! -e scripts/verify-minisign-asset.sh && test ! -e docs/release-signing.md
              - grep -c "release-signing" docs/README.md || true
              - git status --short
            stop_conditions: Stop if removal would use `rm` rather than `trash`, or would delete a file the owner authored outside this initiative.
            escalation: Preserve any unrelated worktree change and report it rather than removing it.
          - task: Strip signing from `.goreleaser.yml` (remove `binary_signs`) and from `.github/workflows/release.yml` (remove minisign download/verification, key materialization, passphrase handling, `transition_public_key` input, and the `release` environment requirement if it exists only for signing secrets), keeping draft creation, remote asset download, verification, and publish-after-verification.
            requirements: [R18]
            dependencies: task: Remove the uncommitted minisign work — `scripts/sign-release-asset/`, `scripts/sign-release-asset.sh`, `scripts/verify-minisign-asset.sh`, and `docs/release-signing.md` — with `trash`, and drop the `docs/README.md` link to the removed document.
            expected_output: The tag workflow builds, drafts, verifies the remote binary and checksum matrix, and publishes, with no signing secret referenced anywhere.
            checks:
              - go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/release.yml
              - grep -ci "minisign\|MINISIGN_" .github/workflows/release.yml .goreleaser.yml || true
              - make release-check
              - git diff --check
            stop_conditions: Stop if the workflow would publish before verification, overwrite assets under an existing tag, or lose the draft-first behavior.
            escalation: Keep the existing verified workflow structure and remove only signing-specific steps.
          - task: Reduce `scripts/verify-release-assets.sh` to a checksum-and-completeness verifier for the eight release binaries plus `checksums.txt`, removing the minisign CLI dependency and public-key argument, and remove `MINISIGN_BIN` and the `release-signed-snapshot` target from `Makefile`.
            requirements: [R17, R18]
            dependencies: task: Strip signing from `.goreleaser.yml` and from `.github/workflows/release.yml`, keeping draft creation, remote asset download, verification, and publish-after-verification.
            expected_output: `scripts/verify-release-assets.sh <dir>` verifies asset presence and SHA-256 agreement with no external tool beyond coreutils, and works on both a GoReleaser `dist/` layout and a flat downloaded-asset directory.
            checks:
              - sh -n scripts/verify-release-assets.sh
              - make release-snapshot
              - scripts/verify-release-assets.sh dist
              - make -n release-signed-snapshot 2>&1 | grep -q "No rule to make target" && echo removed
            stop_conditions: Stop if the verifier would accept a missing or renamed asset, or if removing the signed-snapshot target breaks an unrelated Makefile target.
            escalation: Keep the verifier's existing directory-layout handling and change only its signature-checking section.

  - phase_slug: self-update-engine
    story_id: 01M02G1FHEZPGWRJ121CR95FXC
    status: checked
    goal: Implement bounded release discovery, strict version and asset selection, SHA-256 verification, candidate probing, and staging, exposed through early CLI commands.
    depends_on: release-pipeline-cleanup
    allowed_surfaces:
      - go.mod and go.sum for the pinned semantic-version dependency
      - internal/updater/*.go and internal/updater/*_test.go (new)
      - cmd/server/main.go, cmd/server/self_update.go, cmd/server/self_update_test.go
      - Makefile deprecated-target containment and help text
    avoided_surfaces:
      - internal/api/handlers/management behavior until the panel phase
      - runtime config, Postgres stores, provider executors, and web source
      - scripts/install-local.sh and any root-executed path until the privileged-apply phase
      - live GitHub Releases and replacement of the running test or development executable
    waves:
      - wave: 1
        tasks:
          - task: Add `golang.org/x/mod/semver` and implement an injectable GitHub release client with production URL and user-agent defaults, environment proxy support, HTTPS-only redirect checks, status handling, separate metadata and binary timeouts, and strict response-size limits.
            requirements: [R3, R5, R12]
            dependencies: none
            expected_output: The updater retrieves bounded latest-release metadata and asset bytes through deterministic seams without importing Gin, runtime config, or management handlers.
            checks:
              - go test ./internal/updater/... -run 'Test(Client|LatestRelease|HTTP|Redirect|ResponseLimit)' -count=1
              - go list -m golang.org/x/mod
            stop_conditions: Stop if the production client accepts non-HTTPS release URLs or redirects, has an unbounded body path, or hides non-200 diagnostics.
            escalation: Keep endpoints and timeouts as private constants and expose only the smallest constructor seam `httptest` fixtures need.
          - task: Implement current and latest version normalization, no-development/no-downgrade/no-same-version decisions, supported-platform mapping, exact binary asset selection, and duplicate-safe parsing of the selected SHA-256 checksum entry.
            requirements: [R2, R3, R4, R6, R12]
            dependencies: task: Add `golang.org/x/mod/semver` and implement an injectable GitHub release client with production URL and user-agent defaults, environment proxy support, HTTPS-only redirect checks, status handling, separate metadata and binary timeouts, and strict response-size limits.
            expected_output: Stable releases map deterministically to `llmhub-{os}-{arch}[.exe]`; malformed or ambiguous metadata fails closed; development and unsupported builds cannot reach staging.
            checks:
              - go test ./internal/updater/... -run 'Test(Version|Asset|Platform|Checksum)' -count=1
              - Cover leading `v`, bare versions, prerelease metadata, malformed tags, duplicate assets and checksum entries, same version, downgrade, `dev`, Linux and macOS support, and Windows and FreeBSD rejection.
            stop_conditions: Stop if comparison becomes lexical, asset matching becomes fuzzy, a duplicate is silently accepted, or an unsupported platform proceeds past decision logic.
            escalation: Preserve the established asset contract and reject unknown release shapes rather than adding aliases or channels.
      - wave: 2
        tasks:
          - task: Implement download-verify-probe-stage: stream the asset under a size limit, verify its SHA-256 against the single matching `checksums.txt` entry, run a bounded sanitized `version --short` subprocess from an isolated temporary working directory requiring exact normalized stdout, then write `${DATA_DIR}/update/llmhub.staged` at mode 0755 and `${DATA_DIR}/update/staged.json` recording version and digest.
            requirements: [R5, R6, R7, R8]
            dependencies: wave: 1
            expected_output: Corrupted, truncated, oversized, wrong-version, or non-starting candidates fail before anything is staged and leave no executable residue; a valid candidate produces exactly the two staging files.
            checks:
              - go test ./internal/updater/... -run 'Test(Download|Checksum|Probe|Stage|StagedManifest)' -count=1
              - go test -race ./internal/updater/... -run 'Test(Download|Probe|Stage)' -count=1
              - Assert probe tests verify timeout, sanitized environment, exact stdout, isolated working directory, and absence of runtime config, database, or service calls.
            stop_conditions: Stop if a candidate executes before checksum verification, the probe writes outside the staging directory or reaches server startup, or staging touches any path outside `${DATA_DIR}/update/`.
            escalation: Narrow the engine API around explicit staging-directory and probe seams rather than adding a general updater framework.
          - task: Add `runVersion` and `runSelfUpdate` command functions with isolated `flag.FlagSet`s and injected engine and output dependencies, then move early positional dispatch ahead of the startup banner and Postgres path while preserving `init-db-from-env`, `migrate-local-to-db`, server flags, login modes, and normal banner output.
            requirements: [R1, R2, R4, R11, R12, R15]
            dependencies: task: Implement download-verify-probe-stage.
            expected_output: `version [--short]`, `update --check`, and `update` have deterministic usage, stdout and stderr, and 0/1/2 exit behavior; ordinary server startup is unchanged.
            checks:
              - go test ./cmd/server/... -run 'Test(VersionCommand|UpdateCommand|UpdateCheck|EarlyCommandDispatch|ExistingDBCommands|Usage)' -count=1
              - env -u PGSTORE_DSN -u pgstore_dsn go run ./cmd/server version --short
            stop_conditions: Stop if command parsing introduces a CLI framework, accepts downgrade or force flags, or an early command reaches `loadRuntimeConfigFromPostgres`.
            escalation: Match the existing `flag.NewFlagSet` pattern from the database commands and isolate a small `dispatchEarlyCommand(args)` function.
          - task: Contain `make download-latest` and `make install-latest`: remove their curl and install recipes and help entries and make each return a nonzero deprecation result before any network or filesystem work, directing operators to the panel or `llmhub update`. Leave `make install-local` unchanged.
            requirements: [R17]
            dependencies: task: Add `runVersion` and `runSelfUpdate` command functions and move early positional dispatch ahead of the startup banner and Postgres path.
            expected_output: No documented Makefile target fetches or replaces a release binary; invoking either deprecated target performs no network request or installation.
            checks:
              - make -n download-latest
              - make -n install-latest
              - Run both targets and assert a nonzero result with no `dist/downloads` or installed-target mutation.
            stop_conditions: Stop if either target still invokes curl, writes a downloaded release, or installs a release binary.
            escalation: Limit the change to these two targets and their help text.

  - phase_slug: privileged-apply
    story_id: 01M02G1FHP5AK6BKM2PXFZ8KW7
    status: checked
    goal: Make a staged candidate take effect at restart through a root-run apply subcommand that re-verifies it independently, retains one previous binary, and reverts automatically after a failed start.
    depends_on: self-update-engine
    allowed_surfaces:
      - internal/updater/apply*.go and tests (new)
      - cmd/server/self_update.go early dispatch for `apply-staged-update`
      - scripts/install-local.sh unit generation, directory creation, and rollback command
      - README.md operator update and recovery usage
    avoided_surfaces:
      - management handlers and web source until the panel phase
      - sudoers installation until the panel phase
      - provider, runtime, and Postgres code
    waves:
      - wave: 1
        tasks:
          - task: Implement `apply-staged-update` as an early-dispatch subcommand that refuses to run unless euid is 0, reads the staged version, re-fetches `checksums.txt` for that version over HTTPS with its own bounded client, re-hashes `${DATA_DIR}/update/llmhub.staged`, requires the staged version to be strictly newer than the installed binary's, then installs it over the resolved target preserving mode 0755 and root ownership, retaining the replaced binary as `<target>.previous`. Every skip and failure path exits 0 after logging, so startup is never blocked.
            requirements: [R6, R9, R11, R15]
            dependencies: none
            expected_output: A genuine staged candidate is installed at restart; a forged staged binary, a stale or downgraded version, or a network failure leaves the installed target untouched and the service starting normally.
            checks:
              - go test ./internal/updater/... -run 'Test(Apply|ApplyReverify|ApplyDowngrade|ApplyForged|ApplyNetworkFailure|ApplyNonRoot|ApplyExitCode)' -count=1
              - Assert every failure-path test observes exit code 0 and an unchanged temporary target file.
              - Assert apply tests use temporary target paths rather than `os.Executable()`.
            stop_conditions: Stop if apply trusts `staged.json` for authenticity, can exit nonzero on a skip path, crosses filesystems, changes target mode or ownership, or deletes the only runnable binary.
            escalation: Keep the apply path narrow and fail closed toward "start on the current binary" rather than attempting partial recovery.
          - task: Implement the boot-loop guard: before swapping, write a marker recording the outgoing version to a root-owned path outside `${DATA_DIR}`; have the server clear it once startup completes successfully; and have the next apply invocation, on finding a surviving marker, restore `<target>.previous` and refuse to apply anything new until the marker is cleared.
            requirements: [R10, R15]
            dependencies: task: Implement `apply-staged-update` as an early-dispatch subcommand.
            expected_output: A candidate that passes probe but crashes during real startup is reverted automatically on the following start attempt, and the failed candidate remains staged for inspection.
            checks:
              - go test ./internal/updater/... -run 'Test(BootMarker|MarkerRevert|MarkerCleared|MarkerUnwritable)' -count=1
              - go test -race ./internal/updater/... -run 'Test(BootMarker|MarkerRevert)' -count=1
            stop_conditions: Stop if the marker lives in a service-writable path, if a revert can delete the only runnable binary, or if a missing `<target>.previous` causes a nonzero exit.
            escalation: Prefer leaving the current binary in place and reporting the surviving paths over any destructive recovery.
      - wave: 2
        tasks:
          - task: Update `scripts/install-local.sh` to create `${DATA_DIR}/update/` owned by the service user and the root-owned marker directory, and to generate the unit with `ExecStartPre=+${INSTALL_DIR}/${BINARY} apply-staged-update` as the first `ExecStartPre`, ahead of the existing `init-db-from-env` step.
            requirements: [R9, R10, R11]
            dependencies: wave: 1
            expected_output: A freshly installed unit runs the privileged apply before migrations, so `init-db-from-env` and `ExecStart` both use the newly installed binary; the marker directory is not writable by the service user.
            checks:
              - sh -n scripts/install-local.sh
              - Generate the unit into a temporary directory and assert `ExecStartPre=+` precedes the `init-db-from-env` line and that `ReadWritePaths` still excludes the marker directory.
              - systemd-analyze verify on the generated unit file
            stop_conditions: Stop if the apply step is ordered after `init-db-from-env`, if the marker directory falls inside `ReadWritePaths`, or if `ProtectSystem=full` or `NoNewPrivileges=true` is weakened.
            escalation: Change only the unit's `ExecStartPre` block and directory creation; leave the rest of the installer untouched.
          - task: Implement `llmhub update rollback` for manual recovery from `<target>.previous`, and document in README the operator flow: check, update, restart, automatic revert behavior, manual rollback, Linux and macOS support, and the binary-only recovery boundary.
            requirements: [R11, R12, R15]
            dependencies: task: Update `scripts/install-local.sh` to create the staging and marker directories and generate the unit with `ExecStartPre=+`.
            expected_output: An operator can recover without network access or runtime configuration when either the current or previous executable can start; documented commands match implemented parsing exactly.
            checks:
              - go test ./internal/updater/... ./cmd/server/... -run 'Test(Rollback|RollbackMissingBackup|RollbackPaths)' -count=1
              - Run each non-mutating documented command against a local build and compare output and exit behavior.
              - git diff --check
            stop_conditions: Stop if rollback accepts an arbitrary unrelated target, deletes the only runnable binary, or is documented as restoring database state.
            escalation: Return explicit manual instructions and surviving paths rather than forcing a destructive recovery.

  - phase_slug: panel-one-click
    story_id: 01M02G1FHY3P2ZBAEQRH8ZGM23
    status: in-progress
    goal: Expose staging and restart through management endpoints and a panel button, install the narrow sudoers rule, and pass the full repository verification gate.
    depends_on: privileged-apply
    allowed_surfaces:
      - internal/api/handlers/management/ (new self-update handler) and internal/api/server.go route registration
      - web/src/pages/SystemPage.tsx, web/src/services/api/version.ts, web/src/i18n/locales/en.json, web/src/i18n/locales/vi.json
      - scripts/install-local.sh sudoers drop-in installation
      - README.md panel update documentation
    avoided_surfaces:
      - frontend test files anywhere under web/
      - the existing latest-version response shape consumed by `handleVersionCheck`
      - provider, runtime, and Postgres code
    waves:
      - wave: 1
        tasks:
          - task: Add `POST /v0/management/self-update` and `POST /v0/management/self-update/apply` behind the existing management-key middleware. The first runs the engine's discover-verify-probe-stage path and returns a typed outcome (already current, staged with version, rejected with reason, unsupported platform). The second executes `sudo -n /usr/bin/systemctl restart llmhub.service` and returns before the process is terminated.
            requirements: [R5, R6, R7, R8, R12, R13]
            dependencies: none
            expected_output: The panel can stage an update and trigger a restart; failures return distinguishable typed reasons without exposing remote response bodies.
            checks:
              - go test ./internal/api/... -run 'Test(SelfUpdateStage|SelfUpdateApply|SelfUpdateAuth|SelfUpdateUnsupported)' -count=1
              - Assert the apply handler invokes exactly one fixed command through an injected runner seam and never constructs it from request input.
            stop_conditions: Stop if either endpoint is reachable without the management key, accepts a client-supplied version or command, or leaks remote bodies into responses.
            escalation: Keep both handlers thin adapters over `internal/updater` and format outcomes only in the handler.
          - task: Install a sudoers drop-in from `scripts/install-local.sh` granting the service user exactly `NOPASSWD: /usr/bin/systemctl restart llmhub.service` with no wildcard, validated with `visudo -c` before activation.
            requirements: [R14]
            dependencies: task: Add `POST /v0/management/self-update` and `POST /v0/management/self-update/apply` behind the existing management-key middleware.
            expected_output: The service user can restart only its own unit and gains no other root capability.
            checks:
              - visudo -cf the generated drop-in
              - grep -c '\*' on the generated drop-in returns 0
              - Confirm the drop-in names the absolute systemctl path and the full unit name.
            stop_conditions: Stop if the rule contains a wildcard, a variable argument, a shell invocation, or any command other than the single unit restart.
            escalation: Ship the panel without the apply endpoint's restart step and require a manual restart rather than widening the rule.
      - wave: 2
        tasks:
          - task: Extend `web/src/pages/SystemPage.tsx` so the existing version check surfaces an Update action when a newer version is available: stage with progress feedback, show the staged version, then trigger restart and poll for the server returning on the new version. Add all new strings to both `en.json` and `vi.json` and the two endpoints to `web/src/services/api/version.ts`.
            requirements: [R13]
            dependencies: wave: 1
            expected_output: An operator presses check, sees the available version, presses update, and observes staging, restart, and the new running version without leaving the panel.
            checks:
              - bun run type-check
              - bun run lint
              - bun build
              - Confirm every new `t()` key exists in both locale files.
            stop_conditions: Stop if any frontend test file is created under `web/`, or if the update action is reachable when no newer version was reported.
            escalation: Keep the change inside `SystemPage.tsx` and the version API module; do not restructure the page or its refresh pattern.
          - task: Run the full verification gate: focused updater, command, and API suites with race detection, then repository-wide tests, vet, frontend build and embed, backend build, release configuration validation, snapshot release, and diff and dependency hygiene.
            requirements: [R15, R16]
            dependencies: task: Extend `web/src/pages/SystemPage.tsx` so the existing version check surfaces an Update action.
            expected_output: All gates pass with no skipped platform build, no leaked temporary files, no races, and a surgical diff.
            checks:
              - go test ./internal/updater/... ./cmd/server/... ./internal/api/... -count=1
              - go test -race ./internal/updater/... ./cmd/server/... ./internal/api/... -count=1
              - go test ./...
              - go vet ./...
              - make build-web
              - make build
              - make release-check
              - make release-snapshot
              - git diff --check
              - git status --short
            stop_conditions: Stop on any command failure or skipped gate; do not attribute unrelated output without proving it predates the initiative.
            escalation: Return the failure to the phase owning the affected invariant and preserve the exact command and output.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- 2026-08-14 / release-signing-foundation / wave 1 / task "Provision a dedicated minisign release key" / task_status=blocked / run_id=none / trace_id=none: `internal/updater/release.pub` and minisign tooling are absent. Production key generation/provisioning and protected GitHub `release` secret changes require explicit secret-handling authorization; no key, secret, or release-signing file was created.
- 2026-08-14 / audit-remediation / step 1 / task_status=superseded / run_id=none / trace_id=none: clean `master` contained no referenced pending web UI batch, so the stale exact-next-action no longer applied; no empty or synthetic commit was created.
- 2026-08-14 / release-signing-foundation / phase decision / task_status=blocked / run_id=none / trace_id=none: owner selected stop before signing-key provisioning; production key material, protected secrets, and placeholder trust files remain untouched.
- 2026-08-14 / release-signing-foundation / wave 1 / task "Extend GoReleaser signing configuration" / task_status=partial / run_id=none / trace_id=none: added an isolated Go signer, `binary_signs` configuration, unsigned-snapshot skip behavior, signed-snapshot input gating, and a complete eight-asset checksum/signature verifier. Nested signer tests, shell syntax checks, `make release-check`, unsigned `make release-snapshot`, and missing-input fail-closed checks pass; cryptographic signed-snapshot verification remains blocked until authorized key/tool inputs exist.
- 2026-08-14 / release-signing-foundation / wave 2 / task "Harden the tag workflow" / task_status=partial / run_id=none / trace_id=none: changed `.github/workflows/release.yml` to use the protected `release` environment, verified minisign 0.12 installation, temporary mode-0600 secret materialization, draft-first upload, remote matrix verification, optional transition public-key input, and runner cleanup without provisioning secrets or publishing a release. The verifier now accepts both GoReleaser snapshot directories and flat downloaded release assets.
- 2026-08-14 / release-signing-foundation / wave 2 / task "Add operator release documentation" / task_status=partial / run_id=none / trace_id=none: added `docs/release-signing.md` and linked it from `docs/README.md`; documents only public contracts, secret names, signed/unsigned snapshot commands, immutable tags, rotation, compromise response, and trusted reinstall boundaries.
- 2026-08-14 / release-signing-foundation / verification / task "Verify pinned minisign tooling" / task_status=passed / run_id=none / trace_id=none: cryptographically verified `minisign-0.12-linux.tar.gz` and its detached signature with the upstream public key through `scripts/verify-minisign-asset.sh`; execution of the Linux binary was not attempted on macOS.
- 2026-08-14 / release-signing-foundation / verification / task "Resolve GoReleaser signer paths" / task_status=passed / run_id=none / trace_id=none: the nested-module wrapper changed cwd before signing, so GoReleaser's relative artifact paths failed; both wrappers now canonicalize file arguments before entering the nested module, with a regression test and a successful eight-asset signed snapshot.
- 2026-08-14 / release-signing-foundation / wave 2 / verification / task_status=passed / run_id=none / trace_id=none: shell syntax, nested signer tests/vet, `make release-check`, actionlint, flat downloaded-asset verification, and a full `make release-signed-snapshot` using an ephemeral non-production fixture key all pass. Production key provisioning, protected secrets, and release publication remain blocked and untouched.
- 2026-08-15 / plan revision / task_status=superseded / run_id=none / trace_id=none: owner reviewed the initiative against the original intent — press check and update inside the management panel on a systemd VPS — and found the previous four phases delivered a CLI-only updater that could not satisfy it, because `scripts/install-local.sh:441` sets `ProtectSystem=full` with `ReadWritePaths` excluding `/usr/local/bin`, so the service cannot write its own binary. Owner then selected checksum-only verification over minisign. All `release-signing-foundation` work above is superseded and scheduled for removal by `release-pipeline-cleanup`; the draft-first workflow and asset-matrix verifier are retained with signature checks stripped.
- 2026-08-17 / privileged-apply / phase start / task_status=in-progress / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / trace_id=none: run created, plan phase status set to in-progress; surfaces: `internal/updater/apply*.go` + tests, `cmd/server/self_update.go` early dispatch, `scripts/install-local.sh` unit/directories/rollback, README operator usage.
- 2026-08-17 / privileged-apply / wave 1 / task "T1" / task_status=DONE / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / trace_id=01M06WDP9GJ9W50T41Y21K72QW: apply-staged-update subcommand: cmd/server/self_update.go routes to updater.ApplyEntry; apply.go implements root gate (processEUID), staged.json untrusted (only names the tag), HTTPS re-fetch of checksums.txt for the staged release, re-hash of llmhub.staged, strictly-newer semver check, backup-copy to `<target>.previous` then atomic tmp+rename swap. Non-root, network failure, forged digest, downgrade all skip with exit 0, target untouched. Proofs: TestApply* (8) + self_update_test.go pass.
- 2026-08-17 / privileged-apply / wave 1 / task "T2" / task_status=DONE / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / trace_id=01M06WDP9GJ9W50T41Y558870D: boot-loop guard (R10): root-owned marker `${LLMHUB_MARKER_DIR:-/var/lib/llmhub-apply}/apply.marker` records outgoing version before swap; server writes service-writable healthy-start token `${DATA_DIR}/update/.booted` (markBooted in main.go after config+Postgres load); apply with marker+booted clears both and proceeds, marker without booted reverts to `<target>.previous` atomically (never clobbers backup) and refuses new applies; missing backup leaves target in place and keeps marker. Proofs: Test(BootMarker|MarkerRevert|MarkerCleared|MarkerUnwritable) + -race pass.
- 2026-08-17 / privileged-apply / wave 1 / task_status=DONE / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / trace_id=01M06WNZKJA53JG631G44WM9CF: wave complete; booted-token design decision recorded (01M06WDDPWH3CWR7XD8BZ53RDM).
- 2026-08-17 / privileged-apply / wave 2 / task "T3" / task_status=DONE / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / trace_id=01M06WMGPSF77PZR948TD0SDPN: install-local.sh creates service-owned `${DATA_DIR}/update` (0750 service user) and root-owned `${LLMHUB_MARKER_DIR:-/var/lib/llmhub-apply}` (0750 root) after user creation; unit generated with `ExecStartPre=+${INSTALL_DIR}/${BINARY} apply-staged-update` first, ahead of init-db-from-env; ReadWritePaths=${DATA_DIR} ${LOG_DIR} ${CONFIG_DIR} excludes the marker dir; ProtectSystem=full and NoNewPrivileges=true unchanged. Proofs: sh -n pass; extracted unit asserts ExecStartPre=+ precedes init-db-from-env with no marker path in ReadWritePaths; systemd-analyze verify PASS.
- 2026-08-17 / privileged-apply / wave 2 / task "T4" / task_status=DONE / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / trace_id=01M06WMGPSF77PZR948WQFS58J: `llmhub update rollback`: updater.RollbackEntry/Rollback restore `<target>.previous` atomically via revertToPrevious (never clobbers backup), clear boot marker + booted token so the next apply starts a fresh cycle, leave the staged candidate for inspection; root-gated (non-root exit 0); missing backup exit 0; restore failure exit 1 (operator command, unlike startup apply). Wired as `update rollback` in runSelfUpdate before flag parsing. README "Updating LLMHub" documents check/stage/apply/auto-revert/manual-rollback, Linux+macOS amd64/arm64 scope, binary-only recovery boundary. Proofs: Test(Rollback|RollbackMissingBackup|RollbackPaths) + non-root/failure + TestUpdateCommandRollback* pass; git diff --check clean.
- 2026-08-17 / privileged-apply / wave 2 / task_status=DONE / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / trace_id=01M06WNZKTH6GB39FG2SFZDMSV: wave complete; all phase tasks DONE, ready for gate.
- 2026-08-17 / panel-one-click / phase start / task_status=in-progress / run_id=01M06X26GDNRKSBKR0CMW1GFHE / trace_id=none: run created, plan phase status set to in-progress; surfaces: `internal/api/handlers/management/` new self-update handler + `internal/api/server.go` routes, `web/src/pages/SystemPage.tsx` + `web/src/services/api/version.ts` + en/vi locales, `scripts/install-local.sh` sudoers drop-in, README panel docs.
- 2026-08-17 / panel-one-click / wave 1 / task "T1" / task_status=DONE / run_id=01M06X26GDNRKSBKR0CMW1GFHE / trace_id=01M06XB188EVNC8ZKBMSR9B00N: `POST /v0/management/self-update` + `/self-update/apply` behind management-key middleware: SelfUpdateEngine interface (implemented by *updater.Engine, fakeable full outcome space), SelfUpdateStage maps errors.Is outcomes to typed JSON (staged+version / current / unsupported / rejected+reason / error), never leaks remote bodies (R13), serialized via selfUpdateMu; SelfUpdateApply responds 202 {"status":"restarting"} FIRST then runs the injected runner (default: fixed `sudo -n /usr/bin/systemctl restart llmhub.service`, no shell, 30s timeout) in a goroutine. Wired: WithSelfUpdateEngine/WithSelfUpdateRestartRunner ServerOptions in server.go, routes after /api-call, both builder call sites in main.go. Proofs: 9 handler tests (stage outcomes, typed-failure no-leak, 503 unconfigured, apply exactly-once via channel, fixed-command assertion, auth denial) PASS; `go test ./internal/api/... -count=1` PASS; `-race` on stage+apply PASS.
- 2026-08-17 / panel-one-click / wave 1 / task "T2" / task_status=DONE / run_id=01M06X26GDNRKSBKR0CMW1GFHE / trace_id=01M06XNGZ9XAAZ9XBFFB5XCH6G: install-local.sh installs a sudoers drop-in `${SUDOERS_DIR:-/etc/sudoers.d}/llmhub` granting `${SERVICE_USER} ALL=(root) NOPASSWD: /usr/bin/systemctl restart ${SERVICE_NAME}.service` (absolute systemctl path, full unit name, no wildcard/glob/variable argument); written via temp file, wildcard grep gate (any `*` aborts), `visudo -cf` validation before activation, installed `install -m 0440 -o root -g root`; skipped with a warning when visudo/sudo is absent (panel restart then needs manual restart — plan escalation, no rule widening). Unit change: `NoNewPrivileges=true` removed with an explanatory comment (decision 01M06XNNSZ145PZA8BV5FG89RD) because no_new_privs blocks setuid and therefore `sudo`; `ProtectSystem=full`, `PrivateTmp`, and the `ExecStartPre=+` ordering are untouched. Proofs: `sh -n` PASS; `visudo -cf` PASS on both the template and real-value rule; wildcard count 0; exact-rule match 1; regenerated unit `systemd-analyze verify` exit 0 with no `NoNewPrivileges` directive.
- 2026-08-17 / panel-one-click / wave 1 / task_status=DONE / run_id=01M06X26GDNRKSBKR0CMW1GFHE / trace_id=none: wave 1 complete; both management endpoints and the sudoers drop-in landed; next is wave 2 panel UI.
- 2026-08-17 / panel-one-click / wave 2 / task "T3" / task_status=DONE / run_id=01M06X26GDNRKSBKR0CMW1GFHE / trace_id=01M06XSXH7519NYVG9M3D7644S: panel update UI: `versionApi` gains `stageUpdate` (POST /self-update) and `applyUpdate` (POST /self-update/apply); SystemPage's version card surfaces an Update button only when the check reports a newer version (`availableVersion`), stages with progress text, shows the staged version, then a danger "Restart to apply" button behind a confirmation dialog triggers the restart and polls `checkLatest` every 3s (90s cap) until `useAuthStore.getState().serverVersion` equals the staged version; every typed status (staged/current/unsupported/rejected/error) maps to a distinct toast. 12 new `system_info.update_*` keys added to en.json + vi.json. Surfaces touched: SystemPage.tsx, version.ts, en/vi locales only — no frontend test files. Proofs: `bun run type-check` PASS, `bun run lint` PASS (0 errors, none in changed files), `bun build` PASS, all 12 new `t()` keys present in both locales.
- 2026-08-17 / panel-one-click / wave 2 / task "T4" / task_status=DONE / run_id=01M06X26GDNRKSBKR0CMW1GFHE / trace_id=01M06Y14XW466N2S9KKQBPE5RJ: README "Updating LLMHub" gains the panel flow (System page version card: check, Update stage, Restart to apply; prerequisites: management key, systemd install, sudo + visudo drop-in). Full verification gate PASS with verdict APPROVED (check 01M06Y10VWS91VZ0BMPG08T57X) — all ten proof commands green; pre-existing `gin.SetMode` race between `t.Parallel()` tests fixed by removing 6 unsynchronized calls (5 in `config_lists_delete_keys_test.go`, 1 in the new `self_update_test.go`); built binary and dist/ trashed after the gate.
- 2026-08-17 / panel-one-click / phase complete / task_status=DONE / run_id=01M06X26GDNRKSBKR0CMW1GFHE / trace_id=none: all four tasks DONE, gate APPROVED (01M06Y10VWS91VZ0BMPG08T57X); panel-triggered binary self-update is fully implemented end-to-end. Commit remains; plan phase lifecycle moves to checked after the changeset lands.

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- 2026-08-14 / plan review revision: accepted R15-R19 and amended the phase tasks to serialize per-target operations, contain unsigned Makefile update paths, define protected minisign tooling/secrets and two-release key rotation with manual emergency recovery, narrow the candidate probe to observable startup behavior, and document selfupdate v0.6.0 stale-backup/fsync/interruption limits. Rationale: the original plan left concurrent mutation, trust-anchor rotation, probe purity, legacy update bypasses, and power-loss recovery underspecified.
- 2026-08-14 / release-signing-foundation / wave 1 task 2: selected a narrowly scoped Go minisign-compatible signer for encrypted-key signing because minisign 0.12 has no stdin, file, or environment passphrase input; retain pinned minisign 0.12 for independent release-asset verification. Rationale: this preserves the non-argv passphrase boundary without adding a PTY helper or exposing the passphrase in process arguments.
- 2026-08-15 / plan revision: replaced the CLI-only in-place updater with a stage-then-privileged-apply design and rewrote all four phases. Rationale: the deployment's systemd sandbox makes in-place self-replacement impossible, and the owner's actual requirement is a panel button with automatic restart.
- 2026-08-15 / plan revision: dropped minisign signing in favor of SHA-256 verification against `checksums.txt`. Rationale: the signing key would have lived in GitHub Secrets, so it defended only against a partial GitHub compromise rather than a repository takeover; the owner accepted that residual risk to remove key provisioning, rotation, and release tooling. Consequence: the root apply step re-fetches `checksums.txt` over HTTPS itself rather than trusting the service-writable staged manifest.
- 2026-08-15 / plan revision: removed `github.com/minio/selfupdate`, the per-target advisory lock, and the fsync/pending-sibling power-loss handling. Rationale: `systemctl restart` stops the process before `ExecStartPre` runs, so the target is never busy and a plain rename suffices; a single-operator single-VPS deployment applying updates serially at boot does not exhibit the concurrency or torn-transaction cases those mechanisms defended against.
- 2026-08-17 / privileged-apply / T2 (recorded as decision 01M06WDDPWH3CWR7XD8BZ53RDM): server records a healthy start via a service-writable `.booted` token inside `${DATA_DIR}/update`; the root apply step clears the root-owned boot marker gated on that token. Rationale: R10's literal wording has the server clear the marker, but the marker is root-owned and outside `${DATA_DIR}`, where the service user cannot write. Resolved: the server writes a healthy-start token in the service-writable staging dir; the root apply consumes the marker only when the token proves the previous swap booted, otherwise it reverts to `<target>.previous` and blocks new applies.
- 2026-08-17 / panel-one-click / T2 (recorded as decision 01M06XNNSZ145PZA8BV5FG89RD): removed `NoNewPrivileges=true` from the generated systemd unit. Rationale: no_new_privs blocks setuid exec, and the plan-mandated restart mechanism (`sudo -n systemctl restart`, R14) requires the setuid-root sudo binary, so the flag made the panel restart impossible. The owner approved dropping it; `ProtectSystem=full`, `PrivateTmp`, the narrow single-command sudoers rule, and the nologin system user remain, so privilege elevation stays bounded to exactly one unit restart.

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- 2026-08-17 / release-pipeline-cleanup / gate / verdict=APPROVED / judge=same-session (deepseek-v4-flash) / run_id=01M06TZ104KKEDMXYXPJVG3DX5 / check_id=01M06V6Z8A5W97VQC6GHZMB8EV: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/release.yml` PASS; `test -z "$(grep -il minisign .github/workflows/release.yml .goreleaser.yml)"` PASS (0 references); `sh -n scripts/verify-release-assets.sh` PASS; `make release-check` PASS (goreleaser v2.16.0, 1 configuration file validated); `BINARY=llmhub scripts/verify-release-assets.sh dist` PASS (verified 8 release binary assets); `git diff --check` PASS. CLI re-ran all six proof commands at record time. Proof gaps: the GitHub Actions workflow runtime (draft creation, gh upload with 9-asset assertion, publish-after-verify) is statically reviewed only — a real tag-push run was not executed; the verifier's flat downloaded-asset-directory layout path is code-reviewed but not runtime-exercised locally (CI exercises it).
- 2026-08-17 / self-update-engine / gate / verdict=APPROVED / judge=same-session (deepseek-v4-flash) / run_id=01M06V9S1ADNWC6GTQDA5KRTXR / check_id=01M06VYHYNRV5S6EF7W8WVSGH7: `go build ./...` PASS; `go test ./...` PASS; `go vet ./...` PASS; task 1 `go test ./internal/updater/... -run 'Test(Client|LatestRelease|HTTP|Redirect|ResponseLimit)' -count=1` PASS (11 tests); task 2 `-run 'Test(Version|Asset|Platform|Checksum)'` PASS (16 tests); task 3 `-run 'Test(Download|Checksum|Probe|Stage|StagedManifest)'` PASS (engine + probe aspects) and `go test -race -run 'Test(Download|Probe|Stage)'` PASS; task 4 `go test ./cmd/server/... -run 'Test(VersionCommand|UpdateCommand|UpdateCheck|EarlyCommandDispatch|ExistingDBCommands|Usage)'` PASS (17 tests) and `env -u PGSTORE_DSN -u pgstore_dsn go run ./cmd/server version --short` PASS; task 5 `make -n download-latest` / `make -n install-latest` PASS (deprecation recipes only, both targets exit nonzero with no dist mutation); `go list -m golang.org/x/mod` PASS (v0.40.0). CLI re-ran all twelve proof commands at record time. Proof gaps: live GitHub endpoint behavior is exercised only against TLS test servers; the darwin/arm64 staging path is code-reviewed but runtime-tested only on linux/amd64 (other-platform tests self-skip); a real tag-push release was not staged against production assets.
- 2026-08-17 / privileged-apply / gate / verdict=APPROVED / judge=same-session (deepseek-v4-flash) / run_id=01M06W4W6YG2R0SBBJE1S2EC8P / check_id=01M06WQ9MPXCBY8YTREPN1A33P: `go build ./...` PASS; `go test ./internal/updater/... ./cmd/server/... -count=1` PASS (updater + cmd/server suites); `go test -race ./internal/updater/... -run 'Test(BootMarker|MarkerRevert)' -count=1` PASS; `sh -n scripts/install-local.sh` PASS; `git diff --check` PASS. In-session (non-proof) checks also executed: T1 filter `-run 'Test(Apply|ApplyReverify|ApplyDowngrade|ApplyForged|ApplyNetworkFailure|ApplyNonRoot|ApplyExitCode)'` PASS (8 tests); T2 filter + `-race` PASS; T3 extracted-unit assertion `ExecStartPre=+` apply precedes init-db-from-env with marker dir absent from `ReadWritePaths` PASS, `systemd-analyze verify` PASS (dummy binary in temp INSTALL_DIR); T4 `-run 'Test(Rollback|RollbackMissingBackup|RollbackPaths)'` + non-root/failure + `TestUpdateCommandRollback*` PASS. CLI re-ran all five proof commands at record time. `zharness audit --json`: only expected pointer_drift (latest_check 01M06VYHYNRV5S6EF7W8WVSGH7 belonged to the previous run), resolved by this record; zero contract violations, zero unlinked proofs. Proof gaps: root-owned marker-dir ownership, `ExecStartPre=+` privilege elevation, and the full swap→restart→healthy-boot cycle are unit-tested and code-reviewed only — no real systemd host run; `Rollback`/`Apply` behavior on a true root euid is simulated via the `processEUID` seam; macOS darwin/arm64 remains code-reviewed only (tests self-skip).
- 2026-08-17 / panel-one-click / gate / verdict=APPROVED / judge=same-session (deepseek-v4-flash) / run_id=01M06X26GDNRKSBKR0CMW1GFHE / check_id=01M06Y10VWS91VZ0BMPG08T57X: `go test ./internal/updater/... ./cmd/server/... ./internal/api/... -count=1` PASS (7 packages); `go test -race ./internal/updater/... ./cmd/server/... ./internal/api/... -count=1` PASS (7 packages) — after removing 6 unsynchronized `gin.SetMode` calls that raced between `t.Parallel()` tests (5 pre-existing in `config_lists_delete_keys_test.go`, 1 in the new `self_update_test.go`); `go test ./... -count=1` PASS; `go vet ./...` PASS; `make build-web` PASS; `make build` PASS (Version=v7.2.114-10-g01b1dd02-dirty); `make release-check` PASS (goreleaser v2.16.0); `make release-snapshot` PASS (8 platform binaries + checksums.txt + artifacts.json, 13s); `git diff --check` PASS; `git status --short` shows only the 11 modified + 2 new planned surfaces (built binary and dist/ trashed after the gate). In-session (non-proof) checks: T1 `-run 'Test(SelfUpdateStage|SelfUpdateApply|SelfUpdateAuth|SelfUpdateUnsupported)'` PASS (9 tests) + `-race` PASS; T2 `sh -n scripts/install-local.sh` PASS, `visudo -cf` PASS on the extracted drop-in (template and real values), wildcard count 0, exact-rule match 1, regenerated unit `systemd-analyze verify` exit 0; T3 `bun run type-check` PASS, `bun run lint` PASS (0 errors, none in changed files), `bun build` PASS, all 12 new `t()` keys present in both en.json and vi.json. CLI re-ran all ten proof commands at record time. Proof gaps: the full check→stage→restart→new-version-return cycle is unit-tested (fake engine + injected runner + poll logic) and code-reviewed only — no live host run against a real systemd service with the sudoers drop-in; the sudoers drop-in was validated with `visudo -cf` but never activated on a real host; the panel restart poll was exercised through type-check/lint/build, not in a live browser against a restarting server; macOS darwin/arm64 staging remains code-reviewed only (tests self-skip).

## Current State and Next Action
- active_phase: panel-one-click
- lifecycle_status: checked (matches DB; commit e9a04d35 + docs 54cc3ed1)
- latest_run_id: 01M06X26GDNRKSBKR0CMW1GFHE
- latest_trace_ids: [01M06XB188EVNC8ZKBMSR9B00N, 01M06XNGZ9XAAZ9XBFFB5XCH6G, 01M06XSXH7519NYVG9M3D7644S, 01M06Y14XW466N2S9KKQBPE5RJ]
- latest_check_id: 01M06Y10VWS91VZ0BMPG08T57X
- latest_handoff_id: 01M02GJ4CD1R938033MY4PV0HN
- blockers: none
- db_registration: The four phases were registered in the harness DB on 2026-08-17 with the story IDs recorded above (the earlier "registered 2026-08-15" note was inaccurate — only the superseded signing-era stories existed until today's reconciliation changeset `01M06TX3PKRATR6X58KGYZ7AEX`). The superseded signing-era stories remain as inert rows with zero runs: `release-signing-foundation`, `self-update-engine-superseded` (renamed), `operator-update-cli`, `self-update-verification-gate` — never anchor this plan's run, check, or handoff rows to them. The DB's prior position `model-combos/checked` belongs to a different initiative; never anchor this plan's run, check, or handoff rows to that initiative's IDs either.
- worktree_state: clean after commits e9a04d35 (feat(panel): one-click self-update via management API and panel button) + 54cc3ed1 (docs(plan): record panel-one-click phase completion and gate approval). The self-update plan is complete: all four phases checked.
- open_items: []
- exact_next_action: none — plan complete. Self-update is implemented end-to-end (engine, CLI, privileged apply with boot-loop revert, panel one-click with sudoers-bounded restart) and gated APPROVED in all four phases.
