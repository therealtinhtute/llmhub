package cliproxy

import (
	"context"
	"strings"

	"github.com/therealtinhtute/llmhub/internal/quotaalert"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

// NewQuotaAlertAuthSource adapts the core auth manager to quota-alert runtime snapshots.
func NewQuotaAlertAuthSource(manager *coreauth.Manager) quotaalert.AuthSource {
	return quotaAlertAuthSource{manager: manager}
}

type quotaAlertAuthSource struct {
	manager *coreauth.Manager
}

func (s quotaAlertAuthSource) ListQuotaAlertAuths(context.Context) ([]quotaalert.AuthSnapshot, error) {
	if s.manager == nil {
		return nil, nil
	}
	auths := s.manager.List()
	snapshots := make([]quotaalert.AuthSnapshot, 0, len(auths))
	for _, auth := range auths {
		snapshot, ok := newQuotaAlertAuthSnapshot(auth)
		if ok {
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots, nil
}

type quotaAlertAuthSnapshot struct {
	id         string
	provider   quotaalert.Provider
	label      string
	proxyURL   string
	attributes map[string]string
	metadata   map[string]any
}

func newQuotaAlertAuthSnapshot(auth *coreauth.Auth) (quotaAlertAuthSnapshot, bool) {
	if auth == nil || strings.TrimSpace(auth.ID) == "" || auth.Disabled {
		return quotaAlertAuthSnapshot{}, false
	}
	provider, ok := quotaAlertProvider(auth.Provider)
	if !ok {
		return quotaAlertAuthSnapshot{}, false
	}
	label := strings.TrimSpace(auth.Label)
	if label == "" {
		label = strings.TrimSpace(auth.FileName)
	}
	if label == "" {
		label = auth.ID
	}
	snapshot := quotaAlertAuthSnapshot{
		id:         strings.TrimSpace(auth.ID),
		provider:   provider,
		label:      label,
		proxyURL:   strings.TrimSpace(auth.ProxyURL),
		attributes: make(map[string]string, len(auth.Attributes)),
		metadata:   make(map[string]any, len(auth.Metadata)),
	}
	for key, value := range auth.Attributes {
		snapshot.attributes[key] = value
	}
	for key, value := range auth.Metadata {
		snapshot.metadata[key] = value
	}
	return snapshot, true
}

func quotaAlertProvider(provider string) (quotaalert.Provider, bool) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude":
		return quotaalert.ProviderClaude, true
	case "codex":
		return quotaalert.ProviderCodex, true
	case "gemini", "gemini-cli":
		return quotaalert.ProviderGeminiCLI, true
	case "antigravity":
		return quotaalert.ProviderAntigravity, true
	case "kimi":
		return quotaalert.ProviderKimi, true
	case "xai":
		return quotaalert.ProviderXAI, true
	case "kiro":
		return quotaalert.ProviderKiro, true
	default:
		return "", false
	}
}

func (s quotaAlertAuthSnapshot) AuthID() string                { return s.id }
func (s quotaAlertAuthSnapshot) Provider() quotaalert.Provider { return s.provider }
func (s quotaAlertAuthSnapshot) RedactedLabel() string         { return s.label }
func (s quotaAlertAuthSnapshot) ProxyURL() string              { return s.proxyURL }

func (s quotaAlertAuthSnapshot) Attribute(key string) (string, bool) {
	value, ok := s.attributes[key]
	return value, ok
}

func (s quotaAlertAuthSnapshot) Metadata(key string) (any, bool) {
	value, ok := s.metadata[key]
	return value, ok
}
