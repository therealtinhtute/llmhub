package quotaalert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	codexCollectorBaseURL        = "https://chatgpt.com"
	codexUsagePath               = "/backend-api/wham/usage"
	codexResetCreditsPath        = "/backend-api/wham/rate-limit-reset-credits"
	codexFiveHourSeconds         = 18000
	codexWeeklyWindowSeconds     = 604800
	codexMonthlyWindowMinSeconds = 2419200
	codexMonthlyWindowMaxSeconds = 2678400
)

type CodexCollector struct {
	httpClient *CollectorHTTPClient
	refresh    CollectorRefreshFunc
	now        func() time.Time
}

type codexUsagePayload struct {
	RateLimit             *codexRateLimitInfo        `json:"rate_limit"`
	RateLimitCamel        *codexRateLimitInfo        `json:"rateLimit"`
	CodeReviewRateLimit   *codexRateLimitInfo        `json:"code_review_rate_limit"`
	CodeReviewLimitCamel  *codexRateLimitInfo        `json:"codeReviewRateLimit"`
	AdditionalRateLimits  []codexAdditionalRateLimit `json:"additional_rate_limits"`
	AdditionalLimitsCamel []codexAdditionalRateLimit `json:"additionalRateLimits"`
	ResetCredits          *codexResetCreditSummary   `json:"rate_limit_reset_credits"`
	ResetCreditsCamel     *codexResetCreditSummary   `json:"rateLimitResetCredits"`
}

type codexRateLimitInfo struct {
	Allowed         any               `json:"allowed"`
	LimitReached    any               `json:"limit_reached"`
	LimitReachedAlt any               `json:"limitReached"`
	PrimaryWindow   *codexUsageWindow `json:"primary_window"`
	PrimaryCamel    *codexUsageWindow `json:"primaryWindow"`
	SecondaryWindow *codexUsageWindow `json:"secondary_window"`
	SecondaryCamel  *codexUsageWindow `json:"secondaryWindow"`
}

type codexUsageWindow struct {
	UsedPercent        any `json:"used_percent"`
	UsedPercentCamel   any `json:"usedPercent"`
	LimitWindowSeconds any `json:"limit_window_seconds"`
	WindowSecondsCamel any `json:"limitWindowSeconds"`
	ResetAfterSeconds  any `json:"reset_after_seconds"`
	ResetAfterCamel    any `json:"resetAfterSeconds"`
	ResetAt            any `json:"reset_at"`
	ResetAtCamel       any `json:"resetAt"`
}

type codexAdditionalRateLimit struct {
	LimitName      any                 `json:"limit_name"`
	LimitNameCamel any                 `json:"limitName"`
	MeteredFeature any                 `json:"metered_feature"`
	MeteredCamel   any                 `json:"meteredFeature"`
	RateLimit      *codexRateLimitInfo `json:"rate_limit"`
	RateLimitCamel *codexRateLimitInfo `json:"rateLimit"`
}

type codexResetCreditSummary struct {
	AvailableCount any `json:"available_count"`
	AvailableCamel any `json:"availableCount"`
}

func NewCodexCollector(deps CollectorDependencies) (Collector, error) {
	client := deps.HTTPClient
	var err error
	if client == nil {
		client, err = NewCollectorHTTPClient(CollectorHTTPConfig{
			BaseURL:      codexCollectorBaseURL,
			AllowedHosts: []string{"chatgpt.com"},
		})
		if err != nil {
			return nil, err
		}
	}
	return &CodexCollector{httpClient: client, refresh: deps.Refresh, now: time.Now}, nil
}

func (c *CodexCollector) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("codex quota collector is not configured")
	}
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token", "id_token", "account_id"}, []string{"access_token", "id_token", "account_id"})
	if err != nil {
		return nil, err
	}
	accessToken, ok := snapshotString(cloned, "access_token")
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("codex quota collector access token is missing")
	}

	headersFor := func(a AuthSnapshot) map[string]string {
		token, _ := snapshotString(a, "access_token")
		headers := map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
			"User-Agent":    "codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal",
		}
		if accountID, ok := snapshotString(a, "account_id"); ok {
			headers["Chatgpt-Account-Id"] = accountID
		}
		return headers
	}

	var payload codexUsagePayload
	if err = c.httpClient.JSON(ctx, cloned, http.MethodGet, codexUsagePath, headersFor, &payload, c.refresh); err != nil {
		return nil, fmt.Errorf("codex quota usage request failed: %s", RedactCollectorError(err, cloned))
	}

	observedAt := c.now().UTC()
	observations := make([]Observation, 0, 6)
	observations = appendCodexWindows(observations, cloned, observedAt, "code", firstRateLimit(payload.RateLimit, payload.RateLimitCamel))
	observations = appendCodexWindows(observations, cloned, observedAt, "code-review", firstRateLimit(payload.CodeReviewRateLimit, payload.CodeReviewLimitCamel))

	additional := payload.AdditionalRateLimits
	if len(additional) == 0 {
		additional = payload.AdditionalLimitsCamel
	}
	for index, item := range additional {
		limit := firstRateLimit(item.RateLimit, item.RateLimitCamel)
		if limit == nil {
			continue
		}
		name, ok := stringFromAny(firstAny(item.LimitName, item.LimitNameCamel, item.MeteredFeature, item.MeteredCamel))
		if !ok {
			name = fmt.Sprintf("additional-%d", index+1)
		}
		slug := slugID(name)
		if slug == "" {
			slug = fmt.Sprintf("additional-%d", index+1)
		}
		observations = appendCodexWindows(observations, cloned, observedAt, "additional-"+slug, limit)
	}

	if len(observations) == 0 {
		return nil, fmt.Errorf("codex quota usage response contains no recognized windows")
	}

	var resetCredits json.RawMessage
	_ = c.httpClient.JSON(ctx, cloned, http.MethodGet, codexResetCreditsPath, headersFor, &resetCredits, c.refresh)
	return observations, nil
}

type namedCodexWindow struct {
	name   string
	window *codexUsageWindow
}

func appendCodexWindows(observations []Observation, auth AuthSnapshot, observedAt time.Time, resource string, limit *codexRateLimitInfo) []Observation {
	if limit == nil {
		return observations
	}
	primary := firstWindow(limit.PrimaryWindow, limit.PrimaryCamel)
	secondary := firstWindow(limit.SecondaryWindow, limit.SecondaryCamel)
	for _, item := range classifyCodexWindows(primary, secondary) {
		if item.window == nil {
			continue
		}
		observation, ok := buildCodexObservation(auth, observedAt, resource, item.name, item.window, limit)
		if ok {
			observations = append(observations, observation)
		}
	}
	return observations
}

// classifyCodexWindows names primary/secondary windows by duration:
// 5h → five-hour, 7d → weekly, 28–31d → monthly (mirrors isMonthlyWindow in
// quotaConfigs.ts). Windows with unrecognized or absent durations fall back
// to positional naming (primary→five-hour, secondary→weekly), and the
// fallback never relabels a window already claimed by duration.
func classifyCodexWindows(primary, secondary *codexUsageWindow) []namedCodexWindow {
	var fiveHour, weekly, monthly *codexUsageWindow
	for _, window := range []*codexUsageWindow{primary, secondary} {
		if window == nil {
			continue
		}
		seconds, ok := numberFromAny(firstAny(window.LimitWindowSeconds, window.WindowSecondsCamel))
		if !ok {
			continue
		}
		switch {
		case int(seconds) == codexFiveHourSeconds:
			if fiveHour == nil {
				fiveHour = window
			}
		case int(seconds) == codexWeeklyWindowSeconds:
			if weekly == nil {
				weekly = window
			}
		case seconds >= codexMonthlyWindowMinSeconds && seconds <= codexMonthlyWindowMaxSeconds:
			if monthly == nil {
				monthly = window
			}
		}
	}
	if fiveHour == nil && primary != nil && primary != weekly && primary != monthly {
		fiveHour = primary
	}
	if weekly == nil && secondary != nil && secondary != fiveHour && secondary != monthly {
		weekly = secondary
	}
	return []namedCodexWindow{
		{name: "five-hour", window: fiveHour},
		{name: "weekly", window: weekly},
		{name: "monthly", window: monthly},
	}
}

func buildCodexObservation(auth AuthSnapshot, observedAt time.Time, resource, window string, usageWindow *codexUsageWindow, limit *codexRateLimitInfo) (Observation, bool) {
	resetAt, resetKnown := parseUnixOrRelativeReset(
		firstAny(usageWindow.ResetAt, usageWindow.ResetAtCamel),
		firstAny(usageWindow.ResetAfterSeconds, usageWindow.ResetAfterCamel),
		observedAt,
	)
	usedPercent, usedKnown := numberFromAny(firstAny(usageWindow.UsedPercent, usageWindow.UsedPercentCamel))
	limitReached, limitReachedKnown := boolFromAny(firstAny(limit.LimitReached, limit.LimitReachedAlt))
	allowed, allowedKnown := boolFromAny(limit.Allowed)
	inferredExhausted := resetKnown && ((limitReachedKnown && limitReached) || (allowedKnown && !allowed))
	if !usedKnown {
		if !inferredExhausted {
			return Observation{}, false
		}
		usedPercent = 100
	}
	remaining, err := NormalizePercentage(100 - usedPercent)
	if err != nil {
		return Observation{}, false
	}
	observation, err := (Observation{
		Identity: StateIdentity{
			AuthID:   auth.AuthID(),
			Provider: ProviderCodex,
			Resource: resource,
			Window:   window,
		},
		AuthLabel:           auth.RedactedLabel(),
		Health:              CollectionReliable,
		Remaining:           remaining,
		RemainingKnown:      true,
		ExplicitlyExhausted: remaining == 0 && (usedKnown || inferredExhausted),
		ResetAt:             resetAt,
		ResetKnown:          resetKnown,
		ObservedAt:          observedAt,
	}).Normalize()
	if err != nil {
		return Observation{}, false
	}
	return observation, true
}

func firstRateLimit(left, right *codexRateLimitInfo) *codexRateLimitInfo {
	if left != nil {
		return left
	}
	return right
}

func firstWindow(left, right *codexUsageWindow) *codexUsageWindow {
	if left != nil {
		return left
	}
	return right
}

func firstAny(values ...any) any {
	for _, value := range values {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		return value
	}
	return nil
}
