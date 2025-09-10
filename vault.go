package web

import (
	"context"
	"log/slog"

	hashivault "github.com/hashicorp/vault/api"
	"github.com/spf13/viper"

	"github.com/jacobbrewer1/web/k8s"
	"github.com/jacobbrewer1/web/vault"
)

// defaultVaultAddr is the default address of the Vault server.
//
// This constant defines the default URL for the active Vault server,
// which is used when no custom address is provided in the configuration.
const (
	// defaultVaultAddr is the default address of the Vault server
	defaultVaultAddr = "http://vault-active.vault:8200"
)

// VaultClient creates and configures a new Vault client.
//
// Note:
//   - This is defined as a variable to allow users of the package to override it if needed.
var VaultClient = func(ctx context.Context, l *slog.Logger, v *viper.Viper) (vault.Client, error) {
	addr := v.GetString("vault.address")
	if addr == "" {
		addr = defaultVaultAddr
	}

	vaultCfg := hashivault.DefaultConfig()
	vaultCfg.Address = addr

	return vault.NewClient(
		vault.WithContext(ctx),
		vault.WithConfig(vaultCfg),
		vault.WithKubernetesServiceAccountAuth(k8s.ServiceAccountName()),
		vault.WithKvv2Mount(v.GetString("vault.kvv2_mount")),
		vault.WithLogger(l),
	)
}
