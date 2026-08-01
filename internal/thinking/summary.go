package thinking

import (
	"strings"

	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type SummaryMode int

const (
	SummaryUnspecified SummaryMode = iota
	SummaryDisabled
	SummaryEnabled
)

type SummaryConfig struct {
	Mode   SummaryMode
	Detail string
}

func ExtractSummaryConfig(body []byte, format string) SummaryConfig {
	format = strings.ToLower(strings.TrimSpace(format))
	if !summaryFormatSupported(format) || len(body) == 0 || !gjson.ValidBytes(body) {
		return SummaryConfig{}
	}

	switch format {
	case "openai":
		if config, ok := extractOpenAIExplicitSummaryConfig(body); ok {
			return config
		}
		if effort := gjson.GetBytes(body, "reasoning_effort"); effort.Type == gjson.String {
			value := strings.ToLower(strings.TrimSpace(effort.String()))
			if value == "" {
				return SummaryConfig{}
			}
			if value == "none" {
				return SummaryConfig{Mode: SummaryDisabled}
			}
			return SummaryConfig{Mode: SummaryEnabled, Detail: "auto"}
		}
	case "openai-response", "codex":
		if config, ok := responsesSummaryConfig(body, "reasoning.summary"); ok {
			return config
		}
		if config, ok := responsesSummaryConfig(body, "reasoning.generate_summary"); ok {
			return config
		}
	case "claude":
		if !claudeThinkingAcceptsDisplay(body) {
			return SummaryConfig{}
		}
		if config, ok := claudeSummaryConfig(body, "thinking.display"); ok {
			return config
		}
	case "gemini":
		if config, ok := firstSummaryBoolConfig(body, []string{
			"generationConfig.thinkingConfig.includeThoughts",
			"generationConfig.thinkingConfig.include_thoughts",
			"generation_config.thinking_config.include_thoughts",
			"generation_config.thinking_config.includeThoughts",
		}); ok {
			return config
		}
	case "gemini-cli", "antigravity":
		if config, ok := firstSummaryBoolConfig(body, []string{
			"request.generationConfig.thinkingConfig.includeThoughts",
			"request.generationConfig.thinkingConfig.include_thoughts",
			"request.generationConfig.thinking_config.includeThoughts",
			"request.generationConfig.thinking_config.include_thoughts",
		}); ok {
			return config
		}
	}

	return SummaryConfig{}
}

func ExtractExplicitSummaryConfig(body []byte, format string) SummaryConfig {
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "openai" {
		return ExtractSummaryConfig(body, format)
	}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return SummaryConfig{}
	}
	config, _ := extractOpenAIExplicitSummaryConfig(body)
	return config
}

func ApplySummaryConfig(body []byte, format string, config SummaryConfig) []byte {
	return ApplySummaryConfigForModel(body, format, "", config)
}

func ApplySummaryConfigForModel(body []byte, format, model string, config SummaryConfig) []byte {
	format = strings.ToLower(strings.TrimSpace(format))
	if config.Mode == SummaryUnspecified || !summaryFormatSupported(format) || len(body) == 0 || !gjson.ValidBytes(body) {
		return body
	}

	enabled := config.Mode == SummaryEnabled
	switch format {
	case "openai":
		body = applyOpenAIChatSummaryConfig(body, enabled)
	case "claude":
		if enabled && !gjson.GetBytes(body, "thinking.type").Exists() {
			body = enableClaudeThinkingForSummary(body, model)
		}
		if !claudeThinkingAcceptsDisplay(body) {
			return body
		}
		value := "omitted"
		if enabled {
			value = "summarized"
		}
		body, _ = sjson.SetBytes(body, "thinking.display", value)
	case "gemini":
		body, _ = sjson.SetBytes(body, "generationConfig.thinkingConfig.includeThoughts", enabled)
		for _, path := range []string{
			"generationConfig.thinkingConfig.include_thoughts",
			"generation_config.thinking_config.include_thoughts",
			"generation_config.thinking_config.includeThoughts",
		} {
			body, _ = sjson.DeleteBytes(body, path)
		}
	case "gemini-cli", "antigravity":
		body, _ = sjson.SetBytes(body, "request.generationConfig.thinkingConfig.includeThoughts", enabled)
		for _, path := range []string{
			"request.generationConfig.thinkingConfig.include_thoughts",
			"request.generationConfig.thinking_config.include_thoughts",
			"request.generationConfig.thinking_config.includeThoughts",
		} {
			body, _ = sjson.DeleteBytes(body, path)
		}
	case "openai-response", "codex":
		if enabled {
			body, _ = sjson.SetBytes(body, "reasoning.summary", normalizedSummaryDetail(config.Detail))
			body, _ = sjson.DeleteBytes(body, "reasoning.generate_summary")
			break
		}
		body, _ = sjson.DeleteBytes(body, "reasoning.summary")
		body, _ = sjson.DeleteBytes(body, "reasoning.generate_summary")
		if reasoning := gjson.GetBytes(body, "reasoning"); reasoning.IsObject() && len(reasoning.Map()) == 0 {
			body, _ = sjson.DeleteBytes(body, "reasoning")
		}
	}
	return body
}

func summaryFormatSupported(format string) bool {
	switch format {
	case "openai", "openai-response", "codex", "claude", "gemini", "gemini-cli", "antigravity":
		return true
	default:
		return false
	}
}

func claudeThinkingAcceptsDisplay(body []byte) bool {
	switch strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "thinking.type").String())) {
	case "adaptive":
		return true
	case "enabled":
		budget := gjson.GetBytes(body, "thinking.budget_tokens")
		if budget.Type != gjson.Number {
			return true
		}
		value := budget.Int()
		return value == -1 || value > 0
	default:
		return false
	}
}

func applyOpenAIChatSummaryConfig(body []byte, enabled bool) []byte {
	if gjson.GetBytes(body, "reasoning.exclude").IsBool() {
		body, _ = sjson.SetBytes(body, "reasoning.exclude", !enabled)
	}
	if gjson.GetBytes(body, "include_reasoning").IsBool() {
		body, _ = sjson.SetBytes(body, "include_reasoning", enabled)
	}
	return body
}

func extractOpenAIExplicitSummaryConfig(body []byte) (SummaryConfig, bool) {
	for _, path := range []string{
		"extra_body.google.thinking_config.include_thoughts",
		"extra_body.google.thinking_config.includeThoughts",
		"extra_body.google.thinkingConfig.include_thoughts",
		"extra_body.google.thinkingConfig.includeThoughts",
		"extra_body.extra_body.google.thinking_config.include_thoughts",
		"extra_body.extra_body.google.thinking_config.includeThoughts",
		"google.thinking_config.include_thoughts",
		"google.thinking_config.includeThoughts",
		"thinking.includeThoughts",
		"thinking.include_thoughts",
		"reasoning.includeThoughts",
		"reasoning.include_thoughts",
		"generationConfig.thinkingConfig.includeThoughts",
		"generationConfig.thinkingConfig.include_thoughts",
		"generation_config.thinking_config.include_thoughts",
		"generation_config.thinking_config.includeThoughts",
	} {
		if config, ok := summaryBoolConfig(body, path); ok {
			return config, true
		}
	}

	for _, path := range []string{"reasoning.summary", "reasoning.generate_summary"} {
		if config, ok := responsesSummaryConfig(body, path); ok {
			return config, true
		}
	}
	if exclude := gjson.GetBytes(body, "reasoning.exclude"); exclude.IsBool() {
		if exclude.Bool() {
			return SummaryConfig{Mode: SummaryDisabled}, true
		}
		return SummaryConfig{Mode: SummaryEnabled, Detail: "auto"}, true
	}
	if include := gjson.GetBytes(body, "include_reasoning"); include.IsBool() {
		if include.Bool() {
			return SummaryConfig{Mode: SummaryEnabled, Detail: "auto"}, true
		}
		return SummaryConfig{Mode: SummaryDisabled}, true
	}
	if enabled := gjson.GetBytes(body, "reasoning.enabled"); enabled.IsBool() {
		if enabled.Bool() {
			return SummaryConfig{Mode: SummaryEnabled, Detail: "auto"}, true
		}
		return SummaryConfig{Mode: SummaryDisabled}, true
	}
	return SummaryConfig{}, false
}

func firstSummaryBoolConfig(body []byte, paths []string) (SummaryConfig, bool) {
	for _, path := range paths {
		if config, ok := summaryBoolConfig(body, path); ok {
			return config, true
		}
	}
	return SummaryConfig{}, false
}

func summaryBoolConfig(body []byte, path string) (SummaryConfig, bool) {
	switch value := gjson.GetBytes(body, path); value.Type {
	case gjson.True:
		return SummaryConfig{Mode: SummaryEnabled, Detail: "auto"}, true
	case gjson.False:
		return SummaryConfig{Mode: SummaryDisabled}, true
	default:
		return SummaryConfig{}, false
	}
}

func responsesSummaryConfig(body []byte, path string) (SummaryConfig, bool) {
	value := gjson.GetBytes(body, path)
	if value.Raw == "" {
		return SummaryConfig{}, false
	}
	if value.Type == gjson.Null {
		return SummaryConfig{Mode: SummaryDisabled}, true
	}
	if value.Type != gjson.String {
		return SummaryConfig{}, false
	}

	switch raw := strings.ToLower(strings.TrimSpace(value.String())); raw {
	case "auto", "concise", "detailed":
		return SummaryConfig{Mode: SummaryEnabled, Detail: raw}, true
	case "none":
		return SummaryConfig{Mode: SummaryDisabled}, true
	default:
		return SummaryConfig{}, false
	}
}

func claudeSummaryConfig(body []byte, path string) (SummaryConfig, bool) {
	value := gjson.GetBytes(body, path)
	if value.Type != gjson.String {
		return SummaryConfig{}, false
	}
	switch strings.ToLower(strings.TrimSpace(value.String())) {
	case "summarized":
		return SummaryConfig{Mode: SummaryEnabled, Detail: "auto"}, true
	case "omitted":
		return SummaryConfig{Mode: SummaryDisabled}, true
	default:
		return SummaryConfig{}, false
	}
}

func enableClaudeThinkingForSummary(body []byte, model string) []byte {
	baseModel := ParseSuffix(model).ModelName
	if baseModel == "" {
		baseModel = ParseSuffix(gjson.GetBytes(body, "model").String()).ModelName
	}
	modelInfo := registry.LookupModelInfo(baseModel, "claude")
	if modelInfo == nil || modelInfo.Thinking == nil {
		return body
	}
	if len(modelInfo.Thinking.Levels) > 0 {
		body, _ = sjson.SetBytes(body, "thinking.type", "adaptive")
		body, _ = sjson.DeleteBytes(body, "thinking.budget_tokens")
		return body
	}
	budget := modelInfo.Thinking.Min
	if budget <= 0 {
		return body
	}
	if maxTokens := gjson.GetBytes(body, "max_tokens"); maxTokens.Exists() && maxTokens.Int() <= int64(budget) {
		return body
	}
	body, _ = sjson.SetBytes(body, "thinking.type", "enabled")
	body, _ = sjson.SetBytes(body, "thinking.budget_tokens", budget)
	return body
}

func normalizedSummaryDetail(detail string) string {
	switch strings.ToLower(strings.TrimSpace(detail)) {
	case "concise":
		return "concise"
	case "detailed":
		return "detailed"
	default:
		return "auto"
	}
}
