package vaulty

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	hashiVault "github.com/hashicorp/vault/api"
)

var (
	// ErrSecretNotFound is returned when a secret is not found.
	ErrSecretNotFound = hashiVault.ErrSecretNotFound

	// ErrInvalidClient is returned when the client is nil.
	ErrInvalidClient = errors.New("client is nil")

	// ErrInvalidAuth is returned when the auth method is nil.
	ErrInvalidAuth = errors.New("auth method is nil")
)

// ClientHandler is an interface that provides access to the Vault client.
type ClientHandler interface {
	Client() *hashiVault.Client
}

// Client is an interface that provides access to the Vault client and its secrets.
type Client interface {
	ClientHandler

	// Path returns the secret path for the given name.
	Path(name string, opts ...PathOption) Repository
}

type (
	// RenewalFunc is a function that renews the lease for a secret.
	RenewalFunc = func() (*hashiVault.Secret, error)

	// loginFunc is a function that logs in to Vault and returns the secret.
	loginFunc = func(v *hashiVault.Client) (*hashiVault.Secret, error)
)

// client is a struct that implements the Client interface.
type client struct {
	ctx       context.Context
	l         *slog.Logger
	kvv2Mount string
	auth      loginFunc
	config    *hashiVault.Config

	// Below are set on initialization
	v         *hashiVault.Client
	authCreds *hashiVault.Secret
}

// NewClient creates a new Vault client with the given options.
func NewClient(opts ...ClientOption) (Client, error) {
	c := &client{
		ctx:       context.Background(),
		l:         slog.Default(),
		kvv2Mount: "",
		auth:      nil,
		config:    hashiVault.DefaultConfig(),
		v:         nil,
		authCreds: nil,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, fmt.Errorf("unable to apply client option: %w", err)
		}
	}

	if c.ctx == nil {
		c.ctx = context.Background()
	}

	vc, err := hashiVault.NewClient(c.config)
	if err != nil {
		return nil, fmt.Errorf("unable to create vault client: %w", err)
	} else if vc == nil {
		return nil, ErrInvalidClient
	}

	c.v = vc

	if c.auth == nil {
		return nil, ErrInvalidAuth
	}

	authCreds, err := c.auth(c.v)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate with Vault: %w", err)
	}

	c.authCreds = authCreds

	return c, nil
}

// renewAuthInfo renews the authentication information for the client.
func (c *client) renewAuthInfo() {
	err := RenewLease(c.ctx, c.l, c, "auth", c.authCreds, func() (*hashiVault.Secret, error) {
		authInfo, err := c.auth(c.v)
		if err != nil {
			return nil, fmt.Errorf("unable to renew auth info: %w", err)
		}

		c.authCreds = authInfo

		return authInfo, nil
	})
	if err != nil { // nolint:revive // Traditional error handling
		c.l.Error("unable to renew auth info", slog.String(loggingKeyError, err.Error()))
		os.Exit(1)
	}
}

// Client returns the Vault client.
func (c *client) Client() *hashiVault.Client {
	return c.v
}

// Path returns the secret path for the given name.
func (c *client) Path(name string, opts ...PathOption) Repository {
	p := &SecretPath{
		client: c,
		mount:  c.kvv2Mount, // Default to kvv2
		name:   name,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}
