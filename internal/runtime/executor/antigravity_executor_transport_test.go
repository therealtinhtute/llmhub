package executor

// Ported from upstream CLIProxyAPI commits:
//   - d0fb44ca95e8 (fix(antigravity): strip tool config, labels, and session id in token counting)
//   - d5397905f09e (fix(antigravity): default to short connections and harden connection pool lifecycle)
//   - 68dd99d56f68 (fix(antigravity): include resolved pool settings in transport cache key)
// Local symbols under test: AntigravityExecutor.CountTokens,
// resolveAntigravityPoolSettings, applyAntigravityPoolLimits,
// antigravityHTTP11Transport, antigravityTransportKey,
// closeAntigravityAuthIdleTransports, ResetAntigravityTransports,
// AntigravityTransportsLen.
// NOTE: upstream nests pool config under cfg.Antigravity.ConnectionPool; this
// repository exposes it as cfg.AntigravityConnectionPool (flat antigravity-* schema).

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
)

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

func TestAntigravityExecutorCountTokensStripsStatefulFields(t *testing.T) {
	var upstreamBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != antigravityCountTokensPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, antigravityCountTokensPath)
		}
		body, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			t.Fatalf("read countTokens body: %v", errRead)
		}
		upstreamBody = append([]byte(nil), body...)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalTokens":42}`))
	}))
	defer server.Close()

	payload := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"toolConfig":{"functionCallingConfig":{"mode":"AUTO"}},"labels":{"source":"ide"},"sessionId":"session-123"}`)
	exec := NewAntigravityExecutor(&config.Config{RequestRetry: 1})
	_, errCount := exec.CountTokens(context.Background(), testAntigravityAuth(server.URL), cliproxyexecutor.Request{
		Model:   "gemini-3-pro-preview",
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatGemini,
	})
	if errCount != nil {
		t.Fatalf("CountTokens() error = %v", errCount)
	}
	if len(upstreamBody) == 0 {
		t.Fatal("countTokens upstream body was not captured")
	}
	if gjson.GetBytes(upstreamBody, "request.toolConfig").Exists() {
		t.Fatalf("upstream countTokens body retained request.toolConfig: %s", upstreamBody)
	}
	if gjson.GetBytes(upstreamBody, "request.labels").Exists() {
		t.Fatalf("upstream countTokens body retained request.labels: %s", upstreamBody)
	}
	if gjson.GetBytes(upstreamBody, "request.sessionId").Exists() {
		t.Fatalf("upstream countTokens body retained request.sessionId: %s", upstreamBody)
	}
}

func TestResolveAntigravityPoolSettings(t *testing.T) {
	t.Run("nil config defaults to short mode", func(t *testing.T) {
		settings := resolveAntigravityPoolSettings(nil)
		if !settings.shortMode || settings.maxIdleConnsPerHost != -1 {
			t.Fatalf("settings = %+v, want short mode", settings)
		}
	})
	t.Run("empty config defaults to short mode", func(t *testing.T) {
		settings := resolveAntigravityPoolSettings(&config.Config{})
		if !settings.shortMode || settings.maxIdleConnsPerHost != -1 {
			t.Fatalf("settings = %+v, want short mode", settings)
		}
	})
	t.Run("disabled pooling stays short mode", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(false)
		settings := resolveAntigravityPoolSettings(cfg)
		if !settings.shortMode || settings.maxIdleConnsPerHost != -1 {
			t.Fatalf("settings = %+v, want short mode", settings)
		}
	})
	t.Run("enabled pooling applies defaults", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
		settings := resolveAntigravityPoolSettings(cfg)
		if settings.shortMode {
			t.Fatalf("settings = %+v, want pooling enabled", settings)
		}
		if settings.maxIdleConnsPerHost != antigravityDefaultMaxIdleConnsPerHost {
			t.Fatalf("maxIdleConnsPerHost = %d, want %d", settings.maxIdleConnsPerHost, antigravityDefaultMaxIdleConnsPerHost)
		}
		if settings.idleConnTimeout != antigravityDefaultIdleConnTimeout {
			t.Fatalf("idleConnTimeout = %v, want %v", settings.idleConnTimeout, antigravityDefaultIdleConnTimeout)
		}
	})
	t.Run("custom values honored within bounds", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
		cfg.AntigravityConnectionPool.IdleConnTimeout = "60s"
		cfg.AntigravityConnectionPool.MaxIdleConnsPerHost = intPtr(10)
		settings := resolveAntigravityPoolSettings(cfg)
		if settings.shortMode || settings.maxIdleConnsPerHost != 10 || settings.idleConnTimeout != 60*time.Second {
			t.Fatalf("settings = %+v, want {10, 60s}", settings)
		}
	})
	t.Run("timeout above cap is clamped", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
		cfg.AntigravityConnectionPool.IdleConnTimeout = "300s"
		settings := resolveAntigravityPoolSettings(cfg)
		if settings.idleConnTimeout != antigravityMaxAllowedIdleConnTimeout {
			t.Fatalf("idleConnTimeout = %v, want cap %v", settings.idleConnTimeout, antigravityMaxAllowedIdleConnTimeout)
		}
	})
	t.Run("non-positive timeout forces short mode", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
		cfg.AntigravityConnectionPool.IdleConnTimeout = "0s"
		settings := resolveAntigravityPoolSettings(cfg)
		if !settings.shortMode || settings.maxIdleConnsPerHost != -1 {
			t.Fatalf("settings = %+v, want short mode", settings)
		}
	})
	t.Run("invalid timeout falls back to default", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
		cfg.AntigravityConnectionPool.IdleConnTimeout = "not-a-duration"
		settings := resolveAntigravityPoolSettings(cfg)
		if settings.shortMode || settings.idleConnTimeout != antigravityDefaultIdleConnTimeout {
			t.Fatalf("settings = %+v, want default timeout", settings)
		}
	})
	t.Run("negative max idle forces short mode", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
		cfg.AntigravityConnectionPool.MaxIdleConnsPerHost = intPtr(-1)
		settings := resolveAntigravityPoolSettings(cfg)
		if !settings.shortMode || settings.maxIdleConnsPerHost != -1 {
			t.Fatalf("settings = %+v, want short mode", settings)
		}
	})
	t.Run("max idle above cap is clamped", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
		cfg.AntigravityConnectionPool.MaxIdleConnsPerHost = intPtr(500)
		settings := resolveAntigravityPoolSettings(cfg)
		if settings.maxIdleConnsPerHost != antigravityMaxAllowedMaxIdleConnsPerHost {
			t.Fatalf("maxIdleConnsPerHost = %d, want cap %d", settings.maxIdleConnsPerHost, antigravityMaxAllowedMaxIdleConnsPerHost)
		}
	})
}

func TestApplyAntigravityPoolLimitsShortMode(t *testing.T) {
	transport := &http.Transport{}
	applyAntigravityPoolLimits(transport, &config.Config{})
	if transport.MaxIdleConnsPerHost != -1 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want -1 in short mode", transport.MaxIdleConnsPerHost)
	}
	if transport.DisableKeepAlives {
		t.Fatal("DisableKeepAlives must stay false so no Connection: close header leaks")
	}
	if transport.IdleConnTimeout != 0 {
		t.Fatalf("IdleConnTimeout = %v, want 0 in short mode", transport.IdleConnTimeout)
	}
}

func TestApplyAntigravityPoolLimitsEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.AntigravityConnectionPool.Enabled = boolPtr(true)
	transport := &http.Transport{}
	applyAntigravityPoolLimits(transport, cfg)
	if transport.MaxIdleConnsPerHost != antigravityDefaultMaxIdleConnsPerHost {
		t.Fatalf("MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, antigravityDefaultMaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != antigravityDefaultIdleConnTimeout {
		t.Fatalf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, antigravityDefaultIdleConnTimeout)
	}
}

// TestAntigravityTransportKeySeparatesPoolSettingsAcrossReload is a regression
// test for stale transport reuse when hot-reload flips pool settings: the same
// credential and base transport under different resolved settings must produce
// distinct cache entries.
func TestAntigravityTransportKeySeparatesPoolSettingsAcrossReload(t *testing.T) {
	antigravityTransports.Purge()
	defer antigravityTransports.Purge()

	auth := testAntigravityAuth("http://upstream.invalid")
	auth.ID = "cred-1"

	shortCfg := &config.Config{}
	pooledCfg := &config.Config{}
	pooledCfg.AntigravityConnectionPool.Enabled = boolPtr(true)

	first := antigravityHTTP11Transport(auth, antigravityBaseTransport, shortCfg)
	if first == nil {
		t.Fatal("short-mode transport is nil")
	}
	second := antigravityHTTP11Transport(auth, antigravityBaseTransport, shortCfg)
	if second != first {
		t.Fatal("identical pool settings should reuse the cached transport")
	}

	reloaded := antigravityHTTP11Transport(auth, antigravityBaseTransport, pooledCfg)
	if reloaded == nil {
		t.Fatal("pooled transport is nil")
	}
	if reloaded == first {
		t.Fatal("transport key did not separate short-mode and pooled settings")
	}
	if got := AntigravityTransportsLen(); got != 2 {
		t.Fatalf("AntigravityTransportsLen() = %d, want 2", got)
	}
}

func TestCloseAntigravityAuthIdleTransports(t *testing.T) {
	antigravityTransports.Purge()
	defer antigravityTransports.Purge()

	auth := testAntigravityAuth("http://upstream.invalid")
	auth.ID = "cred-close"
	other := testAntigravityAuth("http://upstream.invalid")
	other.ID = "cred-other"

	antigravityHTTP11Transport(auth, antigravityBaseTransport, &config.Config{})
	antigravityHTTP11Transport(other, antigravityBaseTransport, &config.Config{})
	if got := AntigravityTransportsLen(); got != 2 {
		t.Fatalf("AntigravityTransportsLen() = %d, want 2", got)
	}

	closeAntigravityAuthIdleTransports(auth)
	if got := AntigravityTransportsLen(); got != 1 {
		t.Fatalf("after close, AntigravityTransportsLen() = %d, want 1", got)
	}
	closeAntigravityAuthIdleTransports(other)
	if got := AntigravityTransportsLen(); got != 0 {
		t.Fatalf("after closing both, AntigravityTransportsLen() = %d, want 0", got)
	}
}

func TestAntigravityTransportScopeFallbacks(t *testing.T) {
	if scope := antigravityTransportScope(nil); scope != antigravityAnonymousTransportScope {
		t.Fatalf("nil auth scope = %q, want %q", scope, antigravityAnonymousTransportScope)
	}
	if scope := antigravityTransportScope(&cliproxyauth.Auth{ID: "  abc "}); scope != "id:abc" {
		t.Fatalf("id scope = %q, want id:abc", scope)
	}
	byPath := &cliproxyauth.Auth{Attributes: map[string]string{"path": "/tmp/cred.json"}}
	if scope := antigravityTransportScope(byPath); scope != "path:/tmp/cred.json" {
		t.Fatalf("path scope = %q", scope)
	}
	byToken := &cliproxyauth.Auth{Metadata: map[string]any{"refresh_token": "refresh-secret"}}
	scope := antigravityTransportScope(byToken)
	if scope == "" || scope == antigravityAnonymousTransportScope || scope == "refresh-secret" {
		t.Fatalf("refresh-token scope must be a stable digest, got %q", scope)
	}
	// Anonymous auth shares one scope so it never allocates a pool per request.
	if s := antigravityTransportScope(&cliproxyauth.Auth{}); s != antigravityAnonymousTransportScope {
		t.Fatalf("empty auth scope = %q, want anonymous", s)
	}
}
