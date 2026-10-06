// Source: src/util/mpz.cpp:1871-1929; src/util/mpz.h:820.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

var ErrPerfectSquarePending = errors.New("mpnum: retained perfect-square raw invocation")

// Actual six source RAW numerals, embedded in the charged Integer header. They
// have no automatic destructor: source Del is reached only at the end label.
type integerPerfectSquareInvocationReceipt struct {
	manager    *IntegerManager
	frame      *integerPerfectSquareScope
	invocation uint64
}

type integerPerfectSquareScope struct {
	owner                                    *IntegerManager
	numeric                                  polynomialNumericRoots
	directReceipt                            integerPerfectSquareInvocationReceipt
	parentContext                            integerAllocationInvocationContext
	executing, sourceExecuting, ownExecuting bool
	values                                   [6]Int
	pending                                  memory.ObjectRef
	raw                                      [3]memory.Block
	outputStarted                            bool
	outputChoice                             uint8
	generation                               uint64
	active, endEntered, stopped              bool
	sourceIndex, cleanupIndex                uint8
	bodyCause, sourceCause, cleanupCause     error
}

func (m *IntegerManager) finishPerfectSquare() error {
	s := &m.perfectSquare
	if s.directReceipt.invocation != 0 {
		if !s.executing || s.sourceExecuting || s.ownExecuting {
			return ErrPolynomialPending
		}
		s.sourceExecuting = true
		defer func() { s.sourceExecuting = false }()
	}
	if s.stopped {
		return s.sourceCause
	}
	if !s.active || !s.endEntered {
		return nil
	}
	for s.sourceIndex < 6 {
		if _, e := upolynomialSourceNumeralDel(m, &s.values[s.sourceIndex]); e != nil {
			s.stopped = true
			s.sourceCause = e
			return e
		}
		s.sourceIndex++
	}
	s.active = false
	return nil
}

// CleanupPerfectSquare is an explicit owning abort, never a source end-label
// execution. It resumes only these actual six raw fields in ascending order.
func (m *IntegerManager) CleanupPerfectSquare() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	s := &m.perfectSquare
	if s.directReceipt.invocation != 0 {
		return m.cleanupPerfectSquareInvocation(&s.directReceipt, true)
	}
	if s.directReceipt.invocation != 0 && (s.executing || s.sourceExecuting || s.ownExecuting) {
		return ErrPolynomialPending
	}
	if !s.active {
		return nil
	}
	for s.cleanupIndex < 6 {
		p := &s.values[s.cleanupIndex]
		var e error
		if m.sourceFinalized || m.constructorUnwound {
			if p.cell != 0 && !m.isFailureRef(p.cell) {
				e = m.freeAfterSourceFinalize(p.cell)
			}
			if e == nil {
				p.cell = 0
			}
		} else {
			e = m.Del(p)
		}
		if e != nil {
			s.cleanupCause = e
			return e
		}
		s.cleanupIndex++
	}
	if s.pending != (memory.ObjectRef{}) {
		if m.pendingCell.object == s.pending {
			if e := m.cleanupPendingAllocation(); e != nil {
				s.cleanupCause = e
				return e
			}
		}
		s.pending = memory.ObjectRef{}
	}
	if s.raw != ([3]memory.Block{}) {
		now, e := m.allocator.PendingRoots()
		if e != nil {
			return e
		}
		if now == s.raw {
			e = m.allocator.CleanupPending()
			after, x := m.allocator.PendingRoots()
			s.raw = after
			if e != nil {
				s.cleanupCause = e
				return errors.Join(e, x)
			}
			if x != nil {
				return x
			}
		}
		s.raw = [3]memory.Block{}
	}
	s.active = false
	s.cleanupCause = nil
	return nil
}

// Only actual internal RAW-local operations acquire pending physical roots.
// Final SET(root,lo/hi) is caller-owned and deliberately bypasses this adapter.
func (m *IntegerManager) perfectSquareCall(op uint8, a, b, out *Int) (cause error) {
	before := m.pendingCell.object
	raw, e := m.allocator.PendingRoots()
	if e != nil {
		return e
	}
	switch op {
	case 0:
		cause = m.Copy(out, a)
	case 1:
		cause = m.Add(a, b, out)
	case 2:
		cause = m.Mul(a, b, out)
	case 3:
		cause = m.Div(a, b, out)
	}
	s := &m.perfectSquare
	if before == (memory.ObjectRef{}) && m.pendingCell.object != (memory.ObjectRef{}) {
		s.pending = m.pendingCell.object
	}
	after, e := m.allocator.PendingRoots()
	if raw == ([3]memory.Block{}) && after != ([3]memory.Block{}) {
		s.raw = after
	}
	if e != nil {
		return errors.Join(cause, e)
	}
	return cause
}

func (m *IntegerManager) beginPerfectSquareRawScope() error {
	return m.beginPerfectSquareRawScopeWithContext(integerAllocationInvocationContext{})
}
func (m *IntegerManager) beginPerfectSquareRawScopeWithContext(context integerAllocationInvocationContext) error {
	s := &m.perfectSquare
	if s.active {
		return ErrPerfectSquarePending
	}
	if m.perfectSquareGeneration == ^uint64(0) || (context != (integerAllocationInvocationContext{}) && context.factorFrontendFrame(m) == nil) {
		return ErrPolynomialPending
	}
	m.perfectSquareGeneration++
	*s = integerPerfectSquareScope{owner: m, active: true, generation: m.perfectSquareGeneration, parentContext: context}
	if context != (integerAllocationInvocationContext{}) {
		s.directReceipt = integerPerfectSquareInvocationReceipt{m, s, s.generation}
		context.factorFrontend.perfectSquare = s.directReceipt
	}
	return nil
}
func (m *IntegerManager) endPerfectSquareRawScope() error {
	m.perfectSquare.endEntered = true
	return m.finishPerfectSquare()
}

// Actual direct allocation identity is stamped at the real six-RAW start.
func (m *IntegerManager) matchesPerfectSquareHistory(r *integerPerfectSquareInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame == &m.perfectSquare && r.frame.owner == m && r.invocation != 0 && r.invocation == r.frame.generation && r.frame.directReceipt == *r
}
func (m *IntegerManager) matchesPerfectSquareInvocation(r *integerPerfectSquareInvocationReceipt) bool {
	return m.matchesPerfectSquareHistory(r) && r.frame.active
}
func (m *IntegerManager) perfectSquareContextRoots(c integerAllocationInvocationContext) *polynomialNumericRoots {
	r := c.perfectSquare
	if !m.matchesPerfectSquareInvocation(r) || c.perfectSquareInvocation != r.invocation || !r.frame.executing || r.frame.sourceExecuting || r.frame.ownExecuting {
		return nil
	}
	if r.frame.parentContext != (integerAllocationInvocationContext{}) && r.frame.parentContext.roots(m) == nil {
		return nil
	}
	return &r.frame.numeric
}
func (m *IntegerManager) perfectSquareCallWithContext(op uint8, a, b, out *Int, context integerAllocationInvocationContext) (cause error) {
	if context == (integerAllocationInvocationContext{}) {
		return m.perfectSquareCall(op, a, b, out)
	}
	if m.perfectSquareContextRoots(context) == nil {
		return ErrPolynomialPending
	}
	switch op {
	case 0:
		cause = m.copyFromOwnerWithAllocationContext(out, m, a, nil, context)
	case 1:
		cause = m.addSubWithAllocationContext(a, b, out, false, context)
	case 2:
		cause = m.mulWithAllocationContext(a, b, out, context)
	case 3:
		cause = m.divWithAllocationContext(a, b, out, context)
	}
	s := &m.perfectSquare
	if s.sourceCause == nil && s.parentContext.factorFrontend != nil {
		q := s.parentContext.factorFrontend
		s.sourceCause = m.rawInvocationSourceCause(&q.divisionRaw)
		if s.sourceCause == nil {
			s.sourceCause = q.numericSourceCause()
		}
	}
	return cause
}
func (m *IntegerManager) perfectSquareInvocationSourceCause(r *integerPerfectSquareInvocationReceipt) error {
	if !m.matchesPerfectSquareHistory(r) {
		return nil
	}
	return r.frame.sourceCause
}
func (m *IntegerManager) cleanupPerfectSquareInvocation(r *integerPerfectSquareInvocationReceipt, abort bool) (cause error) {
	if !m.matchesPerfectSquareInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing || s.sourceExecuting || s.ownExecuting {
		return ErrPolynomialPending
	}
	if !abort && !s.endEntered {
		return nil
	}
	s.ownExecuting = true
	defer func() {
		s.ownExecuting = false
		if cause != nil && s.cleanupCause == nil {
			s.cleanupCause = cause
		}
	}()
	for s.cleanupIndex < 6 {
		if e := m.cleanupRawNumeral(&s.values[s.cleanupIndex]); e != nil {
			return e
		}
		s.cleanupIndex++
	}
	if abort {
		if e := cleanupUPolynomialNumericRoots(m, &s.numeric); e != nil {
			return e
		}
		s.active = false
	}
	return nil
}
