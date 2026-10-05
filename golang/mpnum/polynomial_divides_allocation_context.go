// Source: src/math/polynomial/polynomial.cpp:5154-5194;
// src/util/mpzzp.h:131,192,195-208; src/util/mpz.h:468-485.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *IntegerManager) dividesAllocationRoots(r *polynomialDividesInvocationReceipt) *polynomialNumericRoots {
	if m == nil || m.self != m || r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesDividesInvocation(r) || r.frame.constructed < 2 || !r.frame.executing || r.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerManager) dividesContextRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	if c.divides == nil || c.dividesInvocation == 0 || c.dividesInvocation != c.divides.invocation {
		return nil
	}
	return m.dividesAllocationRoots(c.divides)
}
func (m *IntegerModularManager) dividesAllocationContext(r *polynomialDividesInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.dividesAllocationRoots(r) == nil || r.core.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{divides: r, dividesInvocation: r.invocation}
}
func (m *IntegerModularManager) divWithDividesReceipt(a, b, out *Int, r *polynomialDividesInvocationReceipt) error {
	context := m.dividesAllocationContext(r)
	e := m.divWithAllocationContext(a, b, out, context)
	if m != nil && m.manager != nil && m.manager.dividesContextRoots(context) != nil && r.frame.sourceCause == nil {
		r.frame.sourceCause = m.manager.dividesNumericSourceCause(r)
	}
	return e
}
func (m *IntegerModularManager) negWithDividesReceipt(out *Int, r *polynomialDividesInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.dividesAllocationContext(r)
	if e := m.manager.negWithAllocationContext(out, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}

// Bind only the child returned by the actual guarded RAW declaration action.
func (m *IntegerManager) bindDividesRaw(c integerAllocationInvocationContext, child *integerRawInvocationReceipt, division bool) {
	if m.dividesContextRoots(c) == nil || !m.matchesRawInvocation(child) {
		return
	}
	child.frame.divides = *c.divides
	if division {
		c.divides.frame.divisionRaw = *child
	} else {
		c.divides.frame.extendedRaw = *child
	}
}

// History is read independently of the allocation/executing predicate.
func (m *IntegerManager) dividesNumericSourceCause(r *polynomialDividesInvocationReceipt) error {
	if r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesDividesInvocationHistory(r) {
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
func (m *IntegerManager) cleanupDividesNumeric(r *polynomialDividesInvocationReceipt, abort bool) error {
	if r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesDividesInvocationHistory(r) {
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

func (m *IntegerModularManager) dividesWithDividesReceipt(a, b *Int, r *polynomialDividesInvocationReceipt) (bool, error) {
	context := m.dividesAllocationContext(r)
	result, e := m.dividesWithAllocationContext(a, b, context)
	if m != nil && m.manager != nil && m.manager.dividesContextRoots(context) != nil && r.frame.sourceCause == nil {
		r.frame.sourceCause = m.manager.dividesNumericSourceCause(r)
	}
	return result, e
}
func (m *IntegerModularManager) dividesWithAllocationContext(a, b *Int, c integerAllocationInvocationContext) (bool, error) {
	f, e := m.Field()
	if e != nil {
		return false, e
	}
	if f {
		zero, e := m.IsZero(a)
		if e != nil {
			return false, e
		}
		if !zero {
			return true, nil
		}
	}
	return m.manager.dividesWithAllocationContext(a, b, c)
}
func (m *IntegerManager) dividesNumericPending(r *polynomialDividesInvocationReceipt) bool {
	if r == nil || r.core == nil || r.core.integer != m || !r.core.matchesDividesInvocationHistory(r) {
		return false
	}
	s := r.frame
	if polynomialNegNumericPending(&s.numeric) || m.natural.matchesPNVerification(&s.verification) {
		return true
	}
	if m.matchesDividesNumeralInvocation(&s.boolNumeral) {
		b := s.boolNumeral.frame
		if b.pending(m) || b.constructed && !b.ended && !b.owned {
			return true
		}
	}
	if m.matchesRawInvocation(&s.extendedRaw) || m.matchesRawInvocation(&s.divisionRaw) {
		return true
	}
	if r.core.modular.matchesPNormalizeInvocation(&s.normalize) {
		return true
	}
	for i := range s.buffers {
		if s.buffers[i].pending != (naturalPNBufferReceipt{}).pending {
			return true
		}
	}
	return false
}
