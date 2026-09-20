package common

import "testing"

// Ported from upstream b681a1e0f7b8: SystemReminderText wraps directives in the
// <system-reminder> envelope for demoted mid-session system/developer messages.
func TestSystemReminderText(t *testing.T) {
	text := "Please call a tool now."
	got := SystemReminderText(text)
	want := "<system-reminder>\nPlease call a tool now.\n</system-reminder>"
	if got != want {
		t.Fatalf("SystemReminderText(%q) = %q, want %q", text, got, want)
	}
}
