// Source: src/util/mpf.cpp:1695-1711,1922-1945; src/util/mpz.h:133-164.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import "github.com/glycerine/g3/golang/memory"

func (c integerAllocationInvocationContext) conversionFrame(z *IntegerManager, executing bool) *arbitraryFloatRationalConversionScope {
	s := c.mpfConversion
	if z == nil || z.self != z || s == nil || s.manager == nil || s.manager.self != s.manager || s != &s.manager.rationalConversion || c.mpfConversionInvocation == 0 || c.mpfConversionInvocation != s.invocation {
		return nil
	}
	if c.mpfConversionOutput {
		if s.outputManager == nil || s.outputManager.self != s.outputManager || s.outputManager.z != z {
			return nil
		}
	} else if s.manager.integer != z {
		return nil
	}

	if executing && (!s.active || !s.executing || s.finishing || s.manager.closed || s.manager.sourceEnded || z.memory.FailureTransportActive()) {
		return nil
	}
	return s
}
func (s *arbitraryFloatRationalConversionScope) allocationContext() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{mpfConversion: s, mpfConversionInvocation: s.invocation}
}

// The exact allocation outcome is recorded only at the actual failed leaf.
// This is a transfer of an issued root, never a later allocator census.
func (s *arbitraryFloatRationalConversionScope) outputAllocationContext() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{mpfConversion: s, mpfConversionInvocation: s.invocation, mpfConversionOutput: true}
}

func (z *IntegerManager) captureConversionAllocationOutcome(c integerAllocationInvocationContext, backing sourceCellBacking, out *memory.SmallAllocatorAllocationOutcome) error {
	s := c.conversionFrame(z, true)
	if s == nil || c.raw != nil {
		return nil
	}
	original, pending, transferred, roots := &s.originalCell, &s.originalPending, &s.transferred, &s.numeric
	if c.mpfConversionOutput {
		o := &s.outputAllocation
		original, pending, transferred, roots = &o.originalCell, &o.originalPending, &o.transferred, &o.numeric
	}
	if backing.object != (memory.ObjectRef{}) {
		*original = backing
		if c.mpfConversionOutput {
			s.bindDetachedOutput(z)
		}
		if z.pendingCell == backing {
			z.pendingCell = sourceCellBacking{}
		}
	}
	if out.Pending != (memory.Block{}) {
		*pending = out.Pending
		if e := z.allocator.TransferAllocationPending(out, transferred); e != nil {
			return e
		}
		if roots.allocator[0] == out.Pending {
			roots.allocator[0] = memory.Block{}
		}
	}
	return nil
}

func (c integerAllocationInvocationContext) recordConversionRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	s := c.conversionFrame(z, true)
	if s == nil || !c.mpfConversionOutput {
		return
	}
	r.frame.conversionContext = c
	switch r.frame {
	case &z.gcdRaw:
		s.outputGCD = *r
	case &z.divisionRaw:
		s.outputDivision = *r
	case &z.extendedRaw:
		s.outputExtended = *r
	}
}
