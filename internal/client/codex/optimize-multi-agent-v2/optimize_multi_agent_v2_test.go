package multiagentv2

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
)

func TestCodexSpawnAgentModelsCacheInvalidation(t *testing.T) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID1 := "cache-invalidation-client-1"
	clientID2 := "cache-invalidation-client-2"

	// 1. Initial registration
	modelRegistry.RegisterClient(clientID1, "openai", []*registry.ModelInfo{
		{
			ID:          "test-spawn-model-alpha",
			DisplayName: "Test Spawn Model Alpha",
			Description: "Initial description.",
			Thinking: &registry.ThinkingSupport{
				Levels: []string{"low", "medium"},
			},
		},
	})
	t.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID1)
		modelRegistry.UnregisterClient(clientID2)
	})

	formatted1 := formatCodexSpawnAgentModelsForRequest(context.Background(), nil, false)
	if !strings.Contains(formatted1, "test-spawn-model-alpha") {
		t.Fatalf("expected initial markdown to contain test-spawn-model-alpha, got: %s", formatted1)
	}
	if !strings.Contains(formatted1, "Reasoning efforts: low, medium") {
		t.Fatalf("expected initial reasoning efforts low, medium, got: %s", formatted1)
	}

	// 2. Cache hit returns identical content
	formattedHit := formatCodexSpawnAgentModelsForRequest(context.Background(), nil, false)
	if formattedHit != formatted1 {
		t.Fatalf("cache hit expected identical output, got %s vs %s", formattedHit, formatted1)
	}

	// 3. Registering second model invalidates cache
	modelRegistry.RegisterClient(clientID2, "openai", []*registry.ModelInfo{
		{
			ID:          "test-spawn-model-beta",
			DisplayName: "Test Spawn Model Beta",
			Description: "Second model.",
		},
	})

	formatted2 := formatCodexSpawnAgentModelsForRequest(context.Background(), nil, false)
	if !strings.Contains(formatted2, "test-spawn-model-beta") {
		t.Fatalf("expected cache invalidation to include test-spawn-model-beta, got: %s", formatted2)
	}

	// 4. Modifying model thinking levels invalidates cache
	modelRegistry.RegisterClient(clientID1, "openai", []*registry.ModelInfo{
		{
			ID:          "test-spawn-model-alpha",
			DisplayName: "Test Spawn Model Alpha",
			Description: "Initial description.",
			Thinking: &registry.ThinkingSupport{
				Levels: []string{"low", "medium", "high", "max"},
			},
		},
	})

	formatted3 := formatCodexSpawnAgentModelsForRequest(context.Background(), nil, false)
	if !strings.Contains(formatted3, "low, medium (default), high, max") {
		t.Fatalf("expected updated thinking levels to reflect in markdown, got: %s", formatted3)
	}

	// 5. Unregistering client invalidates cache
	modelRegistry.UnregisterClient(clientID2)
	formatted4 := formatCodexSpawnAgentModelsForRequest(context.Background(), nil, false)
	if strings.Contains(formatted4, "test-spawn-model-beta") {
		t.Fatalf("expected test-spawn-model-beta to be removed after unregistering, got: %s", formatted4)
	}
}

func BenchmarkCodexSpawnAgentModelsForRequest(b *testing.B) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID := "bench-client-models"
	modelRegistry.RegisterClient(clientID, "openai", []*registry.ModelInfo{
		{
			ID:          "gpt-5.5",
			DisplayName: "Default model",
			Description: "Default model description.",
		},
		{
			ID:          "claude-3-7-sonnet",
			DisplayName: "Claude 3.7 Sonnet",
			Description: "Claude model description.",
		},
	})
	b.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codexSpawnAgentModelsForRequest(ctx, nil, false)
	}
}

func BenchmarkPrepareCodexMultiAgentV2Tools(b *testing.B) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID := "bench-client-prepare"
	modelRegistry.RegisterClient(clientID, "openai", []*registry.ModelInfo{
		{
			ID:          "gpt-5.5",
			DisplayName: "Default model",
			Description: "Default model description.",
		},
	})
	b.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	payload := []byte(`{
		"tools":[
			{"type":"namespace","name":"collaboration","tools":[
				{"type":"function","name":"spawn_agent","description":"Spawns an agent.\n","parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}},
				{"type":"function","name":"send_message","parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}},
				{"type":"function","name":"followup_task","parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}}
			]}
		]
	}`)
	headers := http.Header{"User-Agent": []string{"Codex Desktop/0.146.0-alpha.3"}}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PrepareCodexMultiAgentV2Tools(ctx, headers, payload, true, false)
	}
}

func BenchmarkOptimizeCodexMultiAgentV2Request(b *testing.B) {
	modelRegistry := registry.GetGlobalRegistry()
	clientID := "bench-client-opt"
	modelRegistry.RegisterClient(clientID, "openai", []*registry.ModelInfo{
		{
			ID:          "gpt-5.5",
			DisplayName: "Default model",
			Description: "Default model description.",
		},
	})
	b.Cleanup(func() {
		modelRegistry.UnregisterClient(clientID)
	})

	payload := []byte(`{
		"tools":[
			{"type":"namespace","name":"collaboration","tools":[
				{"type":"function","name":"spawn_agent","description":"Spawns an agent.\n","parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}},
				{"type":"function","name":"send_message","parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}},
				{"type":"function","name":"followup_task","parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}}
			]}
		]
	}`)
	headers := http.Header{"User-Agent": []string{"Codex Desktop/0.146.0-alpha.3"}}
	cfg := &config.Config{CodexOptimizeMultiAgentV2: true}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OptimizeCodexMultiAgentV2Request(ctx, headers, payload, cfg)
	}
}
