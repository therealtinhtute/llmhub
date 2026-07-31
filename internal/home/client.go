package home

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/config"
)

const (
	redisKeyConfig             = "config"
	redisChannelConfig         = "config"
	redisKeyModels             = "models"
	redisKeyUsage              = "usage"
	redisKeyInFlightSnapshot   = "in-flight-snapshot"
	redisKeyConcurrencyRelease = "concurrency-release"
	redisKeyRequestLog         = "request-log"

	homeReconnectInterval          = time.Second
	homeReconnectFailoverThreshold = 3
	homeRedisOperationTimeout      = 3 * time.Second
	homeSubscriptionReceiveTimeout = 3 * time.Second
	redisChannelCluster            = "cluster"
)

// DispatchError classifies whether Home may have processed an auth dispatch request.
type DispatchError struct {
	Err       error
	Ambiguous bool
}

func (e *DispatchError) Error() string {
	if e == nil || e.Err == nil {
		return "home auth dispatch failed"
	}
	return e.Err.Error()
}

func (e *DispatchError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewAmbiguousDispatchError(err error) error {
	if err == nil {
		return nil
	}
	return &DispatchError{Err: err, Ambiguous: true}
}

func IsAmbiguousDispatchError(err error) bool {
	var dispatchErr *DispatchError
	return errors.As(err, &dispatchErr) && dispatchErr.Ambiguous
}

var (
	ErrDisabled                  = errors.New("home client disabled")
	ErrNotConnected              = errors.New("home not connected")
	ErrEmptyResponse             = errors.New("home returned empty response")
	ErrAuthNotFound              = errors.New("home auth not found")
	ErrConfigNotFound            = errors.New("home config not found")
	ErrModelsNotFound            = errors.New("home models not found")
	ErrDispatchFenced            = errors.New("home auth dispatch is fenced")
	ErrCompareAndSwapUnsupported = errors.New("home compare-and-swap is unsupported")
	errClusterDiscoveryTransport = errors.New("home cluster discovery transport failed")
)

func isHomeCommandUnsupported(err error) bool {
	for err != nil {
		message := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(message, "unknown command") || strings.Contains(message, "unsupported command") {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// IsMembershipTakeoverUnavailableError reports whether Home cannot preserve the previous membership state.
func IsMembershipTakeoverUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.TrimSpace(strings.ToLower(err.Error()))
	return message == "membership_takeover_unavailable" || message == "err membership_takeover_unavailable"
}

// IsLegacyMembershipProtocolError reports whether Home rejected protocol-one subscription arguments.
func IsLegacyMembershipProtocolError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.TrimSpace(strings.ToLower(err.Error()))
	return message == "wrong number of arguments for 'subscribe' command" || message == "err wrong number of arguments for 'subscribe' command"
}

type recoveryState uint32

const (
	recoveryStateStable recoveryState = iota
	recoveryStateTakeoverEligible
	recoveryStateSwitching
	recoveryStateSwitchingTakeover
)

type clusterNode struct {
	IP          string    `json:"ip"`
	Port        int       `json:"port"`
	ClientCount int       `json:"client_count"`
	IsMaster    bool      `json:"is_master"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type clusterNodesEnvelope struct {
	OK    bool          `json:"ok"`
	Nodes []clusterNode `json:"nodes"`
}

type KVSetOptions struct {
	EX time.Duration
	PX time.Duration
	NX bool
	XX bool
}

type Client struct {
	mu sync.Mutex

	homeCfg  config.HomeConfig
	seedHost string
	seedPort int

	cmd         *redis.Client
	sub         *redis.Client
	release     *redis.Client
	connections map[*homeDispatchConn]struct{}
	closing     chan struct{}
	limiter     atomic.Pointer[config.CredentialConcurrencyConfig]

	heartbeatOK       atomic.Bool
	dispatchFenced    atomic.Bool
	ambiguousDispatch atomic.Bool
	recoveryState     atomic.Uint32
	casUnsupported    atomic.Bool
	instanceID        string
	legacyMembership  bool
	clusterNodes      []clusterNode
	reconnectFailures int
}

func New(homeCfg config.HomeConfig) *Client {
	return &Client{
		homeCfg:    homeCfg,
		seedHost:   strings.TrimSpace(homeCfg.Host),
		seedPort:   homeCfg.Port,
		instanceID: uuid.NewString(),
	}
}

// NewLifetime creates a fresh client while preserving cluster failover and membership recovery state.
func (c *Client) NewLifetime() *Client {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	next := &Client{
		homeCfg:           c.homeCfg,
		seedHost:          c.seedHost,
		seedPort:          c.seedPort,
		clusterNodes:      append([]clusterNode(nil), c.clusterNodes...),
		reconnectFailures: c.reconnectFailures,
		instanceID:        c.instanceID,
		legacyMembership:  c.legacyMembership,
	}
	state := c.recoveryState.Load()
	if c.ambiguousDispatch.Load() && state == uint32(recoveryStateSwitchingTakeover) {
		state = uint32(recoveryStateSwitching)
	}
	next.recoveryState.Store(state)
	return next
}

// MembershipInstanceID returns the process-scoped Home membership identity.
func (c *Client) MembershipInstanceID() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.instanceID
}

// LegacyMembership reports whether this subscriber has downgraded to the legacy protocol.
func (c *Client) LegacyMembership() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.legacyMembership
}

// EnableLegacyMembership permanently downgrades this subscriber lifetime chain.
func (c *Client) EnableLegacyMembership() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.legacyMembership = true
	c.mu.Unlock()
	c.SuppressTakeover()
}

func (c *Client) Enabled() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.homeCfg.Enabled
}

func (c *Client) HeartbeatOK() bool {
	if c == nil {
		return false
	}
	if !c.Enabled() {
		return false
	}
	return c.heartbeatOK.Load()
}

func (c *Client) Close() {
	if c == nil {
		return
	}
	c.heartbeatOK.Store(false)
	c.dispatchFenced.Store(true)
	c.mu.Lock()
	cmd, sub, connections := c.detachClientsLocked()
	release := c.release
	c.release = nil
	closing := c.closing
	c.mu.Unlock()
	closeDetachedClients(cmd, sub, connections)
	if release != nil {
		_ = release.Close()
	}
	if closing != nil {
		<-closing
	}
}

func (c *Client) detachClientsLocked() (*redis.Client, *redis.Client, []*homeDispatchConn) {
	connections := make([]*homeDispatchConn, 0, len(c.connections))
	for conn := range c.connections {
		connections = append(connections, conn)
	}
	cmd := c.cmd
	sub := c.sub
	c.cmd = nil
	c.sub = nil
	c.connections = nil
	return cmd, sub, connections
}

func closeDetachedClients(cmd *redis.Client, sub *redis.Client, connections []*homeDispatchConn) {
	for _, conn := range connections {
		_ = conn.Close()
	}
	if cmd != nil {
		_ = cmd.Close()
	}
	if sub != nil {
		_ = sub.Close()
	}
}

func (c *Client) closeClientsLocked() {
	cmd, sub, connections := c.detachClientsLocked()
	release := c.release
	c.release = nil
	previousClosing := c.closing
	done := make(chan struct{})
	c.closing = done
	go func() {
		defer close(done)
		if previousClosing != nil {
			<-previousClosing
		}
		closeDetachedClients(cmd, sub, connections)
		if release != nil {
			_ = release.Close()
		}
	}()
}

func (c *Client) waitForClientsClosed() {
	for {
		c.mu.Lock()
		closing := c.closing
		c.mu.Unlock()
		if closing == nil {
			return
		}
		<-closing
		c.mu.Lock()
		if c.closing == closing {
			c.closing = nil
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
	}
}

// AbortAmbiguousDispatch fences dispatch after a response whose accounting state is unknown.
func (c *Client) AbortAmbiguousDispatch() {
	if c == nil {
		return
	}
	c.ambiguousDispatch.Store(true)
	c.SuppressTakeover()
	c.dispatchFenced.Store(true)
	c.heartbeatOK.Store(false)
	c.mu.Lock()
	cmd, sub, connections := c.detachClientsLocked()
	release := c.release
	c.release = nil
	c.mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
	if cmd != nil {
		go func() { _ = cmd.Close() }()
	}
	if sub != nil {
		go func() { _ = sub.Close() }()
	}
	if release != nil {
		go func() { _ = release.Close() }()
	}
}

// AmbiguousDispatch reports whether this lifetime observed an issued dispatch with an unknown delivery result.
func (c *Client) AmbiguousDispatch() bool {
	return c != nil && c.ambiguousDispatch.Load()
}

// SuppressTakeover forces the next subscriber lifetime through normal membership recovery.
func (c *Client) SuppressTakeover() {
	if c == nil {
		return
	}
	if !c.recoveryState.CompareAndSwap(uint32(recoveryStateTakeoverEligible), uint32(recoveryStateStable)) {
		c.recoveryState.CompareAndSwap(uint32(recoveryStateSwitchingTakeover), uint32(recoveryStateSwitching))
	}
}

func (c *Client) addr() (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.addrLocked()
}

func (c *Client) addrLocked() (string, bool) {
	host := strings.TrimSpace(c.homeCfg.Host)
	if host == "" {
		return "", false
	}
	if c.homeCfg.Port <= 0 {
		return "", false
	}
	return net.JoinHostPort(host, strconv.Itoa(c.homeCfg.Port)), true
}

func (c *Client) ensureClients() error {
	if c == nil {
		return ErrDisabled
	}
	if c.dispatchFenced.Load() {
		return ErrDispatchFenced
	}
	if !c.Enabled() {
		return ErrDisabled
	}
	c.waitForClientsClosed()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dispatchFenced.Load() {
		return ErrDispatchFenced
	}

	addr, ok := c.addrLocked()
	if !ok {
		return fmt.Errorf("home: invalid address (host=%q port=%d)", c.homeCfg.Host, c.homeCfg.Port)
	}

	if c.cmd == nil {
		options, errOptions := c.redisOptionsLocked(addr)
		if errOptions != nil {
			return errOptions
		}
		c.cmd = redis.NewClient(options)
	}
	if c.sub == nil {
		options, errOptions := c.redisOptionsLocked(addr)
		if errOptions != nil {
			return errOptions
		}
		c.sub = redis.NewClient(options)
	}
	return nil
}

func (c *Client) redisOptionsLocked(addr string) (*redis.Options, error) {
	tlsConfig, errTLS := c.homeTLSConfigLocked(addr)
	if errTLS != nil {
		return nil, errTLS
	}
	options := &redis.Options{
		Addr:                  addr,
		TLSConfig:             tlsConfig,
		DialTimeout:           homeRedisOperationTimeout,
		ReadTimeout:           homeRedisOperationTimeout,
		WriteTimeout:          homeRedisOperationTimeout,
		MaxRetries:            -1,
		DialerRetries:         1,
		ContextTimeoutEnabled: true,
	}
	options.Dialer = c.trackedRedisDialer(redis.NewDialer(options))
	return options, nil
}

type homeDispatchConn struct {
	net.Conn
	client *Client
	once   sync.Once
}

func (c *Client) trackedRedisDialer(dialer func(context.Context, string, string) (net.Conn, error)) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		conn, errDial := dialer(ctx, network, address)
		if errDial != nil {
			return nil, errDial
		}
		wrapped := &homeDispatchConn{Conn: conn, client: c}
		if c == nil {
			return wrapped, nil
		}
		c.mu.Lock()
		if c.dispatchFenced.Load() {
			c.mu.Unlock()
			_ = wrapped.Close()
			return nil, ErrDispatchFenced
		}
		if c.connections == nil {
			c.connections = make(map[*homeDispatchConn]struct{})
		}
		c.connections[wrapped] = struct{}{}
		c.mu.Unlock()
		return wrapped, nil
	}
}

func (c *homeDispatchConn) Close() error {
	if c == nil {
		return nil
	}
	var err error
	c.once.Do(func() {
		if c.client != nil {
			c.client.mu.Lock()
			delete(c.client.connections, c)
			c.client.mu.Unlock()
		}
		err = c.Conn.Close()
	})
	return err
}

func (c *Client) homeTLSConfigLocked(addr string) (*tls.Config, error) {
	serverName := strings.TrimSpace(c.homeCfg.TLS.ServerName)
	if serverName == "" {
		if c.homeCfg.TLS.UseTargetServerName {
			serverName = hostFromAddress(addr)
		} else {
			serverName = strings.TrimSpace(c.seedHost)
		}
	}
	if serverName == "" {
		serverName = strings.TrimSpace(c.homeCfg.Host)
	}
	return newHomeTLSConfig(c.homeCfg.TLS, serverName)
}

func hostFromAddress(addr string) string {
	host, _, errSplit := net.SplitHostPort(strings.TrimSpace(addr))
	if errSplit == nil {
		return strings.TrimSpace(host)
	}
	return strings.TrimSpace(addr)
}

func newHomeTLSConfig(cfg config.HomeTLSConfig, fallbackServerName string) (*tls.Config, error) {
	if !cfg.Enable {
		return nil, nil
	}

	serverName := strings.TrimSpace(cfg.ServerName)
	if serverName == "" {
		serverName = strings.TrimSpace(fallbackServerName)
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         serverName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	clientCertPath := strings.TrimSpace(cfg.ClientCert)
	clientKeyPath := strings.TrimSpace(cfg.ClientKey)
	if clientCertPath != "" || clientKeyPath != "" {
		if clientCertPath == "" || clientKeyPath == "" {
			return nil, fmt.Errorf("home tls: client certificate and key must be set together")
		}
		certPair, errLoad := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if errLoad != nil {
			return nil, fmt.Errorf("home tls: load client certificate: %w", errLoad)
		}
		tlsConfig.Certificates = []tls.Certificate{certPair}
	}

	caCertPath := strings.TrimSpace(cfg.CACert)
	if caCertPath == "" {
		return tlsConfig, nil
	}

	caCertPEM, errRead := os.ReadFile(caCertPath)
	if errRead != nil {
		return nil, fmt.Errorf("home tls: read ca-cert: %w", errRead)
	}

	certPool, errPool := x509.SystemCertPool()
	if errPool != nil || certPool == nil {
		certPool = x509.NewCertPool()
	}
	if !certPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("home tls: ca-cert contains no PEM certificates")
	}
	tlsConfig.RootCAs = certPool

	return tlsConfig, nil
}

func (c *Client) commandClient() (*redis.Client, error) {
	if c == nil || c.dispatchFenced.Load() {
		return nil, ErrDispatchFenced
	}
	if errEnsure := c.ensureClients(); errEnsure != nil {
		return nil, errEnsure
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dispatchFenced.Load() {
		return nil, ErrDispatchFenced
	}
	if c.cmd == nil {
		return nil, ErrNotConnected
	}
	return c.cmd, nil
}

func (c *Client) subscriptionClient() (*redis.Client, error) {
	if c == nil || c.dispatchFenced.Load() {
		return nil, ErrDispatchFenced
	}
	if errEnsure := c.ensureClients(); errEnsure != nil {
		return nil, errEnsure
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dispatchFenced.Load() {
		return nil, ErrDispatchFenced
	}
	if c.sub == nil {
		return nil, ErrNotConnected
	}
	return c.sub, nil
}

func (c *Client) Ping(ctx context.Context) error {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return errClient
	}
	return cmd.Ping(ctx).Err()
}

func (c *Client) clusterDiscoveryEnabled() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clusterDiscoveryEnabledLocked()
}

func (c *Client) clusterDiscoveryEnabledLocked() bool {
	return !c.homeCfg.DisableClusterDiscovery
}

func (c *Client) refreshBestClusterNode(ctx context.Context) error {
	if !c.clusterDiscoveryEnabled() {
		return nil
	}
	switched, errRefresh := c.refreshClusterNodes(ctx)
	if errRefresh != nil {
		log.Debugf("home cluster nodes unavailable: %v", errRefresh)
		return errRefresh
	}
	if switched {
		if addr, ok := c.addr(); ok {
			log.Infof("home cluster target switched to %s", addr)
		}
	}
	return nil
}

func (c *Client) refreshClusterNodes(ctx context.Context) (bool, error) {
	if !c.clusterDiscoveryEnabled() {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return false, fmt.Errorf("%w: %w", errClusterDiscoveryTransport, errClient)
	}
	nodesCommand := cmd.Do(ctx, "CLUSTER", "NODES")
	errDo := nodesCommand.Err()
	if errDo != nil {
		var redisErr redis.Error
		if !errors.As(errDo, &redisErr) {
			return false, fmt.Errorf("%w: %w", errClusterDiscoveryTransport, errDo)
		}
		return false, errDo
	}
	raw, errText := nodesCommand.Text()
	if errText != nil {
		return false, errText
	}

	nodes, errParse := parseClusterNodesPayload([]byte(raw))
	if errParse != nil {
		return false, errParse
	}
	if len(nodes) == 0 {
		return false, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.clusterNodes = nodes
	c.reconnectFailures = 0
	return c.switchToNodeLocked(nodes[0]), nil
}

func parseClusterNodesPayload(raw []byte) ([]clusterNode, error) {
	var envelope clusterNodesEnvelope
	if errUnmarshal := json.Unmarshal(raw, &envelope); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	return normalizeClusterNodes(envelope.Nodes), nil
}

func (c *Client) updateClusterNodesFromPayload(raw []byte) error {
	if c == nil || !c.clusterDiscoveryEnabled() {
		return nil
	}
	nodes, errParse := parseClusterNodesPayload(raw)
	if errParse != nil {
		return errParse
	}
	c.mu.Lock()
	c.clusterNodes = nodes
	c.mu.Unlock()
	return nil
}

func normalizeClusterNodes(nodes []clusterNode) []clusterNode {
	out := make([]clusterNode, 0, len(nodes))
	for _, node := range nodes {
		node.IP = strings.TrimSpace(node.IP)
		if node.IP == "" || node.Port <= 0 {
			continue
		}
		if node.ClientCount < 0 {
			node.ClientCount = 0
		}
		out = append(out, node)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ClientCount < out[j].ClientCount
	})
	return out
}

func (c *Client) switchToNodeLocked(node clusterNode) bool {
	host := strings.TrimSpace(node.IP)
	if host == "" || node.Port <= 0 {
		return false
	}
	if strings.TrimSpace(c.homeCfg.Host) == host && c.homeCfg.Port == node.Port {
		return false
	}
	c.homeCfg.Host = host
	c.homeCfg.Port = node.Port
	if !c.recoveryState.CompareAndSwap(uint32(recoveryStateStable), uint32(recoveryStateSwitching)) {
		c.recoveryState.CompareAndSwap(uint32(recoveryStateTakeoverEligible), uint32(recoveryStateSwitchingTakeover))
	}
	c.closeClientsLocked()
	return true
}

func (c *Client) markReconnectFailure(reason string) {
	switched, addr := c.failoverAfterReconnectFailure()
	if switched {
		log.Warnf("home control center unavailable after repeated %s failures; switching to %s", reason, addr)
	}
}

func (c *Client) failoverAfterReconnectFailure() (bool, string) {
	if c == nil {
		return false, ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.clusterDiscoveryEnabledLocked() {
		c.reconnectFailures = 0
		return false, ""
	}
	c.reconnectFailures++
	if c.reconnectFailures < homeReconnectFailoverThreshold {
		return false, ""
	}
	c.reconnectFailures = 0

	return c.switchToNextNodeLocked()
}

func (c *Client) failoverAfterSubscriptionTimeout() (bool, string) {
	if c == nil {
		return false, ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.clusterDiscoveryEnabledLocked() {
		c.reconnectFailures = 0
		return false, ""
	}
	c.reconnectFailures = 0
	return c.switchToNextNodeLocked()
}

func (c *Client) switchToNextNodeLocked() (bool, string) {
	currentHost := strings.TrimSpace(c.homeCfg.Host)
	currentPort := c.homeCfg.Port
	candidates := append([]clusterNode(nil), c.clusterNodes...)
	if strings.TrimSpace(c.seedHost) != "" && c.seedPort > 0 {
		candidates = append(candidates, clusterNode{IP: c.seedHost, Port: c.seedPort})
	}
	for _, node := range candidates {
		host := strings.TrimSpace(node.IP)
		if host == "" || node.Port <= 0 {
			continue
		}
		if host == currentHost && node.Port == currentPort {
			continue
		}
		if c.switchToNodeLocked(clusterNode{IP: host, Port: node.Port}) {
			addr, _ := c.addrLocked()
			return true, addr
		}
	}
	return false, ""
}

func (c *Client) markSubscriptionTimeout() {
	switched, addr := c.failoverAfterSubscriptionTimeout()
	if switched {
		log.Warnf("home subscription heartbeat timeout; switching to %s", addr)
	}
}

func (c *Client) resetReconnectFailures() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.reconnectFailures = 0
	c.mu.Unlock()
}

func (c *Client) GetConfig(ctx context.Context) ([]byte, error) {
	if errRefresh := c.refreshBestClusterNode(ctx); errors.Is(errRefresh, errClusterDiscoveryTransport) {
		return nil, errRefresh
	}
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return nil, errClient
	}
	raw, err := cmd.Get(ctx, redisKeyConfig).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrEmptyResponse
	}
	return raw, nil
}

func (c *Client) GetModels(ctx context.Context) ([]byte, error) {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return nil, errClient
	}
	raw, err := cmd.Get(ctx, redisKeyModels).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrModelsNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrEmptyResponse
	}
	return raw, nil
}

func headersToLowerMap(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		k := strings.ToLower(strings.TrimSpace(key))
		if k == "" {
			continue
		}
		if len(values) == 0 {
			out[k] = ""
			continue
		}
		trimmed := make([]string, 0, len(values))
		for _, v := range values {
			trimmed = append(trimmed, strings.TrimSpace(v))
		}
		out[k] = strings.Join(trimmed, ", ")
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func newAuthDispatchRequest(requestedModel string, sessionID string, headers http.Header, count int) authDispatchRequest {
	if count <= 0 {
		count = 1
	}
	return authDispatchRequest{
		Type:                "auth",
		Model:               requestedModel,
		Count:               count,
		ConcurrencyProtocol: 1,
		SessionID:           strings.TrimSpace(sessionID),
		Headers:             headersToLowerMap(headers),
	}
}

func (c *Client) RPopAuth(ctx context.Context, requestedModel string, sessionID string, headers http.Header, count int) ([]byte, error) {
	if c == nil || c.dispatchFenced.Load() {
		return nil, ErrDispatchFenced
	}
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return nil, errClient
	}
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return nil, fmt.Errorf("home: requested model is empty")
	}
	req := newAuthDispatchRequest(requestedModel, sessionID, headers, count)
	keyBytes, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	raw, err := cmd.RPop(ctx, string(keyBytes)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrAuthNotFound
	}
	if err != nil {
		var redisErr redis.Error
		if !errors.As(err, &redisErr) {
			return nil, NewAmbiguousDispatchError(err)
		}
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrEmptyResponse
	}
	return raw, nil
}

func (c *Client) GetRefreshAuth(ctx context.Context, authIndex string) ([]byte, error) {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return nil, errClient
	}
	authIndex = strings.TrimSpace(authIndex)
	if authIndex == "" {
		return nil, fmt.Errorf("home: auth_index is empty")
	}
	req := refreshRequest{
		Type:      "refresh",
		AuthIndex: authIndex,
	}
	keyBytes, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	raw, err := cmd.Get(ctx, string(keyBytes)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrAuthNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrEmptyResponse
	}
	return raw, nil
}

func (c *Client) LPushUsage(ctx context.Context, payload []byte) error {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return errClient
	}
	if len(payload) == 0 {
		return nil
	}
	return cmd.LPush(ctx, redisKeyUsage, payload).Err()
}

// LPushInFlightSnapshot publishes one bounded in-flight observation frame.
func (c *Client) LPushInFlightSnapshot(ctx context.Context, payload []byte) error {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return errClient
	}
	if len(payload) == 0 {
		return nil
	}
	return cmd.LPush(ctx, redisKeyInFlightSnapshot, payload).Err()
}

// PushConcurrencyRelease sends one cumulative concurrency release frame.
func buildKVSetArgs(key string, value []byte, opts KVSetOptions) ([]any, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("home kv: key is empty")
	}
	if opts.EX > 0 && opts.PX > 0 {
		return nil, fmt.Errorf("home kv: EX and PX are mutually exclusive")
	}
	if opts.EX < 0 || opts.PX < 0 {
		return nil, fmt.Errorf("home kv: ttl must not be negative")
	}
	if opts.NX && opts.XX {
		return nil, fmt.Errorf("home kv: NX and XX are mutually exclusive")
	}

	args := []any{key, append([]byte(nil), value...)}
	if opts.EX > 0 {
		args = append(args, "EX", durationCeil(opts.EX, time.Second))
	}
	if opts.PX > 0 {
		args = append(args, "PX", durationCeil(opts.PX, time.Millisecond))
	}
	if opts.NX {
		args = append(args, "NX")
	}
	if opts.XX {
		args = append(args, "XX")
	}
	return args, nil
}

func durationCeil(value time.Duration, unit time.Duration) int64 {
	if value <= 0 || unit <= 0 {
		return 0
	}
	return int64((value + unit - 1) / unit)
}

func (c *Client) KVGet(ctx context.Context, key string) ([]byte, bool, error) {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return nil, false, errClient
	}
	raw, errGet := cmd.Get(ctx, key).Bytes()
	if errors.Is(errGet, redis.Nil) {
		return nil, false, nil
	}
	if errGet != nil {
		return nil, false, errGet
	}
	return append([]byte(nil), raw...), true, nil
}

func (c *Client) KVSet(ctx context.Context, key string, value []byte, opts KVSetOptions) (bool, error) {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return false, errClient
	}
	args, errArgs := buildKVSetArgs(key, value, opts)
	if errArgs != nil {
		return false, errArgs
	}
	result, errSet := cmd.Do(ctx, append([]any{"SET"}, args...)...).Result()
	if errors.Is(errSet, redis.Nil) {
		return false, nil
	}
	if errSet != nil {
		return false, errSet
	}
	if result == nil {
		return false, nil
	}
	return true, nil
}

func (c *Client) KVCompareAndSwap(ctx context.Context, key string, expected []byte, expectedExists bool, value []byte, ttl time.Duration) (bool, error) {
	if c == nil {
		return false, ErrNotConnected
	}
	if c.casUnsupported.Load() {
		return false, ErrCompareAndSwapUnsupported
	}
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return false, errClient
	}
	expectedFlag := "0"
	if expectedExists {
		expectedFlag = "1"
	}
	args := make([]any, 0, 7)
	args = append(args, "CAS", key, expectedFlag, expected, value)
	if milliseconds := durationCeil(ttl, time.Millisecond); milliseconds > 0 {
		args = append(args, "PX", milliseconds)
	}
	result, errCAS := cmd.Do(ctx, args...).Int64()
	if errCAS != nil {
		if isHomeCommandUnsupported(errCAS) {
			if c.casUnsupported.CompareAndSwap(false, true) {
				log.Warnf("home kv: this Home does not implement the CAS command; reasoning replay is disabled until Home is upgraded")
			}
			return false, ErrCompareAndSwapUnsupported
		}
		return false, errCAS
	}
	return result == 1, nil
}

func (c *Client) KVExpire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return false, errClient
	}
	return cmd.Expire(ctx, key, ttl).Result()
}

func (c *Client) PushConcurrencyRelease(ctx context.Context, frame ConcurrencyReleaseFrame) error {
	if frame.CredentialID == "" || frame.Model == "" || frame.ReleaseSeq <= 0 {
		return fmt.Errorf("invalid concurrency release frame")
	}
	if !c.membershipReadyForRelease() {
		return ErrNotConnected
	}
	cmd, errClient := c.concurrencyReleaseClient()
	if errClient != nil {
		return errClient
	}
	payload, errMarshal := json.Marshal(frame)
	if errMarshal != nil {
		return fmt.Errorf("marshal concurrency release frame: %w", errMarshal)
	}
	return cmd.LPush(ctx, redisKeyConcurrencyRelease, payload).Err()
}

func (c *Client) membershipReadyForRelease() bool {
	if c == nil {
		return false
	}
	state := recoveryState(c.recoveryState.Load())
	return state != recoveryStateTakeoverEligible && state != recoveryStateSwitching && state != recoveryStateSwitchingTakeover
}

func (c *Client) concurrencyReleaseClient() (*redis.Client, error) {
	if c == nil || c.dispatchFenced.Load() {
		return nil, ErrDispatchFenced
	}
	if !c.Enabled() {
		return nil, ErrDisabled
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dispatchFenced.Load() {
		return nil, ErrDispatchFenced
	}
	if c.release != nil {
		return c.release, nil
	}
	addr, ok := c.addrLocked()
	if !ok {
		return nil, fmt.Errorf("home: invalid address (host=%q port=%d)", c.homeCfg.Host, c.homeCfg.Port)
	}
	options, errOptions := c.redisOptionsLocked(addr)
	if errOptions != nil {
		return nil, errOptions
	}
	options.Dialer = redis.NewDialer(options)
	c.release = redis.NewClient(options)
	return c.release, nil
}

func (c *Client) SetLifecycleConfig(cfg config.CredentialConcurrencyConfig) error {
	if c == nil {
		return ErrDisabled
	}
	cfg = cfg.WithDefaults()
	if errValidate := config.ValidateCredentialConcurrency(cfg); errValidate != nil {
		return fmt.Errorf("validate credential concurrency lifecycle config: %w", errValidate)
	}
	c.limiter.Store(&cfg)
	return nil
}

func (c *Client) LimiterConfig() config.CredentialConcurrencyConfig {
	if c == nil {
		return config.CredentialConcurrencyConfig{}.WithDefaults()
	}
	if cfg := c.limiter.Load(); cfg != nil {
		return *cfg
	}
	return config.CredentialConcurrencyConfig{}.WithDefaults()
}

func (c *Client) subscriptionParameters() ([]string, time.Duration) {
	if c == nil {
		return []string{redisChannelConfig}, config.CredentialConcurrencyConfig{}.WithDefaults().CPAHeartbeatTimeout
	}
	cfg := c.LimiterConfig().WithDefaults()
	c.mu.Lock()
	instanceID := c.instanceID
	legacyMembership := c.legacyMembership
	c.mu.Unlock()

	args := []string{redisChannelConfig}
	if cfg.LifecycleConfigRevision > 0 {
		args = append(args, strconv.FormatInt(cfg.LifecycleConfigRevision, 10))
		if legacyMembership {
			return args, cfg.CPAHeartbeatTimeout
		}
		state := recoveryState(c.recoveryState.Load())
		if state == recoveryStateTakeoverEligible || state == recoveryStateSwitchingTakeover {
			args = append(args, "takeover")
		}
		args = append(args, instanceID)
	}
	return args, cfg.CPAHeartbeatTimeout
}

func (c *Client) markMembershipTakeoverEligible() {
	if c == nil {
		return
	}
	if !c.recoveryState.CompareAndSwap(uint32(recoveryStateStable), uint32(recoveryStateTakeoverEligible)) {
		c.recoveryState.CompareAndSwap(uint32(recoveryStateSwitching), uint32(recoveryStateSwitchingTakeover))
	}
}

func (c *Client) RPushRequestLog(ctx context.Context, payload []byte) error {
	cmd, errClient := c.commandClient()
	if errClient != nil {
		return errClient
	}
	if len(payload) == 0 {
		return nil
	}
	return cmd.RPush(ctx, redisKeyRequestLog, payload).Err()
}

func (c *Client) handleSubscriptionPayload(channel string, payload string, onConfig func([]byte) error) error {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(channel)) {
	case redisChannelConfig:
		if onConfig == nil {
			return nil
		}
		return onConfig([]byte(payload))
	case redisChannelCluster:
		return c.updateClusterNodesFromPayload([]byte(payload))
	default:
		return nil
	}
}

// StartConfigSubscriber connects to home, fetches config once via GET config, then subscribes to
// the "config" channel to receive runtime config updates.
//
// The subscription connection is treated as the home heartbeat. HeartbeatOK is set to true only
// after the initial GET config succeeds and the SUBSCRIBE connection is established. When the
// subscription ends unexpectedly, HeartbeatOK becomes false and the loop reconnects.
func (c *Client) StartConfigSubscriber(ctx context.Context, onConfig func([]byte) error) {
	if c == nil {
		return
	}
	if !c.Enabled() {
		return
	}
	if onConfig == nil {
		return
	}

	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				c.heartbeatOK.Store(false)
				return
			default:
			}
		}

		c.heartbeatOK.Store(false)
		c.Close()

		if errEnsure := c.ensureClients(); errEnsure != nil {
			log.Warn("unable to connect to home control center, retrying in 1 second")
			c.markReconnectFailure("connect")
			sleepWithContext(ctx, homeReconnectInterval)
			continue
		}

		if errPing := c.Ping(ctx); errPing != nil {
			log.Warn("unable to connect to home control center, retrying in 1 second")
			c.markReconnectFailure("ping")
			sleepWithContext(ctx, homeReconnectInterval)
			continue
		}

		raw, errGet := c.GetConfig(ctx)
		if errGet != nil {
			log.Warn("unable to fetch config from home control center, retrying in 1 second")
			c.markReconnectFailure("config fetch")
			sleepWithContext(ctx, homeReconnectInterval)
			continue
		}
		if errApply := onConfig(raw); errApply != nil {
			log.Warn("unable to apply config from home control center, retrying in 1 second")
			sleepWithContext(ctx, homeReconnectInterval)
			continue
		}

		sub, errSubClient := c.subscriptionClient()
		if errSubClient != nil {
			c.markReconnectFailure("subscribe client")
			sleepWithContext(ctx, homeReconnectInterval)
			continue
		}

		args, receiveTimeout := c.subscriptionParameters()
		pubsub := sub.Subscribe(ctx, args...)
		if pubsub == nil {
			c.markReconnectFailure("subscribe")
			sleepWithContext(ctx, homeReconnectInterval)
			continue
		}

		// Ensure the subscription is established before marking heartbeat OK.
		if _, errReceive := pubsub.ReceiveTimeout(ctx, receiveTimeout); errReceive != nil {
			_ = pubsub.Close()
			if IsLegacyMembershipProtocolError(errReceive) {
				c.EnableLegacyMembership()
			} else if IsMembershipTakeoverUnavailableError(errReceive) {
				c.SuppressTakeover()
			}
			c.markReconnectFailure("subscribe")
			sleepWithContext(ctx, homeReconnectInterval)
			continue
		}
		if len(args) > 1 {
			c.markMembershipTakeoverEligible()
		}

		c.resetReconnectFailures()
		c.heartbeatOK.Store(true)

		for {
			_, receiveTimeout = c.subscriptionParameters()
			event, errMsg := pubsub.ReceiveTimeout(ctx, receiveTimeout)
			if errMsg != nil {
				_ = pubsub.Close()
				c.heartbeatOK.Store(false)
				if isTimeoutError(errMsg) {
					c.markSubscriptionTimeout()
				} else {
					c.markReconnectFailure("subscription")
				}
				sleepWithContext(ctx, homeReconnectInterval)
				break
			}
			switch msg := event.(type) {
			case *redis.Message:
				if msg == nil {
					continue
				}
				if errApply := c.handleSubscriptionPayload(msg.Channel, msg.Payload, onConfig); errApply != nil {
					if strings.EqualFold(strings.TrimSpace(msg.Channel), redisChannelCluster) {
						log.Warn("failed to apply cluster update from home control center, ignoring")
					} else {
						log.Warn("failed to apply config update from home control center, ignoring")
					}
				}
			case *redis.Pong:
				c.resetReconnectFailures()
			case *redis.Subscription:
				continue
			default:
				log.Debugf("home subscription returned unsupported message type %T", event)
			}
		}
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	if ctx == nil {
		<-timer.C
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}
