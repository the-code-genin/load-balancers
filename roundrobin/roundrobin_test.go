package roundrobin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	t.Run("registers a component", func(t *testing.T) {
		const (
			componentID = "api"
			weight      = 5
		)
		b := NewLoadBalancer()

		require.NoError(t, b.Register(componentID, weight), "registering a component with a positive weight should succeed")
		require.Contains(t, b.components, componentID, "the registered component should be stored")
		require.Equal(t, weight, b.components[componentID].weight, "the registered component should retain its weight")
	})

	t.Run("updates an existing component weight", func(t *testing.T) {
		const (
			componentID   = "api"
			initialWeight = 5
		)
		b := NewLoadBalancer()
		require.NoError(t, b.Register(componentID, initialWeight), "initial component registration should succeed")

		const updatedWeight = 10
		require.NoError(t, b.Register(componentID, updatedWeight), "updating an existing component weight should succeed")
		require.Len(t, b.components, 1, "updating a component should not add a duplicate")
		require.Equal(t, updatedWeight, b.components[componentID].weight, "the component should use its updated weight")
	})

	t.Run("rejects negative weights", func(t *testing.T) {
		b := NewLoadBalancer()

		err := b.Register("invalid", -1)
		require.ErrorIs(t, err, ErrNegativeComponentWeight, "negative weights must be rejected")
		require.Empty(t, b.components, "a rejected component must not be registered")
	})
}

func TestUnregister(t *testing.T) {
	t.Run("removes a registered component", func(t *testing.T) {
		const (
			removedComponentID  = "api"
			retainedComponentID = "worker"
		)
		b := NewLoadBalancer()
		require.NoError(t, b.Register(removedComponentID, 5), "component registration should succeed")
		require.NoError(t, b.Register(retainedComponentID, 10), "component registration should succeed")

		require.NoError(t, b.Unregister(removedComponentID), "unregistering an existing component should succeed")
		require.NotContains(t, b.components, removedComponentID, "the unregistered component should be removed")
		require.Contains(t, b.components, retainedComponentID, "unregistering one component should retain the others")
	})

	t.Run("rejects an unknown component", func(t *testing.T) {
		b := NewLoadBalancer()

		err := b.Unregister("missing")
		require.ErrorIs(t,
			err, ErrComponentNotRegistered,
			"unregistering an unknown component should return the expected error",
		)
	})
}

func Test_recomputeComponentBounds(t *testing.T) {
	t.Run("tracks smallest bounds", func(t *testing.T) {
		const (
			smallWeight = 1
			largeWeight = 19_999
		)
		b := NewLoadBalancer()

		require.NoError(t, b.Register("small", smallWeight), "registering the smallest component should succeed")
		require.NoError(t, b.Register("large", largeWeight), "registering the largest component should succeed")

		require.Equal(t,
			float64(smallWeight)/float64(smallWeight+largeWeight), b.smallestBounds,
			"smallest bounds should match the smallest component's share of the total weight",
		)
	})
}

func Test_selectionPrecision(t *testing.T) {
	t.Run("resolves smallest bounds", func(t *testing.T) {
		const (
			smallWeight = 1
			largeWeight = 19_999
		)
		b := NewLoadBalancer()
		require.NoError(t, b.Register("small", smallWeight), "registering the smallest component should succeed")
		require.NoError(t, b.Register("large", largeWeight), "registering the largest component should succeed")

		require.Equal(t,
			int64(smallWeight+largeWeight),
			b.selectionPrecision(),
			"precision should provide at least one random value for the smallest component range",
		)
	})

	t.Run("keeps baseline", func(t *testing.T) {
		const componentWeight = 1
		b := NewLoadBalancer()
		require.NoError(t, b.Register("first", componentWeight), "registering the first component should succeed")
		require.NoError(t, b.Register("second", componentWeight), "registering the second component should succeed")

		require.Equal(t,
			int64(defaultSelectionPrecision), b.selectionPrecision(),
			"precision should not fall below the default for evenly weighted components",
		)
	})
}
