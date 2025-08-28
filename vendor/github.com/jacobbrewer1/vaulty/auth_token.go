package vaulty

import hashiVault "github.com/hashicorp/vault/api"

// tokenLogin authenticates with Vault using a token.
func tokenLogin(v *hashiVault.Client, token string) (*hashiVault.Secret, error) {
	v.SetToken(token)
	return v.Auth().Token().LookupSelf()
}
