package util

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestApplyCustomHeadersFromAttrs_StaticHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com", nil)
	attrs := map[string]string{
		"header:X-Custom-Static": "static-value",
		"header:Host":            "custom.host.com",
	}

	ApplyCustomHeadersFromAttrs(req, attrs)

	if got := req.Header.Get("X-Custom-Static"); got != "static-value" {
		t.Errorf("X-Custom-Static = %q, want %q", got, "static-value")
	}
	if got := req.Host; got != "custom.host.com" {
		t.Errorf("req.Host = %q, want %q", got, "custom.host.com")
	}
}

func TestApplyCustomHeadersFromAttrs_MagicVariable(t *testing.T) {
	t.Run("present in client headers sets header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com", nil)
		attrs := map[string]string{
			"header:X-Claude-Code-Session-Id": "$ABC",
			"header:X-Target-Session":         "$X-Claude-Code-Session-Id",
			"header:Static-Header":            "static-123",
		}
		clientHeaders := http.Header{
			"Abc":                      []string{"session-abc-456"},
			"X-Claude-Code-Session-Id": []string{"claude-code-uuid-789"},
		}

		ApplyCustomHeadersFromAttrs(req, attrs, clientHeaders)

		if got := req.Header.Get("X-Claude-Code-Session-Id"); got != "session-abc-456" {
			t.Errorf("X-Claude-Code-Session-Id = %q, want %q", got, "session-abc-456")
		}
		if got := req.Header.Get("X-Target-Session"); got != "claude-code-uuid-789" {
			t.Errorf("X-Target-Session = %q, want %q", got, "claude-code-uuid-789")
		}
		if got := req.Header.Get("Static-Header"); got != "static-123" {
			t.Errorf("Static-Header = %q, want %q", got, "static-123")
		}
	})

	t.Run("uses first non-empty client value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com", nil)
		ApplyCustomHeadersFromAttrs(req, map[string]string{
			"header:X-Target-Session": "$ABC",
		}, http.Header{
			"ABC": []string{"", "session-abc-456"},
		})

		if got := req.Header.Get("X-Target-Session"); got != "session-abc-456" {
			t.Errorf("X-Target-Session = %q, want first non-empty value", got)
		}
	})

	t.Run("absent in client headers omits dynamic headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com", nil)
		attrs := map[string]string{
			"header:X-Claude-Code-Session-Id": "$ABC",
			"header:X-Other":                  "$NONEXISTENT",
			"header:Static-Header":            "static-123",
		}
		clientHeaders := http.Header{"Other-Header": []string{"some-value"}}

		ApplyCustomHeadersFromAttrs(req, attrs, clientHeaders)

		if _, exists := req.Header["X-Claude-Code-Session-Id"]; exists {
			t.Errorf("expected X-Claude-Code-Session-Id to be omitted, got %q", req.Header.Get("X-Claude-Code-Session-Id"))
		}
		if _, exists := req.Header["X-Other"]; exists {
			t.Errorf("expected X-Other to be omitted, got %q", req.Header.Get("X-Other"))
		}
		if got := req.Header.Get("Static-Header"); got != "static-123" {
			t.Errorf("Static-Header = %q, want %q", got, "static-123")
		}
	})

	t.Run("nil client headers omit dynamic headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com", nil)
		attrs := map[string]string{
			"header:X-Claude-Code-Session-Id": "$ABC",
			"header:Static-Header":            "static-123",
		}

		ApplyCustomHeadersFromAttrs(req, attrs)

		if _, exists := req.Header["X-Claude-Code-Session-Id"]; exists {
			t.Errorf("expected X-Claude-Code-Session-Id to be omitted, got %q", req.Header.Get("X-Claude-Code-Session-Id"))
		}
		if got := req.Header.Get("Static-Header"); got != "static-123" {
			t.Errorf("Static-Header = %q, want %q", got, "static-123")
		}
	})

	t.Run("fallback to gin context in request context", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(w)
		ginReq := httptest.NewRequest(http.MethodPost, "/", nil)
		ginReq.Header.Set("ABC", "from-gin-ctx-123")
		ginCtx.Request = ginReq

		req := httptest.NewRequest(http.MethodPost, "https://api.example.com", nil)
		req = req.WithContext(ginCtx)

		ApplyCustomHeadersFromAttrs(req, map[string]string{
			"header:X-Claude-Code-Session-Id": "$ABC",
		})

		if got := req.Header.Get("X-Claude-Code-Session-Id"); got != "from-gin-ctx-123" {
			t.Errorf("X-Claude-Code-Session-Id = %q, want %q", got, "from-gin-ctx-123")
		}
	})
}
