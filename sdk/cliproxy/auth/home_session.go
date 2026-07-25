package auth

import (
	"context"
	"strings"
	"sync"

	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type homeSessionSelectionKey struct {
	credentialID string
	routeModel   string
}

func (m *Manager) lockHomeWebsocketSession(ctx context.Context, opts cliproxyexecutor.Options) func() {
	if m == nil || !cliproxyexecutor.DownstreamWebsocket(ctx) {
		return nil
	}
	sessionID := homeExecutionSessionIDFromMetadata(opts.Metadata)
	if sessionID == "" {
		return nil
	}
	lockValue, _ := m.homeSessionLocks.LoadOrStore(sessionID, &sync.Mutex{})
	lock, ok := lockValue.(*sync.Mutex)
	if !ok || lock == nil {
		return nil
	}
	lock.Lock()
	return lock.Unlock
}

func (m *Manager) retainedHomeSessionSelection(ctx context.Context, opts cliproxyexecutor.Options, model string) (*HomeDispatchSelection, bool, error) {
	if m == nil || !cliproxyexecutor.DownstreamWebsocket(ctx) {
		return nil, false, nil
	}
	sessionID := homeExecutionSessionIDFromMetadata(opts.Metadata)
	if sessionID == "" {
		return nil, false, nil
	}
	credentialID := pinnedAuthIDFromMetadata(opts.Metadata)
	routeModel, validRouteModel := validCanonicalHomeConcurrencyModelKey(model)
	fallbackAttempt := homeAuthCountFromMetadata(opts.Metadata) > 1

	var retained *HomeDispatchSelection
	var ended []*HomeDispatchSelection
	m.mu.Lock()
	selections := m.homeSessionSelections[sessionID]
	for key, selection := range selections {
		if selection == nil {
			delete(selections, key)
			continue
		}
		matchesCredential := credentialID == "" || key.credentialID == credentialID
		matchesRoute := validRouteModel && key.routeModel == routeModel
		if !fallbackAttempt && matchesCredential && matchesRoute && selection.Active() && retained == nil {
			retained = selection
			continue
		}
		delete(selections, key)
		ended = append(ended, selection)
	}
	if len(selections) == 0 {
		delete(m.homeSessionSelections, sessionID)
	}
	m.mu.Unlock()

	for _, selection := range ended {
		if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "target_changed"); errEnd != nil {
			return nil, false, errEnd
		}
	}
	return retained, retained != nil, nil
}

func (m *Manager) endMismatchedHomeSessionSelections(ctx context.Context, sessionID, credentialID, model string, waitForAck bool) error {
	if m == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	routeModel, validRouteModel := validCanonicalHomeConcurrencyModelKey(model)
	var ended []*HomeDispatchSelection
	m.mu.Lock()
	selections := m.homeSessionSelections[sessionID]
	for key, selection := range selections {
		if selection == nil {
			delete(selections, key)
			continue
		}
		if key.credentialID == credentialID && validRouteModel && key.routeModel == routeModel {
			continue
		}
		delete(selections, key)
		ended = append(ended, selection)
	}
	if len(selections) == 0 {
		delete(m.homeSessionSelections, sessionID)
	}
	m.mu.Unlock()

	for _, selection := range ended {
		if !waitForAck {
			selection.End("target_changed")
			continue
		}
		if errEnd := m.endHomeSelectionBeforeRedispatch(ctx, selection, "target_changed"); errEnd != nil {
			return errEnd
		}
	}
	return nil
}

func (m *Manager) retainHomeWebsocketSelection(ctx context.Context, opts cliproxyexecutor.Options, model string, selection *HomeDispatchSelection) bool {
	if m == nil || selection == nil || selection.Auth == nil || !selection.Retained() || !cliproxyexecutor.DownstreamWebsocket(ctx) {
		return false
	}
	sessionID := homeExecutionSessionIDFromMetadata(opts.Metadata)
	credentialID := strings.TrimSpace(selection.Auth.ID)
	routeModel, validRouteModel := validCanonicalHomeConcurrencyModelKey(model)
	if selection.accountedModel == "" {
		selection.accountedModel, _ = m.predictedHomeConcurrencyModel(selection.Auth, model)
	}
	if sessionID == "" || credentialID == "" || !validRouteModel || selection.accountedModel == "" {
		return false
	}
	_ = m.endMismatchedHomeSessionSelections(ctx, sessionID, credentialID, routeModel, false)
	key := homeSessionSelectionKey{credentialID: credentialID, routeModel: routeModel}

	m.mu.Lock()
	if m.homeSessionSelections == nil {
		m.homeSessionSelections = make(map[string]map[homeSessionSelectionKey]*HomeDispatchSelection)
	}
	selections := m.homeSessionSelections[sessionID]
	if selections == nil {
		selections = make(map[homeSessionSelectionKey]*HomeDispatchSelection)
		m.homeSessionSelections[sessionID] = selections
	}
	previous := selections[key]
	selections[key] = selection
	m.mu.Unlock()
	m.rememberHomeSelectionRuntimeAuth(sessionID, selection)
	if previous != nil && previous != selection {
		previous.End("target_replaced")
	}
	return true
}

func (m *Manager) predictedHomeConcurrencyModel(auth *Auth, routeModel string) (string, bool) {
	models, _ := m.preparedExecutionModels(auth, routeModel)
	if len(models) != 1 {
		return "", false
	}
	return validCanonicalHomeConcurrencyModelKey(models[0])
}

func (m *Manager) bindHomeSelectionRuntimeAuth(ctx context.Context, opts cliproxyexecutor.Options, selection *HomeDispatchSelection) error {
	if m == nil || selection == nil || selection.Auth == nil || !cliproxyexecutor.DownstreamWebsocket(ctx) || !authWebsocketsEnabled(selection.Auth) {
		return nil
	}
	sessionID := homeExecutionSessionIDFromMetadata(opts.Metadata)
	authID := strings.TrimSpace(selection.Auth.ID)
	if sessionID == "" || authID == "" || !selection.runtimeAuthBound.CompareAndSwap(false, true) {
		return nil
	}
	m.rememberHomeSelectionRuntimeAuth(sessionID, selection)
	if errBind := selection.Bind(func() error {
		m.forgetHomeRuntimeAuth(sessionID, authID, selection)
		return nil
	}); errBind != nil {
		selection.runtimeAuthBound.Store(false)
		m.forgetHomeRuntimeAuth(sessionID, authID, selection)
		return errBind
	}
	return nil
}

func (m *Manager) rememberHomeSelectionRuntimeAuth(sessionID string, selection *HomeDispatchSelection) {
	if m == nil || selection == nil || selection.Auth == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	authID := strings.TrimSpace(selection.Auth.ID)
	if sessionID == "" || authID == "" {
		return
	}
	m.mu.Lock()
	if m.homeRuntimeAuths == nil {
		m.homeRuntimeAuths = make(map[string]map[string]*Auth)
	}
	if m.homeRuntimeAuthOwners == nil {
		m.homeRuntimeAuthOwners = make(map[string]map[string]*HomeDispatchSelection)
	}
	if m.homeRuntimeAuths[sessionID] == nil {
		m.homeRuntimeAuths[sessionID] = make(map[string]*Auth)
	}
	if m.homeRuntimeAuthOwners[sessionID] == nil {
		m.homeRuntimeAuthOwners[sessionID] = make(map[string]*HomeDispatchSelection)
	}
	m.homeRuntimeAuths[sessionID][authID] = selection.Auth.Clone()
	m.homeRuntimeAuthOwners[sessionID][authID] = selection
	m.mu.Unlock()
}

func (m *Manager) forgetHomeRuntimeAuth(sessionID, authID string, owner *HomeDispatchSelection) {
	if m == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	authID = strings.TrimSpace(authID)
	if sessionID == "" || authID == "" {
		return
	}
	m.mu.Lock()
	if owner != nil && m.homeRuntimeAuthOwners[sessionID][authID] != owner {
		m.mu.Unlock()
		return
	}
	delete(m.homeRuntimeAuths[sessionID], authID)
	delete(m.homeRuntimeAuthOwners[sessionID], authID)
	if len(m.homeRuntimeAuths[sessionID]) == 0 {
		delete(m.homeRuntimeAuths, sessionID)
	}
	if len(m.homeRuntimeAuthOwners[sessionID]) == 0 {
		delete(m.homeRuntimeAuthOwners, sessionID)
	}
	m.mu.Unlock()
}

func (m *Manager) clearHomeSessionLocks() {
	if m == nil {
		return
	}
	m.homeSessionLocks.Range(func(key, _ any) bool {
		m.homeSessionLocks.Delete(key)
		return true
	})
}

func (m *Manager) takeHomeSessionSelectionsLocked(sessionID string) []*HomeDispatchSelection {
	if m == nil {
		return nil
	}
	selections := m.homeSessionSelections[sessionID]
	delete(m.homeSessionSelections, sessionID)
	result := make([]*HomeDispatchSelection, 0, len(selections))
	for _, selection := range selections {
		result = append(result, selection)
	}
	return result
}

func (m *Manager) takeAllHomeSessionSelectionsLocked() []*HomeDispatchSelection {
	if m == nil {
		return nil
	}
	result := make([]*HomeDispatchSelection, 0)
	for sessionID, selections := range m.homeSessionSelections {
		delete(m.homeSessionSelections, sessionID)
		for _, selection := range selections {
			result = append(result, selection)
		}
	}
	return result
}
