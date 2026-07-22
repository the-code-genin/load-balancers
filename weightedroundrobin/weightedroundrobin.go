package weightedroundrobin

import (
	"cmp"
	"maps"
	"slices"
	"sync"
)

type LoadBalancer struct {
	mu           *sync.Mutex
	components   map[string]component
	counter      int
	totalWeights int
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		mu:         new(sync.Mutex),
		components: make(map[string]component),
	}
}

func (b *LoadBalancer) recomputeComponentBounds() {
	// Sort components by weight, then ID, to keep equal-weight ordering stable.
	sortedComponents := slices.SortedStableFunc(
		maps.Values(b.components),
		func(a, b component) int {
			if weightComparison := cmp.Compare(a.weight, b.weight); weightComparison != 0 {
				return weightComparison
			}

			return cmp.Compare(a.id, b.id)
		},
	)

	// Recalculate bounds for the sorted components
	// and ensure the lower bound for each component is the upper bound for the previous component.
	//
	// The lower bound for the component with the lowest weight will be zero
	// while the upper bound for the component with the highest weight will be b.totalWeights.
	//
	// No component can have overlapping bounds.
	prevUpperBound := 0
	b.totalWeights = 0
	for _, component := range sortedComponents {
		component.lowerBound = prevUpperBound
		component.upperBound = prevUpperBound + component.weight

		b.components[component.id] = component
		b.totalWeights += component.weight

		prevUpperBound = component.upperBound
	}

	// Reset the counter to zero
	b.counter = 0
}

func (b *LoadBalancer) Register(id string, weight int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if weight <= 0 {
		return ErrNonPositiveComponentWeight
	}

	// Update the component weight in the components list
	c, ok := b.components[id]
	if !ok { // New component
		c = component{id: id}
	}
	c.weight = weight
	b.components[id] = c

	b.recomputeComponentBounds()

	return nil
}

func (b *LoadBalancer) Unregister(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove the component from the components list
	c, ok := b.components[id]
	if !ok { // Component not registered
		return ErrComponentNotRegistered
	}
	delete(b.components, c.id)

	b.recomputeComponentBounds()

	return nil
}

func (b *LoadBalancer) Select() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.components) == 0 {
		return "", ErrNoComponentsRegistered
	}

	// Ensure 0 <= b.counter < b.totalWeights
	if b.counter < 0 || b.counter >= b.totalWeights {
		b.counter = 0
	}

	// Select the component with lowerBound <= b.counter < upperBound
	// Increment the counter after finding the matching component for the next call to Select()
	for _, component := range b.components {
		if b.counter >= component.lowerBound && b.counter < component.upperBound {
			b.counter++
			return component.id, nil
		}
	}

	return "", ErrUnableToFindComponentWithinBounds
}
