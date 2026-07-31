package codexlive

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/therealtinhtute/llmhub/internal/client/codex/live"
	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

type Handler struct {
	authManager *coreauth.Manager
	settings    runtimecontrol.SettingsStore
	sessions    *live.Store
}

func New(authManager *coreauth.Manager, settings runtimecontrol.SettingsStore) *Handler {
	return &Handler{
		authManager: authManager,
		settings:    settings,
		sessions:    live.NewStore(),
	}
}

func (h *Handler) CreateCall(c *gin.Context) {
	if !h.enabled(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Codex Live is disabled"})
		return
	}
	auth := h.codexAuth()
	if auth == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no Codex credential available"})
		return
	}

	body, err := live.ReadBody(c.Request.Body)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, live.ErrBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	payload, contentType, model, err := live.PrepareCallRequest(body, c.GetHeader("Content-Type"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	headers := live.ProtocolHeaders(c.Request.Header)
	headers.Set("Content-Type", contentType)
	upstreamReq, err := h.authManager.NewHttpRequest(c.Request.Context(), auth, http.MethodPost, live.UpstreamCallURL, payload, headers)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.authManager.HttpRequest(c.Request.Context(), auth, upstreamReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	responseHeaders := live.CallResponseHeaders(resp.Header)
	live.WriteResponseHeaders(c.Writer.Header(), responseHeaders)
	if callID := live.CallIDFromLocation(responseHeaders.Get("Location")); callID != "" {
		h.sessions.Put(callID, live.Session{AuthID: auth.ID, Model: model})
	}
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func (h *Handler) Sideband(c *gin.Context) {
	if !h.enabled(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Codex Live is disabled"})
		return
	}
	_, callID, ok := live.SidebandTarget(c.Request.URL.Path, map[string]string{"call_id": c.Param("call_id")}, c.Request.URL.Query())
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid Codex Live call ID"})
		return
	}
	if _, ok := h.sessions.Peek(callID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Codex Live session not found"})
		return
	}
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Codex Live media relay is not implemented yet"})
}

func (h *Handler) enabled(c *gin.Context) bool {
	if h == nil || h.settings == nil || h.authManager == nil {
		return false
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	settings, err := h.settings.LoadRuntimeSettings(ctx)
	return err == nil && settings.CodexLive.Enabled
}

func (h *Handler) codexAuth() *coreauth.Auth {
	if h == nil || h.authManager == nil {
		return nil
	}
	for _, auth := range h.authManager.List() {
		if auth == nil || auth.Disabled || auth.Unavailable || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
			continue
		}
		return auth
	}
	return nil
}
