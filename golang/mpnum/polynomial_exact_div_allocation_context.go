// Source: src/math/polynomial/polynomial.cpp:5114-5156;
// src/util/mpzzp.h:167-185; src/util/mpz.h:468-485.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *IntegerManager) exactDivAllocationRoots(r *polynomialExactDivInvocationReceipt) *polynomialNumericRoots {
	if m == nil || m.self != m || r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesExactDivInvocation(r) || !r.frame.executing || r.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerManager) exactDivContextRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	if c.exactDiv == nil || c.exactDivInvocation == 0 || c.exactDivInvocation != c.exactDiv.invocation {
		return nil
	}
	return m.exactDivAllocationRoots(c.exactDiv)
}
func (m *IntegerModularManager) exactDivAllocationContext(r *polynomialExactDivInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.exactDivAllocationRoots(r) == nil || r.core.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{exactDiv: r, exactDivInvocation: r.invocation}
}
func (m *IntegerModularManager) divWithExactDivReceipt(a, b, out *Int, r *polynomialExactDivInvocationReceipt) error {
	context := m.exactDivAllocationContext(r)
	e := m.divWithAllocationContext(a, b, out, context)
	if m != nil && m.manager != nil && m.manager.exactDivContextRoots(context) != nil && r.frame.sourceCause == nil {
		r.frame.sourceCause = m.manager.exactDivNumericSourceCause(r)
	}
	return e
}
func (m *IntegerModularManager) negWithExactDivReceipt(out *Int, r *polynomialExactDivInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.exactDivAllocationContext(r)
	if e := m.manager.negWithAllocationContext(out, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}

// Bind only the child returned by the actual guarded RAW declaration action.
func (m *IntegerManager) bindExactDivRaw(c integerAllocationInvocationContext, child *integerRawInvocationReceipt, division bool) {
	if m.exactDivContextRoots(c) == nil || !m.matchesRawInvocation(child) {
		return
	}
	child.frame.exactDiv = *c.exactDiv
	if division {
		c.exactDiv.frame.divisionRaw = *child
	} else {
		c.exactDiv.frame.extendedRaw = *child
	}
}

// History is read independently of the allocation/executing predicate.
func (m *IntegerManager) exactDivNumericSourceCause(r *polynomialExactDivInvocationReceipt) error {
	if r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesExactDivInvocation(r) {
		return nil
	}
	s := r.frame
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
func (m *IntegerManager) cleanupExactDivNumeric(r *polynomialExactDivInvocationReceipt, abort bool) error {
	if r == nil || r.core == nil || r.core.self != r.core || r.core.integer != m || !r.core.matchesExactDivInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
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
