package auth

import (
	"context"
	"time"

	internalkiro "github.com/therealtinhtute/llmhub/internal/auth/kiro"
	"github.com/therealtinhtute/llmhub/internal/config"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
)

type KiroAuthenticator struct{}

func NewKiroAuthenticator() Authenticator { return &KiroAuthenticator{} }

func (KiroAuthenticator) Provider() string { return internalkiro.Provider }

func (KiroAuthenticator) RefreshLead() *time.Duration {
	lead := 5 * time.Minute
	return &lead
}

func (KiroAuthenticator) Login(context.Context, *config.Config, *LoginOptions) (*coreauth.Auth, error) {
	return nil, ErrRefreshNotSupported
}
