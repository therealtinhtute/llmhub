package quotaalert

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	kimiCollectorBaseURL = "https://api.kimi.com"
	kimiUsagePath        = "/coding/v1/usages"
)

type KimiCollector struct {
	httpClient *CollectorHTTPClient
	refresh    CollectorRefreshFunc
	now        func() time.Time
}

type kimiUsagePayload struct {
	Usage  map[string]any         `json:"usage"`
	Limits []kimiLimitPayloadItem `json:"limits"`
}

type kimiLimitPayloadItem struct {
	Name       any            `json:"name"`
	Title      any            `json:"title"`
	Scope      any            `json:"scope"`
	Detail     map[string]any `json:"detail"`
	Window     map[string]any `json:"window"`
	Used       any            `json:"used"`
	Limit      any            `json:"limit"`
	Remaining  any            `json:"remaining"`
	ResetAt    any            `json:"resetAt"`
	ResetAtAlt any            `json:"reset_at"`
	ResetIn    any            `json:"resetIn"`
	ResetInAlt any            `json:"reset_in"`
	TTL        any            `json:"ttl"`
}

type kimiQuotaRow struct {
	id        string
	used      float64
	limit     float64
	resetTime string
}

func NewKimiCollector(deps CollectorDependencies) (Collector, error) {
	client := deps.HTTPClient
	var err error
	if client == nil {
		client, err = NewCollectorHTTPClient(CollectorHTTPConfig{
			BaseURL:      kimiCollectorBaseURL,
			AllowedHosts: []string{"api.kimi.com"},
		})
		if err != nil {
			return nil, err
		}
	}
	return &KimiCollector{httpClient: client, refresh: deps.Refresh, now: time.Now}, nil
}

func (c *KimiCollector) Collect(ctx context.Context, auth AuthSnapshot) ([]Observation, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("kimi quota collector is not configured")
	}
	cloned, err := CloneAuthSnapshot(auth, []string{"access_token"}, []string{"access_token"})
	if err != nil {
		return nil, err
	}
	accessToken, ok := snapshotString(cloned, "access_token")
	if !ok {
		return nil, fmt.Errorf("kimi quota collector access token is missing")
	}
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	var payload kimiUsagePayload
	if err = c.httpClient.JSON(ctx, cloned, http.MethodGet, kimiUsagePath, headers, &payload, c.refresh); err != nil {
		return nil, fmt.Errorf("kimi quota request failed: %s", RedactCollectorError(err, cloned))
	}
	observations := buildKimiObservations(cloned, payload, c.now().UTC())
	if len(observations) == 0 {
		return nil, fmt.Errorf("kimi quota response contains no recognized rows")
	}
	return observations, nil
}

func buildKimiObservations(auth AuthSnapshot, payload kimiUsagePayload, observedAt time.Time) []Observation {
	rows := make([]kimiQuotaRow, 0, 1+len(payload.Limits))
	if row, ok := kimiRowFromMap("summary", payload.Usage, observedAt); ok {
		rows = append(rows, row)
	}
	for index, item := range payload.Limits {
		data := item.Detail
		if len(data) == 0 {
			data = map[string]any{
				"name":      item.Name,
				"title":     item.Title,
				"scope":     item.Scope,
				"used":      item.Used,
				"limit":     item.Limit,
				"remaining": item.Remaining,
				"resetAt":   firstAny(item.ResetAt, item.ResetAtAlt),
				"resetIn":   firstAny(item.ResetIn, item.ResetInAlt, item.TTL),
			}
		}
		if row, ok := kimiRowFromMap(fmt.Sprintf("limit-%d", index), data, observedAt); ok {
			rows = append(rows, row)
		}
	}
	observations := make([]Observation, 0, len(rows))
	for _, row := range rows {
		remainingKnown := false
		remaining := Percentage(0)
		if row.limit > 0 {
			remainingKnown = true
			value, err := NormalizePercentage(((row.limit - row.used) / row.limit) * 100)
			if err != nil {
				continue
			}
			remaining = value
		} else if row.used > 0 {
			remainingKnown = true
			remaining = 0
		}
		if !remainingKnown {
			continue
		}
		resetAt, resetKnown := parseProviderTime(row.resetTime, observedAt)
		observation, err := (Observation{
			Identity: StateIdentity{
				AuthID:   auth.AuthID(),
				Provider: ProviderKimi,
				Resource: row.id,
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

func kimiRowFromMap(id string, data map[string]any, observedAt time.Time) (kimiQuotaRow, bool) {
	if len(data) == 0 {
		return kimiQuotaRow{}, false
	}
	limit, limitOK := numberFromAny(data["limit"])
	used, usedOK := numberFromAny(data["used"])
	if !usedOK {
		if remaining, remainingOK := numberFromAny(data["remaining"]); remainingOK && limitOK {
			used = limit - remaining
			usedOK = true
		}
	}
	if !usedOK && !limitOK {
		return kimiQuotaRow{}, false
	}
	resetTime, _ := stringFromAny(firstAny(data["reset_at"], data["resetAt"], data["reset_time"], data["resetTime"]))
	if resetTime == "" {
		if seconds, ok := numberFromAny(firstAny(data["reset_in"], data["resetIn"], data["ttl"])); ok && seconds > 0 {
			resetTime = fmt.Sprintf("%d", observedAt.Add(time.Duration(seconds)*time.Second).Unix())
		}
	}
	return kimiQuotaRow{id: id, used: used, limit: limit, resetTime: resetTime}, true
}
