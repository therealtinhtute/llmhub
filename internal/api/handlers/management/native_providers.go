package management

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/nativeproviders"
)

type nativeProviderResourceRequest struct {
	ID      string                   `json:"id"`
	APIKey  *string                  `json:"api_key"`
	Enabled *bool                    `json:"enabled"`
	Models  *[]nativeproviders.Model `json:"models"`
}

// GetNativeProviderResources returns redacted provider-owned resources.
func (h *Handler) GetNativeProviderResources(c *gin.Context) {
	provider, err := nativeproviders.NormalizeProvider(c.Param("provider"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if h == nil || h.nativeProviderStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "native provider store is not configured"})
		return
	}
	records, err := h.nativeProviderStore.ListNativeProviderResources(c.Request.Context(), provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resources := make([]nativeproviders.PublicResource, 0, len(records))
	for _, record := range records {
		resource, errDecode := nativeproviders.DecodeResource(provider, record.Payload)
		if errDecode != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errDecode.Error()})
			return
		}
		publicResource, errPublic := nativeproviders.ToPublicResource(provider, resource)
		if errPublic != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errPublic.Error()})
			return
		}
		resources = append(resources, publicResource)
	}
	c.JSON(http.StatusOK, gin.H{
		"provider":  provider,
		"resources": resources,
	})
}

// PutNativeProviderResource creates or replaces one native provider resource.
func (h *Handler) PutNativeProviderResource(c *gin.Context) {
	h.upsertNativeProviderResource(c, "")
}

// PatchNativeProviderResource partially updates one native provider resource.
func (h *Handler) PatchNativeProviderResource(c *gin.Context) {
	h.upsertNativeProviderResource(c, strings.TrimSpace(c.Param("id")))
}

func (h *Handler) upsertNativeProviderResource(c *gin.Context, pathID string) {
	provider, err := nativeproviders.NormalizeProvider(c.Param("provider"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if h == nil || h.nativeProviderStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "native provider store is not configured"})
		return
	}
	var request nativeProviderResourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	id := strings.TrimSpace(pathID)
	if id == "" {
		id = strings.TrimSpace(request.ID)
	}
	records, err := h.nativeProviderStore.ListNativeProviderResources(c.Request.Context(), provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resource any
	var exists bool
	for _, record := range records {
		if strings.TrimSpace(record.ID) != id || id == "" {
			continue
		}
		resource, err = nativeproviders.DecodeResource(provider, record.Payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		exists = true
		break
	}
	if id == "" {
		id, err = nativeproviders.NewID(provider)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if !exists {
		switch provider {
		case nativeproviders.ProviderOpenRouter:
			resource = &nativeproviders.OpenRouterResource{ID: id, Enabled: true}
		case nativeproviders.ProviderOpenCode:
			resource = &nativeproviders.OpenCodeResource{ID: id, Enabled: true}
		}
	}
	if err := applyNativeProviderRequest(provider, resource, request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	payload, err := nativeproviders.EncodeResource(provider, resource)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.nativeProviderStore.SaveNativeProviderResource(c.Request.Context(), provider, id, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.notifyNativeProviderChange(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	publicResource, err := nativeproviders.ToPublicResource(provider, resource)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": provider, "resource": publicResource})
}

func applyNativeProviderRequest(provider string, resource any, request nativeProviderResourceRequest) error {
	if _, err := nativeproviders.NormalizeProvider(provider); err != nil {
		return err
	}
	switch provider {
	case nativeproviders.ProviderOpenRouter:
		value, ok := resource.(*nativeproviders.OpenRouterResource)
		if !ok || value == nil {
			return fmt.Errorf("invalid OpenRouter resource")
		}
		if request.APIKey != nil {
			value.APIKey = strings.TrimSpace(*request.APIKey)
		}
		if request.Enabled != nil {
			value.Enabled = *request.Enabled
		}
		if request.Models != nil {
			value.Models = nativeproviders.NormalizeModels(*request.Models)
		}
	case nativeproviders.ProviderOpenCode:
		value, ok := resource.(*nativeproviders.OpenCodeResource)
		if !ok || value == nil {
			return fmt.Errorf("invalid OpenCode resource")
		}
		if request.APIKey != nil {
			value.APIKey = strings.TrimSpace(*request.APIKey)
		}
		if request.Enabled != nil {
			value.Enabled = *request.Enabled
		}
		if request.Models != nil {
			value.Models = nativeproviders.NormalizeModels(*request.Models)
		}
	}
	return nil
}

// DeleteNativeProviderResource removes one native resource and its runtime projection.
func (h *Handler) DeleteNativeProviderResource(c *gin.Context) {
	provider, err := nativeproviders.NormalizeProvider(c.Param("provider"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if h == nil || h.nativeProviderStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "native provider store is not configured"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if err := h.nativeProviderStore.DeleteNativeProviderResource(c.Request.Context(), provider, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.notifyNativeProviderChange(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetNativeProviderModels loads remote models server-side and degrades to a fallback catalog.
func (h *Handler) GetNativeProviderModels(c *gin.Context) {
	provider, err := nativeproviders.NormalizeProvider(c.Param("provider"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if h == nil || h.nativeProviderStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "native provider store is not configured"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	records, err := h.nativeProviderStore.ListNativeProviderResources(c.Request.Context(), provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var apiKey string
	found := false
	for _, record := range records {
		if record.ID != id {
			continue
		}
		resource, errDecode := nativeproviders.DecodeResource(provider, record.Payload)
		if errDecode != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errDecode.Error()})
			return
		}
		switch provider {
		case nativeproviders.ProviderOpenRouter:
			apiKey = resource.(*nativeproviders.OpenRouterResource).APIKey
		case nativeproviders.ProviderOpenCode:
			apiKey = resource.(*nativeproviders.OpenCodeResource).APIKey
		}
		found = true
		break
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	models, source, fetchErr := nativeproviders.FetchRemoteModels(c.Request.Context(), provider, apiKey)
	response := gin.H{
		"provider":    provider,
		"resource_id": id,
		"source":      source,
		"models":      models,
	}
	if fetchErr != nil {
		response["error"] = fetchErr.Error()
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) notifyNativeProviderChange(ctx context.Context) error {
	if h == nil || h.nativeProviderStore == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	h.mu.Lock()
	if h.cfg == nil {
		h.mu.Unlock()
		return nil
	}
	if err := nativeproviders.HydrateConfig(ctx, h.cfg, h.nativeProviderStore); err != nil {
		h.mu.Unlock()
		return err
	}
	cfg := h.cfg
	hook := h.configChangeHook
	h.mu.Unlock()
	if hook != nil {
		hook(cfg)
	}
	return nil
}
