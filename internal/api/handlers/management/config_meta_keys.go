package management

// Meta API-key management handlers for the meta-api-key config family.
// Ported from upstream CLIProxyAPI commit e475807a
// (internal/api/handlers/management/config_lists.go meta-api-key section).
// MetaKey = CodexKey, so the codex-style normalization/weight helpers apply.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/config"
)

// metaDefaultBaseURL is the upstream default Meta API base URL
// (SanitizeMetaKeys applies the same default on load).
const metaDefaultBaseURL = "https://api.meta.ai/v1"

// meta-api-key: []MetaKey
func (h *Handler) GetMetaKeys(c *gin.Context) {
	c.JSON(200, gin.H{"meta-api-key": h.metaKeysWithAuthIndex()})
}

func (h *Handler) PutMetaKeys(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read body"})
		return
	}
	var arr []config.MetaKey
	if err = json.Unmarshal(data, &arr); err != nil {
		var obj struct {
			Items []config.MetaKey `json:"items"`
		}
		if err2 := json.Unmarshal(data, &obj); err2 != nil || len(obj.Items) == 0 {
			c.JSON(400, gin.H{"error": "invalid body"})
			return
		}
		arr = obj.Items
	}
	// Unlike codex-api-key, empty base-url defaults to the Meta API endpoint
	// instead of dropping the entry (upstream e475807a).
	filtered := make([]config.MetaKey, 0, len(arr))
	for i := range arr {
		entry := arr[i]
		normalizeCodexKey(&entry)
		if entry.BaseURL == "" {
			entry.BaseURL = metaDefaultBaseURL
		}
		if err := validateConfigCredentialWeight(entry.Weight); err != nil {
			c.JSON(400, gin.H{"error": fmt.Sprintf("meta-api-key[%d].weight: %v", i, err)})
			return
		}
		filtered = append(filtered, entry)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg.MetaKey = filtered
	h.cfg.SanitizeMetaKeys()
	h.persistLockedAndRespond(c)
}

func (h *Handler) PatchMetaKey(c *gin.Context) {
	type metaKeyPatch struct {
		APIKey              *string                          `json:"api-key"`
		Priority            *int                             `json:"priority"`
		Weight              *int64                           `json:"weight"`
		Prefix              *string                          `json:"prefix"`
		BaseURL             *string                          `json:"base-url"`
		ProxyURL            *string                          `json:"proxy-url"`
		Models              *[]config.CodexModel             `json:"models"`
		Headers             *map[string]string               `json:"headers"`
		ExcludedModels      *[]string                        `json:"excluded-models"`
		DisableCooling      json.RawMessage                  `json:"disable-cooling"`
		RequestRetry        *int                             `json:"request-retry"`
		RequestScopedErrors *[]config.RequestScopedErrorRule `json:"request-scoped-errors"`
	}
	var body struct {
		Index *int          `json:"index"`
		Match *string       `json:"match"`
		Value *metaKeyPatch `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Value == nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	targetIndex := -1
	if body.Index != nil && *body.Index >= 0 && *body.Index < len(h.cfg.MetaKey) {
		targetIndex = *body.Index
	}
	if targetIndex == -1 && body.Match != nil {
		match := strings.TrimSpace(*body.Match)
		for i := range h.cfg.MetaKey {
			if strings.TrimSpace(h.cfg.MetaKey[i].APIKey) == match {
				targetIndex = i
				break
			}
		}
	}
	if targetIndex == -1 {
		c.JSON(404, gin.H{"error": "item not found"})
		return
	}

	entry := h.cfg.MetaKey[targetIndex]
	if body.Value.APIKey != nil {
		entry.APIKey = strings.TrimSpace(*body.Value.APIKey)
	}
	if body.Value.Priority != nil {
		entry.Priority = *body.Value.Priority
	}
	if err := applyConfigCredentialWeight(&entry.Weight, body.Value.Weight); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if body.Value.Prefix != nil {
		entry.Prefix = strings.TrimSpace(*body.Value.Prefix)
	}
	if body.Value.BaseURL != nil {
		trimmed := strings.TrimSpace(*body.Value.BaseURL)
		// Upstream meta semantics: empty base-url resets to the default
		// endpoint rather than deleting the entry.
		if trimmed == "" {
			trimmed = metaDefaultBaseURL
		}
		entry.BaseURL = trimmed
	}
	if body.Value.ProxyURL != nil {
		entry.ProxyURL = strings.TrimSpace(*body.Value.ProxyURL)
	}
	if body.Value.Models != nil {
		entry.Models = append([]config.CodexModel(nil), (*body.Value.Models)...)
	}
	if body.Value.Headers != nil {
		entry.Headers = config.NormalizeHeaders(*body.Value.Headers)
	}
	if body.Value.ExcludedModels != nil {
		entry.ExcludedModels = config.NormalizeExcludedModels(*body.Value.ExcludedModels)
	}
	if !applyDisableCoolingPatch(c, body.Value.DisableCooling, &entry.DisableCooling) {
		return
	}
	if body.Value.RequestRetry != nil {
		entry.RequestRetry = body.Value.RequestRetry
	}
	if body.Value.RequestScopedErrors != nil {
		entry.RequestScopedErrors = append([]config.RequestScopedErrorRule(nil), *body.Value.RequestScopedErrors...)
	}
	normalizeCodexKey(&entry)
	h.cfg.MetaKey[targetIndex] = entry
	h.cfg.SanitizeMetaKeys()
	h.persistLockedAndRespond(c)
}

func (h *Handler) DeleteMetaKey(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if val := strings.TrimSpace(c.Query("api-key")); val != "" {
		if baseRaw, okBase := c.GetQuery("base-url"); okBase {
			base := strings.TrimSpace(baseRaw)
			// Upstream meta semantics: with api-key + base-url, remove every
			// matching entry (no ambiguity error).
			out := make([]config.MetaKey, 0, len(h.cfg.MetaKey))
			for _, entry := range h.cfg.MetaKey {
				if strings.TrimSpace(entry.APIKey) == val && strings.TrimSpace(entry.BaseURL) == base {
					continue
				}
				out = append(out, entry)
			}
			h.cfg.MetaKey = out
			h.cfg.SanitizeMetaKeys()
			h.persistLockedAndRespond(c)
			return
		}

		matchIndex := -1
		matchCount := 0
		for i := range h.cfg.MetaKey {
			if strings.TrimSpace(h.cfg.MetaKey[i].APIKey) == val {
				matchCount++
				if matchIndex == -1 {
					matchIndex = i
				}
			}
		}
		if matchCount > 1 {
			c.JSON(400, gin.H{"error": "multiple items match api-key; base-url is required"})
			return
		}
		if matchIndex != -1 {
			h.cfg.MetaKey = append(h.cfg.MetaKey[:matchIndex], h.cfg.MetaKey[matchIndex+1:]...)
		}
		h.cfg.SanitizeMetaKeys()
		h.persistLockedAndRespond(c)
		return
	}
	if idxStr := c.Query("index"); idxStr != "" {
		var idx int
		_, err := fmt.Sscanf(idxStr, "%d", &idx)
		if err == nil && idx >= 0 && idx < len(h.cfg.MetaKey) {
			h.cfg.MetaKey = append(h.cfg.MetaKey[:idx], h.cfg.MetaKey[idx+1:]...)
			h.cfg.SanitizeMetaKeys()
			h.persistLockedAndRespond(c)
			return
		}
	}
	c.JSON(400, gin.H{"error": "missing api-key or index"})
}
