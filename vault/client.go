package vault

import (
	"context"
	"errors"
	"log/slog"

	hashivault "github.com/hashicorp/vault/api"
)

var (
	// ErrSecretNotFound is returned when a secret is not found.
	ErrSecretNotFound = hashivault.ErrSecretNotFound

	// ErrInvalidClient is returned when the client is nil.
	ErrInvalidClient = errors.New("client is nil")

	// ErrInvalidAuth is returned when the loginFunc method is nil.
	ErrInvalidAuth = errors.New("loginFunc method is nil")
)

type (
	ClientGetter interface {
		Client() *hashivault.Client
	}

	// Client is an interface that provides access to the Vault client and its secrets.
	Client interface {
		ClientGetter
		// Path returns the secret path for the given name.
		Path(name string, opts ...PathOption) SecretRepository
	}
)

type (
	// RenewalFunc is a function that renews the lease for a secret.
	RenewalFunc = func() (*hashivault.Secret, error)

	// LoginFunc is a function that logs in to Vault and returns the secret.
	LoginFunc = func(v *hashivault.Client) (*hashivault.Secret, error)
)

type (
	client struct {
		ctx       context.Context
		l         *slog.Logger
		kvv2Mount string
		loginFunc LoginFunc
		config    *hashivault.Config

		// Below are set on initialization
		v               *hashivault.Client
		authCredentials *hashivault.Secret
	}
)

// NewClient creates a new Vault client with the given options.
func NewClient(opts ...ClientOption) (Client, error) {
	c := &client{
		ctx:    context.Background(),
		l:      slog.Default(),
		config: hashivault.DefaultConfig(),
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if c.loginFunc == nil {
		return nil, ErrInvalidAuth
	}

	v, err := hashivault.NewClient(c.config)
	if err != nil {
		return nil, err
	}
	c.v = v

	authCreds, err := c.loginFunc(c.v)
	if err != nil {
		return nil, err
	}
	c.authCredentials = authCreds

	return c, nil
}

// Client returns the Vault client.
func (c *client) Client() *hashivault.Client {
	return c.v
}

// Path returns the secret path for the given name.
func (c *client) Path(name string, opts ...PathOption) SecretRepository {
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
