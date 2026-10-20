package executor

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/wsrelay"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy/usage"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestAIStudioTranslateRequestPrependsLeadingUser(t *testing.T) {
	executor := NewAIStudioExecutor(&config.Config{}, "aistudio", nil)

	tests := []struct {
		name   string
		stream bool
		action string
	}{
		{name: "normal", action: "generateContent"},
		{name: "stream", stream: true, action: "streamGenerateContent"},
		{name: "count tokens", action: "countTokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := cliproxyexecutor.Request{
				Model:   "gemini-3.7-flash",
				Payload: []byte(`{"contents":[{"role":"model","parts":[{"text":"prior output"}]},{"role":"user","parts":[{"text":"continue"}]}]}`),
			}
			if tt.action == "countTokens" {
				req.Metadata = map[string]any{"action": "countTokens"}
			}
			_, body, err := executor.translateRequest(req, cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatGemini}, tt.stream)
			if err != nil {
				t.Fatalf("translateRequest() error = %v", err)
			}
			contents := gjson.GetBytes(body.payload, "contents").Array()
			if len(contents) != 3 || contents[0].Get("role").String() != "user" || contents[1].Get("role").String() != "model" {
				t.Fatalf("leading contents malformed: %s", body.payload)
			}
			if text := contents[0].Get("parts.0.text"); !text.Exists() || text.String() != "" {
				t.Fatalf("synthetic leading user missing: %s", body.payload)
			}
			if body.action != tt.action {
				t.Fatalf("action = %q, want %q", body.action, tt.action)
			}
		})
	}
}

type malformedFunctionCallUsageCapture struct {
	authID  string
	records chan usage.Record
}

func (c *malformedFunctionCallUsageCapture) HandleUsage(_ context.Context, record usage.Record) {
	if c == nil || record.AuthID != c.authID {
		return
	}
	select {
	case c.records <- record:
	default:
	}
}

type malformedFunctionCallUsageNoop struct{}

func (malformedFunctionCallUsageNoop) HandleUsage(context.Context, usage.Record) {}

func assertMalformedFunctionCallFailedUsage(t *testing.T, records <-chan usage.Record) {
	t.Helper()
	var record usage.Record
	select {
	case record = <-records:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for usage record")
	}
	if !record.Failed {
		t.Fatalf("usage record Failed = false, want true; record=%+v", record)
	}
	if record.Detail.InputTokens != 120 || record.Detail.OutputTokens != 10 || record.Detail.ReasoningTokens != 313 || record.Detail.TotalTokens != 443 {
		t.Fatalf("usage detail = %+v, want Input=120 Output=10 Reasoning=313 Total=443", record.Detail)
	}
}

func TestAIStudioExecutorClaudeMalformedFunctionCallReturnsRetryableError(t *testing.T) {
	malformedBody := `{"candidates":[{"content":{"role":"model","parts":[]},"finishReason":"MALFORMED_FUNCTION_CALL"}],"usageMetadata":{"promptTokenCount":120,"candidatesTokenCount":10,"thoughtsTokenCount":313,"totalTokenCount":443},"modelVersion":"gemini-4-argon"}`
	authID := "aistudio-" + t.Name()
	connected := make(chan struct{})
	relay := wsrelay.NewManager(wsrelay.Options{
		ProviderFactory: func(*http.Request) (string, error) { return authID, nil },
		OnConnected:     func(string) { close(connected) },
	})
	server := httptest.NewServer(relay.Handler())
	defer server.Close()
	defer func() {
		if errStop := relay.Stop(context.Background()); errStop != nil {
			t.Error(errStop)
		}
	}()
	conn, _, errDial := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+relay.Path(), nil)
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer func() {
		_ = conn.Close()
	}()
	<-connected

	clientDone := make(chan error, 1)
	go func() {
		for i := range 3 {
			var msg wsrelay.Message
			if errReadJSON := conn.ReadJSON(&msg); errReadJSON != nil {
				clientDone <- errReadJSON
				return
			}
			var responses []wsrelay.Message
			switch i {
			case 0, 2:
				responses = []wsrelay.Message{{
					ID:   msg.ID,
					Type: wsrelay.MessageTypeHTTPResp,
					Payload: map[string]any{
						"status":  float64(http.StatusOK),
						"headers": map[string]any{"Content-Type": "application/json"},
						"body":    malformedBody,
					},
				}}
			case 1:
				responses = []wsrelay.Message{
					{ID: msg.ID, Type: wsrelay.MessageTypeStreamStart, Payload: map[string]any{"status": 200}},
					{ID: msg.ID, Type: wsrelay.MessageTypeStreamChunk, Payload: map[string]any{"data": "data: " + malformedBody + "\n\n"}},
					{ID: msg.ID, Type: wsrelay.MessageTypeStreamEnd},
				}
			}
			for _, response := range responses {
				if errWriteJSON := conn.WriteJSON(response); errWriteJSON != nil {
					clientDone <- errWriteJSON
					return
				}
			}
		}
		clientDone <- nil
	}()

	capture := &malformedFunctionCallUsageCapture{authID: authID, records: make(chan usage.Record, 3)}
	usage.RegisterNamedPlugin(t.Name(), capture)
	t.Cleanup(func() {
		usage.RegisterNamedPlugin(t.Name(), malformedFunctionCallUsageNoop{})
	})

	exec := NewAIStudioExecutor(&config.Config{}, "aistudio", relay)
	auth := &cliproxyauth.Auth{ID: authID, Provider: "aistudio"}
	req := cliproxyexecutor.Request{
		Model:   "gemini-4-argon",
		Payload: []byte(`{"model":"gemini-4-argon","max_tokens":1024,"messages":[{"role":"user","content":"hi"}]}`),
	}
	opts := cliproxyexecutor.Options{
		SourceFormat:   sdktranslator.FormatClaude,
		ResponseFormat: sdktranslator.FormatClaude,
	}

	_, errExec := exec.Execute(context.Background(), auth, req, opts)
	if errExec == nil {
		t.Fatal("Execute() error = nil, want retryable 502 error for MALFORMED_FUNCTION_CALL")
	}
	status, okStatus := errExec.(interface{ StatusCode() int })
	if !okStatus || status.StatusCode() != http.StatusBadGateway {
		t.Fatalf("Execute() error = %T %v, want status 502", errExec, errExec)
	}
	if got := gjson.Get(errExec.Error(), "error.type").String(); got != "api_error" {
		t.Fatalf("Execute() error.type = %q, want api_error. Error: %s", got, errExec.Error())
	}
	assertMalformedFunctionCallFailedUsage(t, capture.records)

	for _, mode := range []string{"stream_chunk", "http_resp"} {
		streamRes, errStream := exec.ExecuteStream(context.Background(), auth, req, opts)
		if errStream != nil {
			t.Fatalf("%s ExecuteStream() error = %v", mode, errStream)
		}
		var combined bytes.Buffer
		var streamErr error
		for chunk := range streamRes.Chunks {
			if chunk.Err != nil {
				streamErr = chunk.Err
				continue
			}
			combined.Write(chunk.Payload)
		}
		if streamErr == nil {
			t.Fatalf("%s ExecuteStream() chunk.Err = nil, want 502 error propagated", mode)
		}
		streamText := combined.String()
		if strings.Contains(streamText, `"stop_reason":"refusal"`) {
			t.Fatalf("%s stream unexpectedly emitted stop_reason refusal:\n%s", mode, streamText)
		}
		if !strings.Contains(streamText, "event: error\n") || (!strings.Contains(streamText, `"type": "api_error"`) && !strings.Contains(streamText, `"type":"api_error"`)) {
			t.Fatalf("%s stream missing retryable api_error event:\n%s", mode, streamText)
		}
		assertMalformedFunctionCallFailedUsage(t, capture.records)
	}

	if errClient := <-clientDone; errClient != nil {
		t.Fatal(errClient)
	}
}

func TestAIStudioExecutorClaudeMalformedFunctionCallCancelAfterErrorRecordsConductorFailure(t *testing.T) {
	malformedBody := `{"candidates":[{"content":{"role":"model","parts":[]},"finishReason":"MALFORMED_FUNCTION_CALL"}],"usageMetadata":{"promptTokenCount":120,"candidatesTokenCount":10,"thoughtsTokenCount":313,"totalTokenCount":443},"modelVersion":"gemini-4-argon"}`
	for _, mode := range []string{"stream_end", "http_resp"} {
		t.Run(mode, func(t *testing.T) {
			authID := "aistudio-cancel-" + mode
			connected := make(chan struct{})
			canceled := make(chan struct{})
			relay := wsrelay.NewManager(wsrelay.Options{
				ProviderFactory: func(*http.Request) (string, error) { return authID, nil },
				OnConnected:     func(string) { close(connected) },
			})
			server := httptest.NewServer(relay.Handler())
			defer server.Close()
			defer func() {
				if errStop := relay.Stop(context.Background()); errStop != nil {
					t.Error(errStop)
				}
			}()
			conn, _, errDial := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+relay.Path(), nil)
			if errDial != nil {
				t.Fatal(errDial)
			}
			defer func() {
				_ = conn.Close()
			}()
			<-connected

			clientDone := make(chan error, 1)
			go func() {
				var msg wsrelay.Message
				if errReadJSON := conn.ReadJSON(&msg); errReadJSON != nil {
					clientDone <- errReadJSON
					return
				}
				if mode == "http_resp" {
					if errWriteJSON := conn.WriteJSON(wsrelay.Message{
						ID:   msg.ID,
						Type: wsrelay.MessageTypeHTTPResp,
						Payload: map[string]any{
							"status":  float64(http.StatusOK),
							"headers": map[string]any{"Content-Type": "application/json"},
							"body":    malformedBody,
						},
					}); errWriteJSON != nil {
						clientDone <- errWriteJSON
						return
					}
					clientDone <- nil
					return
				}
				for _, response := range []wsrelay.Message{
					{ID: msg.ID, Type: wsrelay.MessageTypeStreamStart, Payload: map[string]any{"status": 200}},
					{ID: msg.ID, Type: wsrelay.MessageTypeStreamChunk, Payload: map[string]any{"data": "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"thought\":true,\"text\":\"planning\"}]}}],\"modelVersion\":\"gemini-4-argon\"}\n\n"}},
					{ID: msg.ID, Type: wsrelay.MessageTypeStreamChunk, Payload: map[string]any{"data": "data: " + malformedBody + "\n\n"}},
				} {
					if errWriteJSON := conn.WriteJSON(response); errWriteJSON != nil {
						clientDone <- errWriteJSON
						return
					}
				}
				<-canceled
				if errWriteJSON := conn.WriteJSON(wsrelay.Message{ID: msg.ID, Type: wsrelay.MessageTypeStreamEnd}); errWriteJSON != nil {
					clientDone <- errWriteJSON
					return
				}
				clientDone <- nil
			}()

			capture := &malformedFunctionCallUsageCapture{authID: authID, records: make(chan usage.Record, 1)}
			usage.RegisterNamedPlugin(t.Name(), capture)
			t.Cleanup(func() {
				usage.RegisterNamedPlugin(t.Name(), malformedFunctionCallUsageNoop{})
			})

			exec := NewAIStudioExecutor(&config.Config{}, "aistudio", relay)
			auth := &cliproxyauth.Auth{ID: authID, Provider: "aistudio", Status: cliproxyauth.StatusActive}
			manager := cliproxyauth.NewManager(nil, nil, nil)
			manager.RegisterExecutor(exec)
			if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
				t.Fatalf("Register() error = %v", errRegister)
			}
			registry.GetGlobalRegistry().RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{ID: "gemini-4-argon"}})
			t.Cleanup(func() {
				registry.GetGlobalRegistry().UnregisterClient(auth.ID)
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := cliproxyexecutor.Request{
				Model:   "gemini-4-argon",
				Payload: []byte(`{"model":"gemini-4-argon","max_tokens":1024,"messages":[{"role":"user","content":"hi"}]}`),
			}
			opts := cliproxyexecutor.Options{
				SourceFormat:   sdktranslator.FormatClaude,
				ResponseFormat: sdktranslator.FormatClaude,
				Stream:         true,
			}

			streamRes, errStream := manager.ExecuteStream(ctx, []string{"aistudio"}, req, opts)
			if errStream != nil {
				t.Fatalf("manager.ExecuteStream() error = %v", errStream)
			}
			var cancelOnce sync.Once
			for chunk := range streamRes.Chunks {
				if strings.Contains(string(chunk.Payload), "event: error") {
					cancelOnce.Do(func() {
						cancel()
						close(canceled)
					})
				}
			}
			assertMalformedFunctionCallFailedUsage(t, capture.records)
			updated, okAuth := manager.GetByID(auth.ID)
			if !okAuth || updated.ModelStates["gemini-4-argon"] == nil || updated.ModelStates["gemini-4-argon"].LastError == nil {
				t.Fatalf("expected conductor to record failure on model state after client cancel on error event, got %+v", updated)
			}
			if updated.ModelStates["gemini-4-argon"].LastError.HTTPStatus != http.StatusBadGateway {
				t.Fatalf("LastError.HTTPStatus = %d, want 502; err=%+v", updated.ModelStates["gemini-4-argon"].LastError.HTTPStatus, updated.ModelStates["gemini-4-argon"].LastError)
			}
			if errClient := <-clientDone; errClient != nil {
				t.Fatal(errClient)
			}
		})
	}
}

func TestAIStudioExecutorClaudeMalformedFunctionCallTransportErrorAfterStreamErrorPrefersTranslatedError(t *testing.T) {
	malformedBody := `{"candidates":[{"content":{"role":"model","parts":[]},"finishReason":"MALFORMED_FUNCTION_CALL"}],"usageMetadata":{"promptTokenCount":120,"candidatesTokenCount":10,"thoughtsTokenCount":313,"totalTokenCount":443},"modelVersion":"gemini-4-argon"}`
	authID := "aistudio-both-errors"
	connected := make(chan struct{})
	relay := wsrelay.NewManager(wsrelay.Options{
		ProviderFactory: func(*http.Request) (string, error) { return authID, nil },
		OnConnected:     func(string) { close(connected) },
	})
	server := httptest.NewServer(relay.Handler())
	defer server.Close()
	defer func() {
		if errStop := relay.Stop(context.Background()); errStop != nil {
			t.Error(errStop)
		}
	}()
	conn, _, errDial := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+relay.Path(), nil)
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer func() {
		_ = conn.Close()
	}()
	<-connected

	clientDone := make(chan error, 1)
	go func() {
		var msg wsrelay.Message
		if errReadJSON := conn.ReadJSON(&msg); errReadJSON != nil {
			clientDone <- errReadJSON
			return
		}
		// Send start and chunk with MALFORMED_FUNCTION_CALL
		for _, response := range []wsrelay.Message{
			{ID: msg.ID, Type: wsrelay.MessageTypeStreamStart, Payload: map[string]any{"status": 200}},
			{ID: msg.ID, Type: wsrelay.MessageTypeStreamChunk, Payload: map[string]any{"data": "data: " + malformedBody + "\n\n"}},
		} {
			if errWriteJSON := conn.WriteJSON(response); errWriteJSON != nil {
				clientDone <- errWriteJSON
				return
			}
		}
		// Close socket directly so event.Err is triggered on wsStream
		_ = conn.Close()
		clientDone <- nil
	}()

	capture := &malformedFunctionCallUsageCapture{authID: authID, records: make(chan usage.Record, 1)}
	usage.RegisterNamedPlugin(t.Name(), capture)
	t.Cleanup(func() {
		usage.RegisterNamedPlugin(t.Name(), malformedFunctionCallUsageNoop{})
	})

	exec := NewAIStudioExecutor(&config.Config{}, "aistudio", relay)
	auth := &cliproxyauth.Auth{ID: authID, Provider: "aistudio"}
	req := cliproxyexecutor.Request{
		Model:   "gemini-4-argon",
		Payload: []byte(`{"model":"gemini-4-argon","max_tokens":1024,"messages":[{"role":"user","content":"hi"}]}`),
	}
	opts := cliproxyexecutor.Options{
		SourceFormat:   sdktranslator.FormatClaude,
		ResponseFormat: sdktranslator.FormatClaude,
	}

	streamRes, errStream := exec.ExecuteStream(context.Background(), auth, req, opts)
	if errStream != nil {
		t.Fatalf("ExecuteStream() error = %v", errStream)
	}
	var errCount int
	var streamErr error
	for chunk := range streamRes.Chunks {
		if chunk.Err != nil {
			errCount++
			streamErr = chunk.Err
		}
	}
	if errCount != 1 {
		t.Fatalf("expected exactly 1 error chunk, got %d", errCount)
	}
	status, okStatus := streamErr.(interface{ StatusCode() int })
	if !okStatus || status.StatusCode() != http.StatusBadGateway {
		t.Fatalf("expected 502 status error for translated MALFORMED_FUNCTION_CALL, got: %T %v", streamErr, streamErr)
	}
	if !strings.Contains(streamErr.Error(), "MALFORMED_FUNCTION_CALL") {
		t.Fatalf("expected MALFORMED_FUNCTION_CALL error message, got: %v", streamErr)
	}
	assertMalformedFunctionCallFailedUsage(t, capture.records)

	if errClient := <-clientDone; errClient != nil {
		t.Fatal(errClient)
	}
}
