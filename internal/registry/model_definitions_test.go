package registry

import (
	"reflect"
	"testing"
)

func TestGrok45ModelDefinition(t *testing.T) {
	got := requireModelDefinition(t, GetXAIModels(), "grok-4.5")
	want := &ModelInfo{
		ID:                  "grok-4.5",
		Object:              "model",
		Created:             1783526400,
		OwnedBy:             "xai",
		Type:                "xai",
		DisplayName:         "Grok 4.5",
		Name:                "grok-4.5",
		Description:         "SpaceXAI's intelligent coding model for agentic software, engineering, and workflow tasks.",
		ContextLength:       500000,
		MaxCompletionTokens: 65536,
		Thinking: &ThinkingSupport{
			Levels: []string{"low", "medium", "high"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("grok-4.5 definition = %#v, want %#v", got, want)
	}
}

func TestGeminiProductionModelDefinitions(t *testing.T) {
	generationMethods := []string{
		"generateContent",
		"countTokens",
		"createCachedContent",
		"batchGenerateContent",
	}
	tests := []struct {
		id          string
		created     int64
		displayName string
		version     string
		description string
		levels      []string
	}{
		{
			id:          "gemini-3-pro",
			created:     1737158400,
			displayName: "Gemini 3 Pro",
			version:     "3.0",
			description: "Gemini 3 Pro",
			levels:      []string{"low", "high"},
		},
		{
			id:          "gemini-3-flash",
			created:     1765929600,
			displayName: "Gemini 3 Flash",
			version:     "3.0",
			description: "Our most intelligent model built for speed, combining frontier intelligence with superior search and grounding.",
			levels:      []string{"minimal", "low", "medium", "high"},
		},
		{
			id:          "gemini-3.1-pro",
			created:     1771459200,
			displayName: "Gemini 3.1 Pro",
			version:     "3.1",
			description: "Gemini 3.1 Pro",
			levels:      []string{"low", "medium", "high"},
		},
		{
			id:          "gemini-3.1-flash-image",
			created:     1771459200,
			displayName: "Gemini 3.1 Flash Image",
			version:     "3.1",
			description: "Gemini 3.1 Flash Image",
			levels:      []string{"minimal", "high"},
		},
		{
			id:          "gemini-3.1-flash-lite",
			created:     1776288000,
			displayName: "Gemini 3.1 Flash Lite",
			version:     "3.1",
			description: "Our smallest and most cost effective model, built for at scale usage.",
			levels:      []string{"minimal", "low", "medium", "high"},
		},
		{
			id:          "gemini-3-pro-image",
			created:     1737158400,
			displayName: "Gemini 3 Pro Image",
			version:     "3.0",
			description: "Gemini 3 Pro Image",
			levels:      []string{"low", "high"},
		},
	}

	models := GetGeminiVertexModels()
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := requireModelDefinition(t, models, tt.id)
			want := &ModelInfo{
				ID:                         tt.id,
				Object:                     "model",
				Created:                    tt.created,
				OwnedBy:                    "google",
				Type:                       "gemini",
				DisplayName:                tt.displayName,
				Name:                       "models/" + tt.id,
				Version:                    tt.version,
				Description:                tt.description,
				InputTokenLimit:            1048576,
				OutputTokenLimit:           65536,
				SupportedGenerationMethods: generationMethods,
				Thinking: &ThinkingSupport{
					Min:            128,
					Max:            32768,
					DynamicAllowed: true,
					Levels:         tt.levels,
				},
			}

			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s definition = %#v, want %#v", tt.id, got, want)
			}
		})
	}
}

func TestGeminiProductionAndPreviewModelsCoexist(t *testing.T) {
	pairs := []struct {
		production string
		preview    string
	}{
		{production: "gemini-3-pro", preview: "gemini-3-pro-preview"},
		{production: "gemini-3-flash", preview: "gemini-3-flash-preview"},
		{production: "gemini-3.1-pro", preview: "gemini-3.1-pro-preview"},
		{production: "gemini-3.1-flash-image", preview: "gemini-3.1-flash-image-preview"},
		{production: "gemini-3.1-flash-lite", preview: "gemini-3.1-flash-lite-preview"},
		{production: "gemini-3-pro-image", preview: "gemini-3-pro-image-preview"},
	}

	models := GetGeminiVertexModels()
	for _, pair := range pairs {
		t.Run(pair.production, func(t *testing.T) {
			requireModelDefinition(t, models, pair.production)
			requireModelDefinition(t, models, pair.preview)
		})
	}
}

func requireModelDefinition(t *testing.T, models []*ModelInfo, id string) *ModelInfo {
	t.Helper()
	for _, model := range models {
		if model != nil && model.ID == id {
			return model
		}
	}
	t.Fatalf("model %q not found", id)
	return nil
}
