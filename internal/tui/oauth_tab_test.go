package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Ported from upstream CLIProxyAPI internal/tui/oauth_tab_test.go
// (6e819ab62257, v7.3.4 end-state). Local symbols under test:
// shouldAcceptOAuthPoll/shouldAcceptOAuthStart generation+state filtering,
// shouldFailOAuthStatusPoll, and the device-flow-aware Update paths.

func TestShouldAcceptOAuthPollFiltersStaleMessages(t *testing.T) {
	msg := oauthPollMsg{state: "state-a", generation: 1, done: true, message: "ok"}

	if shouldAcceptOAuthPoll(msg, "state-a", 2, oauthRemote) {
		t.Fatal("accepted poll with stale generation")
	}
	if shouldAcceptOAuthPoll(msg, "state-b", 1, oauthRemote) {
		t.Fatal("accepted poll with mismatched state")
	}
	if shouldAcceptOAuthPoll(msg, "state-a", 1, oauthIdle) {
		t.Fatal("accepted poll while not in remote state")
	}
	if !shouldAcceptOAuthPoll(msg, "state-a", 1, oauthRemote) {
		t.Fatal("rejected valid poll message")
	}
}

func TestShouldAcceptOAuthStartFiltersStaleMessages(t *testing.T) {
	msg := oauthStartMsg{state: "state-a", generation: 1, url: "https://example.com"}
	if shouldAcceptOAuthStart(msg, 2) {
		t.Fatal("accepted start with stale generation")
	}
	if !shouldAcceptOAuthStart(msg, 1) {
		t.Fatal("rejected valid start message")
	}
}

func TestShouldFailOAuthStatusPoll(t *testing.T) {
	if shouldFailOAuthStatusPoll(4, 5) {
		t.Fatal("failed too early on transient errors")
	}
	if !shouldFailOAuthStatusPoll(5, 5) {
		t.Fatal("did not fail after max consecutive errors")
	}
	if !shouldFailOAuthStatusPoll(1, 0) {
		t.Fatal("maxErrors<=0 should fail on first error")
	}
}

func TestOAuthTabUpdateIgnoresStalePollMsg(t *testing.T) {
	m := newOAuthTabModel(nil)
	m.state = oauthRemote
	m.authState = "state-current"
	m.pollGeneration = 2
	m.ready = true
	m.viewport = viewport.New(80, 24)
	m.viewport.SetContent(m.renderContent())

	updated, cmd := m.Update(oauthPollMsg{
		state:      "state-old",
		generation: 1,
		done:       true,
		message:    "should be ignored",
	})
	if cmd != nil {
		t.Fatal("expected no command for stale poll")
	}
	if updated.state != oauthRemote {
		t.Fatalf("state = %v, want oauthRemote", updated.state)
	}
	if updated.message != "" {
		t.Fatalf("message changed by stale poll: %q", updated.message)
	}
}

func TestOAuthTabUpdateAcceptsCurrentPollMsg(t *testing.T) {
	m := newOAuthTabModel(nil)
	m.state = oauthRemote
	m.authState = "state-current"
	m.pollGeneration = 3
	m.ready = true
	m.viewport = viewport.New(80, 24)
	m.viewport.SetContent(m.renderContent())

	updated, _ := m.Update(oauthPollMsg{
		state:      "state-current",
		generation: 3,
		done:       true,
		message:    "Authentication successful",
	})
	if updated.state != oauthSuccess {
		t.Fatalf("state = %v, want oauthSuccess", updated.state)
	}
}

func TestOAuthTabEscRemoteIncrementsGenerationAndClearsState(t *testing.T) {
	m := newOAuthTabModel(nil)
	m.state = oauthRemote
	m.authState = "state-to-cancel"
	m.authURL = "https://example.com"
	m.deviceFlow = true
	m.pollGeneration = 4
	m.ready = true
	m.viewport = viewport.New(80, 24)
	m.viewport.SetContent(m.renderContent())

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.state != oauthIdle {
		t.Fatalf("state = %v, want oauthIdle", updated.state)
	}
	if updated.pollGeneration != 5 {
		t.Fatalf("pollGeneration = %d, want 5", updated.pollGeneration)
	}
	if updated.authState != "" || updated.authURL != "" || updated.deviceFlow {
		t.Fatalf("remote fields not cleared: state=%q url=%q device=%v", updated.authState, updated.authURL, updated.deviceFlow)
	}
	// client is nil, so cancel command should be nil
	if cmd != nil {
		t.Fatal("expected nil cancel command when client is nil")
	}
}

func TestOAuthTabEscWithActiveCallbackInputCancelsRemoteSession(t *testing.T) {
	m := newOAuthTabModel(nil)
	m.state = oauthRemote
	m.authState = "state-to-cancel"
	m.authURL = "https://example.com"
	m.deviceFlow = false
	m.inputActive = true
	m.callbackInput.Focus()
	m.callbackInput.SetValue("https://callback.example/?code=abc&state=state-to-cancel")
	m.pollGeneration = 7
	m.ready = true
	m.viewport = viewport.New(80, 24)
	m.viewport.SetContent(m.renderContent())

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.state != oauthIdle {
		t.Fatalf("state = %v, want oauthIdle", updated.state)
	}
	if updated.pollGeneration != 8 {
		t.Fatalf("pollGeneration = %d, want 8", updated.pollGeneration)
	}
	if updated.inputActive {
		t.Fatal("inputActive still true after esc cancel")
	}
	if updated.callbackInput.Value() != "" {
		t.Fatalf("callback input not cleared: %q", updated.callbackInput.Value())
	}
	if updated.authState != "" || updated.authURL != "" {
		t.Fatalf("remote fields not cleared: state=%q url=%q", updated.authState, updated.authURL)
	}
	// client is nil, so cancel command should be nil
	if cmd != nil {
		t.Fatal("expected nil cancel command when client is nil")
	}
}

func TestOAuthTabStaleStartIsIgnored(t *testing.T) {
	m := newOAuthTabModel(nil)
	m.state = oauthIdle
	m.pollGeneration = 2
	m.ready = true
	m.viewport = viewport.New(80, 24)
	m.viewport.SetContent(m.renderContent())

	updated, cmd := m.Update(oauthStartMsg{
		url:        "https://example.com",
		state:      "stale-state",
		generation: 1,
	})
	if updated.state != oauthIdle {
		t.Fatalf("state = %v, want oauthIdle after stale start", updated.state)
	}
	// client is nil in this unit test; cancel is skipped but state remains idle.
	if cmd != nil {
		t.Fatal("expected nil cancel command when client is nil")
	}
	if updated.authState != "" {
		t.Fatalf("stale start should not set authState, got %q", updated.authState)
	}
}

// TestOAuthTabDeviceFlowSkipsCallbackInput covers the device-flow branch the
// upstream test file implies via m.deviceFlow: an oauthStartMsg carrying
// flow=device/user_code must not focus the callback input, and the rendered
// screen must surface the user code (upstream 6e819ab62257 + 23c16e2985bb).
func TestOAuthTabDeviceFlowSkipsCallbackInput(t *testing.T) {
	m := newOAuthTabModel(nil)
	m.state = oauthPending
	m.pollGeneration = 1
	m.width = 80
	m.ready = true
	m.viewport = viewport.New(80, 24)

	updated, cmd := m.Update(oauthStartMsg{
		url:          "https://auth.meta.com/device?user_code=ABCD-1234",
		state:        "meta-state",
		providerName: "Meta",
		userCode:     "ABCD-1234",
		deviceFlow:   true,
		expiresIn:    900,
		generation:   1,
	})
	if updated.state != oauthRemote {
		t.Fatalf("state = %v, want oauthRemote", updated.state)
	}
	if updated.inputActive {
		t.Fatal("callback input activated for device flow")
	}
	if !updated.deviceFlow || updated.userCode != "ABCD-1234" || updated.expiresIn != 900 {
		t.Fatalf("device fields not stored: device=%v code=%q expires=%d", updated.deviceFlow, updated.userCode, updated.expiresIn)
	}
	if cmd == nil {
		t.Fatal("expected status poll command for device flow")
	}

	content := updated.renderDeviceMode()
	if !strings.Contains(content, "ABCD-1234") {
		t.Fatal("device screen does not show the user code")
	}
	if strings.Contains(content, T("oauth_callback_url")) {
		t.Fatal("device screen renders callback input")
	}

	// 'c' is a no-op in device mode (no callback URL to paste).
	updated2, cmd2 := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if updated2.inputActive {
		t.Fatal("'c' activated callback input during device flow")
	}
	_ = cmd2
}
