# Release signing

Release assets are signed with minisign detached signatures. The release job runs in the protected GitHub `release` environment and publishes only after a draft release passes local and remote verification.

## Local checks

Run the unsigned snapshot when validating build output only:

```bash
make release-check
make release-snapshot
```

The unsigned target intentionally skips signing and must not be used as a release artifact. A signed snapshot requires an encrypted private key, an owner-only password file, and a minisign public-key file:

```bash
MINISIGN_KEY_PATH=/secure/path/release.key \
MINISIGN_PASSWORD_FILE=/secure/path/release.password \
MINISIGN_PUBLIC_KEY_PATH=/secure/path/release.pub \
make release-signed-snapshot
```

`MINISIGN_KEY_PATH` and `MINISIGN_PASSWORD_FILE` must refer to regular mode-0600 files. The private key must remain passphrase-protected. `MINISIGN_PUBLIC_KEY_PATH` uses the normal two-line minisign public-key format. The target verifies all eight binary assets, their SHA-256 entries, and their `.minisig` sidecars before succeeding. `MINISIGN_BIN` can select a separately verified minisign 0.12 executable.

Never put private-key bytes or the password in the repository, command arguments, logs, normal repository variables, or build artifacts.

## Protected release workflow

The tag workflow uses the protected `release` environment and expects these environment secrets:

- `MINISIGN_PRIVATE_KEY_B64`: base64-encoded encrypted private key.
- `MINISIGN_PASSWORD`: passphrase for that key.
- `MINISIGN_PUBLIC_KEY`: raw public-key line; the workflow adds the required minisign comment line.

`GITHUB_TOKEN` is the GitHub-provided upload token. The workflow downloads minisign 0.12, verifies its release tarball with the pinned upstream public key, and uses the verified executable only for independent release-asset verification. The Go wrapper under `scripts/sign-release-asset/` handles encrypted-key signing without placing the passphrase in argv.

For a normal release, push an immutable `v*` tag. The workflow builds and signs locally, creates a draft release, uploads the exact eight binaries, eight sidecars, and `checksums.txt`, downloads the draft assets again, verifies the complete matrix, and publishes only after verification. It never overwrites assets under an existing tag. A failed verification leaves the release draft blocked.

`workflow_dispatch` accepts an existing immutable `v*` tag. Its optional `transition_public_key` input selects the public key used for that release's asset verification; it does not change the public key embedded in the application. Use it only after the transition binary already embeds the new trust key.

## Key lifecycle

Routine rotation is a two-release handoff:

1. Sign a transition binary with the old key while embedding the new public key in the application.
2. Verify that transition release through the old trust anchor.
3. Sign the next release with the new key and verify it with the transition key.

Treat release tags and published assets as immutable. Do not replace a sidecar, checksum, or binary beneath an existing tag.

If the signing key is suspected to be compromised, stop automatic updates, retain the last known-good release, rotate the protected key material, and ship trust-anchor changes only through a separately reviewed release. Recovery requires an out-of-band trusted reinstall from a verified source; never weaken signature verification or accept a runtime-supplied trust-on-first-use key.
