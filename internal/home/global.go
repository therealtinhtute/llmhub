package home

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"
)

var currentClient atomic.Value // *Client

// SetCurrent sets the active home client used by runtime integrations.
func SetCurrent(client *Client) {
	currentClient.Store(client)
}

// Current returns the active home client instance, if any.
func Current() *Client {
	if v := currentClient.Load(); v != nil {
		if client, ok := v.(*Client); ok {
			return client
		}
	}
	return nil
}

// CurrentKVClient returns the active Home client when Home-backed KV is available.
// A configured-but-disabled or disconnected client reports homeMode=true with an
// error so callers fail the Home write rather than silently using local storage.
// Ported from upstream CLIProxyAPI (internal/home/kv_helpers.go).
func CurrentKVClient() (*Client, bool, error) {
	client := Current()
	if client == nil {
		return nil, false, nil
	}
	if !client.Enabled() {
		return nil, true, fmt.Errorf("home kv store unavailable: %w", ErrDisabled)
	}
	if !client.HeartbeatOK() {
		return nil, true, fmt.Errorf("home kv store unavailable: %w", ErrNotConnected)
	}
	return client, true, nil
}

func HashKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty"
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// ClearCurrent removes the active home client.
func ClearCurrent() {
	currentClient.Store((*Client)(nil))
}
