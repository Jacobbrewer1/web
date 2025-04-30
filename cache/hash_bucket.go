package cache

// HashBucket is an interface that defines a method to check if a given key belongs to the current bucket.
type HashBucket interface { // nolint:iface // Used outside the package and allows for future different caches to take to the interface
	// InBucket checks if the provided key belongs to the current bucket.
	InBucket(key string) bool
}
