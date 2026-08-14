---
id: 01KZZVZETP16SJH4Y5157WSP0Z
type: plan
intake_id: 01KZZVZVDSKW58XYPNR1NK44V7
lane: high-risk
status: active
created: 2026-08-14
updated: 2026-08-14
---

# Plan: Secure binary self-update

## Outcome
- result: LLMHub release binaries can be checked, cryptographically verified, staged, and replaced through an explicit operator-run CLI command without requiring Postgres startup, mutating the binary during normal server startup, or trusting GitHub-hosted checksums alone.
- success_signals:
  - `llmhub version --short`, `llmhub update --check`, `llmhub update`, and `llmhub update rollback` dispatch before runtime configuration or database loading and return deterministic exit codes.
  - A supported Linux or macOS release binary discovers the latest stable GitHub Release, selects the exact existing `llmhub-{os}-{arch}` asset, refuses malformed versions and downgrades, and reports an already-current version without modifying disk state.
  - Every candidate is downloaded through bounded HTTPS requests, matched against the existing GoReleaser SHA-256 checksum manifest, verified with a detached minisign signature and an embedded trusted public key, and rejected before execution or replacement when any check fails.
  - A verified candidate passes a side-effect-free staged self-test before `minio/selfupdate` replaces the resolved executable target, preserves executable mode, and retains one same-filesystem previous binary for recovery.
  - Failed permission checks, downloads, checksum/signature checks, candidate probes, and apply operations leave the installed target unchanged or restore it; any rollback failure reports the exact surviving recovery path.
  - The updater never restarts systemd, performs database work, elevates privileges, or runs periodically; the operator receives an explicit restart instruction after a successful update.
  - The release workflow publishes checksums plus one minisign sidecar per binary from a protected signing environment, and local snapshot verification proves the complete asset set before a tagged release.
  - Focused updater tests, repository Go tests, vet, release configuration validation, snapshot release build, frontend embed build, backend build, and diff hygiene checks pass.

## Authority and Requirements
- authority:
  - Owner instruction on 2026-08-14 approved GitHub Releases, an explicit update command, SHA-256 plus minisign verification, staged candidate validation, one retained backup, and no startup/background mutation for the first implementation.
  - `github.com/minio/selfupdate` v0.6.0 public API and source behavior: `Options.CheckPermissions`, `Apply`, `RollbackError`, checksum verification, minisign `Verifier`, target-mode handling, sibling `.new`/`.old` replacement, and caller-controlled `OldSavePath`.
  - GitHub Releases latest-release API and immutable-release model for public release metadata and asset hosting.
  - `cmd/server/main.go`, `cmd/server/db_runtime.go`, and `internal/buildinfo/buildinfo.go`: current early positional command dispatch and linker-injected version metadata.
  - `internal/api/handlers/management/config_basic.go`: current GitHub latest-release URL, timeout, user agent, and notification-only management behavior.
  - `.goreleaser.yml`, `.github/workflows/release.yml`, and `Makefile`: current release tag validation, existing bare-binary asset names, SHA-256 `checksums.txt`, build metadata injection, snapshot gate, and GitHub release automation.
  - `scripts/install-local.sh`: `/usr/local/bin` installation, root-owned update boundary, restricted `llmhub` service user, `ProtectSystem=full`, and explicit systemd restart lifecycle.
  - Repository `CLAUDE.md`: minimal surgical changes, Postgres-only runtime configuration, no frontend test files under `web/`, and proof through Go, frontend, and build checks.
- requirements:
  - R1 [accepted]: Add clean `version` and self-update positional commands that dispatch before the unconditional startup banner, flag parsing for server mode, Postgres loading, configuration access, or service startup. | source: owner-approved trigger policy; `cmd/server/main.go`; `cmd/server/db_runtime.go`.
  - R2 [accepted]: Reuse `internal/buildinfo` and existing linker flags as the only current-version authority; `version --short` must emit a normalized machine-readable release version with no extra banner or logging. | source: `internal/buildinfo/buildinfo.go`; `.goreleaser.yml`; `Makefile`.
  - R3 [accepted]: Use the public GitHub Releases latest endpoint and existing `llmhub-{os}-{arch}[.exe]` naming, require an exact supported asset match, and treat all remote metadata, names, sizes, URLs, and bodies as untrusted input. | source: `internal/api/handlers/management/config_basic.go`; `.goreleaser.yml`; owner-approved hosting decision.
  - R4 [accepted]: Validate normalized semantic versions, refuse malformed current/latest versions, refuse downgrades and same-version replacement, and exclude draft/prerelease behavior from the first stable channel. Development binaries must not silently replace themselves. | source: owner-approved stable-channel and anti-downgrade behavior; existing SemVer tag gate in `Makefile`.
  - R5 [accepted]: All release metadata, checksum, signature, and binary requests must use HTTPS, explicit context/timeout handling, status checks, response-size limits, and deterministic cleanup; any network failure must occur before executable replacement. | source: owner-approved failure behavior; current bounded latest-version handler; `minio/selfupdate` reader behavior.
  - R6 [accepted]: Verify the downloaded binary against its raw SHA-256 digest from the existing `checksums.txt`, requiring exactly one well-formed checksum entry for the selected asset. A checksum mismatch must leave the target unchanged. | source: existing `.goreleaser.yml` checksum asset; owner-approved integrity requirement.
  - R7 [accepted]: Verify a detached per-binary minisign signature against trusted public-key text embedded in the released binary before staging or executing the candidate, and pass the same verifier to `selfupdate.Apply` for commit-time re-verification. Runtime key download and trust-on-first-use are forbidden. | source: owner-approved authenticity boundary; `minio/selfupdate.Verifier` API.
  - R8 [accepted]: Stage the verified candidate beside the installed target, preserve executable mode, and run a bounded side-effect-free self-test that proves the candidate starts and reports the expected release version before replacement. | source: owner-approved failed-start protection; same-filesystem/noexec constraints.
  - R9 [accepted]: Resolve the real executable target, fail early on insufficient permissions, call `selfupdate.Apply` with an explicit same-filesystem `OldSavePath`, call `RollbackError` for every apply failure, retain one previous binary after success, and expose a recovery command that does not require runtime configuration. | source: `minio/selfupdate` replacement and rollback contract; `scripts/install-local.sh` installation boundary.
  - R10 [accepted]: The updater must not invoke sudo, restart systemd, load or migrate Postgres, alter configuration, or update from the service background; after success it must state that the running process still uses the old image until an operator restarts it. | source: owner-approved trigger and restart policy; `scripts/install-local.sh` service permissions.
  - R11 [accepted]: Enable self-update only for Linux and macOS on amd64 and arm64 in the first implementation; retain existing Windows and FreeBSD release builds but return an explicit unsupported-platform result there until separately tested. | source: owner-approved platform scope; current GoReleaser matrix.
  - R12 [accepted]: Extend the existing release pipeline to create and publish one `.minisig` sidecar per binary using a dedicated protected release key, preserve `checksums.txt`, and ensure release snapshots/checks fail when required update assets are absent or unverifiable. | source: owner-approved release-signing workflow; `.goreleaser.yml`; `.github/workflows/release.yml`.
  - R13 [accepted]: Add deterministic Go coverage for version normalization/comparison, release parsing and asset selection, bounded HTTP failures, checksum/signature rejection, staged self-test failure, target-path and mode behavior, apply rollback reporting, and unsupported platforms without touching the running test binary. | source: owner-approved reliability criteria; repository testing rules.
  - R14 [accepted]: Final verification must include focused updater tests, `go test ./...`, `go vet ./...`, `make release-check`, `make release-snapshot`, `make build-web`, `make build`, `git diff --check`, and working-tree inspection, with no skipped or failing gate reported as complete. | source: repository `CLAUDE.md`; existing Makefile release gates.

## Non-goals
- NG1: Automatic update mutation during server startup, periodic background checks, unattended daemon updates, or silent update installation from the management UI.
- NG2: Automatic privilege elevation, package-manager integration, systemd restart/orchestration, rolling fleet deployment, or zero-downtime process replacement.
- NG3: Windows or FreeBSD executable replacement support in the first implementation; their existing release artifacts remain unchanged.
- NG4: A custom HTTPS update service, private-GitHub-release authentication, embedded GitHub credentials, update channels, prerelease opt-in, or arbitrary-version selection.
- NG5: TUF/go-tuf metadata, cosign/Rekor runtime policy, GPG verification, threshold signatures, transparent key discovery, or automatic signing-key rotation in the first implementation.
- NG6: Binary delta patching, archive extraction, multi-file application updates, external asset migration, or updates to package-managed installations.
- NG7: Automatic rollback of Postgres state, schema migrations, configuration data, or any non-binary side effect.
- NG8: Changes to the current management latest-version response or frontend update notification beyond compatibility adjustments strictly required by the new shared code; no frontend test files under `web/`.
- NG9: Refactoring unrelated CLI startup, release tooling, installer behavior, logging, service configuration, or provider/runtime code.

## Approach and Risks
- approach: Deliver the updater in four dependency-ordered phases. First establish the release-signing trust boundary and prove that the existing bare GoReleaser assets can carry detached minisign signatures without exposing the private key. Next build a testable `internal/updater` engine for bounded release discovery, strict version/asset selection, checksum and signature verification, staged probing, replacement, and recovery. Then wire explicit positional commands into `cmd/server` before the current banner and Postgres path. Finish with signed-snapshot, negative-path, cross-platform build, and repository-wide verification. Keep the existing management notification path, server lifecycle, Postgres configuration, release asset names, and unsigned runtime behavior outside explicit update commands unchanged.
- constraints:
  - The public update source is the stable public GitHub Releases endpoint for `therealtinhtute/llmhub`; runtime database configuration and the management API proxy setting are unavailable to early CLI commands, so the updater uses a dedicated client with standard environment proxy behavior.
  - `internal/buildinfo` and current linker flags remain the only version source; no new config, database row, environment setting, or working-directory state controls updater behavior.
  - `checksums.txt` already exists in `.goreleaser.yml`; release work adds per-binary minisign sidecars and verification rather than replacing the checksum pipeline.
  - Production private-key bytes must never enter the repository, plan, logs, command traces, build artifacts, or test fixtures. Only the public key may be committed. Provisioning or changing the GitHub secret/protected environment is an outward-facing action and requires explicit authorization during execution.
  - Candidate, target, backup, and rollback staging paths remain on the target filesystem. Tests must use temporary target files or injected seams and must never replace the running test or development executable.
  - First-version self-update support is Linux and macOS on amd64/arm64. Windows and FreeBSD binaries continue to build, but updater entry points reject those platforms before mutation.
  - A successful update replaces only bytes on disk. It does not restart the process/service, invoke sudo, modify systemd, read Postgres, run migrations, or promise rollback of non-binary state.
  - No frontend test files are added under `web/`; existing frontend behavior is verified only through the repository build/embed gates because the plan does not change the update-notification product flow.
  - Phase/story definitions and verification commands are immutable after this planning pass; lifecycle status changes only through workflow transitions.
- decisions:
  - Add a server-internal package at `internal/updater/` and a thin command adapter at `cmd/server/self_update.go`; do not move or refactor the existing management latest-version handler unless compilation requires a compatibility-only change.
  - Pin `github.com/minio/selfupdate` to v0.6.0 and use `golang.org/x/mod/semver` for strict normalized comparison rather than implementing custom lexical version ordering.
  - Store the auditable production public key as `internal/updater/release.pub` and embed it into the binary; use clearly non-production key material under `internal/updater/testdata/` only for deterministic tests.
  - Keep the existing release asset name `llmhub-{os}-{arch}[.exe]`, checksum asset `checksums.txt`, and define the signature asset as the exact binary name plus `.minisig`.
  - Download signatures with the updater's bounded HTTP client and load the downloaded bytes through a mode-0600 temporary file into `selfupdate.Verifier`; do not call `Verifier.LoadFromURL`, whose v0.6.0 API does not expose an overall client timeout.
  - Validate SHA-256 and minisign before candidate execution, run a bounded `version --short` probe from a sibling candidate file, then pass the same raw checksum and verifier to `selfupdate.Apply` for commit-time re-verification.
  - Resolve symlinks before replacement, preserve the current executable permission bits, keep one fixed sibling backup at `<target>.previous`, and call `selfupdate.RollbackError` for every apply error.
  - Support recovery both from the current executable and by running `<target>.previous update rollback`; when invoked from the `.previous` path, infer the original target by removing that suffix instead of accepting an arbitrary replacement target.
  - Make GitHub Actions create a draft release, verify the required binary/checksum/signature assets, and publish only after verification. Preserve `make release-snapshot` for ordinary local build proof and add an explicit signed-snapshot target requiring supplied test/release key paths.
- rejected_alternatives:
  - Checksums over the same GitHub channel without signatures: detects corruption but does not protect against repository/release credential compromise.
  - Custom HTTPS update hosting: duplicates version metadata, asset hosting, authentication, retention, and release operations already supplied by GitHub Releases.
  - Startup or background mutation: conflicts with `/usr/local/bin` ownership, `ProtectSystem=full`, predictable server startup, and explicit operator restart control.
  - Direct `selfupdate.Apply` immediately after download: can install a correctly signed but non-starting or wrong-version candidate without a pre-commit probe.
  - Runtime cosign, GPG, or TUF verification in the first version: materially increases policy, dependency, and key-rotation scope beyond the accepted minisign design.
  - Automatic sudo/systemd integration or package-manager replacement: mixes binary verification with privilege escalation and service/deployment policy.
- risks:
  - risk: The production minisign private key is exposed through source control, shell tracing, CI logs, artifacts, or an overly broad GitHub secret scope.
    mitigation: Generate/provision it outside the worktree, commit only `release.pub`, materialize it under the runner temporary directory with mode 0600, disable tracing around secret handling, scope it to a protected `release` environment, and verify tracked/untracked paths without printing contents.
    recovery: Stop immediately, do not publish, revoke and rotate the key, replace the embedded public key through a trusted release path, and invalidate any draft assets signed by the exposed key.
  - risk: GoReleaser signs a build-stage filename rather than each final `llmhub-{os}-{arch}[.exe]` release asset, or omits sidecars from publishing.
    mitigation: Use `signs.artifacts: binary` with the current `archives.formats: [binary]`, assert the exact asset matrix locally, create releases as drafts, and inspect/download every remote draft asset before publication.
    recovery: Leave the release in draft, correct the signing/publish pipeline, and never replace assets beneath an already published immutable tag.
  - risk: Remote metadata or responses cause downgrade, duplicate-asset ambiguity, unbounded memory use, redirect abuse, or partial-body acceptance.
    mitigation: Normalize strict SemVer, reject development/malformed/same/older versions, require one exact asset/checksum entry, enforce HTTPS and repository path checks, constrain redirects to HTTPS, set separate metadata and binary timeouts, validate declared size when present, and detect limit overflow.
    recovery: Fail closed before staging; retain the installed binary and report the rejected field/status without logging response bodies beyond a small diagnostic limit.
  - risk: Symlink resolution, target permissions, filesystem boundaries, or a failure between rename operations leaves the target missing or the wrong mode.
    mitigation: Resolve the executable, stat and preserve mode, stage and retain backup beside the target, run `CheckPermissions` before download/apply, call `RollbackError`, and report target/backup paths on every recovery failure.
    recovery: Do not restart the service; restore `<target>.previous` manually or by executing it with `update rollback`, preserving any failed candidate for diagnosis.
  - risk: A downloaded candidate is authentic but cannot start on the host or its version output does not match release metadata.
    mitigation: Verify before execution, run a bounded side-effect-free sibling probe, compare normalized version output, and perform a post-commit installed-target probe while the old updater process is still alive.
    recovery: Skip commit on preflight failure; after a post-commit failure atomically restore `.previous` and return a nonzero result with both probe and rollback outcomes.
  - risk: Command wiring accidentally prints the normal banner, loads Postgres, mutates runtime configuration, or restarts the service during `version`, `update`, or `rollback`.
    mitigation: Dispatch through small functions before the existing banner and server flags, inject engine dependencies for tests, use direct CLI output rather than server startup, and add subprocess tests with database variables absent.
    recovery: Stop the CLI phase and restore the previous dispatch order until early-command tests prove no runtime initialization occurs.
  - risk: `minio/selfupdate` v0.6.0 behavior or platform assumptions diverge from the planned wrapper, especially around rollback and Windows locks.
    mitigation: Pin the module, test `TargetPath` against temporary files, inspect `RollbackError` on every failure, restrict enabled platforms, and keep wrapper behavior narrower than the library's advertised platform set.
    recovery: Disable the affected platform or apply path rather than introducing a helper process or custom replacement protocol inside this initiative.
  - risk: Binary rollback occurs after a release has already applied an incompatible database/schema change.
    mitigation: The update command never starts the server or runs migrations; restart remains an explicit later operator action, and documentation states that `.previous` restores only binary bytes.
    recovery: Use the service's separate database recovery procedure; do not present binary rollback as data rollback.
- stop_conditions:
  - Any implementation or test would write, print, commit, or upload production private-key material outside the explicitly authorized protected secret destination.
  - The updater requires Postgres/config loading, management authentication, automatic privilege elevation, automatic service restart, or a new runtime configuration surface.
  - A candidate must be executed before checksum and signature verification, or an apply path cannot preserve/recover the current target on a synchronous failure.
  - The exact final release assets cannot be signed and verified in a draft/snapshot without changing their established names.
  - A test would replace the currently running binary, require a live provider/database, or depend on timing/network services rather than deterministic fixtures.
  - Windows/FreeBSD support requires a helper process, installer redesign, or OS-specific replacement protocol; leave it explicitly unsupported.
  - Any focused or broad verification command fails; retain the command/output and return to the owning phase rather than weakening the check.

## Phases and Verification
<!-- Phase and task definitions are immutable after to-plan. Do not add task status fields. Append-only Progress is the sole task execution-status source. Only each phase lifecycle status changes to mirror DB transitions: to-plan=planned; work after run create=in-progress; clean durable check=checked; closing handoff=done. Each planned phase records phase_slug, story_id, status, goal, depends_on, waves, tasks, and checks. -->
- planning_status: planned
- phases:
  - phase_slug: release-signing-foundation
    story_id: 01KZZWAEEV2E1M1Z2KHJYQ7B3V
    status: planned
    goal: Establish the minisign trust material, per-binary signature contract, and protected draft-release pipeline without exposing the private signing key.
    depends_on: none
    allowed_surfaces:
      - internal/updater/release.pub (new public key file only)
      - internal/updater/testdata/ non-production signing fixtures only where required
      - .goreleaser.yml
      - .github/workflows/release.yml
      - Makefile release targets/help
      - scripts/verify-release-assets.sh or one equivalent narrowly scoped release-asset verifier
      - README.md release/update security instructions only
    avoided_surfaces:
      - production private-key files, bytes, logs, artifacts, or test fixtures
      - cmd/server runtime behavior and Postgres/config packages
      - web application source and frontend tests
      - existing binary names, build matrix, checksum filename, and linker metadata contract
    waves:
      - wave: 1
        tasks:
          - task: Provision a dedicated minisign release key under explicit secret-handling authorization, commit only the public key at `internal/updater/release.pub`, and add clearly labeled non-production fixture material only if deterministic repository tests need it.
            requirements: [R7, R12]
            dependencies: none
            expected_output: The repository contains one auditable production public key and no production private key; the private key has a documented protected-secret destination and rotation owner.
            checks:
              - test -s internal/updater/release.pub
              - test -z "$(git ls-files | grep -E '(^|/)(release|minisign)([-_.].*)?\.key$' || true)"
              - git status --short
            stop_conditions: Stop before generating, copying, or uploading private-key material unless the exact local and GitHub secret destinations are explicitly authorized; never print the key while validating paths.
            escalation: Record the missing public-key/protected-secret prerequisite and leave all release/update code unable to accept production signatures until the owner provisions it.
          - task: Extend GoReleaser signing configuration so every final bare binary asset receives an exact `<asset>.minisig` sidecar while preserving the current build matrix, `checksums.txt`, filenames, and ldflags.
            requirements: [R3, R6, R7, R11, R12]
            dependencies: task: Provision a dedicated minisign release key under explicit secret-handling authorization, commit only the public key at `internal/updater/release.pub`, and add clearly labeled non-production fixture material only if deterministic repository tests need it.
            expected_output: GoReleaser accepts a custom minisign signer with `artifacts: binary`; a signed snapshot produces binaries, `checksums.txt`, and matching sidecars for Linux, macOS, Windows, and FreeBSD assets without renaming them.
            checks:
              - make release-check
              - MINISIGN_KEY_PATH=<authorized-test-key> MINISIGN_PUBLIC_KEY_PATH=<matching-public-key> make release-signed-snapshot
              - scripts/verify-release-assets.sh dist <matching-public-key>
            stop_conditions: Stop if GoReleaser signs intermediate filenames, omits final sidecars, modifies binary bytes after signing, or requires checking a secret into configuration.
            escalation: Keep the release as a local snapshot and adjust only the signer/artifact pipeline before changing updater asset lookup.
      - wave: 2
        tasks:
          - task: Harden the tag workflow to install a pinned minisign tool, materialize the protected key in runner temporary storage, create a draft release, verify the full remote asset/checksum/signature matrix, and publish only after verification.
            requirements: [R5, R7, R12]
            dependencies: wave: 1
            expected_output: A `v*` workflow cannot publish a release missing a required binary, checksum, or valid sidecar; failed verification leaves the release in draft and secret values absent from logs.
            checks:
              - go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/release.yml
              - make release-check
              - git diff --check
              - Inspect the workflow diff to confirm `environment: release`, no shell tracing around key material, mode-0600 temporary key creation, draft-first publication, and cleanup confined to runner temporary storage.
            stop_conditions: Stop if the workflow publishes before verification, expands the private key in command output, uses an unpinned third-party action for signing, or overwrites assets beneath an existing tag.
            escalation: Preserve the existing unsigned workflow until a draft-first signed pipeline passes local configuration and asset checks; do not push a release tag as a test without explicit outward-action approval.
          - task: Add concise operator/release documentation for public-key ownership, protected secret provisioning, signed-snapshot commands, immutable tags, and emergency key rotation without documenting private-key contents.
            requirements: [R7, R10, R12]
            dependencies: task: Harden the tag workflow to install a pinned minisign tool, materialize the protected key in runner temporary storage, create a draft release, verify the full remote asset/checksum/signature matrix, and publish only after verification.
            expected_output: Maintainers can reproduce the signed snapshot and understand that a compromised key blocks publication and requires a trusted public-key replacement release.
            checks:
              - Review README commands against Makefile target names and workflow secret names.
              - git diff --check
            stop_conditions: Stop if documentation encourages committing a private key, placing it in normal repository variables, replacing immutable release assets, or bypassing draft verification.
            escalation: Document only the public contract and direct maintainers to the protected secret manager for sensitive operational steps.

  - phase_slug: self-update-engine
    story_id: 01KZZWAV7G861YEYJQ0WC4N3F2
    status: planned
    goal: Implement bounded GitHub release discovery, strict version and asset selection, SHA-256 plus minisign verification, staged candidate probing, and recoverable binary replacement.
    depends_on: release-signing-foundation
    allowed_surfaces:
      - go.mod and go.sum for pinned updater and semantic-version dependencies
      - internal/updater/*.go (new)
      - internal/updater/*_test.go and internal/updater/testdata/ non-production fixtures
      - internal/updater/release.pub established by the preceding phase
    avoided_surfaces:
      - cmd/server command dispatch until the operator-update-cli phase
      - internal/api/handlers/management latest-version behavior
      - runtime config, Postgres stores, systemd installer, provider executors, and web source
      - live GitHub Releases or replacement of the running test/development executable
    waves:
      - wave: 1
        tasks:
          - task: Add pinned `github.com/minio/selfupdate` and semantic-version dependencies, then implement an injectable GitHub release client with production URL/user-agent defaults, standard environment proxy support, HTTPS redirect checks, status handling, separate metadata/binary timeouts, and strict response-size limits.
            requirements: [R3, R4, R5, R11]
            dependencies: none
            expected_output: The updater can retrieve bounded latest-release metadata and selected asset bytes through deterministic client seams without importing Gin, runtime config, or management handlers.
            checks:
              - go test ./internal/updater/... -run 'Test(Client|LatestRelease|HTTP|Redirect|ResponseLimit)' -count=1
              - go list -m github.com/minio/selfupdate golang.org/x/mod
            stop_conditions: Stop if the production client accepts non-HTTPS release URLs/redirects, has an unbounded body path, requires database proxy configuration, or hides non-200 diagnostics.
            escalation: Keep endpoints/timeouts as private constants and expose only the smallest constructor seam needed by `httptest` fixtures.
          - task: Implement strict current/latest version normalization, no-development/no-downgrade/no-same-version decisions, supported-platform mapping, exact binary/signature asset selection, and duplicate-safe parsing of the selected SHA-256 checksum entry.
            requirements: [R2, R3, R4, R6, R11]
            dependencies: task: Add pinned `github.com/minio/selfupdate` and semantic-version dependencies, then implement an injectable GitHub release client with production URL/user-agent defaults, standard environment proxy support, HTTPS redirect checks, status handling, separate metadata/binary timeouts, and strict response-size limits.
            expected_output: Stable releases map deterministically to `llmhub-{os}-{arch}[.exe]`, malformed/ambiguous metadata fails closed, and development or unsupported builds cannot enter mutation code.
            checks:
              - go test ./internal/updater/... -run 'Test(Version|Asset|Platform|Checksum)' -count=1
              - Cover leading `v`, bare versions, prerelease metadata, malformed tags, duplicate assets/checksums, same version, downgrade, `dev`, Linux/macOS support, and Windows/FreeBSD rejection.
            stop_conditions: Stop if comparison becomes lexical, asset fallback is fuzzy, a duplicate is silently accepted, or unsupported platforms proceed beyond decision logic.
            escalation: Preserve the established asset contract and reject unknown release shapes rather than adding aliases/channels.
      - wave: 2
        tasks:
          - task: Implement SHA-256 and embedded minisign verification, downloading signature bytes through the bounded client, loading them through a mode-0600 temporary file, verifying before any candidate execution, and supplying the same verifier/checksum to commit-time apply options.
            requirements: [R5, R6, R7]
            dependencies: wave: 1
            expected_output: Corrupted, unsigned, wrong-key, duplicate-checksum, truncated, or oversized candidates fail before staging and leave no sensitive or executable temporary residue.
            checks:
              - go test ./internal/updater/... -run 'Test(Checksum|Signature|Verifier|TemporarySignature)' -count=1
              - go test -race ./internal/updater/... -run 'Test(Checksum|Signature|Verifier)' -count=1
            stop_conditions: Stop if public keys are fetched at runtime, `Verifier.LoadFromURL` is used, a candidate executes before signature verification, or failed verification leaves a target/candidate mutation.
            escalation: Keep the verifier wrapper local to `internal/updater` and report any library API mismatch before replacing the verification primitive.
          - task: Implement target resolution, permission preflight, sibling candidate staging, mode preservation, bounded side-effect-free version probing, `selfupdate.Apply`, same-filesystem `.previous` retention, post-commit probing, and complete `RollbackError` reporting.
            requirements: [R8, R9, R10, R13]
            dependencies: task: Implement SHA-256 and embedded minisign verification, downloading signature bytes through the bounded client, loading them through a mode-0600 temporary file, verifying before any candidate execution, and supplying the same verifier/checksum to commit-time apply options.
            expected_output: A verified candidate replaces only an explicitly resolved target after it starts and reports the expected version; synchronous commit failures restore the old target or return both update and rollback errors with recovery paths.
            checks:
              - go test ./internal/updater/... -run 'Test(Target|Permissions|Candidate|Apply|PostCommit|RollbackError|Mode|Symlink)' -count=1
              - Assert all replacement tests use temporary targets and injected probes rather than `os.Executable()`'s running test path.
            stop_conditions: Stop if candidate staging crosses filesystems, a probe performs server/database startup, target mode changes, an apply error skips `RollbackError`, or the test process executable can be replaced.
            escalation: Narrow the engine API around explicit target/probe seams and disable post-commit mutation until rollback invariants are deterministic.
          - task: Implement one-generation rollback path resolution and atomic restore behavior for both `<target> update rollback` and `<target>.previous update rollback`, retaining a failed current binary when restoration succeeds and reporting every surviving path.
            requirements: [R9, R10, R13]
            dependencies: task: Implement target resolution, permission preflight, sibling candidate staging, mode preservation, bounded side-effect-free version probing, `selfupdate.Apply`, same-filesystem `.previous` retention, post-commit probing, and complete `RollbackError` reporting.
            expected_output: An operator can recover without network or runtime configuration when either the current or previous executable can start, while missing/ambiguous backups fail without deleting the current target.
            checks:
              - go test ./internal/updater/... -run 'Test(RollbackPaths|RollbackSuccess|RollbackMissingBackup|RollbackRestoreFailure)' -count=1
            stop_conditions: Stop if rollback accepts an arbitrary unrelated target, deletes the only runnable binary, crosses filesystems, or claims to restore database/configuration state.
            escalation: Preserve both files and return explicit manual rename instructions rather than forcing a destructive recovery.
      - wave: 3
        tasks:
          - task: Complete deterministic negative-path and concurrency coverage for network loss, bad status/body, oversize/truncation, checksum/signature mismatch, failed probe, permission denial, apply/rollback failure, stale backup, and unsupported platform invariants.
            requirements: [R4, R5, R6, R7, R8, R9, R11, R13]
            dependencies: wave: 2
            expected_output: Every accepted failure mode proves that the installed temporary target remains byte-for-byte unchanged or is restored and that all HTTP/temp resources close cleanly.
            checks:
              - go test ./internal/updater/... -count=1
              - go test -race ./internal/updater/... -count=1
            stop_conditions: Stop if tests depend on public GitHub, real release credentials, sleeps instead of bounded contexts, or mutation of a non-temporary executable.
            escalation: Add injectable filesystem/HTTP/probe seams only where a concrete failure cannot otherwise be reproduced; avoid a general updater framework.

  - phase_slug: operator-update-cli
    story_id: 01KZZWB3F8S0GEMTWQ03JFB41G
    status: planned
    goal: Expose version, update check, signed apply, and rollback commands before runtime initialization with explicit platform, permission, and service-restart behavior.
    depends_on: self-update-engine
    allowed_surfaces:
      - cmd/server/main.go
      - cmd/server/self_update.go (new)
      - cmd/server/self_update_test.go and focused package-main tests
      - internal/buildinfo/buildinfo.go only if a compatibility-only helper is required
      - README.md operator update/rollback usage
    avoided_surfaces:
      - normal server flags and login behavior beyond moving the banner behind early dispatch
      - cmd/server/db_runtime.go database command behavior
      - internal/api/handlers/management latest-version response and frontend notification flow
      - automatic sudo, systemctl calls, Postgres access, config mutation, or background goroutines
    waves:
      - wave: 1
        tasks:
          - task: Add `runVersion` and `runSelfUpdate` command functions using isolated `flag.FlagSet`s and inject the updater engine/output dependencies so command tests return exit codes instead of calling `os.Exit` internally.
            requirements: [R1, R2, R4, R10, R11, R13]
            dependencies: none
            expected_output: `version [--short]`, `update --check`, `update`, and `update rollback` have deterministic usage, stdout/stderr, and 0/1/2 exit behavior without server/runtime imports in the engine.
            checks:
              - go test ./cmd/server/... -run 'Test(VersionCommand|UpdateCommand|UpdateCheck|RollbackCommand|Usage)' -count=1
            stop_conditions: Stop if command parsing introduces Cobra/another framework, accepts downgrade/force flags, or requires global flag state/runtime configuration.
            escalation: Match the existing `flag.NewFlagSet` pattern from database commands and keep command-specific parsing in one new file.
          - task: Move early positional dispatch ahead of the normal startup banner and Postgres path while preserving existing `init-db-from-env`, `migrate-local-to-db`, server flags, login modes, and normal banner output.
            requirements: [R1, R2, R10]
            dependencies: task: Add `runVersion` and `runSelfUpdate` command functions using isolated `flag.FlagSet`s and inject the updater engine/output dependencies so command tests return exit codes instead of calling `os.Exit` internally.
            expected_output: Early commands run with no database environment and clean `version --short` output; ordinary server startup still prints current build metadata and follows its existing path.
            checks:
              - go test ./cmd/server/... -run 'Test(EarlyCommandDispatch|VersionCommand|ExistingDBCommands)' -count=1
              - env -u PGSTORE_DSN -u pgstore_dsn go run ./cmd/server version --short
            stop_conditions: Stop if existing database subcommands or normal flags change behavior, or an early command reaches `loadRuntimeConfigFromPostgres`.
            escalation: Isolate a small `dispatchEarlyCommand(args)` function and leave the rest of `main` untouched.
      - wave: 2
        tasks:
          - task: Map engine outcomes to concise operator output for current/latest versions, unsupported platforms, permission/network/verification failures, successful disk replacement, retained backup, and explicit process/service restart guidance without invoking restart or privilege escalation.
            requirements: [R3, R5, R6, R7, R9, R10, R11]
            dependencies: wave: 1
            expected_output: Operators can distinguish no update, available update, rejected update, completed replacement, and rollback recovery from output and exit code; success states that the running process still uses the old image.
            checks:
              - go test ./cmd/server/... -run 'Test(UpdateOutput|UpdateFailureOutput|RestartMessage|UnsupportedPlatform)' -count=1
              - Confirm tests assert no `sudo`, `systemctl`, database loader, or configuration mutation call is made.
            stop_conditions: Stop if output leaks remote bodies/signature material, claims the running process changed immediately, or suggests automatic data rollback.
            escalation: Return typed engine outcomes and format them only in the command adapter rather than adding logging/config coupling to the engine.
          - task: Document operator check/update/restart/rollback commands, Linux/macOS support, `/usr/local/bin` permission expectations, `.previous` recovery invocation, package-manager caveat, and binary-only rollback boundary in README.
            requirements: [R9, R10, R11]
            dependencies: task: Map engine outcomes to concise operator output for current/latest versions, unsupported platforms, permission/network/verification failures, successful disk replacement, retained backup, and explicit process/service restart guidance without invoking restart or privilege escalation.
            expected_output: The documented commands exactly match implemented parsing and clearly separate update, process restart, and data recovery responsibilities.
            checks:
              - Run each non-mutating documented command against a local build and compare output/exit behavior.
              - git diff --check
            stop_conditions: Stop if documentation recommends running the long-lived service as root, self-updating package-managed binaries, or treating `.previous` as a database backup.
            escalation: Keep instructions to explicit CLI and existing installer/systemd conventions; defer package-manager-specific flows.

  - phase_slug: self-update-verification-gate
    story_id: 01KZZWBCBT2Y3Z3YY01VF8WJPE
    status: planned
    goal: Prove the signed asset set, updater failure invariants, supported-platform behavior, and full repository quality gates before release activation.
    depends_on: operator-update-cli
    allowed_surfaces:
      - focused corrections in surfaces owned by the preceding three phases
      - local `dist/` and temporary signing/test paths created by verification commands
      - repository test, build, release-check, and diff outputs
    avoided_surfaces:
      - new product behavior, broader platform support, live provider/database dependencies, or frontend tests
      - publishing a real tag/release without separate explicit outward-action approval
      - weakening checks, deleting evidence, or committing any signing secret
    waves:
      - wave: 1
        tasks:
          - task: Run focused engine and CLI suites with race detection, including end-to-end local signed-update fixtures that prove unchanged/restored target bytes across every rejection and rollback path.
            requirements: [R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, R11, R13]
            dependencies: release-signing-foundation, self-update-engine, operator-update-cli
            expected_output: Focused tests pass without live GitHub/Postgres access, secret material, current-executable mutation, leaked temporary files, or races.
            checks:
              - go test ./internal/updater/... ./cmd/server/... -count=1
              - go test -race ./internal/updater/... ./cmd/server/... -count=1
            stop_conditions: Stop on any failure or if a fixture's target escapes its temporary directory; do not proceed to release/build gates.
            escalation: Return the failure to the phase that owns the affected invariant and preserve the exact command/output.
          - task: Validate unsigned and signed GoReleaser snapshots, then inspect the exact platform asset, checksum, and minisign sidecar matrix with a non-production or explicitly authorized key.
            requirements: [R3, R6, R7, R11, R12, R14]
            dependencies: release-signing-foundation, self-update-engine, operator-update-cli
            expected_output: Snapshot output contains all current release binaries and checksums; signed snapshot output contains a valid `.minisig` for each binary, and the updater's embedded production key path is distinct from non-production verification keys.
            checks:
              - make release-check
              - make release-snapshot
              - MINISIGN_KEY_PATH=<authorized-test-key> MINISIGN_PUBLIC_KEY_PATH=<matching-public-key> make release-signed-snapshot
              - scripts/verify-release-assets.sh dist <matching-public-key>
            stop_conditions: Stop if an asset is missing/renamed, a signature verifies against the wrong key, snapshot signing requires committing a private key, or GoReleaser output differs from updater lookup.
            escalation: Leave all release output local, correct the owning release/updater contract, and rerun the full matrix before any tag is created.
      - wave: 2
        tasks:
          - task: Run repository-wide tests, vet, frontend production build/embed, backend build, and cross-platform release compilation after focused security checks pass.
            requirements: [R14]
            dependencies: wave: 1
            expected_output: All repository packages and existing frontend/backend release paths pass with the pinned updater dependency and embedded public key.
            checks:
              - go test ./...
              - go vet ./...
              - make build-web
              - make build
              - make release-check
              - make release-snapshot
            stop_conditions: Stop on any command failure or skipped platform build; do not attribute unrelated output without proving it predates the initiative.
            escalation: Fix only regressions introduced by the planned surfaces or report the exact external/pre-existing blocker with proof.
          - task: Inspect dependency, secret, generated-asset, and diff hygiene before handing the implementation to the normal check/git workflow.
            requirements: [R7, R12, R14]
            dependencies: task: Run repository-wide tests, vet, frontend production build/embed, backend build, and cross-platform release compilation after focused security checks pass.
            expected_output: The diff is surgical, dependency versions are pinned, no production private key or generated release binary is tracked, and the working tree contains only intended source/config/documentation changes.
            checks:
              - go list -m github.com/minio/selfupdate golang.org/x/mod
              - test -z "$(git ls-files | grep -E '(^|/)(release|minisign)([-_.].*)?\.key$' || true)"
              - git diff --check
              - git status --short
            stop_conditions: Stop if a secret-like key file, generated release asset, unrelated refactor, failing check, or unexplained worktree change remains.
            escalation: Preserve user-owned changes, remove only initiative-generated residue through the repository-safe cleanup workflow, and rerun affected checks.
      - wave: 3
        tasks:
          - task: When separately authorized for an outward-facing release, exercise the protected tag workflow, verify the GitHub draft contains and cryptographically validates every required asset, then publish or leave the draft blocked with exact evidence.
            requirements: [R5, R6, R7, R11, R12]
            dependencies: wave: 2
            expected_output: The first authorized signed release is published only after remote draft verification, or remains unpublished with a precise missing/invalid asset report.
            checks:
              - gh release view <authorized-tag> --json isDraft,tagName,assets,url
              - Download the draft assets to an isolated temporary directory and run `scripts/verify-release-assets.sh <directory> internal/updater/release.pub` before publication.
              - Confirm `GET /repos/therealtinhtute/llmhub/releases/latest` does not expose the draft and exposes the release only after successful publication.
            stop_conditions: Do not create/push a tag, modify GitHub environment secrets, or publish a release without explicit authorization; never publish after a failed remote signature/asset check.
            escalation: Leave the release draft and route the exact failure to release-signing-foundation; rotate the key if authenticity rather than packaging failed.

## Progress
<!-- Append-only durable entries record timestamp, phase, wave, task, task_status, run_id, trace_id, exact verification/result, and changed surfaces or blocker. -->
- none

## Decisions
<!-- Append-only durable entries record timestamp, phase/task, decision, and rationale. -->
- none

## Validation
<!-- Append-only durable entries record timestamp, phase, exact command/result/output, run_id, check_id, verdict, and proof_gaps. -->
- none

## Current State and Next Action
- active_phase: release-signing-foundation
- lifecycle_status: planned
- latest_run_id: none
- latest_trace_ids: []
- latest_check_id: none
- latest_handoff_id: none
- blockers: none
- open_items: [work phases release-signing-foundation, self-update-engine, operator-update-cli, self-update-verification-gate]
- exact_next_action: work full phase release-signing-foundation
