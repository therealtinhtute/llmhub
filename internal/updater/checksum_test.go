package updater

import (
	"strings"
	"testing"
)

const goodDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestChecksumParsesExactEntry(t *testing.T) {
	body := goodDigest + "  llmhub-linux-amd64\n" +
		"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  llmhub-darwin-arm64\n"
	got, err := ParseChecksum(body, "llmhub-linux-amd64")
	if err != nil || got != goodDigest {
		t.Fatalf("ParseChecksum = %q, %v", got, err)
	}
}

func TestChecksumDuplicateEntries(t *testing.T) {
	body := goodDigest + "  llmhub-linux-amd64\n" +
		goodDigest + "  llmhub-linux-amd64\n"
	if _, err := ParseChecksum(body, "llmhub-linux-amd64"); err == nil {
		t.Fatal("ParseChecksum accepted duplicate entries")
	}
}

func TestChecksumMissingEntry(t *testing.T) {
	body := goodDigest + "  llmhub-darwin-arm64\n"
	if _, err := ParseChecksum(body, "llmhub-linux-amd64"); err == nil {
		t.Fatal("ParseChecksum accepted a missing entry")
	}
}

func TestChecksumMalformedDigest(t *testing.T) {
	for _, digest := range []string{"short", "xyz" + strings.Repeat("0", 61)} {
		if _, err := ParseChecksum(digest+"  llmhub-linux-amd64\n", "llmhub-linux-amd64"); err == nil {
			t.Fatalf("ParseChecksum accepted malformed digest %q", digest)
		}
	}
}

func TestChecksumUppercaseDigestNormalized(t *testing.T) {
	upper := strings.ToUpper(goodDigest)
	got, err := ParseChecksum(upper+"  llmhub-linux-amd64\n", "llmhub-linux-amd64")
	if err != nil || got != goodDigest {
		t.Fatalf("ParseChecksum = %q, %v; want lowercase %q", got, err, goodDigest)
	}
}
