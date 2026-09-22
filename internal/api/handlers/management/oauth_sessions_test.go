package management

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

// Ported from upstream CLIProxyAPI
// internal/api/handlers/management/oauth_sessions_test.go cancel slice
// (6e819ab62257, end-state v7.3.4). Local symbols under test:
// oauthSessionStore.Cancel/Complete tombstone, CancelOAuthSession,
// guardOAuthSessionPendingForSave, Handler.CancelAuthSession,
// Handler.GetAuthStatus.

func TestOAuthSessionStoreCancelRemovesPendingSession(t *testing.T) {
	store := newOAuthSessionStore(time.Minute)
	store.Register("pending-state", "codex")

	if !store.Cancel("pending-state") {
		t.Fatal("Cancel() = false for pending session, want true")
	}
	if store.IsPending("pending-state", "codex") {
		t.Fatal("cancelled session remained pending")
	}
}

func TestOAuthSessionStoreCancelRejectsUnknownSession(t *testing.T) {
	store := newOAuthSessionStore(time.Minute)
	if store.Cancel("unknown-state") {
		t.Fatal("Cancel() = true for unknown session, want false")
	}
}

func TestOAuthSessionStoreCancelRejectsErroredSession(t *testing.T) {
	store := newOAuthSessionStore(time.Minute)
	store.Register("errored-state", "codex")
	store.SetError("errored-state", "exchange failed")
	if store.Cancel("errored-state") {
		t.Fatal("Cancel() = true for errored session, want false")
	}
}

func TestOAuthSessionStoreCancelRejectsEmptyState(t *testing.T) {
	store := newOAuthSessionStore(time.Minute)
	store.Register("real-state", "codex")
	if store.Cancel("  ") {
		t.Fatal("Cancel() = true for empty state, want false")
	}
	if !store.IsPending("real-state", "codex") {
		t.Fatal("empty-state cancel removed an unrelated session")
	}
}

// TestOAuthSessionStoreCompleteTombstone covers the upstream completed-session
// tombstone (7115e7e00c4d + d1ef06cb5e34): Complete retains the session for
// completedTTL, hides it from IsPending/GetOAuthSession/Cancel, and rejects
// further callback submission.
func TestOAuthSessionStoreCompleteTombstone(t *testing.T) {
	store := newOAuthSessionStore(time.Minute)
	store.Register("done-state", "codex")
	store.SetCallback("done-state", "codex", oauthCallbackFilePayload{Code: "code", State: "done-state"})
	store.Complete("done-state")

	if store.IsPending("done-state", "codex") {
		t.Fatal("completed session still pending")
	}
	if store.Cancel("done-state") {
		t.Fatal("Cancel() = true for completed tombstone, want false")
	}
	if err := store.SetCallback("done-state", "codex", oauthCallbackFilePayload{Code: "x"}); !errors.Is(err, errOAuthSessionNotPending) {
		t.Fatalf("SetCallback on tombstone = %v, want errOAuthSessionNotPending", err)
	}
	if _, _, err := store.TakeCallback("done-state", "codex"); !errors.Is(err, errOAuthSessionNotPending) {
		t.Fatalf("TakeCallback on tombstone = %v, want errOAuthSessionNotPending", err)
	}
	session, ok := store.Get("done-state")
	if !ok || !session.Completed {
		t.Fatal("tombstone missing after Complete")
	}
	// Idempotent completion: a second Complete is a no-op.
	store.Complete("done-state")
	if session, ok = store.Get("done-state"); !ok || !session.Completed {
		t.Fatal("tombstone lost after second Complete")
	}
}

// TestCompleteOAuthSessionHidesFromLegacyGetter mirrors upstream d1ef06cb5e34:
// GetOAuthSession reports completed tombstones as missing while
// GetOAuthSessionDetails still observes them.
func TestCompleteOAuthSessionHidesFromLegacyGetter(t *testing.T) {
	state := "legacy-getter-state"
	RegisterOAuthSession(state, "anthropic")

	CompleteOAuthSession(state)
	if _, _, ok := GetOAuthSession(state); ok {
		t.Fatal("GetOAuthSession exposed a completed tombstone")
	}
	_, _, completed, ok := GetOAuthSessionDetails(state)
	if !ok || !completed {
		t.Fatal("GetOAuthSessionDetails did not report completed tombstone")
	}
	if err := guardOAuthSessionPendingForSave(state, "anthropic"); !errors.Is(err, errOAuthSessionNotPending) {
		t.Fatalf("guard on completed session = %v, want errOAuthSessionNotPending", err)
	}
}

// TestCancelOAuthSessionCallbackRejectAfterCancel mirrors upstream
// TestCancelOAuthSessionAndCallbackRejectAfterCancel: a cancelled session can
// no longer accept a callback submission.
func TestCancelOAuthSessionCallbackRejectAfterCancel(t *testing.T) {
	state := "cancel-callback-state"
	RegisterOAuthSession(state, "anthropic")

	if !CancelOAuthSession(state) {
		t.Fatal("CancelOAuthSession() = false, want true")
	}
	if err := SubmitOAuthCallbackForPendingSession("anthropic", state, "code", ""); !errors.Is(err, errOAuthSessionNotPending) {
		t.Fatalf("callback submit after cancel = %v, want errOAuthSessionNotPending", err)
	}
}

func TestCancelOAuthSessionExportedRoundTrip(t *testing.T) {
	state := "cancel-export-state"
	RegisterOAuthSession(state, "anthropic")
	t.Cleanup(func() { CompleteOAuthSession(state) })

	if !IsOAuthSessionPending(state, "anthropic") {
		t.Fatal("session not pending after register")
	}
	if !CancelOAuthSession(state) {
		t.Fatal("CancelOAuthSession() = false, want true")
	}
	if IsOAuthSessionPending(state, "anthropic") {
		t.Fatal("session still pending after cancel")
	}
	if CancelOAuthSession(state) {
		t.Fatal("CancelOAuthSession() = true on already-cancelled session, want false")
	}
}

func TestGuardOAuthSessionPendingForSave(t *testing.T) {
	// Provider coverage mirrors upstream oauth_sessions_test.go (e475807a96c9
	// added "meta" to this list at v7.3.4); the local store is the package
	// global rather than a replaced fixture.
	providers := []string{"anthropic", "codex", "antigravity", "xai", "kimi", "kimi-ai", "kimi.ai", "meta"}
	for _, provider := range providers {
		state := provider + "-save-guard"
		RegisterOAuthSession(state, provider)

		if err := guardOAuthSessionPendingForSave(state, provider); err != nil {
			t.Fatalf("%s: guard returned %v for pending session, want nil", provider, err)
		}
		if !CancelOAuthSession(state) {
			t.Fatalf("%s: CancelOAuthSession() = false, want true", provider)
		}
		if err := guardOAuthSessionPendingForSave(state, provider); !errors.Is(err, errOAuthSessionNotPending) {
			t.Fatalf("%s: guard returned %v after cancel, want errOAuthSessionNotPending", provider, err)
		}
	}

	// Local extra: a mismatched provider must also refuse the save.
	state := "guard-state"
	RegisterOAuthSession(state, "codex")
	t.Cleanup(func() { CompleteOAuthSession(state) })
	if err := guardOAuthSessionPendingForSave(state, "anthropic"); !errors.Is(err, errOAuthSessionNotPending) {
		t.Fatalf("guard returned %v for mismatched provider, want errOAuthSessionNotPending", err)
	}
	CancelOAuthSession(state)
	if err := guardOAuthSessionPendingForSave(state, "codex"); !errors.Is(err, errOAuthSessionNotPending) {
		t.Fatalf("guard returned %v after cancel, want errOAuthSessionNotPending", err)
	}
}

func TestCancelAuthSessionEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	router := gin.New()
	router.DELETE("/oauth-session", h.CancelAuthSession)

	state := "endpoint-cancel-state"
	RegisterOAuthSession(state, "codex")
	t.Cleanup(func() { CompleteOAuthSession(state) })

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/oauth-session?state="+state, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !containsJSON(rec, `"cancelled":true`) {
		t.Fatalf("pending cancel response = %s, want cancelled:true", rec.Body.String())
	}
	if IsOAuthSessionPending(state, "codex") {
		t.Fatal("session still pending after endpoint cancel")
	}

	// Already-cancelled session reports cancelled:false.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/oauth-session?state="+state, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !containsJSON(rec, `"cancelled":false`) {
		t.Fatalf("already-cancelled response = %s, want cancelled:false", rec.Body.String())
	}

	// Status after cancel must not report success (upstream
	// TestCancelAuthSessionHandler).
	statusRouter := gin.New()
	statusRouter.GET("/status", h.GetAuthStatus)
	rec = httptest.NewRecorder()
	statusRouter.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status?state="+state, nil))
	if !containsJSON(rec, `"status":"error"`) || !containsJSON(rec, "unknown or expired state") {
		t.Fatalf("status after cancel = %s, want unknown/expired error", rec.Body.String())
	}
}

// TestGetAuthStatusCompletedTombstone verifies the completed tombstone reports
// ok while it is retained (upstream GetAuthStatus end-state).
func TestGetAuthStatusCompletedTombstone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	router := gin.New()
	router.GET("/status", h.GetAuthStatus)

	state := "status-completed-state"
	RegisterOAuthSession(state, "codex")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status?state="+state, nil))
	if !containsJSON(rec, `"status":"wait"`) {
		t.Fatalf("pending status = %s, want wait", rec.Body.String())
	}

	CompleteOAuthSession(state)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status?state="+state, nil))
	if !containsJSON(rec, `"status":"ok"`) {
		t.Fatalf("completed status = %s, want ok", rec.Body.String())
	}
}

func TestCancelAuthSessionEndpointRejectsMissingAndInvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	router := gin.New()
	router.DELETE("/oauth-session", h.CancelAuthSession)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/oauth-session", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing state status = %d, want %d, body %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/oauth-session?state=bad/state", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid state status = %d, want %d, body %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func containsJSON(rec *httptest.ResponseRecorder, needle string) bool {
	return bytes.Contains(rec.Body.Bytes(), []byte(needle))
}
