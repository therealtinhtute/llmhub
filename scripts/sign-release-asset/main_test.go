package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadProtectedFileRejectsPermissiveMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX permission bits")
	}

	path := filepath.Join(t.TempDir(), "secret-input")
	if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("seed protected input: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("set protected input mode: %v", err)
	}
	if _, err := readProtectedFile(path); err == nil {
		t.Fatal("readProtectedFile() accepted a group/world-readable file")
	}
}

func TestReadProtectedFileAcceptsOwnerOnlyMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX permission bits")
	}

	path := filepath.Join(t.TempDir(), "secret-input")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("seed protected input: %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("set protected input mode: %v", err)
	}
	contents, err := readProtectedFile(path)
	if err != nil {
		t.Fatalf("readProtectedFile() error = %v", err)
	}
	if string(contents) != "fixture" {
		t.Fatalf("protected input contents = %q, want fixture", contents)
	}
}

func TestResolveArtifactPathFromGoReleaserSignaturePath(t *testing.T) {
	directory := t.TempDir()
	artifactPath := filepath.Join(directory, "llmhub")
	signaturePath := artifactPath + ".minisig"
	if err := os.WriteFile(artifactPath, []byte("fixture"), 0o755); err != nil {
		t.Fatalf("seed artifact: %v", err)
	}

	resolved, err := resolveArtifactPath("llmhub", signaturePath)
	if err != nil {
		t.Fatalf("resolveArtifactPath() error = %v", err)
	}
	if resolved != artifactPath {
		t.Fatalf("resolved artifact path = %q, want %q", resolved, artifactPath)
	}
}

func TestWriteSignatureAtomically(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows directory sync semantics differ")
	}

	path := filepath.Join(t.TempDir(), "asset.minisig")
	if err := writeSignatureAtomically(path, []byte("signature\n")); err != nil {
		t.Fatalf("writeSignatureAtomically() error = %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read signature: %v", err)
	}
	if string(contents) != "signature\n" {
		t.Fatalf("signature contents = %q, want signature", contents)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat signature: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("signature mode = %04o, want 0644", got)
	}
}
