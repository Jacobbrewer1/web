package cache

import (
	"strconv"

	"github.com/serialx/hashring"
)

var _ HashBucket = new(FixedHashBucket)

// FixedHashBucket represents a mechanism which determines whether supplied keys belong to the current bucket index.
// It uses a consistent hashing ring to determine the bucket index for each key.
// The bucket index is advanced in a round-robin fashion, and the keys are distributed across the buckets based on their hash values.
type FixedHashBucket struct {
	hr    *hashring.HashRing
	index int
}

func NewFixedHashBucket(size uint) *FixedHashBucket {
	h := &FixedHashBucket{
		hr:    hashring.New(make([]string, 0)),
		index: 0,
	}

	for i := range uint64(size) {
		h.hr = h.hr.AddNode(strconv.FormatUint(i, 10))
	}

	return h
}

func (h *FixedHashBucket) InBucket(key string) bool {
	pos, _ := h.hr.GetNodePos(key)
	return pos == h.index
}

func (h *FixedHashBucket) Advance() {
	if h.index+1 >= h.hr.Size() {
		h.index = 0
	} else {
		h.index++
	}
}

func (h *FixedHashBucket) GetIndex() int {
	return h.index
}

func (h *FixedHashBucket) GetSize() int {
	return h.hr.Size()
}
