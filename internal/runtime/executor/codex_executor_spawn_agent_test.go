package executor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestCodexExecutorsMultiAgentV2UsesSelectedHomeModel(t *testing.T) {
	capturedPayload := make(chan []byte, 1)
	upgrader := websocket.Upgrader{}
	completed := []byte(`{"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[]}}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if websocket.IsWebSocketUpgrade(request) {
			conn, errUpgrade := upgrader.Upgrade(w, request, nil)
			if errUpgrade != nil {
				t.Errorf("upgrade websocket: %v", errUpgrade)
				return
			}
			defer func() {
				if errClose := conn.Close(); errClose != nil {
					t.Errorf("close websocket: %v", errClose)
				}
			}()
			_, payload, errRead := conn.ReadMessage()
			if errRead != nil {
				t.Errorf("read websocket request: %v", errRead)
				return
			}
			capturedPayload <- payload
			if errWrite := conn.WriteMessage(websocket.TextMessage, completed); errWrite != nil {
				t.Errorf("write websocket response: %v", errWrite)
			}
			return
		}
		payload, errRead := io.ReadAll(request.Body)
		if errRead != nil {
			t.Errorf("read request: %v", errRead)
			http.Error(w, errRead.Error(), http.StatusInternalServerError)
			return
		}
		capturedPayload <- payload
		if request.URL.Path == "/responses/compact" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"resp_1","object":"response.compaction","output":[]}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: %s\n\n", completed)
	}))
	t.Cleanup(server.Close)

	for _, tc := range []struct {
		name, model string
		isCompat    bool
	}{
		{name: "native alias", model: "gpt-5.4"},
		{name: "compat alias", model: "gpt-5.4", isCompat: true},
		{name: "native suffix", model: "gpt-5.4(high)"},
		{name: "compat suffix", model: "gpt-5.4(high)", isCompat: true},
	} {
		// WebSocket modes exercise upstream's per-session multi-agent restore
		// machinery (sess.isMultiAgentV2Optimized), which this fork does not carry.
		for _, mode := range []string{"execute", "stream", "compact"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				cfg := &config.Config{Home: config.HomeConfig{Enabled: true}, CodexOptimizeMultiAgentV2: true}
				var executor cliproxyauth.ProviderExecutor = NewCodexExecutor(cfg)
				if strings.HasPrefix(mode, "websocket") {
					executor = NewCodexWebsocketsExecutor(cfg)
				}
				auth := &cliproxyauth.Auth{
					ID: t.Name(), Provider: "codex", Prefix: "tenant",
					Attributes: map[string]string{
						"base_url": server.URL, "api_key": "test", "auth_kind": cliproxyauth.AuthKindAPIKey,
						"home_upstream_model": tc.model,
					},
					Metadata: map[string]any{"credential_options": map[string]any{"models": []config.CodexModel{
						{Name: "gpt-5.4", Alias: "other", IsCompat: !tc.isCompat},
						{Name: tc.model, Alias: "chosen", IsCompat: tc.isCompat},
					}}},
				}
				req := cliproxyexecutor.Request{
					Model: tc.model, Payload: codexSpawnAgentTestPayload(),
					Metadata: map[string]any{"cliproxy.resolved_home_model_info": &registry.ModelInfo{
						ID: "gpt-5.4", IsCompat: tc.isCompat, Thinking: &registry.ThinkingSupport{Levels: []string{"high"}},
					}},
				}
				opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-response")}
				ctx := codexSpawnAgentTestContext()
				if strings.HasSuffix(mode, "stream") {
					result, errStream := executor.ExecuteStream(ctx, auth, req, opts)
					if errStream != nil {
						t.Fatal(errStream)
					}
					for chunk := range result.Chunks {
						if chunk.Err != nil {
							t.Fatal(chunk.Err)
						}
					}
				} else {
					if mode == "compact" {
						opts.Alt = "responses/compact"
					}
					if _, errExecute := executor.Execute(ctx, auth, req, opts); errExecute != nil {
						t.Fatal(errExecute)
					}
				}
				upstreamPayload := <-capturedPayload
				// Local NormalizeResponsesToolsForCodex promotes additional_tools into
				// top-level tools, so the message lands at input.0 here; the compat
				// rewrite renames the type to "message".
				message := gjson.GetBytes(upstreamPayload, `input.#(type=="agent_message")`)
				if !message.Exists() {
					message = gjson.GetBytes(upstreamPayload, `input.#(type=="message")`)
				}
				if tc.isCompat {
					if message.Get("type").String() != "message" || message.Get("role").String() != "user" {
						t.Fatalf("selected compat model must receive a user message: %s", message.Raw)
					}
					if message.Get("author").Exists() || message.Get("recipient").Exists() || message.Get("internal_chat_message_metadata_passthrough").Exists() {
						t.Fatalf("compat message retained internal metadata: %s", message.Raw)
					}
				} else {
					assertCodexSpawnAgentRequestMessage(t, upstreamPayload, true)
				}
			})
		}
	}
}

func codexSpawnAgentTestPayload() []byte {
	return []byte(`{
		"model":"gpt-5.4",
		"input":[{
			"type":"additional_tools",
			"role":"developer",
			"tools":[{
				"type":"namespace",
				"name":"collaboration",
				"tools":[{
					"type":"function",
					"name":"spawn_agent",
					"description":"Available model overrides (optional; inherited parent model is preferred):\n- old-model\nSpawns an agent.",
					"parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}
				}]
			}]
		},{
			"type":"agent_message",
			"id":"amsg_1",
			"author":"/root",
			"recipient":"/root/worker",
			"content":[
				{"type":"input_text","text":"Payload:\n"},
				{"type":"encrypted_content","encrypted_content":"delegated task"}
			],
			"internal_chat_message_metadata_passthrough":{"turn_id":"turn_1"}
		}]
	}`)
}

func codexSpawnAgentTestContext() context.Context {
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	request.Header.Set("User-Agent", "codex-tui/0.154.0")
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = request
	return context.WithValue(context.Background(), "gin", ginCtx)
}

func assertCodexSpawnAgentRequestMessage(t *testing.T, payload []byte, enabled bool) {
	t.Helper()
	// Local NormalizeResponsesToolsForCodex promotes additional_tools into
	// top-level tools, so the agent message lands at input.0 here.
	message := gjson.GetBytes(payload, `input.#(type=="agent_message")`)
	if !message.Exists() {
		message = gjson.GetBytes(payload, `input.#(id=="amsg_1")`)
	}
	if message.Get("type").String() != "agent_message" || message.Get("role").Exists() {
		t.Fatalf("Codex executor changed outer agent message: %s", payload)
	}
	if message.Get("author").String() != "/root" || message.Get("recipient").String() != "/root/worker" || message.Get("internal_chat_message_metadata_passthrough.turn_id").String() != "turn_1" {
		t.Fatalf("Codex executor changed agent message metadata: %s", payload)
	}
	if enabled {
		if message.Get("content.1.type").String() != "input_text" || message.Get("content.1.text").String() != "delegated task" {
			t.Fatalf("Codex executor did not normalize agent message content: %s", payload)
		}
		if message.Get("content.1.encrypted_content").Exists() {
			t.Fatalf("Codex executor preserved encrypted_content: %s", payload)
		}
		return
	}
	if message.Get("content.1.type").String() != "encrypted_content" || message.Get("content.1.encrypted_content").String() != "delegated task" {
		t.Fatalf("disabled optimization changed agent message content: %s", payload)
	}
}
