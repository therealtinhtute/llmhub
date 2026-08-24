package misc

import (
	"fmt"
	"testing"
	"time"
)

func TestAntigravityLatestVersionUsesCurrentFallback(t *testing.T) {
	cached := cachedAntigravityVersion
	expiry := antigravityVersionExpiry
	defer func() {
		antigravityVersionMu.Lock()
		cachedAntigravityVersion = cached
		antigravityVersionExpiry = expiry
		antigravityVersionMu.Unlock()
	}()

	antigravityVersionMu.Lock()
	cachedAntigravityVersion = "9.9.9"
	antigravityVersionExpiry = time.Now().Add(-time.Minute)
	antigravityVersionMu.Unlock()

	version := AntigravityLatestVersion()
	if version != antigravityFallbackVersion {
		t.Fatalf("AntigravityLatestVersion() = %q, want %q", version, antigravityFallbackVersion)
	}
}

// The Antigravity backend resolves newer models only for clients reporting at
// least 2.9.0; older versions get 404 Requested entity was not found.
func TestAntigravityFallbackVersionMeetsBackendFloor(t *testing.T) {
	const floorMajor, floorMinor = 2, 9

	var major, minor, patch int
	if _, err := fmt.Sscanf(antigravityFallbackVersion, "%d.%d.%d", &major, &minor, &patch); err != nil {
		t.Fatalf("antigravityFallbackVersion = %q is not a dotted version: %v", antigravityFallbackVersion, err)
	}
	if major < floorMajor || (major == floorMajor && minor < floorMinor) {
		t.Fatalf("antigravityFallbackVersion = %q, want at least %d.%d.0", antigravityFallbackVersion, floorMajor, floorMinor)
	}
}
