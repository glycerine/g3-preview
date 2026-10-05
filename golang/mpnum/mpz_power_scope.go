// Source: src/util/mpz.cpp:1453-1507; src/util/mpz.h:778.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

var ErrPowerPending = errors.New("mpnum: retained integer power RAW invocation")

// One actual RAW numeral field are charged in the enclosing Integer header.
// Body errors do not execute the source end-only ascending Del sequence.
type integerPowerScope struct {
	value                                Int
	pending                              sourceCellBacking
	raw                                  [3]memory.Block
	generation                           uint64
	active, endEntered, stopped          bool
	sourceIndex, cleanupIndex            uint8
	bodyCause, sourceCause, cleanupCause error
}

func (m *IntegerManager) finishPower() error {
	s := &m.powerRaw
	if s.stopped {
		return s.sourceCause
	}
	if !s.active || !s.endEntered {
		return nil
	}
	for s.sourceIndex < 1 {
		if _, e := upolynomialSourceNumeralDel(m, &s.value); e != nil {
			s.stopped = true
			s.sourceCause = e
			return e
		}
		s.sourceIndex++
	}
	s.active = false
	return nil
}

// CleanupPower is an explicit owning abort, never a source end-label
// execution. It resumes only this actual raw field in ascending order.
func (m *IntegerManager) CleanupPower() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	s := &m.powerRaw
	if !s.active {
		return nil
	}
	for s.cleanupIndex < 1 {
		p := &s.value
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
	if s.pending.object != (memory.ObjectRef{}) {
		if m.pendingCell == s.pending {
			if e := m.cleanupPendingAllocation(); e != nil {
				s.pending = m.pendingCell
				s.cleanupCause = e
				return e
			}
		}
		s.pending = sourceCellBacking{}
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

// Only internal RAW COPY and final-square acquire local pending roots.
// Output SET/MUL stay caller-owned, including a stopped late publication.
func (m *IntegerManager) powerRawCall(op uint8, a, b, out *Int) (cause error) {
	before := m.pendingCell
	raw, e := m.allocator.PendingRoots()
	if e != nil {
		return e
	}
	switch op {
	case 0:
		cause = m.Copy(out, a)
	case 1:
		cause = m.Mul(a, b, out)
	default:
		return ErrIntegerAssertion
	}
	s := &m.powerRaw
	if before.object == (memory.ObjectRef{}) && m.pendingCell.object != (memory.ObjectRef{}) {
		s.pending = m.pendingCell
	}
	after, e := m.allocator.PendingRoots()
	if raw == ([3]memory.Block{}) && after != ([3]memory.Block{}) {
		s.raw = after
	}
	return errors.Join(cause, e)
}

type integerPowerInvocationReceipt struct {
	manager    *IntegerManager
	scope      *integerPowerScope
	invocation uint64
}

func (m *IntegerManager) matchesPowerInvocation(r *integerPowerInvocationReceipt) bool {
	return r != nil && r.manager == m && r.scope == &m.powerRaw && r.invocation != 0 && r.scope.generation == r.invocation && r.scope.active
}
func (m *IntegerManager) powerInvocationSourceCause(r *integerPowerInvocationReceipt) error {
	if m.matchesPowerInvocation(r) && r.scope.stopped {
		return r.scope.sourceCause
	}
	return nil
}
func (m *IntegerManager) cleanupPowerInvocation(r *integerPowerInvocationReceipt, abort bool) error {
	if !m.matchesPowerInvocation(r) {
		return nil
	}
	if !abort && !r.scope.endEntered {
		return nil
	}
	return m.CleanupPower()
}
func (m *IntegerManager) beginPowerRawScope() error { return m.beginPowerRawScopeWithReceipt(nil) }
func (m *IntegerManager) beginPowerRawScopeWithReceipt(receipt *integerPowerInvocationReceipt) error {
	s := &m.powerRaw
	if s.active {
		return ErrPowerPending
	}
	if m.powerGeneration == ^uint64(0) {
		return ErrPowerPending
	}
	m.powerGeneration++
	*s = integerPowerScope{active: true, generation: m.powerGeneration}
	if receipt != nil {
		*receipt = integerPowerInvocationReceipt{m, s, s.generation}
	}
	return nil
}
func (m *IntegerManager) endPowerRawScope() error {
	m.powerRaw.endEntered = true
	return m.finishPower()
}
