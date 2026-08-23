package auth

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"

	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// captureWarnOutput redirects the standard logger into a buffer for the
// duration of the test.
func captureWarnOutput(t *testing.T) *bytes.Buffer {
	var buf bytes.Buffer
	previousOut := log.StandardLogger().Out
	previousLevel := log.GetLevel()
	log.SetOutput(&buf)
	log.SetLevel(log.WarnLevel)
	t.Cleanup(func() {
		log.SetOutput(previousOut)
		log.SetLevel(previousLevel)
	})
	return &buf
}

func TestWarnLogUpstreamFailureIdentifiesCredentialWithoutLeakingKey(t *testing.T) {
	buf := captureWarnOutput(t)

	auth := &Auth{
		ID:         "auth-1",
		Provider:   "vertex",
		Attributes: map[string]string{"api_key": "super-secret-key-value"},
	}
	warnLogUpstreamFailure(context.Background(), nil, "vertex", "gemini-3.7-flash", auth, 0, errors.New("upstream exploded"))

	output := buf.String()
	if !strings.Contains(output, "upstream execution failed") {
		t.Fatalf("expected a warn-level upstream failure diagnostic, got: %q", output)
	}
	if !strings.Contains(output, "vertex") || !strings.Contains(output, "gemini-3.7-flash") {
		t.Fatalf("diagnostic must identify provider and model: %q", output)
	}
	if strings.Contains(output, "super-secret-key-value") {
		t.Fatalf("diagnostic leaked the api key: %q", output)
	}
}

func TestWarnLogUpstreamFailureSkipsCancellation(t *testing.T) {
	buf := captureWarnOutput(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	warnLogUpstreamFailure(ctx, nil, "codex", "gpt-5.2", &Auth{ID: "a"}, 0, errors.New("boom"))
	warnLogUpstreamFailure(context.Background(), nil, "codex", "gpt-5.2", &Auth{ID: "a"}, 0, context.Canceled)

	if strings.Contains(buf.String(), "upstream execution failed") {
		t.Fatalf("cancellations must not produce diagnostics: %q", buf.String())
	}
}

func TestMarkResultEmitsFailureDiagnostic(t *testing.T) {
	buf := captureWarnOutput(t)

	m := NewManager(nil, nil, nil)
	auth := &Auth{ID: "auth-diag", Provider: "claude", Status: StatusActive}
	m.mu.Lock()
	m.auths[auth.ID] = auth
	m.mu.Unlock()

	m.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: "claude",
		Model:    "claude-sonnet-5",
		Success:  false,
		Error:    &Error{Message: "429 from upstream"},
		Options:  cliproxyexecutor.Options{},
	})

	if !strings.Contains(buf.String(), "upstream execution failed") || !strings.Contains(buf.String(), "claude") {
		t.Fatalf("expected MarkResult failure to emit credential diagnostic, got: %q", buf.String())
	}
}
