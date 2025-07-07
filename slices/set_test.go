package slices

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewSet(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		NewSet(2, 2, 2),
		NewSet(2),
	)
}

func Test_Set(t *testing.T) {
	t.Parallel()

	t.Run("Difference", func(t *testing.T) {
		t.Parallel()
		a := NewSet(1, 2, 3, 4, 5)
		b := NewSet(1, 3, 4)

		require.Equal(
			t,
			NewSet(2, 5),
			a.Difference(b),
		)
	})

	t.Run("Items", func(t *testing.T) {
		t.Parallel()
		a := NewSet(1, 2, 3, 4, 5)
		items := a.Items()
		require.Len(t, items, 5)
		require.Contains(t, items, 1)
		require.Contains(t, items, 2)
		require.Contains(t, items, 3)
		require.Contains(t, items, 4)
		require.Contains(t, items, 5)
	})

	t.Run("Each", func(t *testing.T) {
		t.Parallel()
		items := make([]int, 0)
		NewSet(1, 2, 3, 4, 5).Each(func(item int) {
			items = append(items, item)
		})

		require.Len(t, items, 5)
		require.Contains(t, items, 1)
		require.Contains(t, items, 2)
		require.Contains(t, items, 3)
		require.Contains(t, items, 4)
		require.Contains(t, items, 5)
	})

	t.Run("Union", func(t *testing.T) {
		t.Parallel()
		a := NewSet(1, 2, 3, 4, 5)
		b := NewSet(1, 3, 4, 6)

		require.Equal(
			t,
			NewSet(1, 2, 3, 4, 5, 6),
			a.Union(b),
		)
	})
}
