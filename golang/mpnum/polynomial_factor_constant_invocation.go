// Source: src/math/polynomial/polynomial.cpp:6577-6590;
// src/math/polynomial/polynomial.h:153-157; src/util/scoped_numeral.h:31-33.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"reflect"
)

type polynomialFactorConstantInvocationReceipt struct {
	manager    *PolynomialFactors
	frame      *polynomialFactorConstantScope
	invocation uint64
}
type polynomialFactorConstantScope struct {
	invocation                                     uint64
	historyIndex                                   int
	historyRecorded                                bool
	active, executing, finishing, localConstructed bool
	kind, phase                                    uint8
	numeric                                        polynomialNumericRoots
	normalize                                      [3]integerPNormalizeInvocationReceipt
	direct                                         polynomialFactorConstantInvocationReceipt
	bodyCause, sourceCause, cleanupCause           error
}
type polynomialFactorConstantHistory struct {
	invocation                           uint64
	kind                                 uint8
	normalize                            [3]integerPNormalizeInvocationReceipt
	bodyCause, sourceCause, cleanupCause error
}

func (f *PolynomialFactors) matchesFactorConstantInvocation(r *polynomialFactorConstantInvocationReceipt) bool {
	return f != nil && f.self == f && f.manager != nil && f.manager.self == f.manager && f.manager.core != nil && f.manager.core.self == f.manager.core && r != nil && r.manager == f && r.frame == &f.constantInvocation && r.invocation != 0 && r.invocation == r.frame.invocation
}
func (f *PolynomialFactors) factorConstantDiagnostic(r *polynomialFactorConstantInvocationReceipt) (polynomialFactorConstantHistory, bool) {
	if f.matchesFactorConstantInvocation(r) {
		s := r.frame
		return polynomialFactorConstantHistory{s.invocation, s.kind, s.normalize, s.bodyCause, s.sourceCause, s.cleanupCause}, true
	}
	if f == nil || f.self != f || f.manager == nil || f.manager.self != f.manager || f.manager.core == nil || f.manager.core.self != f.manager.core || r == nil || r.manager != f || r.frame != &f.constantInvocation || r.invocation == 0 {
		return polynomialFactorConstantHistory{}, false
	}
	for i := range f.constantHistory {
		if f.constantHistory[i].invocation == r.invocation {
			return f.constantHistory[i], true
		}
	}
	return polynomialFactorConstantHistory{}, false
}
func (f *PolynomialFactors) archiveFactorConstantHistory() error {
	s := &f.constantInvocation
	if s.historyRecorded {
		return nil
	}
	if len(f.constantHistory) == cap(f.constantHistory) {
		old := cap(f.constantHistory)
		next := old + old/2 + 1
		stride := uint64(reflect.TypeOf(polynomialFactorConstantHistory{}).Size())
		if e := f.manager.core.integer.budget.Reserve(uint64(next) * stride); e != nil {
			return e
		}
		rows := make([]polynomialFactorConstantHistory, len(f.constantHistory), next)
		copy(rows, f.constantHistory)
		f.constantHistory = rows
		f.manager.core.integer.budget.Release(uint64(old) * stride)
	}
	s.historyIndex = len(f.constantHistory)
	s.historyRecorded = true
	f.constantHistory = append(f.constantHistory, polynomialFactorConstantHistory{s.invocation, s.kind, s.normalize, s.bodyCause, s.sourceCause, s.cleanupCause})
	return nil
}
func (f *PolynomialFactors) beginFactorConstant(kind uint8, r *polynomialFactorConstantInvocationReceipt) (*polynomialFactorConstantScope, error) {
	if e := f.open(); e != nil {
		return nil, e
	}
	s := &f.constantInvocation
	if s.active || s.executing || s.finishing || s.invocation == ^uint64(0) || f.manager.core.callbackDepth != 0 {
		return nil, ErrPolynomialPending
	}
	*s = polynomialFactorConstantScope{invocation: s.invocation + 1, kind: kind, active: true, executing: true}
	s.direct = polynomialFactorConstantInvocationReceipt{f, s, s.invocation}
	if r != nil {
		*r = s.direct
	}
	return s, nil
}
func (m *IntegerManager) factorConstantContextRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	r := c.factorConstant
	if m == nil || m.self != m || r == nil || r.manager == nil || !r.manager.matchesFactorConstantInvocation(r) || c.factorConstantInvocation == 0 || c.factorConstantInvocation != r.invocation || r.manager.manager.core.integer != m || !r.frame.active || !r.frame.executing || r.manager.manager.core.callbackDepth != 0 {
		return nil
	}
	return &r.frame.numeric
}
func (f *PolynomialFactors) factorConstantNumericContext() integerAllocationInvocationContext {
	s := &f.constantInvocation
	return integerAllocationInvocationContext{factorConstant: &s.direct, factorConstantInvocation: s.invocation}
}
func (f *PolynomialFactors) factorConstantPending() bool {
	s := &f.constantInvocation
	if polynomialNegNumericPending(&s.numeric) {
		return true
	}
	for i := range s.normalize {
		if f.manager.core.modular.matchesPNormalizeInvocation(&s.normalize[i]) {
			return true
		}
	}
	return false
}
func (f *PolynomialFactors) finishFactorConstantSource() error {
	s := &f.constantInvocation
	if s.sourceCause != nil {
		return s.sourceCause
	}
	s.finishing = true
	defer func() { s.finishing = false; f.updateFactorConstantHistory() }()
	for i := len(s.normalize) - 1; i >= 0; i-- {
		if e := f.manager.core.modular.pNormalizeInvocationSourceCause(&s.normalize[i]); e != nil {
			s.sourceCause = e
			return e
		}
	}
	if s.localConstructed {
		if _, e := upolynomialSourceNumeralDel(f.manager.core.integer, &f.local); e != nil {
			s.sourceCause = e
			return e
		}
		s.localConstructed = false
		f.localActive = false
	}
	if !f.factorConstantPending() {
		s.active = false
	}
	return nil
}
func (f *PolynomialFactors) cleanupFactorConstantInvocation(r *polynomialFactorConstantInvocationReceipt, abort bool) (cause error) {
	if !f.matchesFactorConstantInvocation(r) {
		return nil
	}
	s := r.frame
	if !s.active {
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
		f.updateFactorConstantHistory()
	}()
	for i := len(s.normalize) - 1; i >= 0; i-- {
		if e := f.manager.core.modular.cleanupPNormalizeInvocation(&s.normalize[i], false); e != nil {
			return e
		}
	}
	// This is separate owning acknowledgment of the helper's actual scoped NUM.
	if s.localConstructed {
		if e := f.manager.core.integer.Del(&f.local); e != nil {
			return e
		}
		s.localConstructed = false
		f.localActive = false
	}
	if abort {
		// Unjoined child allocation roots are drained only after the helper's
		// actual older scoped NUM has completed its owning acknowledgment.
		for i := len(s.normalize) - 1; i >= 0; i-- {
			if e := f.manager.core.modular.cleanupPNormalizeInvocation(&s.normalize[i], true); e != nil {
				return e
			}
		}
		if e := s.numeric.cleanup(f.manager.core.integer); e != nil {
			return e
		}
	}
	if !f.factorConstantPending() {
		s.active = false
	}
	return nil
}
func (m *MultivariatePolynomialManager) accumulateFactorConstantWithReceipt(f *PolynomialFactors, a *Int, r *polynomialFactorConstantInvocationReceipt) (cause error) {
	if e := m.ready(); e != nil {
		return e
	}
	if f == nil || f.manager != m {
		return ErrPolynomialOwner
	}
	s, e := f.beginFactorConstant(1, r)
	if e != nil {
		return e
	}
	f.local = Int{}
	f.localActive = true
	s.localConstructed = true
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = errors.Join(cause, f.finishFactorConstantSource())
	}()
	return f.invoke(func() error {
		if e := m.core.modular.mulWithFactorConstantReceipt(&f.constant, a, &f.local, &s.direct); e != nil {
			return e
		}
		s.phase = 1
		return m.core.modular.copyWithFactorConstantReceipt(&f.constant, &f.local, &s.direct)
	})
}
func (m *MultivariatePolynomialManager) flipFactorSignWithReceipt(f *PolynomialFactors, r *polynomialFactorConstantInvocationReceipt) (cause error) {
	if e := m.ready(); e != nil {
		return e
	}
	if f == nil || f.manager != m {
		return ErrPolynomialOwner
	}
	s, e := f.beginFactorConstant(2, r)
	if e != nil {
		return e
	}
	f.local = Int{}
	f.localActive = true
	s.localConstructed = true
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = errors.Join(cause, f.finishFactorConstantSource())
	}()
	return f.invoke(func() error {
		if e := m.core.modular.copyWithFactorConstantReceipt(&f.local, &f.constant, &s.direct); e != nil {
			return e
		}
		s.phase = 1
		if e := m.core.modular.negWithFactorConstantReceipt(&f.local, &s.direct); e != nil {
			return e
		}
		s.phase = 2
		return m.core.modular.copyWithFactorConstantReceipt(&f.constant, &f.local, &s.direct)
	})
}
func (f *PolynomialFactors) setConstantWithReceipt(a *Int, r *polynomialFactorConstantInvocationReceipt) (cause error) {
	s, e := f.beginFactorConstant(0, r)
	if e != nil {
		return e
	}
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = errors.Join(cause, f.finishFactorConstantSource())
	}()
	return f.invoke(func() error {
		return f.manager.core.modular.copyWithFactorConstantReceipt(&f.constant, a, &s.direct)
	})
}

func (f *PolynomialFactors) factorConstantInvocationSourceCause(r *polynomialFactorConstantInvocationReceipt) error {
	h, ok := f.factorConstantDiagnostic(r)
	if !ok {
		return nil
	}
	return h.sourceCause
}

// This is a lifecycle refusal only; acquisition remains exclusively in the
// explicitly passed concrete receipt at the actual numerical leaf.
func (m *MultivariatePolynomialCore) factorConstantExecuting() bool {
	if m == nil || m.self != m {
		return false
	}
	f := m.factorsOwner
	return f != nil && f.self == f && f.manager != nil && f.manager.core == m && ((f.matchesFactorConstantInvocation(&f.constantInvocation.direct) && (f.constantInvocation.executing || f.constantInvocation.finishing)) || (f.matchesFactorMemberSet(&f.memberSet.direct) && (f.memberSet.executing || f.memberSet.finishing)))
}

func (m *IntegerModularManager) factorConstantAllocationContext(r *polynomialFactorConstantInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.self != m || r == nil || r.manager == nil || !r.manager.matchesFactorConstantInvocation(r) || r.manager.manager.core.modular != m || !r.frame.active || !r.frame.executing || r.manager.manager.core.callbackDepth != 0 {
		return integerAllocationInvocationContext{}
	}
	return r.manager.factorConstantNumericContext()
}
func (m *IntegerModularManager) copyWithFactorConstantReceipt(out, a *Int, r *polynomialFactorConstantInvocationReceipt) error {
	context := m.factorConstantAllocationContext(r)
	if r != nil {
		if context.factorConstant == nil {
			return ErrPolynomialOwner
		}
		if m.manager.factorNumeralExecuting {
			return ErrPolynomialPending
		}
	}
	if context.factorConstant != nil {
		previous := m.manager.factorNumeralExecuting
		m.manager.factorNumeralExecuting = true
		defer func() { m.manager.factorNumeralExecuting = previous }()
	}
	return m.copyWithSourceContext(out, a, context)
}
func (m *IntegerModularManager) mulWithFactorConstantReceipt(a, b, out *Int, r *polynomialFactorConstantInvocationReceipt) error {
	context := m.factorConstantAllocationContext(r)
	if r != nil {
		if context.factorConstant == nil {
			return ErrPolynomialOwner
		}
		if m.manager.factorNumeralExecuting {
			return ErrPolynomialPending
		}
	}
	if context.factorConstant != nil {
		previous := m.manager.factorNumeralExecuting
		m.manager.factorNumeralExecuting = true
		defer func() { m.manager.factorNumeralExecuting = previous }()
	}
	return m.mulWithSourceContext(a, b, out, context)
}
func (m *IntegerModularManager) negWithFactorConstantReceipt(out *Int, r *polynomialFactorConstantInvocationReceipt) error {
	context := m.factorConstantAllocationContext(r)
	if r != nil {
		if context.factorConstant == nil {
			return ErrPolynomialOwner
		}
		if m.manager.factorNumeralExecuting {
			return ErrPolynomialPending
		}
	}
	if context.factorConstant != nil {
		previous := m.manager.factorNumeralExecuting
		m.manager.factorNumeralExecuting = true
		defer func() { m.manager.factorNumeralExecuting = previous }()
	}
	return m.negWithSourceContext(out, context)
}
