package cache

type HashBucket interface {
	InBucket(key string) bool
}
