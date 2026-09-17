package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/interfaces"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	sdkconfig "github.com/therealtinhtute/llmhub/sdk/config"
)

func TestWriteErrorResponse_AddonHeadersDisabledByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := NewBaseAPIHandlers(nil, nil)
	handler.WriteErrorResponse(c, &interfaces.ErrorMessage{
		StatusCode: http.StatusTooManyRequests,
		Error:      errors.New("rate limit"),
		Addon: http.Header{
			"Retry-After":  {"30"},
			"X-Request-Id": {"req-1"},
		},
	})

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
	if got := recorder.Header().Get("Retry-After"); got != "" {
		t.Fatalf("Retry-After should be empty when passthrough is disabled, got %q", got)
	}
	if got := recorder.Header().Get("X-Request-Id"); got != "" {
		t.Fatalf("X-Request-Id should be empty when passthrough is disabled, got %q", got)
	}
}

func TestWriteErrorResponse_AddonHeadersEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Writer.Header().Set("X-Request-Id", "old-value")

	handler := NewBaseAPIHandlers(&sdkconfig.SDKConfig{PassthroughHeaders: true}, nil)
	handler.WriteErrorResponse(c, &interfaces.ErrorMessage{
		StatusCode: http.StatusTooManyRequests,
		Error:      errors.New("rate limit"),
		Addon: http.Header{
			"Retry-After":  {"30"},
			"X-Request-Id": {"new-1", "new-2"},
		},
	})

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
	if got := recorder.Header().Get("Retry-After"); got != "30" {
		t.Fatalf("Retry-After = %q, want %q", got, "30")
	}
	if got := recorder.Header().Values("X-Request-Id"); !reflect.DeepEqual(got, []string{"new-1", "new-2"}) {
		t.Fatalf("X-Request-Id = %#v, want %#v", got, []string{"new-1", "new-2"})
	}
}

func TestWriteErrorResponse_TrustedHomeBusyRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler := NewBaseAPIHandlers(nil, nil)
	handler.WriteErrorResponse(c, &interfaces.ErrorMessage{
		StatusCode: http.StatusTooManyRequests,
		Error:      coreauth.NewHomeConcurrencyBusyError("busy", 750*time.Millisecond),
	})

	if got := recorder.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want 1", got)
	}
}

func TestEnrichAuthSelectionError_DefaultsTo503WithContext(t *testing.T) {
	in := &coreauth.Error{Code: "auth_not_found", Message: "no auth available"}
	out := enrichAuthSelectionError(in, []string{"claude"}, "claude-sonnet-4-6")

	var got *coreauth.Error
	if !errors.As(out, &got) || got == nil {
		t.Fatalf("expected coreauth.Error, got %T", out)
	}
	if got.StatusCode() != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", got.StatusCode(), http.StatusServiceUnavailable)
	}
	if !strings.Contains(got.Message, "providers=claude") {
		t.Fatalf("message missing provider context: %q", got.Message)
	}
	if !strings.Contains(got.Message, "model=claude-sonnet-4-6") {
		t.Fatalf("message missing model context: %q", got.Message)
	}
	if !strings.Contains(got.Message, "/v0/management/auth-files") {
		t.Fatalf("message missing management hint: %q", got.Message)
	}
}

func TestEnrichAuthSelectionError_PreservesExplicitStatus(t *testing.T) {
	in := &coreauth.Error{Code: "auth_unavailable", Message: "no auth available", HTTPStatus: http.StatusTooManyRequests}
	out := enrichAuthSelectionError(in, []string{"gemini"}, "gemini-2.5-pro")

	var got *coreauth.Error
	if !errors.As(out, &got) || got == nil {
		t.Fatalf("expected coreauth.Error, got %T", out)
	}
	if got.StatusCode() != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", got.StatusCode(), http.StatusTooManyRequests)
	}
}

func TestEnrichAuthSelectionError_IgnoresOtherErrors(t *testing.T) {
	in := errors.New("boom")
	out := enrichAuthSelectionError(in, []string{"claude"}, "claude-sonnet-4-6")
	if out != in {
		t.Fatalf("expected original error to be returned unchanged")
	}
}

// Ported from upstream CLIProxyAPI commit aedc9e6a3987.
func TestBuildErrorResponseBodyWithError_TerminalAuthEnforcesContractOnJSONInput(t *testing.T) {
	terminalErr := coreauth.NewTerminalAuthError(&coreauth.Error{
		Code:       "auth_unavailable",
		Message:    "no auth available",
		HTTPStatus: http.StatusServiceUnavailable,
	}, errors.New("upstream failed"))

	// Valid JSON errText should NOT bypass terminal classification.
	jsonErrText := `{"error":{"message":"token refresh failed: revoked","type":"server_error","code":"internal_server_error"}}`
	body := BuildErrorResponseBodyWithError(http.StatusServiceUnavailable, jsonErrText, terminalErr)

	var payload struct {
		Error struct {
			Type      string `json:"type"`
			Code      string `json:"code"`
			Message   string `json:"message"`
			Retryable *bool  `json:"retryable"`
		} `json:"error"`
	}
	if errUnmarshal := json.Unmarshal(body, &payload); errUnmarshal != nil {
		t.Fatalf("unmarshal error body: %v", errUnmarshal)
	}
	if payload.Error.Type != "authentication_error" {
		t.Fatalf("type = %q, want authentication_error", payload.Error.Type)
	}
	if payload.Error.Code != "upstream_authentication_required" {
		t.Fatalf("code = %q, want upstream_authentication_required", payload.Error.Code)
	}
	if payload.Error.Retryable == nil || *payload.Error.Retryable {
		t.Fatalf("retryable = %v, want false", payload.Error.Retryable)
	}
	if !strings.Contains(payload.Error.Message, "token refresh failed: revoked") {
		t.Fatalf("message = %q, want extracted upstream message", payload.Error.Message)
	}

	// Non-terminal error with JSON errText preserves original JSON.
	normalBody := BuildErrorResponseBodyWithError(http.StatusInternalServerError, jsonErrText, errors.New("normal error"))
	if string(normalBody) != jsonErrText {
		t.Fatalf("expected untouched JSON for non-terminal error, got %s", string(normalBody))
	}
}

// Ported from upstream CLIProxyAPI commit aedc9e6a3987.
func TestEnrichAuthSelectionError_PropagatesTerminalAuth(t *testing.T) {
	terminalErr := coreauth.NewTerminalAuthError(&coreauth.Error{
		Code:       "auth_unavailable",
		Message:    "no auth available",
		HTTPStatus: http.StatusServiceUnavailable,
	}, errors.New("token refresh failed with status 401"))

	enriched := enrichAuthSelectionError(terminalErr, []string{"codex"}, "gpt-5.6-sol")
	if !coreauth.IsTerminalAuthError(enriched) {
		t.Fatalf("expected IsTerminalAuthError to remain true after enrichment, got %T: %v", enriched, enriched)
	}
}
