package executor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/auth/kiro"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/registry"
	"github.com/therealtinhtute/llmhub/internal/runtime/executor/helps"
	"github.com/therealtinhtute/llmhub/internal/thinking"
	"github.com/therealtinhtute/llmhub/internal/util"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	coreusage "github.com/therealtinhtute/llmhub/sdk/cliproxy/usage"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
)

const (
	kiroGeneratePath          = "/generateAssistantResponse"
	kiroListModelsPath        = "/ListAvailableModels"
	kiroListProfilesPath      = "/ListAvailableProfiles"
	kiroSetUserPreferencePath = "/setUserPreference"
	kiroAMZTarget             = "AmazonCodeWhispererStreamingService.GenerateAssistantResponse"
	kiroAmazonQTarget         = "AmazonQDeveloperStreamingService.SendMessage"
	kiroDefaultContextLength  = 200000
	kiroLargeContextLength    = 1000000
	kiroDefaultMaxOutputToken = 64000
	kiroToolNameMaxLength     = 64
	kiroToolResultsPrefix     = "Tool results:"
	kiroUsageDetailMetadata   = "kiro_usage_detail"
	kiroUsageEstimatedMeta    = "kiro_usage_estimated"
	kiroInternalToolNameMap   = "__llmhub_tool_name_map"
	kiroMaxPayloadBytes       = 900 * 1024
	kiroTruncationPlaceholder = "[Earlier conversation history was truncated to fit Kiro's input limit. Older messages and tool activity have been omitted.]"
	kiroMinRecentHistoryTurns = 4
)

var kiroClaudeVersionPattern = regexp.MustCompile(`claude-(?:opus|sonnet|haiku)-(\d+)[.-](\d+)`)

type KiroExecutor struct {
	cfg *config.Config
}

type kiroMessage struct {
	Role        string
	Content     any
	ToolResults []kiroToolResult
	ToolCalls   []map[string]any
	Images      []any
}

type kiroToolResult struct {
	ToolUseID string
	Content   any
}

func NewKiroExecutor(cfg *config.Config) *KiroExecutor {
	return &KiroExecutor{cfg: cfg}
}

func (e *KiroExecutor) Identifier() string { return kiro.Provider }

func (e *KiroExecutor) PrepareRequest(req *http.Request, auth *cliproxyauth.Auth) error {
	if req == nil {
		return nil
	}
	token := metadataString(auth, "access_token")
	if token == "" {
		return statusErr{code: http.StatusUnauthorized, msg: "kiro executor: missing access token"}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	applyKiroHeaders(req, auth)
	return nil
}

func (e *KiroExecutor) HttpRequest(ctx context.Context, auth *cliproxyauth.Auth, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("kiro executor: request is nil")
	}
	if ctx == nil {
		ctx = req.Context()
	}
	httpReq := req.WithContext(ctx)
	if err := e.PrepareRequest(httpReq, auth); err != nil {
		return nil, err
	}
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	return httpClient.Do(httpReq)
}

func (e *KiroExecutor) Refresh(ctx context.Context, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, error) {
	if auth == nil {
		return nil, fmt.Errorf("kiro executor: auth is nil")
	}
	refreshToken := metadataString(auth, "refresh_token")
	psd := kiroProviderSpecificData(auth)
	result, err := kiro.RefreshAccessToken(ctx, refreshToken, psd, helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 30*time.Second))
	if err != nil {
		return nil, err
	}
	updated := auth.Clone()
	if updated.Metadata == nil {
		updated.Metadata = make(map[string]any)
	}
	updated.Metadata["type"] = kiro.Provider
	updated.Metadata["access_token"] = result.AccessToken
	updated.Metadata["refresh_token"] = result.RefreshToken
	if result.ProfileARN != "" {
		updated.Metadata["profile_arn"] = result.ProfileARN
	}
	updated.Metadata["expires_in"] = result.ExpiresIn
	updated.Metadata["expired"] = result.ExpiresAt.Format(time.RFC3339)
	updated.Metadata["last_refresh"] = time.Now().UTC().Format(time.RFC3339)
	updated.LastRefreshedAt = time.Now().UTC()
	updated.UpdatedAt = updated.LastRefreshedAt
	return updated, nil
}

type kiroModelCatalogResponse struct {
	Models []kiroUpstreamModel `json:"models"`
}

type kiroProfileCatalogResponse struct {
	Profiles []struct {
		Arn string `json:"arn"`
	} `json:"profiles"`
}

type kiroUpstreamModel struct {
	ID             string                 `json:"id"`
	ModelID        string                 `json:"modelId"`
	ModelName      string                 `json:"modelName"`
	Description    string                 `json:"description"`
	RateMultiplier float64                `json:"rateMultiplier"`
	TokenLimits    kiroUpstreamTokenLimit `json:"tokenLimits"`
}

type kiroUpstreamTokenLimit struct {
	MaxInputTokens int `json:"maxInputTokens"`
}

// ResolveModels fetches the live Kiro model catalog for an auth and expands
// upstream models into the public Kiro model variants accepted by this executor.
func (e *KiroExecutor) ResolveModels(ctx context.Context, auth *cliproxyauth.Auth) ([]*registry.ModelInfo, *cliproxyauth.Auth, error) {
	if auth == nil {
		return nil, nil, fmt.Errorf("kiro models: auth is nil")
	}
	if metadataString(auth, "access_token") == "" {
		return nil, nil, fmt.Errorf("kiro models: missing access token")
	}
	raw, err := e.fetchKiroModelCatalog(ctx, auth)
	if err != nil {
		if status, ok := err.(interface{ StatusCode() int }); ok && status.StatusCode() == http.StatusUnauthorized && metadataString(auth, "refresh_token") != "" {
			refreshed, errRefresh := e.Refresh(ctx, auth)
			if errRefresh != nil {
				return nil, nil, errRefresh
			}
			raw, err = e.fetchKiroModelCatalog(ctx, refreshed)
			if err != nil {
				return nil, refreshed, err
			}
			return buildKiroModelCatalog(raw), refreshed, nil
		}
		return nil, nil, err
	}
	return buildKiroModelCatalog(raw), nil, nil
}

func (e *KiroExecutor) fetchKiroModelCatalog(ctx context.Context, auth *cliproxyauth.Auth) ([]kiroUpstreamModel, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kiroListAvailableModelsURL(auth), nil)
	if err != nil {
		return nil, fmt.Errorf("kiro models: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+metadataString(auth, "access_token"))
	applyKiroModelListHeaders(req, auth)
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 30*time.Second)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kiro models: request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kiro models: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, statusErr{code: resp.StatusCode, msg: fmt.Sprintf("kiro models: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))}
	}
	var payload kiroModelCatalogResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("kiro models: decode response: %w", err)
	}
	return payload.Models, nil
}

func (e *KiroExecutor) ShouldPrepareRequestAuth(auth *cliproxyauth.Auth) bool {
	if auth == nil || !strings.EqualFold(strings.TrimSpace(auth.Provider), kiro.Provider) {
		return false
	}
	if auth.Disabled || auth.Status == cliproxyauth.StatusDisabled {
		return false
	}
	return metadataString(auth, "access_token") != "" && metadataString(auth, "profile_arn") == ""
}

func (e *KiroExecutor) PrepareRequestAuth(ctx context.Context, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, error) {
	if auth == nil || !e.ShouldPrepareRequestAuth(auth) {
		return nil, nil
	}
	_, updated, err := e.ResolveProfileARN(ctx, auth)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (e *KiroExecutor) ResolveProfileARN(ctx context.Context, auth *cliproxyauth.Auth) (string, *cliproxyauth.Auth, error) {
	if auth == nil {
		return "", nil, fmt.Errorf("kiro profile: auth is nil")
	}
	if profileARN := metadataString(auth, "profile_arn"); profileARN != "" {
		return profileARN, nil, nil
	}
	if metadataString(auth, "access_token") == "" {
		return "", nil, statusErr{code: http.StatusUnauthorized, msg: "kiro profile: missing access token"}
	}

	profileARN, err := e.listAvailableProfilesWithRetry(ctx, auth)
	if err == nil && profileARN != "" {
		updated := auth.Clone()
		if updated.Metadata == nil {
			updated.Metadata = make(map[string]any)
		}
		updated.Metadata["profile_arn"] = profileARN
		updated.UpdatedAt = time.Now().UTC()
		return profileARN, updated, nil
	}

	if metadataString(auth, "refresh_token") != "" {
		refreshed, errRefresh := e.Refresh(ctx, auth)
		if errRefresh == nil {
			if refreshedARN := metadataString(refreshed, "profile_arn"); refreshedARN != "" {
				return refreshedARN, refreshed, nil
			}
		}
	}

	return "", nil, fmt.Errorf("no available Kiro profile")
}

func (e *KiroExecutor) listAvailableProfilesWithRetry(ctx context.Context, auth *cliproxyauth.Auth) (string, error) {
	const maxAttempts = 3
	var lastErr error
	backoff := 200 * time.Millisecond
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		profileARN, err := e.listAvailableProfiles(ctx, auth)
		if err == nil {
			return profileARN, nil
		}
		lastErr = err
		if !isTransientKiroProfileFetchError(err) || attempt == maxAttempts {
			return "", err
		}
		if ctx == nil {
			time.Sleep(backoff)
		} else {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", ctx.Err()
			case <-timer.C:
			}
		}
		backoff *= 2
	}
	return "", lastErr
}

func (e *KiroExecutor) listAvailableProfiles(ctx context.Context, auth *cliproxyauth.Auth) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	body := []byte(`{"maxResults":10}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kiroListAvailableProfilesURL(auth), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("kiro profile: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+metadataString(auth, "access_token"))
	req.Header.Set("Content-Type", "application/json")
	applyKiroModelListHeaders(req, auth)
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 30*time.Second)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("kiro profile: request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("kiro profile: read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", statusErr{code: resp.StatusCode, msg: fmt.Sprintf("kiro profile: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))}
	}
	var payload kiroProfileCatalogResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return "", fmt.Errorf("kiro profile: decode response: %w", err)
	}
	for _, profile := range payload.Profiles {
		if profileARN := strings.TrimSpace(profile.Arn); profileARN != "" {
			return profileARN, nil
		}
	}
	return "", fmt.Errorf("kiro profile: empty profile list")
}

func isTransientKiroProfileFetchError(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(strings.ToLower(err.Error()), "empty profile list") {
		return false
	}
	var status interface{ StatusCode() int }
	if errors.As(err, &status) {
		code := status.StatusCode()
		return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
	}
	return true
}

func (e *KiroExecutor) SetOverageStatus(ctx context.Context, auth *cliproxyauth.Auth, enabled bool) (KiroQuotaState, *cliproxyauth.Auth, error) {
	if auth == nil {
		return KiroQuotaState{}, nil, fmt.Errorf("kiro overage: auth is nil")
	}
	status := "DISABLED"
	if enabled {
		status = "ENABLED"
	}

	profileARN, updatedAuth, err := e.ResolveProfileARN(ctx, auth)
	if err != nil {
		return KiroQuotaState{}, nil, fmt.Errorf("kiro overage: resolve profileArn: %w", err)
	}
	if updatedAuth != nil {
		auth = updatedAuth
	}

	refreshedOn401 := false
	for {
		err = e.postKiroOverageStatus(ctx, auth, profileARN, status)
		if err == nil {
			break
		}
		var statusProvider interface{ StatusCode() int }
		if !errors.As(err, &statusProvider) || statusProvider.StatusCode() != http.StatusUnauthorized || refreshedOn401 || metadataString(auth, "refresh_token") == "" {
			return KiroQuotaState{}, auth, err
		}
		refreshedOn401 = true
		refreshed, errRefresh := e.Refresh(ctx, auth)
		if errRefresh != nil {
			return KiroQuotaState{}, auth, errRefresh
		}
		auth = refreshed
		profileARN, updatedAuth, err = e.ResolveProfileARN(ctx, auth)
		if err != nil {
			return KiroQuotaState{}, auth, fmt.Errorf("kiro overage: resolve refreshed profileArn: %w", err)
		}
		if updatedAuth != nil {
			auth = updatedAuth
		}
	}

	quota, refreshedQuotaAuth, err := e.FetchQuota(ctx, auth)
	if refreshedQuotaAuth != nil {
		auth = refreshedQuotaAuth
	}
	if err != nil {
		quota = KiroQuotaState{
			ProviderQuotaAvailable: false,
			Message:                err.Error(),
			OverageStatus:          status,
			CheckedAt:              time.Now().UTC(),
		}
	} else {
		quota.OverageStatus = status
	}
	return quota, auth, nil
}

func (e *KiroExecutor) postKiroOverageStatus(ctx context.Context, auth *cliproxyauth.Auth, profileARN, status string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	body, err := json.Marshal(map[string]any{
		"profileArn": profileARN,
		"overageConfiguration": map[string]string{
			"overageStatus": status,
		},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kiroSetUserPreferenceURL(auth), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("kiro overage: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+metadataString(auth, "access_token"))
	req.Header.Set("Content-Type", "application/json")
	applyKiroUsageLimitsHeaders(req, auth)
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 30*time.Second)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kiro overage: request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("kiro overage: read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return statusErr{code: resp.StatusCode, msg: fmt.Sprintf("kiro overage: setUserPreference status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))}
	}
	return nil
}

func (e *KiroExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	baseBody, originalOpenAI, from, err := buildKiroRequest(req, opts, false)
	if err != nil {
		return resp, err
	}
	toolNameMap := extractKiroToolNameMap(baseBody)
	reporter := helps.NewUsageReporter(ctx, e.Identifier(), stripKiroModelSuffix(req.Model), auth)
	body := applyKiroProfileARN(baseBody, auth)
	httpResp, err := e.doKiroRequest(ctx, auth, body)
	if err != nil {
		reporter.PublishFailure(ctx, err)
		return resp, err
	}
	if httpResp.StatusCode == http.StatusUnauthorized {
		_ = httpResp.Body.Close()
		refreshed, errRefresh := e.Refresh(ctx, auth)
		if errRefresh != nil {
			reporter.PublishFailure(ctx, errRefresh)
			return resp, errRefresh
		}
		auth = refreshed
		body = applyKiroProfileARN(baseBody, auth)
		httpResp, err = e.doKiroRequest(ctx, auth, body)
		if err != nil {
			reporter.PublishFailure(ctx, err)
			return resp, err
		}
	}
	defer func() {
		if errClose := httpResp.Body.Close(); errClose != nil {
			log.Errorf("kiro executor: close response body error: %v", errClose)
		}
	}()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		data, _ := io.ReadAll(httpResp.Body)
		err = statusErr{code: httpResp.StatusCode, msg: string(data)}
		reporter.PublishFailure(ctx, err)
		return resp, err
	}
	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		reporter.PublishFailure(ctx, err)
		return resp, err
	}
	events := newKiroEventParser().Feed(data)
	detail, hasUsage, estimated := kiroUsageFromEventsForModel(events, req.Model)
	if hasUsage {
		reporter.Publish(ctx, detail)
	} else {
		reporter.EnsurePublished(ctx)
	}
	openaiPayload := kiroEventsToOpenAINonStream(req.Model, events, detail, hasUsage, toolNameMap)
	var param any
	translatedBody := stripKiroInternalFields(body)
	out := sdktranslator.TranslateNonStream(ctx, sdktranslator.FromString("openai"), from, req.Model, originalOpenAI, translatedBody, openaiPayload, &param)
	metadata := map[string]any{}
	if hasUsage {
		metadata[kiroUsageDetailMetadata] = kiroUsageDetailMap(detail)
		metadata[kiroUsageEstimatedMeta] = estimated
	}
	return cliproxyexecutor.Response{Payload: out, Metadata: metadata, Headers: httpResp.Header.Clone()}, nil
}

func (e *KiroExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (_ *cliproxyexecutor.StreamResult, err error) {
	baseBody, originalOpenAI, from, err := buildKiroRequest(req, opts, true)
	if err != nil {
		return nil, err
	}
	toolNameMap := extractKiroToolNameMap(baseBody)
	reporter := helps.NewUsageReporter(ctx, e.Identifier(), stripKiroModelSuffix(req.Model), auth)
	body := applyKiroProfileARN(baseBody, auth)
	httpResp, err := e.doKiroRequest(ctx, auth, body)
	if err != nil {
		reporter.PublishFailure(ctx, err)
		return nil, err
	}
	if httpResp.StatusCode == http.StatusUnauthorized {
		_ = httpResp.Body.Close()
		refreshed, errRefresh := e.Refresh(ctx, auth)
		if errRefresh != nil {
			reporter.PublishFailure(ctx, errRefresh)
			return nil, errRefresh
		}
		auth = refreshed
		body = applyKiroProfileARN(baseBody, auth)
		httpResp, err = e.doKiroRequest(ctx, auth, body)
		if err != nil {
			reporter.PublishFailure(ctx, err)
			return nil, err
		}
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		data, _ := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		err = statusErr{code: httpResp.StatusCode, msg: string(data)}
		reporter.PublishFailure(ctx, err)
		return nil, err
	}

	headers := httpResp.Header.Clone()
	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		defer httpResp.Body.Close()

		parser := newKiroEventParser()
		completionID := "chatcmpl-" + uuid.NewString()
		created := time.Now().Unix()
		buf := make([]byte, 8192)
		var param any
		translatedBody := stripKiroInternalFields(body)
		first := true
		toolIndexes := map[string]int{}
		nextToolIndex := 0
		usageAcc := newKiroUsageAccumulatorForModel(req.Model)
		for {
			n, readErr := httpResp.Body.Read(buf)
			if n > 0 {
				events := parser.Feed(buf[:n])
				for _, event := range events {
					usageAcc.Observe(event)
					var chunks [][]byte
					switch event.Type {
					case "content":
						if event.Content == "" {
							continue
						}
						chunks = kiroContentEventToOpenAIStream(completionID, req.Model, created, event.Content, first)
						first = false
					case "reasoning":
						if event.Content == "" {
							continue
						}
						chunks = kiroReasoningEventToOpenAIStream(completionID, req.Model, created, event.Content, first)
						first = false
					case "tool_calls":
						if len(event.ToolCalls) == 0 {
							continue
						}
						chunks = kiroToolCallEventToOpenAIStream(completionID, req.Model, created, event.ToolCalls, first, toolIndexes, &nextToolIndex, toolNameMap)
						first = false
					default:
						continue
					}
					for _, chunk := range chunks {
						outChunks := translateKiroStreamChunk(ctx, from, req.Model, originalOpenAI, translatedBody, chunk, &param)
						for i := range outChunks {
							select {
							case <-ctx.Done():
								out <- cliproxyexecutor.StreamChunk{Err: ctx.Err()}
								return
							case out <- cliproxyexecutor.StreamChunk{Payload: outChunks[i]}:
							}
						}
					}
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					reporter.PublishFailure(ctx, readErr)
					out <- cliproxyexecutor.StreamChunk{Err: readErr}
					return
				}
				detail, hasUsage, estimated := usageAcc.Detail()
				if hasUsage {
					reporter.Publish(ctx, detail)
					finishReason := "stop"
					if len(toolIndexes) > 0 {
						finishReason = "tool_calls"
					}
					finish := kiroOpenAIStreamFinishPayload(completionID, req.Model, created, finishReason, detail)
					outChunks := translateKiroStreamChunk(ctx, from, req.Model, originalOpenAI, translatedBody, finish, &param)
					metadata := map[string]any{
						kiroUsageDetailMetadata: kiroUsageDetailMap(detail),
						kiroUsageEstimatedMeta:  estimated,
					}
					for i := range outChunks {
						out <- cliproxyexecutor.StreamChunk{Payload: outChunks[i], Metadata: metadata}
					}
				} else {
					reporter.EnsurePublished(ctx)
				}
				done := []byte("data: [DONE]\n\n")
				outChunks := translateKiroStreamChunk(ctx, from, req.Model, originalOpenAI, translatedBody, done, &param)
				for i := range outChunks {
					out <- cliproxyexecutor.StreamChunk{Payload: outChunks[i]}
				}
				return
			}
		}
	}()
	return &cliproxyexecutor.StreamResult{Headers: headers, Chunks: out}, nil
}

func translateKiroStreamChunk(ctx context.Context, from sdktranslator.Format, model string, originalOpenAI, body, chunk []byte, param *any) [][]byte {
	if from.String() == "" || from.String() == "openai" {
		return [][]byte{chunk}
	}
	return sdktranslator.TranslateStream(ctx, sdktranslator.FromString("openai"), from, model, originalOpenAI, body, chunk, param)
}

func (e *KiroExecutor) CountTokens(context.Context, *cliproxyauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{Payload: []byte(`{"total_tokens":0}`)}, nil
}

func (e *KiroExecutor) doKiroRequest(ctx context.Context, auth *cliproxyauth.Auth, body []byte) (*http.Response, error) {
	var authID, authLabel, authType, authValue string
	if auth != nil {
		authID = auth.ID
		authLabel = auth.Label
		authType, authValue = auth.AccountInfo()
	}
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	sendBody := stripKiroInternalFields(body)
	var lastErr error
	for _, endpoint := range kiroGenerationEndpoints(auth) {
		attemptBody := applyKiroEndpointOrigin(sendBody, endpoint.origin)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.rawURL, bytes.NewReader(attemptBody))
		if err != nil {
			lastErr = err
			if !endpoint.fallback {
				return nil, err
			}
			continue
		}
		if err := e.PrepareRequest(httpReq, auth); err != nil {
			return nil, err
		}
		if endpoint.amzTarget != "" {
			httpReq.Header.Set("X-Amz-Target", endpoint.amzTarget)
		} else {
			httpReq.Header.Del("X-Amz-Target")
		}
		helps.RecordAPIRequest(ctx, e.cfg, helps.UpstreamRequestLog{
			URL:       endpoint.rawURL,
			Method:    http.MethodPost,
			Headers:   httpReq.Header.Clone(),
			Body:      attemptBody,
			Provider:  e.Identifier(),
			AuthID:    authID,
			AuthLabel: authLabel,
			AuthType:  authType,
			AuthValue: authValue,
		})
		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			helps.RecordAPIResponseError(ctx, e.cfg, err)
			lastErr = err
			if !endpoint.fallback {
				return nil, err
			}
			continue
		}
		helps.RecordAPIResponseMetadata(ctx, e.cfg, httpResp.StatusCode, httpResp.Header.Clone())
		if shouldFallbackKiroEndpoint(httpResp.StatusCode) && endpoint.fallback {
			data, _ := io.ReadAll(httpResp.Body)
			_ = httpResp.Body.Close()
			lastErr = statusErr{code: httpResp.StatusCode, msg: strings.TrimSpace(string(data))}
			continue
		}
		return httpResp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("kiro executor: no generation endpoints configured")
}

func applyKiroProfileARN(body []byte, auth *cliproxyauth.Auth) []byte {
	profileARN := metadataString(auth, "profile_arn")
	if profileARN == "" {
		return body
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	payload["profileArn"] = profileARN
	raw, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return raw
}

func kiroGenerationEndpoints(auth *cliproxyauth.Auth) []kiroGenerationEndpoint {
	if rawURL := firstMetadataString(auth, "generation_url", "generate_url"); rawURL != "" {
		return []kiroGenerationEndpoint{{name: "custom", rawURL: rawURL, origin: "AI_EDITOR", amzTarget: kiroAMZTarget}}
	}
	if baseURL := firstMetadataString(auth, "base_url", "generation_base_url"); baseURL != "" {
		return []kiroGenerationEndpoint{{
			name:      "codewhisperer",
			rawURL:    strings.TrimRight(baseURL, "/") + kiroGeneratePath,
			origin:    "AI_EDITOR",
			amzTarget: kiroAMZTarget,
		}}
	}
	region := kiroRegion(auth)
	qBase := fmt.Sprintf("https://q.%s.amazonaws.com", region)
	codeWhispererBase := kiro.DefaultAPIBaseURL
	if region != kiro.DefaultRegion {
		codeWhispererBase = qBase
	}
	kiroURL := firstMetadataString(auth, "kiro_generate_url", "kiro_q_generate_url")
	if kiroURL == "" {
		kiroURL = qBase + kiroGeneratePath
	}
	codeWhispererURL := firstMetadataString(auth, "codewhisperer_generate_url")
	if codeWhispererURL == "" {
		codeWhispererURL = codeWhispererBase + kiroGeneratePath
	}
	amazonQURL := firstMetadataString(auth, "amazon_q_generate_url", "amazonq_generate_url")
	if amazonQURL == "" {
		amazonQURL = qBase + kiroGeneratePath
	}
	return []kiroGenerationEndpoint{
		{name: "kiro", rawURL: kiroURL, origin: "AI_EDITOR", fallback: true},
		{name: "codewhisperer", rawURL: codeWhispererURL, origin: "AI_EDITOR", amzTarget: kiroAMZTarget, fallback: true},
		{name: "amazonq", rawURL: amazonQURL, origin: "AI_EDITOR", amzTarget: kiroAmazonQTarget},
	}
}

func firstMetadataString(auth *cliproxyauth.Auth, keys ...string) string {
	for _, key := range keys {
		if value := metadataString(auth, key); value != "" {
			return value
		}
	}
	return ""
}

func kiroRegion(auth *cliproxyauth.Auth) string {
	if region := regionFromKiroProfileARN(metadataString(auth, "profile_arn")); region != "" {
		return region
	}
	if region := metadataString(auth, "region"); region != "" {
		return region
	}
	return kiro.DefaultRegion
}

func shouldFallbackKiroEndpoint(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func applyKiroEndpointOrigin(body []byte, origin string) []byte {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return body
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	state, _ := payload["conversationState"].(map[string]any)
	current, _ := state["currentMessage"].(map[string]any)
	userInput, _ := current["userInputMessage"].(map[string]any)
	if userInput == nil {
		return body
	}
	userInput["origin"] = origin
	raw, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return raw
}

func stripKiroInternalFields(body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	if _, ok := payload[kiroInternalToolNameMap]; !ok {
		return body
	}
	delete(payload, kiroInternalToolNameMap)
	raw, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return raw
}

func extractKiroToolNameMap(body []byte) map[string]string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	raw, ok := payload[kiroInternalToolNameMap]
	if !ok {
		return nil
	}
	out := map[string]string{}
	switch typed := raw.(type) {
	case map[string]any:
		for k, v := range typed {
			if original := strings.TrimSpace(kiroStringValue(v)); original != "" {
				out[k] = original
			}
		}
	case map[string]string:
		for k, v := range typed {
			if original := strings.TrimSpace(v); original != "" {
				out[k] = original
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func applyKiroHeaders(req *http.Request, auth *cliproxyauth.Auth) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.amazon.eventstream")
	req.Header.Set("X-Amz-Target", kiroAMZTarget)
	applyKiroFingerprintHeaders(req, auth, "codewhispererstreaming", "1.0.27", "m/E", 3)
	if auth != nil {
		util.ApplyCustomHeadersFromAttrs(req, auth.Attributes)
	}
}

func applyKiroModelListHeaders(req *http.Request, auth *cliproxyauth.Auth) {
	req.Header.Set("Accept", "application/json")
	applyKiroFingerprintHeaders(req, auth, "codewhispererruntime", "1.0.0", "m/N,E", 1)
	if auth != nil {
		util.ApplyCustomHeadersFromAttrs(req, auth.Attributes)
	}
}

func applyKiroFingerprintHeaders(req *http.Request, auth *cliproxyauth.Auth, apiName, sdkVersion, mode string, maxAttempts int) {
	fingerprint := kiroFingerprint(auth)
	req.Header.Set("User-Agent", fmt.Sprintf("aws-sdk-js/%s ua/2.1 os/windows#10.0.26200 lang/js md/nodejs#22.21.1 api/%s#%s %s KiroIDE-0.10.32-%s", sdkVersion, apiName, sdkVersion, mode, fingerprint))
	req.Header.Set("X-Amz-User-Agent", fmt.Sprintf("aws-sdk-js/%s KiroIDE-0.10.32-%s", sdkVersion, fingerprint))
	req.Header.Set("X-Amzn-Codewhisperer-Optout", "true")
	req.Header.Set("X-Amzn-Kiro-Agent-Mode", "vibe")
	req.Header.Set("Amz-Sdk-Invocation-Id", uuid.NewString())
	req.Header.Set("Amz-Sdk-Request", fmt.Sprintf("attempt=1; max=%d", maxAttempts))
}

func buildKiroRequest(req cliproxyexecutor.Request, opts cliproxyexecutor.Options, stream bool) ([]byte, []byte, sdktranslator.Format, error) {
	from := opts.SourceFormat
	if from.String() == "" {
		from = sdktranslator.FromString("openai")
	}
	to := sdktranslator.FromString("openai")
	original := req.Payload
	if len(opts.OriginalRequest) > 0 {
		original = opts.OriginalRequest
	}
	openaiPayload := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(req.Payload), stream)
	originalOpenAI := sdktranslator.TranslateRequest(from, to, req.Model, bytes.Clone(original), stream)
	body, err := buildKiroPayloadFromOpenAI(openaiPayload, req.Model)
	if err != nil {
		return nil, nil, sdktranslator.FromString("openai"), err
	}
	return body, originalOpenAI, from, nil
}

func buildKiroPayloadFromOpenAI(openaiPayload []byte, requestedModel string) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(openaiPayload, &payload); err != nil {
		return nil, fmt.Errorf("kiro translator: invalid OpenAI payload: %w", err)
	}
	model := stripKiroModelSuffix(strings.TrimSpace(stringFromMap(payload, "model")))
	if model == "" {
		model = stripKiroModelSuffix(requestedModel)
	}
	if model == "" {
		model = "auto"
	}
	modelID := normalizeKiroModelID(model)

	messages, _ := payload["messages"].([]any)
	tools, _ := payload["tools"].([]any)
	systemPrompt, unified := kiroOpenAIMessages(messages)
	if len(unified) == 0 {
		return nil, fmt.Errorf("kiro translator: messages are required")
	}
	tools = synthesizeKiroToolsFromHistory(tools, unified)
	toolNameMap := map[string]string{}
	tools = shortenKiroToolDefinitions(tools, toolNameMap)
	unified = shortenKiroMessageToolCalls(unified, toolNameMap)
	unified = ensureKiroUserFirst(unified)
	current := unified[len(unified)-1]
	history := unified[:len(unified)-1]
	history = mergeAdjacentKiroUserHistory(history)
	historyToolNames := kiroToolCallNameMap(history)
	currentToolResultIDs := collectKiroToolResultIDs(current.ToolResults)
	keepCurrentToolResults := currentKiroToolResultsMatchLastAssistant(history, currentToolResultIDs)
	if keepCurrentToolResults {
		history = sanitizeKiroHistory(history, currentToolResultIDs)
	} else {
		history = sanitizeKiroHistory(history, nil)
		current.Content = flattenKiroToolResultsIntoContent(current.Content, current.ToolResults, historyToolNames)
		current.ToolResults = nil
	}
	currentContent := textContent(current.Content)
	if systemPrompt != "" && len(history) == 0 {
		currentContent = strings.TrimSpace(systemPrompt + "\n\n" + currentContent)
	}
	if current.Role == "assistant" {
		history = append(history, current)
		current = kiroMessage{Role: "user", Content: "(empty placeholder)"}
		history = mergeAdjacentKiroUserHistory(history)
		currentContent = "(empty placeholder)"
	}
	if currentContent == "" {
		currentContent = "(empty placeholder)"
	}
	if shouldInjectKiroThinking(payload, requestedModel) {
		currentContent = "<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>16000</max_thinking_length>\n\n" + currentContent
	}
	if strings.Contains(requestedModel, "-agentic") {
		currentContent = "Use concise agentic coding steps and prefer chunked file edits.\n\n" + currentContent
	}
	currentContent = fmt.Sprintf("[Context: Current time is %s]\n\n%s", time.Now().UTC().Format(time.RFC3339), currentContent)

	if systemPrompt != "" && len(history) > 0 && history[0].Role == "user" {
		history[0].Content = strings.TrimSpace(systemPrompt + "\n\n" + textContent(history[0].Content))
	}

	userInput := map[string]any{
		"content": currentContent,
		"modelId": modelID,
		"origin":  "AI_EDITOR",
	}
	if images := current.Images; len(images) > 0 {
		userInput["images"] = images
	}
	contextValue := map[string]any{}
	if convertedTools := convertKiroTools(tools); len(convertedTools) > 0 {
		contextValue["tools"] = convertedTools
	}
	if toolResults := buildKiroToolResults(current.ToolResults); len(toolResults) > 0 {
		contextValue["toolResults"] = toolResults
	}
	if len(contextValue) > 0 {
		userInput["userInputMessageContext"] = contextValue
	}

	conversationState := map[string]any{
		"chatTriggerType": "MANUAL",
		"conversationId":  uuid.NewString(),
		"currentMessage":  map[string]any{"userInputMessage": userInput},
	}
	if len(history) > 0 {
		conversationState["history"] = buildKiroHistory(history, modelID)
	}
	out := map[string]any{
		"conversationState": conversationState,
		"inferenceConfig": map[string]any{
			"maxTokens": 32000,
		},
	}
	if len(toolNameMap) > 0 {
		out[kiroInternalToolNameMap] = toolNameMap
	}
	if temperature, ok := payload["temperature"]; ok {
		out["inferenceConfig"].(map[string]any)["temperature"] = temperature
	}
	if topP, ok := payload["top_p"]; ok {
		out["inferenceConfig"].(map[string]any)["topP"] = topP
	}
	compactKiroPayload(out)
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func kiroOpenAIMessages(messages []any) (string, []kiroMessage) {
	system := strings.Builder{}
	out := make([]kiroMessage, 0, len(messages))
	for _, raw := range messages {
		msg, _ := raw.(map[string]any)
		role := strings.TrimSpace(stringFromMap(msg, "role"))
		content := msg["content"]
		switch role {
		case "system", "developer":
			if text := textContent(content); text != "" {
				if system.Len() > 0 {
					system.WriteString("\n")
				}
				system.WriteString(text)
			}
			continue
		case "tool":
			out = append(out, kiroMessage{
				Role:    "user",
				Content: content,
				ToolResults: []kiroToolResult{{
					ToolUseID: stringFromMap(msg, "tool_call_id"),
					Content:   content,
				}},
			})
		case "assistant":
			var toolCalls []map[string]any
			if rawCalls, ok := msg["tool_calls"].([]any); ok {
				for _, rawCall := range rawCalls {
					if call, ok := rawCall.(map[string]any); ok {
						toolCalls = append(toolCalls, call)
					}
				}
			}
			out = append(out, kiroMessage{Role: "assistant", Content: content, ToolCalls: toolCalls})
		default:
			out = append(out, kiroMessage{Role: "user", Content: content, Images: extractKiroImages(content)})
		}
	}
	return strings.TrimSpace(system.String()), out
}

func buildKiroHistory(messages []kiroMessage, modelID string) []any {
	history := make([]any, 0, len(messages))
	for _, msg := range messages {
		content := textContent(msg.Content)
		if content == "" {
			content = "(empty placeholder)"
		}
		if msg.Role == "assistant" {
			entry := map[string]any{"content": content}
			if len(msg.ToolCalls) > 0 {
				entry["toolUses"] = openAIToolCallsToKiro(msg.ToolCalls)
			}
			history = append(history, map[string]any{"assistantResponseMessage": entry})
			continue
		}
		user := map[string]any{"content": content, "modelId": modelID, "origin": "AI_EDITOR"}
		if toolResults := buildKiroToolResults(msg.ToolResults); len(toolResults) > 0 {
			user["userInputMessageContext"] = map[string]any{"toolResults": toolResults}
		}
		if images := msg.Images; len(images) > 0 {
			user["images"] = images
		}
		history = append(history, map[string]any{"userInputMessage": user})
	}
	return history
}

func buildKiroToolResults(results []kiroToolResult) []any {
	if len(results) == 0 {
		return nil
	}
	out := make([]any, 0, len(results))
	for _, result := range results {
		toolUseID := strings.TrimSpace(result.ToolUseID)
		if toolUseID == "" {
			continue
		}
		out = append(out, map[string]any{
			"toolUseId": toolUseID,
			"status":    "success",
			"content":   []any{map[string]any{"text": textContent(result.Content)}},
		})
	}
	return out
}

func convertKiroTools(tools []any) []any {
	out := make([]any, 0, len(tools))
	for _, raw := range tools {
		tool, _ := raw.(map[string]any)
		fn, _ := tool["function"].(map[string]any)
		name := strings.TrimSpace(stringFromMap(fn, "name"))
		if name == "" {
			continue
		}
		description := strings.TrimSpace(stringFromMap(fn, "description"))
		if description == "" {
			description = "Tool: " + name
		}
		out = append(out, map[string]any{"toolSpecification": map[string]any{
			"name":        name,
			"description": description,
			"inputSchema": map[string]any{"json": sanitizeKiroSchema(fn["parameters"])},
		}})
	}
	return out
}

func synthesizeKiroToolsFromHistory(tools []any, messages []kiroMessage) []any {
	if len(tools) > 0 {
		return tools
	}
	out := make([]any, 0)
	seen := make(map[string]struct{})
	for _, msg := range messages {
		for _, call := range msg.ToolCalls {
			fn, _ := call["function"].(map[string]any)
			name := strings.TrimSpace(stringFromMap(fn, "name"))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        name,
					"description": "Tool: " + name,
					"parameters": map[string]any{
						"type":       "object",
						"properties": map[string]any{},
						"required":   []any{},
					},
				},
			})
		}
	}
	return out
}

func shortenKiroToolDefinitions(tools []any, nameMap map[string]string) []any {
	if len(tools) == 0 {
		return tools
	}
	out := make([]any, 0, len(tools))
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		fn, ok := tool["function"].(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		name := strings.TrimSpace(stringFromMap(fn, "name"))
		short := shortenKiroToolName(name, nameMap)
		if short != name {
			tool = cloneStringAnyMap(tool)
			fn = cloneStringAnyMap(fn)
			fn["name"] = short
			tool["function"] = fn
		}
		out = append(out, tool)
	}
	return out
}

func shortenKiroMessageToolCalls(messages []kiroMessage, nameMap map[string]string) []kiroMessage {
	if len(messages) == 0 {
		return messages
	}
	out := make([]kiroMessage, len(messages))
	copy(out, messages)
	for i := range out {
		if len(out[i].ToolCalls) == 0 {
			continue
		}
		calls := make([]map[string]any, len(out[i].ToolCalls))
		for j, call := range out[i].ToolCalls {
			calls[j] = call
			fn, ok := call["function"].(map[string]any)
			if !ok {
				continue
			}
			name := strings.TrimSpace(stringFromMap(fn, "name"))
			short := shortenKiroToolName(name, nameMap)
			if short == name {
				continue
			}
			updatedCall := cloneStringAnyMap(call)
			updatedFn := cloneStringAnyMap(fn)
			updatedFn["name"] = short
			updatedCall["function"] = updatedFn
			calls[j] = updatedCall
		}
		out[i].ToolCalls = calls
	}
	return out
}

func shortenKiroToolName(name string, nameMap map[string]string) string {
	name = strings.TrimSpace(name)
	if name == "" || len(name) <= kiroToolNameMaxLength {
		return name
	}
	sum := sha256.Sum256([]byte(name))
	suffix := hex.EncodeToString(sum[:6])
	prefixLimit := kiroToolNameMaxLength - len(suffix) - 2
	if prefixLimit < 1 {
		prefixLimit = 1
	}
	short := strings.TrimRight(name[:prefixLimit], "_-. ")
	if short == "" {
		short = "tool"
	}
	short = short + "__" + suffix
	if nameMap != nil {
		nameMap[short] = name
	}
	return short
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func restoreKiroToolName(name string, nameMap map[string]string) string {
	if nameMap == nil {
		return name
	}
	if original := strings.TrimSpace(nameMap[name]); original != "" {
		return original
	}
	return name
}

func compactKiroPayload(payload map[string]any) {
	if len(payload) == 0 || kiroPayloadSize(payload) <= kiroMaxPayloadBytes {
		return
	}
	state, _ := payload["conversationState"].(map[string]any)
	history, _ := state["history"].([]any)
	if len(history) == 0 {
		return
	}
	removed := false
	for kiroPayloadSize(payload) > kiroMaxPayloadBytes && len(history) > kiroMinRecentHistoryTurns+1 {
		history = append(history[:1], history[2:]...)
		removed = true
		state["history"] = history
	}
	for kiroPayloadSize(payload) > kiroMaxPayloadBytes && len(history) > kiroMinRecentHistoryTurns {
		history = history[1:]
		removed = true
		state["history"] = history
	}
	if !removed {
		return
	}
	placeholder := map[string]any{
		"userInputMessage": map[string]any{
			"content": kiroTruncationPlaceholder,
			"origin":  "AI_EDITOR",
		},
	}
	insertAt := 0
	if len(history) > 0 {
		if first, ok := history[0].(map[string]any); ok {
			if _, ok := first["userInputMessage"]; ok {
				insertAt = 1
			}
		}
	}
	if insertAt >= len(history) {
		history = append(history, placeholder)
	} else {
		history = append(history[:insertAt], append([]any{placeholder}, history[insertAt:]...)...)
	}
	state["history"] = history
	for kiroPayloadSize(payload) > kiroMaxPayloadBytes && len(history) > kiroMinRecentHistoryTurns {
		history = append(history[:insertAt], history[insertAt+1:]...)
		state["history"] = history
		if len(history) == 0 {
			break
		}
		if insertAt >= len(history) {
			insertAt = len(history) - 1
		}
	}
}

func kiroPayloadSize(payload map[string]any) int {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0
	}
	return len(raw)
}

func validateKiroToolDefinitions(tools []any) error {
	var violations []string
	for _, raw := range tools {
		tool, _ := raw.(map[string]any)
		fn, _ := tool["function"].(map[string]any)
		name := strings.TrimSpace(stringFromMap(fn, "name"))
		if name == "" {
			continue
		}
		if len(name) > kiroToolNameMaxLength {
			violations = append(violations, fmt.Sprintf("tool %q is %d characters", name, len(name)))
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return statusErr{
		code: http.StatusBadRequest,
		msg: fmt.Sprintf(
			"kiro translator: tool name(s) exceed Kiro API limit of %d characters: %s",
			kiroToolNameMaxLength,
			strings.Join(violations, "; "),
		),
	}
}

func validateKiroMessageToolCalls(messages []kiroMessage) error {
	var violations []string
	for _, msg := range messages {
		for _, call := range msg.ToolCalls {
			fn, _ := call["function"].(map[string]any)
			name := strings.TrimSpace(stringFromMap(fn, "name"))
			if name == "" || len(name) <= kiroToolNameMaxLength {
				continue
			}
			violations = append(violations, fmt.Sprintf("assistant tool call %q is %d characters", name, len(name)))
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return statusErr{
		code: http.StatusBadRequest,
		msg: fmt.Sprintf(
			"kiro translator: tool name(s) exceed Kiro API limit of %d characters: %s",
			kiroToolNameMaxLength,
			strings.Join(violations, "; "),
		),
	}
}

func sanitizeKiroSchema(raw any) any {
	sanitized, ok := sanitizeKiroSchemaValue(raw).(map[string]any)
	if !ok {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []any{},
		}
	}
	out := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []any{},
	}
	for k, v := range sanitized {
		out[k] = v
	}
	if required, ok := normalizeKiroRequired(out["required"]); ok {
		out["required"] = required
	} else {
		out["required"] = []any{}
	}
	if _, ok := out["properties"]; !ok {
		out["properties"] = map[string]any{}
	}
	if _, ok := out["type"]; !ok {
		out["type"] = "object"
	}
	return out
}

func sanitizeKiroSchemaValue(raw any) any {
	switch typed := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			if k == "additionalProperties" {
				continue
			}
			if k == "required" {
				if required, ok := normalizeKiroRequired(v); ok {
					out[k] = required
				}
				continue
			}
			out[k] = sanitizeKiroSchemaValue(v)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = sanitizeKiroSchemaValue(typed[i])
		}
		return out
	default:
		if raw == nil {
			return map[string]any{}
		}
		return raw
	}
}

func normalizeKiroRequired(v any) ([]any, bool) {
	switch typed := v.(type) {
	case []any:
		return typed, true
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out, true
	default:
		return nil, false
	}
}

func openAIToolCallsToKiro(calls []map[string]any) []any {
	out := make([]any, 0, len(calls))
	for _, call := range calls {
		fn, _ := call["function"].(map[string]any)
		name := stringFromMap(fn, "name")
		var input any = map[string]any{}
		if args := strings.TrimSpace(stringFromMap(fn, "arguments")); args != "" {
			_ = json.Unmarshal([]byte(args), &input)
		}
		out = append(out, map[string]any{
			"name":      name,
			"input":     input,
			"toolUseId": stringFromMap(call, "id"),
		})
	}
	return out
}

func collectKiroToolResultIDs(results []kiroToolResult) map[string]struct{} {
	if len(results) == 0 {
		return nil
	}
	ids := make(map[string]struct{}, len(results))
	for _, result := range results {
		id := strings.TrimSpace(result.ToolUseID)
		if id == "" {
			continue
		}
		ids[id] = struct{}{}
	}
	return ids
}

func currentKiroToolResultsMatchLastAssistant(history []kiroMessage, currentToolResultIDs map[string]struct{}) bool {
	if len(currentToolResultIDs) == 0 || len(history) == 0 {
		return false
	}
	last := history[len(history)-1]
	if last.Role != "assistant" || len(last.ToolCalls) == 0 {
		return false
	}
	for _, call := range last.ToolCalls {
		callID := strings.TrimSpace(stringFromMap(call, "id"))
		if callID == "" {
			return false
		}
		if _, ok := currentToolResultIDs[callID]; !ok {
			return false
		}
	}
	return true
}

func kiroToolCallNameMap(messages []kiroMessage) map[string]string {
	names := make(map[string]string)
	for _, msg := range messages {
		for _, call := range msg.ToolCalls {
			callID := strings.TrimSpace(stringFromMap(call, "id"))
			if callID == "" {
				continue
			}
			fn, _ := call["function"].(map[string]any)
			name := strings.TrimSpace(stringFromMap(fn, "name"))
			if name == "" {
				continue
			}
			names[callID] = name
		}
	}
	return names
}

func narrateKiroToolResults(results []kiroToolResult, toolNames map[string]string) string {
	if len(results) == 0 {
		return ""
	}
	parts := make([]string, 0, len(results))
	for _, result := range results {
		body := strings.TrimSpace(textContent(result.Content))
		if body == "" {
			body = "(no output)"
		}
		toolName := strings.TrimSpace(toolNames[strings.TrimSpace(result.ToolUseID)])
		if toolName != "" {
			parts = append(parts, fmt.Sprintf("[%s] %s", toolName, body))
			continue
		}
		parts = append(parts, body)
	}
	if len(parts) == 0 {
		return ""
	}
	return kiroToolResultsPrefix + "\n\n" + strings.Join(parts, "\n\n")
}

func joinKiroContent(existing any, extra string) any {
	existingText := strings.TrimSpace(textContent(existing))
	extra = strings.TrimSpace(extra)
	switch {
	case existingText != "" && extra != "":
		return existingText + "\n\n" + extra
	case extra != "":
		return extra
	default:
		return existingText
	}
}

func flattenKiroToolResultsIntoContent(existing any, results []kiroToolResult, toolNames map[string]string) any {
	narrated := strings.TrimSpace(narrateKiroToolResults(results, toolNames))
	existingText := strings.TrimSpace(textContent(existing))
	if narrated == "" {
		return existingText
	}
	if existingText == "" {
		return narrated
	}
	plain := strings.TrimSpace(plainKiroToolResultText(results))
	remainder := existingText
	if plain != "" {
		switch {
		case remainder == plain:
			remainder = ""
		case strings.HasPrefix(remainder, plain+"\n\n"):
			remainder = strings.TrimSpace(strings.TrimPrefix(remainder, plain+"\n\n"))
		}
	}
	switch {
	case remainder != "":
		return narrated + "\n\n" + remainder
	default:
		return narrated
	}
}

func plainKiroToolResultText(results []kiroToolResult) string {
	parts := make([]string, 0, len(results))
	for _, result := range results {
		body := strings.TrimSpace(textContent(result.Content))
		if body == "" {
			continue
		}
		parts = append(parts, body)
	}
	return strings.Join(parts, "\n\n")
}

func sanitizeKiroHistory(messages []kiroMessage, currentToolResultIDs map[string]struct{}) []kiroMessage {
	if len(messages) == 0 {
		return messages
	}
	toolNames := kiroToolCallNameMap(messages)
	activeIdx := -1
	if len(currentToolResultIDs) > 0 {
		last := messages[len(messages)-1]
		if last.Role == "assistant" && len(last.ToolCalls) > 0 {
			allCovered := true
			for _, call := range last.ToolCalls {
				callID := strings.TrimSpace(stringFromMap(call, "id"))
				if callID == "" {
					allCovered = false
					break
				}
				if _, ok := currentToolResultIDs[callID]; !ok {
					allCovered = false
					break
				}
			}
			if allCovered {
				activeIdx = len(messages) - 1
			}
		}
	}

	sanitized := make([]kiroMessage, 0, len(messages))
	for idx, msg := range messages {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 && idx != activeIdx {
			msg.ToolCalls = nil
		}
		if msg.Role == "user" && len(msg.ToolResults) > 0 {
			msg.Content = flattenKiroToolResultsIntoContent(msg.Content, msg.ToolResults, toolNames)
			msg.ToolResults = nil
		}
		if msg.Role == "assistant" && len(msg.ToolCalls) == 0 && strings.TrimSpace(textContent(msg.Content)) == "" {
			continue
		}
		if msg.Role == "user" && strings.TrimSpace(textContent(msg.Content)) == "" && len(msg.Images) == 0 {
			msg.Content = "(empty placeholder)"
		}
		sanitized = append(sanitized, msg)
	}
	sanitized = mergeAdjacentKiroUserHistory(sanitized)
	return trimLeadingKiroAssistantHistory(sanitized)
}

func trimLeadingKiroAssistantHistory(messages []kiroMessage) []kiroMessage {
	start := 0
	for start < len(messages) && messages[start].Role == "assistant" {
		start++
	}
	if start == 0 {
		return messages
	}
	if start >= len(messages) {
		return nil
	}
	return messages[start:]
}

func ensureKiroUserFirst(messages []kiroMessage) []kiroMessage {
	if len(messages) == 0 || messages[0].Role == "user" {
		return messages
	}
	return append([]kiroMessage{{Role: "user", Content: "(empty placeholder)"}}, messages...)
}

func mergeAdjacentKiroUserHistory(messages []kiroMessage) []kiroMessage {
	if len(messages) < 2 {
		return messages
	}
	out := make([]kiroMessage, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "user" && len(out) > 0 && out[len(out)-1].Role == "user" {
			out[len(out)-1] = mergeKiroUserMessages(out[len(out)-1], msg)
			continue
		}
		out = append(out, msg)
	}
	return out
}

func mergeKiroUserMessages(left, right kiroMessage) kiroMessage {
	merged := left
	leftText := strings.TrimSpace(textContent(left.Content))
	rightText := strings.TrimSpace(textContent(right.Content))
	switch {
	case leftText == "":
		merged.Content = rightText
	case rightText == "":
		merged.Content = leftText
	default:
		merged.Content = leftText + "\n\n" + rightText
	}
	if len(right.Images) > 0 {
		merged.Images = append(append([]any{}, left.Images...), right.Images...)
	}
	if len(right.ToolResults) > 0 {
		merged.ToolResults = append(append([]kiroToolResult{}, left.ToolResults...), right.ToolResults...)
	}
	return merged
}

func shouldInjectKiroThinking(payload map[string]any, requestedModel string) bool {
	if strings.Contains(requestedModel, "-thinking") {
		return true
	}
	effort := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "reasoning_effort")))
	return effort != "" && effort != "none"
}

func stripKiroModelSuffix(model string) string {
	result := thinking.ParseSuffix(model)
	model = result.ModelName
	model = strings.TrimSuffix(model, "-agentic")
	model = strings.TrimSuffix(model, "-thinking")
	return model
}

func buildKiroModelCatalog(raw []kiroUpstreamModel) []*registry.ModelInfo {
	models := make([]*registry.ModelInfo, 0, len(raw)*4)
	for _, upstream := range raw {
		upstreamID := strings.TrimSpace(upstream.ModelID)
		if upstreamID == "" {
			upstreamID = strings.TrimSpace(upstream.ID)
		}
		if upstreamID == "" {
			continue
		}
		display := formatKiroModelDisplayName(upstream.ModelName, upstreamID, upstream.RateMultiplier)
		contextLength := upstream.TokenLimits.MaxInputTokens
		if contextLength <= 0 {
			contextLength = kiroDefaultContextLength
		}
		models = append(models, kiroModelVariant(upstreamID, display, upstream.Description, contextLength, false, false))
		models = append(models, kiroModelVariant(upstreamID+"-thinking", display+" (Thinking)", upstream.Description, contextLength, true, false))
		if !strings.EqualFold(upstreamID, "auto") {
			models = append(models, kiroModelVariant(upstreamID+"-agentic", display+" (Agentic)", upstream.Description, contextLength, false, true))
			models = append(models, kiroModelVariant(upstreamID+"-thinking-agentic", display+" (Thinking + Agentic)", upstream.Description, contextLength, true, true))
		}
	}
	return models
}

func kiroModelVariant(id, displayName, description string, contextLength int, thinkingEnabled, agentic bool) *registry.ModelInfo {
	info := &registry.ModelInfo{
		ID:                  id,
		Object:              "model",
		Created:             1751328000,
		OwnedBy:             kiro.Provider,
		Type:                kiro.Provider,
		DisplayName:         displayName,
		Name:                id,
		Description:         strings.TrimSpace(description),
		ContextLength:       contextLength,
		MaxCompletionTokens: kiroDefaultMaxOutputToken,
	}
	if thinkingEnabled {
		info.Thinking = &registry.ThinkingSupport{Levels: []string{"low", "medium", "high"}}
	}
	if agentic && info.Description == "" {
		info.Description = "Kiro model with agentic coding prompt injection."
	}
	return info
}

func formatKiroModelDisplayName(modelName, modelID string, rateMultiplier float64) string {
	base := strings.TrimSpace(modelName)
	if base == "" {
		base = strings.TrimSpace(modelID)
	}
	if base == "" {
		base = "Kiro"
	}
	if rateMultiplier > 0 && (rateMultiplier < 0.999999999 || rateMultiplier > 1.000000001) {
		return fmt.Sprintf("Kiro %s (%.1fx credit)", base, rateMultiplier)
	}
	return "Kiro " + base
}

func kiroListAvailableModelsURL(auth *cliproxyauth.Auth) string {
	rawURL := metadataString(auth, "models_url")
	if rawURL == "" {
		rawURL = metadataString(auth, "list_models_url")
	}
	if rawURL == "" {
		region := metadataString(auth, "region")
		if region == "" {
			region = regionFromKiroProfileARN(metadataString(auth, "profile_arn"))
		}
		if region == "" {
			region = kiro.DefaultRegion
		}
		rawURL = fmt.Sprintf("https://q.%s.amazonaws.com%s", region, kiroListModelsPath)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	if query.Get("origin") == "" {
		query.Set("origin", "AI_EDITOR")
	}
	if profileARN := metadataString(auth, "profile_arn"); profileARN != "" && query.Get("profileArn") == "" {
		query.Set("profileArn", profileARN)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func kiroListAvailableProfilesURL(auth *cliproxyauth.Auth) string {
	if rawURL := metadataString(auth, "profiles_url"); rawURL != "" {
		return rawURL
	}
	if rawURL := metadataString(auth, "list_profiles_url"); rawURL != "" {
		return rawURL
	}
	baseURL := metadataString(auth, "codewhisperer_base_url")
	if baseURL == "" {
		baseURL = kiroCodeWhispererUsageBaseURL
	}
	return strings.TrimRight(baseURL, "/") + kiroListProfilesPath
}

func kiroSetUserPreferenceURL(auth *cliproxyauth.Auth) string {
	if rawURL := metadataString(auth, "set_user_preference_url"); rawURL != "" {
		return rawURL
	}
	if rawURL := metadataString(auth, "overage_url"); rawURL != "" {
		return rawURL
	}
	region := metadataString(auth, "region")
	if region == "" {
		region = regionFromKiroProfileARN(metadataString(auth, "profile_arn"))
	}
	if region == "" {
		region = kiro.DefaultRegion
	}
	return fmt.Sprintf("https://q.%s.amazonaws.com%s", region, kiroSetUserPreferencePath)
}

func regionFromKiroProfileARN(profileARN string) string {
	parts := strings.Split(profileARN, ":")
	if len(parts) >= 4 {
		return strings.TrimSpace(parts[3])
	}
	return ""
}

func kiroContextWindowSize(model string) int {
	baseModel := stripKiroModelSuffix(strings.TrimSpace(model))
	if info := registry.LookupModelInfo(baseModel, kiro.Provider); info != nil && info.ContextLength > 0 &&
		(strings.EqualFold(info.OwnedBy, kiro.Provider) || strings.EqualFold(info.Type, kiro.Provider)) {
		return info.ContextLength
	}
	lower := strings.ToLower(baseModel)
	if match := kiroClaudeVersionPattern.FindStringSubmatch(lower); match != nil {
		major, errMajor := strconv.Atoi(match[1])
		minor, errMinor := strconv.Atoi(match[2])
		if errMajor == nil && errMinor == nil {
			if major > 4 || (major == 4 && minor >= 6) {
				return kiroLargeContextLength
			}
			return kiroDefaultContextLength
		}
	}
	for _, marker := range []string{"4.6", "4-6", "4.7", "4-7", "4.8", "4-8", "4.9", "4-9"} {
		if strings.Contains(lower, marker) {
			return kiroLargeContextLength
		}
	}
	return kiroDefaultContextLength
}

func normalizeKiroModelID(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "auto"
	}
	return model
}

func textContent(content any) string {
	switch typed := content.(type) {
	case string:
		return typed
	case []any:
		var b strings.Builder
		for _, raw := range typed {
			item, _ := raw.(map[string]any)
			if stringFromMap(item, "type") == "text" {
				b.WriteString(stringFromMap(item, "text"))
			}
		}
		return b.String()
	default:
		return ""
	}
}

func extractKiroImages(content any) []any {
	// Kiro's upstream image handling still rejects otherwise valid Claude/OpenAI
	// multimodal requests with "Improperly formed request". Until the upstream
	// contract is clearer, drop image blocks on this path so Claude tool flows
	// keep working instead of failing hard.
	return nil
}

type kiroParsedEvent struct {
	Type      string
	Content   string
	Usage     any
	ToolCalls []kiroToolCall
}

type kiroUsageAccumulator struct {
	detail                 coreusage.Detail
	hasUsage               bool
	estimated              bool
	contextLength          int
	contextUsagePercentage float64
	hasContextUsage        bool
	hasMetering            bool
	totalContentLength     int64
}

type kiroToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type kiroToolCallAccumulator struct {
	Name             string
	ArgumentsBuilder strings.Builder
	FallbackArgument string
}

type kiroEventParser struct {
	buffer      []byte
	lastContent string
}

type kiroGenerationEndpoint struct {
	name      string
	rawURL    string
	origin    string
	amzTarget string
	fallback  bool
}

func newKiroEventParser() *kiroEventParser { return &kiroEventParser{} }

func newKiroUsageAccumulator() *kiroUsageAccumulator {
	return &kiroUsageAccumulator{contextLength: kiroDefaultContextLength}
}

func newKiroUsageAccumulatorForModel(model string) *kiroUsageAccumulator {
	return &kiroUsageAccumulator{contextLength: kiroContextWindowSize(model)}
}

func (a *kiroUsageAccumulator) Observe(event kiroParsedEvent) {
	if a == nil {
		return
	}
	switch event.Type {
	case "content", "reasoning":
		a.totalContentLength += int64(len(event.Content))
	case "usage":
		if detail, ok := kiroUsageDetailFromPayload(event.Usage); ok {
			a.detail = detail
			a.hasUsage = true
			a.estimated = false
		}
	case "context_usage":
		if pct, ok := kiroContextUsagePercentage(event.Usage); ok {
			a.contextUsagePercentage = pct
			a.hasContextUsage = true
		}
	case "metering":
		a.hasMetering = true
	}
}

func (a *kiroUsageAccumulator) Detail() (coreusage.Detail, bool, bool) {
	if a == nil {
		return coreusage.Detail{}, false, false
	}
	if a.hasUsage {
		return normalizeKiroUsageDetail(a.detail), true, a.estimated
	}
	if !a.hasContextUsage || !a.hasMetering {
		return coreusage.Detail{}, false, false
	}
	outputTokens := int64(0)
	if a.totalContentLength > 0 {
		outputTokens = maxInt64(1, a.totalContentLength/4)
	}
	inputTokens := int64(0)
	if a.contextUsagePercentage > 0 {
		contextLength := a.contextLength
		if contextLength <= 0 {
			contextLength = kiroDefaultContextLength
		}
		inputTokens = int64((a.contextUsagePercentage * float64(contextLength)) / 100)
	}
	detail := normalizeKiroUsageDetail(coreusage.Detail{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
	})
	if !kiroUsageDetailNonZero(detail) {
		return coreusage.Detail{}, false, false
	}
	return detail, true, true
}

func kiroUsageFromEvents(events []kiroParsedEvent) (coreusage.Detail, bool, bool) {
	return kiroUsageFromEventsForModel(events, "")
}

func kiroUsageFromEventsForModel(events []kiroParsedEvent, model string) (coreusage.Detail, bool, bool) {
	acc := newKiroUsageAccumulatorForModel(model)
	for _, event := range events {
		acc.Observe(event)
	}
	return acc.Detail()
}

func (p *kiroEventParser) Feed(chunk []byte) []kiroParsedEvent {
	out := make([]kiroParsedEvent, 0)
	p.buffer = append(p.buffer, chunk...)
	for {
		if len(p.buffer) < 16 || p.buffer[0] == '{' {
			break
		}
		totalLength := int(binary.BigEndian.Uint32(p.buffer[0:4]))
		if totalLength < 16 {
			break
		}
		if len(p.buffer) < totalLength {
			return out
		}
		event, ok := parseKiroEventStreamFrame(p.buffer[:totalLength])
		p.buffer = p.buffer[totalLength:]
		if !ok {
			continue
		}
		out = append(out, event...)
	}
	p.buffer = bytes.TrimLeft(p.buffer, "\x00\r\n\t ")
	for {
		pos, eventType := p.nextEvent()
		if pos < 0 {
			return out
		}
		end := findKiroJSONEnd(p.buffer, pos)
		if end < 0 {
			return out
		}
		raw := p.buffer[pos : end+1]
		p.buffer = p.buffer[end+1:]
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		switch eventType {
		case "content":
			content := stringFromMap(obj, "content")
			if content == "" || content == p.lastContent || obj["followupPrompt"] != nil {
				continue
			}
			p.lastContent = content
			out = append(out, kiroParsedEvent{Type: "content", Content: content})
		case "usage":
			out = append(out, kiroParsedEvent{Type: "usage", Usage: obj["usage"]})
		}
	}
}

func (p *kiroEventParser) nextEvent() (int, string) {
	buffer := string(p.buffer)
	candidates := []struct {
		pattern string
		typ     string
	}{
		{`{"content":`, "content"},
		{`{"reasoning_content":`, "reasoning"},
		{`{"usage":`, "usage"},
	}
	best := -1
	bestType := ""
	for _, candidate := range candidates {
		pos := strings.Index(buffer, candidate.pattern)
		if pos >= 0 && (best < 0 || pos < best) {
			best = pos
			bestType = candidate.typ
		}
	}
	return best, bestType
}

func parseKiroEventStreamFrame(frame []byte) ([]kiroParsedEvent, bool) {
	if len(frame) < 16 {
		return nil, false
	}
	totalLength := int(binary.BigEndian.Uint32(frame[0:4]))
	headersLength := int(binary.BigEndian.Uint32(frame[4:8]))
	if totalLength != len(frame) || headersLength < 0 || 12+headersLength > len(frame)-4 {
		return nil, false
	}
	headers := map[string]string{}
	offset := 12
	headerEnd := 12 + headersLength
	for offset < headerEnd {
		nameLen := int(frame[offset])
		offset++
		if offset+nameLen > headerEnd {
			return nil, false
		}
		name := string(frame[offset : offset+nameLen])
		offset += nameLen
		if offset >= headerEnd {
			return nil, false
		}
		headerType := frame[offset]
		offset++
		if headerType != 7 {
			return nil, false
		}
		if offset+2 > headerEnd {
			return nil, false
		}
		valueLen := int(binary.BigEndian.Uint16(frame[offset : offset+2]))
		offset += 2
		if offset+valueLen > headerEnd {
			return nil, false
		}
		headers[name] = string(frame[offset : offset+valueLen])
		offset += valueLen
	}
	payloadStart := 12 + headersLength
	payloadEnd := len(frame) - 4
	if payloadEnd <= payloadStart {
		return nil, true
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(frame[payloadStart:payloadEnd]), &payload); err != nil {
		return nil, true
	}
	eventType := headers[":event-type"]
	switch eventType {
	case "assistantResponseEvent", "codeEvent":
		content := stringFromMap(payload, "content")
		if content == "" {
			return nil, true
		}
		return []kiroParsedEvent{{Type: "content", Content: content}}, true
	case "reasoningContentEvent":
		content := kiroReasoningContent(payload)
		if content == "" {
			return nil, true
		}
		return []kiroParsedEvent{{Type: "reasoning", Content: content}}, true
	case "toolUseEvent":
		calls := kiroToolCallsFromPayload(payload)
		if len(calls) == 0 {
			return nil, true
		}
		return []kiroParsedEvent{{Type: "tool_calls", ToolCalls: calls}}, true
	case "metricsEvent":
		return []kiroParsedEvent{{Type: "usage", Usage: payload}}, true
	case "contextUsageEvent":
		return []kiroParsedEvent{{Type: "context_usage", Usage: payload}}, true
	case "meteringEvent":
		return []kiroParsedEvent{{Type: "metering", Usage: payload}}, true
	default:
		return nil, true
	}
}

func kiroReasoningContent(payload map[string]any) string {
	if text := stringFromMap(payload, "text"); text != "" {
		return text
	}
	if text := stringFromMap(payload, "content"); text != "" {
		return text
	}
	if nested, ok := payload["reasoningContentEvent"].(map[string]any); ok {
		if text := stringFromMap(nested, "text"); text != "" {
			return text
		}
		return stringFromMap(nested, "content")
	}
	return ""
}

func kiroToolCallsFromPayload(payload map[string]any) []kiroToolCall {
	if len(payload) == 0 {
		return nil
	}
	items := []any{payload}
	for _, key := range []string{"toolUseEvent", "toolUses", "toolUse"} {
		if raw, ok := payload[key]; ok {
			if arr, ok := raw.([]any); ok {
				items = arr
			} else {
				items = []any{raw}
			}
			break
		}
	}
	out := make([]kiroToolCall, 0, len(items))
	for _, item := range items {
		obj, _ := item.(map[string]any)
		name := stringFromMap(obj, "name")
		if name == "" {
			continue
		}
		id := stringFromMap(obj, "toolUseId")
		if id == "" {
			id = "call_" + uuid.NewString()
		}
		out = append(out, kiroToolCall{ID: id, Name: name, Arguments: kiroToolArguments(obj["input"])})
	}
	return out
}

func kiroToolArguments(input any) string {
	switch typed := input.(type) {
	case string:
		return typed
	case nil:
		return "{}"
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return "{}"
		}
		return string(raw)
	}
}

func findKiroJSONEnd(raw []byte, start int) int {
	text := string(raw)
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func kiroEventsToOpenAINonStream(model string, events []kiroParsedEvent, detail coreusage.Detail, hasUsage bool, toolNameMap map[string]string) []byte {
	var content strings.Builder
	toolCalls := make([]any, 0)
	toolAccumulators := map[string]*kiroToolCallAccumulator{}
	toolOrder := make([]string, 0)
	for _, event := range events {
		if event.Type == "content" {
			content.WriteString(event.Content)
		}
		if event.Type == "tool_calls" {
			for _, call := range event.ToolCalls {
				acc := toolAccumulators[call.ID]
				if acc == nil {
					acc = &kiroToolCallAccumulator{}
					toolAccumulators[call.ID] = acc
					toolOrder = append(toolOrder, call.ID)
				}
				if call.Name != "" {
					acc.Name = restoreKiroToolName(call.Name, toolNameMap)
				}
				switch call.Arguments {
				case "":
					continue
				case "{}":
					if acc.ArgumentsBuilder.Len() == 0 {
						acc.FallbackArgument = call.Arguments
					}
				default:
					acc.ArgumentsBuilder.WriteString(call.Arguments)
				}
			}
		}
	}
	for _, id := range toolOrder {
		acc := toolAccumulators[id]
		if acc == nil || acc.Name == "" {
			continue
		}
		arguments := acc.ArgumentsBuilder.String()
		if arguments == "" {
			arguments = acc.FallbackArgument
		}
		if arguments == "" {
			arguments = "{}"
		}
		toolCalls = append(toolCalls, map[string]any{
			"id":   id,
			"type": "function",
			"function": map[string]any{
				"name":      acc.Name,
				"arguments": arguments,
			},
		})
	}
	finishReason := "stop"
	message := map[string]any{
		"role":    "assistant",
		"content": content.String(),
	}
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		message["tool_calls"] = toolCalls
	}
	payload := map[string]any{
		"id":      "chatcmpl-" + uuid.NewString(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{map[string]any{
			"index":         0,
			"finish_reason": finishReason,
			"message":       message,
		}},
	}
	if hasUsage {
		payload["usage"] = kiroUsageDetailMap(detail)
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func kiroOpenAIStreamFinishPayload(id, model string, created int64, finishReason string, detail coreusage.Detail) []byte {
	if finishReason == "" {
		finishReason = "stop"
	}
	payload := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []any{map[string]any{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": finishReason,
		}},
		"usage": kiroUsageDetailMap(detail),
	}
	raw, _ := json.Marshal(payload)
	return append(append([]byte("data: "), raw...), []byte("\n\n")...)
}

func kiroUsageDetailMap(detail coreusage.Detail) map[string]any {
	detail = normalizeKiroUsageDetail(detail)
	out := map[string]any{
		"prompt_tokens":     detail.InputTokens,
		"completion_tokens": detail.OutputTokens,
		"total_tokens":      detail.TotalTokens,
	}
	if detail.ReasoningTokens > 0 {
		out["completion_tokens_details"] = map[string]any{"reasoning_tokens": detail.ReasoningTokens}
	}
	if detail.CachedTokens > 0 {
		out["prompt_tokens_details"] = map[string]any{"cached_tokens": detail.CachedTokens}
	}
	return out
}

func normalizeKiroUsageDetail(detail coreusage.Detail) coreusage.Detail {
	if detail.TotalTokens == 0 {
		detail.TotalTokens = detail.InputTokens + detail.OutputTokens + detail.ReasoningTokens
	}
	return detail
}

func kiroUsageDetailNonZero(detail coreusage.Detail) bool {
	return detail.InputTokens != 0 ||
		detail.OutputTokens != 0 ||
		detail.ReasoningTokens != 0 ||
		detail.CachedTokens != 0 ||
		detail.TotalTokens != 0
}

func kiroUsageDetailFromPayload(raw any) (coreusage.Detail, bool) {
	source, ok := raw.(map[string]any)
	if !ok || source == nil {
		return coreusage.Detail{}, false
	}
	metrics := source
	if nested, ok := source["metricsEvent"].(map[string]any); ok {
		metrics = nested
	}
	input := numberPtr(firstMapValue(metrics, "inputTokens", "input_tokens", "prompt_tokens"))
	output := numberPtr(firstMapValue(metrics, "outputTokens", "output_tokens", "completion_tokens"))
	total := numberPtr(firstMapValue(metrics, "totalTokens", "total_tokens"))
	detail := coreusage.Detail{}
	if input != nil {
		detail.InputTokens = int64(*input)
	}
	if output != nil {
		detail.OutputTokens = int64(*output)
	}
	if total != nil {
		detail.TotalTokens = int64(*total)
	}
	detail = normalizeKiroUsageDetail(detail)
	return detail, kiroUsageDetailNonZero(detail)
}

func kiroContextUsagePercentage(raw any) (float64, bool) {
	source, ok := raw.(map[string]any)
	if !ok || source == nil {
		return 0, false
	}
	if nested, ok := source["contextUsageEvent"].(map[string]any); ok {
		source = nested
	}
	value := numberPtr(firstMapValue(source, "contextUsagePercentage", "context_usage_percentage"))
	if value == nil {
		return 0, false
	}
	return *value, true
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func kiroContentEventToOpenAIStream(id, model string, created int64, content string, first bool) [][]byte {
	delta := map[string]any{"content": content}
	if first {
		delta["role"] = "assistant"
	}
	payload := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []any{map[string]any{
			"index":         0,
			"delta":         delta,
			"finish_reason": nil,
		}},
	}
	raw, _ := json.Marshal(payload)
	return [][]byte{append(append([]byte("data: "), raw...), []byte("\n\n")...)}
}

func kiroReasoningEventToOpenAIStream(id, model string, created int64, content string, first bool) [][]byte {
	delta := map[string]any{"reasoning_content": content}
	if first {
		delta["role"] = "assistant"
	}
	payload := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []any{map[string]any{
			"index":         0,
			"delta":         delta,
			"finish_reason": nil,
		}},
	}
	raw, _ := json.Marshal(payload)
	return [][]byte{append(append([]byte("data: "), raw...), []byte("\n\n")...)}
}

func kiroToolCallEventToOpenAIStream(id, model string, created int64, calls []kiroToolCall, first bool, indexes map[string]int, nextIndex *int, toolNameMap map[string]string) [][]byte {
	chunks := make([][]byte, 0, len(calls)*2)
	for _, call := range calls {
		index, ok := indexes[call.ID]
		if !ok {
			index = *nextIndex
			*nextIndex++
			indexes[call.ID] = index
			delta := map[string]any{
				"tool_calls": []any{map[string]any{
					"index": index,
					"id":    call.ID,
					"type":  "function",
					"function": map[string]any{
						"name":      restoreKiroToolName(call.Name, toolNameMap),
						"arguments": "",
					},
				}},
			}
			if first {
				delta["role"] = "assistant"
				first = false
			}
			chunks = append(chunks, kiroOpenAIStreamPayload(id, model, created, delta))
		}
		delta := map[string]any{
			"tool_calls": []any{map[string]any{
				"index": index,
				"function": map[string]any{
					"arguments": call.Arguments,
				},
			}},
		}
		chunks = append(chunks, kiroOpenAIStreamPayload(id, model, created, delta))
	}
	return chunks
}

func kiroOpenAIStreamPayload(id, model string, created int64, delta map[string]any) []byte {
	payload := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []any{map[string]any{
			"index":         0,
			"delta":         delta,
			"finish_reason": nil,
		}},
	}
	raw, _ := json.Marshal(payload)
	return append(append([]byte("data: "), raw...), []byte("\n\n")...)
}

func kiroProviderSpecificData(auth *cliproxyauth.Auth) map[string]any {
	out := map[string]any{}
	if auth == nil || auth.Metadata == nil {
		return out
	}
	for _, key := range []string{"client_id", "client_secret", "region", "refresh_url"} {
		if value := metadataString(auth, key); value != "" {
			out[key] = value
		}
	}
	return out
}

func metadataString(auth *cliproxyauth.Auth, key string) string {
	if auth == nil || auth.Metadata == nil {
		return ""
	}
	return kiroStringValue(auth.Metadata[key])
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return kiroStringValue(m[key])
}

func kiroStringValue(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func kiroFingerprint(auth *cliproxyauth.Auth) string {
	seed := ""
	if auth != nil {
		seed = metadataString(auth, "client_id")
		if seed == "" {
			seed = metadataString(auth, "refresh_token")
		}
		if seed == "" {
			seed = metadataString(auth, "profile_arn")
		}
		if seed == "" {
			seed = metadataString(auth, "access_token")
		}
	}
	if seed == "" {
		seed = "kiro-anonymous"
	}
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}
