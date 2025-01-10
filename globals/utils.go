package common_globals

// DeleteIndex removes a value from a slice with the given index
func DeleteIndex(s []uint32, index int) []uint32 {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

// MoveToFront moves a value in a slice with a given index to the front of the slice
func MoveToFront(s []uint32, index int) []uint32 {
	if index <= 0 {
		return s
	}

	// * Swap 'em
	s[0], s[index] = s[index], s[0]
	return s
}
