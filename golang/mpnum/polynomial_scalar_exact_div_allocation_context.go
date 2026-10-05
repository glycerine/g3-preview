// Source: src/math/polynomial/polynomial.cpp:5094-5114;
// src/util/mpzzp.h:167-185; src/util/mpz.h:468-485.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *IntegerManager) scalarExactAllocationRoots(r *polynomialScalarExactDivInvocationReceipt) *polynomialNumericRoots {
	if m == nil || m.self != m || r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesScalarExactDivInvocation(r) || !r.frame.executing || r.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerManager) scalarExactContextRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	if c.scalarExact == nil || c.scalarExactInvocation == 0 || c.scalarExactInvocation != c.scalarExact.invocation {
		return nil
	}
	return m.scalarExactAllocationRoots(c.scalarExact)
}
func (m *IntegerModularManager) scalarExactAllocationContext(r *polynomialScalarExactDivInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.scalarExactAllocationRoots(r) == nil || r.core.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{scalarExact: r, scalarExactInvocation: r.invocation}
}
func (m *IntegerModularManager) divWithScalarExactDivReceipt(a, b, out *Int, r *polynomialScalarExactDivInvocationReceipt) error {
	context := m.scalarExactAllocationContext(r)
	e := m.divWithAllocationContext(a, b, out, context)
	if m != nil && m.manager != nil && m.manager.scalarExactContextRoots(context) != nil && r.frame.sourceCause == nil {
		r.frame.sourceCause = m.manager.scalarExactNumericSourceCause(r)
	}
	return e
}

// Bind only the child returned by the actual guarded RAW declaration action.
func (m *IntegerManager) bindScalarExactDivRaw(c integerAllocationInvocationContext, child *integerRawInvocationReceipt, division bool) {
	if m.scalarExactContextRoots(c) == nil || !m.matchesRawInvocation(child) {
		return
	}
	child.frame.scalarExact = *c.scalarExact
	if division {
		c.scalarExact.frame.divisionRaw = *child
	} else {
		c.scalarExact.frame.extendedRaw = *child
	}
}

// History is read independently of the allocation/executing predicate.
func (m *IntegerManager) scalarExactNumericSourceCause(r *polynomialScalarExactDivInvocationReceipt) error {
	if r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesScalarExactDivHistory(r) {
		return nil
	}
	s := r.frame
	if e := m.dividesNumeralInvocationSourceCause(&s.boolNumeral); e != nil {
		return e
	}
	if e := r.core.modular.pNormalizeInvocationSourceCause(&s.normalize); e != nil {
		return e
	}
	if e := m.rawInvocationSourceCause(&s.extendedRaw); e != nil {
		return e
	}
	if e := m.rawInvocationSourceCause(&s.divisionRaw); e != nil {
		return e
	}
	return m.natural.pnVerificationSourceCause(&s.verification)
}

// False completes reached younger source children. True drains unjoined
// acquisitions only when the caller has completed its older lexical suffix.
func (m *IntegerManager) cleanupScalarExactDivNumeric(r *polynomialScalarExactDivInvocationReceipt, abort bool) error {
	if r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesScalarExactDivInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := m.cleanupDividesNumeralInvocation(&s.boolNumeral, abort); e != nil {
		return e
	}
	if e := r.core.modular.cleanupPNormalizeInvocation(&s.normalize, abort); e != nil {
		return e
	}
	if e := m.cleanupRawInvocation(&s.extendedRaw, abort); e != nil {
		return e
	}
	if e := m.cleanupRawInvocation(&s.divisionRaw, abort); e != nil {
		return e
	}
	if e := m.natural.cleanupPNVerification(&s.verification, abort); e != nil {
		return e
	}
	if abort {
		for i := len(s.buffers) - 1; i >= 0; i-- {
			if e := s.buffers[i].cleanup(); e != nil {
				return e
			}
		}
		return s.numeric.cleanup(m)
	}
	return nil
}

// The modular engine erases the integer scoped NUM in the actual nonzero-field
// branch; otherwise the same scoped bool method receives this exact caller.
func (m *IntegerModularManager) dividesWithScalarExactDivReceipt(a, b *Int, r *polynomialScalarExactDivInvocationReceipt) (bool, error) {
	result, e := m.dividesWithAllocationContext(a, b, m.scalarExactAllocationContext(r))
	if m != nil && m.manager != nil && m.manager.scalarExactAllocationRoots(r) != nil && r.frame.sourceCause == nil {
		r.frame.sourceCause = m.manager.dividesNumeralInvocationSourceCause(&r.frame.boolNumeral)
	}
	return result, e
}
func (m *MultivariatePolynomialCore) scalarExactDirectPending(s *polynomialExactDivisionScope) bool {
	return m.integer.matchesDividesNumeralInvocation(&s.boolNumeral) || m.exactDivDirectPending(s)
}
