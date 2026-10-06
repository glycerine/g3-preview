// Source: src/math/polynomial/polynomial.cpp:1863-1876;
// src/util/mpzzp.h:167-185,211; src/util/mpz.cpp:730-817.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

type polynomialSOMNormalizeInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	buffer     *polynomialSOM
	frame      *polynomialSOMNormalizeScope
	invocation uint64
}
type polynomialSOMNormalizeScope struct {
	active, executing, finishing, ownEnded bool
	invocation                             uint64
	receipt                                polynomialSOMNormalizeInvocationReceipt
	normalize                              integerPNormalizeInvocationReceipt
	gcdRaw, divisionRaw, extendedRaw       integerRawInvocationReceipt
	buffers                                [4]naturalPNBufferReceipt
	verification                           naturalPNVerificationReceipt
	sourceCause, ownCause                  error
}

// Both persistent imp members and the actual separately charged source SOM
// wrapper have concrete member addresses. A matching owner pointer alone is
// never sufficient for a copied or fabricated buffer.
func (m *MultivariatePolynomialCore) actualSOMNormalizeBuffer(b *polynomialSOM) bool {
	if m == nil || m.self != m || b == nil || b.owner != m {
		return false
	}
	if b == &m.som || b == &m.som2 || b == &m.cheap || b == &m.cheap2 {
		return true
	}
	o := b.retirementOwner
	return o != nil && o.self == o && o.storage == m && b == &o.buffer
}

func (m *MultivariatePolynomialCore) hasSOMNormalizeInvocationHistory(r *polynomialSOMNormalizeInvocationReceipt) bool {
	if m == nil || m.self != m || r == nil || r.manager != m || r.buffer == nil || r.invocation == 0 {
		return false
	}
	b := r.buffer
	if !m.actualSOMNormalizeBuffer(b) {
		return false
	}
	return b.owner == m && r.frame == &b.normalizeScope && r.frame.invocation == r.invocation
}
func (m *IntegerManager) somNormalizeAllocationRoots(r *polynomialSOMNormalizeInvocationReceipt) *polynomialNumericRoots {
	if m == nil || m.self != m || r == nil || r.manager == nil || !r.manager.hasSOMNormalizeInvocationHistory(r) || r.manager.integer != m || !r.frame.active || !r.frame.executing || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.buffer.gcdNumeric
}
func (c integerAllocationInvocationContext) somNormalizeReceipt(m *IntegerManager) *polynomialSOMNormalizeInvocationReceipt {
	if c.somNormalize == nil || c.somNormalizeInvocation == 0 || c.somNormalizeInvocation != c.somNormalize.invocation || m.somNormalizeAllocationRoots(c.somNormalize) == nil {
		return nil
	}
	return c.somNormalize
}
func (m *IntegerModularManager) somNormalizeAllocationContext(r *polynomialSOMNormalizeInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || m.manager.somNormalizeAllocationRoots(r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{somNormalize: r, somNormalizeInvocation: r.invocation}
}
func (m *IntegerModularManager) gcdManyIndexedWithSOMNormalizeReceipt(as integerArrayView, out *Int, r *polynomialSOMNormalizeInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.gcdManyIndexedWithAllocationContext(as, out, m.somNormalizeAllocationContext(r)); e != nil {
		return e
	}
	return m.assertNormalized(out)
}
func (m *IntegerModularManager) divWithSOMNormalizeReceipt(a, b, out *Int, r *polynomialSOMNormalizeInvocationReceipt) error {
	return m.divWithAllocationContext(a, b, out, m.somNormalizeAllocationContext(r))
}
func (c integerAllocationInvocationContext) recordSOMNormalizationRaw(m *IntegerManager, r *integerRawInvocationReceipt) {
	p := c.somNormalizeReceipt(m)
	if p == nil || !m.matchesRawInvocation(r) {
		return
	}
	r.frame.somNormalize = *p
	switch r.frame {
	case &m.gcdRaw:
		p.frame.gcdRaw = *r
	case &m.divisionRaw:
		p.frame.divisionRaw = *r
	case &m.extendedRaw:
		p.frame.extendedRaw = *r
	}
}
func (m *IntegerManager) somNormalizeNumericSourceCause(r *polynomialSOMNormalizeInvocationReceipt) error {
	if r == nil || r.manager == nil || !r.manager.hasSOMNormalizeInvocationHistory(r) || r.manager.integer != m {
		return nil
	}
	s := r.frame
	if e := r.manager.modular.pNormalizeInvocationSourceCause(&s.normalize); e != nil {
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
func (m *IntegerManager) cleanupSOMNormalizationNumeric(r *polynomialSOMNormalizeInvocationReceipt, abort bool) error {
	if r == nil || r.manager == nil || !r.manager.hasSOMNormalizeInvocationHistory(r) || r.manager.integer != m {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := r.manager.modular.cleanupPNormalizeInvocation(&s.normalize, abort); e != nil {
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
		return r.buffer.gcdNumeric.cleanup(m)
	}
	return nil
}
