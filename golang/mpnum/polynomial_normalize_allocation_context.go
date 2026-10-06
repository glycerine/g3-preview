// Source: src/math/polynomial/polynomial.cpp:5904-5940;
// src/util/mpzzp.h:167-185,211; src/util/mpz.cpp:730-817.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (m *MultivariatePolynomialManager) hasNormalizeInvocationHistory(r *polynomialNormalizeInvocationReceipt) bool {
	return m != nil && m.self == m && m.core != nil && m.core.self == m.core && r != nil && r.manager == m && r.frame == &m.normalization && r.invocation != 0 && r.invocation == r.frame.invocation
}
func (m *IntegerManager) normalizationAllocationRoots(r *polynomialNormalizeInvocationReceipt) *polynomialNumericRoots {
	if m == nil || m.self != m || r == nil || r.manager == nil || !r.manager.hasNormalizeInvocationHistory(r) || r.manager.core.integer != m || !r.frame.active || !r.frame.executing || r.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (c integerAllocationInvocationContext) normalizationReceipt(m *IntegerManager) *polynomialNormalizeInvocationReceipt {
	if c.normalization == nil || c.normalizationInvocation == 0 || c.normalizationInvocation != c.normalization.invocation || m.normalizationAllocationRoots(c.normalization) == nil {
		return nil
	}
	return c.normalization
}
func (m *IntegerModularManager) normalizationAllocationContext(r *polynomialNormalizeInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.normalizationAllocationRoots(r) == nil || r.manager.core.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{normalization: r, normalizationInvocation: r.invocation}
}
func (m *IntegerModularManager) copyWithNormalizeReceipt(out, value *Int, r *polynomialNormalizeInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := m.normalizationAllocationContext(r)
	if e := m.manager.copyFromOwnerWithAllocationContext(out, m.manager, value, nil, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(out, context)
}
func (m *IntegerModularManager) gcdManyIndexedWithNormalizeReceipt(as integerArrayView, out *Int, r *polynomialNormalizeInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.gcdManyIndexedWithAllocationContext(as, out, m.normalizationAllocationContext(r)); e != nil {
		return e
	}
	return m.assertNormalized(out)
}
func (m *IntegerModularManager) divWithNormalizeReceipt(a, b, out *Int, r *polynomialNormalizeInvocationReceipt) error {
	return m.divWithAllocationContext(a, b, out, m.normalizationAllocationContext(r))
}
func (c integerAllocationInvocationContext) recordNormalizationRaw(m *IntegerManager, r *integerRawInvocationReceipt) {
	p := c.normalizationReceipt(m)
	if p == nil || !m.matchesRawInvocation(r) {
		return
	}
	r.frame.normalization = *p
	switch r.frame {
	case &m.gcdRaw:
		p.frame.gcdRaw = *r
	case &m.divisionRaw:
		p.frame.divisionRaw = *r
	case &m.extendedRaw:
		p.frame.extendedRaw = *r
	}
}
func (m *IntegerManager) normalizationNumericSourceCause(r *polynomialNormalizeInvocationReceipt) error {
	if r == nil || r.manager == nil || !r.manager.hasNormalizeInvocationHistory(r) || r.manager.core.integer != m {
		return nil
	}
	s := r.frame
	if e := r.manager.core.modular.pNormalizeInvocationSourceCause(&s.normalize); e != nil {
		return e
	}
	if e := m.rawInvocationSourceCause(&s.extendedRaw); e != nil {
		return e
	}
	if e := m.rawInvocationSourceCause(&s.divisionRaw); e != nil {
		return e
	}
	if e := m.rawInvocationSourceCause(&s.gcdRaw); e != nil {
		return e
	}
	return m.natural.pnVerificationSourceCause(&s.verification)
}
func (m *IntegerManager) cleanupNormalizationNumeric(r *polynomialNormalizeInvocationReceipt, abort bool) error {
	if r == nil || r.manager == nil || !r.manager.hasNormalizeInvocationHistory(r) || r.manager.core.integer != m {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := r.manager.core.modular.cleanupPNormalizeInvocation(&s.normalize, abort); e != nil {
		return e
	}
	if e := m.cleanupRawInvocation(&s.extendedRaw, abort); e != nil {
		return e
	}
	if e := m.cleanupRawInvocation(&s.divisionRaw, abort); e != nil {
		return e
	}
	if e := m.cleanupRawInvocation(&s.gcdRaw, abort); e != nil {
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

// Exact captured child identity is a refusal guard for owner destruction only;
// numerical acquisition remains exclusively the explicit passed context.
func (m *MultivariatePolynomialCore) normalizationPending() bool {
	if m.hasSOMNormalizeInvocationHistory(&m.somNormalizationOwner) && (m.somNormalizationOwner.frame.active || m.somNormalizationOwner.frame.finishing) {
		return true
	}
	r := &m.normalizationOwner
	if r.manager != nil && r.manager.hasNormalizeInvocationHistory(r) && r.manager.core == m && r.frame.active {
		return true
	}
	for _, b := range [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		if m.hasSOMNormalizeInvocationHistory(&b.normalizeScope.receipt) && b.normalizeScope.active {
			return true
		}
	}
	return false
}
