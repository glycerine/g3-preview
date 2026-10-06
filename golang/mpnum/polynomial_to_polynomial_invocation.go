// Source: src/math/polynomial/polynomial.cpp:6832-6841.
package mpnum

import "reflect"

type polynomialToPolynomialHistory struct {
	invocation                           uint64
	bodyCause, sourceCause, cleanupCause error
	normalize                            integerPNormalizeInvocationReceipt
	append                               polynomialCheapAppendInvocationReceipt
	factory                              polynomialFactoryInvocationReceipt
	monomialAllocator                    monomialAllocatorInvocationReceipt
}

func (m *MultivariatePolynomialManager) archiveToPolynomial() error {
	s := &m.toPolynomial
	if s.invocation == 0 {
		return nil
	}
	old := cap(m.toPolynomialHistory)
	if len(m.toPolynomialHistory) == old {
		next := old + old/2 + 1
		stride := uint64(reflect.TypeOf(polynomialToPolynomialHistory{}).Size())
		if e := m.originalBudget.Reserve(uint64(next) * stride); e != nil {
			return e
		}
		rows := make([]polynomialToPolynomialHistory, len(m.toPolynomialHistory), next)
		copy(rows, m.toPolynomialHistory)
		m.toPolynomialHistory = rows
		m.originalBudget.Release(uint64(old) * stride)
	}
	m.toPolynomialHistory = append(m.toPolynomialHistory, polynomialToPolynomialHistory{s.invocation, s.bodyCause, s.sourceCause, s.cleanupCause, s.normalize, s.append, s.factory, s.monomialAllocator})
	return nil
}
func (m *MultivariatePolynomialManager) closeToPolynomialHistory() {
	m.originalBudget.Release(uint64(cap(m.toPolynomialHistory)) * uint64(reflect.TypeOf(polynomialToPolynomialHistory{}).Size()))
	m.toPolynomialHistory = nil
}
func (m *MultivariatePolynomialManager) toPolynomialInvocationSourceCause(r *polynomialToPolynomialInvocationReceipt) error {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame != &m.toPolynomial || r.invocation == 0 {
		return nil
	}
	if r.invocation == m.toPolynomial.invocation {
		return m.toPolynomial.sourceCause
	}
	for i := range m.toPolynomialHistory {
		if m.toPolynomialHistory[i].invocation == r.invocation {
			return m.toPolynomialHistory[i].sourceCause
		}
	}
	return nil
}

type polynomialToPolynomialInvocationReceipt struct {
	manager    *MultivariatePolynomialManager
	frame      *polynomialToPolynomialScope
	invocation uint64
}

func (m *MultivariatePolynomialManager) matchesToPolynomialInvocation(r *polynomialToPolynomialInvocationReceipt) bool {
	return m != nil && m.self == m && m.core != nil && m.core.self == m.core && r != nil && r.manager == m && r.frame == &m.toPolynomial && r.invocation != 0 && r.invocation == r.frame.invocation
}
func (m *IntegerManager) toPolynomialAllocationRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	r := c.toPolynomial
	if m == nil || m.self != m || r == nil || !r.manager.matchesToPolynomialInvocation(r) || c.toPolynomialInvocation != r.invocation || r.manager.core.integer != m || !r.frame.active || !r.frame.executing || r.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (b *scopedIntegerModularBuffer128) setToPolynomial(i uint32, v *Int, r *polynomialToPolynomialInvocationReceipt) error {
	if e := b.open(); e != nil {
		return e
	}
	if r == nil || !r.manager.matchesToPolynomialInvocation(r) || b != &r.frame.buffer {
		return ErrPolynomialOwner
	}
	s := r.frame
	context := integerAllocationInvocationContext{toPolynomial: r, toPolynomialInvocation: r.invocation}
	if b.manager.manager.toPolynomialAllocationRoots(context) == nil || b.manager != r.manager.core.modular {
		return ErrPolynomialOwner
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.bodyCause != nil {
		return b.bodyCause
	}
	if s.transportBound {
		return ErrPolynomialPending
	}
	a, e := b.get(i)
	if e != nil {
		return e
	}
	s.transport = a
	s.transportBound = true
	e = b.manager.copyWithToPolynomialReceipt(&s.transport, v, r)
	write := b.store(i, s.transport)
	if write == nil {
		s.transport = Int{}
		s.transportBound = false
	}
	cause := polynomialNewtonJoin(e, write)
	if cause != nil && b.bodyCause == nil {
		b.bodyCause = cause
	}
	return cause
}
func (b *scopedIntegerModularBuffer128) pushToPolynomial(v *Int, r *polynomialToPolynomialInvocationReceipt) error {
	if e := b.open(); e != nil {
		return e
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.bodyCause != nil {
		return b.bodyCause
	}
	if e := b.pushZero(); e != nil {
		b.bodyCause = e
		return e
	}
	return b.setToPolynomial(b.count-1, v, r)
}
func (m *MultivariatePolynomialManager) cleanupToPolynomialInvocation(r *polynomialToPolynomialInvocationReceipt, abort bool) (cause error) {
	if !m.matchesToPolynomialInvocation(r) {
		return nil
	}
	s := r.frame
	if !s.active {
		return nil
	}
	if s.executing || s.finishing || m.core.callbackDepth != 0 {
		return ErrPolynomialPending
	}
	s.finishing = true
	defer func() {
		s.finishing = false
		if cause != nil && s.cleanupCause == nil {
			s.cleanupCause = cause
		}
	}()
	if e := m.core.modular.cleanupPNormalizeInvocation(&s.normalize, false); e != nil {
		return e
	}
	if !abort {
		return nil
	}
	// Actual coefficient buffer is older than the returned factory, but the
	// unjoined factory has no source destructor. Abort it after buffer NUMs.
	if e := s.buffer.cleanup(); e != nil {
		return e
	}
	if e := m.core.modular.Del(&s.transport); e != nil {
		return e
	}
	s.transportBound = false
	if e := m.core.modular.cleanupPNormalizeInvocation(&s.normalize, true); e != nil {
		return e
	}
	if e := s.numeric.cleanup(m.core.integer); e != nil {
		return e
	}
	if e := m.core.cleanupFactoryInvocation(&s.factory, true); e != nil {
		return e
	}
	if e := m.core.cheap.cleanupAppendInvocation(&s.append, true); e != nil {
		return e
	}
	if s.generated != (Monomial{}) {
		if e := m.core.monomials.CleanupValue(&s.generated); e != nil {
			return e
		}
	}
	if s.calleeGeneration != 0 && m.core.monomials.creation.toPolynomialParent == s && m.core.monomials.creation.toPolynomialInvocation == s.invocation {
		if e := m.core.monomials.CleanupCreation(); e != nil {
			return e
		}
		if m.core.generatedCalleeGeneration == s.calleeGeneration {
			m.core.generatedCalleePending = false
		}
	}
	if e := m.core.monomials.cleanupMonomialAllocatorInvocation(&s.monomialAllocator, true); e != nil {
		return e
	}
	s.active = false
	return nil
}

func (m *IntegerModularManager) copyWithToPolynomialReceipt(out, value *Int, r *polynomialToPolynomialInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	context := integerAllocationInvocationContext{toPolynomial: r}
	if r != nil {
		context.toPolynomialInvocation = r.invocation
	}
	if r == nil || m.manager.toPolynomialAllocationRoots(context) == nil || r.manager.core.modular != m {
		return ErrPolynomialOwner
	}
	if m.manager.factorNumeralExecuting {
		return ErrPolynomialPending
	}
	prior := m.manager.factorNumeralExecuting
	m.manager.factorNumeralExecuting = true
	defer func() { m.manager.factorNumeralExecuting = prior }()
	return m.copyWithSourceContext(out, value, context)
}

// Lifecycle exclusion only. Actual acquisition always uses the passed receipt.
func (m *MultivariatePolynomialCore) toPolynomialCallerExecuting() bool {
	if m == nil {
		return false
	}
	f := m.toPolynomialCaller
	return m != nil && m.self == m && f != nil && f.self == f && f.core == m && f.matchesToPolynomialInvocation(&f.toPolynomial.direct) && (f.toPolynomial.executing || f.toPolynomial.finishing)
}
