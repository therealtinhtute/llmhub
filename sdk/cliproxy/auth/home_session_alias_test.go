package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type sessionAliasCaptureDispatcher struct {
	mu       sync.Mutex
	sessions []string
}

type homeSessionAliasExecutor struct {
	schedulerTestExecutor
}

func (homeSessionAliasExecutor) Identifier() string { return "home-session-alias" }

func (*sessionAliasCaptureDispatcher) HeartbeatOK() bool { return true }

func (d *sessionAliasCaptureDispatcher) RPopAuth(_ context.Context, _ string, sessionID string, _ http.Header, _ int) ([]byte, error) {
	d.mu.Lock()
	d.sessions = append(d.sessions, sessionID)
	d.mu.Unlock()
	return json.Marshal(homeAuthDispatchResponse{Auth: Auth{
		ID:       "home-session-alias-auth",
		Provider: "home-session-alias",
		Status:   StatusActive,
	}})
}

func (*sessionAliasCaptureDispatcher) AbortAmbiguousDispatch() {}

func (d *sessionAliasCaptureDispatcher) sessionIDs() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.sessions...)
}

func TestHomeDispatchCanonicalizesPromptCacheAndConversationAliases(t *testing.T) {
	tests := []struct {
		name     string
		payloads []string
		want     string
	}{
		{
			name: "conversation then combined then prompt cache",
			payloads: []string{
				`{"conversation":{"id":"conversation-session"}}`,
				`{"conversation":{"id":"conversation-session"},"prompt_cache_key":"shared-cache-bucket"}`,
				`{"prompt_cache_key":"shared-cache-bucket"}`,
			},
			want: "conv:conversation-session",
		},
		{
			name: "prompt cache then combined then conversation",
			payloads: []string{
				`{"prompt_cache_key":"shared-cache-bucket"}`,
				`{"conversation":{"id":"conversation-session"},"prompt_cache_key":"shared-cache-bucket"}`,
				`{"conversation":{"id":"conversation-session"}}`,
			},
			want: "pck:shared-cache-bucket",
		},
		{
			name: "combined request establishes prompt cache primary",
			payloads: []string{
				`{"conversation":{"id":"conversation-session"},"prompt_cache_key":"shared-cache-bucket"}`,
				`{"conversation":{"id":"conversation-session"}}`,
				`{"prompt_cache_key":"shared-cache-bucket"}`,
			},
			want: "pck:shared-cache-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &sessionAliasCaptureDispatcher{}
			manager := newHomeSelectionTestManager(t, dispatcher)
			manager.RegisterExecutor(homeSessionAliasExecutor{})

			for _, payload := range tt.payloads {
				selection, errSelection := manager.pickHomeDispatchSelection(context.Background(), "gpt-test", cliproxyexecutor.Options{
					OriginalRequest: []byte(payload),
				})
				if errSelection != nil {
					t.Fatalf("pickHomeDispatchSelection() error = %v", errSelection)
				}
				selection.End("test_complete")
			}

			got := dispatcher.sessionIDs()
			if len(got) != len(tt.payloads) {
				t.Fatalf("Home session IDs = %#v, want %d entries", got, len(tt.payloads))
			}
			for index, sessionID := range got {
				if sessionID != tt.want {
					t.Fatalf("Home session ID[%d] = %q, want %q; all=%#v", index, sessionID, tt.want, got)
				}
			}
		})
	}
}

func TestHomeSessionAliasCachePrimaryAccessRefreshesWholeAliasGroup(t *testing.T) {
	var cache homeSessionAliasCache
	now := time.Now()
	const primary = "pck:shared-cache-bucket"
	const fallback = "conv:conversation-session"

	if got := cache.canonical(primary, fallback, time.Minute, now); got != primary {
		t.Fatalf("initial canonical = %q, want %q", got, primary)
	}
	cache.mu.Lock()
	fallbackEntry := cache.entries[fallback]
	fallbackEntry.expiresAt = now.Add(-time.Second)
	cache.entries[fallback] = fallbackEntry
	cache.mu.Unlock()

	if got := cache.canonical(primary, "", time.Minute, now.Add(10*time.Second)); got != primary {
		t.Fatalf("primary-only canonical = %q, want %q", got, primary)
	}
	if got := cache.canonical(fallback, "", time.Minute, now.Add(20*time.Second)); got != primary {
		t.Fatalf("fallback canonical after active primary traffic = %q, want %q", got, primary)
	}
}

func TestHomeSessionAliasCacheClearsWhenConfiguredTTLChanges(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{
		Home:    internalconfig.HomeConfig{Enabled: true},
		Routing: internalconfig.RoutingConfig{SessionAffinityTTL: "1h"},
	})
	combined := cliproxyexecutor.Options{OriginalRequest: []byte(
		`{"conversation":{"id":"ttl-conversation"},"prompt_cache_key":"ttl-prompt"}`,
	)}
	conversationOnly := cliproxyexecutor.Options{OriginalRequest: []byte(
		`{"conversation":{"id":"ttl-conversation"}}`,
	)}
	if got := manager.homeDispatchSessionID(combined); got != "pck:ttl-prompt" {
		t.Fatalf("combined canonical = %q, want pck:ttl-prompt", got)
	}
	if got := manager.homeDispatchSessionID(conversationOnly); got != "pck:ttl-prompt" {
		t.Fatalf("conversation canonical before reload = %q, want existing prompt canonical", got)
	}

	manager.SetConfig(&internalconfig.Config{
		Home:    internalconfig.HomeConfig{Enabled: true},
		Routing: internalconfig.RoutingConfig{SessionAffinityTTL: "1m"},
	})
	if got := manager.homeDispatchSessionID(conversationOnly); got != "conv:ttl-conversation" {
		t.Fatalf("conversation canonical after TTL change = %q, want cleared alias cache", got)
	}
}
