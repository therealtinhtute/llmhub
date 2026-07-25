package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	xaiauth "github.com/therealtinhtute/llmhub/internal/auth/xai"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/runtime/executor/helps"
	"github.com/therealtinhtute/llmhub/internal/thinking"
	"github.com/therealtinhtute/llmhub/internal/util"
	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
	sdktranslator "github.com/therealtinhtute/llmhub/sdk/translator"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/tiktoken-go/tokenizer"
)

var (
	xaiDataTag  = []byte("data:")
	xaiEventTag = []byte("event:")
)

const (
	xaiImageHandlerType         = "openai-image"
	xaiVideoHandlerType         = "openai-video"
	xaiCustomToolType           = "custom"
	xaiFunctionToolType         = "function"
	xaiImageGenerationToolType  = "image_generation"
	xaiNamespaceToolType        = "namespace"
	xaiToolSearchType           = "tool_search"
	xaiWebSearchToolType        = "web_search"
	xaiImagesGenerationsPath    = "/images/generations"
	xaiImagesEditsPath          = "/images/edits"
	xaiDefaultImageEndpointPath = xaiImagesGenerationsPath
	xaiVideosGenerationsPath    = "/videos/generations"
	xaiVideosEditsPath          = "/videos/edits"
	xaiVideosExtensionsPath     = "/videos/extensions"
	xaiVideosPath               = "/videos"
	xaiIdempotencyKeyMetaKey    = "idempotency_key"
)

// XAIExecutor is a stateless executor for xAI Grok's Responses API.
type XAIExecutor struct {
	cfg *config.Config
}

// NewXAIExecutor creates a new xAI executor.
func NewXAIExecutor(cfg *config.Config) *XAIExecutor {
	return &XAIExecutor{cfg: cfg}
}

// Identifier returns the provider identifier.
func (e *XAIExecutor) Identifier() string {
	return "xai"
}

// PrepareRequest injects xAI credentials into the outgoing HTTP request.
func (e *XAIExecutor) PrepareRequest(req *http.Request, auth *cliproxyauth.Auth) error {
	if req == nil {
		return nil
	}
	token, _ := xaiCreds(auth)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	var attrs map[string]string
	if auth != nil {
		attrs = auth.Attributes
	}
	util.ApplyCustomHeadersFromAttrs(req, attrs)
	return nil
}

// HttpRequest injects xAI credentials into the request and executes it.
func (e *XAIExecutor) HttpRequest(ctx context.Context, auth *cliproxyauth.Auth, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("xai executor: request is nil")
	}
	if ctx == nil {
		ctx = req.Context()
	}
	httpReq := req.WithContext(ctx)
	if errPrepare := e.PrepareRequest(httpReq, auth); errPrepare != nil {
		return nil, errPrepare
	}
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	return httpClient.Do(httpReq)
}

func (e *XAIExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	if endpointPath := xaiImageEndpointPath(opts); endpointPath != "" {
		return e.executeImages(ctx, auth, req, endpointPath)
	}
	if xaiIsVideoRequest(opts) {
		return e.executeVideos(ctx, auth, req, opts)
	}

	token, baseURL := xaiCreds(auth)
	if baseURL == "" {
		baseURL = xaiauth.DefaultAPIBaseURL
	}

	prepared, err := e.prepareResponsesRequest(ctx, req, opts, true)
	if err != nil {
		return resp, err
	}

	reporter := helps.NewUsageReporter(ctx, e.Identifier(), prepared.baseModel, auth)
	defer reporter.TrackFailure(ctx, &err)

	url := strings.TrimSuffix(baseURL, "/") + "/responses"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(prepared.body))
	if err != nil {
		return resp, err
	}
	applyXAIHeaders(httpReq, auth, token, true, prepared.sessionID)
	e.recordXAIRequest(ctx, auth, url, httpReq.Header.Clone(), prepared.body)

	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		helps.RecordAPIResponseError(ctx, e.cfg, err)
		return resp, err
	}
	defer func() {
		if errClose := httpResp.Body.Close(); errClose != nil {
			log.Errorf("xai executor: close response body error: %v", errClose)
		}
	}()
	helps.RecordAPIResponseMetadata(ctx, e.cfg, httpResp.StatusCode, httpResp.Header.Clone())
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		data, errRead := io.ReadAll(httpResp.Body)
		if errRead != nil {
			helps.RecordAPIResponseError(ctx, e.cfg, errRead)
			return resp, errRead
		}
		helps.AppendAPIResponseChunk(ctx, e.cfg, data)
		helps.LogWithRequestID(ctx).Debugf("request error, error status: %d, error message: %s", httpResp.StatusCode, helps.SummarizeErrorBody(httpResp.Header.Get("Content-Type"), data))
		return resp, statusErr{code: httpResp.StatusCode, msg: string(data)}
	}

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		helps.RecordAPIResponseError(ctx, e.cfg, err)
		return resp, err
	}
	helps.AppendAPIResponseChunk(ctx, e.cfg, data)

	outputItemsByIndex := make(map[int64][]byte)
	var outputItemsFallback [][]byte
	responseFilter := newXAIInternalXSearchResponseFilter(prepared.declarations)
	responseRestorer := newXAIToolResponseRestorer(prepared.declarations)
	for _, line := range bytes.Split(data, []byte("\n")) {
		if !bytes.HasPrefix(line, xaiDataTag) {
			continue
		}
		eventData := xaiNormalizeReasoningSummaryData(bytes.TrimSpace(line[len(xaiDataTag):]))
		eventData = responseFilter.apply(eventData)
		if len(eventData) == 0 {
			continue
		}
		eventData = responseRestorer.apply(eventData)
		switch gjson.GetBytes(eventData, "type").String() {
		case "response.output_item.done":
			xaiCollectOutputItemDone(eventData, outputItemsByIndex, &outputItemsFallback)
		case "response.completed":
			if detail, ok := helps.ParseCodexUsage(eventData); ok {
				reporter.Publish(ctx, detail)
			}
			completedData := xaiPatchCompletedOutput(eventData, outputItemsByIndex, outputItemsFallback)
			completedData = xaiNormalizeReasoningSummaryData(completedData)
			var param any
			out := sdktranslator.TranslateNonStream(ctx, prepared.to, prepared.from, req.Model, prepared.originalPayload, prepared.body, completedData, &param)
			return cliproxyexecutor.Response{Payload: out, Headers: httpResp.Header.Clone()}, nil
		}
	}

	return resp, statusErr{code: http.StatusRequestTimeout, msg: "xai stream error: stream disconnected before response.completed"}
}

func (e *XAIExecutor) executeImages(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, endpointPath string) (resp cliproxyexecutor.Response, err error) {
	token, baseURL := xaiCreds(auth)
	if baseURL == "" {
		baseURL = xaiauth.DefaultAPIBaseURL
	}
	if endpointPath == "" {
		endpointPath = xaiDefaultImageEndpointPath
	}

	url := strings.TrimSuffix(baseURL, "/") + endpointPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(req.Payload))
	if err != nil {
		return resp, err
	}
	applyXAIHeaders(httpReq, auth, token, false, "")
	e.recordXAIRequest(ctx, auth, url, httpReq.Header.Clone(), req.Payload)

	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		helps.RecordAPIResponseError(ctx, e.cfg, err)
		return resp, err
	}
	defer func() {
		if errClose := httpResp.Body.Close(); errClose != nil {
			log.Errorf("xai executor: close response body error: %v", errClose)
		}
	}()
	helps.RecordAPIResponseMetadata(ctx, e.cfg, httpResp.StatusCode, httpResp.Header.Clone())

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		helps.RecordAPIResponseError(ctx, e.cfg, err)
		return resp, err
	}
	helps.AppendAPIResponseChunk(ctx, e.cfg, data)

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		helps.LogWithRequestID(ctx).Debugf("request error, error status: %d, error message: %s", httpResp.StatusCode, helps.SummarizeErrorBody(httpResp.Header.Get("Content-Type"), data))
		return resp, statusErr{code: httpResp.StatusCode, msg: string(data)}
	}

	return cliproxyexecutor.Response{Payload: data, Headers: httpResp.Header.Clone()}, nil
}

func (e *XAIExecutor) executeVideos(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	token, baseURL := xaiCreds(auth)
	if baseURL == "" {
		baseURL = xaiauth.DefaultAPIBaseURL
	}

	method := http.MethodPost
	endpointPath := xaiVideosGenerationsPath
	var body io.Reader = bytes.NewReader(req.Payload)

	switch path := xaiVideoEndpointPath(opts); path {
	case xaiVideosGenerationsPath, xaiVideosEditsPath, xaiVideosExtensionsPath:
		endpointPath = path
	default:
		if requestID := strings.TrimSpace(gjson.GetBytes(req.Payload, "request_id").String()); requestID != "" {
			method = http.MethodGet
			endpointPath = xaiVideosPath + "/" + url.PathEscape(requestID)
			body = nil
		}
	}
	requestURL := strings.TrimSuffix(baseURL, "/") + endpointPath
	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return resp, err
	}
	applyXAIHeaders(httpReq, auth, token, false, "")
	if method == http.MethodPost {
		key := xaiMetadataString(opts.Metadata, xaiIdempotencyKeyMetaKey)
		if key == "" && opts.Headers != nil {
			key = strings.TrimSpace(opts.Headers.Get("x-idempotency-key"))
		}
		if key != "" {
			httpReq.Header.Set("x-idempotency-key", key)
		}
	}
	e.recordXAIRequest(ctx, auth, requestURL, httpReq.Header.Clone(), req.Payload)

	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		helps.RecordAPIResponseError(ctx, e.cfg, err)
		return resp, err
	}
	defer func() {
		if errClose := httpResp.Body.Close(); errClose != nil {
			log.Errorf("xai executor: close response body error: %v", errClose)
		}
	}()
	helps.RecordAPIResponseMetadata(ctx, e.cfg, httpResp.StatusCode, httpResp.Header.Clone())

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		helps.RecordAPIResponseError(ctx, e.cfg, err)
		return resp, err
	}
	helps.AppendAPIResponseChunk(ctx, e.cfg, data)

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		helps.LogWithRequestID(ctx).Debugf("request error, error status: %d, error message: %s", httpResp.StatusCode, helps.SummarizeErrorBody(httpResp.Header.Get("Content-Type"), data))
		return resp, statusErr{code: httpResp.StatusCode, msg: string(data)}
	}

	return cliproxyexecutor.Response{Payload: data, Headers: httpResp.Header.Clone()}, nil
}

func (e *XAIExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (_ *cliproxyexecutor.StreamResult, err error) {
	token, baseURL := xaiCreds(auth)
	if baseURL == "" {
		baseURL = xaiauth.DefaultAPIBaseURL
	}

	prepared, err := e.prepareResponsesRequest(ctx, req, opts, true)
	if err != nil {
		return nil, err
	}

	reporter := helps.NewUsageReporter(ctx, e.Identifier(), prepared.baseModel, auth)
	defer reporter.TrackFailure(ctx, &err)

	url := strings.TrimSuffix(baseURL, "/") + "/responses"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(prepared.body))
	if err != nil {
		return nil, err
	}
	applyXAIHeaders(httpReq, auth, token, true, prepared.sessionID)
	e.recordXAIRequest(ctx, auth, url, httpReq.Header.Clone(), prepared.body)

	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		helps.RecordAPIResponseError(ctx, e.cfg, err)
		return nil, err
	}
	helps.RecordAPIResponseMetadata(ctx, e.cfg, httpResp.StatusCode, httpResp.Header.Clone())
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		data, errRead := io.ReadAll(httpResp.Body)
		if errClose := httpResp.Body.Close(); errClose != nil {
			log.Errorf("xai executor: close response body error: %v", errClose)
		}
		if errRead != nil {
			helps.RecordAPIResponseError(ctx, e.cfg, errRead)
			return nil, errRead
		}
		helps.AppendAPIResponseChunk(ctx, e.cfg, data)
		helps.LogWithRequestID(ctx).Debugf("request error, error status: %d, error message: %s", httpResp.StatusCode, helps.SummarizeErrorBody(httpResp.Header.Get("Content-Type"), data))
		return nil, statusErr{code: httpResp.StatusCode, msg: string(data)}
	}

	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		defer func() {
			if errClose := httpResp.Body.Close(); errClose != nil {
				log.Errorf("xai executor: close response body error: %v", errClose)
			}
		}()
		scanner := bufio.NewScanner(httpResp.Body)
		scanner.Buffer(nil, 52_428_800)
		claudeInputTokens := helps.NewClaudeInputTokenState(prepared.from, prepared.to, prepared.from, prepared.originalPayload)
		var param any
		outputItemsByIndex := make(map[int64][]byte)
		var outputItemsFallback [][]byte
		responseFilter := newXAIInternalXSearchResponseFilter(prepared.declarations)
		responseRestorer := newXAIToolResponseRestorer(prepared.declarations)
		var pendingEventLine []byte
		emitTranslatedLine := func(translatedLine []byte) bool {
			chunks := helps.TranslateStreamWithClaudeInputTokens(ctx, prepared.to, prepared.from, req.Model, prepared.originalPayload, prepared.body, translatedLine, &param, claudeInputTokens)
			for i := range chunks {
				select {
				case out <- cliproxyexecutor.StreamChunk{Payload: chunks[i]}:
				case <-ctx.Done():
					return false
				}
			}
			return true
		}
		for scanner.Scan() {
			line := scanner.Bytes()
			helps.AppendAPIResponseChunk(ctx, e.cfg, line)

			if bytes.HasPrefix(line, xaiEventTag) {
				if pendingEventLine != nil && !emitTranslatedLine(xaiNormalizeReasoningSummaryEventLine(pendingEventLine, "")) {
					return
				}
				pendingEventLine = bytes.Clone(line)
				continue
			}

			if bytes.HasPrefix(line, xaiDataTag) {
				eventDataList := xaiNormalizeReasoningSummaryDataEvents(bytes.TrimSpace(line[len(xaiDataTag):]))
				hasPendingEventLine := pendingEventLine != nil
				for i, eventData := range eventDataList {
					eventData = responseFilter.apply(eventData)
					if len(eventData) == 0 {
						if hasPendingEventLine && i == 0 {
							pendingEventLine = nil
						}
						continue
					}
					eventData = responseRestorer.apply(eventData)
					normalizedEventName := gjson.GetBytes(eventData, "type").String()
					switch normalizedEventName {
					case "response.output_item.done":
						xaiCollectOutputItemDone(eventData, outputItemsByIndex, &outputItemsFallback)
					case "response.completed":
						if detail, ok := helps.ParseCodexUsage(eventData); ok {
							reporter.Publish(ctx, detail)
						}
						eventData = xaiPatchCompletedOutput(eventData, outputItemsByIndex, outputItemsFallback)
						eventData = xaiNormalizeReasoningSummaryData(eventData)
						normalizedEventName = gjson.GetBytes(eventData, "type").String()
					}

					if hasPendingEventLine {
						eventLine := []byte("event: " + normalizedEventName)
						if i == 0 {
							eventLine = xaiNormalizeReasoningSummaryEventLine(pendingEventLine, normalizedEventName)
							pendingEventLine = nil
						}
						if !emitTranslatedLine(eventLine) {
							return
						}
					}
					if !emitTranslatedLine(append([]byte("data: "), eventData...)) {
						return
					}
				}
				continue
			}

			if pendingEventLine != nil {
				if !emitTranslatedLine(xaiNormalizeReasoningSummaryEventLine(pendingEventLine, "")) {
					return
				}
				pendingEventLine = nil
			}
			if !emitTranslatedLine(bytes.Clone(line)) {
				return
			}
		}
		if pendingEventLine != nil {
			emitTranslatedLine(xaiNormalizeReasoningSummaryEventLine(pendingEventLine, ""))
		}
		if errScan := scanner.Err(); errScan != nil {
			helps.RecordAPIResponseError(ctx, e.cfg, errScan)
			reporter.PublishFailure(ctx, errScan)
			select {
			case out <- cliproxyexecutor.StreamChunk{Err: errScan}:
			case <-ctx.Done():
			}
		}
	}()
	return &cliproxyexecutor.StreamResult{Headers: httpResp.Header.Clone(), Chunks: out}, nil
}

// CountTokens estimates token count for xAI Responses requests.
func (e *XAIExecutor) CountTokens(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	prepared, err := e.prepareResponsesRequest(ctx, req, opts, false)
	if err != nil {
		return cliproxyexecutor.Response{}, err
	}
	enc, err := tokenizer.Get(tokenizer.O200kBase)
	if err != nil {
		return cliproxyexecutor.Response{}, fmt.Errorf("xai executor: tokenizer init failed: %w", err)
	}
	count, err := countXAIInputTokens(enc, prepared.body)
	if err != nil {
		return cliproxyexecutor.Response{}, fmt.Errorf("xai executor: token counting failed: %w", err)
	}
	usageJSON := fmt.Sprintf(`{"response":{"usage":{"input_tokens":%d,"output_tokens":0,"total_tokens":%d}}}`, count, count)
	translated := sdktranslator.TranslateTokenCount(ctx, prepared.to, prepared.from, count, []byte(usageJSON))
	return cliproxyexecutor.Response{Payload: translated}, nil
}

func countXAIInputTokens(enc tokenizer.Codec, body []byte) (int64, error) {
	if enc == nil {
		return 0, fmt.Errorf("encoder is nil")
	}
	if len(body) == 0 {
		return 0, nil
	}

	root := gjson.ParseBytes(body)
	segments := make([]string, 0, 32)
	xaiAppendTokenString(&segments, root.Get("instructions"))
	xaiCollectInputTokenSegments(root.Get("input"), &segments)
	xaiCollectToolTokenSegments(root.Get("tools"), &segments)
	if textFormat := root.Get("text.format"); textFormat.Exists() {
		xaiAppendTokenString(&segments, textFormat.Get("name"))
		xaiAppendTokenJSON(&segments, textFormat.Get("schema"))
	}
	if len(segments) == 0 {
		return 0, nil
	}
	count, err := enc.Count(strings.Join(segments, "\n"))
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func xaiCollectInputTokenSegments(input gjson.Result, segments *[]string) {
	if input.Type == gjson.String {
		xaiAppendTokenString(segments, input)
		return
	}
	if !input.IsArray() {
		return
	}
	for _, item := range input.Array() {
		switch item.Get("type").String() {
		case "message":
			xaiCollectContentTokenSegments(item.Get("content"), segments)
		case "function_call":
			xaiAppendTokenString(segments, item.Get("name"))
			xaiAppendTokenJSON(segments, item.Get("arguments"))
		case "function_call_output":
			xaiAppendTokenJSON(segments, item.Get("output"))
		case "reasoning":
			for _, part := range item.Get("summary").Array() {
				xaiAppendTokenString(segments, part.Get("text"))
			}
		}
	}
}

func xaiCollectContentTokenSegments(content gjson.Result, segments *[]string) {
	if content.Type == gjson.String {
		xaiAppendTokenString(segments, content)
		return
	}
	if !content.IsArray() {
		return
	}
	for _, part := range content.Array() {
		switch part.Get("type").String() {
		case "text", "input_text", "output_text":
			xaiAppendTokenString(segments, part.Get("text"))
		case "refusal":
			xaiAppendTokenString(segments, part.Get("refusal"))
		case "input_image":
			xaiAppendTokenString(segments, part.Get("image_url"))
			xaiAppendTokenString(segments, part.Get("file_id"))
		case "input_file":
			xaiAppendTokenString(segments, part.Get("file_data"))
			xaiAppendTokenString(segments, part.Get("file_url"))
			xaiAppendTokenString(segments, part.Get("file_id"))
			xaiAppendTokenString(segments, part.Get("filename"))
		case "input_audio":
			xaiAppendTokenString(segments, part.Get("data"))
			xaiAppendTokenString(segments, part.Get("input_audio.data"))
		}
	}
}

func xaiCollectToolTokenSegments(tools gjson.Result, segments *[]string) {
	if !tools.IsArray() {
		return
	}
	for _, tool := range tools.Array() {
		if tool.Get("type").String() != xaiFunctionToolType {
			continue
		}
		xaiAppendTokenString(segments, tool.Get("name"))
		xaiAppendTokenString(segments, tool.Get("description"))
		xaiAppendTokenJSON(segments, tool.Get("parameters"))
	}
}

func xaiAppendTokenString(segments *[]string, value gjson.Result) {
	if text := strings.TrimSpace(value.String()); text != "" {
		*segments = append(*segments, text)
	}
}

func xaiAppendTokenJSON(segments *[]string, value gjson.Result) {
	if !value.Exists() {
		return
	}
	if value.Type == gjson.String {
		xaiAppendTokenString(segments, value)
		return
	}
	if text := strings.TrimSpace(value.Raw); text != "" {
		*segments = append(*segments, text)
	}
}

// Refresh refreshes xAI OAuth credentials using the stored refresh token.
func (e *XAIExecutor) Refresh(ctx context.Context, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, error) {
	log.Debugf("xai executor: refresh called")
	if refreshed, handled, err := helps.RefreshAuthViaHome(ctx, e.cfg, auth); handled {
		return refreshed, err
	}
	if auth == nil {
		return nil, statusErr{code: http.StatusInternalServerError, msg: "xai executor: auth is nil"}
	}
	refreshToken := xaiMetadataString(auth.Metadata, "refresh_token")
	if refreshToken == "" {
		return auth, nil
	}
	tokenEndpoint := xaiMetadataString(auth.Metadata, "token_endpoint")
	svc := xaiauth.NewXAIAuthWithProxyURL(e.cfg, auth.ProxyURL)
	td, err := svc.RefreshTokens(ctx, refreshToken, tokenEndpoint)
	if err != nil {
		return nil, err
	}
	if auth.Metadata == nil {
		auth.Metadata = make(map[string]any)
	}
	auth.Metadata["type"] = "xai"
	auth.Metadata["auth_kind"] = "oauth"
	auth.Metadata["access_token"] = td.AccessToken
	if td.RefreshToken != "" {
		auth.Metadata["refresh_token"] = td.RefreshToken
	}
	if td.IDToken != "" {
		auth.Metadata["id_token"] = td.IDToken
	}
	if td.TokenType != "" {
		auth.Metadata["token_type"] = td.TokenType
	}
	if td.ExpiresIn > 0 {
		auth.Metadata["expires_in"] = td.ExpiresIn
	}
	if td.Expire != "" {
		auth.Metadata["expired"] = td.Expire
	}
	if td.Email != "" {
		auth.Metadata["email"] = td.Email
	}
	if td.Subject != "" {
		auth.Metadata["sub"] = td.Subject
	}
	if tokenEndpoint != "" {
		auth.Metadata["token_endpoint"] = tokenEndpoint
	}
	if xaiMetadataString(auth.Metadata, "base_url") == "" {
		auth.Metadata["base_url"] = xaiauth.DefaultAPIBaseURL
	}
	auth.Metadata["last_refresh"] = time.Now().UTC().Format(time.RFC3339)
	if auth.Attributes == nil {
		auth.Attributes = make(map[string]string)
	}
	auth.Attributes["auth_kind"] = "oauth"
	if strings.TrimSpace(auth.Attributes["base_url"]) == "" {
		auth.Attributes["base_url"] = xaiauth.DefaultAPIBaseURL
	}
	return auth, nil
}

type xaiPreparedRequest struct {
	baseModel       string
	from            sdktranslator.Format
	to              sdktranslator.Format
	originalPayload []byte
	body            []byte
	declarations    *xaiToolDeclarations
	sessionID       string
}

func (e *XAIExecutor) prepareResponsesRequest(ctx context.Context, req cliproxyexecutor.Request, opts cliproxyexecutor.Options, stream bool) (*xaiPreparedRequest, error) {
	baseModel := thinking.ParseSuffix(req.Model).ModelName
	from := opts.SourceFormat
	to := sdktranslator.FromString("codex")
	originalPayloadSource := req.Payload
	if len(opts.OriginalRequest) > 0 {
		originalPayloadSource = opts.OriginalRequest
	}
	originalPayload := bytes.Clone(originalPayloadSource)
	originalTranslated := sdktranslator.TranslateRequest(from, to, baseModel, originalPayload, stream)
	body := sdktranslator.TranslateRequest(from, to, baseModel, bytes.Clone(req.Payload), stream)

	var err error
	body, err = thinking.ApplyThinking(body, req.Model, from.String(), e.Identifier(), e.Identifier())
	if err != nil {
		return nil, err
	}

	requestedModel := helps.PayloadRequestedModel(opts, req.Model)
	requestPath := helps.PayloadRequestPath(opts)
	body = helps.ApplyPayloadConfigWithRequest(e.cfg, baseModel, to.String(), from.String(), "", body, originalTranslated, requestedModel, requestPath, opts.Headers)
	body, _ = sjson.SetBytes(body, "model", baseModel)
	body, _ = sjson.SetBytes(body, "stream", stream)
	body, _ = sjson.DeleteBytes(body, "previous_response_id")
	body, _ = sjson.DeleteBytes(body, "prompt_cache_retention")
	body, _ = sjson.DeleteBytes(body, "safety_identifier")
	body, _ = sjson.DeleteBytes(body, "stream_options")
	declarationSource := body
	if from == sdktranslator.FormatOpenAIResponse {
		declarationSource = originalPayload
	}
	declarations, err := buildXAIToolDeclarations(declarationSource)
	if err != nil {
		return nil, err
	}
	if from == sdktranslator.FormatOpenAIResponse {
		translatedDeclarations, errTranslated := buildXAIToolDeclarations(body)
		if errTranslated != nil {
			return nil, errTranslated
		}
		declarations = mergeXAIToolDeclarations(declarations, translatedDeclarations)
	}
	body = normalizeXAIToolsWithDeclarations(body, declarations)
	body = promoteXAIAdditionalTools(body)
	body = normalizeXAIToolChoices(body, declarations)
	body = normalizeXAIInputReasoningItems(body)
	body = normalizeCodexInstructions(body)
	body = sanitizeXAIResponsesBody(body, baseModel)

	sessionID := xaiExecutionSessionID(req, opts)
	if sessionID != "" {
		body, _ = sjson.SetBytes(body, "prompt_cache_key", sessionID)
	}

	return &xaiPreparedRequest{
		baseModel:       baseModel,
		from:            from,
		to:              to,
		originalPayload: originalPayload,
		body:            body,
		declarations:    declarations,
		sessionID:       sessionID,
	}, nil
}

func (e *XAIExecutor) recordXAIRequest(ctx context.Context, auth *cliproxyauth.Auth, url string, headers http.Header, body []byte) {
	var authID, authLabel, authType, authValue string
	if auth != nil {
		authID = auth.ID
		authLabel = auth.Label
		authType, authValue = auth.AccountInfo()
	}
	helps.RecordAPIRequest(ctx, e.cfg, helps.UpstreamRequestLog{
		URL:       url,
		Method:    http.MethodPost,
		Headers:   headers,
		Body:      body,
		Provider:  e.Identifier(),
		AuthID:    authID,
		AuthLabel: authLabel,
		AuthType:  authType,
		AuthValue: authValue,
	})
}

func xaiCreds(auth *cliproxyauth.Auth) (token, baseURL string) {
	if auth == nil {
		return "", ""
	}
	if auth.Attributes != nil {
		token = strings.TrimSpace(auth.Attributes["api_key"])
		baseURL = strings.TrimSpace(auth.Attributes["base_url"])
	}
	if auth.Metadata != nil {
		if token == "" {
			token = xaiMetadataString(auth.Metadata, "access_token")
		}
		if baseURL == "" {
			baseURL = xaiMetadataString(auth.Metadata, "base_url")
		}
	}
	return token, baseURL
}

func applyXAIHeaders(r *http.Request, auth *cliproxyauth.Auth, token string, stream bool, sessionID string) {
	r.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if stream {
		r.Header.Set("Accept", "text/event-stream")
	} else {
		r.Header.Set("Accept", "application/json")
	}
	r.Header.Set("Connection", "Keep-Alive")
	if sessionID != "" {
		r.Header.Set("x-grok-conv-id", sessionID)
	}
	var attrs map[string]string
	if auth != nil {
		attrs = auth.Attributes
	}
	util.ApplyCustomHeadersFromAttrs(r, attrs)
}

func xaiExecutionSessionID(req cliproxyexecutor.Request, opts cliproxyexecutor.Options) string {
	if value := xaiMetadataString(opts.Metadata, cliproxyexecutor.ExecutionSessionMetadataKey); value != "" {
		return value
	}
	if value := xaiMetadataString(req.Metadata, cliproxyexecutor.ExecutionSessionMetadataKey); value != "" {
		return value
	}
	if promptCacheKey := gjson.GetBytes(req.Payload, "prompt_cache_key"); promptCacheKey.Exists() {
		return strings.TrimSpace(promptCacheKey.String())
	}
	return ""
}

func xaiImageEndpointPath(opts cliproxyexecutor.Options) string {
	if opts.SourceFormat.String() != xaiImageHandlerType {
		return ""
	}

	path := xaiMetadataString(opts.Metadata, cliproxyexecutor.RequestPathMetadataKey)
	if strings.HasSuffix(path, "/images/edits") {
		return xaiImagesEditsPath
	}
	if strings.HasSuffix(path, "/images/generations") {
		return xaiImagesGenerationsPath
	}
	return xaiDefaultImageEndpointPath
}

func xaiIsVideoRequest(opts cliproxyexecutor.Options) bool {
	return opts.SourceFormat.String() == xaiVideoHandlerType
}

func xaiVideoEndpointPath(opts cliproxyexecutor.Options) string {
	if !xaiIsVideoRequest(opts) {
		return ""
	}
	path := xaiMetadataString(opts.Metadata, cliproxyexecutor.RequestPathMetadataKey)
	if strings.HasSuffix(path, "/videos/edits") {
		return xaiVideosEditsPath
	}
	if strings.HasSuffix(path, "/videos/extensions") {
		return xaiVideosExtensionsPath
	}
	if strings.HasSuffix(path, "/videos/generations") {
		return xaiVideosGenerationsPath
	}
	return ""
}

func xaiMetadataString(meta map[string]any, key string) string {
	if len(meta) == 0 || key == "" {
		return ""
	}
	value, ok := meta[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func sanitizeXAIResponsesBody(body []byte, model string) []byte {
	body = removeXAIEncryptedReasoningInclude(body)
	if !xaiSupportsReasoningEffort(model) {
		body, _ = sjson.DeleteBytes(body, "reasoning")
	}
	return body
}

type xaiToolIdentity struct {
	namespace string
	name      string
	toolType  string
}

type xaiToolDeclaration struct {
	original      xaiToolIdentity
	effectiveName string
	effectiveType string
}

type xaiToolDeclarations struct {
	byOriginal      map[xaiToolIdentity]xaiToolDeclaration
	byEffectiveName map[string]xaiToolDeclaration
}

func buildXAIToolDeclarations(body []byte) (*xaiToolDeclarations, error) {
	declarations := &xaiToolDeclarations{
		byOriginal:      make(map[xaiToolIdentity]xaiToolDeclaration),
		byEffectiveName: make(map[string]xaiToolDeclaration),
	}
	collect := func(tools gjson.Result, namespace string) error {
		return collectXAIToolDeclarations(tools, namespace, declarations)
	}
	if err := collect(gjson.GetBytes(body, "tools"), ""); err != nil {
		return nil, err
	}
	input := gjson.GetBytes(body, "input")
	if input.IsArray() {
		for _, item := range input.Array() {
			if item.Get("type").String() != "additional_tools" {
				continue
			}
			if err := collect(item.Get("tools"), ""); err != nil {
				return nil, err
			}
		}
	}
	return declarations, nil
}

func mergeXAIToolDeclarations(primary, fallback *xaiToolDeclarations) *xaiToolDeclarations {
	if primary == nil {
		return fallback
	}
	if fallback == nil {
		return primary
	}
	for effectiveName, declaration := range fallback.byEffectiveName {
		if _, exists := primary.byEffectiveName[effectiveName]; exists {
			continue
		}
		primary.byEffectiveName[effectiveName] = declaration
		primary.byOriginal[declaration.original] = declaration
	}
	return primary
}

func collectXAIToolDeclarations(tools gjson.Result, namespace string, declarations *xaiToolDeclarations) error {
	if declarations == nil || !tools.IsArray() {
		return nil
	}
	for _, tool := range tools.Array() {
		toolType := tool.Get("type").String()
		if toolType == xaiNamespaceToolType {
			if err := collectXAIToolDeclarations(tool.Get("tools"), tool.Get("name").String(), declarations); err != nil {
				return err
			}
			continue
		}
		if toolType != xaiFunctionToolType && toolType != xaiCustomToolType {
			continue
		}
		name := tool.Get("name").String()
		if name == "" || (toolType == xaiCustomToolType && name == "apply_patch") {
			continue
		}
		identity := xaiToolIdentity{namespace: namespace, name: name, toolType: toolType}
		effectiveName := name
		if namespace != "" {
			effectiveName = qualifyXAINamespaceToolName(namespace, name)
		}
		declaration := xaiToolDeclaration{
			original:      identity,
			effectiveName: effectiveName,
			effectiveType: xaiFunctionToolType,
		}
		if existing, ok := declarations.byEffectiveName[effectiveName]; ok && existing.original != identity {
			return newXAIToolNameCollisionError(existing, declaration)
		}
		declarations.byOriginal[identity] = declaration
		declarations.byEffectiveName[effectiveName] = declaration
	}
	return nil
}

func newXAIToolNameCollisionError(existing, incoming xaiToolDeclaration) error {
	message := fmt.Sprintf("tool declarations %s and %s resolve to the same outbound name %q", xaiToolIdentityLabel(existing.original), xaiToolIdentityLabel(incoming.original), incoming.effectiveName)
	payload, _ := json.Marshal(map[string]any{
		"error": map[string]string{
			"message": message,
			"type":    "invalid_request_error",
			"code":    "tool_name_collision",
		},
	})
	return statusErr{code: http.StatusBadRequest, msg: string(payload)}
}

func xaiToolIdentityLabel(identity xaiToolIdentity) string {
	name := identity.name
	if identity.namespace != "" {
		name = identity.namespace + "." + name
	}
	return fmt.Sprintf("%q (%s)", name, identity.toolType)
}

func qualifyXAINamespaceToolName(namespace, name string) string {
	if namespace == "" || name == "" || strings.HasPrefix(name, "mcp__") {
		return name
	}
	prefix := namespace
	if !strings.HasSuffix(prefix, "__") {
		prefix += "__"
	}
	if strings.HasPrefix(name, prefix) {
		return name
	}
	return prefix + name
}

func normalizeXAITools(body []byte) []byte {
	declarations, _ := buildXAIToolDeclarations(body)
	return normalizeXAIToolsWithDeclarations(body, declarations)
}

func normalizeXAIToolsWithDeclarations(body []byte, declarations *xaiToolDeclarations) []byte {
	original := body
	normalizeAtPath := func(path string) bool {
		tools := gjson.GetBytes(body, path)
		if !tools.IsArray() {
			return true
		}
		filtered, changed, ok := normalizeXAIToolArray(tools, "", declarations)
		if !ok {
			return false
		}
		if !changed {
			return true
		}
		updated, errSet := sjson.SetRawBytes(body, path, filtered)
		if errSet != nil {
			return false
		}
		body = updated
		return true
	}
	if !normalizeAtPath("tools") {
		return original
	}
	input := gjson.GetBytes(body, "input")
	if input.IsArray() {
		for index, item := range input.Array() {
			if item.Get("type").String() != "additional_tools" {
				continue
			}
			if !normalizeAtPath(fmt.Sprintf("input.%d.tools", index)) {
				return original
			}
		}
	}
	return body
}

func normalizeXAIToolArray(tools gjson.Result, namespace string, declarations *xaiToolDeclarations) ([]byte, bool, bool) {
	filtered := make([][]byte, 0, len(tools.Array()))
	changed := false
	for _, tool := range tools.Array() {
		if tool.Get("type").String() == xaiNamespaceToolType {
			changed = true
			nestedNamespace := tool.Get("name").String()
			for _, nestedTool := range tool.Get("tools").Array() {
				raw, nestedChanged, ok := normalizeXAIToolWithDeclaration(nestedTool, nestedNamespace, declarations)
				if !ok {
					return nil, false, false
				}
				changed = changed || nestedChanged
				if len(raw) > 0 {
					filtered = append(filtered, raw)
				}
			}
			continue
		}
		raw, toolChanged, ok := normalizeXAIToolWithDeclaration(tool, namespace, declarations)
		if !ok {
			return nil, false, false
		}
		changed = changed || toolChanged
		if len(raw) > 0 {
			filtered = append(filtered, raw)
		}
	}
	if !changed {
		return nil, false, true
	}
	return xaiJoinRawJSONArray(filtered), true, true
}

func xaiJoinRawJSONArray(items [][]byte) []byte {
	var joined bytes.Buffer
	joined.WriteByte('[')
	for index, item := range items {
		if index > 0 {
			joined.WriteByte(',')
		}
		joined.Write(item)
	}
	joined.WriteByte(']')
	return joined.Bytes()
}

func normalizeXAITool(tool gjson.Result) ([]byte, bool, bool) {
	return normalizeXAIToolWithDeclaration(tool, "", nil)
}

func normalizeXAIToolWithDeclaration(tool gjson.Result, namespace string, declarations *xaiToolDeclarations) ([]byte, bool, bool) {
	toolType := tool.Get("type").String()
	changed := false
	if toolType == xaiToolSearchType || toolType == xaiImageGenerationToolType {
		return nil, true, true
	}
	raw := []byte(tool.Raw)
	if toolType == xaiCustomToolType {
		if tool.Get("name").String() == "apply_patch" {
			return nil, true, true
		}
		updatedTool, errSet := sjson.SetBytes(raw, "type", xaiFunctionToolType)
		if errSet != nil {
			return nil, false, false
		}
		raw = updatedTool
		toolType = xaiFunctionToolType
		changed = true
	}
	if toolType == xaiWebSearchToolType && tool.Get("external_web_access").Exists() {
		updatedTool, errDel := sjson.DeleteBytes(raw, "external_web_access")
		if errDel != nil {
			return nil, false, false
		}
		raw = updatedTool
		changed = true
	}
	if toolType == xaiFunctionToolType && !tool.Get("parameters").Exists() {
		updatedTool, errSet := sjson.SetRawBytes(raw, "parameters", []byte(`{"type":"object","properties":{}}`))
		if errSet != nil {
			return nil, false, false
		}
		raw = updatedTool
		changed = true
	}
	originalType := tool.Get("type").String()
	if declarations != nil && (originalType == xaiFunctionToolType || originalType == xaiCustomToolType) {
		identity := xaiToolIdentity{namespace: namespace, name: tool.Get("name").String(), toolType: originalType}
		if declaration, ok := declarations.byOriginal[identity]; ok && declaration.effectiveName != identity.name {
			updatedTool, errSet := sjson.SetBytes(raw, "name", declaration.effectiveName)
			if errSet != nil {
				return nil, false, false
			}
			raw = updatedTool
			changed = true
		}
	}
	return raw, changed, true
}

func promoteXAIAdditionalTools(body []byte) []byte {
	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		return body
	}
	inputItems := input.Array()
	remainingInput := make([]json.RawMessage, 0, len(inputItems))
	promotedTools := make([]json.RawMessage, 0)
	for _, item := range inputItems {
		if item.Get("type").String() != "additional_tools" {
			remainingInput = append(remainingInput, json.RawMessage(item.Raw))
			continue
		}
		for _, tool := range item.Get("tools").Array() {
			promotedTools = append(promotedTools, json.RawMessage(tool.Raw))
		}
	}
	if len(remainingInput) == len(inputItems) {
		return body
	}
	rawInput, errMarshal := json.Marshal(remainingInput)
	if errMarshal != nil {
		return body
	}
	updated, errSet := sjson.SetRawBytes(body, "input", rawInput)
	if errSet != nil || len(promotedTools) == 0 {
		return updated
	}
	tools := make([]json.RawMessage, 0, len(gjson.GetBytes(updated, "tools").Array())+len(promotedTools))
	for _, tool := range gjson.GetBytes(updated, "tools").Array() {
		tools = append(tools, json.RawMessage(tool.Raw))
	}
	tools = append(tools, promotedTools...)
	rawTools, errMarshal := json.Marshal(tools)
	if errMarshal != nil {
		return body
	}
	updated, errSet = sjson.SetRawBytes(updated, "tools", rawTools)
	if errSet != nil {
		return body
	}
	return updated
}

func normalizeXAIToolChoices(body []byte, declarations *xaiToolDeclarations) []byte {
	body = normalizeXAIToolChoiceAtPath(body, "tool_choice", declarations)
	for index := range gjson.GetBytes(body, "tool_choice.tools").Array() {
		body = normalizeXAIToolChoiceAtPath(body, fmt.Sprintf("tool_choice.tools.%d", index), declarations)
	}
	return body
}

func normalizeXAIToolChoiceAtPath(body []byte, path string, declarations *xaiToolDeclarations) []byte {
	if declarations == nil {
		return body
	}
	choice := gjson.GetBytes(body, path)
	if !choice.IsObject() {
		return body
	}
	identity := xaiToolIdentity{
		namespace: choice.Get("namespace").String(),
		name:      choice.Get("name").String(),
		toolType:  choice.Get("type").String(),
	}
	declaration, ok := declarations.byOriginal[identity]
	if !ok && identity.namespace == "" {
		candidate, found := declarations.byEffectiveName[identity.name]
		if found && (identity.toolType == candidate.original.toolType || identity.toolType == candidate.effectiveType) {
			declaration, ok = candidate, true
		}
	}
	if !ok {
		return body
	}
	updated, errSet := sjson.SetBytes(body, path+".name", declaration.effectiveName)
	if errSet != nil {
		return body
	}
	updated, errSet = sjson.SetBytes(updated, path+".type", declaration.effectiveType)
	if errSet != nil {
		return body
	}
	updated, errSet = sjson.DeleteBytes(updated, path+".namespace")
	if errSet != nil {
		return body
	}
	return updated
}

type xaiToolResponseRestorer struct {
	declarations    *xaiToolDeclarations
	declarationByID map[string]xaiToolDeclaration
}

func newXAIToolResponseRestorer(declarations *xaiToolDeclarations) *xaiToolResponseRestorer {
	return &xaiToolResponseRestorer{
		declarations:    declarations,
		declarationByID: make(map[string]xaiToolDeclaration),
	}
}

func (r *xaiToolResponseRestorer) apply(eventData []byte) []byte {
	if r == nil || r.declarations == nil || len(eventData) == 0 || !gjson.ValidBytes(eventData) {
		return eventData
	}
	if item := gjson.GetBytes(eventData, "item"); item.IsObject() {
		eventData = r.restoreItemAtPath(eventData, "item")
	}
	for index := range gjson.GetBytes(eventData, "response.output").Array() {
		eventData = r.restoreItemAtPath(eventData, fmt.Sprintf("response.output.%d", index))
	}
	return r.restoreCustomToolInputEvent(eventData)
}

func (r *xaiToolResponseRestorer) restoreItemAtPath(eventData []byte, path string) []byte {
	item := gjson.GetBytes(eventData, path)
	declaration, ok := r.declarations.matchResponseItem(item)
	if !ok {
		return eventData
	}
	if id := item.Get("id").String(); id != "" {
		r.declarationByID[id] = declaration
	}
	updated, errSet := sjson.SetBytes(eventData, path+".name", declaration.original.name)
	if errSet != nil {
		return eventData
	}
	if declaration.original.namespace != "" {
		updated, errSet = sjson.SetBytes(updated, path+".namespace", declaration.original.namespace)
		if errSet != nil {
			return eventData
		}
	}
	if declaration.original.toolType != xaiCustomToolType {
		return updated
	}
	updated, errSet = sjson.SetBytes(updated, path+".type", "custom_tool_call")
	if errSet != nil {
		return eventData
	}
	if arguments := gjson.GetBytes(updated, path+".arguments"); arguments.Exists() {
		updated, errSet = sjson.SetBytes(updated, path+".input", arguments.String())
		if errSet != nil {
			return eventData
		}
		updated, errSet = sjson.DeleteBytes(updated, path+".arguments")
		if errSet != nil {
			return eventData
		}
	}
	return updated
}

func (r *xaiToolResponseRestorer) restoreCustomToolInputEvent(eventData []byte) []byte {
	eventType := gjson.GetBytes(eventData, "type").String()
	if eventType != "response.function_call_arguments.delta" && eventType != "response.function_call_arguments.done" {
		return eventData
	}
	declaration, ok := r.declarationByID[gjson.GetBytes(eventData, "item_id").String()]
	if !ok || declaration.original.toolType != xaiCustomToolType {
		return eventData
	}
	if eventType == "response.function_call_arguments.delta" {
		updated, errSet := sjson.SetBytes(eventData, "type", "response.custom_tool_call_input.delta")
		if errSet != nil {
			return eventData
		}
		return updated
	}
	updated, errSet := sjson.SetBytes(eventData, "type", "response.custom_tool_call_input.done")
	if errSet != nil {
		return eventData
	}
	if arguments := gjson.GetBytes(updated, "arguments"); arguments.Exists() {
		updated, errSet = sjson.SetBytes(updated, "input", arguments.String())
		if errSet != nil {
			return eventData
		}
		updated, errSet = sjson.DeleteBytes(updated, "arguments")
		if errSet != nil {
			return eventData
		}
	}
	return updated
}

func (d *xaiToolDeclarations) matchResponseItem(item gjson.Result) (xaiToolDeclaration, bool) {
	if d == nil || item.Get("type").String() != "function_call" {
		return xaiToolDeclaration{}, false
	}
	name := item.Get("name").String()
	if namespace := item.Get("namespace").String(); namespace != "" {
		for _, toolType := range []string{xaiFunctionToolType, xaiCustomToolType} {
			if declaration, ok := d.byOriginal[xaiToolIdentity{namespace: namespace, name: name, toolType: toolType}]; ok {
				return declaration, true
			}
		}
		return xaiToolDeclaration{}, false
	}
	declaration, ok := d.byEffectiveName[name]
	if !ok || declaration.effectiveType != xaiFunctionToolType {
		return xaiToolDeclaration{}, false
	}
	return declaration, true
}

type xaiInternalXSearchResponseFilter struct {
	declarations         *xaiToolDeclarations
	droppedOutputIndexes map[int64]struct{}
	droppedItemIDs       map[string]struct{}
}

func newXAIInternalXSearchResponseFilter(declarations *xaiToolDeclarations) *xaiInternalXSearchResponseFilter {
	return &xaiInternalXSearchResponseFilter{
		declarations:         declarations,
		droppedOutputIndexes: make(map[int64]struct{}),
		droppedItemIDs:       make(map[string]struct{}),
	}
}

func xaiIsInternalXSearchToolName(name string) bool {
	switch name {
	case "x_user_search", "x_semantic_search", "x_keyword_search", "x_thread_fetch":
		return true
	default:
		return false
	}
}

func xaiIsInternalXSearchCallID(callID string) bool {
	return strings.HasPrefix(callID, "xs_call")
}

func (f *xaiInternalXSearchResponseFilter) isInternalCall(item gjson.Result) bool {
	itemType := item.Get("type").String()
	if itemType != "function_call" && itemType != "custom_tool_call" {
		return false
	}
	if !xaiIsInternalXSearchToolName(item.Get("name").String()) {
		return false
	}
	if f != nil && f.declarations != nil && item.Get("namespace").String() != "" {
		if _, declared := f.declarations.matchResponseItem(item); declared {
			return false
		}
	}
	if xaiIsInternalXSearchCallID(item.Get("call_id").String()) {
		return true
	}
	if f != nil && f.declarations != nil {
		if _, declared := f.declarations.matchResponseItem(item); declared {
			return false
		}
	}
	return true
}

func (f *xaiInternalXSearchResponseFilter) apply(eventData []byte) []byte {
	if f == nil || len(eventData) == 0 || !gjson.ValidBytes(eventData) {
		return eventData
	}
	if item := gjson.GetBytes(eventData, "item"); f.isInternalCall(item) {
		f.recordDroppedItem(eventData, item)
		return nil
	}
	eventData = f.filterCompletedOutput(eventData)
	if f.referencesDroppedItem(eventData) {
		return nil
	}
	return f.compactOutputIndex(eventData)
}

func (f *xaiInternalXSearchResponseFilter) recordDroppedItem(eventData []byte, item gjson.Result) {
	if outputIndex := gjson.GetBytes(eventData, "output_index"); outputIndex.Exists() {
		f.droppedOutputIndexes[outputIndex.Int()] = struct{}{}
	}
	for _, path := range []string{"id", "call_id"} {
		if id := item.Get(path).String(); id != "" {
			f.droppedItemIDs[id] = struct{}{}
		}
	}
}

func (f *xaiInternalXSearchResponseFilter) referencesDroppedItem(eventData []byte) bool {
	if outputIndex := gjson.GetBytes(eventData, "output_index"); outputIndex.Exists() {
		if _, dropped := f.droppedOutputIndexes[outputIndex.Int()]; dropped {
			return true
		}
	}
	for _, path := range []string{"item_id", "call_id"} {
		id := gjson.GetBytes(eventData, path).String()
		if _, dropped := f.droppedItemIDs[id]; id != "" && dropped {
			return true
		}
	}
	return false
}

func (f *xaiInternalXSearchResponseFilter) compactOutputIndex(eventData []byte) []byte {
	outputIndex := gjson.GetBytes(eventData, "output_index")
	if !outputIndex.Exists() {
		return eventData
	}
	original := outputIndex.Int()
	removedBefore := int64(0)
	for dropped := range f.droppedOutputIndexes {
		if dropped < original {
			removedBefore++
		}
	}
	if removedBefore == 0 {
		return eventData
	}
	updated, errSet := sjson.SetBytes(eventData, "output_index", original-removedBefore)
	if errSet != nil {
		return eventData
	}
	return updated
}

func (f *xaiInternalXSearchResponseFilter) filterCompletedOutput(eventData []byte) []byte {
	output := gjson.GetBytes(eventData, "response.output")
	if !output.IsArray() {
		return eventData
	}
	items := make([]json.RawMessage, 0, len(output.Array()))
	changed := false
	for _, item := range output.Array() {
		if f.isInternalCall(item) {
			changed = true
			continue
		}
		items = append(items, json.RawMessage(item.Raw))
	}
	if !changed {
		return eventData
	}
	rawOutput, errMarshal := json.Marshal(items)
	if errMarshal != nil {
		return eventData
	}
	updated, errSet := sjson.SetRawBytes(eventData, "response.output", rawOutput)
	if errSet != nil {
		return eventData
	}
	return updated
}

func normalizeXAIInputReasoningItems(body []byte) []byte {
	input := gjson.GetBytes(body, "input")
	if !input.Exists() || !input.IsArray() {
		return body
	}

	updated := body
	for i, item := range input.Array() {
		if item.Get("type").String() != "reasoning" {
			continue
		}
		contentPath := fmt.Sprintf("input.%d.content", i)
		if content := gjson.GetBytes(updated, contentPath); content.Exists() && content.Type == gjson.Null {
			updatedBody, errDel := sjson.DeleteBytes(updated, contentPath)
			if errDel != nil {
				return body
			}
			updated = updatedBody
		}
		encryptedContentPath := fmt.Sprintf("input.%d.encrypted_content", i)
		if encryptedContent := gjson.GetBytes(updated, encryptedContentPath); encryptedContent.Exists() && encryptedContent.Type == gjson.Null {
			updatedBody, errDel := sjson.DeleteBytes(updated, encryptedContentPath)
			if errDel != nil {
				return body
			}
			updated = updatedBody
		}
	}
	return mergeAdjacentXAIInputReasoningSummaries(updated)
}

func mergeAdjacentXAIInputReasoningSummaries(body []byte) []byte {
	input := gjson.GetBytes(body, "input")
	if !input.Exists() || !input.IsArray() {
		return body
	}

	changed := false
	items := make([]json.RawMessage, 0, len(input.Array()))
	for _, item := range input.Array() {
		if len(items) > 0 && canMergeXAIReasoningSummary(items[len(items)-1], item) {
			merged, ok := appendXAIReasoningSummary(items[len(items)-1], item.Get("summary").Array())
			if ok {
				items[len(items)-1] = json.RawMessage(merged)
				changed = true
				continue
			}
		}
		items = append(items, json.RawMessage(item.Raw))
	}
	if !changed {
		return body
	}

	rawInput, errMarshal := json.Marshal(items)
	if errMarshal != nil {
		return body
	}
	updated, errSet := sjson.SetRawBytes(body, "input", rawInput)
	if errSet != nil {
		return body
	}
	return updated
}

func canMergeXAIReasoningSummary(previous json.RawMessage, current gjson.Result) bool {
	previousItem := gjson.ParseBytes(previous)
	if previousItem.Get("type").String() != "reasoning" || current.Get("type").String() != "reasoning" {
		return false
	}
	if !previousItem.Get("summary").IsArray() || !current.Get("summary").IsArray() {
		return false
	}
	if len(current.Get("summary").Array()) == 0 {
		return false
	}
	for name := range current.Map() {
		if name != "type" && name != "summary" {
			return false
		}
	}
	return true
}

func appendXAIReasoningSummary(previous json.RawMessage, currentSummary []gjson.Result) ([]byte, bool) {
	updated := []byte(previous)
	summary := gjson.GetBytes(updated, "summary")
	if !summary.IsArray() {
		return previous, false
	}
	nextIndex := len(summary.Array())
	for i, item := range currentSummary {
		updatedItem, errSet := sjson.SetRawBytes(updated, fmt.Sprintf("summary.%d", nextIndex+i), []byte(item.Raw))
		if errSet != nil {
			return previous, false
		}
		updated = updatedItem
	}
	return updated, true
}

func xaiNormalizeReasoningSummaryEventLine(line []byte, eventName string) []byte {
	if eventName == "" && bytes.HasPrefix(line, xaiEventTag) {
		eventName = strings.TrimSpace(string(line[len(xaiEventTag):]))
	}
	eventName = xaiNormalizeReasoningSummaryEventName(eventName)
	if eventName == "" {
		return bytes.Clone(line)
	}
	return []byte("event: " + eventName)
}

func xaiNormalizeReasoningSummaryEventName(eventName string) string {
	switch eventName {
	case "response.reasoning_text.delta":
		return "response.reasoning_summary_text.delta"
	case "response.reasoning_text.done":
		return "response.reasoning_summary_part.done"
	default:
		return eventName
	}
}

func xaiNormalizeReasoningSummaryData(eventData []byte) []byte {
	if len(eventData) == 0 || !gjson.ValidBytes(eventData) {
		return eventData
	}

	normalized := eventData
	switch gjson.GetBytes(normalized, "type").String() {
	case "response.reasoning_text.delta":
		normalized, _ = sjson.SetBytes(normalized, "type", "response.reasoning_summary_text.delta")
		normalized = xaiNormalizeReasoningSummaryIndex(normalized)
	case "response.reasoning_text.done":
		normalized, _ = sjson.SetBytes(normalized, "type", "response.reasoning_summary_part.done")
		normalized, _ = sjson.SetBytes(normalized, "part.type", "summary_text")
		if text := gjson.GetBytes(normalized, "text"); text.Exists() {
			normalized, _ = sjson.SetBytes(normalized, "part.text", text.String())
		}
		normalized, _ = sjson.DeleteBytes(normalized, "text")
		normalized = xaiNormalizeReasoningSummaryIndex(normalized)
	case "response.content_part.added":
		if gjson.GetBytes(normalized, "part.type").String() == "reasoning_text" {
			normalized, _ = sjson.SetBytes(normalized, "type", "response.reasoning_summary_part.added")
			normalized, _ = sjson.SetBytes(normalized, "part.type", "summary_text")
			normalized = xaiNormalizeReasoningSummaryIndex(normalized)
		}
	case "response.content_part.done":
		if gjson.GetBytes(normalized, "part.type").String() == "reasoning_text" {
			normalized, _ = sjson.SetBytes(normalized, "type", "response.reasoning_summary_part.done")
			normalized, _ = sjson.SetBytes(normalized, "part.type", "summary_text")
			normalized = xaiNormalizeReasoningSummaryIndex(normalized)
		}
	}

	if item := gjson.GetBytes(normalized, "item"); item.Exists() && item.Type == gjson.JSON {
		updatedItem := xaiNormalizeReasoningOutputItem([]byte(item.Raw))
		if !bytes.Equal(updatedItem, []byte(item.Raw)) {
			normalized, _ = sjson.SetRawBytes(normalized, "item", updatedItem)
		}
	}
	if output := gjson.GetBytes(normalized, "response.output"); output.IsArray() {
		updatedOutput, changed := xaiNormalizeReasoningOutputItems(output.Array())
		if changed {
			normalized, _ = sjson.SetRawBytes(normalized, "response.output", updatedOutput)
		}
	}
	return normalized
}

func xaiNormalizeReasoningSummaryDataEvents(eventData []byte) [][]byte {
	if len(eventData) == 0 || !gjson.ValidBytes(eventData) {
		return [][]byte{eventData}
	}
	if gjson.GetBytes(eventData, "type").String() != "response.reasoning_text.done" {
		return [][]byte{xaiNormalizeReasoningSummaryData(eventData)}
	}

	textDone, _ := sjson.SetBytes(eventData, "type", "response.reasoning_summary_text.done")
	textDone = xaiNormalizeReasoningSummaryIndex(textDone)
	partDone := xaiNormalizeReasoningSummaryData(eventData)
	return [][]byte{textDone, partDone}
}

func xaiNormalizeReasoningSummaryIndex(eventData []byte) []byte {
	contentIndex := gjson.GetBytes(eventData, "content_index")
	if contentIndex.Exists() && contentIndex.Raw != "" && !gjson.GetBytes(eventData, "summary_index").Exists() {
		eventData, _ = sjson.SetRawBytes(eventData, "summary_index", []byte(contentIndex.Raw))
	}
	eventData, _ = sjson.DeleteBytes(eventData, "content_index")
	return eventData
}

func xaiNormalizeReasoningOutputItems(items []gjson.Result) ([]byte, bool) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	changed := false
	for i, item := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		updatedItem := xaiNormalizeReasoningOutputItem([]byte(item.Raw))
		if !bytes.Equal(updatedItem, []byte(item.Raw)) {
			changed = true
		}
		buf.Write(updatedItem)
	}
	buf.WriteByte(']')
	return buf.Bytes(), changed
}

func xaiNormalizeReasoningOutputItem(item []byte) []byte {
	if !gjson.ValidBytes(item) || gjson.GetBytes(item, "type").String() != "reasoning" {
		return item
	}

	normalized := item
	if summary := gjson.GetBytes(normalized, "summary"); summary.IsArray() {
		updatedSummary, changed := xaiNormalizeReasoningSummaryItems(summary.Array())
		if changed {
			normalized, _ = sjson.SetRawBytes(normalized, "summary", updatedSummary)
		}
	}

	content := gjson.GetBytes(normalized, "content")
	if !content.IsArray() {
		return normalized
	}

	summaryItems := make([]gjson.Result, 0, len(content.Array()))
	for _, part := range content.Array() {
		if part.Get("type").String() == "reasoning_text" {
			summaryItems = append(summaryItems, part)
		}
	}
	if len(summaryItems) == 0 {
		return normalized
	}

	updatedSummary, _ := xaiNormalizeReasoningSummaryItems(summaryItems)
	normalized, _ = sjson.SetRawBytes(normalized, "summary", updatedSummary)
	normalized, _ = sjson.DeleteBytes(normalized, "content")
	return normalized
}

func xaiNormalizeReasoningSummaryItems(items []gjson.Result) ([]byte, bool) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	changed := false
	for i, item := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		itemRaw := []byte(item.Raw)
		if item.Get("type").String() == "reasoning_text" {
			var errSet error
			itemRaw, errSet = sjson.SetBytes(itemRaw, "type", "summary_text")
			if errSet == nil {
				changed = true
			}
		}
		buf.Write(itemRaw)
	}
	buf.WriteByte(']')
	return buf.Bytes(), changed
}

func removeXAIEncryptedReasoningInclude(body []byte) []byte {
	include := gjson.GetBytes(body, "include")
	if !include.Exists() || !include.IsArray() {
		return body
	}
	kept := make([]string, 0, len(include.Array()))
	for _, item := range include.Array() {
		value := strings.TrimSpace(item.String())
		if value == "" || value == "reasoning.encrypted_content" {
			continue
		}
		kept = append(kept, value)
	}
	body, _ = sjson.SetBytes(body, "include", kept)
	return body
}

func xaiSupportsReasoningEffort(model string) bool {
	name := strings.ToLower(strings.TrimSpace(thinking.ParseSuffix(model).ModelName))
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	switch {
	case strings.HasPrefix(name, "grok-3-mini"):
		return true
	case strings.HasPrefix(name, "grok-4.20-multi-agent"):
		return true
	case strings.HasPrefix(name, "grok-4.3"):
		return true
	default:
		return false
	}
}

func xaiCollectOutputItemDone(eventData []byte, outputItemsByIndex map[int64][]byte, outputItemsFallback *[][]byte) {
	itemResult := gjson.GetBytes(eventData, "item")
	if !itemResult.Exists() || itemResult.Type != gjson.JSON {
		return
	}
	outputIndexResult := gjson.GetBytes(eventData, "output_index")
	if outputIndexResult.Exists() {
		outputItemsByIndex[outputIndexResult.Int()] = []byte(itemResult.Raw)
		return
	}
	*outputItemsFallback = append(*outputItemsFallback, []byte(itemResult.Raw))
}

func xaiPatchCompletedOutput(eventData []byte, outputItemsByIndex map[int64][]byte, outputItemsFallback [][]byte) []byte {
	outputResult := gjson.GetBytes(eventData, "response.output")
	shouldPatchOutput := (!outputResult.Exists() || !outputResult.IsArray() || len(outputResult.Array()) == 0) && (len(outputItemsByIndex) > 0 || len(outputItemsFallback) > 0)
	if !shouldPatchOutput {
		return eventData
	}

	indexes := make([]int64, 0, len(outputItemsByIndex))
	for idx := range outputItemsByIndex {
		indexes = append(indexes, idx)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i] < indexes[j]
	})

	outputArray := []byte("[]")
	var buf bytes.Buffer
	buf.WriteByte('[')
	wrote := false
	for _, idx := range indexes {
		if wrote {
			buf.WriteByte(',')
		}
		buf.Write(outputItemsByIndex[idx])
		wrote = true
	}
	for _, item := range outputItemsFallback {
		if wrote {
			buf.WriteByte(',')
		}
		buf.Write(item)
		wrote = true
	}
	buf.WriteByte(']')
	if wrote {
		outputArray = buf.Bytes()
	}

	patched, _ := sjson.SetRawBytes(eventData, "response.output", outputArray)
	return patched
}
