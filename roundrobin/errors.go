package roundrobin

import "errors"

var (
	ErrComponentNotRegistered            = errors.New("component not registered")
	ErrNegativeComponentWeight           = errors.New("component weight cannot be negative")
	ErrUnableToFindComponentWithinBounds = errors.New("unable to find component within bounds")
	ErrNoComponentsRegistered            = errors.New("no components registered")
)
