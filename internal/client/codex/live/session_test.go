package live

import (
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestStoreClaimReleaseCompleteAndClose(t *testing.T) {
	store := NewStore()
	closed := atomic.Int32{}
	session := StoreSessionWithCloser(store, "call-123", func() error {
		closed.Add(1)
		return nil
	})

	if got, ok := store.Peek("call-123"); !ok || got.CallID != session.CallID || got.AuthID != "auth-1" {
		t.Fatalf("Peek() = %#v, %t", got, ok)
	}
	claimed, claim := store.Claim("call-123")
	if claim != ClaimAcquired || claimed.CallID != "call-123" {
		t.Fatalf("Claim() = %#v, %v", claimed, claim)
	}
	if _, claim = store.Claim("call-123"); claim != ClaimBusy {
		t.Fatalf("second Claim() = %v, want busy", claim)
	}
	store.Release(claimed)
	claimed, claim = store.Claim("call-123")
	if claim != ClaimAcquired {
		t.Fatalf("Claim() after Release = %v, want acquired", claim)
	}
	store.Complete(claimed)
	if _, ok := store.Peek("call-123"); ok {
		t.Fatal("completed session still stored")
	}
	if got := closed.Load(); got != 1 {
		t.Fatalf("closer calls = %d, want 1", got)
	}
	store.CloseAll()
	if got := closed.Load(); got != 1 {
		t.Fatalf("CloseAll() called completed closer again: %d", got)
	}
}

func TestStoreExpiresUnclaimedSession(t *testing.T) {
	store := NewStore()
	store.SetLifetime(10 * time.Millisecond)
	closed := atomic.Int32{}
	StoreSessionWithCloser(store, "call-expire", func() error {
		closed.Add(1)
		return nil
	})

	deadline := time.After(time.Second)
	for {
		if _, ok := store.Peek("call-expire"); !ok {
			break
		}
		select {
		case <-deadline:
			t.Fatal("session did not expire")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if got := closed.Load(); got != 1 {
		t.Fatalf("closer calls = %d, want 1", got)
	}
}

func TestStoreRejectsInvalidCallID(t *testing.T) {
	store := NewStore()
	closed := atomic.Int32{}
	stored := StoreSessionWithCloser(store, "../bad", func() error {
		closed.Add(1)
		return nil
	})
	if stored.CallID != "" {
		t.Fatalf("stored invalid session = %#v", stored)
	}
	if got := closed.Load(); got != 1 {
		t.Fatalf("invalid closer calls = %d, want 1", got)
	}
	if _, claim := store.Claim("../bad"); claim != ClaimMissing {
		t.Fatalf("invalid Claim() = %v, want missing", claim)
	}
}

func TestSessionResourcesCloseAddedAfterClose(t *testing.T) {
	resources := &SessionResources{}
	closed := atomic.Int32{}
	resources.Close()
	resources.Add(func() error {
		closed.Add(1)
		return nil
	})
	resources.Close()
	if got := closed.Load(); got != 1 {
		t.Fatalf("closer calls = %d, want 1", got)
	}
}

func TestSidebandTargetsAndURLs(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		param string
		query string
		style SidebandStyle
		url   string
	}{
		{name: "frameless", path: "/live/call_123", param: "call_123", style: SidebandFrameless, url: "wss://api.example/live/call_123"},
		{name: "realtime calls", path: "/realtime/calls/call-123", param: "call-123", style: SidebandRealtimeCalls, url: "wss://api.example/realtime/calls/call-123"},
		{name: "query", path: "/realtime", query: "call_with_escape", style: SidebandRealtimeQuery, url: "wss://api.example/realtime?intent=quicksilver&call_id=call_with_escape"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := url.Values{}
			if tt.query != "" {
				query.Set("call_id", tt.query)
			}
			params := map[string]string{}
			if tt.param != "" {
				params["call_id"] = tt.param
			}
			style, callID, ok := SidebandTarget(tt.path, params, query)
			if !ok || style != tt.style {
				t.Fatalf("SidebandTarget() = %v, %q, %t", style, callID, ok)
			}
			if got := BuildSidebandURL("wss://api.example/", style, callID); got != tt.url {
				t.Fatalf("BuildSidebandURL() = %q, want %q", got, tt.url)
			}
		})
	}
}

func StoreSessionWithCloser(store *Store, callID string, closer func() error) Session {
	resources := &SessionResources{}
	resources.Add(closer)
	return store.Put(callID, Session{AuthID: "auth-1", Model: DefaultLiveModel, Resources: resources})
}
