package vault

import (
	"context"
	"errors"
	"log/slog"
	"os"

	hashivault "github.com/hashicorp/vault/api"
	kubernetesauth "github.com/hashicorp/vault/api/auth/kubernetes"
)

const (
	// EnvServiceAccountName is the environment variable that contains the service account name.
	EnvServiceAccountName = "SERVICE_ACCOUNT_NAME" // nolint:gosec // This is detected as a secret

	// KubernetesServiceAccountTokenPath is the path to the Kubernetes service account token.
	kubernetesServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token" // nolint:gosec // This is detected as a secret
)

// ClientOption is a function that configures the client.
type ClientOption func(c *client) error

// WithContext sets the context for the client.
func WithContext(ctx context.Context) ClientOption {
	return func(c *client) error {
		c.ctx = ctx
		return nil
	}
}

// WithLogger sets the logger for the client.
func WithLogger(l *slog.Logger) ClientOption {
	return func(c *client) error {
		c.l = l
		return nil
	}
}

// WithConfig sets the config for the client.
func WithConfig(config *hashivault.Config) ClientOption {
	return func(c *client) error {
		c.config = config
		return nil
	}
}

// WithKvv2Mount sets the KVv2 mount point for the client.
func WithKvv2Mount(mount string) ClientOption {
	return func(c *client) error {
		c.kvv2Mount = mount
		return nil
	}
}

// WithKubernetesServiceAccountAuth sets the Kubernetes authentication method for the client using a service account.
func WithKubernetesServiceAccountAuth(roleName string) ClientOption {
	return func(c *client) error {
		if roleName == "" {
			return errors.New("role name is empty")
		}

		c.loginFunc = func(v *hashivault.Client) (*hashivault.Secret, error) {
			sec, err := kubernetesLogin(v, roleName, kubernetesauth.WithServiceAccountTokenPath(kubernetesServiceAccountTokenPath))
			if err != nil {
				return nil, err
			}

			go c.renewAuthInfo()

			return sec, nil
		}
		return nil
	}
}

// WithKubernetesAuthFromEnv sets the Kubernetes authentication method for the client using a service account from environment variables.
func WithKubernetesAuthFromEnv() ClientOption {
	return func(c *client) error {
		roleFromEnv := os.Getenv(EnvServiceAccountName)
		if roleFromEnv == "" {
			return errors.New("role name is not set in environment variable " + EnvServiceAccountName)
		}

		return WithKubernetesServiceAccountAuth(roleFromEnv)(c)
	}
}

// WithKubernetesAuth sets the Kubernetes authentication method for the client using the role and token provided.
func WithKubernetesAuth(role, token string) ClientOption {
	return func(c *client) error {
		if role == "" {
			return errors.New("role name is empty")
		} else if token == "" {
			return errors.New("token is empty")
		}

		c.loginFunc = func(v *hashivault.Client) (*hashivault.Secret, error) {
			sec, err := kubernetesLogin(v, role, kubernetesauth.WithServiceAccountToken(token))
			if err != nil {
				return nil, err
			}

			go c.renewAuthInfo()

			return sec, nil
		}
		return nil
	}
}
