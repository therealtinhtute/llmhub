package runtimecontrol

import (
	"strings"
	"testing"
)

func TestDefaultSettingsAreDisabledAndValid(t *testing.T) {
	settings := DefaultSettings()
	if err := settings.Validate(); err != nil {
		t.Fatalf("DefaultSettings().Validate() error = %v", err)
	}
	if settings.Revision != 0 || settings.Home.Enabled || settings.CodexLive.Enabled || settings.CooldownPersistenceEnabled {
		t.Fatalf("default settings unexpectedly enable runtime behavior: %#v", settings)
	}
	if settings.CredentialRouting.Strategy != RoutingRoundRobin || settings.CodexLive.MaxSessions != DefaultLiveMaxSessions {
		t.Fatalf("default settings = %#v", settings)
	}
	if settings.CredentialRouting.Weights == nil || settings.CodexLive.ICEServers == nil {
		t.Fatal("default collection fields must be non-nil")
	}
}

func TestSettingsNormalizeCanonicalizesAndSorts(t *testing.T) {
	settings := Settings{
		CredentialRouting: CredentialRoutingSettings{
			Strategy: "WRR",
			Weights: []CredentialWeight{
				{AuthID: " beta ", Provider: " CODEX ", Weight: 0},
				{AuthID: "alpha", Provider: "claude", Weight: 2},
			},
		},
		CodexLive: CodexLiveSettings{
			PublicIP: " 127.0.0.1 ",
			ICEServers: []ICEServer{{
				URLs:     []string{" stun:stun.example.com:3478 "},
				Username: " relay ",
			}},
		},
	}

	normalized, err := settings.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized.CredentialRouting.Strategy != RoutingWeightedRoundRobin {
		t.Fatalf("strategy = %q", normalized.CredentialRouting.Strategy)
	}
	if got := normalized.CredentialRouting.Weights[0]; got.Provider != "claude" || got.AuthID != "alpha" {
		t.Fatalf("first weight = %#v", got)
	}
	if got := normalized.CredentialRouting.Weights[1]; got.Provider != "codex" || got.AuthID != "beta" || got.Weight != 0 {
		t.Fatalf("second weight = %#v", got)
	}
	if normalized.CodexLive.MaxSessions != DefaultLiveMaxSessions || normalized.CodexLive.PublicIP != "127.0.0.1" {
		t.Fatalf("Codex Live settings = %#v", normalized.CodexLive)
	}
	if normalized.CodexLive.ICEServers[0].URLs[0] != "stun:stun.example.com:3478" || normalized.CodexLive.ICEServers[0].Username != "relay" {
		t.Fatalf("ICE server = %#v", normalized.CodexLive.ICEServers[0])
	}
}

func TestSettingsValidateRejectsInvalidControls(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Settings)
		match  string
	}{
		{name: "negative revision", mutate: func(s *Settings) { s.Revision = -1 }, match: "revision"},
		{name: "routing strategy", mutate: func(s *Settings) { s.CredentialRouting.Strategy = "random" }, match: "routing strategy"},
		{name: "missing auth", mutate: func(s *Settings) { s.CredentialRouting.Weights = []CredentialWeight{{Provider: "codex", Weight: 1}} }, match: "auth ID"},
		{name: "missing provider", mutate: func(s *Settings) { s.CredentialRouting.Weights = []CredentialWeight{{AuthID: "auth", Weight: 1}} }, match: "provider"},
		{name: "negative weight", mutate: func(s *Settings) {
			s.CredentialRouting.Weights = []CredentialWeight{{AuthID: "auth", Provider: "codex", Weight: -1}}
		}, match: "credential weight"},
		{name: "large weight", mutate: func(s *Settings) {
			s.CredentialRouting.Weights = []CredentialWeight{{AuthID: "auth", Provider: "codex", Weight: MaxCredentialWeight + 1}}
		}, match: "credential weight"},
		{name: "duplicate weight", mutate: func(s *Settings) {
			s.CredentialRouting.Weights = []CredentialWeight{{AuthID: "auth", Provider: "codex", Weight: 1}, {AuthID: "auth", Provider: "codex", Weight: 2}}
		}, match: "duplicate"},
		{name: "large live sessions", mutate: func(s *Settings) { s.CodexLive.MaxSessions = MaxLiveMaxSessions + 1 }, match: "max sessions"},
		{name: "public IP", mutate: func(s *Settings) { s.CodexLive.PublicIP = "example.com" }, match: "public IP"},
		{name: "half port range", mutate: func(s *Settings) { s.CodexLive.UDPPortMin = 5000 }, match: "both be zero"},
		{name: "reversed port range", mutate: func(s *Settings) { s.CodexLive.UDPPortMin, s.CodexLive.UDPPortMax = 5001, 5000 }, match: "range is invalid"},
		{name: "small port range", mutate: func(s *Settings) {
			s.CodexLive.MaxSessions, s.CodexLive.UDPPortMin, s.CodexLive.UDPPortMax = 2, 5000, 5002
		}, match: "two ports"},
		{name: "missing ICE URL", mutate: func(s *Settings) { s.CodexLive.ICEServers = []ICEServer{{}} }, match: "ICE server"},
		{name: "bad ICE scheme", mutate: func(s *Settings) { s.CodexLive.ICEServers = []ICEServer{{URLs: []string{"https://example.com"}}} }, match: "scheme"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := DefaultSettings()
			test.mutate(&settings)
			err := settings.Validate()
			if err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.match)
			}
		})
	}
}

func TestSettingsCloneDoesNotShareCollections(t *testing.T) {
	settings := DefaultSettings()
	settings.CredentialRouting.Weights = []CredentialWeight{{AuthID: "auth", Provider: "codex", Weight: 1}}
	settings.CodexLive.ICEServers = []ICEServer{{URLs: []string{"stun:one.example"}}}

	clone := settings.Clone()
	clone.CredentialRouting.Weights[0].AuthID = "other"
	clone.CodexLive.ICEServers[0].URLs[0] = "stun:two.example"
	if settings.CredentialRouting.Weights[0].AuthID != "auth" {
		t.Fatal("credential weights share backing storage")
	}
	if settings.CodexLive.ICEServers[0].URLs[0] != "stun:one.example" {
		t.Fatal("ICE server URLs share backing storage")
	}
}
