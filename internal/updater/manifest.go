package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StagedManifest records what llmhub.staged is: the normalized release
// version and the lowercase-hex SHA-256 digest it was verified against. The
// root-run apply step re-reads it and re-verifies independently.
type StagedManifest struct {
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

// WriteStagedManifest atomically writes staged.json in updateDir.
func WriteStagedManifest(updateDir string, m StagedManifest) error {
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("encoding staged manifest: %w", err)
	}
	tmp, err := os.CreateTemp(updateDir, ".manifest-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("writing staged manifest: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(updateDir, stagedManifestName)); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// ReadStagedManifest reads staged.json from updateDir.
func ReadStagedManifest(updateDir string) (StagedManifest, error) {
	var m StagedManifest
	b, err := os.ReadFile(filepath.Join(updateDir, stagedManifestName))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("decoding staged manifest: %w", err)
	}
	return m, nil
}
