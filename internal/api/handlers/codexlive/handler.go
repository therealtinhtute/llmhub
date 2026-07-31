package codexlive

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/therealtinhtute/llmhub/internal/client/codex/live"
	"github.com/therealtinhtute/llmhub/internal/runtimecontrol"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/therealtinhtute/llmhub/sdk/proxyutil"
	"golang.org/x/net/proxy"
)

type Handler struct {
	authManager *coreauth.Manager
	settings    runtimecontrol.SettingsStore
	sessions    *live.Store

	relayMu       sync.Mutex
	relaySettings runtimecontrol.CodexLiveSettings
	relay         *live.PionMediaRelay
	limiter       *live.MediaSessionLimiter
}

func New(authManager *coreauth.Manager, settings runtimecontrol.SettingsStore) *Handler {
	return &Handler{
		authManager: authManager,
		settings:    settings,
		sessions:    live.NewStore(),
	}
}

func (h *Handler) CreateCall(c *gin.Context) {
	settings, ok := h.liveSettings(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Codex Live is disabled"})
		return
	}
	auth := h.codexAuth()
	if auth == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no Codex credential available"})
		return
	}

	body, err := live.ReadBody(c.Request.Body)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, live.ErrBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	payload, contentType, model, err := live.PrepareCallRequest(body, c.GetHeader("Content-Type"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var mediaSession live.MediaRelaySession
	if clientOffer, errSDP := live.CallRequestSDP(payload, contentType); errSDP == nil {
		relay, errRelay := h.mediaRelay(settings)
		if errRelay != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": errRelay.Error()})
			return
		}
		mediaSession, payload, contentType, err = h.prepareMediaRequest(c.Request.Context(), relay, auth, clientOffer, payload, contentType)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}

	headers := live.ProtocolHeaders(c.Request.Header)
	headers.Set("Content-Type", contentType)
	upstreamReq, err := h.authManager.NewHttpRequest(c.Request.Context(), auth, http.MethodPost, live.UpstreamCallURL, payload, headers)
	if err != nil {
		closeMediaSession(mediaSession)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.authManager.HttpRequest(c.Request.Context(), auth, upstreamReq)
	if err != nil {
		closeMediaSession(mediaSession)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	responseHeaders := live.CallResponseHeaders(resp.Header)
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		closeMediaSession(mediaSession)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && mediaSession != nil {
		responseBody, responseHeaders, err = acceptMediaAnswer(c.Request.Context(), mediaSession, responseBody, resp.Header, responseHeaders)
		if err != nil {
			closeMediaSession(mediaSession)
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}

	callID := live.CallIDFromLocation(responseHeaders.Get("Location"))
	if callID != "" {
		resources := &live.SessionResources{}
		if mediaSession != nil {
			resources.Add(mediaSession.Close)
		}
		h.sessions.Put(callID, live.Session{AuthID: auth.ID, Model: model, Resources: resources})
		if mediaSession != nil {
			mediaSession.SetCallID(callID)
			mediaSession.SetCloseHandler(func(string) {
				h.sessions.CompleteCall(callID)
			})
		}
	} else {
		closeMediaSession(mediaSession)
	}
	live.WriteResponseHeaders(c.Writer.Header(), responseHeaders)
	c.Status(resp.StatusCode)
	_, _ = c.Writer.Write(responseBody)
}

func (h *Handler) Sideband(c *gin.Context) {
	if !h.enabled(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Codex Live is disabled"})
		return
	}
	if !websocket.IsWebSocketUpgrade(c.Request) {
		c.JSON(http.StatusUpgradeRequired, gin.H{"error": "WebSocket upgrade required"})
		return
	}
	style, callID, ok := live.SidebandTarget(c.Request.URL.Path, map[string]string{"call_id": c.Param("call_id")}, c.Request.URL.Query())
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid Codex Live call ID"})
		return
	}
	session, claim := h.sessions.Claim(callID)
	switch claim {
	case live.ClaimAcquired:
	case live.ClaimBusy:
		c.JSON(http.StatusConflict, gin.H{"error": "Codex Live session already joining"})
		return
	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "Codex Live session not found"})
		return
	}
	completeSession := false
	defer func() {
		if completeSession {
			h.sessions.Complete(session)
			return
		}
		h.sessions.Release(session)
	}()

	auth, ok := h.authManager.GetByID(session.AuthID)
	if !ok || auth == nil || auth.Disabled || auth.Unavailable || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Codex auth unavailable"})
		return
	}

	upstreamURL := live.BuildSidebandURL(sidebandBaseURL(), style, callID)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, live.WebsocketHTTPURL(upstreamURL), nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	req.Header = live.ProtocolHeaders(c.Request.Header)
	if err = h.authManager.PrepareHttpRequest(c.Request.Context(), auth, req); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	dialer := sidebandDialer(auth.ProxyURL)
	dialer.Subprotocols = websocket.Subprotocols(c.Request)
	upstream, handshake, err := dialer.DialContext(c.Request.Context(), upstreamURL, req.Header)
	if err != nil {
		status := http.StatusBadGateway
		if handshake != nil && handshake.StatusCode > 0 {
			status = handshake.StatusCode
			if handshake.Body != nil {
				_ = handshake.Body.Close()
			}
		}
		c.JSON(status, gin.H{"error": "Codex Live sideband upstream unavailable"})
		return
	}
	defer upstream.Close()
	if handshake != nil && handshake.Body != nil {
		_ = handshake.Body.Close()
	}

	upgradeHeaders := http.Header{}
	if protocol := upstream.Subprotocol(); protocol != "" {
		upgradeHeaders.Set("Sec-WebSocket-Protocol", protocol)
	}
	downstream, err := sidebandUpgrader.Upgrade(c.Writer, c.Request, upgradeHeaders)
	if err != nil {
		return
	}
	defer downstream.Close()
	if session.Resources != nil {
		session.Resources.Add(websocketCloseFunc(upstream), websocketCloseFunc(downstream))
	}
	completeSession = true
	if err = relayWebsockets(downstream, upstream); err != nil && !normalWebsocketClose(err) {
		return
	}
}

func (h *Handler) Close() {
	if h == nil {
		return
	}
	h.sessions.CloseAll()
}

func (h *Handler) liveSettings(c *gin.Context) (runtimecontrol.CodexLiveSettings, bool) {
	if h == nil || h.settings == nil || h.authManager == nil {
		return runtimecontrol.CodexLiveSettings{}, false
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	settings, err := h.settings.LoadRuntimeSettings(ctx)
	if err != nil {
		return runtimecontrol.CodexLiveSettings{}, false
	}
	settings, err = settings.Normalize()
	if err != nil || !settings.CodexLive.Enabled {
		return runtimecontrol.CodexLiveSettings{}, false
	}
	return settings.CodexLive, true
}

func (h *Handler) enabled(c *gin.Context) bool {
	_, ok := h.liveSettings(c)
	return ok
}

func (h *Handler) mediaRelay(settings runtimecontrol.CodexLiveSettings) (*live.PionMediaRelay, error) {
	h.relayMu.Lock()
	defer h.relayMu.Unlock()
	if h.limiter == nil {
		h.limiter = &live.MediaSessionLimiter{}
	}
	if h.relay != nil && reflect.DeepEqual(h.relaySettings, settings) {
		return h.relay, nil
	}
	relay, err := live.NewPionMediaRelayWithLimiter(settings, h.limiter)
	if err != nil {
		return nil, err
	}
	h.relay = relay
	h.relaySettings = settings
	return relay, nil
}

func (h *Handler) prepareMediaRequest(ctx context.Context, relay *live.PionMediaRelay, auth *coreauth.Auth, clientOffer string, payload []byte, contentType string) (live.MediaRelaySession, []byte, string, error) {
	mediaSession, upstreamOffer, err := relay.NewSession(ctx, clientOffer, live.MediaSessionRoute{
		ProxyURL:   auth.ProxyURL,
		Credential: credentialName(auth),
		AuthIndex:  auth.Index,
	})
	if err != nil {
		return nil, nil, "", err
	}
	rewritten, rewrittenContentType, err := live.ReplaceCallRequestSDP(payload, contentType, upstreamOffer)
	if err != nil {
		closeMediaSession(mediaSession)
		return nil, nil, "", err
	}
	return mediaSession, rewritten, rewrittenContentType, nil
}

func acceptMediaAnswer(ctx context.Context, mediaSession live.MediaRelaySession, responseBody []byte, sourceHeaders, responseHeaders http.Header) ([]byte, http.Header, error) {
	upstreamAnswer, err := live.CallResponseSDP(responseBody, sourceHeaders.Get("Content-Type"))
	if err != nil {
		return nil, nil, err
	}
	downstreamAnswer, err := mediaSession.AcceptUpstreamAnswer(ctx, upstreamAnswer)
	if err != nil {
		return nil, nil, err
	}
	responseHeaders = responseHeaders.Clone()
	responseHeaders.Set("Content-Type", "application/sdp")
	return []byte(downstreamAnswer), responseHeaders, nil
}

func closeMediaSession(session live.MediaRelaySession) {
	if session != nil {
		_ = session.Close()
	}
}

func credentialName(auth *coreauth.Auth) string {
	if auth == nil {
		return ""
	}
	if label := strings.TrimSpace(auth.Label); label != "" {
		return label
	}
	if fileName := strings.TrimSpace(auth.FileName); fileName != "" {
		return fileName
	}
	if id := strings.TrimSpace(auth.ID); id != "" {
		return id
	}
	return strings.TrimSpace(auth.Index)
}

func (h *Handler) codexAuth() *coreauth.Auth {
	if h == nil || h.authManager == nil {
		return nil
	}
	for _, auth := range h.authManager.List() {
		if auth == nil || auth.Disabled || auth.Unavailable || !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
			continue
		}
		return auth
	}
	return nil
}

var sidebandUpgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
var sidebandBaseURL = func() string { return live.UpstreamSidebandBaseURL }

func sidebandDialer(proxyURL string) *websocket.Dialer {
	dialer := &websocket.Dialer{Proxy: http.ProxyFromEnvironment}
	if strings.TrimSpace(proxyURL) == "" {
		return dialer
	}
	proxyDialer, mode, err := proxyutil.BuildDialer(proxyURL)
	if err != nil || proxyDialer == nil {
		return dialer
	}
	switch mode {
	case proxyutil.ModeDirect:
		dialer.Proxy = nil
	case proxyutil.ModeProxy:
		dialer.Proxy = nil
		dialer.NetDialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			if contextDialer, ok := proxyDialer.(proxy.ContextDialer); ok {
				return contextDialer.DialContext(ctx, network, address)
			}
			result := make(chan struct {
				conn net.Conn
				err  error
			}, 1)
			go func() {
				conn, errDial := proxyDialer.Dial(network, address)
				result <- struct {
					conn net.Conn
					err  error
				}{conn: conn, err: errDial}
			}()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case got := <-result:
				return got.conn, got.err
			}
		}
	}
	return dialer
}

func relayWebsockets(downstream, upstream *websocket.Conn) error {
	results := make(chan error, 2)
	go func() { results <- copyWebsocket(upstream, downstream) }()
	go func() { results <- copyWebsocket(downstream, upstream) }()

	firstErr := <-results
	closeCode, closeReason := websocketCloseDetails(firstErr)
	payload := websocket.FormatCloseMessage(closeCode, closeReason)
	_ = downstream.WriteControl(websocket.CloseMessage, payload, time.Now().Add(time.Second))
	_ = upstream.WriteControl(websocket.CloseMessage, payload, time.Now().Add(time.Second))
	_ = downstream.Close()
	_ = upstream.Close()
	<-results
	return firstErr
}

func copyWebsocket(destination, source *websocket.Conn) error {
	for {
		messageType, reader, err := source.NextReader()
		if err != nil {
			return err
		}
		writer, err := destination.NextWriter(messageType)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, reader)
		closeErr := writer.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}

func websocketCloseDetails(err error) (int, string) {
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure, websocket.CloseTLSHandshake:
			return websocket.CloseNormalClosure, ""
		default:
			return closeErr.Code, closeErr.Text
		}
	}
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return websocket.CloseNormalClosure, ""
	}
	return websocket.CloseInternalServerErr, "relay closed"
}

func normalWebsocketClose(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived)
}

func websocketCloseFunc(conn *websocket.Conn) func() error {
	var once sync.Once
	var err error
	return func() error {
		once.Do(func() { err = conn.Close() })
		return err
	}
}
