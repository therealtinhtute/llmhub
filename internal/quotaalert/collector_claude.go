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
	claudeCollectorBaseURL = "https://api.anthropic.com"
	claudeUsagePath        = "/api/oauth/usage"
	claudeProfilePath      = "/api/oauth/profile"
)

var claudeUsageWindows = []struct {
	key      string
	resource string
	window   string
}{
	{key: "five_hour", resource: "messages", window: "five-hour"},
	{key: "seven_day", resource: "messages", window: "seven-day"},
	{key: "seven_day_oauth_apps", resource: "oauth-apps", window: "seven-day"},
	{key: "seven_day_opus", resource: "opus", window: "seven-day"},
	{key: "seven_day_sonnet", resource: "sonnet", window: "seven-day"},
	{key: "seven_day_cowork", resource: "cowork", window: "seven-day"},
	{key: "iguana_necktie", resource: "iguana-necktie", window: "default"},
}

type ClaudeCollector struct {
	httpClient *CollectorHTTPClient
	refresh    CollectorRefreshFunc
	now        func() time.Time
}

// claudeUsagePayload decodes the usage response key-by-key so the top-level
// `limits` array (fable plan limits) cannot fail the whole unmarshal — it did
// when the payload was map[string]claudeUsageWindow. Unknown future keys
// (e.g. the 2026-07-05 rate_limit_info drift) decode-tolerantly: a payload
// with no recognized windows still errors downstream rather than reporting
// silent healthy data.
type claudeUsagePayload struct {
	windows map[string]claudeUsageWindow
	limits  []claudeUsageLimit
}

func (p *claudeUsagePayload) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.windows = make(map[string]claudeUsageWindow, len(raw))
	for key, value := range raw {
		if key == "limits" {
			var limits []claudeUsageLimit
			if err := json.Unmarshal(value, &limits); err != nil {
				return fmt.Errorf("decode claude usage limits: %w", err)
			}
			p.limits = limits
			continue
		}
		var window claudeUsageWindow
		if err := json.Unmarshal(value, &window); err != nil {
			continue
		}
		p.windows[key] = window
	}
	return nil
}

type claudeUsageWindow struct {
	Utilization any    `json:"utilization"`
	ResetsAt    string `json:"resets_at"`
}

// claudeUsageLimit mirrors the limits[] entries consumed by findFableUsageLimit
// in web/src/components/quota/quotaConfigs.ts (fable weekly-scoped limits).
type claudeUsageLimit struct {
	Kind     any `json:"kind"`
	IsActive any `json:"is_active"`
	Percent  any `json:"percent"`
	ResetsAt any `json:"resets_at"`
	Scope    struct {
		Model struct {
			DisplayName any `json:"display_name"`
		} `json:"model"`
	} `json:"scope"`
}

func NewClaudeCollector(deps CollectorDependencies) (Collector, error) {
	client := deps.HTTPClient
	var err error
	if client == nil {
		client, err = NewCollectorHTTPClient(CollectorHTTPConfig{
			BaseURL:      claudeCollectorBaseURL,
			AllowedHosts: []string{"api.anthropic.com"},
		})
		if err != nil {
			return nil, err
		}
	}
	return &ClaudeCollector{httpClient: client, refresh: deps.Refresh, now: time.Now}, nil
}

func (c *ClaudeCollector) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("claude quota collector is not configured")
	}
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token"}, []string{"access_token"})
	if err != nil {
		return nil, err
	}
	accessToken, ok := snapshotString(cloned, "access_token")
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("claude quota collector access token is missing")
	}

	headersFor := func(a AuthSnapshot) map[string]string {
		token, _ := snapshotString(a, "access_token")
		return map[string]string{
			"Authorization":  "Bearer " + token,
			"Content-Type":   "application/json",
			"anthropic-beta": "oauth-2025-04-20",
		}
	}
	var payload claudeUsagePayload
	if err = c.httpClient.JSON(ctx, cloned, http.MethodGet, claudeUsagePath, headersFor, &payload, c.refresh); err != nil {
		return nil, fmt.Errorf("claude quota usage request failed: %s", RedactCollectorError(err, cloned))
	}

	observedAt := c.now().UTC()
	fableLimit := findClaudeFableLimit(payload.limits)
	observations := make([]Observation, 0, len(claudeUsageWindows)+1)
	for _, meta := range claudeUsageWindows {
		// The quota cards treat iguana_necktie as the legacy alias for the
		// fable weekly limit — skip it when the limits[] entry exists so both
		// screens render the same single fable window.
		if meta.key == "iguana_necktie" && fableLimit != nil {
			continue
		}
		window, exists := payload.windows[meta.key]
		if !exists {
			continue
		}
		usedPercent, ok := numberFromAny(window.Utilization)
		if !ok {
			continue
		}
		observation, ok := buildClaudeObservation(cloned, observedAt, meta.resource, meta.window, usedPercent, window.ResetsAt)
		if !ok {
			return nil, fmt.Errorf("claude quota usage window %q is malformed", meta.key)
		}
		observations = append(observations, observation)
	}
	if fableLimit != nil {
		usedPercent, _ := numberFromAny(fableLimit.Percent)
		resetsAt, _ := stringFromAny(fableLimit.ResetsAt)
		observation, ok := buildClaudeObservation(cloned, observedAt, "fable", "seven-day", usedPercent, resetsAt)
		if !ok {
			return nil, fmt.Errorf("claude quota usage fable limit is malformed")
		}
		observations = append(observations, observation)
	}
	if len(observations) == 0 {
		return nil, fmt.Errorf("claude quota usage response contains no recognized windows")
	}

	var ignored map[string]any
	_ = c.httpClient.JSON(ctx, cloned, http.MethodGet, claudeProfilePath, headersFor, &ignored, c.refresh)
	return observations, nil
}

// findClaudeFableLimit mirrors findFableUsageLimit (quotaConfigs.ts): a fable
// limit is weekly_scoped, scoped to model display_name "fable"/"fable 5", and
// carries a percent. The first is_active==true candidate wins; otherwise the
// first candidate is used.
func findClaudeFableLimit(limits []claudeUsageLimit) *claudeUsageLimit {
	var first *claudeUsageLimit
	for index := range limits {
		limit := &limits[index]
		kind, _ := stringFromAny(limit.Kind)
		if !strings.EqualFold(strings.TrimSpace(kind), "weekly_scoped") {
			continue
		}
		name, _ := stringFromAny(limit.Scope.Model.DisplayName)
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "fable" && name != "fable 5" {
			continue
		}
		if _, ok := numberFromAny(limit.Percent); !ok {
			continue
		}
		if first == nil {
			first = limit
		}
		if active, ok := boolFromAny(limit.IsActive); ok && active {
			return limit
		}
	}
	return first
}

func buildClaudeObservation(auth AuthSnapshot, observedAt time.Time, resource, window string, usedPercent float64, resetsAt string) (Observation, bool) {
	remaining, err := NormalizePercentage(100 - usedPercent)
	if err != nil {
		return Observation{}, false
	}
	resetAt, resetKnown := parseProviderTime(resetsAt, observedAt)
	observation, err := (Observation{
		Identity: StateIdentity{
			AuthID:   auth.AuthID(),
			Provider: ProviderClaude,
			Resource: resource,
			Window:   window,
		},
		AuthLabel:           auth.RedactedLabel(),
		Health:              CollectionReliable,
		Remaining:           remaining,
		RemainingKnown:      true,
		ExplicitlyExhausted: remaining == 0,
		ResetAt:             resetAt,
		ResetKnown:          resetKnown,
		ObservedAt:          observedAt,
	}).Normalize()
	if err != nil {
		return Observation{}, false
	}
	return observation, true
}
