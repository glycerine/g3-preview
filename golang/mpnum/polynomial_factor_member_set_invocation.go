// Source: src/math/polynomial/polynomial.cpp:1613-1642.
// The reached constructor/reset SET writes the existing class NUM member.
// Its adapter creates no source numeral or source destructor.
package mpnum

import (
	"errors"
)

type polynomialFactorMemberSetReceipt struct {
	owner      *PolynomialFactors
	frame      *polynomialFactorMemberSetScope
	invocation uint64
}
type polynomialFactorMemberSetScope struct {
	invocation                           uint64
	historyIndex                         int
	historyRecorded                      bool
	active, executing, finishing         bool
	direct                               polynomialFactorMemberSetReceipt
	numeric                              polynomialNumericRoots
	normalize                            integerPNormalizeInvocationReceipt
	bodyCause, sourceCause, cleanupCause error
}
type polynomialFactorMemberSetHistory struct {
	invocation                           uint64
	normalize                            integerPNormalizeInvocationReceipt
	bodyCause, sourceCause, cleanupCause error
}

func (f *PolynomialFactors) matchesFactorMemberSet(r *polynomialFactorMemberSetReceipt) bool {
	return f != nil && f.self == f && f.manager != nil && f.manager.self == f.manager && f.manager.core != nil && f.manager.core.self == f.manager.core && r != nil && r.owner == f && r.frame == &f.memberSet && r.invocation != 0 && r.invocation == f.memberSet.invocation
}
func (m *IntegerManager) factorMemberSetRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	r := c.factorMemberSet
	if m == nil || m.self != m || r == nil || r.owner == nil || !r.owner.matchesFactorMemberSet(r) || c.factorMemberSetInvocation == 0 || c.factorMemberSetInvocation != r.invocation || r.owner.manager.core.integer != m || !r.frame.active || !r.frame.executing || r.owner.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (f *PolynomialFactors) factorMemberSetPending() bool {
	return polynomialNegNumericPending(&f.memberSet.numeric) || f.manager.core.modular.matchesPNormalizeInvocation(&f.memberSet.normalize)
}
func (f *PolynomialFactors) finishFactorMemberSetSource() error {
	s := &f.memberSet
	if s.sourceCause != nil {
		return s.sourceCause
	}
	s.finishing = true
	defer func() { s.finishing = false; f.updateFactorMemberSetHistory() }()
	if e := f.manager.core.modular.pNormalizeInvocationSourceCause(&s.normalize); e != nil {
		s.sourceCause = e
		return e
	}
	if !f.factorMemberSetPending() {
		s.active = false
	}
	return nil
}
func (f *PolynomialFactors) setMemberConstantOne() (cause error) {
	s := &f.memberSet
	if s.active || s.executing || s.finishing || s.invocation == ^uint64(0) || f.manager.core.callbackDepth != 0 {
		return ErrPolynomialPending
	}

	*s = polynomialFactorMemberSetScope{invocation: s.invocation + 1, active: true, executing: true}
	s.direct = polynomialFactorMemberSetReceipt{f, s, s.invocation}
	context := integerAllocationInvocationContext{factorMemberSet: &s.direct, factorMemberSetInvocation: s.invocation}
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = errors.Join(cause, f.finishFactorMemberSetSource())
	}()
	return f.invoke(func() error {
		z := f.manager.core.integer
		prior := z.factorNumeralExecuting
		z.factorNumeralExecuting = true
		defer func() { z.factorNumeralExecuting = prior }()
		return f.manager.core.modular.setInt32WithSourceContext(&f.constant, 1, context)
	})
}
func (f *PolynomialFactors) cleanupFactorMemberSet(abort bool) (cause error) {
	s := &f.memberSet
	if !s.active {
		return nil
	}
	if !f.matchesFactorMemberSet(&s.direct) {
		return nil
	}
	if s.executing || s.finishing || f.manager.core.callbackDepth != 0 {
		return ErrPolynomialPending
	}
	s.finishing = true
	defer func() {
		s.finishing = false
		if cause != nil && s.cleanupCause == nil {
			s.cleanupCause = cause
		}
		f.updateFactorMemberSetHistory()
	}()
	if e := f.manager.core.modular.cleanupPNormalizeInvocation(&s.normalize, abort); e != nil {
		return e
	}
	if abort {
		if e := s.numeric.cleanup(f.manager.core.integer); e != nil {
			return e
		}
	}
	if !f.factorMemberSetPending() {
		s.active = false
	}
	return nil
}
