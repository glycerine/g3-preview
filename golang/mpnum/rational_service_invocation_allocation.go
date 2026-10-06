// Source: src/util/rational.h; src/util/mpz.h:allocate.
// Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/memory"

func (c *RationalServiceInvocation) allocationContext() integerAllocationInvocationContext {
	if c == nil {
		return integerAllocationInvocationContext{}
	}
	return integerAllocationInvocationContext{rationalService: c, rationalServiceInvocation: c.invocation}
}
func (c integerAllocationInvocationContext) rationalServiceFrame(z *IntegerManager) *RationalServiceInvocation {
	s := c.rationalService
	if z == nil || z.self != z || s == nil || !s.valid() || s.manager == nil || s.integer != z || s.manager.z != z || s.service.manager != s.manager || c.rationalServiceInvocation == 0 || c.rationalServiceInvocation != s.invocation || !s.executing || s.service.lexicalExecuting != s || z.memory.FailureTransportActive() {
		return nil
	}
	return s
}
func (z *IntegerManager) captureRationalServiceAllocationOutcome(c integerAllocationInvocationContext, backing sourceCellBacking, out *memory.SmallAllocatorAllocationOutcome) error {
	s := c.rationalServiceFrame(z)
	if s == nil || c.raw != nil {
		return nil
	}
	if backing.object != (memory.ObjectRef{}) {
		s.originalCell = backing
		if z.pendingCell == backing {
			z.pendingCell = sourceCellBacking{}
		}
	}
	if out.Pending != (memory.Block{}) {
		s.originalPending = out.Pending
		if e := z.allocator.TransferAllocationPending(out, &s.transferred); e != nil {
			return e
		}
		if s.numeric.allocator[0] == out.Pending {
			s.numeric.allocator[0] = memory.Block{}
		}
	}
	return nil
}

func (c *RationalServiceInvocation) childSource() error {
	if c == nil {
		return nil
	}
	r := &c.manager.rawArithmetic
	if c.normGeneration != 0 && c.normGeneration == r.normGeneration && r.normEnd && r.normCause != nil {
		return r.normCause
	}
	if c.addGeneration != 0 && c.addGeneration == r.addGeneration && r.addEnd && r.addCause != nil {
		return r.addCause
	}
	for _, r := range []*integerRawInvocationReceipt{&c.extended, &c.division, &c.gcd} {
		if e := c.integer.rawInvocationSourceCause(r); e != nil {
			return e
		}
	}
	return nil
}
func (c integerAllocationInvocationContext) recordRationalServiceRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	if s := c.rationalServiceFrame(z); s != nil {
		r.frame.formatContext = c
		switch r.frame {
		case &z.divisionRaw:
			s.division = *r
		case &z.gcdRaw:
			s.gcd = *r
		case &z.extendedRaw:
			s.extended = *r
		}
	}
}

func (c *RationalServiceInvocation) cleanupRationalRaw(abort bool) error {
	r := &c.manager.rawArithmetic
	if c.normGeneration != 0 && c.normGeneration == r.normGeneration && r.normActive && (abort || r.normEnd) {
		if e := c.integer.cleanupRawNumeral(&r.norm); e != nil {
			return e
		}
		r.normActive = false
	}
	if c.addGeneration != 0 && c.addGeneration == r.addGeneration && r.addActive && (abort || r.addEnd) {
		for r.addCursor < 2 {
			p := &r.addOne
			if r.addCursor == 1 {
				p = &r.addTwo
			}
			if e := c.integer.cleanupRawNumeral(p); e != nil {
				return e
			}
			r.addCursor++
		}
		r.addActive = false
	}
	return nil
}
