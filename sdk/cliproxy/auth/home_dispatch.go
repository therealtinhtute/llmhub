package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/home"
	"github.com/therealtinhtute/llmhub/internal/logging"
	"github.com/therealtinhtute/llmhub/sdk/cliproxy/executionregistry"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

func (m *Manager) executeHome(ctx context.Context, req cliproxyexecutor.Request, opts cliproxyexecutor.Options, countTokens bool) (cliproxyexecutor.Response, error) {
	if unlockSession := m.lockHomeWebsocketSession(ctx, opts); unlockSession != nil {
		defer unlockSession()
	}
	routeModel := req.Model
	opts = ensureRequestedModelMetadata(opts, routeModel)
	tried := make(map[string]struct{})
	var lastErr error
	for count := 1; ; count++ {
		selection, errSelection := m.pickHomeDispatchSelection(ctx, routeModel, withHomeAuthCount(opts, count))
		if errSelection != nil {
			if lastErr != nil && isHomeRequestRetryExceededError(errSelection) {
				return cliproxyexecutor.Response{}, lastErr
			}
			return cliproxyexecutor.Response{}, errSelection
		}
		auth := selection.CloneAuthForRoute(routeModel)
		if auth == nil || selection.Executor == nil {
			selection.End("missing_execution_target")
			return cliproxyexecutor.Response{}, &Error{Code: "executor_not_found", Message: "executor not registered"}
		}
		if _, seen := tried[auth.ID]; seen {
			selection.End("repeated_auth")
			if lastErr != nil {
				return cliproxyexecutor.Response{}, lastErr
			}
			return cliproxyexecutor.Response{}, &Error{Code: homeRequestRetryExceededErrorCode, Message: "home returned a previously tried auth", HTTPStatus: http.StatusServiceUnavailable}
		}
		if errRuntimeAuth := m.bindHomeSelectionRuntimeAuth(ctx, opts, selection); errRuntimeAuth != nil {
			selection.End("runtime_auth_bind_failed")
			return cliproxyexecutor.Response{}, errRuntimeAuth
		}
		tried[auth.ID] = struct{}{}
		publishSelectedAuthMetadata(opts.Metadata, auth.ID)

		execCtx, releaseAttempt, errBind := selection.AttemptContext(ctx)
		if errBind != nil {
			selection.End("attempt_bind_failed")
			return cliproxyexecutor.Response{}, errBind
		}
		if rt := m.roundTripperFor(auth); rt != nil {
			execCtx = context.WithValue(execCtx, roundTripperContextKey{}, rt)
			execCtx = context.WithValue(execCtx, "cliproxy.roundtripper", rt)
		}
		execCtx = contextWithRequestedModelAlias(execCtx, opts, routeModel)

		models, pooled := m.preparedExecutionModels(auth, routeModel)
		if len(models) > 1 {
			models = models[:1]
			pooled = false
		}
		if len(models) == 0 {
			releaseAttempt()
			if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "no_execution_models"); errEnd != nil {
				return cliproxyexecutor.Response{}, errEnd
			}
			lastErr = &Error{Code: "auth_not_found", Message: "no execution models available"}
			continue
		}

		preparedAuth, errPrepare := m.prepareHomeRequestAuth(execCtx, selection.Executor, selection)
		if errPrepare != nil {
			m.reportHomeResult(execCtx, Result{AuthID: auth.ID, Provider: selection.Provider, Model: routeModel, Success: false, Error: resultErrorFromError(errPrepare), Options: opts}, auth)
			releaseAttempt()
			if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "prepare_failed"); errEnd != nil {
				return cliproxyexecutor.Response{}, errEnd
			}
			lastErr = errPrepare
			continue
		}

		upstreamModel := models[0]
		resultModel := m.stateModelForExecution(preparedAuth, routeModel, upstreamModel, pooled)
		execReq := req
		execReq.Model = upstreamModel
		execOpts := opts
		execOpts.ExecutionLifecycle = selection
		if errCtx := execCtx.Err(); errCtx != nil {
			releaseAttempt()
			selection.End("attempt_canceled")
			return cliproxyexecutor.Response{}, errCtx
		}

		var response cliproxyexecutor.Response
		var errExecute error
		if countTokens {
			response, errExecute = selection.Executor.CountTokens(execCtx, preparedAuth, execReq, execOpts)
		} else {
			response, errExecute = selection.Executor.Execute(execCtx, preparedAuth, execReq, execOpts)
		}
		result := Result{AuthID: preparedAuth.ID, Provider: selection.Provider, Model: resultModel, Success: errExecute == nil, Options: execOpts}
		if errExecute == nil {
			applyKiroUsageResultFromResponse(&result, response)
			m.reportHomeResult(execCtx, result, preparedAuth)
			releaseAttempt()
			if !m.retainHomeWebsocketSelection(ctx, opts, routeModel, selection) {
				selection.End("completed")
			}
			return response, nil
		}
		result.Error = resultErrorFromError(errExecute)
		result.RetryAfter = retryAfterFromError(errExecute)
		action, okAction := matchRequestScopedErrorAction(preparedAuth, errExecute, m.runtimeConfigSnapshot())
		applyRequestScopedActionToResult(action, okAction, &result)
		m.reportHomeResult(execCtx, result, preparedAuth)
		releaseAttempt()
		if okAction {
			if isRequestScopedStop(action, okAction) {
				selection.End("request_stopped")
				return cliproxyexecutor.Response{}, wrapRequestStopError(errExecute)
			}
			if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "execution_failed"); errEnd != nil {
				return cliproxyexecutor.Response{}, errEnd
			}
			if errCtx := execCtx.Err(); errCtx != nil && ctx != nil && ctx.Err() != nil {
				return cliproxyexecutor.Response{}, errCtx
			}
			lastErr = errExecute
			continue
		}
		if isRequestInvalidError(errExecute) {
			selection.End("request_invalid")
			return cliproxyexecutor.Response{}, errExecute
		}
		if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "execution_failed"); errEnd != nil {
			return cliproxyexecutor.Response{}, errEnd
		}
		if errCtx := execCtx.Err(); errCtx != nil && ctx != nil && ctx.Err() != nil {
			return cliproxyexecutor.Response{}, errCtx
		}
		lastErr = errExecute
	}
}

func (m *Manager) executeHomeStream(ctx context.Context, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	if unlockSession := m.lockHomeWebsocketSession(ctx, opts); unlockSession != nil {
		defer unlockSession()
	}
	routeModel := req.Model
	opts = ensureRequestedModelMetadata(opts, routeModel)
	tried := make(map[string]struct{})
	var lastErr error
	for count := 1; ; count++ {
		selection, errSelection := m.pickHomeDispatchSelection(ctx, routeModel, withHomeAuthCount(opts, count))
		if errSelection != nil {
			if lastErr != nil && isHomeRequestRetryExceededError(errSelection) {
				return nil, lastErr
			}
			return nil, errSelection
		}
		auth := selection.CloneAuthForRoute(routeModel)
		if auth == nil || selection.Executor == nil {
			selection.End("missing_execution_target")
			return nil, &Error{Code: "executor_not_found", Message: "executor not registered"}
		}
		if _, seen := tried[auth.ID]; seen {
			selection.End("repeated_auth")
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, &Error{Code: homeRequestRetryExceededErrorCode, Message: "home returned a previously tried auth", HTTPStatus: http.StatusServiceUnavailable}
		}
		if errRuntimeAuth := m.bindHomeSelectionRuntimeAuth(ctx, opts, selection); errRuntimeAuth != nil {
			selection.End("runtime_auth_bind_failed")
			return nil, errRuntimeAuth
		}
		tried[auth.ID] = struct{}{}
		publishSelectedAuthMetadata(opts.Metadata, auth.ID)

		execCtx, releaseAttempt, errBind := selection.AttemptContext(ctx)
		if errBind != nil {
			selection.End("attempt_bind_failed")
			return nil, errBind
		}
		if rt := m.roundTripperFor(auth); rt != nil {
			execCtx = context.WithValue(execCtx, roundTripperContextKey{}, rt)
			execCtx = context.WithValue(execCtx, "cliproxy.roundtripper", rt)
		}
		models, _ := m.preparedExecutionModels(auth, routeModel)
		if len(models) == 0 {
			releaseAttempt()
			if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "no_execution_models"); errEnd != nil {
				return nil, errEnd
			}
			lastErr = &Error{Code: "auth_not_found", Message: "no execution models available"}
			continue
		}
		preparedAuth, errPrepare := m.prepareHomeRequestAuth(execCtx, selection.Executor, selection)
		if errPrepare != nil {
			m.reportHomeResult(execCtx, Result{AuthID: auth.ID, Provider: selection.Provider, Model: routeModel, Success: false, Error: resultErrorFromError(errPrepare), Options: opts}, auth)
			releaseAttempt()
			if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "prepare_failed"); errEnd != nil {
				return nil, errEnd
			}
			lastErr = errPrepare
			continue
		}
		execReq := req
		execReq.Model = models[0]
		execOpts := opts
		execOpts.ExecutionLifecycle = selection
		streamResult, errStream := selection.Executor.ExecuteStream(execCtx, preparedAuth, execReq, execOpts)
		result := Result{AuthID: preparedAuth.ID, Provider: selection.Provider, Model: routeModel, Success: errStream == nil, Options: execOpts}
		if errStream != nil {
			result.Error = resultErrorFromError(errStream)
			result.RetryAfter = retryAfterFromError(errStream)
			m.reportHomeResult(execCtx, result, preparedAuth)
			releaseAttempt()
			if isRequestInvalidError(errStream) {
				selection.End("request_invalid")
				return nil, errStream
			}
			if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "stream_start_failed"); errEnd != nil {
				return nil, errEnd
			}
			lastErr = errStream
			continue
		}
		if streamResult == nil || streamResult.Chunks == nil {
			errStream = &Error{Code: "invalid_stream", Message: "executor returned no stream", HTTPStatus: http.StatusBadGateway}
			m.reportHomeResult(execCtx, Result{
				AuthID:   preparedAuth.ID,
				Provider: selection.Provider,
				Model:    routeModel,
				Success:  false,
				Error:    resultErrorFromError(errStream),
				Options:  execOpts,
			}, preparedAuth)
			releaseAttempt()
			selection.End("stream_missing")
			return nil, errStream
		}
		m.reportHomeResult(execCtx, result, preparedAuth)
		if m.retainHomeWebsocketSelection(ctx, opts, routeModel, selection) {
			return wrapHomeSelectionStream(execCtx, streamResult, nil, releaseAttempt), nil
		}
		return wrapHomeSelectionStream(execCtx, streamResult, selection, releaseAttempt), nil
	}
}

func wrapHomeSelectionStream(ctx context.Context, result *cliproxyexecutor.StreamResult, selection *HomeDispatchSelection, releaseAttempt func()) *cliproxyexecutor.StreamResult {
	if result == nil || result.Chunks == nil {
		if releaseAttempt != nil {
			releaseAttempt()
		}
		if selection != nil {
			selection.End("stream_missing")
		}
		return result
	}
	out := make(chan cliproxyexecutor.StreamChunk)
	wrapped := &cliproxyexecutor.StreamResult{Headers: result.Headers, Chunks: out}
	go func() {
		defer close(out)
		defer releaseAttempt()
		if selection != nil {
			defer selection.End("stream_closed")
		}
		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-result.Chunks:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- chunk:
				}
			}
		}
	}()
	return wrapped
}

func (m *Manager) prepareHomeRequestAuth(ctx context.Context, executor ProviderExecutor, selection *HomeDispatchSelection) (*Auth, error) {
	if m == nil || executor == nil || selection == nil {
		return nil, nil
	}
	auth := selection.CloneAuth()
	preparer, ok := executor.(RequestAuthPreparer)
	if !ok || preparer == nil || auth == nil || !preparer.ShouldPrepareRequestAuth(auth) {
		return auth, nil
	}

	prepare := func() (*Auth, error) {
		target := auth.Clone()
		if !preparer.ShouldPrepareRequestAuth(target) {
			return target, nil
		}
		updated, errPrepare := preparer.PrepareRequestAuth(ctx, target)
		if errPrepare != nil {
			return auth, errPrepare
		}
		if updated == nil {
			return target, nil
		}
		return updated, nil
	}

	id := strings.TrimSpace(auth.ID)
	if id == "" {
		return prepare()
	}
	lockValue, _ := m.requestPrepareLocks.LoadOrStore(id, &requestAuthPrepareLock{})
	lock, ok := lockValue.(*requestAuthPrepareLock)
	if !ok || lock == nil {
		return prepare()
	}
	lock.mu.Lock()
	defer lock.mu.Unlock()
	return prepare()
}

func (m *Manager) reportHomeResult(ctx context.Context, result Result, auth *Auth) {
	if m == nil || result.AuthID == "" {
		return
	}
	m.hook.OnResult(ctx, result)
}

func (m *Manager) endHomeSelectionBeforeRedispatch(ctx context.Context, selection *HomeDispatchSelection, reason string) error {
	if selection == nil {
		return nil
	}
	ticket := selection.EndWithRelease(reason)
	if ticket == nil {
		return nil
	}
	bound := internalconfig.CredentialConcurrencyConfig{}.WithDefaults().CPACancelBound
	if cfg, ok := m.runtimeConfig.Load().(*internalconfig.Config); ok && cfg != nil {
		bound = cfg.CredentialConcurrency.WithDefaults().CPACancelBound
	}
	waitCtx := ctx
	if waitCtx == nil {
		waitCtx = context.Background()
	}
	waitCtx, cancel := context.WithTimeout(waitCtx, bound)
	defer cancel()
	if errWait := ticket.Wait(waitCtx); errWait != nil {
		return &Error{Code: "home_unavailable", Message: fmt.Sprintf("Home did not acknowledge credential release: %v", errWait), Retryable: true, HTTPStatus: http.StatusServiceUnavailable}
	}
	return nil
}

func (m *Manager) pickHomeDispatchSelection(ctx context.Context, model string, opts cliproxyexecutor.Options) (*HomeDispatchSelection, error) {
	if m == nil {
		return nil, &Error{Code: "auth_not_found", Message: "no auth available"}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	requestedModel := requestedModelFromMetadata(opts.Metadata, model)
	retained, retainedOK, errRetained := m.retainedHomeSessionSelection(ctx, opts, requestedModel)
	if errRetained != nil {
		return nil, errRetained
	}
	if retainedOK {
		return retained, nil
	}
	if executionSessionID := homeExecutionSessionIDFromMetadata(opts.Metadata); executionSessionID != "" {
		if credentialID := pinnedAuthIDFromMetadata(opts.Metadata); credentialID != "" {
			if errEnd := m.endMismatchedHomeSessionSelections(ctx, executionSessionID, credentialID, requestedModel, true); errEnd != nil {
				return nil, errEnd
			}
		}
	}
	bundle := m.HomeDispatchBundle()
	if bundle == nil || bundle.client == nil || bundle.registry == nil {
		return nil, &Error{Code: "home_unavailable", Message: "home dispatch bundle unavailable", HTTPStatus: http.StatusServiceUnavailable}
	}
	client := bundle.client
	registry := bundle.registry
	if !client.HeartbeatOK() {
		return nil, &Error{Code: "home_unavailable", Message: "home control center unavailable", HTTPStatus: http.StatusServiceUnavailable}
	}

	pending, errBegin := registry.BeginDispatch()
	if errBegin != nil {
		return nil, &Error{Code: "home_unavailable", Message: "home execution registry unavailable", Retryable: true, HTTPStatus: http.StatusServiceUnavailable}
	}

	sessionID := m.homeDispatchSessionID(opts)
	raw, errRPop := client.RPopAuth(ctx, requestedModel, sessionID, homeDispatchHeaders(ctx, opts.Headers), homeAuthCountFromMetadata(opts.Metadata))
	if errRPop != nil {
		if home.IsAmbiguousDispatchError(errRPop) {
			client.AbortAmbiguousDispatch()
		}
		pending.End()
		if errors.Is(errRPop, home.ErrAuthNotFound) {
			return nil, &Error{Code: "auth_not_found", Message: errRPop.Error(), HTTPStatus: http.StatusServiceUnavailable}
		}
		return nil, &Error{Code: "home_unavailable", Message: errRPop.Error(), Retryable: true, HTTPStatus: http.StatusServiceUnavailable}
	}

	envelope, errEnvelope := decodeHomeDispatchConcurrencyEnvelope(raw)
	if errEnvelope != nil {
		if envelope.Present {
			client.AbortAmbiguousDispatch()
		}
		pending.End()
		if envelope.Present {
			return nil, invalidHomeConcurrencyResponse("Home returned malformed concurrency tuple")
		}
		return nil, &Error{Code: "invalid_auth", Message: "home returned invalid auth payload", HTTPStatus: http.StatusBadGateway}
	}

	kind := "http"
	if cliproxyexecutor.DownstreamWebsocket(ctx) {
		kind = "websocket"
	} else if opts.Stream {
		kind = "stream"
	}
	baseScope := executionregistry.ScopeSpec{
		RequestID: logging.GetRequestID(ctx),
		Model:     requestedModel,
		Kind:      kind,
		StartedAt: time.Now(),
	}
	var scope *executionregistry.Scope
	if envelope.Present {
		var errInstall error
		scope, errInstall = installHomeConcurrencyScope(registry, pending, envelope.Tuple, baseScope)
		if errInstall != nil {
			client.AbortAmbiguousDispatch()
			pending.End()
			return nil, homeConcurrencyInstallError(errInstall)
		}
	}
	endScope := func() {
		if scope != nil {
			scope.End("local_validation_failed")
			return
		}
		pending.End()
	}

	if errHome := decodeHomeDispatchError(raw); errHome != nil {
		if envelope.Present {
			client.AbortAmbiguousDispatch()
			endScope()
			return nil, invalidHomeConcurrencyResponse("Home returned both accounted concurrency and an error")
		}
		pending.End()
		return nil, errHome
	}

	var dispatch homeAuthDispatchResponse
	if errUnmarshal := json.Unmarshal(raw, &dispatch); errUnmarshal != nil {
		endScope()
		return nil, &Error{Code: "invalid_auth", Message: "home returned invalid auth payload", HTTPStatus: http.StatusBadGateway}
	}
	auth := dispatch.Auth
	if strings.TrimSpace(auth.ID) == "" {
		if errUnmarshal := json.Unmarshal(raw, &auth); errUnmarshal != nil {
			endScope()
			return nil, &Error{Code: "invalid_auth", Message: "home returned invalid auth payload", HTTPStatus: http.StatusBadGateway}
		}
	}

	observedModel := canonicalHomeDispatchModel(dispatch.Model, requestedModel)
	if envelope.Present {
		observedConcurrencyModel, validModel := validCanonicalHomeConcurrencyModelKey(observedModel)
		if !validModel || envelope.Tuple.Model != observedConcurrencyModel {
			client.AbortAmbiguousDispatch()
			endScope()
			return nil, invalidHomeConcurrencyResponse("Home concurrency model does not match dispatched model")
		}
	} else {
		baseScope.Model = observedModel
	}

	setHomeUserAPIKeyOnGinContext(ctx, dispatch.UserAPIKey)
	if upstreamModel := strings.TrimSpace(dispatch.Model); upstreamModel != "" {
		if auth.Attributes == nil {
			auth.Attributes = make(map[string]string, 3)
		}
		auth.Attributes[homeUpstreamModelAttributeKey] = upstreamModel
	}
	if originalAlias := strings.TrimSpace(dispatch.OriginalAlias); dispatch.ForceMapping && originalAlias != "" {
		if auth.Attributes == nil {
			auth.Attributes = make(map[string]string, 2)
		}
		auth.Attributes[homeForceMappingAttributeKey] = "true"
		auth.Attributes[homeOriginalAliasAttributeKey] = originalAlias
	}
	if strings.TrimSpace(auth.ID) == "" {
		endScope()
		return nil, &Error{Code: "invalid_auth", Message: "home returned auth without id", HTTPStatus: http.StatusBadGateway}
	}
	if errIdentity := verifyAccountedHomeConcurrencyIdentity(envelope.Tuple, &auth, dispatch.AuthIndex); errIdentity != nil {
		endScope()
		return nil, errIdentity
	}

	providerKey := strings.ToLower(strings.TrimSpace(auth.Provider))
	if providerKey == "" {
		endScope()
		return nil, &Error{Code: "invalid_auth", Message: "home returned auth without provider", HTTPStatus: http.StatusBadGateway}
	}
	if homeAuthIndex := strings.TrimSpace(dispatch.AuthIndex); homeAuthIndex != "" {
		auth.Index = homeAuthIndex
		auth.indexAssigned = true
	} else {
		auth.EnsureIndex()
	}

	executor, okExecutor := m.Executor(providerKey)
	if !okExecutor && auth.Attributes != nil && strings.TrimSpace(auth.Attributes["base_url"]) != "" {
		executor, okExecutor = m.Executor("openai-compatibility")
	}
	if !okExecutor {
		endScope()
		return nil, &Error{Code: "executor_not_found", Message: "executor not registered", HTTPStatus: http.StatusBadGateway}
	}
	if scope == nil {
		var errInstall error
		scope, errInstall = installHomeConcurrencyScope(registry, pending, homeConcurrencyTuple{}, executionregistry.ScopeSpec{
			RequestID:    baseScope.RequestID,
			CredentialID: strings.TrimSpace(auth.ID),
			Model:        baseScope.Model,
			Kind:         baseScope.Kind,
			StartedAt:    baseScope.StartedAt,
		})
		if errInstall != nil {
			client.AbortAmbiguousDispatch()
			pending.End()
			return nil, homeConcurrencyInstallError(errInstall)
		}
	}

	selection, errSelection := newHomeDispatchSelection(auth.Clone(), executor, providerKey, scope)
	if errSelection != nil {
		endScope()
		return nil, &Error{Code: "home_unavailable", Message: "home execution registry unavailable", Retryable: true, HTTPStatus: http.StatusServiceUnavailable}
	}
	if envelope.Present {
		selection.accountedModel = envelope.Tuple.Model
	}
	if executionSessionID := homeExecutionSessionIDFromMetadata(opts.Metadata); executionSessionID != "" && cliproxyexecutor.DownstreamWebsocket(ctx) {
		if errEnd := m.endMismatchedHomeSessionSelections(ctx, executionSessionID, strings.TrimSpace(auth.ID), requestedModel, true); errEnd != nil {
			selection.End("target_change_release_failed")
			return nil, errEnd
		}
	}
	return selection, nil
}
