package utils

// Lazy is a value type that evaluates only when needed.
type Lazy[T any] func() T

// If returns onTrue when cond is true, otherwise returns onFalse.
func If[T any](cond bool, onTrue, onFalse T) T {
	if cond {
		return onTrue
	}
	return onFalse
}

// IfLazy is a variant of [If], accepts [Lazy] values.
func IfLazy[T any](cond bool, onTrue, onFalse Lazy[T]) T {
	if cond {
		return onTrue()
	}
	return onFalse()
}

// IfLazyL is a variant of [If], accepts [Lazy] onTrue value.
func IfLazyL[T any](cond bool, onTrue Lazy[T], onFalse T) T {
	if cond {
		return onTrue()
	}
	return onFalse
}

// IfLazyR is a variant of [If], accepts [Lazy] onFalse value.
func IfLazyR[T any](cond bool, onTrue T, onFalse Lazy[T]) T {
	if cond {
		return onTrue
	}
	return onFalse()
}
