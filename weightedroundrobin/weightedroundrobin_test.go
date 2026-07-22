package weightedroundrobin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	const apiComponentID = "api"

	t.Run("registers a component", func(t *testing.T) {
		const (
			weight = 5
		)
		b := NewLoadBalancer()

		require.NoError(t, b.Register(apiComponentID, weight),
			"registering a component with a positive weight should succeed")
		require.Contains(t, b.components, apiComponentID, "the registered component should be stored")
		require.Equal(t, weight, b.components[apiComponentID].weight, "the registered component should retain its weight")
	})

	t.Run("updates an existing component weight", func(t *testing.T) {
		const (
			initialWeight = 5
		)
		b := NewLoadBalancer()
		require.NoError(t, b.Register(apiComponentID, initialWeight), "initial component registration should succeed")

		const updatedWeight = 10
		require.NoError(t, b.Register(apiComponentID, updatedWeight), "updating an existing component weight should succeed")
		require.Len(t, b.components, 1, "updating a component should not add a duplicate")
		require.Equal(t, updatedWeight, b.components[apiComponentID].weight, "the component should use its updated weight")
	})

	t.Run("rejects non-positive weights", func(t *testing.T) {
		b := NewLoadBalancer()

		for _, weight := range []int{-1, 0} {
			err := b.Register("invalid", weight)
			require.ErrorIs(t, err, ErrNonPositiveComponentWeight, "non-positive weights must be rejected")
		}
		require.Empty(t, b.components, "a rejected component must not be registered")
	})
}

func TestUnregister(t *testing.T) {
	const (
		apiComponentID    = "api"
		workerComponentID = "worker"
	)

	t.Run("removes a registered component", func(t *testing.T) {
		b := NewLoadBalancer()
		require.NoError(t, b.Register(apiComponentID, 5), "component registration should succeed")
		require.NoError(t, b.Register(workerComponentID, 10), "component registration should succeed")

		require.NoError(t, b.Unregister(apiComponentID), "unregistering an existing component should succeed")
		require.NotContains(t, b.components, apiComponentID, "the unregistered component should be removed")
		require.Contains(t, b.components, workerComponentID, "unregistering one component should retain the others")
	})

	t.Run("rejects an unknown component", func(t *testing.T) {
		b := NewLoadBalancer()

		err := b.Unregister("missing")
		require.ErrorIs(t, err, ErrComponentNotRegistered,
			"unregistering an unknown component should return the expected error")
	})
}

func TestSelect(t *testing.T) {
	const (
		apiComponentID   = "api"
		alphaComponentID = "alpha"
		bravoComponentID = "bravo"
	)

	t.Run("returns an error when no components are registered", func(t *testing.T) {
		b := NewLoadBalancer()

		_, err := b.Select()
		require.ErrorIs(t, err, ErrNoComponentsRegistered,
			"selecting from an empty load balancer should return the expected error")
	})

	t.Run("returns a registered component", func(t *testing.T) {
		b := NewLoadBalancer()
		require.NoError(t, b.Register(apiComponentID, 1), "component registration should succeed")

		selectedComponentID, err := b.Select()
		require.NoError(t, err, "selecting from a load balancer with a component should succeed")
		require.Equal(t, apiComponentID, selectedComponentID, "the only registered component should always be selected")
	})

	t.Run("selects components according to their weights", func(t *testing.T) {
		// Register the components
		type weightedComponent struct {
			id     string
			weight int
		}
		components := []weightedComponent{
			{id: "one", weight: 1},
			{id: "two_1", weight: 2},
			{id: "two_2", weight: 2},
			{id: "four", weight: 4},
		}

		b := NewLoadBalancer()

		totalWeight := 0
		for _, component := range components {
			require.NoError(t, b.Register(component.id, component.weight),
				"registering component %q should succeed", component.id)
			totalWeight += component.weight
		}

		// Simulate 900 selections.
		selectionCounts := make(map[string]int, len(components))

		const selectionCount = 900
		for range selectionCount {
			selectedComponentID, err := b.Select()
			require.NoError(t, err, "each selection should succeed")

			selectionCounts[selectedComponentID]++
		}

		// Confirm that the actual percentage of selections per component equals to the expected percentage of selections.
		for _, component := range components {
			require.Equalf(t, selectionCount*component.weight/totalWeight, selectionCounts[component.id],
				"component %q should be selected in exact proportion to its weight", component.id)
		}
	})

	t.Run("orders equally weighted components by ID", func(t *testing.T) {
		const charlieComponentID = "charlie"

		b := NewLoadBalancer()
		for _, componentID := range []string{charlieComponentID, alphaComponentID, bravoComponentID} {
			require.NoError(t, b.Register(componentID, 1), "registering component %q should succeed", componentID)
		}

		for _, wantComponentID := range []string{alphaComponentID, bravoComponentID, charlieComponentID} {
			selectedComponentID, err := b.Select()
			require.NoError(t, err, "each selection should succeed")
			require.Equal(t, wantComponentID, selectedComponentID, "equal-weight components should be selected by ID")
		}
	})
}

func Test_recomputeComponentBounds(t *testing.T) {
	const (
		alphaComponentID = "alpha"
		bravoComponentID = "bravo"
	)

	t.Run("resets the cycle after component changes", func(t *testing.T) {
		b := NewLoadBalancer()
		require.NoError(t, b.Register(alphaComponentID, 1), "component registration should succeed")
		require.NoError(t, b.Register(bravoComponentID, 1), "component registration should succeed")

		selectedComponentID, err := b.Select()
		require.NoError(t, err, "selecting before an update should succeed")
		require.NoError(t, b.Register(bravoComponentID, 2), "updating a component should succeed")
		require.Equal(t, alphaComponentID, selectedComponentID, "should select the first component")

		selectedComponentID, err = b.Select()
		require.NoError(t, err, "selecting after an update should succeed")
		require.Equal(t, alphaComponentID, selectedComponentID,
			"updating components should restart the selection cycle")
	})
}
