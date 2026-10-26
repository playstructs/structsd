package types

// SubstationUsableCapacity returns floor(90% of aggregate incoming power).
// Divide before multiplying so every uint64 input is safe. Apply once to the
// total, not per incoming allocation, so splitting cannot change the loss.
func SubstationUsableCapacity(incoming uint64) uint64 {
	return incoming/10*9 + incoming%10*9/10
}
