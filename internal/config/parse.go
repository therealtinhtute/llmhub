package config

import (
	"fmt"
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// ParseConfigBytes parses a YAML configuration payload into Config and applies the same
// in-memory normalizations as LoadConfigOptional, without persisting any changes to disk.
func ParseConfigBytes(data []byte) (*Config, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("config payload is empty")
	}

	var cfg Config
	// Keep defaults aligned with LoadConfigOptional.
	cfg.Host = "" // Default empty: binds to all interfaces (IPv4 + IPv6)
	cfg.LoggingToFile = false
	cfg.LogsMaxTotalSizeMB = 0
	cfg.ErrorLogsMaxFiles = 10
	cfg.UsageStatisticsEnabled = false
	cfg.RedisUsageQueueRetentionSeconds = 60
	cfg.DisableCooling = false
	cfg.DisableImageGeneration = DisableImageGenerationOff
	cfg.Pprof.Enable = false
	cfg.Pprof.Addr = DefaultPprofAddr
	cfg.Discovery.Enabled = false
	cfg.Discovery.ServiceType = DefaultDiscoveryServiceType
	cfg.Discovery.Subtypes = []string{"_chat-completions", "_responses", "_messages", "_generate-content", "_interactions"}
	cfg.CredentialInFlight = DefaultCredentialInFlightConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config payload: %w", err)
	}
	if errValidate := validateTrustedProxies(cfg.TrustedProxies); errValidate != nil {
		return nil, errValidate
	}
	cfg.CredentialConcurrency = cfg.CredentialConcurrency.WithDefaults()
	cfg.CredentialInFlight = cfg.CredentialInFlight.WithDefaults()
	if errValidate := cfg.CredentialInFlight.Validate(); errValidate != nil {
		return nil, errValidate
	}
	if cfg.Discovery.ServiceType == "" {
		cfg.Discovery.ServiceType = DefaultDiscoveryServiceType
	}
	if len(cfg.Discovery.Subtypes) == 0 {
		cfg.Discovery.Subtypes = []string{"_chat-completions", "_responses", "_messages", "_generate-content", "_interactions"}
	}

	// Hash remote management key if plaintext is detected (nested), but do NOT persist.
	if cfg.RemoteManagement.SecretKey != "" && !looksLikeBcrypt(cfg.RemoteManagement.SecretKey) {
		hashed, errHash := bcrypt.GenerateFromPassword([]byte(cfg.RemoteManagement.SecretKey), bcrypt.DefaultCost)
		if errHash != nil {
			return nil, fmt.Errorf("hash remote management key: %w", errHash)
		}
		cfg.RemoteManagement.SecretKey = string(hashed)
	}

	cfg.Pprof.Addr = strings.TrimSpace(cfg.Pprof.Addr)
	if cfg.Pprof.Addr == "" {
		cfg.Pprof.Addr = DefaultPprofAddr
	}

	if cfg.LogsMaxTotalSizeMB < 0 {
		cfg.LogsMaxTotalSizeMB = 0
	}

	if cfg.ErrorLogsMaxFiles < 0 {
		cfg.ErrorLogsMaxFiles = 10
	}

	if cfg.RedisUsageQueueRetentionSeconds <= 0 {
		cfg.RedisUsageQueueRetentionSeconds = 60
	} else if cfg.RedisUsageQueueRetentionSeconds > 3600 {
		log.WithField("value", cfg.RedisUsageQueueRetentionSeconds).Warn("redis-usage-queue-retention-seconds too large; clamping to 3600")
		cfg.RedisUsageQueueRetentionSeconds = 3600
	}

	if cfg.MaxRetryCredentials < 0 {
		cfg.MaxRetryCredentials = 0
	}

	// Apply the same sanitization pipeline.
	cfg.SanitizeGeminiKeys()
	cfg.SanitizeVertexCompatKeys()
	cfg.SanitizeCodexKeys()
	cfg.SanitizeMetaKeys()
	cfg.SanitizeCodexHeaderDefaults()
	cfg.SanitizeClaudeHeaderDefaults()
	cfg.SanitizeClaudeKeys()
	cfg.SanitizeOpenAICompatibility()
	cfg.OAuthExcludedModels = NormalizeOAuthExcludedModels(cfg.OAuthExcludedModels)
	cfg.SanitizeOAuthModelAlias()
	cfg.SanitizeOAuthRequestScopedErrors()
	cfg.SanitizePayloadRules()

	cfg.NormalizeClaudeConfig()
	if errCombos := cfg.ValidateCombos(); errCombos != nil {
		return nil, errCombos
	}

	return &cfg, nil
}

// ValidateCombos validates combo definitions: unique non-empty names not colliding
// with a registered model id, at least one candidate per combo, each candidate
// parsed as provider/model, strategy in {"", "fallback", "round-robin"} (empty
// normalized to fallback), and sticky-limit < 1 normalized to 1.
func (cfg *Config) ValidateCombos() error {
	if cfg == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(cfg.Combos))
	for i := range cfg.Combos {
		combo := &cfg.Combos[i]
		combo.Name = strings.TrimSpace(combo.Name)
		if combo.Name == "" {
			return fmt.Errorf("combo at index %d: name must not be empty", i)
		}
		if _, dup := seen[combo.Name]; dup {
			return fmt.Errorf("combo %q: duplicate name", combo.Name)
		}
		seen[combo.Name] = struct{}{}
		if registry.LookupStaticModelInfo(combo.Name) != nil {
			return fmt.Errorf("combo %q: name collides with a registered model id", combo.Name)
		}
		if len(combo.Models) == 0 {
			return fmt.Errorf("combo %q: must have at least one candidate model", combo.Name)
		}
		for _, candidate := range combo.Models {
			provider, model, ok := ParseComboCandidate(candidate)
			if !ok || provider == "" || model == "" {
				return fmt.Errorf("combo %q: candidate %q must parse as provider/model", combo.Name, candidate)
			}
		}
		switch combo.Strategy {
		case "", "fallback":
			combo.Strategy = "fallback"
		case "round-robin":
		default:
			return fmt.Errorf("combo %q: strategy %q must be fallback or round-robin", combo.Name, combo.Strategy)
		}
		if combo.StickyLimit < 1 {
			combo.StickyLimit = 1
		}
	}
	// A candidate whose model part is itself a combo name would recurse at
	// resolution time; reject it here (R3 no-recursion) rather than at runtime.
	for i := range cfg.Combos {
		for _, candidate := range cfg.Combos[i].Models {
			_, model, ok := ParseComboCandidate(candidate)
			if !ok {
				continue
			}
			if _, isCombo := seen[model]; isCombo {
				return fmt.Errorf("combo %q: candidate model %q must not be another combo name", cfg.Combos[i].Name, model)
			}
		}
	}
	return nil
}

// ClaudeConfig holds Claude-specific runtime configuration.
type ClaudeConfig struct {
	// SessionDatePinDays retains the cloaked currentDate pin per session for
	// this many days, independently of continuity expiry and credential
	// switches. A nil value (unset) defaults to 7 days; an explicit 0 keeps
	// the continuity-scoped pin (1h lifetime).
	SessionDatePinDays *int `yaml:"session-date-pin-days,omitempty" json:"session-date-pin-days,omitempty"`
}

// DefaultClaudeSessionDatePinDays is the default session date pin retention in days.
const DefaultClaudeSessionDatePinDays = 7

// SessionDatePinRetention returns the configured session date pin retention duration.
func (c ClaudeConfig) SessionDatePinRetention() time.Duration {
	days := DefaultClaudeSessionDatePinDays
	if c.SessionDatePinDays != nil {
		days = *c.SessionDatePinDays
	}
	if days <= 0 {
		return 0
	}
	maxDays := int((time.Duration(math.MaxInt64) / (24 * time.Hour)))
	if days > maxDays {
		days = maxDays
	}
	return time.Duration(days) * 24 * time.Hour
}

// ClaudeSessionDatePinRetention returns the configured session date pin retention duration.
func (cfg *Config) ClaudeSessionDatePinRetention() time.Duration {
	if cfg == nil {
		return time.Duration(DefaultClaudeSessionDatePinDays) * 24 * time.Hour
	}
	return cfg.Claude.SessionDatePinRetention()
}

// NormalizeClaudeConfig applies default Claude provider configuration values.
func (cfg *Config) NormalizeClaudeConfig() {
	if cfg == nil {
		return
	}
	if cfg.Upstream.Claude.SessionDatePinDays != nil && cfg.Claude.SessionDatePinDays == nil {
		cfg.Claude.SessionDatePinDays = cfg.Upstream.Claude.SessionDatePinDays
	}
	if cfg.ClaudeSessionDatePinDays != nil && cfg.Claude.SessionDatePinDays == nil {
		cfg.Claude.SessionDatePinDays = cfg.ClaudeSessionDatePinDays
	}
	if cfg.Claude.SessionDatePinDays == nil {
		defaultDays := DefaultClaudeSessionDatePinDays
		cfg.Claude.SessionDatePinDays = &defaultDays
	}
	cfg.ClaudeSessionDatePinDays = cfg.Claude.SessionDatePinDays
}

// ParseComboCandidate splits a combo candidate on the first "/" into provider and model.
func ParseComboCandidate(candidate string) (provider, model string, ok bool) {
	trimmed := strings.TrimSpace(candidate)
	idx := strings.Index(trimmed, "/")
	if idx <= 0 || idx == len(trimmed)-1 {
		return "", "", false
	}
	return strings.TrimSpace(trimmed[:idx]), strings.TrimSpace(trimmed[idx+1:]), true
}
