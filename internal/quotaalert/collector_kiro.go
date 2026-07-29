package quotaalert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/therealtinhtute/llmhub/internal/auth/kiro"
	"github.com/therealtinhtute/llmhub/internal/runtime/executor"
)

const (
	kiroUsageLimitsPath             = "/getUsageLimits"
	kiroCodeWhispererUsageBaseURL   = "https://codewhisperer.us-east-1.amazonaws.com"
	kiroCodeWhispererGetUsageTarget = "AmazonCodeWhispererService.GetUsageLimits"
)

var kiroRegionPattern = regexp.MustCompile(`^[a-z]{2}-[a-z]+-[0-9]+$`)

type KiroCollector struct {
	httpClients []*CollectorHTTPClient
	refresh     CollectorRefreshFunc
	now         func() time.Time
}

type kiroUsageAttempt struct {
	client  *CollectorHTTPClient
	method  string
	path    string
	headers map[string]string
	body    any
}

func NewKiroCollector(deps CollectorDependencies) (Collector, error) {
	if deps.HTTPClient != nil {
		return &KiroCollector{httpClients: []*CollectorHTTPClient{deps.HTTPClient}, refresh: deps.Refresh, now: time.Now}, nil
	}
	client, err := NewCollectorHTTPClient(CollectorHTTPConfig{
		BaseURL:      kiroCodeWhispererUsageBaseURL,
		AllowedHosts: []string{"codewhisperer.us-east-1.amazonaws.com"},
	})
	if err != nil {
		return nil, err
	}
	return &KiroCollector{httpClients: []*CollectorHTTPClient{client}, refresh: deps.Refresh, now: time.Now}, nil
}

func (c *KiroCollector) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	if c == nil || len(c.httpClients) == 0 {
		return nil, fmt.Errorf("kiro quota collector is not configured")
	}
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token", "profile_arn", "region"}, []string{"access_token", "profile_arn", "region"})
	if err != nil {
		return nil, err
	}
	accessToken, ok := snapshotString(cloned, "access_token")
	if !ok {
		return nil, fmt.Errorf("kiro quota collector access token is missing")
	}
	region := kiroCollectorRegion(cloned)
	qClient := c.kiroQClient()
	if qClient == nil {
		qClient, err = NewCollectorHTTPClient(CollectorHTTPConfig{
			BaseURL:      fmt.Sprintf("https://q.%s.amazonaws.com", region),
			AllowedHosts: []string{fmt.Sprintf("q.%s.amazonaws.com", region)},
		})
		if err != nil {
			return nil, err
		}
	}

	attempts := c.usageAttempts(cloned, accessToken, qClient)
	observedAt := c.now().UTC()
	var lastErr error
	for _, attempt := range attempts {
		var payload json.RawMessage
		if attempt.body == nil {
			err = attempt.client.JSON(ctx, cloned, attempt.method, attempt.path, attempt.headers, &payload, c.refresh)
		} else {
			err = attempt.client.JSONBody(ctx, cloned, attempt.method, attempt.path, attempt.headers, attempt.body, &payload, c.refresh)
		}
		if err != nil {
			lastErr = err
			continue
		}
		quota, err := executor.ParseKiroUsageLimits(payload, observedAt)
		if err != nil {
			lastErr = err
			continue
		}
		observations := buildKiroObservations(cloned, quota, observedAt)
		if len(observations) > 0 {
			return observations, nil
		}
		lastErr = fmt.Errorf("kiro quota response contains no recognized rows")
	}
	if lastErr != nil {
		return nil, fmt.Errorf("kiro quota request failed: %s", RedactCollectorError(lastErr, cloned))
	}
	return nil, fmt.Errorf("kiro quota response contains no recognized rows")
}

func (c *KiroCollector) kiroQClient() *CollectorHTTPClient {
	if c == nil || len(c.httpClients) < 2 {
		return nil
	}
	return c.httpClients[1]
}

func (c *KiroCollector) usageAttempts(auth AuthSnapshot, accessToken string, qClient *CollectorHTTPClient) []kiroUsageAttempt {
	codeWhispererClient := c.httpClients[0]
	headers := kiroUsageHeaders(accessToken)
	qPath := kiroUsagePathWithQuery(auth, true)
	postBody := map[string]any{
		"origin":       "AI_EDITOR",
		"resourceType": "AGENTIC_REQUEST",
	}
	if profileARN, ok := snapshotString(auth, "profile_arn"); ok {
		postBody["profileArn"] = profileARN
	}
	postHeaders := kiroUsageHeaders(accessToken)
	postHeaders["Content-Type"] = "application/x-amz-json-1.0"
	postHeaders["x-amz-target"] = kiroCodeWhispererGetUsageTarget
	return []kiroUsageAttempt{
		{client: codeWhispererClient, method: http.MethodGet, path: kiroUsagePathWithQuery(auth, false), headers: headers},
		{client: codeWhispererClient, method: http.MethodPost, path: "/", headers: postHeaders, body: postBody},
		{client: qClient, method: http.MethodGet, path: qPath, headers: headers},
	}
}

func buildKiroObservations(auth AuthSnapshot, quota executor.KiroQuotaState, observedAt time.Time) []Observation {
	observations := make([]Observation, 0, len(quota.Quotas))
	for _, row := range quota.Quotas {
		resource := strings.TrimSpace(row.ID)
		if resource == "" {
			resource = strings.TrimSpace(row.ResourceType)
		}
		if resource == "" {
			resource = strings.TrimSpace(row.Name)
		}
		resource = slugID(resource)
		if resource == "" {
			continue
		}
		remaining, ok := kiroRowRemaining(row)
		if !ok {
			continue
		}
		resetAt := time.Time{}
		resetKnown := false
		if row.ResetAt != nil && !row.ResetAt.IsZero() {
			resetAt = row.ResetAt.UTC()
			resetKnown = true
		} else if quota.NextResetAt != nil && !quota.NextResetAt.IsZero() {
			resetAt = quota.NextResetAt.UTC()
			resetKnown = true
		}
		observation, err := (Observation{
			Identity: StateIdentity{
				AuthID:   auth.AuthID(),
				Provider: ProviderKiro,
				Resource: resource,
				Window:   "default",
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
		if err == nil {
			observations = append(observations, observation)
		}
	}
	return observations
}

func kiroRowRemaining(row executor.KiroQuotaRow) (Percentage, bool) {
	if row.Unlimited {
		return 100, true
	}
	if row.RemainingPercent != nil {
		value, err := NormalizePercentage(*row.RemainingPercent)
		return value, err == nil
	}
	if row.Percent != nil {
		value, err := NormalizePercentage(100 - *row.Percent)
		return value, err == nil
	}
	if row.Remaining != nil && row.Limit != nil && *row.Limit > 0 {
		value, err := NormalizePercentage((*row.Remaining / *row.Limit) * 100)
		return value, err == nil
	}
	current := row.Current
	if current == nil {
		current = row.Used
	}
	limit := row.Limit
	if limit == nil {
		limit = row.Total
	}
	if current == nil || limit == nil {
		return 0, false
	}
	if *limit <= 0 {
		if *current > 0 {
			return 0, true
		}
		return 0, false
	}
	value, err := NormalizePercentage(((*limit - *current) / *limit) * 100)
	return value, err == nil
}

func kiroUsageHeaders(accessToken string) map[string]string {
	return map[string]string{
		"Authorization":               "Bearer " + accessToken,
		"Accept":                      "application/json",
		"User-Agent":                  "aws-sdk-js/1.0.0 ua/2.1 os/windows#10.0.26200 lang/js md/nodejs#22.21.1 api/codewhispererruntime#1.0.0 m/N,E KiroIDE-0.10.32-quota-alert",
		"X-Amz-User-Agent":            "aws-sdk-js/1.0.0 KiroIDE-0.10.32-quota-alert",
		"X-Amzn-Codewhisperer-Optout": "true",
		"X-Amzn-Kiro-Agent-Mode":      "vibe",
		"Amz-Sdk-Request":             "attempt=1; max=1",
		"Amz-Sdk-Invocation-Id":       "quota-alert",
	}
}

func kiroUsagePathWithQuery(auth AuthSnapshot, includeProfileARN bool) string {
	query := "origin=AI_EDITOR&resourceType=AGENTIC_REQUEST&isEmailRequired=true"
	if profileARN, ok := snapshotString(auth, "profile_arn"); includeProfileARN && ok {
		query += "&profileArn=" + urlQueryEscape(profileARN)
	}
	return kiroUsageLimitsPath + "?" + query
}

func kiroCollectorRegion(auth AuthSnapshot) string {
	if profileARN, ok := snapshotString(auth, "profile_arn"); ok {
		parts := strings.Split(profileARN, ":")
		if len(parts) >= 4 && kiroRegionPattern.MatchString(strings.TrimSpace(parts[3])) {
			return strings.TrimSpace(parts[3])
		}
	}
	if region, ok := snapshotString(auth, "region"); ok && kiroRegionPattern.MatchString(region) {
		return region
	}
	return kiro.DefaultRegion
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer("%", "%25", " ", "%20", "\"", "%22", "#", "%23", "&", "%26", "+", "%2B", ":", "%3A", "/", "%2F", "=", "%3D", "?", "%3F")
	return replacer.Replace(value)
}
