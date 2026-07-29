package quotaalert

import (
	"context"
	"fmt"
	"net/http"
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

type claudeUsagePayload map[string]claudeUsageWindow

type claudeUsageWindow struct {
	Utilization any    `json:"utilization"`
	ResetsAt    string `json:"resets_at"`
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
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token"}, nil)
	if err != nil {
		return nil, err
	}
	accessToken, ok := cloned.Attribute("access_token")
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("claude quota collector access token is missing")
	}

	headers := map[string]string{
		"Authorization":  "Bearer " + accessToken,
		"Content-Type":   "application/json",
		"anthropic-beta": "oauth-2025-04-20",
	}
	var payload claudeUsagePayload
	if err = c.httpClient.JSON(ctx, cloned, http.MethodGet, claudeUsagePath, headers, &payload, c.refresh); err != nil {
		return nil, fmt.Errorf("claude quota usage request failed: %s", RedactCollectorError(err, cloned))
	}

	observedAt := c.now().UTC()
	observations := make([]Observation, 0, len(claudeUsageWindows))
	for _, meta := range claudeUsageWindows {
		window, exists := payload[meta.key]
		if !exists {
			continue
		}
		usedPercent, ok := numberFromAny(window.Utilization)
		if !ok {
			continue
		}
		remaining, err := NormalizePercentage(100 - usedPercent)
		if err != nil {
			return nil, err
		}
		resetAt, resetKnown := parseProviderTime(window.ResetsAt, observedAt)
		observation, err := (Observation{
			Identity: StateIdentity{
				AuthID:   cloned.AuthID(),
				Provider: ProviderClaude,
				Resource: meta.resource,
				Window:   meta.window,
			},
			AuthLabel:           cloned.RedactedLabel(),
			Health:              CollectionReliable,
			Remaining:           remaining,
			RemainingKnown:      true,
			ExplicitlyExhausted: remaining == 0,
			ResetAt:             resetAt,
			ResetKnown:          resetKnown,
			ObservedAt:          observedAt,
		}).Normalize()
		if err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	if len(observations) == 0 {
		return nil, fmt.Errorf("claude quota usage response contains no recognized windows")
	}

	var ignored map[string]any
	_ = c.httpClient.JSON(ctx, cloned, http.MethodGet, claudeProfilePath, headers, &ignored, c.refresh)
	return observations, nil
}
