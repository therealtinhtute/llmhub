package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	management "github.com/therealtinhtute/llmhub/internal/api/handlers/management"
)

// Ported from upstream CLIProxyAPI internal/api/server_devin_oauth_test.go
// (44e62bc8acc2). Upstream verified the callback through a persisted
// .oauth-devin-<state>.oauth file; the local flow keeps callbacks in the
// in-memory oauth session store, so the test drains them via
// management.WaitOAuthCallbackForPendingSession instead.
func TestDevinOAuthRoutes(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "test-management-key")
	server := newTestServer(t)

	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v0/management/devin-auth-url", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unprotected login route: %d", w.Code)
	}
	registered := false
	for _, route := range server.engine.Routes() {
		if route.Method == http.MethodGet && route.Path == "/v0/management/devin-auth-url" {
			registered = true
		}
	}
	if !registered {
		t.Fatal("Devin login route not registered")
	}

	for _, test := range []struct {
		name, provider, query string
		want                  int
	}{
		{name: "success", provider: "devin", query: "code=test-code", want: http.StatusOK},
		{name: "denied", provider: "devin", query: "error=access_denied", want: http.StatusOK},
		{name: "wrong provider", provider: "codex", query: "code=test-code", want: http.StatusBadRequest},
		{name: "missing code", provider: "devin", want: http.StatusBadRequest},
		{name: "unknown state", query: "code=test-code", want: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := "devin-route-" + strings.ReplaceAll(test.name, " ", "-")
			if test.provider != "" {
				management.RegisterOAuthSession(state, test.provider)
				defer management.CompleteOAuthSession(state)
			}
			w := httptest.NewRecorder()
			server.engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/devin/callback?state="+state+"&"+test.query, nil))
			if w.Code != test.want {
				t.Fatalf("callback: %d %s", w.Code, w.Body.String())
			}
			if test.want == http.StatusOK {
				payload, errWait := management.WaitOAuthCallbackForPendingSession("devin", state, time.Second)
				if errWait != nil {
					t.Fatal(errWait)
				}
				if payload["state"] != state || (payload["code"] == "" && payload["error"] == "") {
					t.Fatalf("callback payload: %v", payload)
				}
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("callback response is cacheable")
				}
			}
		})
	}
}
