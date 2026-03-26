package utils

// GetMapValue get map value with type check
func GetMapValue[K comparable, V any](m map[K]any, key K) (V, bool) {
	var zero V

	if m == nil {
		return zero, false
	}
	val, ok := m[key]
	if !ok {
		return zero, false
	}

	if val == nil {
		return zero, false
	}

	typedVal, ok := val.(V)
	if !ok {
		return zero, false
	}
	return typedVal, true
}
