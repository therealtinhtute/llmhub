package updater

import "testing"

func TestPlatformLinux(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		got, err := AssetName("linux", arch)
		if err != nil || got != "llmhub-linux-"+arch {
			t.Fatalf("AssetName(linux, %s) = %q, %v", arch, got, err)
		}
	}
}

func TestPlatformDarwin(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		got, err := AssetName("darwin", arch)
		if err != nil || got != "llmhub-darwin-"+arch {
			t.Fatalf("AssetName(darwin, %s) = %q, %v", arch, got, err)
		}
	}
}

func TestPlatformRejectsWindows(t *testing.T) {
	if _, err := AssetName("windows", "amd64"); err == nil {
		t.Fatal("AssetName accepted Windows, want unsupported-platform error")
	}
}

func TestPlatformRejectsFreeBSD(t *testing.T) {
	if _, err := AssetName("freebsd", "amd64"); err == nil {
		t.Fatal("AssetName accepted FreeBSD, want unsupported-platform error")
	}
}

func TestPlatformRejectsUnknown(t *testing.T) {
	if _, err := AssetName("plan9", "amd64"); err == nil {
		t.Fatal("AssetName accepted plan9, want unsupported-platform error")
	}
	if _, err := AssetName("linux", "riscv64"); err == nil {
		t.Fatal("AssetName accepted riscv64, want unsupported-architecture error")
	}
}
