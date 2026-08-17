package management

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/updater"
)

// testManagementKey is the shared env secret the test handlers accept; the
// Middleware would otherwise reject every request before reaching the
// handlers.
const testManagementKey = "test-key"

// fakeSelfUpdateEngine fakes SelfUpdateEngine outcomes, including the
// unsupported-platform result that the real engine cannot produce on
// linux/amd64.
type fakeSelfUpdateEngine struct {
	manifest updater.StagedManifest
	err      error
}

func (f fakeSelfUpdateEngine) StageLatest(ctx context.Context) (updater.StagedManifest, error) {
	return f.manifest, f.err
}

// newTestHandler builds a Handler with a valid management secret and the
// given engine/runner so tests exercise the real auth middleware.
func newTestHandler(engine SelfUpdateEngine, runner func(context.Context) error) *Handler {
	// allowRemoteOverride mirrors NewHandler, which sets it when a
	// MANAGEMENT_PASSWORD env secret is present. failedAttempts mirrors
	// NewHandler's map initialization.
	h := &Handler{cfg: &config.Config{}, envSecret: testManagementKey, allowRemoteOverride: true, failedAttempts: make(map[string]*attemptInfo)}
	if engine != nil {
		h.selfUpdateEngine = engine
	}
	if runner != nil {
		h.selfUpdateRunner = runner
	}
	return h
}

func callHandler(t *testing.T, h *Handler, route, method, key string) *httptest.ResponseRecorder {
	t.Helper()
	// No gin.SetMode: it writes an unsynchronized global, which races when
	// any test in the package runs in parallel.
	engine := gin.New()
	engine.Use(h.Middleware())
	engine.POST("/v0/management"+route, func(c *gin.Context) {
		switch route {
		case "/self-update":
			h.SelfUpdateStage(c)
		case "/self-update/apply":
			h.SelfUpdateApply(c)
		}
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/v0/management"+route, nil)
	if key != "" {
		req.Header.Set("X-Management-Key", key)
	}
	engine.ServeHTTP(rec, req)
	return rec
}

func TestSelfUpdateStage(t *testing.T) {
	h := newTestHandler(fakeSelfUpdateEngine{manifest: updater.StagedManifest{Version: "v9.9.9", Digest: "abc"}}, nil)
	rec := callHandler(t, h, "/self-update", http.MethodPost, testManagementKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var out struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "staged" || out.Version != "v9.9.9" {
		t.Fatalf("body = %s, want staged v9.9.9", rec.Body.String())
	}
}

func TestSelfUpdateStageCurrent(t *testing.T) {
	h := newTestHandler(fakeSelfUpdateEngine{err: updater.ErrUpToDate}, nil)
	rec := callHandler(t, h, "/self-update", http.MethodPost, testManagementKey)
	var out struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "current" {
		t.Fatalf("body = %s, want status current", rec.Body.String())
	}
}

func TestSelfUpdateStageRejected(t *testing.T) {
	for name, err := range map[string]error{
		"downgrade": updater.ErrDowngradeRefused,
		"dev-build": updater.ErrDevelopmentBuild,
	} {
		t.Run(name, func(t *testing.T) {
			h := newTestHandler(fakeSelfUpdateEngine{err: err}, nil)
			rec := callHandler(t, h, "/self-update", http.MethodPost, testManagementKey)
			var out struct {
				Status  string `json:"status"`
				Reason  string `json:"reason"`
				Version string `json:"version"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			if out.Status != "rejected" || out.Reason == "" || out.Version != "" {
				t.Fatalf("body = %s, want rejected with reason and no version", rec.Body.String())
			}
		})
	}
}

func TestSelfUpdateUnsupported(t *testing.T) {
	h := newTestHandler(fakeSelfUpdateEngine{err: updater.ErrUnsupportedPlatform}, nil)
	rec := callHandler(t, h, "/self-update", http.MethodPost, testManagementKey)
	var out struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "unsupported" {
		t.Fatalf("body = %s, want status unsupported", rec.Body.String())
	}
}

func TestSelfUpdateStageFailureIsTyped(t *testing.T) {
	// Any unexpected failure must be a typed outcome, never a leaked remote
	// body (R13): the fake's message would contain the remote text.
	h := newTestHandler(fakeSelfUpdateEngine{err: errors.New("remote body: {\"message\":\"boom\"}")}, nil)
	rec := callHandler(t, h, "/self-update", http.MethodPost, testManagementKey)
	if strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("remote body leaked into response: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "stage failed") {
		t.Fatalf("body = %s, want typed stage-failed outcome", rec.Body.String())
	}
}

func TestSelfUpdateStageUnconfigured(t *testing.T) {
	h := newTestHandler(nil, nil)
	rec := callHandler(t, h, "/self-update", http.MethodPost, testManagementKey)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestSelfUpdateApply(t *testing.T) {
	ran := make(chan struct{}, 1)
	h := newTestHandler(nil, func(ctx context.Context) error {
		ran <- struct{}{}
		return nil
	})
	rec := callHandler(t, h, "/self-update/apply", http.MethodPost, testManagementKey)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %s)", rec.Code, rec.Body.String())
	}
	var out struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "restarting" {
		t.Fatalf("body = %s, want status restarting", rec.Body.String())
	}
	// The runner fires in a goroutine after the response; wait for it once.
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("restart runner was never invoked")
	}
	// And only once.
	select {
	case <-ran:
		t.Fatal("restart runner invoked more than once")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSelfUpdateApplyFixedCommand(t *testing.T) {
	// The default command is the single fixed systemctl restart, with no
	// shell, wildcard, or argument variability (R13, R14).
	want := "sudo -n /usr/bin/systemctl restart llmhub.service"
	if got := strings.Join(restartCommand, " "); got != want {
		t.Fatalf("restartCommand = %q, want %q", got, want)
	}
}

func TestSelfUpdateAuth(t *testing.T) {
	h := newTestHandler(fakeSelfUpdateEngine{manifest: updater.StagedManifest{Version: "v9.9.9"}}, nil)
	for _, route := range []string{"/self-update", "/self-update/apply"} {
		t.Run(route, func(t *testing.T) {
			// No key at all.
			rec := callHandler(t, h, route, http.MethodPost, "")
			if rec.Code == http.StatusOK || rec.Code == http.StatusAccepted {
				t.Fatalf("route reachable without a key: status %d", rec.Code)
			}
			// Wrong key.
			rec = callHandler(t, h, route, http.MethodPost, "wrong-key")
			if rec.Code == http.StatusOK || rec.Code == http.StatusAccepted {
				t.Fatalf("route reachable with a wrong key: status %d", rec.Code)
			}
		})
	}
}
