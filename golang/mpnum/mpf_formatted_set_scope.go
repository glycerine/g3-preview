// Source: src/util/mpf.cpp:335-374; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

type ArbitraryFloatFormattedSetReceipt struct {
	manager    *ArbitraryFloatManager
	frame      *arbitraryFloatFormattedSetHistory
	invocation uint64
}
type arbitraryFloatFormattedSetScope struct {
	manager                          *ArbitraryFloatManager
	invocation                       uint64
	active, executing, finishing     bool
	receipt                          *ArbitraryFloatFormattedSetReceipt
	history                          *arbitraryFloatFormattedSetHistory
	copy                             ArbitraryFloatValueCopyReceipt
	roundChild                       ArbitraryFloatRoundReceipt
	roundFrame                       *arbitraryFloatRoundScope
	roundInvocation                  uint64
	bodyCause, sourceCause, ownCause error
}
type arbitraryFloatFormattedSetHistory struct {
	self                             *arbitraryFloatFormattedSetHistory
	manager                          *ArbitraryFloatManager
	next                             *arbitraryFloatFormattedSetHistory
	issued                           ArbitraryFloatFormattedSetReceipt
	invocation, header               uint64
	owned                            bool
	numeric                          polynomialNumericRoots
	transferred                      memory.SmallAllocatorPendingAllocation
	cellRetired                      bool
	originalCell                     sourceCellBacking
	originalPending                  memory.Block
	copy                             ArbitraryFloatValueCopyReceipt
	roundChild                       ArbitraryFloatRoundReceipt
	roundFrame                       *arbitraryFloatRoundScope
	roundInvocation                  uint64
	bodyCause, sourceCause, ownCause error
}

func (s *arbitraryFloatFormattedSetScope) allocationContext() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{mpfFormattedSet: s, mpfFormattedSetInvocation: s.invocation}
}
func (c integerAllocationInvocationContext) formattedSetFrame(z *IntegerManager, executing bool) *arbitraryFloatFormattedSetScope {
	s := c.mpfFormattedSet
	if z == nil || z.self != z || s == nil || s.manager == nil || s.manager.self != s.manager || s.manager.integer != z || s != &s.manager.formattedSet || c.mpfFormattedSetInvocation == 0 || c.mpfFormattedSetInvocation != s.invocation {
		return nil
	}
	if executing && (!s.active || !s.executing || s.finishing || s.manager.closed || s.manager.sourceEnded || z.memory.FailureTransportActive()) {
		return nil
	}
	return s
}
func (c integerAllocationInvocationContext) prepareFormattedSetHistory(z *IntegerManager) error {
	if c.mpfFormattedSet == nil {
		return nil
	}
	s := c.formattedSetFrame(z, true)
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.history != nil {
		return nil
	}
	bytes := uint64(reflect.TypeOf(arbitraryFloatFormattedSetHistory{}).Size())
	if e := s.manager.budget.Reserve(bytes); e != nil {
		return e
	}
	h := &arbitraryFloatFormattedSetHistory{manager: s.manager, next: s.manager.formattedSetHistories, invocation: s.invocation, header: bytes}
	h.self = h
	h.issued = ArbitraryFloatFormattedSetReceipt{s.manager, h, s.invocation}
	s.history = h
	s.manager.formattedSetHistories = h
	s.manager.formattedSetHistoryBytes += bytes
	if s.receipt != nil {
		*s.receipt = h.issued
	}
	return nil
}
func (z *IntegerManager) captureFormattedSetAllocation(c integerAllocationInvocationContext, b sourceCellBacking, o *memory.SmallAllocatorAllocationOutcome) error {
	s := c.formattedSetFrame(z, true)
	if s == nil {
		return nil
	}
	if b.object != (memory.ObjectRef{}) {
		s.history.originalCell = b
		if z.pendingCell == b {
			z.pendingCell = sourceCellBacking{}
		}
	}
	if o.Pending != (memory.Block{}) {
		s.history.originalPending = o.Pending
		if e := z.allocator.TransferAllocationPending(o, &s.history.transferred); e != nil {
			return e
		}
		if s.history.numeric.allocator[0] == o.Pending {
			s.history.numeric.allocator[0] = memory.Block{}
		}
	}
	return nil
}
func (s *arbitraryFloatFormattedSetScope) recordHistory() {
	if h := s.history; h != nil {
		h.copy = s.copy
		h.roundChild = s.roundChild
		h.roundFrame = s.roundFrame
		h.roundInvocation = s.roundInvocation
		h.bodyCause = s.bodyCause
		h.sourceCause = s.sourceCause
		h.ownCause = s.ownCause
	}
}
func (m *ArbitraryFloatManager) formattedSetHistory(r *ArbitraryFloatFormattedSetReceipt) (*arbitraryFloatFormattedSetHistory, error) {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame == nil || r.invocation == 0 {
		return nil, memory.ErrRuntimeOwner
	}
	for h := m.formattedSetHistories; h != nil; h = h.next {
		if h == r.frame {
			if h.self != h || h.manager != m || h.invocation != r.invocation || h.issued != *r {
				return nil, memory.ErrRuntimeOwner
			}
			return h, nil
		}
	}
	return nil, memory.ErrRuntimeOwner
}
func (m *ArbitraryFloatManager) FormattedSetInvocationCauses(r *ArbitraryFloatFormattedSetReceipt) (body, source, own error) {
	h, e := m.formattedSetHistory(r)
	if e != nil {
		return e, nil, nil
	}
	return h.bodyCause, h.sourceCause, h.ownCause
}
func (m *ArbitraryFloatManager) formattedRoundMatches(s *arbitraryFloatFormattedSetScope) bool {
	return m != nil && m.self == m && s == &m.formattedSet && s.manager == m && s.invocation != 0 && s.roundFrame == &m.roundScope && s.roundInvocation != 0 && s.roundInvocation == m.roundScope.invocation
}
func (m *ArbitraryFloatManager) formattedRoundSourceCause(s *arbitraryFloatFormattedSetScope) error {
	if m == nil || m.self != m || s != &m.formattedSet || s.manager != m || s.invocation == 0 {
		return memory.ErrRuntimeOwner
	}
	if s.roundChild.frame != nil {
		_, source, _ := m.RoundInvocationCauses(&s.roundChild)
		return source
	}
	if m.formattedRoundMatches(s) {
		return m.roundScope.sourceCause
	}
	return nil
}
func (m *ArbitraryFloatManager) formattedRoundCleanup(s *arbitraryFloatFormattedSetScope, abort bool) error {
	if m == nil || m.self != m || s != &m.formattedSet || s.manager != m || s.invocation == 0 {
		return memory.ErrRuntimeOwner
	}
	if s.roundChild.frame != nil {
		return m.CleanupRoundWithReceipt(&s.roundChild, abort)
	}
	if m.formattedRoundMatches(s) {
		return m.CleanupRoundWithReceipt(nil, abort)
	}
	return nil
}
func (m *ArbitraryFloatManager) finishFormattedSetSource() error {
	s := &m.formattedSet
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active {
		return nil
	}
	if s.executing || s.finishing {
		return ErrPolynomialPending
	}
	if e := m.formattedRoundSourceCause(s); e != nil {
		s.sourceCause = e
		s.recordHistory()
		return e
	}
	// No lexical NUM/RAW was declared in this body. Exact unjoined
	// allocations remain in their leaf-created call records, independently
	// of the next embedded execution binding.
	s.active = m.formattedRoundMatches(s) && (m.roundScope.sigmaConstructed || m.roundScope.capConstructed || m.roundScope.stickyConstructed)
	s.recordHistory()
	return nil
}
func (m *ArbitraryFloatManager) CleanupFormattedSet() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	for h := m.formattedSetHistories; h != nil; h = h.next {
		if e := m.CleanupFormattedSetWithReceipt(&h.issued, false); e != nil {
			return e
		}
		if e := m.CleanupFormattedSetWithReceipt(&h.issued, true); e != nil {
			return e
		}
	}
	return m.CleanupFormattedSetWithReceipt(nil, true)
}
func (m *ArbitraryFloatManager) CleanupFormattedSetWithReceipt(r *ArbitraryFloatFormattedSetReceipt, abort bool) error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	s := &m.formattedSet
	if s.executing || s.finishing || m.valueExecuting || m.memory != nil && m.memory.FailureTransportActive() {
		return ErrPolynomialPending
	}
	var h *arbitraryFloatFormattedSetHistory
	if r != nil {
		if *r == (ArbitraryFloatFormattedSetReceipt{}) {
			return nil
		}
		var e error
		h, e = m.formattedSetHistory(r)
		if e != nil {
			return e
		}
	} else {
		h = s.history
	}
	if h != nil && h.owned {
		return nil
	}
	if h == nil && s.manager == nil {
		return nil
	}
	s.finishing = true
	defer func() { s.finishing = false }()
	stop := func(e error) error {
		if e != nil {
			if h != nil && h.ownCause == nil {
				h.ownCause = e
			}
			if (h == nil || s.history == h) && s.ownCause == nil {
				s.ownCause = e
			}
		}
		return e
	}
	copyReceipt := &s.copy
	roundReceipt := &s.roundChild
	roundFrame, roundInvocation := s.roundFrame, s.roundInvocation
	if h != nil {
		copyReceipt = &h.copy
		roundReceipt = &h.roundChild
		roundFrame, roundInvocation = h.roundFrame, h.roundInvocation
	}
	cleanupRound := func(own bool) error {
		if roundReceipt.frame != nil {
			return m.CleanupRoundWithReceipt(roundReceipt, own)
		}
		if roundFrame == &m.roundScope && roundInvocation != 0 && roundInvocation == m.roundScope.invocation {
			return m.CleanupRoundWithReceipt(nil, own)
		}
		return nil
	}
	if e := cleanupRound(false); e != nil {
		return stop(e)
	}
	if e := m.CleanupValueCopy(copyReceipt, false); e != nil {
		return stop(e)
	}
	if !abort {
		return nil
	}
	if e := cleanupRound(true); e != nil {
		return stop(e)
	}
	if e := m.CleanupValueCopy(copyReceipt, true); e != nil {
		return stop(e)
	}
	if h != nil {
		if e := m.integer.cleanupFormatAllocation(&h.numeric, &h.transferred, &h.cellRetired); e != nil {
			return stop(e)
		}
		h.owned = true
	}
	if h == nil || s.history == h {
		s.active = false
	}
	return nil
}
func (m *ArbitraryFloatManager) releaseFormattedSetHistories() {
	for h := m.formattedSetHistories; h != nil; h = h.next {
		m.budget.Release(h.header)
		h.header = 0
	}
	m.formattedSetHistories = nil
	m.formattedSetHistoryBytes = 0
	m.formattedSet.history = nil
	m.formattedSet.receipt = nil
}
func (m *ArbitraryFloatManager) roundWithFormattedSet(mode ArbitraryFloatRoundingMode, out *ArbitraryFloat, s *arbitraryFloatFormattedSetScope) error {
	return m.roundWithParentContext(mode, out, &s.roundChild, s.allocationContext())
}
