package updater

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// ParseChecksum extracts the SHA-256 digest for asset from a GoReleaser
// checksums.txt body, requiring exactly one well-formed entry. The body is
// untrusted; any ambiguity fails closed.
func ParseChecksum(body, asset string) (string, error) {
	count := 0
	digest := ""
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != asset {
			continue
		}
		count++
		digest = fields[0]
	}
	if count != 1 {
		return "", fmt.Errorf("checksums.txt must contain exactly one entry for %q, found %d", asset, count)
	}
	if len(digest) != 64 {
		return "", fmt.Errorf("malformed checksum for %q: want 64 hex characters, got %d", asset, len(digest))
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", fmt.Errorf("malformed checksum for %q: %w", asset, err)
	}
	return strings.ToLower(digest), nil
}
