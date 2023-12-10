package common_globals

import "github.com/PretendoNetwork/nex-go"

// DeleteIndex removes a value from a slice with the given index
func DeleteIndex(s []uint32, index int) []uint32 {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

// ContainsPID reports whether a PID is present in s
func ContainsPID(s []*nex.PID, v *nex.PID) bool {
	for _, pid := range s {
		if pid.Equals(v) {
			return true
		}
	}

	return false
}
