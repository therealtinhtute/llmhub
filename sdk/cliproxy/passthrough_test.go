package cliproxy

import (
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
)

func TestBuildOpenAICompatibilityConfigModels_PassthroughEmptyModelsContributesNone(t *testing.T) {
	models := buildOpenAICompatibilityConfigModels(&config.OpenAICompatibility{
		Name:        "compat",
		Passthrough: true,
	})
	if len(models) != 0 {
		t.Fatalf("expected no model entries for passthrough with empty Models, got %d", len(models))
	}
}

func TestBuildOpenAICompatibilityConfigModels_PassthroughNonEmptyModelsUnchanged(t *testing.T) {
	models := buildOpenAICompatibilityConfigModels(&config.OpenAICompatibility{
		Name:        "compat",
		Passthrough: true,
		Models: []config.OpenAICompatibilityModel{
			{Name: "compat-upstream", Alias: "compat-alias"},
		},
	})
	if findModelInfoByID(models, "compat-alias") == nil {
		t.Fatalf("expected explicit model to still be advertised under passthrough, got %+v", models)
	}
}
