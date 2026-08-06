package management

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/therealtinhtute/llmhub/internal/config/presets"
)

// GetProviderPresets returns the curated, read-only catalog of known
// OpenAI-compatible providers used to prefill the provider-add screen.
func (h *Handler) GetProviderPresets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"presets": presets.All()})
}
