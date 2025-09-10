package vault

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	hashivault "github.com/hashicorp/vault/api"
)

const (
	// pathKeyTransitDecrypt is the path key for decrypting data using the transit engine.
	pathKeyTransitDecrypt = "decrypt"

	// pathKeyTransitEncrypt is the path key for encrypting data using the transit engine.
	pathKeyTransitEncrypt = "encrypt"

	// TransitKeyCipherText is the key for ciphertext in transit operations.
	TransitKeyCipherText = "ciphertext"

	// TransitKeyPlainText is the key for plaintext in transit operations.
	TransitKeyPlainText = "plaintext"
)

type (
	// SecretRepository is an interface for interacting with Vault secrets.
	SecretRepository interface {
		// GetKvSecretV2 returns a map of secrets for the given path.
		GetKvSecretV2(ctx context.Context) (*hashivault.KVSecret, error)

		// GetSecret returns a map of secrets for the given path.
		GetSecret(ctx context.Context) (*hashivault.Secret, error)

		// TransitEncrypt encrypts the given data.
		TransitEncrypt(ctx context.Context, data string) (*hashivault.Secret, error)

		// TransitDecrypt decrypts the given data.
		TransitDecrypt(ctx context.Context, data string) (string, error)
	}

	// SecretPath represents a path to a secret in Vault.
	SecretPath struct {
		client  Client
		mount   string
		prefix  string
		name    string
		version uint
	}
)

// path returns the path of the secret.
func (c *SecretPath) path() string {
	if c.prefix != "" {
		return fmt.Sprintf("%s/%s", c.prefix, c.name)
	}

	return c.name
}

// pathWithType returns the path with the specified type.
func (c *SecretPath) pathWithType(k string) string {
	if c.prefix != "" {
		return fmt.Sprintf("%s/%s/%s", c.prefix, k, c.name)
	}

	return fmt.Sprintf("%s/%s", k, c.name)
}

// GetKvSecretV2 retrieves a versioned secret from the specified path.
func (c *SecretPath) GetKvSecretV2(ctx context.Context) (*hashivault.KVSecret, error) {
	version, err := uintToInt(c.version)
	if err != nil {
		return nil, fmt.Errorf("incompatible version: %w", err)
	}

	secret, err := c.client.Client().KVv2(c.mount).GetVersion(ctx, c.path(), version)
	if err != nil {
		return nil, fmt.Errorf("unable to read secret: %w", err)
	} else if secret == nil {
		return nil, ErrSecretNotFound
	}
	return secret, nil
}

// GetSecret retrieves a secret from the specified path.
func (c *SecretPath) GetSecret(ctx context.Context) (*hashivault.Secret, error) {
	secret, err := c.client.Client().Logical().ReadWithContext(ctx, c.path())
	if err != nil {
		return nil, fmt.Errorf("unable to read secrets: %w", err)
	} else if secret == nil {
		return nil, ErrSecretNotFound
	}
	return secret, nil
}

// TransitEncrypt encrypts the given data using the transit engine.
func (c *SecretPath) TransitEncrypt(ctx context.Context, data string) (*hashivault.Secret, error) {
	plaintext := base64.StdEncoding.EncodeToString([]byte(data))

	// Encrypt the data using the transit engine
	encryptData, err := c.client.Client().Logical().WriteWithContext(ctx, c.pathWithType(pathKeyTransitEncrypt), map[string]any{
		TransitKeyPlainText: plaintext,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt data: %w", err)
	}
	return encryptData, nil
}

// TransitDecrypt decrypts the given data using the transit engine.
func (c *SecretPath) TransitDecrypt(ctx context.Context, data string) (string, error) {
	// Decrypt the data using the transit engine
	decryptData, err := c.client.Client().Logical().WriteWithContext(ctx, c.pathWithType(pathKeyTransitDecrypt), map[string]any{
		TransitKeyCipherText: data,
	})
	if err != nil {
		return "", fmt.Errorf("unable to decrypt data: %w", err)
	}

	decryptDataStr, ok := decryptData.Data[TransitKeyPlainText].(string)
	if !ok {
		return "", errors.New("unable to convert decrypted data to string")
	}
	decoded, err := base64.StdEncoding.DecodeString(decryptDataStr)
	if err != nil {
		return "", fmt.Errorf("unable to decode base64 plaintext: %w", err)
	}
	return string(decoded), nil
}
