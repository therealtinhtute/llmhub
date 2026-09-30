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

func TestLogFormatterFormatsShortRequestID(t *testing.T) {
	formatter := &LogFormatter{}

	// Test case 1: UUID v7 full string truncated to trailing 8 chars in console log
	entryUUID := log.NewEntry(log.New())
	entryUUID.Time = time.Date(2026, 9, 28, 12, 0, 0, 0, time.Local)
	entryUUID.Level = log.InfoLevel
	entryUUID.Message = "handling request"
	entryUUID.Data["request_id"] = "018f3a5b-1234-7abc-def0-12345678abcd"

	formattedUUID, errFormatUUID := formatter.Format(entryUUID)
	if errFormatUUID != nil {
		t.Fatalf("Format() error = %v", errFormatUUID)
	}
	lineUUID := string(formattedUUID)
	if !strings.Contains(lineUUID, "[5678abcd]") {
		t.Fatalf("formatted line %q does not contain expected short request ID [5678abcd]", lineUUID)
	}
	if strings.Contains(lineUUID, "018f3a5b-1234-7abc-def0-12345678abcd") {
		t.Fatalf("formatted line %q should not contain full UUID", lineUUID)
	}

	// Test case 2: legacy or short 8-char request ID preserved
	entryShort := log.NewEntry(log.New())
	entryShort.Time = time.Date(2026, 9, 28, 12, 0, 0, 0, time.Local)
	entryShort.Level = log.InfoLevel
	entryShort.Message = "handling short id request"
	entryShort.Data["request_id"] = "00000042"

	formattedShort, errFormatShort := formatter.Format(entryShort)
	if errFormatShort != nil {
		t.Fatalf("Format() error = %v", errFormatShort)
	}
	lineShort := string(formattedShort)
	if !strings.Contains(lineShort, "[00000042]") {
		t.Fatalf("formatted line %q does not contain expected request ID [00000042]", lineShort)
	}

	// Test case 3: omitted request ID formats placeholder
	entryEmpty := log.NewEntry(log.New())
	entryEmpty.Time = time.Date(2026, 9, 28, 12, 0, 0, 0, time.Local)
	entryEmpty.Level = log.InfoLevel
	entryEmpty.Message = "system event"

	formattedEmpty, errFormatEmpty := formatter.Format(entryEmpty)
	if errFormatEmpty != nil {
		t.Fatalf("Format() error = %v", errFormatEmpty)
	}
	lineEmpty := string(formattedEmpty)
	if !strings.Contains(lineEmpty, "[--------]") {
		t.Fatalf("formatted line %q does not contain expected placeholder [--------]", lineEmpty)
	}
}
