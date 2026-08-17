package updater

import "testing"

func TestAssetSelectionExact(t *testing.T) {
	rel := Release{Tag: "v1.2.3", Assets: []Asset{
		{Name: "llmhub-linux-amd64"},
		{Name: "llmhub-linux-arm64"},
	}}
	got, err := SelectAsset(rel, "llmhub-linux-arm64")
	if err != nil || got.Name != "llmhub-linux-arm64" {
		t.Fatalf("SelectAsset = %+v, %v", got, err)
	}
}

func TestAssetSelectionMissing(t *testing.T) {
	rel := Release{Tag: "v1.2.3", Assets: []Asset{{Name: "llmhub-linux-amd64"}}}
	if _, err := SelectAsset(rel, "llmhub-darwin-arm64"); err == nil {
		t.Fatal("SelectAsset accepted a missing asset")
	}
}

func TestAssetSelectionDuplicate(t *testing.T) {
	rel := Release{Tag: "v1.2.3", Assets: []Asset{
		{Name: "llmhub-linux-amd64"},
		{Name: "llmhub-linux-amd64"},
	}}
	if _, err := SelectAsset(rel, "llmhub-linux-amd64"); err == nil {
		t.Fatal("SelectAsset accepted duplicate assets")
	}
}
