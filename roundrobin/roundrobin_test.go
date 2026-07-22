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

func TestSelect(t *testing.T) {
	t.Run("returns an error when no components are registered", func(t *testing.T) {
		b := NewLoadBalancer()

		_, err := b.Select()
		require.ErrorIs(t, err, ErrNoComponentsRegistered,
			"selecting from an empty load balancer should return the expected error")
	})

	t.Run("returns a registered component", func(t *testing.T) {
		const componentID = "api"

		b := NewLoadBalancer()
		require.NoError(t, b.Register(componentID, 1), "component registration should succeed")

		selectedComponentID, err := b.Select()
		require.NoError(t, err, "selecting from a load balancer with a component should succeed")
		require.Equal(t, componentID, selectedComponentID, "the only registered component should always be selected")
	})

	t.Run("selects components according to their weights", func(t *testing.T) {
		// Register the components
		type weightedComponent struct {
			id     string
			weight int
		}
		components := []weightedComponent{
			{id: "one", weight: 5},
			{id: "two", weight: 10},
			{id: "three", weight: 15},
			{id: "four", weight: 20},
			{id: "five", weight: 50},
		}

		selectionCounts := make(map[string]int, len(components))
		totalWeight := 0

		b := NewLoadBalancer()

		for _, component := range components {
			require.NoError(t, b.Register(component.id, component.weight),
				"registering component %q should succeed", component.id)

			selectionCounts[component.id] = 0
			totalWeight += component.weight
		}

		// Simulate 1m selections
		const selectionCount = 1_000_000
		for range selectionCount {
			selectedComponentID, err := b.Select()
			require.NoError(t, err, "each selection should succeed")

			if _, ok := selectionCounts[selectedComponentID]; !ok {
				t.Fatalf("Select() returned unregistered component %q", selectedComponentID)
			}
			selectionCounts[selectedComponentID]++
		}

		// Confirm that the actual percentage of selections per component approaches the expected percentage of selections.
		// This assumption is based off of the central limit theorem
		for _, component := range components {
			expectedPercentage := float64(component.weight) / float64(totalWeight)
			actualPercentage := float64(selectionCounts[component.id]) / selectionCount

			require.InDeltaf(t, expectedPercentage, actualPercentage, 0.0025,
				"component %q was selected %d times; expected approximately %.2f%% of selections",
				component.id, selectionCounts[component.id], expectedPercentage*100)
		}
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
