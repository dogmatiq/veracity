package filter

// Slice removes all elements from the slice for which pred() returns false.
//
// It modifies the existing slice in-place.
func Slice[E any, S ~[]E](
	slice S,
	pred func(E) bool,
) S {
	filtered := make(S, 0, len(slice))

	for _, e := range slice {
		if pred(e) {
			filtered = append(filtered, e)
		}
	}

	return filtered
}
