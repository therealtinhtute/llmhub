package cliproxy

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/therealtinhtute/llmhub/internal/config"
	"github.com/therealtinhtute/llmhub/internal/nativeproviders"
	"github.com/therealtinhtute/llmhub/internal/watcher"
	"github.com/therealtinhtute/llmhub/internal/watcher/synthesizer"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

const storageWatcherPollInterval = 2 * time.Second

type RuntimeStorage interface {
	LoadConfigBytes(ctx context.Context) ([]byte, error)
	CurrentVersion(ctx context.Context) (int64, error)
	AuthVersion(ctx context.Context) (string, error)
	List(ctx context.Context) ([]*coreauth.Auth, error)
}

func NewStorageWatcherFactory(store RuntimeStorage) WatcherFactory {
	return func(_ string, _ string, reload func(*config.Config)) (*WatcherWrapper, error) {
		w := &storageWatcher{
			store:       store,
			reload:      reload,
			currentAuth: make(map[string]*coreauth.Auth),
		}
		return &WatcherWrapper{
			start: func(ctx context.Context) error {
				return w.start(ctx)
			},
			stop: func() error {
				w.stop()
				return nil
			},
			setConfig: func(cfg *config.Config) {
				w.setConfig(cfg)
			},
			snapshotAuths: func() []*coreauth.Auth {
				return w.snapshotAuths()
			},
			setUpdateQueue: func(queue chan<- watcher.AuthUpdate) {
				w.setUpdateQueue(queue)
			},
			dispatchRuntimeUpdate: func(update watcher.AuthUpdate) bool {
				return w.dispatch(update)
			},
		}, nil
	}
}

type storageWatcher struct {
	store  RuntimeStorage
	reload func(*config.Config)

	mu            sync.Mutex
	cfg           *config.Config
	queue         chan<- watcher.AuthUpdate
	currentAuth   map[string]*coreauth.Auth
	synthAuths    map[string]*coreauth.Auth
	configVersion int64
	authVersion   string
	cancel        context.CancelFunc
}

func (w *storageWatcher) start(ctx context.Context) error {
	if w == nil || w.store == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.cancel = cancel
	w.mu.Unlock()

	if err := w.poll(ctx); err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(storageWatcherPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.poll(ctx); err != nil {
					log.WithError(err).Warn("storage watcher poll failed")
				}
			}
		}
	}()
	return nil
}

func (w *storageWatcher) stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.cancel = nil
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (w *storageWatcher) setConfig(cfg *config.Config) {
	w.mu.Lock()
	w.cfg = cfg
	w.mu.Unlock()
}

func (w *storageWatcher) setUpdateQueue(queue chan<- watcher.AuthUpdate) {
	w.mu.Lock()
	w.queue = queue
	w.mu.Unlock()
}

func (w *storageWatcher) snapshotAuths() []*coreauth.Auth {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]*coreauth.Auth, 0, len(w.currentAuth))
	for _, auth := range w.currentAuth {
		out = append(out, auth.Clone())
	}
	return out
}

func (w *storageWatcher) dispatch(update watcher.AuthUpdate) bool {
	w.mu.Lock()
	queue := w.queue
	w.mu.Unlock()
	if queue == nil {
		return false
	}
	select {
	case queue <- update:
		return true
	default:
		return false
	}
}

func (w *storageWatcher) poll(ctx context.Context) error {
	if err := w.pollConfig(ctx); err != nil {
		return err
	}
	if err := w.pollAuth(ctx); err != nil {
		return err
	}
	return w.pollSynthAuths(ctx)
}

func (w *storageWatcher) pollConfig(ctx context.Context) error {
	version, err := w.store.CurrentVersion(ctx)
	if err != nil {
		return err
	}
	w.mu.Lock()
	changed := w.configVersion != 0 && version != 0 && version != w.configVersion
	if w.configVersion == 0 {
		w.configVersion = version
	}
	w.mu.Unlock()
	if !changed {
		return nil
	}
	raw, err := w.store.LoadConfigBytes(ctx)
	if err != nil {
		return err
	}
	cfg, err := config.ParseConfigBytes(raw)
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.configVersion = version
	w.cfg = cfg
	w.mu.Unlock()
	if w.reload != nil {
		w.reload(cfg)
	}
	return nil
}

func (w *storageWatcher) pollAuth(ctx context.Context) error {
	version, err := w.store.AuthVersion(ctx)
	if err != nil {
		return err
	}
	w.mu.Lock()
	if w.authVersion == version {
		w.mu.Unlock()
		return nil
	}
	first := w.authVersion == ""
	w.authVersion = version
	oldAuth := cloneAuthMap(w.currentAuth)
	w.mu.Unlock()

	items, err := w.store.List(ctx)
	if err != nil {
		return err
	}
	nextAuth := make(map[string]*coreauth.Auth, len(items))
	for _, auth := range items {
		if auth == nil || auth.ID == "" {
			continue
		}
		nextAuth[auth.ID] = auth.Clone()
	}

	updates := make([]watcher.AuthUpdate, 0, len(oldAuth)+len(nextAuth))
	for id, auth := range nextAuth {
		old, ok := oldAuth[id]
		if !ok {
			if !first {
				updates = append(updates, watcher.AuthUpdate{Action: watcher.AuthUpdateActionAdd, ID: id, Auth: auth.Clone()})
			}
			continue
		}
		if !authEqualJSON(old, auth) {
			updates = append(updates, watcher.AuthUpdate{Action: watcher.AuthUpdateActionModify, ID: id, Auth: auth.Clone()})
		}
	}
	for id := range oldAuth {
		if _, ok := nextAuth[id]; !ok {
			updates = append(updates, watcher.AuthUpdate{Action: watcher.AuthUpdateActionDelete, ID: id})
		}
	}

	w.mu.Lock()
	w.currentAuth = nextAuth
	w.mu.Unlock()
	for _, update := range updates {
		w.dispatch(update)
	}
	return nil
}

// pollSynthAuths diffs config-synthesized auths (openai-compatibility entries,
// including native provider projections) and dispatches Add/Modify/Delete for
// them. Unlike pollAuth it runs on every poll: native provider records live in
// their own store and never bump AuthVersion, so the version gate would starve
// them. Dispatch runs unconditionally (Add is idempotent for unchanged state).
func (w *storageWatcher) pollSynthAuths(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	next := w.synthConfigAuths(ctx)
	w.mu.Lock()
	old := w.synthAuths
	w.synthAuths = next
	w.mu.Unlock()

	updates := make([]watcher.AuthUpdate, 0, len(next)+len(old))
	for id, auth := range next {
		if existing, ok := old[id]; !ok {
			updates = append(updates, watcher.AuthUpdate{Action: watcher.AuthUpdateActionAdd, ID: id, Auth: auth.Clone()})
		} else if !authEqualJSON(existing, auth) {
			updates = append(updates, watcher.AuthUpdate{Action: watcher.AuthUpdateActionModify, ID: id, Auth: auth.Clone()})
		}
	}
	for id := range old {
		if _, ok := next[id]; !ok {
			updates = append(updates, watcher.AuthUpdate{Action: watcher.AuthUpdateActionDelete, ID: id})
		}
	}
	for _, update := range updates {
		w.dispatch(update)
	}
	return nil
}

// synthConfigAuths synthesizes auths from the current config (openai-compatibility
// entries) plus native provider projections hydrated from the store. Timestamps
// are zeroed so repeated polls compare equal under authEqualJSON.
func (w *storageWatcher) synthConfigAuths(ctx context.Context) map[string]*coreauth.Auth {
	w.mu.Lock()
	cfg := w.cfg
	w.mu.Unlock()
	if cfg == nil {
		return nil
	}
	if ns, ok := w.store.(nativeproviders.Store); ok {
		if err := nativeproviders.HydrateConfig(ctx, cfg, ns); err != nil {
			log.WithError(err).Warn("storage watcher: failed to hydrate native provider resources")
		}
	}
	out := make(map[string]*coreauth.Auth)
	sctx := &synthesizer.SynthesisContext{
		Config:      cfg,
		Now:         time.Now(),
		IDGenerator: synthesizer.NewStableIDGenerator(),
	}
	auths, err := synthesizer.NewConfigSynthesizer().Synthesize(sctx)
	if err != nil {
		log.WithError(err).Warn("storage watcher: failed to synthesize config auths")
		return out
	}
	for _, a := range auths {
		if a == nil || a.ID == "" {
			continue
		}
		a.CreatedAt = time.Time{}
		a.UpdatedAt = time.Time{}
		out[a.ID] = a.Clone()
	}
	return out
}

func cloneAuthMap(in map[string]*coreauth.Auth) map[string]*coreauth.Auth {
	out := make(map[string]*coreauth.Auth, len(in))
	for id, auth := range in {
		out[id] = auth.Clone()
	}
	return out
}

func authEqualJSON(a, b *coreauth.Auth) bool {
	if a == nil || b == nil {
		return a == b
	}
	rawA, errA := json.Marshal(a)
	rawB, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return reflect.DeepEqual(a, b)
	}
	return string(rawA) == string(rawB)
}
