package authclient

import (
	"context"
)

// An AuthClient retrieves an authentication JWT via the Pomerium login API.
type AuthClient struct {
	cfg *config
}

// New creates a new AuthClient.
func New(options ...Option) *AuthClient {
	cfg := new(config)
	for _, o := range options {
		o(cfg)
	}
	return &AuthClient{
		cfg: cfg,
	}
}

// GetJWT retrieves a JWT from Pomerium.
func (client *AuthClient) GetJWT(ctx context.Context) (rawJWT string, err error) {

}
