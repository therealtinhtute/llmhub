package updater

import "errors"

// Typed outcomes the CLI maps to exit codes. All remote data is untrusted;
// these fail closed.
var (
	// ErrDevelopmentBuild means the running binary has no release version
	// (e.g. "dev") and must never be replaced silently.
	ErrDevelopmentBuild = errors.New("running a development build; self-update unavailable")
	// ErrUpToDate means the stable channel already matches the running version.
	ErrUpToDate = errors.New("already up to date")
	// ErrDowngradeRefused means stable is older than the running version.
	ErrDowngradeRefused = errors.New("latest release is older than the running version; downgrade refused")
	// ErrUnsupportedPlatform means the running platform has no self-update support.
	ErrUnsupportedPlatform = errors.New("self-update is unsupported on this platform")
)
