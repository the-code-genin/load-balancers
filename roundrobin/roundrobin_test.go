package roundrobin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	t.Run("rejects negative weights", func(t *testing.T) {
		b := NewLoadBalancer()

		err := b.Register("invalid", -1)
		require.ErrorIs(t, err, ErrNegativeComponentWeight, "negative weights must be rejected")
		require.Empty(t, b.components, "a rejected component must not be registered")
	})
}

func Test_recomputeComponentBounds(t *testing.T) {
	t.Run("tracks smallest bounds", func(t *testing.T) {
		b := NewLoadBalancer()

		require.NoError(t, b.Register("small", 1), "registering the smallest component should succeed")
		require.NoError(t, b.Register("large", 19_999), "registering the largest component should succeed")

		const want = 1.0 / 20_000.0
		require.Equal(t, want, b.smallestBounds, "smallest bounds should match the smallest component's share of the total weight")
	})
}

func Test_selectionPrecision(t *testing.T) {
	t.Run("resolves smallest bounds", func(t *testing.T) {
		b := NewLoadBalancer()
		require.NoError(t, b.Register("small", 1), "registering the smallest component should succeed")
		require.NoError(t, b.Register("large", 19_999), "registering the largest component should succeed")

		require.Equal(t, int64(20_000), b.selectionPrecision(), "precision should provide at least one random value for the smallest component range")
	})

	t.Run("keeps baseline", func(t *testing.T) {
		b := NewLoadBalancer()
		require.NoError(t, b.Register("first", 1), "registering the first component should succeed")
		require.NoError(t, b.Register("second", 1), "registering the second component should succeed")

		require.Equal(t, int64(defaultSelectionPrecision), b.selectionPrecision(), "precision should not fall below the default for evenly weighted components")
	})
}
