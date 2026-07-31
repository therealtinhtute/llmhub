// Package runtimecontrol defines database-backed runtime control contracts.
package runtimecontrol

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

const (
	DefaultCredentialWeight = int64(1)
	MaxCredentialWeight     = int64(1_000_000)
	DefaultLiveMaxSessions  = 32
	MaxLiveMaxSessions      = 1024
	MaxIdentityLength       = 256
	MaxProviderLength       = 64
	MaxICEServers           = 32
	MaxICEURLsPerServer     = 16
)

// ErrRevisionConflict reports an expected-revision mismatch without mutation.
var ErrRevisionConflict = errors.New("runtime control settings revision conflict")

// RoutingStrategy selects the credential scheduling policy.
type RoutingStrategy string

const (
	RoutingRoundRobin         RoutingStrategy = "round-robin"
	RoutingWeightedRoundRobin RoutingStrategy = "weighted-round-robin"
	RoutingFillFirst          RoutingStrategy = "fill-first"
)

// CredentialWeight overrides the default weight for one durable credential identity.
type CredentialWeight struct {
	AuthID   string `json:"auth_id"`
	Provider string `json:"provider"`
	Weight   int64  `json:"weight"`
}

// CredentialRoutingSettings configures credential scheduling and weight overrides.
type CredentialRoutingSettings struct {
	Strategy RoutingStrategy    `json:"strategy"`
	Weights  []CredentialWeight `json:"weights"`
}

// CloakingSettings controls provider compatibility identity behavior.
type CloakingSettings struct {
	DisableClaudeModelList bool `json:"disable_claude_model_list"`
	DisableCodex           bool `json:"disable_codex"`
}

// ICEServer configures one STUN or TURN server used by Codex Live.
// Credential material is intentionally excluded from this foundation contract.
type ICEServer struct {
	URLs     []string `json:"urls"`
	Username string   `json:"username,omitempty"`
}

// CodexLiveSettings controls the Codex Live media relay.
type CodexLiveSettings struct {
	Enabled                 bool        `json:"enabled"`
	MaxSessions             int         `json:"max_sessions"`
	DisablePrivateRemoteIPs bool        `json:"disable_private_remote_ips"`
	PublicIP                string      `json:"public_ip,omitempty"`
	UDPPortMin              int         `json:"udp_port_min"`
	UDPPortMax              int         `json:"udp_port_max"`
	ICEServers              []ICEServer `json:"ice_servers"`
}

// HomeSettings controls local participation in a Home cluster.
type HomeSettings struct {
	Enabled                 bool `json:"enabled"`
	DisableClusterDiscovery bool `json:"disable_cluster_discovery"`
}

// Settings contains all database-authoritative runtime controls.
type Settings struct {
	Revision                   int64                     `json:"revision"`
	CredentialRouting          CredentialRoutingSettings `json:"credential_routing"`
	Cloaking                   CloakingSettings          `json:"cloaking"`
	CodexLive                  CodexLiveSettings         `json:"codex_live"`
	Home                       HomeSettings              `json:"home"`
	CooldownPersistenceEnabled bool                      `json:"cooldown_persistence_enabled"`
}

// SettingsStore persists runtime controls with optimistic revision checks.
type SettingsStore interface {
	LoadRuntimeSettings(context.Context) (Settings, error)
	SaveRuntimeSettings(context.Context, int64, Settings) (Settings, error)
}

// DefaultSettings returns disabled, backward-compatible runtime controls.
func DefaultSettings() Settings {
	return Settings{
		CredentialRouting: CredentialRoutingSettings{
			Strategy: RoutingRoundRobin,
			Weights:  []CredentialWeight{},
		},
		CodexLive: CodexLiveSettings{
			MaxSessions: DefaultLiveMaxSessions,
			ICEServers:  []ICEServer{},
		},
	}
}

// WithDefaults fills omitted values without enabling a feature.
func (s Settings) WithDefaults() Settings {
	if strings.TrimSpace(string(s.CredentialRouting.Strategy)) == "" {
		s.CredentialRouting.Strategy = RoutingRoundRobin
	}
	if s.CredentialRouting.Weights == nil {
		s.CredentialRouting.Weights = []CredentialWeight{}
	}
	if s.CodexLive.MaxSessions == 0 {
		s.CodexLive.MaxSessions = DefaultLiveMaxSessions
	}
	if s.CodexLive.ICEServers == nil {
		s.CodexLive.ICEServers = []ICEServer{}
	}
	return s
}

// Normalize canonicalizes settings and validates the resulting value.
func (s Settings) Normalize() (Settings, error) {
	s = s.Clone().WithDefaults()
	strategy, err := normalizeRoutingStrategy(s.CredentialRouting.Strategy)
	if err != nil {
		return Settings{}, err
	}
	s.CredentialRouting.Strategy = strategy
	for i := range s.CredentialRouting.Weights {
		s.CredentialRouting.Weights[i].AuthID = strings.TrimSpace(s.CredentialRouting.Weights[i].AuthID)
		s.CredentialRouting.Weights[i].Provider = strings.ToLower(strings.TrimSpace(s.CredentialRouting.Weights[i].Provider))
	}
	sort.Slice(s.CredentialRouting.Weights, func(i, j int) bool {
		left, right := s.CredentialRouting.Weights[i], s.CredentialRouting.Weights[j]
		if left.Provider != right.Provider {
			return left.Provider < right.Provider
		}
		return left.AuthID < right.AuthID
	})
	s.CodexLive.PublicIP = strings.TrimSpace(s.CodexLive.PublicIP)
	for i := range s.CodexLive.ICEServers {
		server := &s.CodexLive.ICEServers[i]
		server.Username = strings.TrimSpace(server.Username)
		for j := range server.URLs {
			server.URLs[j] = strings.TrimSpace(server.URLs[j])
		}
	}
	if err = s.Validate(); err != nil {
		return Settings{}, err
	}
	return s, nil
}

// Validate verifies settings before persistence or activation.
func (s Settings) Validate() error {
	if s.Revision < 0 {
		return fmt.Errorf("runtime control settings revision must not be negative")
	}
	if _, err := normalizeRoutingStrategy(s.CredentialRouting.Strategy); err != nil {
		return err
	}
	seenWeights := make(map[string]struct{}, len(s.CredentialRouting.Weights))
	for _, weight := range s.CredentialRouting.Weights {
		if weight.AuthID == "" {
			return fmt.Errorf("credential weight auth ID is required")
		}
		if len(weight.AuthID) > MaxIdentityLength {
			return fmt.Errorf("credential weight auth ID must not exceed %d bytes", MaxIdentityLength)
		}
		if weight.Provider == "" {
			return fmt.Errorf("credential weight provider is required")
		}
		if len(weight.Provider) > MaxProviderLength {
			return fmt.Errorf("credential weight provider must not exceed %d bytes", MaxProviderLength)
		}
		if weight.Weight < 0 || weight.Weight > MaxCredentialWeight {
			return fmt.Errorf("credential weight must be between 0 and %d", MaxCredentialWeight)
		}
		key := weight.Provider + "\x00" + weight.AuthID
		if _, exists := seenWeights[key]; exists {
			return fmt.Errorf("duplicate credential weight for provider %q auth %q", weight.Provider, weight.AuthID)
		}
		seenWeights[key] = struct{}{}
	}
	return s.CodexLive.Validate()
}

// Validate verifies Codex Live relay bounds without activating the relay.
func (s CodexLiveSettings) Validate() error {
	if s.MaxSessions < 0 || s.MaxSessions > MaxLiveMaxSessions {
		return fmt.Errorf("Codex Live max sessions must be between 0 and %d", MaxLiveMaxSessions)
	}
	if s.PublicIP != "" && net.ParseIP(s.PublicIP) == nil {
		return fmt.Errorf("Codex Live public IP must be a valid IP address")
	}
	if (s.UDPPortMin == 0) != (s.UDPPortMax == 0) {
		return fmt.Errorf("Codex Live UDP port minimum and maximum must both be zero or both be set")
	}
	if s.UDPPortMin != 0 {
		if s.UDPPortMin < 1 || s.UDPPortMax > 65535 || s.UDPPortMin > s.UDPPortMax {
			return fmt.Errorf("Codex Live UDP port range is invalid")
		}
		effectiveSessions := s.MaxSessions
		if effectiveSessions == 0 {
			effectiveSessions = DefaultLiveMaxSessions
		}
		if s.UDPPortMax-s.UDPPortMin+1 < effectiveSessions*2 {
			return fmt.Errorf("Codex Live UDP port range must provide at least two ports per session")
		}
	}
	if len(s.ICEServers) > MaxICEServers {
		return fmt.Errorf("Codex Live ICE servers must not exceed %d", MaxICEServers)
	}
	for _, server := range s.ICEServers {
		if len(server.URLs) == 0 || len(server.URLs) > MaxICEURLsPerServer {
			return fmt.Errorf("each Codex Live ICE server must have between 1 and %d URLs", MaxICEURLsPerServer)
		}
		for _, rawURL := range server.URLs {
			parsed, err := url.Parse(rawURL)
			if err != nil || parsed.Scheme == "" || (parsed.Host == "" && parsed.Opaque == "") {
				return fmt.Errorf("invalid Codex Live ICE URL %q", rawURL)
			}
			switch strings.ToLower(parsed.Scheme) {
			case "stun", "stuns", "turn", "turns":
			default:
				return fmt.Errorf("unsupported Codex Live ICE URL scheme %q", parsed.Scheme)
			}
		}
	}
	return nil
}

// Clone returns a deep copy safe for independent mutation.
func (s Settings) Clone() Settings {
	copySettings := s
	copySettings.CredentialRouting.Weights = append([]CredentialWeight(nil), s.CredentialRouting.Weights...)
	copySettings.CodexLive.ICEServers = make([]ICEServer, len(s.CodexLive.ICEServers))
	for i, server := range s.CodexLive.ICEServers {
		copySettings.CodexLive.ICEServers[i] = server
		copySettings.CodexLive.ICEServers[i].URLs = append([]string(nil), server.URLs...)
	}
	return copySettings
}

func normalizeRoutingStrategy(strategy RoutingStrategy) (RoutingStrategy, error) {
	switch strings.ToLower(strings.TrimSpace(string(strategy))) {
	case "round-robin", "roundrobin", "rr":
		return RoutingRoundRobin, nil
	case "weighted-round-robin", "weightedroundrobin", "wrr":
		return RoutingWeightedRoundRobin, nil
	case "fill-first", "fillfirst", "ff":
		return RoutingFillFirst, nil
	default:
		return "", fmt.Errorf("unsupported routing strategy %q", strategy)
	}
}
