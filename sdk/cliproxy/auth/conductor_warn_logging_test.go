package auth

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// Ported from upstream CLIProxyAPI conductor_warn_logging_test.go (cluster
// 9a2201c36a0a + e4a8f9891344). The upstream file also carries warn-logging
// tests from unrelated commits; only the Home diagnostic coverage is ported here.

func setupTestLoggerHook(t *testing.T) *logtest.Hook {
	_, hook := logtest.NewNullLogger()
	oldLevel := log.GetLevel()
	log.SetLevel(log.WarnLevel)

	// Deep-clone existing hooks
	savedHooks := make(log.LevelHooks)
	for lvl, hs := range log.StandardLogger().Hooks {
		savedHooks[lvl] = append([]log.Hook(nil), hs...)
	}

	log.AddHook(hook)
	t.Cleanup(func() {
		log.SetLevel(oldLevel)
		log.StandardLogger().ReplaceHooks(savedHooks)
	})
	return hook
}

type statusErrorLogTestError struct {
	message    string
	statusCode int
}

func (e statusErrorLogTestError) Error() string { return e.message }

func (e statusErrorLogTestError) StatusCode() int { return e.statusCode }

type markedDiagnosticStatusError struct {
	message    string
	diagnostic string
	statusCode int
}

func (e markedDiagnosticStatusError) Error() string { return e.message }

func (e markedDiagnosticStatusError) StatusCode() int { return e.statusCode }

func (e markedDiagnosticStatusError) LogDiagnostic() string { return e.diagnostic }

// Ported from upstream CLIProxyAPI commit 9a2201c36a0a.
func TestWarnLogUpstreamFailureIncludesStructuredStatusCodeAndSafeDiagnostic(t *testing.T) {
	hook := setupTestLoggerHook(t)
	diagnostic := `  antigravity refresh: upstream request failed with status 400 error="invalid_request" error_description="Malformed request, retry & inspect"  `

	warnLogUpstreamFailure(
		context.Background(),
		nil,
		"antigravity",
		"gemini-3.7-flash-high",
		&Auth{ID: "auth-1", Provider: "antigravity"},
		83*time.Millisecond,
		statusErrorLogTestError{message: diagnostic, statusCode: http.StatusServiceUnavailable},
	)

	for _, entry := range hook.AllEntries() {
		if entry.Level == log.WarnLevel && strings.Contains(entry.Message, "upstream execution failed") {
			if !strings.HasPrefix(entry.Message, "503 |          83ms | upstream execution failed: provider=antigravity") || strings.Contains(entry.Message, "status=503") || !strings.HasSuffix(entry.Message, "err="+strings.TrimSpace(diagnostic)) {
				t.Fatalf("unexpected Warn log content: %s", entry.Message)
			}
			return
		}
	}
	t.Fatalf("expected upstream failure Warn log, got logs: %#v", hook.AllEntries())
}

// Ported from upstream CLIProxyAPI commit e4a8f9891344.
func TestWarnLogUpstreamFailureUsesMarkedHomeDiagnostic(t *testing.T) {
	hook := setupTestLoggerHook(t)
	errRefresh := markedDiagnosticStatusError{
		message:    "credential refresh temporarily unavailable",
		diagnostic: "antigravity refresh failed: stage=transport err=EOF access_token=provider-secret",
		statusCode: http.StatusServiceUnavailable,
	}

	warnLogUpstreamFailure(
		context.Background(),
		nil,
		"antigravity",
		"gemini-3.7-flash-high",
		&Auth{ID: "auth-1", Provider: "antigravity"},
		129*time.Millisecond,
		errRefresh,
	)

	for _, entry := range hook.AllEntries() {
		if entry.Level != log.WarnLevel || !strings.Contains(entry.Message, "upstream execution failed") {
			continue
		}
		if !strings.Contains(entry.Message, "stage=transport") || !strings.Contains(entry.Message, "err=EOF") {
			t.Fatalf("Warn log lost marked Home diagnostic: %q", entry.Message)
		}
		if strings.Contains(entry.Message, "provider-secret") || strings.Contains(entry.Message, errRefresh.message) {
			t.Fatalf("Warn log exposed secret or used generic client message: %q", entry.Message)
		}
		return
	}
	t.Fatalf("expected upstream failure Warn log, got logs: %#v", hook.AllEntries())
}

// Ported from upstream CLIProxyAPI commit 9a2201c36a0a (v7.3.9 end-state version
// using a marked diagnostic, per e4a8f9891344).
func TestHomeCredentialBoundaryWarnLogUsesSafeDiagnostic(t *testing.T) {
	diagnostic := "antigravity refresh: access token expired access_token=provider-secret\nforged log line"
	upstreamErr := markedDiagnosticStatusError{
		message:    "credential refresh temporarily unavailable",
		diagnostic: diagnostic,
		statusCode: http.StatusServiceUnavailable,
	}
	hook := setupTestLoggerHook(t)
	auth := &Auth{ID: "auth-1", Provider: "antigravity"}
	executor := &requestPrepareExecutor{prepareErr: upstreamErr}
	selection := &HomeDispatchSelection{Auth: auth, Executor: executor, Provider: "antigravity"}
	_, errPrepare := NewManager(nil, nil, nil).prepareHomeRequestAuth(context.Background(), executor, selection)
	if errPrepare == nil || errPrepare.Error() != upstreamErr.message {
		t.Fatalf("operation error = %v, want generic error %q", errPrepare, upstreamErr.message)
	}

	matches := 0
	for _, entry := range hook.AllEntries() {
		if entry.Level != log.WarnLevel || !strings.HasPrefix(entry.Message, "Home credential operation failed: err=") {
			continue
		}
		matches++
		if !strings.Contains(entry.Message, "access token expired") {
			t.Fatalf("Warn log lost access-token-expired diagnostic: %q", entry.Message)
		}
		if strings.Contains(entry.Message, "provider-secret") || strings.ContainsAny(entry.Message, "\r\n") {
			t.Fatalf("Warn log included unsafe provider detail: %q", entry.Message)
		}
		if entry.Data["operation"] != "request_auth_preparation" || entry.Data["provider"] != "antigravity" || entry.Data["status"] != http.StatusServiceUnavailable {
			t.Fatalf("Warn log fields = %#v, want operation=request_auth_preparation provider=antigravity status=503", entry.Data)
		}
	}
	if matches != 1 {
		t.Fatalf("matching Warn logs = %d, want 1; logs=%#v", matches, hook.AllEntries())
	}
}
