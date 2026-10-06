// Source: src/util/mpz.cpp:big_set; src/util/mpz.h:allocate.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

type integerCopyAllocationScope struct {
	invocation        uint64
	active, executing bool
	roots             polynomialNumericRoots
	mpfCaller         *arbitraryFloatValueCopyCall
	transferred       memory.SmallAllocatorPendingAllocation
	originalCell      sourceCellBacking
	cellRetired       bool
}
type integerCopyAllocationReceipt struct {
	manager    *IntegerManager
	frame      *integerCopyAllocationScope
	invocation uint64
}

func (m *IntegerManager) validCopyAllocationScope(s *integerCopyAllocationScope) bool {
	if m == nil || m.self != m || s == nil {
		return false
	}
	if s == &m.copyAllocation {
		return true
	}
	q := s.mpfCaller
	if q == nil || q.self != q || s != &q.childFrame || q.header == 0 || q.retired {
		return false
	}
	owner := q.manager
	if owner == nil || owner.self != owner || owner.closed || owner.integer != m || q.issued.manager != owner || q.issued.frame != q || q.issued.invocation == 0 || q.issued.child.manager != m || q.issued.child.frame != s || q.issued.child.invocation != s.invocation {
		return false
	}
	for old := owner.valueCalls; old != nil; old = old.next {
		if old == q {
			return true
		}
	}
	return false
}

func (m *IntegerManager) executingCopyAllocationScope(s *integerCopyAllocationScope) bool {
	if !m.validCopyAllocationScope(s) || !s.active || !s.executing {
		return false
	}
	if s.mpfCaller == nil {
		return true
	}
	q := s.mpfCaller
	c := &q.manager.valueBinding
	return q.executing && c.record == q && c.invocation == s.invocation && m.validateFloatCopyBinding(c) == nil
}

func (m *IntegerManager) matchesCopyAllocationInvocation(r *integerCopyAllocationReceipt) bool {
	return r != nil && r.manager == m && m.validCopyAllocationScope(r.frame) && r.invocation != 0 && r.frame.invocation == r.invocation && r.frame.active
}
func (m *IntegerManager) cleanupCopyAllocationInvocation(r *integerCopyAllocationReceipt) error {
	if !m.matchesCopyAllocationInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing || (s.mpfCaller != nil && m.memory.FailureTransportActive()) {
		return ErrPolynomialPending
	}
	if s.mpfCaller != nil {
		if s.roots.cell.object != (memory.ObjectRef{}) {
			e := m.cleanupDetachedCopyCell(&s.roots.cell)
			if s.roots.cell.object == (memory.ObjectRef{}) {
				s.cellRetired = true
			}
			if e != nil {
				return e
			}
		}
		if s.transferred.Block() != (memory.Block{}) {
			e := m.allocator.CleanupTransferredAllocation(&s.transferred)
			s.roots.allocator[0] = s.transferred.Block()
			if e != nil {
				return e
			}
		}
	} else if e := s.roots.cleanup(m); e != nil {
		return e
	}
	*s = integerCopyAllocationScope{invocation: s.invocation, mpfCaller: s.mpfCaller, originalCell: s.originalCell, cellRetired: s.cellRetired || s.originalCell.object != (memory.ObjectRef{})}
	return nil
}

// A descriptor-preparation failure retains this exact backing outside cells.
// Source chunk destruction may consume pooled backing before its own suffix
// stops. Direct backing survives allocator destruction and belongs to caller.
func (m *IntegerManager) cleanupDetachedCopyCell(c *sourceCellBacking) error {
	if c.offset != 0 {
		if _, e := m.memory.Size(c.raw); errors.Is(e, memory.ErrRuntimeOwner) {
			*c = sourceCellBacking{}
			return nil
		} else if e != nil {
			return e
		}
	}
	ended, e := m.allocator.SourceObjectOwnerEnded()
	if e != nil {
		return e
	}
	if ended && c.offset == 0 {
		raw := c.raw
		e = m.memory.Deallocate(&raw)
		if raw == (memory.Block{}) {
			*c = sourceCellBacking{}
		}
		return e
	}
	if c.offset == 0 {
		raw := c.raw
		handled, e := m.allocator.CleanupSourceRetainedDirect(c.object, &raw)
		if handled {
			if raw == (memory.Block{}) {
				*c = sourceCellBacking{}
			}
			return e
		}
	}
	progress, e := c.deallocate(m.allocator)
	if progress.PhysicalReleased || progress.PooledLinked {
		*c = sourceCellBacking{}
	}
	return e
}
