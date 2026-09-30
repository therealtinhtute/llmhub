package auth

import (
	"encoding/json"
	"maps"
	"strconv"
	"strings"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/modelconfig"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/thinking"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// Ported from upstream CLIProxyAPI commit ed980be34b99
// (sdk/cliproxy/auth/api_key_model_capabilities.go). The fork resolves the
// capability snapshot from the runtime config directly instead of the bundled
// routing snapshot (which upstream shares with alias resolution); semantics are
// identical.
type apiKeyModelCapabilityRoute struct {
	upstreamModel string
	modelInfo     *registry.ModelInfo
}

type apiKeyModelCapabilityTable map[string]map[string][]apiKeyModelCapabilityRoute

type apiKeyModelCapabilitySnapshot struct {
	config       *internalconfig.Config
	capabilities apiKeyModelCapabilityTable
}

func isConfiguredModelRoutingAuth(auth *Auth) bool {
	if auth != nil && auth.AuthKind() == AuthKindAPIKey {
		return true
	}
	if auth == nil || auth.AuthSourceKind() != AuthSourceConfig || auth.Attributes == nil {
		return false
	}
	return strings.TrimSpace(auth.Attributes["compat_name"]) != ""
}

func (m *Manager) loadAPIKeyModelCapabilities() *apiKeyModelCapabilitySnapshot {
	if m == nil {
		return &apiKeyModelCapabilitySnapshot{config: &internalconfig.Config{}}
	}
	snapshot, _ := m.apiKeyModelCapabilities.Load().(*apiKeyModelCapabilitySnapshot)
	if snapshot == nil {
		return &apiKeyModelCapabilitySnapshot{config: &internalconfig.Config{}}
	}
	return snapshot
}

// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
// (sdk/cliproxy/auth/api_key_model_capabilities.go). The fork has no API-key
// capability snapshot producer yet; the API-key key is kept so ResolvedModelInfo
// preserves upstream's home-first precedence when one is introduced.
const (
	resolvedAPIKeyModelInfoMetadataKey     = "cliproxy.resolved_api_key_model_info"
	resolvedCodexOAuthModelInfoMetadataKey = "cliproxy.resolved_codex_oauth_model_info"
	resolvedHomeModelInfoMetadataKey       = "cliproxy.resolved_home_model_info"
	resolvedHomeModelOptionsMetadataKey    = "cliproxy.resolved_home_model_options"
)

// ResolvedAPIKeyModelInfo returns the exact configured model definition bound to
// this API-key execution attempt.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func ResolvedAPIKeyModelInfo(req cliproxyexecutor.Request) (*registry.ModelInfo, bool) {
	modelInfo, ok := req.Metadata[resolvedAPIKeyModelInfoMetadataKey].(*registry.ModelInfo)
	if !ok || modelInfo == nil {
		return nil, false
	}
	return modelInfo, true
}

// ResolvedHomeModelOptions returns the credential model options selected for this attempt.
// A present but unmatched or empty models list yields zero-value options.
func ResolvedHomeModelOptions(req cliproxyexecutor.Request) (internalconfig.OpenAICompatibilityModel, bool) {
	options, ok := req.Metadata[resolvedHomeModelOptionsMetadataKey].(internalconfig.OpenAICompatibilityModel)
	return options, ok
}

// ResolvedModelInfo returns the authoritative model capabilities bound to this
// execution attempt. Home-dispatched model info wins over configured API-key
// model info.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func ResolvedModelInfo(req cliproxyexecutor.Request) (*registry.ModelInfo, bool) {
	if modelInfo, ok := req.Metadata[resolvedHomeModelInfoMetadataKey].(*registry.ModelInfo); ok && modelInfo != nil {
		return modelInfo, true
	}
	if modelInfo, ok := req.Metadata[resolvedCodexOAuthModelInfoMetadataKey].(*registry.ModelInfo); ok && modelInfo != nil {
		return modelInfo, true
	}
	return ResolvedAPIKeyModelInfo(req)
}

// attachResolvedHomeModelInfo binds the home-dispatched model capabilities to
// the downstream executor request via request metadata.
//
// Ported from upstream CLIProxyAPI commit 6ff680e90ab5
func attachResolvedHomeModelInfo(req cliproxyexecutor.Request, auth *Auth, routeModel string, modelInfo *registry.ModelInfo, support *bool) cliproxyexecutor.Request {
	upstreamModel := req.Model
	if modelInfo != nil {
		upstreamModel = modelInfo.ID
	}
	if auth != nil {
		if dispatched := strings.TrimSpace(auth.Attributes[homeUpstreamModelAttributeKey]); dispatched != "" {
			upstreamModel = dispatched
		}
	}
	options, configured := homeAPIKeyModelOptions(auth, upstreamModel, routeModel)
	if modelInfo == nil && !configured {
		return req
	}
	metadata := make(map[string]any, len(req.Metadata)+2)
	maps.Copy(metadata, req.Metadata)
	delete(metadata, resolvedHomeModelOptionsMetadataKey)
	if configured {
		metadata[resolvedHomeModelOptionsMetadataKey] = options
	}
	req.Metadata = metadata
	if modelInfo == nil {
		return req
	}
	selected := *modelInfo

	local, ok := ResolvedAPIKeyModelInfo(req)
	if !ok {
		local, ok = req.Metadata[resolvedCodexOAuthModelInfoMetadataKey].(*registry.ModelInfo)
	}
	sameModel := ok && local != nil && strings.EqualFold(strings.TrimSpace(thinking.ParseSuffix(local.ID).ModelName), strings.TrimSpace(thinking.ParseSuffix(selected.ID).ModelName))
	if support != nil {
		selected.SupportConfigurationUpdate = *support
	} else {
		selected.SupportConfigurationUpdate = sameModel && local.SupportConfigurationUpdate
	}
	if configured {
		selected.IsCompat = options.IsCompat
	}

	metadata[resolvedHomeModelInfoMetadataKey] = &selected
	req.Metadata = metadata
	return req
}

// CodexAPIKeyModelIsCompat reports whether the selected codex-api-key model has
// is-compat enabled. When true and codex.optimize-multi-agent-v2 is also true,
// Codex MultiAgentV2 agent_message items are converted into portable Responses
// message/user input for third-party Responses-compatible endpoints.
func CodexAPIKeyModelIsCompat(cfg *internalconfig.Config, auth *Auth, model string) bool {
	if cfg == nil || auth == nil || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		return false
	}
	entry := resolveCodexAPIKeyConfig(cfg, auth)
	if cfg.Home.Enabled {
		if options, configured := homeAPIKeyModelOptions(auth, model, model); configured {
			return options.IsCompat
		}
	}
	if entry == nil || len(entry.Models) == 0 {
		return false
	}
	requested := strings.TrimSpace(model)
	if requested == "" {
		return false
	}
	baseModel := strings.TrimSpace(thinking.ParseSuffix(requested).ModelName)
	if baseModel == "" {
		baseModel = requested
	}
	for i := range entry.Models {
		name := strings.TrimSpace(entry.Models[i].Name)
		alias := strings.TrimSpace(entry.Models[i].Alias)
		if name == "" {
			name = alias
		}
		if alias == "" {
			alias = name
		}
		if name == "" {
			continue
		}
		if strings.EqualFold(name, requested) || strings.EqualFold(name, baseModel) ||
			strings.EqualFold(alias, requested) || strings.EqualFold(alias, baseModel) {
			return entry.Models[i].IsCompat
		}
	}
	return false
}

// attachResolvedExecutionModelInfo binds the authoritative model capabilities
// for an execution attempt. A restored execution model only consults the Codex
// OAuth capability table (upstream attachResolvedExecutionModelInfo).
func (m *Manager) attachResolvedExecutionModelInfo(req cliproxyexecutor.Request, auth *Auth, routeModel, upstreamModel string, restoreExecutionModel bool) cliproxyexecutor.Request {
	if restoreExecutionModel {
		if auth == nil || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
			return req
		}
		routeModel, upstreamModel = req.Model, req.Model
	}
	return m.attachResolvedAPIKeyModelInfo(req, auth, routeModel, upstreamModel)
}

func (m *Manager) attachResolvedAPIKeyModelInfo(req cliproxyexecutor.Request, auth *Auth, routeModel, upstreamModel string) cliproxyexecutor.Request {
	return attachResolvedAPIKeyModelInfo(m.loadAPIKeyModelCapabilities(), req, auth, routeModel, upstreamModel)
}

func attachResolvedAPIKeyModelInfo(snapshot *apiKeyModelCapabilitySnapshot, req cliproxyexecutor.Request, auth *Auth, routeModel, upstreamModel string) cliproxyexecutor.Request {
	modelInfo, ok := lookupAPIKeyModelCapability(snapshot, auth, routeModel, upstreamModel)
	if !ok {
		modelInfo, ok = lookupUnlistedCodexAPIKeyModelCapability(snapshot, auth, upstreamModel)
	}
	metadataKey := resolvedAPIKeyModelInfoMetadataKey
	if !ok {
		modelInfo, ok = lookupCodexOAuthModelCapability(auth, upstreamModel)
		metadataKey = resolvedCodexOAuthModelInfoMetadataKey
	}
	if !ok {
		_, hadAPIKey := req.Metadata[resolvedAPIKeyModelInfoMetadataKey]
		_, hadOAuth := req.Metadata[resolvedCodexOAuthModelInfoMetadataKey]
		if !hadAPIKey && !hadOAuth {
			return req
		}
	}
	metadata := make(map[string]any, len(req.Metadata)+1)
	maps.Copy(metadata, req.Metadata)
	delete(metadata, resolvedAPIKeyModelInfoMetadataKey)
	delete(metadata, resolvedCodexOAuthModelInfoMetadataKey)
	if ok {
		metadata[metadataKey] = modelInfo
	}
	req.Metadata = metadata
	return req
}

func lookupUnlistedCodexAPIKeyModelCapability(snapshot *apiKeyModelCapabilitySnapshot, auth *Auth, upstreamModel string) (*registry.ModelInfo, bool) {
	if snapshot == nil || auth == nil || auth.AuthKind() != AuthKindAPIKey || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") || strings.TrimSpace(upstreamModel) == "" {
		return nil, false
	}
	entry := resolveCodexAPIKeyConfig(snapshot.config, auth)
	if entry == nil || auth.Attributes == nil {
		return nil, false
	}
	key := strings.TrimSpace(auth.Attributes[AttributeAPIKey])
	baseURL := strings.TrimSpace(auth.Attributes["base_url"])
	if (key == "" && baseURL == "") || !strings.EqualFold(key, strings.TrimSpace(entry.APIKey)) ||
		(strings.TrimSpace(entry.BaseURL) != "" && !strings.EqualFold(baseURL, strings.TrimSpace(entry.BaseURL))) {
		return nil, false
	}
	for _, configured := range entry.Models {
		if strings.EqualFold(strings.TrimSpace(configured.Name), strings.TrimSpace(upstreamModel)) || configuredUpstreamFallbackMatches(configured.Name, upstreamModel) {
			info := modelconfig.ResolveModelInfo(upstreamModel, "codex", configured.Thinking)
			info.SupportConfigurationUpdate = configured.SupportConfigurationUpdate
			info.IsCompat = configured.IsCompat
			return info, true
		}
	}
	info := modelconfig.ResolveModelInfo(upstreamModel, "codex", nil)
	info.SupportConfigurationUpdate = false
	return info, true
}

func lookupCodexOAuthModelCapability(auth *Auth, upstreamModel string) (*registry.ModelInfo, bool) {
	if auth == nil || auth.AuthKind() != AuthKindOAuth || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		return nil, false
	}
	var models []*registry.ModelInfo
	switch strings.ToLower(strings.TrimSpace(auth.Attributes["plan_type"])) {
	case "plus":
		models = registry.GetCodexPlusModels()
	case "team", "business", "go":
		models = registry.GetCodexTeamModels()
	case "free":
		models = registry.GetCodexFreeModels()
	default:
		models = registry.GetCodexProModels()
	}
	selected := strings.TrimSpace(thinking.ParseSuffix(strings.TrimSpace(upstreamModel)).ModelName)
	for _, model := range models {
		if model != nil && strings.EqualFold(model.ID, selected) {
			return model, true
		}
	}
	return nil, false
}

func lookupAPIKeyModelCapability(snapshot *apiKeyModelCapabilitySnapshot, auth *Auth, routeModel, upstreamModel string) (*registry.ModelInfo, bool) {
	if !isConfiguredModelRoutingAuth(auth) || snapshot == nil {
		return nil, false
	}
	byRoute := snapshot.capabilities[strings.TrimSpace(auth.ID)]
	if len(byRoute) == 0 {
		return nil, false
	}
	requestedModel := rewriteModelForAuth(strings.TrimSpace(routeModel), auth)
	_, candidates := modelAliasLookupCandidates(requestedModel)
	routes := make([]apiKeyModelCapabilityRoute, 0)
	for _, candidate := range candidates {
		routes = append(routes, byRoute[strings.ToLower(strings.TrimSpace(candidate))]...)
	}
	selected := strings.TrimSpace(upstreamModel)
	for _, route := range routes {
		if strings.EqualFold(strings.TrimSpace(route.upstreamModel), selected) {
			return route.modelInfo, route.modelInfo != nil
		}
	}
	for _, route := range routes {
		if configuredUpstreamFallbackMatches(route.upstreamModel, selected) {
			return route.modelInfo, route.modelInfo != nil
		}
	}
	return nil, false
}

func configuredUpstreamFallbackMatches(configured, selected string) bool {
	configuredResult := thinking.ParseSuffix(strings.TrimSpace(configured))
	if configuredResult.HasSuffix {
		return false
	}
	selectedResult := thinking.ParseSuffix(strings.TrimSpace(selected))
	return strings.EqualFold(strings.TrimSpace(configuredResult.ModelName), strings.TrimSpace(selectedResult.ModelName))
}

// compileAPIKeyModelCapabilitiesForAuth builds the route -> capability table for
// one auth (upstream compileAPIKeyModelCapabilitiesForAuth). Providers without
// local configured-model support (interactions, xai) are intentionally absent.
func compileAPIKeyModelCapabilitiesForAuth(cfg *internalconfig.Config, auth *Auth) map[string][]apiKeyModelCapabilityRoute {
	if cfg == nil || !isConfiguredModelRoutingAuth(auth) {
		return nil
	}
	out := make(map[string][]apiKeyModelCapabilityRoute)
	switch strings.ToLower(strings.TrimSpace(auth.Provider)) {
	case "gemini":
		if entry := resolveGeminiAPIKeyConfig(cfg, auth); entry != nil {
			compileConfiguredModelCapabilities(out, entry.Models, "gemini")
		}
	case "claude":
		if entry := resolveClaudeAPIKeyConfig(cfg, auth); entry != nil {
			compileConfiguredModelCapabilities(out, entry.Models, "claude")
		}
	case "codex":
		if entry := resolveCodexAPIKeyConfig(cfg, auth); entry != nil {
			for i := range entry.Models {
				configured := entry.Models[i]
				info := addConfiguredModelCapability(out, configured.Name, configured.Alias, "codex", configured.Thinking, configured.IsCompat)
				if info != nil {
					info.SupportConfigurationUpdate = configured.SupportConfigurationUpdate
				}
			}
		}
	case "vertex":
		if entry := resolveVertexAPIKeyConfig(cfg, auth); entry != nil {
			compileConfiguredModelCapabilities(out, entry.Models, "gemini")
		}
	case "meta":
		if entry := resolveMetaAPIKeyConfig(cfg, auth); entry != nil {
			compileConfiguredModelCapabilities(out, entry.Models, "meta")
		}
	default:
		providerKey, compatName := "", ""
		if auth.Attributes != nil {
			providerKey = strings.TrimSpace(auth.Attributes["provider_key"])
			compatName = strings.TrimSpace(auth.Attributes["compat_name"])
		}
		if entry := resolveOpenAICompatConfigForAuth(cfg, auth, providerKey, compatName); entry != nil {
			compileOpenAICompatibleModelCapabilities(out, entry.Models)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func compileConfiguredModelCapabilities[T interface {
	GetName() string
	GetAlias() string
	GetThinking() *registry.ThinkingSupport
}](out map[string][]apiKeyModelCapabilityRoute, models []T, modelType string) {
	for i := range models {
		isCompat := false
		if compatModel, okCompat := any(models[i]).(interface{ GetIsCompat() bool }); okCompat {
			isCompat = compatModel.GetIsCompat()
		}
		addConfiguredModelCapability(out, models[i].GetName(), models[i].GetAlias(), modelType, models[i].GetThinking(), isCompat)
	}
}

func compileOpenAICompatibleModelCapabilities(out map[string][]apiKeyModelCapabilityRoute, models []internalconfig.OpenAICompatibilityModel) {
	for i := range models {
		support := models[i].Thinking
		if support == nil && !models[i].Image {
			support = &registry.ThinkingSupport{Levels: []string{"low", "medium", "high"}}
		}
		addConfiguredModelCapability(out, models[i].Name, models[i].Alias, "openai-compatibility", support, models[i].IsCompat)
	}
}

func addConfiguredModelCapability(out map[string][]apiKeyModelCapabilityRoute, name, alias, modelType string, support *registry.ThinkingSupport, isCompat bool) *registry.ModelInfo {
	name = strings.TrimSpace(name)
	alias = strings.TrimSpace(alias)
	if name == "" {
		name = alias
	}
	if alias == "" {
		alias = name
	}
	if name == "" {
		return nil
	}
	modelInfo := modelconfig.ResolveModelInfo(name, modelType, support)
	modelInfo.IsCompat = isCompat
	route := apiKeyModelCapabilityRoute{upstreamModel: name, modelInfo: modelInfo}
	seenKeys := make(map[string]struct{})
	for _, routeModel := range []string{alias, name} {
		_, candidates := modelAliasLookupCandidates(routeModel)
		for _, candidate := range candidates {
			key := strings.ToLower(strings.TrimSpace(candidate))
			if key == "" {
				continue
			}
			if _, exists := seenKeys[key]; exists {
				continue
			}
			seenKeys[key] = struct{}{}
			duplicate := false
			for _, existing := range out[key] {
				if strings.EqualFold(existing.upstreamModel, route.upstreamModel) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				out[key] = append(out[key], route)
			}
		}
	}
	return modelInfo
}

// resolveOpenAICompatConfigForAuth prefers the exact configured entry recorded
// in auth.Attributes[config_index] before falling back to name matching
// (upstream conductor_models.go resolveOpenAICompatConfigForAuth).
func resolveOpenAICompatConfigForAuth(cfg *internalconfig.Config, auth *Auth, providerKey, compatName string) *internalconfig.OpenAICompatibility {
	if cfg == nil {
		return nil
	}
	if auth != nil && auth.AuthSourceKind() == AuthSourceConfig && auth.Attributes != nil {
		if index, errIndex := strconv.Atoi(strings.TrimSpace(auth.Attributes[AttributeConfigIndex])); errIndex == nil && index >= 0 && index < len(cfg.OpenAICompatibility) && !cfg.OpenAICompatibility[index].Disabled {
			return &cfg.OpenAICompatibility[index]
		}
	}
	authProvider := ""
	if auth != nil {
		authProvider = auth.Provider
	}
	return resolveOpenAICompatConfig(cfg, providerKey, compatName, authProvider)
}

// homeAPIKeyModelOptions reads the selected credential's model configuration.
// A present models list is authoritative, including empty lists and false defaults.
// Ported from upstream CLIProxyAPI commit b9971ee3/ac42accc.
func homeAPIKeyModelOptions(auth *Auth, model, routeModel string) (internalconfig.OpenAICompatibilityModel, bool) {
	var empty internalconfig.OpenAICompatibilityModel
	if auth == nil {
		return empty, false
	}
	raw, exists := auth.Metadata["credential_options"]
	if !exists {
		return empty, false
	}
	data, errMarshal := json.Marshal(raw)
	if errMarshal != nil {
		return empty, false
	}
	var options struct {
		Models json.RawMessage `json:"models"`
	}
	if errDecode := json.Unmarshal(data, &options); errDecode != nil || len(options.Models) == 0 {
		return empty, false
	}
	var models []internalconfig.OpenAICompatibilityModel
	if errDecode := json.Unmarshal(options.Models, &models); errDecode != nil {
		return empty, false
	}
	requested := strings.TrimSpace(model)
	if requested == "" {
		return empty, true
	}
	baseModel := strings.TrimSpace(thinking.ParseSuffix(requested).ModelName)
	_, routeCandidates := modelAliasLookupCandidates(rewriteModelForAuth(strings.TrimSpace(routeModel), auth))
	for _, route := range routeCandidates {
		for _, candidate := range []string{requested, baseModel} {
			for _, configured := range models {
				name := strings.TrimSpace(configured.Name)
				alias := strings.TrimSpace(configured.Alias)
				if name == "" {
					name = alias
				}
				if strings.EqualFold(name, candidate) && (strings.EqualFold(alias, route) || strings.EqualFold(name, route)) {
					return configured, true
				}
			}
		}
	}
	// Prefer upstream names over aliases and exact suffixes over base fallbacks.
	for _, useAlias := range []bool{false, true} {
		for _, candidate := range []string{requested, baseModel} {
			if candidate == "" {
				continue
			}
			for _, configured := range models {
				name := strings.TrimSpace(configured.Name)
				if useAlias || name == "" {
					name = strings.TrimSpace(configured.Alias)
				}
				if strings.EqualFold(name, candidate) {
					return configured, true
				}
			}
		}
	}
	return empty, true
}
