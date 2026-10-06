// Source: src/util/mpf.cpp:1519-1569; src/util/mpq.cpp:156-185.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger,
// Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/memory"

func (c integerAllocationInvocationContext) formatFrame(z *IntegerManager, executing bool) *ArbitraryFloatFormatInvocation {
	s := c.mpfFormat
	if z == nil || z.self != z || s == nil || z.mpfParent == nil || !z.mpfParent.matchesFormatFrame(s) || s.manager.integer != z || c.mpfFormatInvocation == 0 || c.mpfFormatInvocation != s.invocation {
		return nil
	}
	if executing && (!s.active || !s.executing || s.finishing || z.memory.FailureTransportActive()) {
		return nil
	}
	if c.formatDecimal != nil && (c.formatDecimal != &s.decimal || c.formatDecimal.parent != s || (executing && !c.formatDecimal.executing)) {
		return nil
	}
	return s
}
func (s *ArbitraryFloatFormatInvocation) allocationContext() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{mpfFormat: s, mpfFormatInvocation: s.invocation}
}
func (c integerAllocationInvocationContext) recordFormatRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	if s := c.formatFrame(z, true); s != nil {
		r.frame.formatContext = c
		if c.formatDecimal != nil {
			c.formatDecimal.division = *r
			return
		}
		switch r.frame {
		case &z.gcdRaw:
			s.gcd = *r
		case &z.divisionRaw:
			s.division = *r
		case &z.extendedRaw:
			s.extended = *r
		}
	}
}
func (m *ArbitraryFloatManager) formatChildSource(s *ArbitraryFloatFormatInvocation) error {
	z := m.integer
	if e := s.decimal.sourceCause; e != nil {
		return e
	}
	if e := z.rawInvocationSourceCause(&s.extended); e != nil {
		return e
	}
	if e := z.rawInvocationSourceCause(&s.division); e != nil {
		return e
	}
	if e := z.rawInvocationSourceCause(&s.gcd); e != nil {
		return e
	}
	return z.natural.pnVerificationSourceCause(&s.verification)
}
func (m *ArbitraryFloatManager) cleanupFormatChildren(s *ArbitraryFloatFormatInvocation, abort bool) error {
	z := m.integer
	if e := m.cleanupFormatDecimal(&s.decimal, abort); e != nil {
		return e
	}
	for _, r := range []*integerRawInvocationReceipt{&s.extended, &s.division, &s.gcd} {
		if e := z.cleanupRawInvocation(r, abort); e != nil {
			return e
		}
	}
	if e := z.natural.cleanupPNVerification(&s.verification, abort); e != nil {
		return e
	}
	if abort {
		for i := len(s.buffers) - 1; i >= 0; i-- {
			if e := s.buffers[i].cleanup(); e != nil {
				return e
			}
		}
		return z.cleanupFormatAllocation(&s.numeric, &s.transferred, &s.cellRetired)
	}
	return nil
}

func (z *IntegerManager) captureFormatAllocationOutcome(c integerAllocationInvocationContext, backing sourceCellBacking, out *memory.SmallAllocatorAllocationOutcome) error {
	s := c.formatFrame(z, true)
	if s == nil || c.raw != nil {
		return nil
	}
	originalPending := &s.originalPending
	original := &s.originalCell
	transferred := &s.transferred
	roots := &s.numeric
	if c.formatDecimal != nil {
		originalPending = &c.formatDecimal.originalPending
		original = &c.formatDecimal.originalCell
		transferred = &c.formatDecimal.transferred
		roots = &c.formatDecimal.numeric
	}
	if backing.object != (memory.ObjectRef{}) {
		*original = backing
		if z.pendingCell == backing {
			z.pendingCell = sourceCellBacking{}
		}
	}
	if out.Pending != (memory.Block{}) {
		*originalPending = out.Pending
		if e := z.allocator.TransferAllocationPending(out, transferred); e != nil {
			return e
		}
		if roots.allocator[0] == out.Pending {
			roots.allocator[0] = memory.Block{}
		}
	}
	return nil
}
func (z *IntegerManager) cleanupFormatAllocation(roots *polynomialNumericRoots, transferred *memory.SmallAllocatorPendingAllocation, retired *bool) error {
	if roots.cell.object != (memory.ObjectRef{}) {
		if e := z.cleanupDetachedCopyCell(&roots.cell); e != nil {
			return e
		}
		*retired = true
	}
	if transferred.Block() != (memory.Block{}) {
		if e := z.allocator.CleanupTransferredAllocation(transferred); e != nil {
			return e
		}
	}
	if roots.allocator == ([3]memory.Block{}) {
		return nil
	}
	return roots.cleanup(z)
}
