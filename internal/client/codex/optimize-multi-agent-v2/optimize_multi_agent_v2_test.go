package multiagentv2

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/tidwall/gjson"
)

func TestIsCodexMultiAgentClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{
			name:      "Codex Desktop",
			userAgent: "Codex Desktop/0.146.0-alpha.3 (Mac OS 26.5.2; arm64) unknown (Codex Desktop; 26.721.30844)",
			want:      true,
		},
		{
			name:      "codex tui",
			userAgent: "codex-tui/0.154.0 (Mac OS 26.5.2; arm64) iTerm.app/3.6.11 (codex-tui; 0.154.0)",
			want:      true,
		},
		{
			name:      "codex cli rs",
			userAgent: "codex_cli_rs/0.144.1 (Mac OS 26.3.1; arm64) iTerm.app/3.6.9",
			want:      true,
		},
		{
			name:      "bare codex cli rs",
			userAgent: "codex_cli_rs",
			want:      true,
		},
		{
			name:      "codex exec",
			userAgent: "codex_exec/0.153.2 (Mac OS 26.6.2; arm64) unknown (codex_exec; 0.153.2)",
			want:      true,
		},
		{
			name:      "other client",
			userAgent: "curl/8.7.1",
			want:      false,
		},
		{
			name:      "embedded token",
			userAgent: "proxy Codex Desktop/0.146.0",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isCodexMultiAgentClient(tt.userAgent); got != tt.want {
				t.Fatalf("isCodexMultiAgentClient(%q) = %v, want %v", tt.userAgent, got, tt.want)
			}
		})
	}
}

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
func TestRewriteCodexMultiAgentV2Input_StripsAuthorAndRecipient_Issue6136(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"model":"gpt-5.4","input":[{
		"type":"agent_message",
		"id":"amsg_1",
		"author":"/root",
		"recipient":"/root/worker",
		"content":[
			{"type":"input_text","text":"Message Type: NEW_TASK\nTask name: /root/worker\nSender: /root\nPayload:\n"},
			{"type":"encrypted_content","encrypted_content":"test task"}
		],
		"internal_chat_message_metadata_passthrough":{"turn_id":"turn_1"}
	},{
		"type":"message",
		"role":"user",
		"id":"msg_2",
		"author":"/root/worker",
		"recipient":"/root",
		"content":"regular user message with author",
		"internal_chat_message_metadata_passthrough":{"turn_id":"turn_2"}
	},{
		"type":"message",
		"role":"assistant",
		"content":"clean assistant message"
	}]}`)

	t.Run("compat mode strips author, recipient, and passthrough from all items even with custom user-agent", func(t *testing.T) {
		headers := http.Header{"User-Agent": []string{"curl/8.7.1"}}
		cfg := &config.Config{CodexOptimizeMultiAgentV2: true}
		got := RewriteCodexMultiAgentV2Input(context.Background(), headers, payload, cfg, true)

		// Item 0: agent_message -> message/user
		if messageType := gjson.GetBytes(got, "input.0.type").String(); messageType != "message" {
			t.Fatalf("input.0.type = %q, want message", messageType)
		}
		if role := gjson.GetBytes(got, "input.0.role").String(); role != "user" {
			t.Fatalf("input.0.role = %q, want user", role)
		}
		if author := gjson.GetBytes(got, "input.0.author"); author.Exists() {
			t.Fatalf("input.0.author was not stripped; got=%s", author.String())
		}
		if recipient := gjson.GetBytes(got, "input.0.recipient"); recipient.Exists() {
			t.Fatalf("input.0.recipient was not stripped; got=%s", recipient.String())
		}
		if passthrough := gjson.GetBytes(got, "input.0.internal_chat_message_metadata_passthrough"); passthrough.Exists() {
			t.Fatalf("input.0.internal_chat_message_metadata_passthrough was not stripped; got=%s", passthrough.String())
		}

		// Item 1: regular message with non-standard fields -> cleaned
		if messageType := gjson.GetBytes(got, "input.1.type").String(); messageType != "message" {
			t.Fatalf("input.1.type = %q, want message", messageType)
		}
		if role := gjson.GetBytes(got, "input.1.role").String(); role != "user" {
			t.Fatalf("input.1.role = %q, want user", role)
		}
		if author := gjson.GetBytes(got, "input.1.author"); author.Exists() {
			t.Fatalf("input.1.author was not stripped; got=%s", author.String())
		}
		if recipient := gjson.GetBytes(got, "input.1.recipient"); recipient.Exists() {
			t.Fatalf("input.1.recipient was not stripped; got=%s", recipient.String())
		}
		if passthrough := gjson.GetBytes(got, "input.1.internal_chat_message_metadata_passthrough"); passthrough.Exists() {
			t.Fatalf("input.1.internal_chat_message_metadata_passthrough was not stripped; got=%s", passthrough.String())
		}

		// Item 2: clean message remains intact
		if content := gjson.GetBytes(got, "input.2.content").String(); content != "clean assistant message" {
			t.Fatalf("input.2.content = %q", content)
		}
	})

	t.Run("non-compat mode preserves author and recipient on all items", func(t *testing.T) {
		headers := http.Header{"User-Agent": []string{"Codex Desktop/0.146.0-alpha.3"}}
		cfg := &config.Config{CodexOptimizeMultiAgentV2: true}
		got := RewriteCodexMultiAgentV2Input(context.Background(), headers, payload, cfg, false)

		if author := gjson.GetBytes(got, "input.0.author").String(); author != "/root" {
			t.Fatalf("input.0.author = %q, want /root", author)
		}
		if recipient := gjson.GetBytes(got, "input.0.recipient").String(); recipient != "/root/worker" {
			t.Fatalf("input.0.recipient = %q, want /root/worker", recipient)
		}
		if passthrough := gjson.GetBytes(got, "input.0.internal_chat_message_metadata_passthrough.turn_id").String(); passthrough != "turn_1" {
			t.Fatalf("input.0 passthrough = %q", passthrough)
		}
		if author := gjson.GetBytes(got, "input.1.author").String(); author != "/root/worker" {
			t.Fatalf("input.1.author = %q, want /root/worker", author)
		}
		if recipient := gjson.GetBytes(got, "input.1.recipient").String(); recipient != "/root" {
			t.Fatalf("input.1.recipient = %q, want /root", recipient)
		}
	})
}
