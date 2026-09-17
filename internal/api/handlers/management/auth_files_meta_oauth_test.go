package management

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	metaauth "github.com/therealtinhtute/llmhub/internal/auth/meta"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

// Tests for Handler.RequestMetaToken / Handler.completeMetaOAuth, the local
// port of upstream CLIProxyAPI
// internal/api/handlers/management/auth_files_provider_oauth.go RequestMetaToken
// (23c16e2985bb, end-state at v7.3.4 checkpoint 8335eac73194). Upstream ships
// no dedicated meta handler test file; these mirror
// auth_files_devin_oauth_test.go adapted to the RFC 8628 device flow — there is
// no callback step, WaitForAuthorization polls the token endpoint server-side.

type fakeMetaOAuthService struct {
	start   func(context.Context) (*metaauth.DeviceCodeResponse, error)
	wait    func(context.Context, *metaauth.DeviceCodeResponse) (*metaauth.MetaAuthBundle, error)
	storage func(*metaauth.MetaAuthBundle) *metaauth.MetaTokenStorage
}

func (f *fakeMetaOAuthService) StartDeviceFlow(ctx context.Context) (*metaauth.DeviceCodeResponse, error) {
	if f.start != nil {
		return f.start(ctx)
	}
	return &metaauth.DeviceCodeResponse{
		DeviceCode:              "device-code",
		UserCode:                "ABCD-1234",
		VerificationURI:         "https://auth.meta.com/device",
		VerificationURIComplete: "https://auth.meta.com/device?user_code=ABCD-1234",
		ExpiresIn:               900,
		Interval:                5,
	}, nil
}

func (f *fakeMetaOAuthService) WaitForAuthorization(ctx context.Context, dcr *metaauth.DeviceCodeResponse) (*metaauth.MetaAuthBundle, error) {
	if f.wait != nil {
		return f.wait(ctx, dcr)
	}
	return &metaauth.MetaAuthBundle{
		TokenData: &metaauth.TokenData{AccessToken: "dca:test-token", TokenType: "Bearer", ExpiresIn: 3600, ExpiresAt: time.Now().Add(time.Hour).Unix()},
		MintedKey: &metaauth.MintedKeyResponse{APIKey: "mk-test-key", BaseURL: "https://api.meta.ai/v1", UserEmail: "user@example.com", UserFullName: "Meta User"},
		Email:     "user@example.com",
		Name:      "Meta User",
	}, nil
}

func (f *fakeMetaOAuthService) CreateTokenStorage(bundle *metaauth.MetaAuthBundle) *metaauth.MetaTokenStorage {
	if f.storage != nil {
		return f.storage(bundle)
	}
	// Exercise the real bundle→storage mapping (minted key becomes the usable
	// access_token; DCA token retained for on-demand re-mint).
	return metaauth.NewMetaAuth(nil).CreateTokenStorage(bundle)
}

func stubMetaOAuthService(t *testing.T, svc metaOAuthService) {
	t.Helper()
	original := newMetaOAuthService
	newMetaOAuthService = func(*config.Config) metaOAuthService { return svc }
	t.Cleanup(func() { newMetaOAuthService = original })
}

func TestMetaDeviceOAuthFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authDir := t.TempDir()
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir, Port: 8317}, nil)

	waited := make(chan struct{}, 1)
	release := make(chan struct{})
	service := &fakeMetaOAuthService{
		wait: func(ctx context.Context, dcr *metaauth.DeviceCodeResponse) (*metaauth.MetaAuthBundle, error) {
			if dcr.DeviceCode != "device-code" {
				t.Errorf("device code = %q", dcr.DeviceCode)
			}
			if ctx.Err() != nil {
				t.Error("device poll inherited completed HTTP request cancellation")
			}
			waited <- struct{}{}
			// Hold the poll open so the pending-status assertion below is
			// deterministic — otherwise the session can complete (tombstone)
			// before get-auth-status runs and the test flakes.
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return &metaauth.MetaAuthBundle{
				TokenData: &metaauth.TokenData{AccessToken: "dca:test-token", TokenType: "Bearer", ExpiresIn: 3600, ExpiresAt: time.Now().Add(time.Hour).Unix()},
				MintedKey: &metaauth.MintedKeyResponse{APIKey: "mk-test-key", BaseURL: "https://api.meta.ai/v1", UserEmail: "user@example.com", UserFullName: "Meta User"},
				Email:     "user@example.com",
				Name:      "Meta User",
			}, nil
		},
	}
	stubMetaOAuthService(t, service)

	router := gin.New()
	router.GET("/meta-auth-url", h.RequestMetaToken)
	router.GET("/get-auth-status", h.GetAuthStatus)

	requestCtx, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/meta-auth-url?is_webui=true", nil).WithContext(requestCtx))
	cancelRequest()
	if w.Code != http.StatusOK {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var start struct {
		Status   string `json:"status"`
		URL      string `json:"url"`
		State    string `json:"state"`
		Flow     string `json:"flow"`
		UserCode string `json:"user_code"`
		Expires  int    `json:"expires_in"`
	}
	if errDecode := json.Unmarshal(w.Body.Bytes(), &start); errDecode != nil {
		t.Fatal(errDecode)
	}
	t.Cleanup(func() { CompleteOAuthSession(start.State) })

	if start.Status != "ok" || start.State == "" || !strings.HasPrefix(start.State, "meta-") {
		t.Fatalf("invalid start response: %s", w.Body.String())
	}
	// Device flows return the verification URL + user_code, not a plain auth URL.
	if start.Flow != "device" {
		t.Fatalf("flow = %q, want device", start.Flow)
	}
	if start.URL != "https://auth.meta.com/device?user_code=ABCD-1234" {
		t.Fatalf("url = %q, want verification_uri_complete", start.URL)
	}
	if start.UserCode != "ABCD-1234" {
		t.Fatalf("user_code = %q", start.UserCode)
	}
	if start.Expires != 900 {
		t.Fatalf("expires_in = %d, want 900", start.Expires)
	}

	select {
	case <-waited:
	case <-time.After(5 * time.Second):
		t.Fatal("device authorization poll did not start")
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/get-auth-status?state="+start.State, nil))
	if !strings.Contains(w.Body.String(), `"status":"wait"`) {
		t.Fatalf("pending: %s", w.Body.String())
	}

	close(release)

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

	// The minted API key becomes the usable credential; the DCA token is kept
	// for on-demand re-mint (upstream cee799f61af3/be7323f3bf66 semantics).
	matches, errGlob := filepath.Glob(filepath.Join(authDir, "meta-*.json"))
	if errGlob != nil || len(matches) != 1 {
		t.Fatalf("expected exactly one meta credential file, got %v (err=%v)", matches, errGlob)
	}
	data, errRead := os.ReadFile(matches[0])
	if errRead != nil {
		t.Fatal(errRead)
	}
	var record map[string]any
	if errDecode := json.Unmarshal(data, &record); errDecode != nil {
		t.Fatal(errDecode)
	}
	if record["type"] != "meta" || record["auth_kind"] != "oauth" {
		t.Fatalf("unexpected credential type/kind: %v", record)
	}
	if record["api_key"] != "mk-test-key" || record["access_token"] != "mk-test-key" {
		t.Fatalf("minted key not persisted as usable credential: %v", record)
	}
	if record["dca_token"] != "dca:test-token" {
		t.Fatalf("dca_token = %v", record["dca_token"])
	}
	if record["email"] != "user@example.com" || record["name"] != "Meta User" {
		t.Fatalf("identity fields = %v", record)
	}
}

func TestRequestMetaTokenStartFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	stubMetaOAuthService(t, &fakeMetaOAuthService{
		start: func(context.Context) (*metaauth.DeviceCodeResponse, error) {
			return nil, errors.New("auth.meta.com unreachable")
		},
	})

	router := gin.New()
	router.GET("/meta-auth-url", h.RequestMetaToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/meta-auth-url", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "failed to start device authorization flow") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestCompleteMetaOAuthFailures(t *testing.T) {
	for _, test := range []struct {
		name                string
		waitErr             error
		cancelDuringWait    bool
		emptyAccessToken    bool
		cancelDuringStorage bool
		saveErr             bool
		want                string
	}{
		{name: "authorization error", waitErr: errors.New("meta auth: access was denied by user"), want: "Authentication failed"},
		{name: "cancel during wait", waitErr: errors.New("context canceled"), cancelDuringWait: true},
		{name: "empty access token", emptyAccessToken: true, want: "Failed to exchange token"},
		{name: "cancel during storage", cancelDuringStorage: true},
		{name: "save error", saveErr: true, want: "Failed to save token to file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			const state = "meta-test-state"
			authDir := t.TempDir()
			h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)
			RegisterOAuthSession(state, "meta")
			defer CompleteOAuthSession(state)

			service := &fakeMetaOAuthService{}
			service.wait = func(ctx context.Context, dcr *metaauth.DeviceCodeResponse) (*metaauth.MetaAuthBundle, error) {
				if test.cancelDuringWait {
					CancelOAuthSession(state)
				}
				if test.waitErr != nil {
					return nil, test.waitErr
				}
				return &metaauth.MetaAuthBundle{
					TokenData: &metaauth.TokenData{AccessToken: "dca:x", ExpiresAt: time.Now().Add(time.Hour).Unix()},
				}, nil
			}
			if test.emptyAccessToken {
				service.storage = func(*metaauth.MetaAuthBundle) *metaauth.MetaTokenStorage {
					return &metaauth.MetaTokenStorage{}
				}
			}
			if test.cancelDuringStorage {
				service.storage = func(bundle *metaauth.MetaAuthBundle) *metaauth.MetaTokenStorage {
					CancelOAuthSession(state)
					return metaauth.NewMetaAuth(nil).CreateTokenStorage(bundle)
				}
			}
			if test.saveErr {
				h.postAuthHook = func(context.Context, *coreauth.Auth) error { return errors.New("save failed") }
			}

			h.completeMetaOAuth(context.Background(), state, &metaauth.DeviceCodeResponse{DeviceCode: "dc"}, service)

			_, status, ok := GetOAuthSession(state)
			if test.cancelDuringWait || test.cancelDuringStorage {
				if ok {
					t.Fatal("cancelled session was recreated")
				}
			} else if !ok || !strings.HasPrefix(status, test.want) {
				t.Fatalf("status = %q, exists = %v; want prefix %q", status, ok, test.want)
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

// TestCompleteMetaOAuthUnregisteredState mirrors
// TestCompleteDevinOAuthUnregisteredState: with no pending session the waiter
// exits without writing an error or saving credentials.
func TestCompleteMetaOAuthUnregisteredState(t *testing.T) {
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	done := make(chan struct{})
	go func() {
		h.completeMetaOAuth(context.Background(), "meta-no-such-state", &metaauth.DeviceCodeResponse{DeviceCode: "dc"}, &fakeMetaOAuthService{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("completeMetaOAuth blocked on an unregistered state")
	}
	if _, _, ok := GetOAuthSession("meta-no-such-state"); ok {
		t.Fatal("unregistered state created a session")
	}
}

// TestCancelOAuthSessionPreventsMetaSave verifies the save guard stops a
// session cancelled while the device poll was in flight — the meta equivalent
// of TestCancelOAuthSessionPreventsDevinSave (upstream 6e819ab62257 cancel
// semantics).
func TestCancelOAuthSessionPreventsMetaSave(t *testing.T) {
	const state = "meta-cancel-save-state"
	authDir := t.TempDir()
	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: authDir}, nil)
	RegisterOAuthSession(state, "meta")
	defer CompleteOAuthSession(state)

	cancelled := false
	service := &fakeMetaOAuthService{
		storage: func(bundle *metaauth.MetaAuthBundle) *metaauth.MetaTokenStorage {
			if !cancelled {
				cancelled = true
				CancelOAuthSession(state)
			}
			return metaauth.NewMetaAuth(nil).CreateTokenStorage(bundle)
		},
	}
	h.completeMetaOAuth(context.Background(), state, &metaauth.DeviceCodeResponse{DeviceCode: "dc"}, service)
	if !cancelled {
		t.Fatal("storage hook never ran; flow exited before reaching the save guard")
	}
	if IsOAuthSessionPending(state, "meta") {
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
