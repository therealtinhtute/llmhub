package managementasset

import _ "embed"

//go:embed static/management.html
var embeddedManagementHTML []byte

// EmbeddedManagementHTML returns the compiled management panel as raw bytes.
func EmbeddedManagementHTML() []byte {
	return embeddedManagementHTML
}
