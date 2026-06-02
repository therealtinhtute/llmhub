package runtimepolicy

import "github.com/therealtinhtute/llmhub/internal/config"

// RuntimeStorage captures runtime durability rules that override config toggles.
type RuntimeStorage struct {
	PostgresDurable bool
}

func (p RuntimeStorage) PostgresDurableMode() bool {
	return p.PostgresDurable
}

func (p RuntimeStorage) AllowsFileAppLogs(cfg *config.Config) bool {
	return cfg != nil && cfg.LoggingToFile && !p.PostgresDurableMode()
}

func (p RuntimeStorage) AllowsRequestLogArchives(cfg *config.Config) bool {
	return cfg != nil && cfg.RequestLog && !p.PostgresDurableMode()
}

func (p RuntimeStorage) AllowsRequestErrorArchives() bool {
	return !p.PostgresDurableMode()
}
