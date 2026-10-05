// Source: src/util/mpz.cpp:1942-2021; src/util/mpz.h:836-837.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

var ErrRootPending = errors.New("mpnum: retained integer root RAW invocation")

// Four actual RAW numeral fields are charged in the enclosing Integer header.
// Body errors do not execute the source end-only ascending Del sequence.
type integerRootScope struct {
	powerInvocation                      integerPowerInvocationReceipt
	values                               [4]Int
	pending                              sourceCellBacking
	raw                                  [3]memory.Block
	generation                           uint64
	active, endEntered, stopped          bool
	sourceIndex, cleanupIndex            uint8
	bodyCause, sourceCause, cleanupCause error
}

func (m *IntegerManager) finishRoot() error {
	s := &m.rootRaw
	if s.stopped {
		return s.sourceCause
	}
	if !s.active || !s.endEntered {
		return nil
	}
	for s.sourceIndex < 4 {
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

// CleanupRoot is an explicit owning abort, never a source end-label
// execution. It resumes only these actual four raw fields in ascending order.
func (m *IntegerManager) CleanupRoot() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	s := &m.rootRaw
	if e := m.cleanupPowerInvocation(&s.powerInvocation, false); e != nil {
		return e
	}
	if !s.active {
		return nil
	}
	for s.cleanupIndex < 4 {
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
	if e := m.cleanupPowerInvocation(&s.powerInvocation, true); e != nil {
		return e
	}
	s.active = false
	s.cleanupCause = nil
	return nil
}

// Only actual internal RAW-local operations acquire pending physical roots.
// Final SET(root,lo/hi) is caller-owned and deliberately bypasses this adapter.
func (m *IntegerManager) rootRawCall(op uint8, a, b, out *Int) (cause error) {
	before := m.pendingCell.object
	raw, e := m.allocator.PendingRoots()
	if e != nil {
		return e
	}
	switch op {
	case 0:
		cause = m.powerWithReceipt(a, uint32(b.val), out, &m.rootRaw.powerInvocation)
	case 1:
		cause = m.Add(a, b, out)
	case 2:
		cause = m.machineDiv2kCore(out, uint32(b.val))
	case 3:
		cause = m.Div(a, b, out)
	}
	s := &m.rootRaw
	if before == (memory.ObjectRef{}) && m.pendingCell.object != (memory.ObjectRef{}) {
		if !m.matchesPowerInvocation(&s.powerInvocation) || m.powerRaw.pending != m.pendingCell {
			s.pending = m.pendingCell
		}
	}
	after, e := m.allocator.PendingRoots()
	if raw == ([3]memory.Block{}) && after != ([3]memory.Block{}) {
		if !m.matchesPowerInvocation(&s.powerInvocation) || m.powerRaw.raw != after {
			s.raw = after
		}
	}
	if e != nil {
		return errors.Join(cause, e)
	}
	return cause
}

type integerRootInvocationReceipt struct {
	manager    *IntegerManager
	scope      *integerRootScope
	invocation uint64
}

func (m *IntegerManager) matchesRootInvocation(r *integerRootInvocationReceipt) bool {
	return r != nil && r.manager == m && r.scope == &m.rootRaw && r.invocation != 0 && r.scope.generation == r.invocation && r.scope.active
}
func (m *IntegerManager) rootInvocationSourceCause(r *integerRootInvocationReceipt) error {
	if m.matchesRootInvocation(r) {
		if e := m.powerInvocationSourceCause(&r.scope.powerInvocation); e != nil {
			return e
		}
	}
	if m.matchesRootInvocation(r) && r.scope.stopped {
		return r.scope.sourceCause
	}
	return nil
}
func (m *IntegerManager) cleanupRootInvocation(r *integerRootInvocationReceipt, abort bool) error {
	if !m.matchesRootInvocation(r) {
		return nil
	}
	if e := m.cleanupPowerInvocation(&r.scope.powerInvocation, false); e != nil {
		return e
	}
	if !abort && !r.scope.endEntered {
		return nil
	}
	return m.CleanupRoot()
}
func (m *IntegerManager) beginRootRawScope() error { return m.beginRootRawScopeWithReceipt(nil) }
func (m *IntegerManager) beginRootRawScopeWithReceipt(receipt *integerRootInvocationReceipt) error {
	s := &m.rootRaw
	if s.active {
		return ErrRootPending
	}
	if m.rootGeneration == ^uint64(0) {
		return ErrRootPending
	}
	m.rootGeneration++
	*s = integerRootScope{active: true, generation: m.rootGeneration}
	if receipt != nil {
		*receipt = integerRootInvocationReceipt{m, s, s.generation}
	}
	return nil
}
func (m *IntegerManager) endRootRawScope() error {
	m.rootRaw.endEntered = true
	return m.finishRoot()
}
