package vault

import (
	"context"
	"errors"
	"fmt"
	"time"

	hashivault "github.com/hashicorp/vault/api"
	kubeauth "github.com/hashicorp/vault/api/auth/kubernetes"

	"github.com/jacobbrewer1/web/logging"
)

// kubernetesLogin authenticates with Vault using the Kubernetes auth method.
func kubernetesLogin(client *hashivault.Client, role string, token kubeauth.LoginOption) (*hashivault.Secret, error) {
	k8sAuth, err := kubeauth.NewKubernetesAuth(
		role,
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize kubernetes auth method: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	authInfo, err := client.Auth().Login(ctx, k8sAuth)
	if err != nil {
		return nil, fmt.Errorf("unable to login to kubernetes auth method: %w", err)
	}
	if authInfo == nil {
		return nil, errors.New("no auth info was returned after login")
	}

	client.SetToken(authInfo.Auth.ClientToken)
	return authInfo, nil
}

func (c *client) renewAuthInfo() {
	err := RenewLease(c.ctx, c.l, c, "auth", c.authCredentials, func() (*hashivault.Secret, error) {
		authInfo, err := c.loginFunc(c.v)
		if err != nil {
			return nil, fmt.Errorf("unable to renew auth info: %w", err)
		}

		c.authCredentials = authInfo

		return authInfo, nil
	})
	if err != nil {
		c.l.Error("unable to renew auth info",
			logging.KeyError, err,
		)
	}
}
