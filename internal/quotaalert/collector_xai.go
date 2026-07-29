package quotaalert

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	xaiCollectorBaseURL = "https://cli-chat-proxy.grok.com"
	xaiBillingPath      = "/v1/billing"
)

type XAICollector struct {
	httpClient *CollectorHTTPClient
	refresh    CollectorRefreshFunc
	now        func() time.Time
}

type xaiBillingPayload struct {
	Config *xaiBillingConfig `json:"config"`
}

type xaiBillingConfig struct {
	MonthlyLimit    any `json:"monthlyLimit"`
	MonthlyLimitAlt any `json:"monthly_limit"`
	Used            any `json:"used"`
	OnDemandCap     any `json:"onDemandCap"`
	OnDemandCapAlt  any `json:"on_demand_cap"`
	PeriodStart     any `json:"billingPeriodStart"`
	PeriodStartAlt  any `json:"billing_period_start"`
	PeriodEnd       any `json:"billingPeriodEnd"`
	PeriodEndAlt    any `json:"billing_period_end"`
}

func NewXAICollector(deps CollectorDependencies) (Collector, error) {
	client := deps.HTTPClient
	var err error
	if client == nil {
		client, err = NewCollectorHTTPClient(CollectorHTTPConfig{
			BaseURL:      xaiCollectorBaseURL,
			AllowedHosts: []string{"cli-chat-proxy.grok.com"},
		})
		if err != nil {
			return nil, err
		}
	}
	return &XAICollector{httpClient: client, refresh: deps.Refresh, now: time.Now}, nil
}

func (c *XAICollector) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("xAI quota collector is not configured")
	}
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token"}, []string{"access_token"})
	if err != nil {
		return nil, err
	}
	accessToken, ok := snapshotString(cloned, "access_token")
	if !ok {
		return nil, fmt.Errorf("xAI quota collector access token is missing")
	}
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	var payload xaiBillingPayload
	if err = c.httpClient.JSON(ctx, cloned, http.MethodGet, xaiBillingPath, headers, &payload, c.refresh); err != nil {
		return nil, fmt.Errorf("xAI quota request failed: %s", RedactCollectorError(err, cloned))
	}
	observation, ok := buildXAIObservation(cloned, payload.Config, c.now().UTC())
	if !ok {
		return nil, fmt.Errorf("xAI quota response contains no recognized billing summary")
	}
	return []Observation{observation}, nil
}

func buildXAIObservation(auth AuthSnapshot, config *xaiBillingConfig, observedAt time.Time) (Observation, bool) {
	if config == nil {
		return Observation{}, false
	}
	monthlyLimit, monthlyKnown := xaiCentValue(firstAny(config.MonthlyLimit, config.MonthlyLimitAlt))
	used, usedKnown := xaiCentValue(config.Used)
	periodEnd, _ := stringFromAny(firstAny(config.PeriodEnd, config.PeriodEndAlt))
	if !monthlyKnown && !usedKnown && periodEnd == "" {
		return Observation{}, false
	}
	remainingKnown := false
	remaining := Percentage(0)
	if monthlyKnown && monthlyLimit > 0 && usedKnown {
		remainingKnown = true
		value, err := NormalizePercentage(100 - ((used / monthlyLimit) * 100))
		if err != nil {
			return Observation{}, false
		}
		remaining = value
	}
	if !remainingKnown {
		return Observation{}, false
	}
	resetAt, resetKnown := parseProviderTime(periodEnd, observedAt)
	observation, err := (Observation{
		Identity: StateIdentity{
			AuthID:   auth.AuthID(),
			Provider: ProviderXAI,
			Resource: "monthly-credits",
			Window:   "billing-period",
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

func xaiCentValue(value any) (float64, bool) {
	if object, ok := value.(map[string]any); ok {
		return numberFromAny(object["val"])
	}
	return numberFromAny(value)
}
