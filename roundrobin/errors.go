package roundrobin

import "errors"

var (
	ErrComponentNotRegistered            = errors.New("component not registered")
	ErrUnableToFindComponentWithinBounds = errors.New("unable to find component within bounds")
)
