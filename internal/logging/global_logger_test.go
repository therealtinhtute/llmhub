package logging

import (
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestLogFormatterCodexLiveFieldsAndCredentialRedaction(t *testing.T) {
	entry := &log.Entry{
		Time:    time.Date(2026, 8, 2, 1, 2, 3, 0, time.UTC),
		Level:   log.InfoLevel,
		Message: "forwarding codex live media",
		Data: log.Fields{
			"request_id":       "req-1",
			"media_session_id": "media-session",
			"peer":             "remote",
			"call_id":          "call-1",
			"auth_index":       "auth-1",
			"connection":       "via http proxy",
			"remote_transport": "tcp",
			"proxy_scheme":     "http",
			"state":            "connected",
			"credential":       "Voice credential",
		},
	}

	out, err := (&LogFormatter{}).Format(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	line := string(out)

	for _, want := range []string{
		"media_session_id=media-session",
		"peer=remote",
		"call_id=call-1",
		"auth_index=auth-1",
		"connection=via http proxy",
		"remote_transport=tcp",
		"proxy_scheme=http",
		"state=connected",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected formatted log to contain %q, got %s", want, line)
		}
	}
	if strings.Contains(line, "credential") || strings.Contains(line, "Voice credential") {
		t.Fatalf("formatted log leaked credential field: %s", line)
	}
}
