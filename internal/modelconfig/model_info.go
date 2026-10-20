// Package modelconfig resolves configured model entries to capability
// snapshots used by the executor auth layer.
package modelconfig

import (
	"strings"

	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/thinking"
)

// ResolveModelInfo returns a private capability snapshot for a configured model.
// Static capabilities come from the suffix-free upstream name, while explicit
// configuration takes precedence.
//
// Ported from upstream CLIProxyAPI (internal/modelconfig/model_info.go).
func ResolveModelInfo(name, modelType string, support *registry.ThinkingSupport) *registry.ModelInfo {
	trimmedName := strings.TrimSpace(name)
	baseName := strings.TrimSpace(thinking.ParseSuffix(trimmedName).ModelName)
	info := registry.LookupModelInfo(baseName, modelType)
	if info == nil {
		info = registry.LookupStaticModelInfo(baseName)
	}
	if info == nil {
		info = &registry.ModelInfo{}
	}
	info.ID = trimmedName
	info.Type = strings.TrimSpace(modelType)
	if support != nil {
		normalized := NormalizeThinkingSupport(support)
		// Configured levels replace the catalog entry, but the way a known model
		// turns thinking off is an upstream API fact, not a level choice.
		if normalized.DisableType == "" && info.Thinking != nil {
			normalized.DisableType = info.Thinking.DisableType
		}
		info.Thinking = normalized
	}
	info.UserDefined = false
	return info
}

// NormalizeThinkingSupport clones and normalizes configured reasoning levels.
func NormalizeThinkingSupport(raw *registry.ThinkingSupport) *registry.ThinkingSupport {
	if raw == nil {
		return nil
	}
	normalized := *raw
	normalized.Levels = nil
	seen := make(map[string]struct{}, len(raw.Levels))
	for _, value := range raw.Levels {
		level := strings.ToLower(strings.TrimSpace(value))
		if level == "" {
			continue
		}
		switch level {
		case "none":
			normalized.ZeroAllowed = true
		case "auto":
			normalized.DynamicAllowed = true
		}
		if _, exists := seen[level]; exists {
			continue
		}
		seen[level] = struct{}{}
		normalized.Levels = append(normalized.Levels, level)
	}
	return &normalized
}
