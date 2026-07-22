package weightedrandom

import (
	"cmp"
	"crypto/rand"
	"fmt"
	"maps"
	"math"
	"math/big"
	"slices"
	"sync"
)

type LoadBalancer struct {
	mu             *sync.RWMutex
	components     map[string]component
	smallestBounds float64
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		mu:         new(sync.RWMutex),
		components: make(map[string]component),
	}
}

func (b *LoadBalancer) recomputeComponentBounds() {
	// Calculate the total weight for all components
	totalWeights := 0
	for _, component := range b.components {
		totalWeights += component.weight
	}

	// Sort the components from lowest to highest by weight
	sortedComponents := slices.SortedStableFunc(
		maps.Values(b.components),
		func(a, b component) int {
			return cmp.Compare(a.weight, b.weight)
		},
	)

	// Recalculate bounds for the sorted components
	// and ensure the lower bound for each component is the upper bound for the previous component.
	//
	// The lower bound for the component with the lowest weight will be zero
	// while the upper bound for the component with the highest weight will be 1.
	//
	// No component can have overlapping bounds.
	prevUpperBound := float64(0)
	b.smallestBounds = 0
	for i, component := range sortedComponents {
		component.lowerBound = prevUpperBound
		component.upperBound = prevUpperBound + (float64(component.weight) / float64(totalWeights))

		// Handle floating point errors for the last component
		if i == len(sortedComponents)-1 {
			component.upperBound = 1
		}

		b.components[component.id] = component

		// Set the smallest bounds to the upper bound of the component with the smallest weight
		// i.e smallestBounds = upperBound - 0
		if i == 0 {
			b.smallestBounds = component.upperBound
		}

		prevUpperBound = component.upperBound
	}
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

// selectionPrecision returns enough random values to select even the component
// with the smallest range, while retaining the default minimum precision.
func (b *LoadBalancer) selectionPrecision() int64 {
	precision := math.Max(defaultSelectionPrecision, math.Ceil(1/b.smallestBounds))
	if precision >= math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(precision)
}

func (b *LoadBalancer) Select() (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.components) == 0 {
		return "", ErrNoComponentsRegistered
	}

	maxInt := new(big.Int).SetInt64(b.selectionPrecision())
	randInt, err := rand.Int(rand.Reader, maxInt)
	if err != nil {
		return "", fmt.Errorf("unable to generate random number: %w", err)
	}

	// Calculate randInt/maxInt
	quotient := new(big.Float).Quo(new(big.Float).SetInt(randInt), new(big.Float).SetInt(maxInt))

	// Select the component with lowerBound <= quotient < upperBound
	for _, component := range b.components {
		lowerBound := new(big.Float).SetFloat64(component.lowerBound)
		upperBound := new(big.Float).SetFloat64(component.upperBound)
		if quotient.Cmp(lowerBound) >= 0 && quotient.Cmp(upperBound) < 0 {
			return component.id, nil
		}
	}

	return "", ErrUnableToFindComponentWithinBounds
}
