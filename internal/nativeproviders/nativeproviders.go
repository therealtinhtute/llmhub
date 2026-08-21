package nativeproviders

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/therealtinhtute/llmhub/internal/config"
)

const (
	ProviderOpenRouter = "openrouter"
	ProviderOpenCode   = "opencode"

	OpenRouterBaseURL = "https://openrouter.ai/api/v1"
	OpenCodeBaseURL   = "https://opencode.ai/zen/v1"
)

// Model describes a native provider model and its optional client-facing alias.
type Model struct {
	Name        string `json:"name"`
	Alias       string `json:"alias,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// OpenRouterResource is the structured, provider-owned OpenRouter configuration.
type OpenRouterResource struct {
	ID      string  `json:"id"`
	Enabled bool    `json:"enabled"`
	APIKey  string  `json:"api_key,omitempty"`
	Models  []Model `json:"models,omitempty"`
}

// OpenCodeResource is the structured, provider-owned OpenCode configuration.
type OpenCodeResource struct {
	ID      string  `json:"id"`
	Enabled bool    `json:"enabled"`
	APIKey  string  `json:"api_key,omitempty"`
	Models  []Model `json:"models,omitempty"`
}

// ResourceRecord is the persistence-neutral native provider record returned by stores.
type ResourceRecord struct {
	Provider  string
	ID        string
	Payload   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store persists native provider resources independently from the YAML config.
type Store interface {
	ListNativeProviderResources(context.Context, string) ([]ResourceRecord, error)
	SaveNativeProviderResource(context.Context, string, string, []byte) error
	DeleteNativeProviderResource(context.Context, string, string) error
}

// NormalizeProvider validates and canonicalizes the native provider identifier.
func NormalizeProvider(provider string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	switch normalized {
	case ProviderOpenRouter, ProviderOpenCode:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported native provider %q", provider)
	}
}

// NewID returns a stable-enough opaque resource id for a new native provider record.
func NewID(provider string) (string, error) {
	normalized, err := NormalizeProvider(provider)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 8)
	if _, err = rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate native provider id: %w", err)
	}
	return normalized + "-" + hex.EncodeToString(buf), nil
}

// ValidateResource applies provider-specific auth requirements and normalizes models.
func ValidateResource(provider string, resource any) error {
	normalized, err := NormalizeProvider(provider)
	if err != nil {
		return err
	}
	switch normalized {
	case ProviderOpenRouter:
		value, ok := resource.(*OpenRouterResource)
		if !ok || value == nil {
			return fmt.Errorf("invalid OpenRouter resource")
		}
		value.ID = strings.TrimSpace(value.ID)
		value.APIKey = strings.TrimSpace(value.APIKey)
		if value.ID == "" {
			return fmt.Errorf("id is required")
		}
		if value.APIKey == "" {
			return fmt.Errorf("api key is required for OpenRouter")
		}
		value.Models = NormalizeModels(value.Models)
	case ProviderOpenCode:
		value, ok := resource.(*OpenCodeResource)
		if !ok || value == nil {
			return fmt.Errorf("invalid OpenCode resource")
		}
		value.ID = strings.TrimSpace(value.ID)
		value.APIKey = strings.TrimSpace(value.APIKey)
		if value.ID == "" {
			return fmt.Errorf("id is required")
		}
		value.Models = NormalizeModels(value.Models)
	}
	return nil
}

// NormalizeModels trims, de-duplicates, and fills aliases used by runtime routing.
func NormalizeModels(models []Model) []Model {
	if len(models) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(models))
	out := make([]Model, 0, len(models))
	for _, model := range models {
		model.Name = strings.TrimSpace(model.Name)
		model.Alias = strings.TrimSpace(model.Alias)
		model.DisplayName = strings.TrimSpace(model.DisplayName)
		if model.Name == "" {
			continue
		}
		if model.Alias == "" {
			model.Alias = model.Name
		}
		key := strings.ToLower(model.Alias)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if model.DisplayName == "" {
			model.DisplayName = model.Name
		}
		out = append(out, model)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DecodeResource decodes a stored native record into its provider-specific type.
func DecodeResource(provider string, payload []byte) (any, error) {
	normalized, err := NormalizeProvider(provider)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("native provider %s has empty payload", normalized)
	}
	switch normalized {
	case ProviderOpenRouter:
		var resource OpenRouterResource
		if err := json.Unmarshal(payload, &resource); err != nil {
			return nil, fmt.Errorf("decode OpenRouter resource: %w", err)
		}
		return &resource, nil
	case ProviderOpenCode:
		var resource OpenCodeResource
		if err := json.Unmarshal(payload, &resource); err != nil {
			return nil, fmt.Errorf("decode OpenCode resource: %w", err)
		}
		return &resource, nil
	default:
		return nil, fmt.Errorf("unsupported native provider %q", provider)
	}
}

// EncodeResource validates and serializes a provider-specific native resource.
func EncodeResource(provider string, resource any) ([]byte, error) {
	if err := ValidateResource(provider, resource); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("encode native provider resource: %w", err)
	}
	return payload, nil
}

// PublicResource is safe to return from management APIs; the key itself is never exposed.
type PublicResource struct {
	ID            string  `json:"id"`
	Enabled       bool    `json:"enabled"`
	APIKeyPresent bool    `json:"api_key_present"`
	APIKeyPreview string  `json:"api_key_preview,omitempty"`
	Models        []Model `json:"models,omitempty"`
}

// ToPublicResource converts a stored resource to a redacted management response.
func ToPublicResource(provider string, resource any) (PublicResource, error) {
	if err := ValidateResourceForRead(provider, resource); err != nil {
		return PublicResource{}, err
	}
	switch normalized, _ := NormalizeProvider(provider); normalized {
	case ProviderOpenRouter:
		value := resource.(*OpenRouterResource)
		return PublicResource{
			ID:            value.ID,
			Enabled:       value.Enabled,
			APIKeyPresent: value.APIKey != "",
			APIKeyPreview: MaskAPIKey(value.APIKey),
			Models:        append([]Model(nil), value.Models...),
		}, nil
	case ProviderOpenCode:
		value := resource.(*OpenCodeResource)
		return PublicResource{
			ID:            value.ID,
			Enabled:       value.Enabled,
			APIKeyPresent: value.APIKey != "",
			APIKeyPreview: MaskAPIKey(value.APIKey),
			Models:        append([]Model(nil), value.Models...),
		}, nil
	default:
		return PublicResource{}, fmt.Errorf("unsupported native provider %q", provider)
	}
}

func ValidateResourceForRead(provider string, resource any) error {
	normalized, err := NormalizeProvider(provider)
	if err != nil {
		return err
	}
	switch normalized {
	case ProviderOpenRouter:
		value, ok := resource.(*OpenRouterResource)
		if !ok || value == nil || strings.TrimSpace(value.ID) == "" {
			return fmt.Errorf("invalid OpenRouter resource")
		}
	case ProviderOpenCode:
		value, ok := resource.(*OpenCodeResource)
		if !ok || value == nil || strings.TrimSpace(value.ID) == "" {
			return fmt.Errorf("invalid OpenCode resource")
		}
	}
	return nil
}

// MaskAPIKey returns a short, non-reversible preview for UI status displays.
func MaskAPIKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + "..." + value[len(value)-4:]
}

// ProjectedName is the internal OpenAI-compatibility name used by runtime hydration.
func ProjectedName(provider, id string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + ":" + strings.TrimSpace(id)
}

// IsProjectedName reports whether a generic OpenAI config entry belongs to a native resource.
func IsProjectedName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(name, ProviderOpenRouter+":") || strings.HasPrefix(name, ProviderOpenCode+":")
}

// HydrateConfig replaces only native projections in the in-memory runtime config.
// Native records remain outside YAML persistence and generic management responses.
func HydrateConfig(ctx context.Context, cfg *config.Config, store Store) error {
	if cfg == nil || store == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	kept := make([]config.OpenAICompatibility, 0, len(cfg.OpenAICompatibility))
	for _, entry := range cfg.OpenAICompatibility {
		if !IsProjectedName(entry.Name) {
			kept = append(kept, entry)
		}
	}

	for _, provider := range []string{ProviderOpenRouter, ProviderOpenCode} {
		records, err := store.ListNativeProviderResources(ctx, provider)
		if err != nil {
			return fmt.Errorf("list %s resources: %w", provider, err)
		}
		for _, record := range records {
			resource, err := DecodeResource(provider, record.Payload)
			if err != nil {
				return err
			}
			projected, err := projectConfigEntry(provider, resource)
			if err != nil {
				return err
			}
			kept = append(kept, projected)
		}
	}
	cfg.OpenAICompatibility = kept
	return nil
}

// StripProjectedEntries returns a copy suitable for YAML persistence.
func StripProjectedEntries(entries []config.OpenAICompatibility) []config.OpenAICompatibility {
	if len(entries) == 0 {
		return nil
	}
	out := make([]config.OpenAICompatibility, 0, len(entries))
	for _, entry := range entries {
		if !IsProjectedName(entry.Name) {
			out = append(out, entry)
		}
	}
	return out
}

func projectConfigEntry(provider string, resource any) (config.OpenAICompatibility, error) {
	switch normalized, _ := NormalizeProvider(provider); normalized {
	case ProviderOpenRouter:
		value, ok := resource.(*OpenRouterResource)
		if !ok || value == nil {
			return config.OpenAICompatibility{}, fmt.Errorf("invalid OpenRouter resource")
		}
		return config.OpenAICompatibility{
			Name:           ProjectedName(normalized, value.ID),
			Disabled:       !value.Enabled,
			BaseURL:        OpenRouterBaseURL,
			APIKeyEntries:  []config.OpenAICompatibilityAPIKey{{APIKey: value.APIKey}},
			Models:         toConfigModels(value.Models),
			Headers:        map[string]string{"HTTP-Referer": "https://llmhub.local", "X-Title": "LLMHub"},
			Passthrough:    true,
		}, nil
	case ProviderOpenCode:
		value, ok := resource.(*OpenCodeResource)
		if !ok || value == nil {
			return config.OpenAICompatibility{}, fmt.Errorf("invalid OpenCode resource")
		}
		return config.OpenAICompatibility{
			Name:           ProjectedName(normalized, value.ID),
			Disabled:       !value.Enabled,
			BaseURL:        OpenCodeBaseURL,
			APIKeyEntries:  apiKeyEntries(value.APIKey),
			Models:         toConfigModels(value.Models),
			Headers:        map[string]string{"x-opencode-client": "desktop"},
			Passthrough:    true,
		}, nil
	default:
		return config.OpenAICompatibility{}, fmt.Errorf("unsupported native provider %q", provider)
	}
}

func apiKeyEntries(apiKey string) []config.OpenAICompatibilityAPIKey {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	return []config.OpenAICompatibilityAPIKey{{APIKey: strings.TrimSpace(apiKey)}}
}

func toConfigModels(models []Model) []config.OpenAICompatibilityModel {
	normalized := NormalizeModels(models)
	if len(normalized) == 0 {
		return nil
	}
	out := make([]config.OpenAICompatibilityModel, 0, len(normalized))
	for _, model := range normalized {
		out = append(out, config.OpenAICompatibilityModel{
			Name:        model.Name,
			Alias:       model.Alias,
			DisplayName: model.DisplayName,
		})
	}
	return out
}

// FetchRemoteModels loads an OpenAI-compatible model list without exposing credentials to clients.
// If the upstream request fails, it returns the embedded catalog with source="fallback".
func FetchRemoteModels(ctx context.Context, provider, apiKey string) ([]Model, string, error) {
	normalized, err := NormalizeProvider(provider)
	if err != nil {
		return nil, "fallback", err
	}
	apiKey = strings.TrimSpace(apiKey)
	if normalized == ProviderOpenRouter && apiKey == "" {
		return FallbackModels(normalized), "fallback", fmt.Errorf("api key is required for OpenRouter")
	}
	endpoint := OpenRouterBaseURL + "/models"
	headers := map[string]string{
		"Accept": "application/json",
	}
	if normalized == ProviderOpenCode {
		endpoint = OpenCodeBaseURL + "/models"
		headers["x-opencode-client"] = "desktop"
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}

	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return FallbackModels(normalized), "fallback", fmt.Errorf("build model request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return FallbackModels(normalized), "fallback", fmt.Errorf("fetch %s models: %w", normalized, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return FallbackModels(normalized), "fallback", fmt.Errorf("fetch %s models: upstream returned HTTP %d", normalized, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return FallbackModels(normalized), "fallback", fmt.Errorf("read %s models: %w", normalized, err)
	}
	var payload struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return FallbackModels(normalized), "fallback", fmt.Errorf("decode %s models: %w", normalized, err)
	}
	models := make([]Model, 0, len(payload.Data))
	for _, item := range payload.Data {
		name := strings.TrimSpace(item.ID)
		if name == "" {
			name = strings.TrimSpace(item.Name)
		}
		if name == "" {
			continue
		}
		models = append(models, Model{Name: name, Alias: name, DisplayName: strings.TrimSpace(item.DisplayName)})
	}
	models = NormalizeModels(models)
	if len(models) == 0 {
		return FallbackModels(normalized), "fallback", fmt.Errorf("fetch %s models: upstream returned no models", normalized)
	}
	return models, "remote", nil
}

// FallbackModels is intentionally small and stable; saved remote models remain authoritative after selection.
func FallbackModels(provider string) []Model {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderOpenRouter:
		return []Model{
			{Name: "deepseek/deepseek-r1:free", Alias: "deepseek/deepseek-r1:free", DisplayName: "DeepSeek R1 (free)"},
			{Name: "google/gemini-2.0-flash-exp:free", Alias: "google/gemini-2.0-flash-exp:free", DisplayName: "Gemini 2.0 Flash Experimental (free)"},
			{Name: "meta-llama/llama-3.3-8b-instruct:free", Alias: "meta-llama/llama-3.3-8b-instruct:free", DisplayName: "Llama 3.3 8B (free)"},
		}
	case ProviderOpenCode:
		return []Model{
			{Name: "opencode/gpt-5", Alias: "opencode/gpt-5", DisplayName: "OpenCode GPT-5"},
			{Name: "opencode/claude-sonnet-4", Alias: "opencode/claude-sonnet-4", DisplayName: "OpenCode Claude Sonnet 4"},
			{Name: "opencode/big-pickle", Alias: "opencode/big-pickle", DisplayName: "OpenCode Big Pickle"},
		}
	default:
		return nil
	}
}

// ParseBoolField keeps API request parsing predictable for clients that send JSON numbers/strings.
func ParseBoolField(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}
