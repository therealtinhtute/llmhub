package home

import (
	"crypto/sha256"
	"encoding/hex"
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
func CurrentKVClient() (*Client, bool, error) {
	client := Current()
	if client == nil || !client.Enabled() {
		return nil, false, nil
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
