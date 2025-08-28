package vaulty

import (
	"fmt"

	hashiVault "github.com/hashicorp/vault/api"
)

// CipherTextFromSecret extracts the ciphertext from a Vault secret.
func CipherTextFromSecret(transitEncryptSecret *hashiVault.Secret) string {
	switch {
	case transitEncryptSecret == nil:
		return ""
	case transitEncryptSecret.Data == nil:
		return ""
	case transitEncryptSecret.Data[TransitKeyCipherText] == nil:
		return ""
	}

	ct, ok := transitEncryptSecret.Data[TransitKeyCipherText].(string)
	if !ok {
		return ""
	}

	return ct
}

// uintToInt converts a uint to an int, returning an error if the conversion would overflow.
func uintToInt(u uint) (int, error) {
	if u > uint(maxInt) {
		return 0, fmt.Errorf("uint value %d overflows int", u)
	}
	return int(u), nil
}
