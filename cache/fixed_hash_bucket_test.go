package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_FixedHashBucket(t *testing.T) {
	t.Parallel()
	
	h := NewFixedHashBucket(2)
	v := "hello"

	in := h.InBucket(v)
	h.Advance()
	require.Equal(t, 1, h.index)

	in2 := h.InBucket(v)

	// Ensure that the same value cannot be in two buckets.
	require.NotEqual(t, in, in2)

	// Ensure Advance() wraps the index.
	h.Advance()
	require.Zero(t, h.index)
}
