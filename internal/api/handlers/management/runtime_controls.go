package management

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
)

// SetRuntimeSettingsStore configures database-backed runtime control management APIs.
func (h *Handler) SetRuntimeSettingsStore(store runtimecontrol.SettingsStore) {
	if h == nil {
		return
	}
	h.runtimeSettingsStore = store
}

// GetRuntimeControls returns the database-authoritative runtime controls.
func (h *Handler) GetRuntimeControls(c *gin.Context) {
	store, ok := h.requireRuntimeSettingsStore(c)
	if !ok {
		return
	}
	settings, err := store.LoadRuntimeSettings(c.Request.Context())
	if err != nil {
		runtimeControlsError(c, http.StatusServiceUnavailable, "runtime controls unavailable")
		return
	}
	settings, err = settings.Normalize()
	if err != nil {
		runtimeControlsError(c, http.StatusServiceUnavailable, fmt.Sprintf("runtime controls invalid: %v", err))
		return
	}
	c.JSON(http.StatusOK, settings)
}

// PutRuntimeControls replaces runtime controls using optimistic revision checks.
func (h *Handler) PutRuntimeControls(c *gin.Context) {
	store, ok := h.requireRuntimeSettingsStore(c)
	if !ok {
		return
	}
	var settings runtimecontrol.Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		runtimeControlsError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if settings.Revision <= 0 {
		runtimeControlsError(c, http.StatusBadRequest, "runtime controls revision is required")
		return
	}
	normalized, err := settings.Normalize()
	if err != nil {
		runtimeControlsError(c, http.StatusBadRequest, err.Error())
		return
	}
	saved, err := store.SaveRuntimeSettings(c.Request.Context(), settings.Revision, normalized)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, runtimecontrol.ErrRevisionConflict) {
			status = http.StatusConflict
		}
		runtimeControlsError(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, saved)
}

func (h *Handler) requireRuntimeSettingsStore(c *gin.Context) (runtimecontrol.SettingsStore, bool) {
	if h == nil || h.runtimeSettingsStore == nil {
		runtimeControlsError(c, http.StatusServiceUnavailable, "runtime controls store is not configured")
		return nil, false
	}
	return h.runtimeSettingsStore, true
}

func runtimeControlsError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
