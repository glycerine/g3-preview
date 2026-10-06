// Source: src/math/polynomial/upolynomial.h:401-423.
package mpnum

import (
	"errors"
	"reflect"
)

type polynomialFactorNumeralVectorReceipt struct {
	manager    *MultivariatePolynomialManager
	frame      *polynomialFactorNumeralVectorScope
	invocation uint64
}
type polynomialFactorNumeralVectorScope struct {
	phase                                     uint8
	referenceConstructed, sourceReferenceDone bool
	invocation                                uint64
	active, executing, finishing, constructed bool
	direct                                    polynomialFactorNumeralVectorReceipt
	output                                    *upolyVectorState
	np                                        polynomialReference
	normalization                             polynomialNormalizeInvocationReceipt
	numeric                                   polynomialNumericRoots
	normalize                                 integerPNormalizeInvocationReceipt
	bodyCause, sourceCause, cleanupCause      error
}

func (m *MultivariatePolynomialManager) matchesFactorNumeralVector(r *polynomialFactorNumeralVectorReceipt) bool {
	return m != nil && m.self == m && m.core != nil && m.core.self == m.core && r != nil && r.manager == m && r.frame == &m.factorNumeralVector && r.invocation != 0 && r.invocation == r.frame.invocation
}
func (m *IntegerManager) factorNumeralVectorRoots(context integerAllocationInvocationContext) *polynomialNumericRoots {
	r := context.factorNumeralVector
	if m == nil || m.self != m || r == nil || !r.manager.matchesFactorNumeralVector(r) || context.factorNumeralVectorInvocation != r.invocation || r.manager.core.integer != m || !r.frame.active || !r.frame.executing || r.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (m *MultivariatePolynomialManager) toNumeralVectorWithReceipt(p Polynomial, output *upolyVectorState, r *polynomialFactorNumeralVectorReceipt) (cause error) {
	if e := m.ready(); e != nil {
		return e
	}
	if output == nil {
		return ErrUPolynomialOwner
	}
	c := m.core
	if modularSourceDebug {
		ok, e := c.IsUnivariate(p)
		if e != nil {
			return e
		}
		if !ok {
			return ErrPolynomialAssertion
		}
	}
	s := &m.factorNumeralVector
	if s.active || s.executing || s.finishing || c.callbackDepth != 0 || s.invocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	if e := m.archiveFactorNumeralVector(); e != nil {
		return e
	}
	*s = polynomialFactorNumeralVectorScope{invocation: s.invocation + 1, active: true, executing: true, constructed: true, referenceConstructed: true, output: output}
	s.direct = polynomialFactorNumeralVectorReceipt{m, s, s.invocation}
	c.integer.factorNumeralVectorCaller = m
	if r != nil {
		*r = s.direct
	}
	context := integerAllocationInvocationContext{factorNumeralVector: &s.direct, factorNumeralVectorInvocation: s.invocation}
	defer func() {
		s.executing = false
		if s.bodyCause == nil {
			s.bodyCause = cause
		}
		cause = errors.Join(cause, m.finishFactorNumeralVectorSource())
	}()
	np, e := m.normalizeWithReceipt(p, &s.normalization)
	if e != nil {
		return e
	}
	if e = s.np.assign(c, np); e != nil {
		return e
	}
	// Literal reads remain on ORIGINAL p, after assigning unused np.
	size, e := c.readWord(p, 8)
	if e != nil {
		return e
	}
	degree, e := c.TotalDegree(p)
	if e != nil {
		return e
	}
	if e = m.upm.reserve(output, degree+1); e != nil {
		return e
	}
	for i := uint32(0); i <= degree; i++ {
		value, e := m.upm.read(m.upm.stateView(output), i)
		if e != nil {
			return e
		}
		if e = m.upm.modular.Reset(value); e != nil {
			return e
		}
	}
	for i := uint32(0); i < size; i++ {
		term, e := c.monomialCore(p, i)
		if e != nil {
			return e
		}
		k, e := c.monomials.degreeCore(term)
		if e != nil {
			return e
		}
		if modularSourceDebug && k > degree {
			return ErrPolynomialAssertion
		}
		value, e := m.upm.read(m.upm.stateView(output), k)
		if e != nil {
			return e
		}
		coefficient, e := c.coefficientCore(p, i)
		if e != nil {
			return e
		}
		s.phase = 1
		prior := c.integer.factorNumeralExecuting
		c.integer.factorNumeralExecuting = true
		e = m.upm.modular.copyWithSourceContext(value, &coefficient, context)
		c.integer.factorNumeralExecuting = prior
		if e != nil {
			return e
		}
	}
	return m.upm.setSize(output, degree+1)
}
func (m *MultivariatePolynomialManager) finishFactorNumeralVectorSource() error {
	s := &m.factorNumeralVector
	if !m.matchesFactorNumeralVector(&s.direct) {
		return ErrPolynomialOwner
	}
	if s.executing || s.finishing || m.core.callbackDepth != 0 {
		return ErrPolynomialPending
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	s.finishing = true
	defer func() { s.finishing = false }()
	if e := m.upm.modular.pNormalizeInvocationSourceCause(&s.normalize); e != nil {
		s.sourceCause = e
		return e
	}
	if e := m.normalizeInvocationSourceCause(&s.normalization); e != nil {
		s.sourceCause = e
		return e
	}
	if s.constructed {
		if e := s.np.closeValue(m.core, false); e != nil {
			s.sourceCause = e
			return e
		}
		s.constructed = false
		s.sourceReferenceDone = true
	}
	if !polynomialNegNumericPending(&s.numeric) && !m.upm.modular.matchesPNormalizeInvocation(&s.normalize) && !m.matchesNormalizeInvocation(&s.normalization) {
		s.active = false
	}
	return nil
}
func (m *MultivariatePolynomialManager) cleanupFactorNumeralVector(r *polynomialFactorNumeralVectorReceipt, abort bool) (cause error) {
	if !m.matchesFactorNumeralVector(r) {
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
	if e := m.upm.modular.cleanupPNormalizeInvocation(&s.normalize, false); e != nil {
		return e
	}
	if e := m.cleanupNormalizeInvocation(&s.normalization, false); e != nil {
		return e
	}
	if s.constructed {
		if e := s.np.close(m.core); e != nil {
			return e
		}
		s.constructed = false
	}
	if !abort {
		return nil
	}
	if e := m.upm.modular.cleanupPNormalizeInvocation(&s.normalize, true); e != nil {
		return e
	}
	if e := m.cleanupNormalizeInvocation(&s.normalization, true); e != nil {
		return e
	}
	if e := s.numeric.cleanup(m.core.integer); e != nil {
		return e
	}
	s.active = false
	return nil
}

type polynomialFactorNumeralVectorHistory struct {
	invocation                                uint64
	phase                                     uint8
	referenceConstructed, sourceReferenceDone bool
	normalization                             polynomialNormalizeInvocationReceipt
	normalize                                 integerPNormalizeInvocationReceipt
	bodyCause, sourceCause, cleanupCause      error
}

func (m *MultivariatePolynomialManager) archiveFactorNumeralVector() error {
	s := &m.factorNumeralVector
	if s.invocation == 0 {
		return nil
	}
	old := cap(m.factorNumeralVectorHistory)
	if len(m.factorNumeralVectorHistory) == old {
		n := old + old/2 + 1
		stride := uint64(reflect.TypeOf(polynomialFactorNumeralVectorHistory{}).Size())
		if e := m.originalBudget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		rows := make([]polynomialFactorNumeralVectorHistory, len(m.factorNumeralVectorHistory), n)
		copy(rows, m.factorNumeralVectorHistory)
		m.factorNumeralVectorHistory = rows
		m.originalBudget.Release(uint64(old) * stride)
	}
	m.factorNumeralVectorHistory = append(m.factorNumeralVectorHistory, polynomialFactorNumeralVectorHistory{s.invocation, s.phase, s.referenceConstructed, s.sourceReferenceDone, s.normalization, s.normalize, s.bodyCause, s.sourceCause, s.cleanupCause})
	return nil
}
func (m *MultivariatePolynomialManager) factorNumeralVectorDiagnostic(r *polynomialFactorNumeralVectorReceipt) (polynomialFactorNumeralVectorHistory, bool) {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame != &m.factorNumeralVector || r.invocation == 0 {
		return polynomialFactorNumeralVectorHistory{}, false
	}
	s := &m.factorNumeralVector
	if s.invocation == r.invocation {
		return polynomialFactorNumeralVectorHistory{s.invocation, s.phase, s.referenceConstructed, s.sourceReferenceDone, s.normalization, s.normalize, s.bodyCause, s.sourceCause, s.cleanupCause}, true
	}
	for _, row := range m.factorNumeralVectorHistory {
		if row.invocation == r.invocation {
			return row, true
		}
	}
	return polynomialFactorNumeralVectorHistory{}, false
}
func (m *MultivariatePolynomialManager) closeFactorNumeralVectorHistory() {
	m.originalBudget.Release(uint64(cap(m.factorNumeralVectorHistory)) * uint64(reflect.TypeOf(polynomialFactorNumeralVectorHistory{}).Size()))
	m.factorNumeralVectorHistory = nil
}

// This actual executing caller excludes owner retirement only. Numeric authority
// remains the explicit converter or its reached Normalize child receipt.
func (m *IntegerManager) factorNumeralVectorCallerExecuting() bool {
	if m == nil || m.self != m {
		return false
	}
	f := m.factorNumeralVectorCaller
	return f != nil && f.self == f && f.core != nil && f.core.integer == m && f.matchesFactorNumeralVector(&f.factorNumeralVector.direct) && (f.factorNumeralVector.executing || f.factorNumeralVector.finishing)
}
