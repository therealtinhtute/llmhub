package presets

import "testing"

func TestAllPresetsHaveIDAndBaseURL(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range All() {
		if p.ID == "" {
			t.Fatalf("preset has empty id: %+v", p)
		}
		if p.BaseURL == "" {
			t.Fatalf("preset %q has empty base_url", p.ID)
		}
		if seen[p.ID] {
			t.Fatalf("duplicate preset id: %q", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestVerifiedPresetsHaveVerifiedAt(t *testing.T) {
	for _, p := range All() {
		if p.Verified && p.VerifiedAt == "" {
			t.Fatalf("preset %q is verified but has no verified_at", p.ID)
		}
	}
}

func TestAllReturnsSeedCatalog(t *testing.T) {
	presets := All()
	if len(presets) != 3 {
		t.Fatalf("All() returned %d presets, want 3", len(presets))
	}
}
