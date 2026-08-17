package updater

import "fmt"

// AssetName returns the exact release asset name for a platform, or an
// error naming the platform when self-update is unsupported there (R12).
func AssetName(goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin":
		switch goarch {
		case "amd64", "arm64":
			return "llmhub-" + goos + "-" + goarch, nil
		}
		return "", fmt.Errorf("unsupported architecture %q on %s", goarch, goos)
	case "windows":
		return "", fmt.Errorf("unsupported platform: Windows release binaries exist but self-update is not supported")
	case "freebsd":
		return "", fmt.Errorf("unsupported platform: FreeBSD release binaries exist but self-update is not supported")
	default:
		return "", fmt.Errorf("unsupported platform %q", goos)
	}
}
