package quotaalert

import (
	"context"
	"fmt"
	"slices"
	"sort"
)

// CollectorFactory constructs a provider collector from shared runtime dependencies.
type CollectorFactory func(CollectorDependencies) (Collector, error)

// CollectorDependencies contains shared collector infrastructure.
type CollectorDependencies struct {
	HTTPClient *CollectorHTTPClient
	Refresh    CollectorRefreshFunc
}

// CollectorRegistry maps supported providers to their collector factories.
type CollectorRegistry struct {
	factories map[Provider]CollectorFactory
}

// NewCollectorRegistry creates an empty provider collector registry.
func NewCollectorRegistry() *CollectorRegistry {
	return &CollectorRegistry{factories: make(map[Provider]CollectorFactory)}
}

// NewDefaultCollectorRegistry creates a registry with all supported provider collectors.
func NewDefaultCollectorRegistry() (*CollectorRegistry, error) {
	registry := NewCollectorRegistry()
	for _, item := range []struct {
		provider Provider
		factory  CollectorFactory
	}{
		{ProviderClaude, NewClaudeCollector},
		{ProviderCodex, NewCodexCollector},
		{ProviderGeminiCLI, NewGeminiCLICollector},
		{ProviderAntigravity, NewAntigravityCollector},
		{ProviderKimi, NewKimiCollector},
		{ProviderXAI, NewXAICollector},
		{ProviderKiro, NewKiroCollector},
	} {
		if err := registry.Register(item.provider, item.factory); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Register adds one supported provider collector factory.
func (r *CollectorRegistry) Register(provider Provider, factory CollectorFactory) error {
	if r == nil {
		return fmt.Errorf("quota collector registry is nil")
	}
	if err := provider.Validate(); err != nil {
		return err
	}
	if factory == nil {
		return fmt.Errorf("quota collector factory for %q is nil", provider)
	}
	if _, exists := r.factories[provider]; exists {
		return fmt.Errorf("quota collector for %q is already registered", provider)
	}
	r.factories[provider] = factory
	return nil
}

// Collector constructs the collector registered for provider.
func (r *CollectorRegistry) Collector(provider Provider, deps CollectorDependencies) (Collector, error) {
	if r == nil {
		return nil, fmt.Errorf("quota collector registry is nil")
	}
	if err := provider.Validate(); err != nil {
		return nil, err
	}
	factory, exists := r.factories[provider]
	if !exists {
		return nil, fmt.Errorf("quota collector for %q is not registered", provider)
	}
	collector, err := factory(deps)
	if err != nil {
		return nil, err
	}
	if collector == nil {
		return nil, fmt.Errorf("quota collector factory for %q returned nil", provider)
	}
	return collector, nil
}

// Providers returns registered provider keys in deterministic order.
func (r *CollectorRegistry) Providers() []Provider {
	if r == nil || len(r.factories) == 0 {
		return nil
	}
	providers := make([]Provider, 0, len(r.factories))
	for provider := range r.factories {
		providers = append(providers, provider)
	}
	sort.Slice(providers, func(left, right int) bool {
		return providers[left] < providers[right]
	})
	return providers
}

// ClonedAuthSnapshot is a copied, runtime-only credential view for collectors.
type ClonedAuthSnapshot struct {
	AuthIDValue        string
	ProviderValue      Provider
	RedactedLabelValue string
	ProxyURLValue      string
	Attributes         map[string]string
	MetadataValues     map[string]any
}

// CloneAuthSnapshot copies selected auth fields so collectors do not retain mutable store state.
func CloneAuthSnapshot(auth AuthSnapshot, attributeKeys, metadataKeys []string) (ClonedAuthSnapshot, error) {
	if auth == nil {
		return ClonedAuthSnapshot{}, fmt.Errorf("quota collector auth snapshot is nil")
	}
	cloned := ClonedAuthSnapshot{
		AuthIDValue:        auth.AuthID(),
		ProviderValue:      auth.Provider(),
		RedactedLabelValue: auth.RedactedLabel(),
		ProxyURLValue:      auth.ProxyURL(),
		Attributes:         make(map[string]string),
		MetadataValues:     make(map[string]any),
	}
	identity := StateIdentity{
		AuthID:   cloned.AuthIDValue,
		Provider: cloned.ProviderValue,
		Resource: "collector",
		Window:   "snapshot",
	}
	if _, err := identity.Normalize(); err != nil {
		return ClonedAuthSnapshot{}, err
	}
	if cloned.RedactedLabelValue == "" {
		return ClonedAuthSnapshot{}, fmt.Errorf("quota collector auth label is required")
	}
	for _, key := range slices.Compact(slices.Clone(attributeKeys)) {
		if value, ok := auth.Attribute(key); ok {
			cloned.Attributes[key] = value
		}
	}
	for _, key := range slices.Compact(slices.Clone(metadataKeys)) {
		if value, ok := auth.Metadata(key); ok {
			cloned.MetadataValues[key] = value
		}
	}
	return cloned, nil
}

func (s ClonedAuthSnapshot) AuthID() string        { return s.AuthIDValue }
func (s ClonedAuthSnapshot) Provider() Provider    { return s.ProviderValue }
func (s ClonedAuthSnapshot) RedactedLabel() string { return s.RedactedLabelValue }
func (s ClonedAuthSnapshot) ProxyURL() string      { return s.ProxyURLValue }

func (s ClonedAuthSnapshot) Attribute(key string) (string, bool) {
	value, ok := s.Attributes[key]
	return value, ok
}

func (s ClonedAuthSnapshot) Metadata(key string) (any, bool) {
	value, ok := s.MetadataValues[key]
	return value, ok
}

// CollectFunc adapts a function to the Collector interface for tests and small providers.
type CollectFunc func(ctx context.Context, auth AuthSnapshot) ([]Observation, error)

func (f CollectFunc) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	return f(ctx, auth)
}
