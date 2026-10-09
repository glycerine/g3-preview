// Source: src/util/vector.h:467-487.
package util

// VectorGrowth returns the source unsigned-vector capacity and byte extent.
// It remains here for Runtime-backed vector ports that still need the exact
// C++ growth arithmetic.
func VectorGrowth(old uint32) (capacity, bytes uint32, err error) {
	if old == 0 {
		return 2, 16, nil
	}
	n := (3*old + 1) >> 1
	nb := 4*n + 8
	ob := 4*old + 8
	if n <= old || nb <= ob {
		return 0, 0, Failure{Kind: FailureVectorOverflow}
	}
	return n, nb, nil
}
