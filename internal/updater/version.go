package updater

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// Decision is the outcome of comparing the running version with the latest
// release tag.
type Decision string

const (
	// DecisionDevelopmentBuild means the running binary carries a non-semver
	// version (e.g. "dev") and must never replace itself silently.
	DecisionDevelopmentBuild Decision = "development-build"
	// DecisionUpdateAvailable means the stable channel has a newer version.
	DecisionUpdateAvailable Decision = "update-available"
	// DecisionUpToDate means the running version already matches stable.
	DecisionUpToDate Decision = "up-to-date"
	// DecisionDowngradeRefused means stable is older than the running version.
	DecisionDowngradeRefused Decision = "downgrade-refused"
)

// Evaluate compares the running version against the latest release tag.
// current is the linker-injected buildinfo string; tag is untrusted remote
// metadata. A malformed current version is a development build, never an
// error; a malformed or prerelease tag is an error that fails closed.
func Evaluate(current, tag string) (Decision, error) {
	cur, err := normalizeVersion(current)
	if err != nil {
		return DecisionDevelopmentBuild, nil
	}
	latest, err := normalizeStable(tag)
	if err != nil {
		return "", err
	}
	switch semver.Compare(latest, cur) {
	case 1:
		return DecisionUpdateAvailable, nil
	case 0:
		return DecisionUpToDate, nil
	default:
		return DecisionDowngradeRefused, nil
	}
}

// CanonicalVersion validates v and returns its canonical "v"-prefixed form,
// which is what `version --short` prints and the probe compares against.
func CanonicalVersion(v string) (string, error) {
	return normalizeVersion(v)
}

// normalizeVersion validates the running binary's version and returns its
// canonical "v"-prefixed form, which x/mod/semver requires. Prereleases are
// accepted: an operator running v1.3.0-rc1 may legitimately upgrade to v1.3.0.
func normalizeVersion(v string) (string, error) {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) || semver.Canonical(v) != v {
		// semver.IsValid alone accepts shorthand like "v1.2"; a release
		// version must be in canonical form.
		return "", fmt.Errorf("malformed version %q", v)
	}
	return v, nil
}

// normalizeStable validates a release tag and excludes prereleases from the
// stable channel.
func normalizeStable(tag string) (string, error) {
	v, err := normalizeVersion(tag)
	if err != nil {
		return "", err
	}
	if prerelease := semver.Prerelease(v); prerelease != "" {
		return "", fmt.Errorf("prerelease tag %q excluded from stable channel", tag)
	}
	return v, nil
}
