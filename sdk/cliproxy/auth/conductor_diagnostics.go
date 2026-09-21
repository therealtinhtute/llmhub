package auth

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/therealtinhtute/llmhub/internal/logging"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/util"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// Warn-level credential diagnostics for auth cooldown and upstream execution failures,
// ported from upstream 48749717645e. These identify the affected credential without
// leaking secret material so operators can see why requests failed over or stalled.

func isAuthUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	var authErr *Error
	if errors.As(err, &authErr) && authErr != nil {
		return authErr.Code == "auth_unavailable" || authErr.Code == "model_cooldown"
	}
	var cooldownErr *modelCooldownError
	return errors.As(err, &cooldownErr) && cooldownErr != nil
}

func cooldownReason(statusMessage string, quota QuotaState, lastErr *Error) string {
	if reason := strings.TrimSpace(quota.Reason); reason != "" {
		return reason
	}
	if statusMessage = strings.TrimSpace(statusMessage); statusMessage != "" {
		return statusMessage
	}
	if lastErr != nil {
		if code := strings.TrimSpace(lastErr.Code); code != "" {
			return code
		}
		if message := strings.TrimSpace(lastErr.Message); message != "" {
			return message
		}
	}
	return ""
}

func formatAuthIdentity(auth *Auth, provider string) string {
	if auth == nil {
		return "auth=nil"
	}
	accountType, accountInfo := auth.AccountInfo()
	switch accountType {
	case "api_key":
		return fmt.Sprintf("api_key=%s", util.HideAPIKey(accountInfo))
	case "oauth":
		return formatOauthIdentity(auth, provider, accountInfo)
	default:
		if auth.FileName != "" {
			return fmt.Sprintf("auth_file=%s", filepath.Base(auth.FileName))
		}
		if auth.ID != "" {
			return fmt.Sprintf("auth_id=%s", auth.ID)
		}
		if accountInfo != "" {
			return accountInfo
		}
		return "unknown"
	}
}

// warnLogHomeCredentialFailure emits a warn-level diagnostic when a Home-owned
// credential operation (e.g. request auth preparation) fails.
// Ported from upstream CLIProxyAPI commits 9a2201c36a0a and e4a8f9891344.
func warnLogHomeCredentialFailure(ctx context.Context, operation, provider string, auth *Auth, err error) {
	if err == nil {
		return
	}
	provider = strings.TrimSpace(provider)
	if provider == "" && auth != nil {
		provider = strings.TrimSpace(auth.Provider)
	}
	fields := log.Fields{
		"auth":      formatAuthIdentity(auth, provider),
		"operation": operation,
		"provider":  provider,
	}
	if statusCode := statusCodeFromError(err); statusCode != 0 {
		fields["status"] = statusCode
	}
	logEntryWithRequestID(ctx).WithFields(fields).Warnf("Home credential operation failed: err=%s", safeErrorDiagnosticForLog(err))
}

// safeErrorDiagnosticForLog prefers a marked LogDiagnostic() payload (already
// sanitized by the producer) and otherwise redacts the free-form error text.
// Ported from upstream CLIProxyAPI commit e4a8f9891344.
func safeErrorDiagnosticForLog(err error) string {
	if err == nil {
		return ""
	}
	diagnostic := err.Error()
	type logDiagnosticError interface {
		LogDiagnostic() string
	}
	var diagnosticErr logDiagnosticError
	if errors.As(err, &diagnosticErr) && diagnosticErr != nil {
		if markedDiagnostic := strings.TrimSpace(diagnosticErr.LogDiagnostic()); markedDiagnostic != "" {
			diagnostic = markedDiagnostic
		}
	}
	return logging.SafeDiagnosticForLog(diagnostic)
}

func warnLogUpstreamFailure(ctx context.Context, entry *log.Entry, provider, model string, auth *Auth, duration time.Duration, err error) {
	if err == nil {
		return
	}
	if ctx != nil && errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	if isRequestInvalidError(err) {
		return
	}
	if entry == nil {
		if ctx != nil {
			entry = logEntryWithRequestID(ctx)
		} else {
			entry = log.NewEntry(log.StandardLogger())
		}
	}
	authIdent := formatAuthIdentity(auth, provider)
	errSummary := safeErrorDiagnosticForLog(err)
	duration = duration.Round(time.Millisecond)
	if statusCode := statusCodeFromError(err); statusCode != 0 {
		entry.Warnf("%3d | %13v | upstream execution failed: provider=%s model=%s auth=%s err=%s", statusCode, duration, provider, model, authIdent, errSummary)
		return
	}
	entry.Warnf("upstream execution failed: provider=%s model=%s auth=%s duration=%s err=%s", provider, model, authIdent, duration, errSummary)
}

func authCoolingSummary(auth *Auth, model string, next time.Time, now time.Time) string {
	if auth == nil {
		return ""
	}
	ident := formatAuthIdentity(auth, auth.Provider)
	reason := ""
	if model != "" && len(auth.ModelStates) > 0 {
		if state, ok := auth.ModelStates[model]; ok && state != nil {
			reason = cooldownReason(state.StatusMessage, state.Quota, state.LastError)
		} else if state, ok := auth.ModelStates[canonicalModelKey(model)]; ok && state != nil {
			reason = cooldownReason(state.StatusMessage, state.Quota, state.LastError)
		}
	}
	if reason == "" {
		reason = cooldownReason(auth.StatusMessage, auth.Quota, auth.LastError)
	}
	if reason == "" {
		reason = "cooldown"
	}
	remaining := "0s"
	if !next.IsZero() && next.After(now) {
		remaining = next.Sub(now).Round(time.Second).String()
	}
	return fmt.Sprintf("[%s, reason=%s, remaining=%s]", ident, reason, remaining)
}

// warnLogAuthUnavailable emits a warn-level diagnostic when an auth selection fails
// because candidate credentials are cooling down, naming each affected credential.
func (m *Manager) warnLogAuthUnavailable(ctx context.Context, providers []string, model string, opts cliproxyexecutor.Options, tried map[string]struct{}, err error) {
	if m == nil || err == nil || !isAuthUnavailableError(err) {
		return
	}
	now := time.Now()
	m.mu.RLock()
	defer m.mu.RUnlock()
	pinnedAuthID := pinnedAuthIDFromMetadata(opts.Metadata)
	providerSet := make(map[string]struct{}, len(providers))
	for _, p := range providers {
		if norm := strings.TrimSpace(strings.ToLower(p)); norm != "" && norm != "mixed" {
			providerSet[norm] = struct{}{}
		}
	}
	registryRef := registry.GetGlobalRegistry()

	coolingSummaries := make([]string, 0)
	totalCandidates := 0
	for _, candidate := range m.auths {
		if candidate == nil || candidate.Disabled {
			continue
		}
		providerKey := executorKeyFromAuth(candidate)
		if len(providerSet) > 0 {
			if _, ok := providerSet[providerKey]; !ok {
				continue
			}
		}
		if _, ok := m.executors[providerKey]; !ok {
			continue
		}
		if pinnedAuthID != "" && candidate.ID != pinnedAuthID {
			continue
		}
		if tried != nil {
			if _, used := tried[candidate.ID]; used {
				continue
			}
		}
		if model != "" && !m.authSupportsRouteModel(registryRef, candidate, model) {
			continue
		}
		totalCandidates++
		checkModel := m.selectionModelForAuth(candidate, model)
		blocked, reason, next := isAuthBlockedForModel(candidate, checkModel, now)
		if blocked && reason == blockReasonCooldown {
			coolingSummaries = append(coolingSummaries, authCoolingSummary(candidate, checkModel, next, now))
		}
	}

	if len(coolingSummaries) > 0 {
		sort.Strings(coolingSummaries)
		entry := logEntryWithRequestID(ctx)
		providerText := strings.Join(providers, ",")
		if len(providers) == 1 {
			entry.Warnf("auth unavailable: %d of %d candidate(s) for model %q (provider=%s) are in cooldown: %s", len(coolingSummaries), totalCandidates, model, providerText, strings.Join(coolingSummaries, ", "))
		} else {
			entry.Warnf("auth unavailable: %d of %d candidate(s) for model %q (providers=%s) are in cooldown: %s", len(coolingSummaries), totalCandidates, model, providerText, strings.Join(coolingSummaries, ", "))
		}
	}
}
