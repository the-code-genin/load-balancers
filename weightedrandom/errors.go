package weightedrandom

import "errors"

var (
	ErrComponentNotRegistered            = errors.New("component not registered")
	ErrNonPositiveComponentWeight        = errors.New("component weight must be positive")
	ErrUnableToFindComponentWithinBounds = errors.New("unable to find component within bounds")
	ErrNoComponentsRegistered            = errors.New("no components registered")
)
