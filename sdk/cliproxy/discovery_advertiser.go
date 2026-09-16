package cliproxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/discovery"
)

// This file ports upstream CLIProxyAPI's LAN discovery advertiser lifecycle
// (3428110d49be, hardened through 20ec9b83a120). Upstream stores the manager on
// the Service struct and invokes it from Run/applyConfigRuntime/Shutdown.
// llmhub keeps the manager in a per-service registry and wires it through
// Builder.WithDiscoveryAdvertiser: the OnAfterStart hook performs the initial
// apply, the wrapped watcher factory re-applies on every config reload (the
// storage watcher observes management-API writes too), and callers invoke
// Service.ShutdownDiscovery after Run returns.

const defaultDiscoveryRefreshInterval = 15 * time.Second

type discoveryRefresh struct {
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

type discoveryStart struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (r *discoveryRefresh) stopAndWait() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stop)
	})
	<-r.done
}

// discoveryManagers maps each Service to its advertiser manager so discovery
// lifecycle methods can live in this file without modifying the Service struct.
var discoveryManagers sync.Map // map[*Service]*discoveryAdvertiserManager

type discoveryAdvertiserManager struct {
	mu              sync.Mutex
	advertiser      discovery.Advertiser
	enabled         bool
	lastSpec        discovery.ServiceSpec
	lastCfg         *config.Config
	lastPort        int
	lastTLS         bool
	generation      uint64
	refresh         *discoveryRefresh
	activeStart     *discoveryStart
	closed          bool
	boundHost       string
	boundPort       int
	boundTLS        bool
	boundEndpoint   bool
	refreshInterval time.Duration
	newAdvertiser   func() discovery.Advertiser
	buildSpec       func(*config.Config, int, bool) (discovery.ServiceSpec, error)

	// cfgSource returns the service's current config. The refresh loop polls it
	// so config swaps applied outside the watcher callback (e.g. Home-mode
	// applyHomeOverlay) still reach the manager within one interval.
	cfgSource func() *config.Config
}

func newZeroconfAdvertiser() discovery.Advertiser {
	return discovery.NewZeroconfAdvertiser()
}

func newDiscoveryAdvertiserManager() *discoveryAdvertiserManager {
	return &discoveryAdvertiserManager{
		newAdvertiser: newZeroconfAdvertiser,
		buildSpec:     discovery.BuildServiceSpec,
	}
}

func (s *Service) getDiscoveryManager() *discoveryAdvertiserManager {
	if s == nil {
		return nil
	}
	if v, ok := discoveryManagers.Load(s); ok {
		return v.(*discoveryAdvertiserManager)
	}
	mgr := newDiscoveryAdvertiserManager()
	actual, _ := discoveryManagers.LoadOrStore(s, mgr)
	return actual.(*discoveryAdvertiserManager)
}

// currentConfig returns the service's active configuration under read lock.
func (s *Service) currentConfig() *config.Config {
	if s == nil {
		return nil
	}
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

func (s *Service) applyDiscoveryConfig(cfg *config.Config) {
	s.applyDiscoveryConfigContext(context.Background(), cfg)
}

func (s *Service) applyDiscoveryConfigContext(ctx context.Context, cfg *config.Config) bool {
	if s == nil || cfg == nil || (ctx != nil && ctx.Err() != nil) {
		return false
	}
	mgr := s.getDiscoveryManager()
	if mgr == nil {
		return false
	}
	return mgr.ApplyContext(ctx, cfg, cfg.Port, cfg.TLS.Enable)
}

// ShutdownDiscovery stops the service's mDNS advertiser, if any. It is safe to
// call on services built without WithDiscoveryAdvertiser and after Run returns.
func (s *Service) ShutdownDiscovery() error {
	if s == nil {
		return nil
	}
	mgr := s.getDiscoveryManager()
	if mgr == nil {
		return nil
	}
	defer discoveryManagers.Delete(s)
	return mgr.Shutdown()
}

// WithDiscoveryAdvertiser wires LAN discovery advertising into the service
// lifecycle without touching the shared Service struct:
//   - OnAfterStart performs the initial advertiser apply once the API server is up.
//   - The watcher factory's reload callback is wrapped so every config reload
//     (file watcher or Postgres storage watcher, which also observes
//     management-API config writes) re-applies the discovery configuration.
//
// Callers should invoke Service.ShutdownDiscovery once Run returns. Call this
// after other With* options so it wraps the final watcher factory and hooks.
func (b *Builder) WithDiscoveryAdvertiser() *Builder {
	if b == nil {
		return b
	}
	mgr := newDiscoveryAdvertiserManager()

	prevHook := b.hooks.OnAfterStart
	b.hooks.OnAfterStart = func(s *Service) {
		if prevHook != nil {
			prevHook(s)
		}
		if s == nil {
			return
		}
		mgr.mu.Lock()
		mgr.cfgSource = s.currentConfig
		mgr.mu.Unlock()
		discoveryManagers.Store(s, mgr)
		s.applyDiscoveryConfig(s.currentConfig())
	}

	prevFactory := b.watcherFactory
	b.watcherFactory = func(configPath, authDir string, reload func(*config.Config)) (*WatcherWrapper, error) {
		factory := prevFactory
		if factory == nil {
			factory = defaultWatcherFactory
		}
		return factory(configPath, authDir, func(newCfg *config.Config) {
			if newCfg != nil {
				mgr.ApplyContext(context.Background(), newCfg, newCfg.Port, newCfg.TLS.Enable)
			}
			if reload != nil {
				reload(newCfg)
			}
		})
	}
	return b
}

func (m *discoveryAdvertiserManager) ApplyContext(ctx context.Context, cfg *config.Config, port int, tlsEnabled bool) bool {
	return m.applyContext(ctx, cfg, port, tlsEnabled, nil)
}

func (m *discoveryAdvertiserManager) applyContext(ctx context.Context, cfg *config.Config, port int, tlsEnabled bool, expectedRefresh *discoveryRefresh) bool {
	if m == nil || cfg == nil {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return false
	}

	m.mu.Lock()
	if m.closed || (expectedRefresh != nil && (m.refresh != expectedRefresh || m.lastCfg != cfg)) {
		m.mu.Unlock()
		return false
	}
	activeStart := m.activeStart
	m.generation++
	applyGeneration := m.generation
	if !m.boundEndpoint {
		m.boundHost = cfg.Host
		m.boundPort = port
		m.boundTLS = tlsEnabled
		m.boundEndpoint = true
	}
	boundCfg := *cfg
	boundCfg.Host = m.boundHost
	boundPort := m.boundPort
	boundTLS := m.boundTLS
	m.lastCfg = cfg
	m.lastPort = boundPort
	m.lastTLS = boundTLS
	m.mu.Unlock()
	if activeStart != nil {
		activeStart.cancel()
		<-activeStart.done
	}

	m.mu.Lock()
	if m.closed || m.generation != applyGeneration || m.lastCfg != cfg {
		m.mu.Unlock()
		return false
	}
	if !cfg.Discovery.Enabled {
		oldAdv := m.advertiser
		refreshStop := m.detachRefreshLocked()
		m.advertiser = nil
		m.enabled = false
		m.lastSpec = discovery.ServiceSpec{}
		m.mu.Unlock()

		m.stopRefresh(refreshStop)
		if oldAdv != nil {
			log.Info("discovery: stopping mDNS advertisement (disabled by config)")
			_ = oldAdv.Stop()
		}
		return true
	}
	m.ensureRefreshLocked()
	buildSpec := m.buildSpec
	newAdvertiser := m.newAdvertiser
	m.mu.Unlock()

	if buildSpec == nil {
		buildSpec = discovery.BuildServiceSpec
	}
	spec, err := buildSpec(&boundCfg, boundPort, boundTLS)
	if err != nil {
		m.mu.Lock()
		if !m.closed && m.generation == applyGeneration && m.lastCfg == cfg && cfg.Discovery.Enabled {
			oldAdv := m.advertiser
			m.advertiser = nil
			m.enabled = false
			m.lastSpec = discovery.ServiceSpec{}
			m.generation++
			m.mu.Unlock()
			if oldAdv != nil {
				log.Info("discovery: stopping stale mDNS advertisement after spec build failure")
				_ = oldAdv.Stop()
			}
		} else {
			m.mu.Unlock()
		}
		log.Warnf("discovery: failed to build service spec: %v", err)
		return false
	}

	m.mu.Lock()
	if m.closed || m.generation != applyGeneration || m.lastCfg != cfg || !cfg.Discovery.Enabled || m.activeStart != nil {
		m.mu.Unlock()
		return false
	}
	if m.enabled && m.advertiser != nil && specEqual(m.lastSpec, spec) {
		m.mu.Unlock()
		return true
	}

	oldAdv := m.advertiser
	m.advertiser = nil
	m.generation++
	gen := m.generation
	startCtx, startCancel := context.WithCancel(ctx)
	start := &discoveryStart{cancel: startCancel, done: make(chan struct{})}
	m.activeStart = start
	m.mu.Unlock()

	if oldAdv != nil {
		_ = oldAdv.Stop()
	}
	if newAdvertiser == nil {
		newAdvertiser = newZeroconfAdvertiser
	}
	adv := newAdvertiser()
	var errStart error
	if adv == nil {
		errStart = fmt.Errorf("discovery: advertiser factory returned nil")
	} else {
		errStart = adv.Start(startCtx, spec)
	}
	if errStart == nil && startCtx.Err() != nil {
		errStart = startCtx.Err()
	}

	m.mu.Lock()
	valid := !m.closed && m.generation == gen && m.activeStart == start && m.lastCfg == cfg && cfg.Discovery.Enabled && startCtx.Err() == nil
	if valid && errStart == nil {
		m.advertiser = adv
		m.enabled = true
		m.lastSpec = spec
	}
	m.mu.Unlock()
	startCancel()

	if !valid || errStart != nil {
		if adv != nil {
			_ = adv.Stop()
		}
	}
	m.mu.Lock()
	if m.activeStart == start {
		m.activeStart = nil
	}
	m.mu.Unlock()
	close(start.done)

	if !valid || errStart != nil {
		if errStart != nil {
			m.mu.Lock()
			if m.generation == gen {
				m.enabled = false
			}
			m.mu.Unlock()
			log.Warnf("discovery: failed to start mDNS advertiser: %v (degraded, HTTP intact)", errStart)
		}
		return false
	}

	log.Infof("discovery: advertising as '%s.%s' on port %d", spec.InstanceName, spec.ServiceType, boundPort)
	return true
}

func (m *discoveryAdvertiserManager) Shutdown() error {
	m.mu.Lock()
	m.closed = true
	oldAdv := m.advertiser
	activeStart := m.activeStart
	refreshStop := m.detachRefreshLocked()
	m.advertiser = nil
	m.enabled = false
	m.lastSpec = discovery.ServiceSpec{}
	m.lastCfg = nil
	m.generation++
	m.mu.Unlock()

	if activeStart != nil {
		activeStart.cancel()
		<-activeStart.done
	}
	m.stopRefresh(refreshStop)
	if oldAdv != nil {
		return oldAdv.Stop()
	}
	return nil
}

func (m *discoveryAdvertiserManager) refreshPeriod() (time.Duration, bool) {
	if m.refreshInterval < 0 {
		return 0, false
	}
	if m.refreshInterval == 0 {
		return defaultDiscoveryRefreshInterval, true
	}
	return m.refreshInterval, true
}

func (m *discoveryAdvertiserManager) ensureRefreshLocked() {
	if _, ok := m.refreshPeriod(); !ok {
		return
	}
	if m.refresh != nil {
		return
	}
	refresh := &discoveryRefresh{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	m.refresh = refresh
	interval, _ := m.refreshPeriod()
	go func() {
		defer close(refresh.done)
		m.refreshLoop(refresh, interval)
	}()
}

func (m *discoveryAdvertiserManager) detachRefreshLocked() *discoveryRefresh {
	refresh := m.refresh
	m.refresh = nil
	return refresh
}

func (m *discoveryAdvertiserManager) stopRefresh(refresh *discoveryRefresh) {
	if refresh != nil {
		refresh.stopAndWait()
	}
}

func (m *discoveryAdvertiserManager) refreshLoop(refresh *discoveryRefresh, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-refresh.stop:
			return
		case <-ticker.C:
			m.mu.Lock()
			cfg := m.lastCfg
			port := m.lastPort
			tlsEnabled := m.lastTLS
			source := m.cfgSource
			m.mu.Unlock()
			latest := cfg
			if source != nil {
				if current := source(); current != nil {
					latest = current
				}
			}
			if latest == nil {
				continue
			}
			if latest != cfg {
				// Config was swapped by a path that does not reach the manager
				// directly (e.g. Home-mode overlay); run a full apply on it.
				_ = m.applyContext(context.Background(), latest, latest.Port, latest.TLS.Enable, nil)
				continue
			}
			if !cfg.Discovery.Enabled {
				continue
			}
			_ = m.applyContext(context.Background(), cfg, port, tlsEnabled, refresh)
		}
	}
}

func specEqual(a, b discovery.ServiceSpec) bool {
	if a.InstanceName != b.InstanceName ||
		a.ServiceType != b.ServiceType ||
		a.Domain != b.Domain ||
		a.Port != b.Port ||
		len(a.Subtypes) != len(b.Subtypes) ||
		len(a.TextRecords) != len(b.TextRecords) ||
		len(a.Interfaces) != len(b.Interfaces) ||
		len(a.AdvertisedIPs) != len(b.AdvertisedIPs) {
		return false
	}
	for i := range a.Subtypes {
		if a.Subtypes[i] != b.Subtypes[i] {
			return false
		}
	}
	for i := range a.TextRecords {
		if a.TextRecords[i] != b.TextRecords[i] {
			return false
		}
	}
	for i := range a.Interfaces {
		if a.Interfaces[i].Index != b.Interfaces[i].Index || a.Interfaces[i].Name != b.Interfaces[i].Name {
			return false
		}
	}
	for i := range a.AdvertisedIPs {
		if a.AdvertisedIPs[i] != b.AdvertisedIPs[i] {
			return false
		}
	}
	return true
}
