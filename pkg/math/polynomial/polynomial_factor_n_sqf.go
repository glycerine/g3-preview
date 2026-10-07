// Source: src/math/polynomial/polynomial.cpp:6727-6735.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

// The source degree-greater-than-two helper currently preserves p as one
// factor. Its source TODO is retained; no alternative factorization is used.
func (m *MultivariatePolynomialManager) factorHigherSquareFreePrimitive(p Polynomial, r *PolynomialFactors, x, k uint32) error {
	return m.factorHigherSquareFreePrimitiveWithReceipts(p, r, x, k, nil, nil)
}
func (m *MultivariatePolynomialManager) factorHigherSquareFreePrimitiveWithReceipts(p Polynomial, r *PolynomialFactors, x, k uint32, guards *[2]polynomialGCDInvocationReceipt, phase *uint8) error {
	if e := m.ready(); e != nil {
		return e
	}
	if r == nil || r.manager != m {
		return ErrPolynomialOwner
	}
	if e := r.open(); e != nil {
		return e
	}
	if polynomialSourceDebug {
		degree, e := m.core.Degree(p, x)
		if e != nil {
			return e
		}
		if degree <= 2 {
			return ErrPolynomialAssertion
		}
		var guard *polynomialGCDInvocationReceipt
		if guards != nil {
			guard = &guards[0]
		}
		if phase != nil {
			*phase = 13
		}
		primitive, e := m.isPrimitiveWithReceipt(p, x, guard)
		if e != nil {
			return e
		}
		if !primitive {
			return ErrPolynomialAssertion
		}
		guard = nil
		if guards != nil {
			guard = &guards[1]
		}
		if phase != nil {
			*phase = 14
		}
		squareFree, e := m.isSquareFreeWithReceipt(p, x, true, guard)
		if e != nil {
			return e
		}
		if !squareFree {
			return ErrPolynomialAssertion
		}
	}
	return r.Push(p, k)
}
