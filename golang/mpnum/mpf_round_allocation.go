// Source: src/util/mpf.cpp:2100-2131 (shared concrete allocation/lifetime engine).
// Source: src/util/mpf.cpp:1963-2098; explicit reached Integer allocation leaves.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Receipt history is materialized only at an actual allocating Integer leaf.
// It carries no cloned NUM and cannot acquire through a reused embedded frame.
type ArbitraryFloatRoundReceipt struct {
	manager    *ArbitraryFloatManager
	frame      *arbitraryFloatRoundHistory
	invocation uint64
}
type arbitraryFloatRoundHistory struct {
	naturalOriginal                  [5]memory.Block
	issued                           ArbitraryFloatRoundReceipt
	owned                            bool
	self                             *arbitraryFloatRoundHistory
	manager                          *ArbitraryFloatManager
	next                             *arbitraryFloatRoundHistory
	invocation, header               uint64
	originalCell                     sourceCellBacking
	originalPending                  memory.Block
	bodyCause, sourceCause, ownCause error
}

func (c integerAllocationInvocationContext) roundFrame(z *IntegerManager, executing bool) *arbitraryFloatRoundScope {
	s := c.mpfRound
	invocation := c.mpfRoundInvocation
	if c.mpfRoundSqrt != nil {
		if s != nil {
			return nil
		}
		s = c.mpfRoundSqrt
		invocation = c.mpfRoundSqrtInvocation
	}
	if z == nil || z.self != z || s == nil || s.manager == nil || s.manager.self != s.manager || s.manager.integer != z || invocation == 0 || invocation != s.invocation {
		return nil
	}
	if c.mpfRoundSqrt != nil {
		if s != &s.manager.roundSqrtScope {
			return nil
		}
	} else if s != &s.manager.roundScope {
		return nil
	}
	if executing && (!s.active || !s.executing || s.finishing || !s.manager.valueExecuting || z.memory.FailureTransportActive()) {
		return nil
	}
	return s
}
func (c integerAllocationInvocationContext) prepareRoundHistory(z *IntegerManager) error {
	if c.mpfRound == nil && c.mpfRoundSqrt == nil {
		return nil
	}
	s := c.roundFrame(z, true)
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.history != nil {
		return nil
	}
	bytes := uint64(reflect.TypeOf(arbitraryFloatRoundHistory{}).Size())
	if e := s.manager.budget.Reserve(bytes); e != nil {
		return e
	}
	h := &arbitraryFloatRoundHistory{manager: s.manager, next: s.manager.roundHistories, invocation: s.invocation, header: bytes}
	h.self = h
	h.issued = ArbitraryFloatRoundReceipt{s.manager, h, s.invocation}
	if s.receipt != nil {
		*s.receipt = h.issued
	}
	s.history = h
	s.manager.roundHistories = h
	s.manager.roundHistoryBytes += bytes
	return nil
}
func (z *IntegerManager) captureRoundAllocationOutcome(c integerAllocationInvocationContext, backing sourceCellBacking, out *memory.SmallAllocatorAllocationOutcome) error {
	s := c.roundFrame(z, true)
	if s == nil {
		return nil
	}
	if backing.object != (memory.ObjectRef{}) {
		s.history.originalCell = backing
		if z.pendingCell == backing {
			z.pendingCell = sourceCellBacking{}
		}
	}
	if out.Pending != (memory.Block{}) {
		s.history.originalPending = out.Pending
		if e := z.allocator.TransferAllocationPending(out, &s.transferred); e != nil {
			return e
		}
		if s.numeric.allocator[0] == out.Pending {
			s.numeric.allocator[0] = memory.Block{}
		}
	}
	return nil
}
func (s *arbitraryFloatRoundScope) recordRoundHistory() {
	if h := s.history; h != nil {
		h.bodyCause = s.bodyCause
		h.sourceCause = s.sourceCause
		h.ownCause = s.ownCause
	}
}
func (m *ArbitraryFloatManager) releaseRoundHistory() {
	for h := m.roundHistories; h != nil; h = h.next {
		m.budget.Release(h.header)
		h.header = 0
	}
	m.roundScope.history = nil
	m.roundScope.receipt = nil
	m.roundSqrtScope.history = nil
	m.roundSqrtScope.receipt = nil
	m.roundHistories = nil
	m.roundHistoryBytes = 0
}

func (m *ArbitraryFloatManager) roundHistory(r *ArbitraryFloatRoundReceipt) (*arbitraryFloatRoundHistory, error) {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame == nil || r.invocation == 0 {
		return nil, memory.ErrRuntimeOwner
	}
	for h := m.roundHistories; h != nil; h = h.next {
		if h == r.frame {
			if h.self != h || h.manager != m || h.issued != *r {
				return nil, memory.ErrRuntimeOwner
			}
			return h, nil
		}
	}
	return nil, memory.ErrRuntimeOwner
}
func (m *ArbitraryFloatManager) RoundInvocationCauses(r *ArbitraryFloatRoundReceipt) (body, source, own error) {
	h, e := m.roundHistory(r)
	if e != nil {
		return e, nil, nil
	}
	return h.bodyCause, h.sourceCause, h.ownCause
}
