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
type integerPerfectSquareScope struct {
	values                               [6]Int
	pending                              memory.ObjectRef
	raw                                  [3]memory.Block
	outputStarted                        bool
	outputChoice                         uint8
	generation                           uint64
	active, endEntered, stopped          bool
	sourceIndex, cleanupIndex            uint8
	bodyCause, sourceCause, cleanupCause error
}

func (m *IntegerManager) finishPerfectSquare() error {
	s := &m.perfectSquare
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
	s := &m.perfectSquare
	if s.active {
		return ErrPerfectSquarePending
	}
	m.perfectSquareGeneration++
	*s = integerPerfectSquareScope{active: true, generation: m.perfectSquareGeneration}
	return nil
}
func (m *IntegerManager) endPerfectSquareRawScope() error {
	m.perfectSquare.endEntered = true
	return m.finishPerfectSquare()
}
