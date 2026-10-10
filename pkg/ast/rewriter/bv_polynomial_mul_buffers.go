// Source: src/ast/rewriter/poly_rewriter_def.h:334-352; src/util/util.cpp:112-119.
package rewriter

// First coordinate advances fastest, with original unsigned wrap and carry.
func bvPolynomialProductNext(sz, it []uint32) bool {
	for i := range sz {
		it[i]++
		if it[i] < sz[i] {
			return true
		}
		it[i] = 0
	}
	return false
}
