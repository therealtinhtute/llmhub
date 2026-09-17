package cliproxy

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/sdk/config"
)

func TestApplyOAuthModelAlias_Rename(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out))
	}
	if out[0].ID != "g5" {
		t.Fatalf("expected model id %q, got %q", "g5", out[0].ID)
	}
	if out[0].Name != "models/g5" {
		t.Fatalf("expected model name %q, got %q", "models/g5", out[0].Name)
	}
}

func TestApplyOAuthModelAlias_ForkAddsAlias(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5", Fork: true},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 2 {
		t.Fatalf("expected 2 models, got %d", len(out))
	}
	if out[0].ID != "gpt-5" {
		t.Fatalf("expected first model id %q, got %q", "gpt-5", out[0].ID)
	}
	if out[1].ID != "g5" {
		t.Fatalf("expected second model id %q, got %q", "g5", out[1].ID)
	}
	if out[1].Name != "models/g5" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5", out[1].Name)
	}
}

func TestApplyOAuthModelAlias_ForkAddsMultipleAliases(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5", Fork: true},
				{Name: "gpt-5", Alias: "g5-2", Fork: true},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 3 {
		t.Fatalf("expected 3 models, got %d", len(out))
	}
	if out[0].ID != "gpt-5" {
		t.Fatalf("expected first model id %q, got %q", "gpt-5", out[0].ID)
	}
	if out[1].ID != "g5" {
		t.Fatalf("expected second model id %q, got %q", "g5", out[1].ID)
	}
	if out[1].Name != "models/g5" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5", out[1].Name)
	}
	if out[2].ID != "g5-2" {
		t.Fatalf("expected third model id %q, got %q", "g5-2", out[2].ID)
	}
	if out[2].Name != "models/g5-2" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5-2", out[2].Name)
	}
}

// Ported from upstream CLIProxyAPI commit 4311ae874774
// (sdk/cliproxy/service_oauth_model_alias_test.go): prefixed catalog clones must
// inherit a deep copy of NativeCapabilities so mutating the clone cannot corrupt
// the source model's capability metadata. The fork has no MetadataModelID field,
// so only the capability assertions of the upstream test are carried over.
func TestApplyModelPrefixes_ClonesNativeCapabilities(t *testing.T) {
	webSearch := true
	models := []*ModelInfo{
		{ID: "gpt-6-astra"},
		{ID: "codex-main", NativeCapabilities: &registry.NativeCapabilities{WebSearch: &webSearch}},
	}

	out := applyModelPrefixes(models, "1", false)
	entryMap := make(map[string]*ModelInfo, len(out))
	for _, entry := range out {
		if entry == nil {
			continue
		}
		entryMap[entry.ID] = entry
	}

	m, ok := entryMap["1/codex-main"]
	if !ok {
		t.Fatal("missing 1/codex-main")
	} else if m.NativeCapabilities == nil || m.NativeCapabilities.WebSearch == nil || !*m.NativeCapabilities.WebSearch {
		t.Fatalf("1/codex-main did not inherit native capabilities: %+v", m)
	} else {
		*m.NativeCapabilities.WebSearch = false
		if !*entryMap["codex-main"].NativeCapabilities.WebSearch {
			t.Fatal("prefixed capability metadata aliases the source model")
		}
	}
}

// Ported from upstream CLIProxyAPI commit 8335eac7
// (sdk/cliproxy/service_oauth_model_alias_test.go TestApplyOAuthModelAlias_Meta).
func TestApplyOAuthModelAlias_Meta(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"meta": {
				{Name: "muse-spark-1.3", Alias: "muse-latest", DisplayName: "Muse Latest"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "muse-spark-1.3", Name: "models/muse-spark-1.3", DisplayName: "Muse Spark 1.3"},
	}

	out := applyOAuthModelAlias(cfg, "meta", "oauth", models)
	if len(out) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out))
	}
	if out[0].ID != "muse-latest" {
		t.Fatalf("expected model id %q, got %q", "muse-latest", out[0].ID)
	}
	if out[0].Name != "models/muse-latest" {
		t.Fatalf("expected model name %q, got %q", "models/muse-latest", out[0].Name)
	}
	if out[0].DisplayName != "Muse Latest" {
		t.Fatalf("expected display name %q, got %q", "Muse Latest", out[0].DisplayName)
	}
}
