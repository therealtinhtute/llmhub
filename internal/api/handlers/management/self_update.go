package management

import (
	"context"
	"errors"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/updater"
)

// SelfUpdateEngine stages a verified release candidate (R13). Implemented by
// *updater.Engine; an interface so tests can fake the full outcome space,
// including the unsupported-platform result.
type SelfUpdateEngine interface {
	StageLatest(ctx context.Context) (updater.StagedManifest, error)
}

// restartCommand is the single fixed command the apply endpoint may run
// (R13, R14): the service user restarts only its own unit, matching the
// sudoers drop-in installed by scripts/install-local.sh. Never derived from
// request input.
var restartCommand = []string{"sudo", "-n", "/usr/bin/systemctl", "restart", "llmhub.service"}

// restartRunnerTimeout bounds the sudo/systemctl invocation.
const restartRunnerTimeout = 30 * time.Second

// SelfUpdateStage implements POST /v0/management/self-update: it runs the
// engine's discover-verify-probe-stage path and returns a typed outcome the
// panel renders. Remote response bodies are never surfaced (R13).
func (h *Handler) SelfUpdateStage(c *gin.Context) {
	engine := h.selfUpdateEngine
	if engine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "reason": "self-update not configured"})
		return
	}
	h.selfUpdateMu.Lock()
	manifest, err := engine.StageLatest(c.Request.Context())
	h.selfUpdateMu.Unlock()
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "staged", "version": manifest.Version})
	case errors.Is(err, updater.ErrUpToDate):
		c.JSON(http.StatusOK, gin.H{"status": "current"})
	case errors.Is(err, updater.ErrUnsupportedPlatform):
		c.JSON(http.StatusOK, gin.H{"status": "unsupported"})
	case errors.Is(err, updater.ErrDowngradeRefused):
		c.JSON(http.StatusOK, gin.H{"status": "rejected", "reason": "running version is newer than stable"})
	case errors.Is(err, updater.ErrDevelopmentBuild):
		c.JSON(http.StatusOK, gin.H{"status": "rejected", "reason": "development build"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "error", "reason": "stage failed"})
	}
}

// SelfUpdateApply implements POST /v0/management/self-update/apply: it
// responds before triggering the restart, since systemctl restart terminates
// this process (R13). The runner is an injected seam; when unset it invokes
// the single fixed restartCommand.
func (h *Handler) SelfUpdateApply(c *gin.Context) {
	runner := h.selfUpdateRunner
	if runner == nil {
		runner = defaultRestartRunner
	}
	// Respond first: the process may be gone before the runner returns.
	c.JSON(http.StatusAccepted, gin.H{"status": "restarting"})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), restartRunnerTimeout)
		defer cancel()
		if err := runner(ctx); err != nil {
			log.WithError(err).Error("self-update restart failed")
		}
	}()
}

// defaultRestartRunner executes the fixed restartCommand without a shell.
func defaultRestartRunner(ctx context.Context) error {
	return exec.CommandContext(ctx, restartCommand[0], restartCommand[1:]...).Run()
}
