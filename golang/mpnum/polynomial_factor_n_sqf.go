// Source: src/math/polynomial/polynomial.cpp:6727-6735.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// The source degree-greater-than-two helper currently preserves p as one
// factor. Its source TODO is retained; no alternative factorization is used.
func (m *MultivariatePolynomialManager) factorHigherSquareFreePrimitive(p Polynomial, r *PolynomialFactors, x, k uint32) error {
	if e := m.ready(); e != nil {
		return e
	}
	if r == nil || r.manager != m {
		return ErrPolynomialOwner
	}
	if e := r.open(); e != nil {
		return e
	}
	if modularSourceDebug {
		degree, e := m.core.Degree(p, x)
		if e != nil {
			return e
		}
		if degree <= 2 {
			return ErrPolynomialAssertion
		}
		primitive, e := m.IsPrimitive(p, x)
		if e != nil {
			return e
		}
		if !primitive {
			return ErrPolynomialAssertion
		}
		squareFree, e := m.IsSquareFreeVariable(p, x)
		if e != nil {
			return e
		}
		if !squareFree {
			return ErrPolynomialAssertion
		}
	}
	return r.Push(p, k)
}
