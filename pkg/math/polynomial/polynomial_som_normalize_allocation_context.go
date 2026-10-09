// Source: src/math/polynomial/polynomial.cpp:1863-1876;
// src/util/mpzzp.h:167-185,211; src/util/mpz.cpp:730-817.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

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
	normalize                              util.IntegerNormalizeInvocationReceipt
	gcdRaw, divisionRaw, extendedRaw       util.IntegerRawInvocationReceipt
	buffers                                [4]util.NaturalBufferReceipt

	sourceCause, ownCause error
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
func numericIntegerManager_somNormalizeAllocationRoots(m *util.IntegerManager, r *polynomialSOMNormalizeInvocationReceipt) *polynomialNumericRoots {
	if m == nil || !m.ValidOwner() || r == nil || r.manager == nil || !r.manager.hasSOMNormalizeInvocationHistory(r) || r.manager.integer != m || !r.frame.active || !r.frame.executing || r.manager.callbackDepth != 0 {
		return nil
	}
	return &r.buffer.gcdNumeric
}
func (c integerAllocationInvocationContext) somNormalizeReceipt(m *util.IntegerManager) *polynomialSOMNormalizeInvocationReceipt {
	if c.somNormalize == nil || c.somNormalizeInvocation == 0 || c.somNormalizeInvocation != c.somNormalize.invocation || numericIntegerManager_somNormalizeAllocationRoots(m, c.somNormalize) == nil {
		return nil
	}
	return c.somNormalize
}
func numericIntegerModularManager_somNormalizeAllocationContext(m *util.IntegerModularManager, r *polynomialSOMNormalizeInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.Integers() == nil || numericIntegerManager_somNormalizeAllocationRoots(m.Integers(), r) == nil || r.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{somNormalize: r, somNormalizeInvocation: r.invocation}
}
func numericIntegerModularManager_gcdManyIndexedWithSOMNormalizeReceipt(m *util.IntegerModularManager, as util.IntegerArrayView, out *util.Int, r *polynomialSOMNormalizeInvocationReceipt) error {
	if e := m.Check(); e != nil {
		return e
	}
	if e := m.Integers().WithAllocationContext((numericIntegerModularManager_somNormalizeAllocationContext(m, r)).utility(m.Integers())).GcdManyIndexed(as, out); e != nil {
		return e
	}
	return m.AssertNormalized(out)
}
func numericIntegerModularManager_divWithSOMNormalizeReceipt(m *util.IntegerModularManager, a, b, out *util.Int, r *polynomialSOMNormalizeInvocationReceipt) error {
	return m.WithAllocationContext((numericIntegerModularManager_somNormalizeAllocationContext(m, r)).utility(m.Integers())).Div(a, b, out)
}
func (c integerAllocationInvocationContext) recordSOMNormalizationRaw(m *util.IntegerManager, r *util.IntegerRawInvocationReceipt) {
	p := c.somNormalizeReceipt(m)
	if p == nil || !m.MatchesRawInvocation(r) {
		return
	}

	switch r.Kind() {
	case util.IntegerRawGCD:
		p.frame.gcdRaw = *r
	case util.IntegerRawDivision:
		p.frame.divisionRaw = *r
	case util.IntegerRawExtendedGCD:
		p.frame.extendedRaw = *r
	}
}
func numericIntegerManager_somNormalizeNumericSourceCause(m *util.IntegerManager, r *polynomialSOMNormalizeInvocationReceipt) error {
	if r == nil || r.manager == nil || !r.manager.hasSOMNormalizeInvocationHistory(r) || r.manager.integer != m {
		return nil
	}
	s := r.frame

	if e := m.RawInvocationSourceCause(&s.extendedRaw); e != nil {
		return e
	}
	if e := m.RawInvocationSourceCause(&s.divisionRaw); e != nil {
		return e
	}
	if e := m.RawInvocationSourceCause(&s.gcdRaw); e != nil {
		return e
	}
	return nil
}
func numericIntegerManager_cleanupSOMNormalizationNumeric(m *util.IntegerManager, r *polynomialSOMNormalizeInvocationReceipt, abort bool) error {
	if r == nil || r.manager == nil || !r.manager.hasSOMNormalizeInvocationHistory(r) || r.manager.integer != m {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	if e := r.manager.modular.CleanupPNormalizeInvocation(&s.normalize, abort); e != nil {
		return e
	}
	if e := m.CleanupRawInvocation(&s.extendedRaw, abort); e != nil {
		return e
	}
	if e := m.CleanupRawInvocation(&s.divisionRaw, abort); e != nil {
		return e
	}
	if e := m.CleanupRawInvocation(&s.gcdRaw, abort); e != nil {
		return e
	}

	if abort {
		for i := len(s.buffers) - 1; i >= 0; i-- {
			if e := s.buffers[i].Cleanup(); e != nil {
				return e
			}
		}
		return r.buffer.gcdNumeric.cleanup(m)
	}
	return nil
}
