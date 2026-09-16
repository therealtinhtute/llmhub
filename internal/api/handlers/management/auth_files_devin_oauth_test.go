package management

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/auth/devin"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

// Ported from upstream CLIProxyAPI
// internal/api/handlers/management/auth_files_devin_oauth_test.go
// (44e62bc8acc2). Local symbols under test: Handler.RequestDevinToken,
// Handler.completeDevinOAuth, newDevinOAuthService. Upstream's callback-file
// polling is replaced by the in-memory oauth session store
// (SubmitOAuthCallbackForPendingSession / WaitOAuthCallbackForPendingSession /
// oauthSessions.SetCallback).

type fakeDevinOAuthService struct {
	exchange func(context.Context, string, string) (string, error)
	create   func(context.Context, string) (*coreauth.Auth, error)
}

func (f *fakeDevinOAuthService) BuildAuthorizationURL(redirectURI, challenge, state string) string {
	return devin.NewDevinAuthService(nil).BuildAuthorizationURL(redirectURI, challenge, state)
}

func (f *fakeDevinOAuthService) ExchangeCodeForToken(ctx context.Context, code, verifier string) (string, error) {
	if f.exchange != nil {
		return f.exchange(ctx, code, verifier)
	}
	return "eyJ.test.token", nil
}

func (f *fakeDevinOAuthService) CreateAuthRecord(ctx context.Context, token string) (*coreauth.Auth, error) {
	if f.create != nil {
		return f.create(ctx, token)
	}
	return &coreauth.Auth{
		ID: "devin-test.json", FileName: "devin-test.json", Provider: "devin",
		Metadata: map[string]any{"type": "devin", "api_key": devin.FormatSessionToken(token), "auth_kind": "oauth"},
	}, nil
}

func TestDevinRemoteOAuthFlow(t *testing.T) {
	authDir := t.TempDir()
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir, Port: 8317}, nil)
	exchanged := make(chan string, 1)
	service := &fakeDevinOAuthService{exchange: func(ctx context.Context, code, verifier string) (string, error) {
		if code != "remote-code" {
			t.Errorf("code = %q", code)
		}
		if ctx.Err() != nil {
			t.Error("login inherited completed HTTP request cancellation")
		}
		exchanged <- verifier
		return "eyJ.test.token", nil
	}}
	originalFactory := newDevinOAuthService
	newDevinOAuthService = func(*config.Config) devinOAuthService { return service }
	t.Cleanup(func() { newDevinOAuthService = originalFactory })

	router := gin.New()
	router.GET("/devin-auth-url", h.RequestDevinToken)
	router.POST("/oauth-callback", h.PostOAuthCallback)
	router.GET("/get-auth-status", h.GetAuthStatus)
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/devin-auth-url?is_webui=true", nil).WithContext(requestCtx))
	cancelRequest()
	if w.Code != http.StatusOK {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var start struct{ State, URL, Status string }
	if errDecode := json.Unmarshal(w.Body.Bytes(), &start); errDecode != nil {
		t.Fatal(errDecode)
	}
	t.Cleanup(func() { CompleteOAuthSession(start.State) })
	u, errParse := url.Parse(start.URL)
	if errParse != nil {
		t.Fatal(errParse)
	}
	query := u.Query()
	if start.Status != "ok" || start.State == "" || query.Get("state") != start.State || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("invalid authorization response: %s", w.Body.String())
	}
	if got := query.Get("redirect_uri"); got != "http://127.0.0.1:8317/devin/callback" {
		t.Fatalf("redirect_uri = %q", got)
	}
	if query.Get("code_verifier") != "" {
		t.Fatal("PKCE verifier exposed")
	}

	secondState := start.State + "-second"
	RegisterOAuthSession(secondState, "devin")
	defer CompleteOAuthSession(secondState)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/get-auth-status?state="+start.State, nil))
	if !strings.Contains(w.Body.String(), `"status":"wait"`) {
		t.Fatalf("pending: %s", w.Body.String())
	}

	redirect := query.Get("redirect_uri") + "?code=remote-code&state=" + start.State
	body, errMarshal := json.Marshal(map[string]string{"provider": "cognition", "redirect_url": redirect})
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/oauth-callback", strings.NewReader(string(body))))
	if w.Code != http.StatusOK {
		t.Fatalf("callback: %d %s", w.Code, w.Body.String())
	}
	select {
	case verifier := <-exchanged:
		digest := sha256.Sum256([]byte(verifier))
		if base64.RawURLEncoding.EncodeToString(digest[:]) != query.Get("code_challenge") {
			t.Fatal("PKCE challenge does not match verifier")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("token exchange did not start")
	}

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/get-auth-status?state="+start.State, nil))
		var status map[string]string
		if errDecode := json.Unmarshal(w.Body.Bytes(), &status); errDecode != nil {
			t.Fatal(errDecode)
		}
		if status["status"] == "ok" {
			break
		}
		if status["status"] == "error" {
			t.Fatalf("login failed: %v", status)
		}
		select {
		case <-deadline.C:
			t.Fatal("login did not complete")
		case <-ticker.C:
		}
	}
	if !IsOAuthSessionPending(secondState, "devin") {
		t.Fatal("completed another login session")
	}
	data, errRead := os.ReadFile(filepath.Join(authDir, "devin-test.json"))
	if errRead != nil {
		t.Fatal(errRead)
	}
	var record map[string]any
	if errDecode := json.Unmarshal(data, &record); errDecode != nil {
		t.Fatal(errDecode)
	}
	if record["api_key"] != "devin-session-token$eyJ.test.token" || record["type"] != "devin" || record["auth_kind"] != "oauth" {
		t.Fatalf("unexpected credential: %v", record)
	}

	// Completed sessions keep a short-lived tombstone, so a replayed callback
	// reports 409 (upstream v7.3.4 handleOAuthCallback).
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/oauth-callback", strings.NewReader(string(body))))
	if w.Code != http.StatusConflict {
		t.Fatalf("replay: %d", w.Code)
	}
}

func TestCompleteDevinOAuthFailures(t *testing.T) {
	for _, test := range []struct {
		name               string
		payload            oauthCallbackFilePayload
		exchangeErr        bool
		emptyToken         bool
		cancelDuringCreate bool
		saveErr            bool
		want               string
	}{
		{name: "denied", payload: oauthCallbackFilePayload{Error: "access_denied"}, want: "Devin authorization denied"},
		{name: "mismatched state", payload: oauthCallbackFilePayload{State: "wrong", Code: "code"}, want: "State code error"},
		{name: "missing code", payload: oauthCallbackFilePayload{}, want: "Missing authorization code"},
		{name: "exchange error", payload: oauthCallbackFilePayload{Code: "code"}, exchangeErr: true, want: "Failed to exchange authorization code for tokens"},
		{name: "empty token", payload: oauthCallbackFilePayload{Code: "code"}, emptyToken: true, want: "Failed to exchange authorization code for tokens"},
		{name: "cancel during profile", payload: oauthCallbackFilePayload{Code: "code"}, cancelDuringCreate: true},
		{name: "save error", payload: oauthCallbackFilePayload{Code: "code"}, saveErr: true, want: "Failed to save authentication tokens"},
	} {
		t.Run(test.name, func(t *testing.T) {
			const state = "test-state"
			authDir := t.TempDir()
			h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)
			RegisterOAuthSession(state, "devin")
			defer CompleteOAuthSession(state)
			payload := test.payload
			if strings.TrimSpace(payload.State) == "" {
				payload.State = state
			}
			if errSubmit := oauthSessions.SetCallback(state, "devin", payload); errSubmit != nil {
				t.Fatal(errSubmit)
			}
			service := &fakeDevinOAuthService{}
			if test.exchangeErr {
				service.exchange = func(context.Context, string, string) (string, error) { return "", errors.New("secret-upstream-token") }
			}
			if test.emptyToken {
				service.exchange = func(context.Context, string, string) (string, error) { return "", nil }
			}
			if test.cancelDuringCreate {
				service.create = func(context.Context, string) (*coreauth.Auth, error) {
					CancelOAuthSession(state)
					return &coreauth.Auth{}, nil
				}
			}
			if test.saveErr {
				h.postAuthHook = func(context.Context, *coreauth.Auth) error { return errors.New("save failed") }
			}
			h.completeDevinOAuth(context.Background(), state, "verifier", service)
			_, status, ok := GetOAuthSession(state)
			if test.cancelDuringCreate {
				if ok {
					t.Fatal("cancelled session was recreated")
				}
			} else if !ok || status != test.want {
				t.Fatalf("status = %q, exists = %v; want %q", status, ok, test.want)
			}
			entries, errRead := os.ReadDir(authDir)
			if errRead != nil {
				t.Fatal(errRead)
			}
			if len(entries) != 0 {
				t.Fatalf("credentials saved for failed/cancelled flow: %v", entries)
			}
		})
	}
}

func TestCompleteDevinOAuthUnregisteredState(t *testing.T) {
	// With no pending session the waiter exits immediately without writing an
	// error (local equivalent of upstream's cancelled-session early return).
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	done := make(chan struct{})
	go func() {
		h.completeDevinOAuth(context.Background(), "no-such-state", "verifier", &fakeDevinOAuthService{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("completeDevinOAuth blocked on an unregistered state")
	}
	if _, _, ok := GetOAuthSession("no-such-state"); ok {
		t.Fatal("unregistered state created a session")
	}
}

// TestCancelOAuthSessionPreventsDevinSave verifies the cancel verb (ported from
// upstream 44e62bc8acc2-adjacent) aborts the flow at the save guard so no
// credentials are persisted for a cancelled session.
func TestCancelOAuthSessionPreventsDevinSave(t *testing.T) {
	const state = "cancel-save-state"
	authDir := t.TempDir()
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)
	RegisterOAuthSession(state, "devin")
	defer CompleteOAuthSession(state)
	if errSubmit := oauthSessions.SetCallback(state, "devin", oauthCallbackFilePayload{Code: "code", State: state}); errSubmit != nil {
		t.Fatal(errSubmit)
	}
	cancelled := false
	service := &fakeDevinOAuthService{
		create: func(context.Context, string) (*coreauth.Auth, error) {
			if !cancelled {
				cancelled = true
				CancelOAuthSession(state)
			}
			return &coreauth.Auth{ID: "devin-cancel.json", FileName: "devin-cancel.json", Provider: "devin"}, nil
		},
	}
	h.completeDevinOAuth(context.Background(), state, "verifier", service)
	if !cancelled {
		t.Fatal("create hook never ran; flow exited before reaching the save guard")
	}
	if IsOAuthSessionPending(state, "devin") {
		t.Fatal("session still pending after cancel")
	}
	entries, errRead := os.ReadDir(authDir)
	if errRead != nil {
		t.Fatal(errRead)
	}
	if len(entries) != 0 {
		t.Fatalf("credentials saved for cancelled flow: %v", entries)
	}
}
