package auth

import (
	"strings"
	"sync"
	"time"

	internalconfig "github.com/therealtinhtute/llmhub/internal/config"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

const defaultHomeSessionAliasTTL = time.Hour

type homeSessionAliasEntry struct {
	canonical string
	expiresAt time.Time
	aliases   []string
}

type homeSessionAliasCache struct {
	mu      sync.Mutex
	entries map[string]homeSessionAliasEntry
}

func (c *homeSessionAliasCache) canonical(primary, fallback string, ttl time.Duration, now time.Time) string {
	primary = strings.TrimSpace(primary)
	fallback = strings.TrimSpace(fallback)
	if primary == "" {
		return ""
	}
	if ttl <= 0 {
		ttl = defaultHomeSessionAliasTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]homeSessionAliasEntry)
	}
	canonical := primary
	aliases := compactHomeSessionAliases(primary, fallback)
	if existing, ok := c.entryLocked(primary, now); ok {
		canonical = existing.canonical
		aliases = compactHomeSessionAliases(append(aliases, existing.aliases...)...)
	}
	if fallback != "" && fallback != primary {
		if existing, ok := c.entryLocked(fallback, now); ok {
			if canonical == primary {
				canonical = existing.canonical
			}
			aliases = compactHomeSessionAliases(append(aliases, existing.aliases...)...)
		}
	}
	aliases = compactHomeSessionAliases(append(aliases, canonical)...)
	entry := homeSessionAliasEntry{canonical: canonical, expiresAt: now.Add(ttl), aliases: aliases}
	for _, alias := range aliases {
		c.entries[alias] = entry
	}
	return canonical
}

func (c *homeSessionAliasCache) entryLocked(alias string, now time.Time) (homeSessionAliasEntry, bool) {
	entry, ok := c.entries[alias]
	if !ok {
		return homeSessionAliasEntry{}, false
	}
	if now.Before(entry.expiresAt) {
		return entry, true
	}
	for _, stale := range entry.aliases {
		delete(c.entries, stale)
	}
	return homeSessionAliasEntry{}, false
}

func (c *homeSessionAliasCache) clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries = nil
	c.mu.Unlock()
}

func compactHomeSessionAliases(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	aliases := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		aliases = append(aliases, value)
	}
	return aliases
}

func homeSessionAliasTTL(cfg *internalconfig.Config) time.Duration {
	if cfg == nil {
		return defaultHomeSessionAliasTTL
	}
	raw := strings.TrimSpace(cfg.Routing.SessionAffinityTTL)
	if raw == "" {
		return defaultHomeSessionAliasTTL
	}
	parsed, errParse := time.ParseDuration(raw)
	if errParse != nil || parsed <= 0 {
		return defaultHomeSessionAliasTTL
	}
	return parsed
}

func (m *Manager) homeDispatchSessionID(opts cliproxyexecutor.Options) string {
	primary, fallback := extractSessionIDs(opts.Headers, opts.OriginalRequest, opts.Metadata)
	if primary == "" || m == nil {
		return primary
	}
	cfg, _ := m.runtimeConfig.Load().(*internalconfig.Config)
	return m.homeSessionAliases.canonical(primary, fallback, homeSessionAliasTTL(cfg), time.Now())
}
