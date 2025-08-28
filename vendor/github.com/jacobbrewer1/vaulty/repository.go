package vaulty

import (
	"context"

	hashiVault "github.com/hashicorp/vault/api"
)

// Repository is an interface for interacting with Vault secrets.
type Repository interface {
	// GetKvSecretV2 returns a map of secrets for the given path.
	GetKvSecretV2(ctx context.Context) (*hashiVault.KVSecret, error)

	// GetSecret returns a map of secrets for the given path.
	GetSecret(ctx context.Context) (*hashiVault.Secret, error)

	// TransitEncrypt encrypts the given data.
	TransitEncrypt(ctx context.Context, data string) (*hashiVault.Secret, error)

	// TransitDecrypt decrypts the given data.
	TransitDecrypt(ctx context.Context, data string) (string, error)
}
