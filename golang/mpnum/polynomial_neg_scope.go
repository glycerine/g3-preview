// Source: src/math/polynomial/polynomial.cpp:5940-5951;
// src/util/scoped_numeral.h. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

// Embedded in the actual core: these are the existing Neg NUM and result,
// together with exact owning continuation metadata for that lexical scope.
type polynomialNegScope struct {
	negativeCoefficient                     Int
	negativeResult                          Polynomial
	negativeActive, negativeEnded           bool
	negativeBodyCause, negativeCleanupCause error
	invocation                              uint64
	numeric                                 polynomialNumericRoots
}
type polynomialNegInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialNegScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesNegInvocation(r *polynomialNegInvocationReceipt) bool {
	return r != nil && r.manager == m && r.frame == &m.polynomialNegScope && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.negativeActive
}
func (m *MultivariatePolynomialCore) negInvocationSourceCause(r *polynomialNegInvocationReceipt) error {
	if m.matchesNegInvocation(r) {
		return r.frame.negativeCleanupCause
	}
	return nil
}
func (m *MultivariatePolynomialCore) finishNegSource() error {
	s := &m.polynomialNegScope
	if !s.negativeActive || s.negativeEnded {
		return nil
	}
	if s.negativeCleanupCause != nil {
		return s.negativeCleanupCause
	}
	_, e := sourceNumeralDel(m.integer, &s.negativeCoefficient)
	if e != nil {
		s.negativeCleanupCause = e
		return e
	}
	s.negativeEnded = true
	// A returned factory is exposed to the caller only after this source DT.
	s.negativeResult = Polynomial{}
	if !polynomialNegNumericPending(&s.numeric) {
		s.negativeActive = false
	}
	return nil
}
func polynomialNegNumericPending(r *polynomialNumericRoots) bool {
	if r.cell.object.OwnerID() != 0 {
		return true
	}
	for _, p := range r.allocator {
		if p.OwnerID() != 0 {
			return true
		}
	}
	return false
}
func (m *MultivariatePolynomialCore) cleanupNegInvocation(r *polynomialNegInvocationReceipt, abort bool) error {
	if !m.matchesNegInvocation(r) {
		return nil
	}
	return m.cleanupNegPhase(abort)
}
func (m *MultivariatePolynomialCore) cleanupNegPhase(abort bool) error {
	s := &m.polynomialNegScope
	if !s.negativeActive {
		return nil
	}
	if !s.negativeEnded {
		if e := m.deleteNumeral(&s.negativeCoefficient); e != nil {
			return e
		}
		s.negativeEnded = true
		s.negativeCleanupCause = nil
	}
	if abort {
		if s.negativeResult != (Polynomial{}) {
			if e := m.CleanupValue(&s.negativeResult); e != nil {
				return e
			}
		}
		if e := s.numeric.cleanup(m.integer); e != nil {
			return e
		}
	}
	if s.negativeResult == (Polynomial{}) && !polynomialNegNumericPending(&s.numeric) {
		invocation := s.invocation
		*s = polynomialNegScope{invocation: invocation}
	}
	return nil
}
