// Source: src/math/polynomial/polynomial.cpp:6683-6725.
package polynomial

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

type polynomialFactorUnivariateInvocationReceipt struct {
	manager    *MultivariatePolynomialManager
	frame      *polynomialFactorUnivariateScope
	invocation uint64
}
type polynomialFactorUnivariateScope struct {
	callerBinding                           util.NumericCallerBinding
	owner                                   *MultivariatePolynomialManager
	invocation                              uint64
	target                                  *PolynomialFactors
	active, executing, finishing            bool
	constructed                             uint8
	conditionalReference                    bool
	sourceCursor, ownCursor                 uint8
	p1                                      upolyVectorState
	p1SourceCursor                          uint32
	p, f                                    polynomialReference
	fs                                      *UPolynomialFactors
	fsSource, fsOwn                         upolyPreparedFactorCursor
	converter                               polynomialFactorNumeralVectorReceipt
	squareFree                              upolyFactorSquareFreeInvocationReceipt
	toPolynomial                            polynomialToPolynomialInvocationReceipt
	predicatePrimitive, predicateSquareFree polynomialGCDInvocationReceipt
	constant                                polynomialFactorConstantInvocationReceipt
	bodyCause, sourceCause, cleanupCause    error
}

func (m *MultivariatePolynomialManager) matchesFactorUnivariate(r *polynomialFactorUnivariateInvocationReceipt) bool {
	return m != nil && m.self == m && m.core != nil && m.core.self == m.core && r != nil && r.manager == m && r.frame == &m.factorUnivariate && r.invocation != 0 && r.invocation == r.frame.invocation
}
func (m *MultivariatePolynomialManager) factorSquareFreePrimitiveUnivariateWithReceipt(p Polynomial, r *PolynomialFactors, k uint32, params UPolynomialFactorParams, receipt *polynomialFactorUnivariateInvocationReceipt) (cause error) {
	if e := m.ready(); e != nil {
		return e
	}
	if r == nil || r.manager != m {
		return ErrPolynomialOwner
	}
	if e := r.open(); e != nil {
		return e
	}
	s := &m.factorUnivariate
	if s.active || s.executing || s.finishing || m.core.callbackDepth != 0 || s.invocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	if e := m.archiveFactorUnivariate(); e != nil {
		return e
	}
	*s = polynomialFactorUnivariateScope{owner: m, invocation: s.invocation + 1, active: true, executing: true, target: r}
	direct := polynomialFactorUnivariateInvocationReceipt{m, s, s.invocation}
	previousCaller := factorUnivariateCaller(m.core.integer)
	setFactorUnivariateCaller(m.core.integer, m)
	defer setFactorUnivariateCaller(m.core.integer, previousCaller)
	if receipt != nil {
		*receipt = direct
	}
	defer func() {
		s.executing = false
		if s.bodyCause == nil {
			s.bodyCause = cause
		}
		cause = errors.Join(cause, m.finishFactorUnivariateSource(&direct))
	}()
	c := m.core
	if polynomialSourceDebug {
		yes, e := c.IsUnivariate(p)
		if e != nil {
			return e
		}
		if !yes {
			return ErrPolynomialAssertion
		}
		x, e := c.MaxVariable(p)
		if e != nil {
			return e
		}
		yes, e = m.isSquareFreeWithReceipt(p, x, true, &s.predicateSquareFree)
		if e != nil {
			return e
		}
		if !yes {
			return ErrPolynomialAssertion
		}
		yes, e = m.isPrimitiveWithReceipt(p, x, &s.predicatePrimitive)
		if e != nil {
			return e
		}
		if !yes {
			return ErrPolynomialAssertion
		}
		zero, e := c.IsZero(p)
		if e != nil {
			return e
		}
		if zero {
			return ErrPolynomialAssertion
		}
	}
	x, e := c.MaxVariable(p)
	if e != nil {
		return e
	}
	s.constructed = 1 // actual empty scoped_numeral_vector p1
	s.constructed = 2 // actual default polynomial_ref p_ref
	if e = s.p.assign(c, p); e != nil {
		return e
	}
	if e = m.toNumeralVectorWithReceipt(s.p.value, &s.p1, &s.converter); e != nil {
		return e
	}
	s.fs, e = m.upm.NewFactors()
	if e != nil {
		return e
	}
	s.fs.factorUnivariateCaller = m
	s.constructed = 3
	service, e := m.factorizationService()
	if e != nil {
		return e
	}
	_, e = service.factorSquareFreeWithReceipt(m.upm.stateView(&s.p1), s.fs, 1, params, &s.squareFree)
	if e != nil {
		return e
	}
	if polynomialSourceDebug {
		one, e := c.modular.IsOne(&s.fs.constant)
		if e != nil {
			return e
		}
		minus, e := c.modular.IsMinusOne(&s.fs.constant)
		if e != nil {
			return e
		}
		if !one && !minus {
			return ErrPolynomialAssertion
		}
	}
	if s.fs.count == 1 && s.fs.degrees.values[0] == 1 {
		return r.Push(p, k)
	}
	s.conditionalReference = true // actual f constructor only in else
	n := s.fs.count
	for i := uint32(0); i < n; i++ {
		vector, e := s.fs.factor(i)
		if e != nil {
			return e
		}
		multiplicity := s.fs.degrees.values[i]
		result, e := m.toPolynomialWithReceipt(vector.size, m.upm.values(vector), x, &s.toPolynomial)
		if e != nil {
			return e
		}
		if e = s.f.assign(c, result); e != nil {
			return e
		}
		if e = r.Push(s.f.value, k*multiplicity); e != nil {
			return e
		}
	}
	one, e := c.modular.IsOne(&s.fs.constant)
	if e != nil {
		return e
	}
	minus, e := c.modular.IsMinusOne(&s.fs.constant)
	if e != nil {
		return e
	}
	if polynomialSourceDebug && !one && !minus {
		return ErrPolynomialAssertion
	}
	if minus && k%2 == 1 {
		return m.flipFactorSignWithReceipt(r, &s.constant)
	}
	return nil
}
func (m *MultivariatePolynomialManager) factorUnivariateYoungerSource(s *polynomialFactorUnivariateScope) error {
	if s.constant.manager != nil {
		if e := s.constant.manager.factorConstantInvocationSourceCause(&s.constant); e != nil {
			return e
		}
	}
	if e := m.toPolynomialInvocationSourceCause(&s.toPolynomial); e != nil {
		return e
	}
	if m.factorService != nil {
		if e := m.factorService.factorSquareFreeInvocationSourceCause(&s.squareFree); e != nil {
			return e
		}
	}
	if row, ok := m.factorNumeralVectorDiagnostic(&s.converter); ok && row.sourceCause != nil {
		return row.sourceCause
	}
	if e := m.gcdInvocationSourceCause(&s.predicatePrimitive); e != nil {
		return e
	}
	return m.gcdInvocationSourceCause(&s.predicateSquareFree)
}
func (m *MultivariatePolynomialManager) finishFactorUnivariateSource(r *polynomialFactorUnivariateInvocationReceipt) (cause error) {
	if !m.matchesFactorUnivariate(r) {
		return ErrPolynomialOwner
	}
	s := r.frame
	if s.executing || s.finishing || m.core.callbackDepth != 0 {
		return ErrPolynomialPending
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	previousCaller := factorUnivariateCaller(m.core.integer)
	setFactorUnivariateCaller(m.core.integer, m)
	defer setFactorUnivariateCaller(m.core.integer, previousCaller)
	s.finishing = true
	defer func() {
		s.finishing = false
		if cause != nil && s.sourceCause == nil {
			s.sourceCause = cause
		}
	}()
	if e := m.factorUnivariateYoungerSource(s); e != nil {
		return e
	}
	if s.sourceCursor == 0 {
		if s.conditionalReference {
			if e := s.f.closeValue(m.core, false); e != nil {
				return e
			}
		}
		s.sourceCursor = 1
	}
	if s.sourceCursor == 1 {
		if s.constructed >= 3 {
			if e := s.fs.closeWithSourceContext(&s.fsSource, true, integerAllocationInvocationContext{}); e != nil {
				return e
			}
		}
		s.sourceCursor = 2
	}
	if s.sourceCursor == 2 {
		if s.constructed >= 2 {
			if e := s.p.closeValue(m.core, false); e != nil {
				return e
			}
		}
		s.sourceCursor = 3
	}
	if s.sourceCursor == 3 {
		if s.constructed >= 1 {
			if e := upolynomialScopedVectorSourceDel(m.upm, &s.p1, &s.p1SourceCursor); e != nil {
				return e
			}
		}
		s.sourceCursor = 4
	}
	return nil
}
func (m *MultivariatePolynomialManager) cleanupFactorUnivariateInvocation(r *polynomialFactorUnivariateInvocationReceipt, abort bool) (cause error) {
	if !m.matchesFactorUnivariate(r) {
		return nil
	}
	s := r.frame
	if !s.active {
		return nil
	}
	if s.executing || s.finishing || m.core.callbackDepth != 0 {
		return ErrPolynomialPending
	}
	previousCaller := factorUnivariateCaller(m.core.integer)
	setFactorUnivariateCaller(m.core.integer, m)
	defer setFactorUnivariateCaller(m.core.integer, previousCaller)
	s.finishing = true
	defer func() {
		s.finishing = false
		if cause != nil && s.cleanupCause == nil {
			s.cleanupCause = cause
		}
	}()
	if s.constant.manager != nil {
		if e := s.constant.manager.cleanupFactorConstantInvocation(&s.constant, false); e != nil {
			return e
		}
	}
	if e := m.cleanupGCDInvocation(&s.predicatePrimitive, false); e != nil {
		return e
	}
	if e := m.cleanupGCDInvocation(&s.predicateSquareFree, false); e != nil {
		return e
	}
	if e := m.cleanupToPolynomialInvocation(&s.toPolynomial, false); e != nil {
		return e
	}
	if m.factorService != nil {
		if e := m.factorService.cleanupFactorSquareFreeInvocation(&s.squareFree, false); e != nil {
			return e
		}
	}
	if e := m.cleanupFactorNumeralVector(&s.converter, false); e != nil {
		return e
	}
	if s.ownCursor == 0 {
		if s.conditionalReference {
			if e := s.f.close(m.core); e != nil {
				return e
			}
		}
		s.ownCursor = 1
	}
	if s.ownCursor == 1 {
		if s.fs != nil {
			if e := s.fs.closeWithSourceContext(&s.fsOwn, false, integerAllocationInvocationContext{}); e != nil {
				return e
			}
		}
		s.ownCursor = 2
	}
	if s.ownCursor == 2 {
		if s.constructed >= 2 {
			if e := s.p.close(m.core); e != nil {
				return e
			}
		}
		s.ownCursor = 3
	}
	if s.ownCursor == 3 {
		if e := m.upm.reset(&s.p1); e != nil {
			return e
		}
		m.upm.freeBacking(&s.p1)
		s.ownCursor = 4
	}
	if !abort {
		return nil
	}
	if s.constant.manager != nil {
		if e := s.constant.manager.cleanupFactorConstantInvocation(&s.constant, true); e != nil {
			return e
		}
	}
	if e := m.cleanupToPolynomialInvocation(&s.toPolynomial, true); e != nil {
		return e
	}
	if m.factorService != nil {
		if e := m.factorService.cleanupFactorSquareFreeInvocation(&s.squareFree, true); e != nil {
			return e
		}
	}
	if e := m.cleanupFactorNumeralVector(&s.converter, true); e != nil {
		return e
	}
	if e := m.cleanupGCDInvocation(&s.predicatePrimitive, true); e != nil {
		return e
	}
	if e := m.cleanupGCDInvocation(&s.predicateSquareFree, true); e != nil {
		return e
	}
	s.active = false
	return nil
}

type polynomialFactorUnivariateHistory struct {
	invocation                              uint64
	constructed, sourceCursor, ownCursor    uint8
	conditionalReference                    bool
	converter                               polynomialFactorNumeralVectorReceipt
	squareFree                              upolyFactorSquareFreeInvocationReceipt
	toPolynomial                            polynomialToPolynomialInvocationReceipt
	predicatePrimitive, predicateSquareFree polynomialGCDInvocationReceipt
	constant                                polynomialFactorConstantInvocationReceipt
	bodyCause, sourceCause, cleanupCause    error
}

func (m *MultivariatePolynomialManager) factorUnivariateHistoryRow() polynomialFactorUnivariateHistory {
	s := &m.factorUnivariate
	return polynomialFactorUnivariateHistory{s.invocation, s.constructed, s.sourceCursor, s.ownCursor, s.conditionalReference, s.converter, s.squareFree, s.toPolynomial, s.predicatePrimitive, s.predicateSquareFree, s.constant, s.bodyCause, s.sourceCause, s.cleanupCause}
}
func (m *MultivariatePolynomialManager) archiveFactorUnivariate() error {
	s := &m.factorUnivariate
	if s.invocation == 0 {
		return nil
	}
	old := cap(m.factorUnivariateHistory)
	if len(m.factorUnivariateHistory) == old {
		n := old + old/2 + 1
		stride := uint64(reflect.TypeOf(polynomialFactorUnivariateHistory{}).Size())
		if e := m.originalBudget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		rows := make([]polynomialFactorUnivariateHistory, len(m.factorUnivariateHistory), n)
		copy(rows, m.factorUnivariateHistory)
		m.factorUnivariateHistory = rows
		m.originalBudget.Release(uint64(old) * stride)
	}
	m.factorUnivariateHistory = append(m.factorUnivariateHistory, m.factorUnivariateHistoryRow())
	return nil
}
func (m *MultivariatePolynomialManager) closeFactorUnivariateHistory() {
	m.originalBudget.Release(uint64(cap(m.factorUnivariateHistory)) * uint64(reflect.TypeOf(polynomialFactorUnivariateHistory{}).Size()))
	m.factorUnivariateHistory = nil
}
func numericIntegerManager_factorUnivariateCallerExecuting(m *util.IntegerManager) bool {
	if m == nil || !m.ValidOwner() {
		return false
	}
	f := factorUnivariateCaller(m)
	return f != nil && f.self == f && f.core != nil && f.core.integer == m && f.factorUnivariate.invocation != 0 && (f.factorUnivariate.executing || f.factorUnivariate.finishing)
}

func (f *PolynomialFactors) factorCallerExecuting() bool {
	if f == nil || f.self != f || f.manager == nil || f.manager.self != f.manager {
		return false
	}
	for _, frame := range f.manager.factorFrames {
		if frame.target == f && (frame.executing || frame.finishing) {
			return true
		}
	}
	s := &f.manager.factorUnivariate
	return s.invocation != 0 && s.target == f && (s.executing || s.finishing)
}
