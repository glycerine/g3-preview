// Source: src/math/polynomial/polynomial.cpp:3109-3182; src/util/mpz.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import ()

func (s *PolynomialSparseInterpolator) numericBefore() (polynomialNumericRoots, error) {
	return polynomialNumericBefore(s.owner.core.integer)
}
func (s *PolynomialSparseInterpolator) numericAfter(before polynomialNumericRoots) error {
	return s.numeric.after(s.owner.core.integer, before)
}
func (s *PolynomialSparseInterpolator) cleanupNumeric() error {
	return s.numeric.cleanup(s.owner.core.integer)
}
