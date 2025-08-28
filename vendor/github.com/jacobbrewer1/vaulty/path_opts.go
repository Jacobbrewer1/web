package vaulty

// PathOption is a function that modifies the SecretPath struct.
type PathOption func(p *SecretPath)

// WithVersion sets the version of the secret.
func WithVersion(version uint) PathOption {
	return func(p *SecretPath) {
		p.version = version
	}
}

// WithPrefix sets the prefix of the secret path.
func WithPrefix(prefix string) PathOption {
	return func(p *SecretPath) {
		p.prefix = prefix
	}
}

// WithMount sets the mount of the secret path.
func WithMount(mount string) PathOption {
	return func(p *SecretPath) {
		p.mount = mount
	}
}
