// Source: src/util/mpf.cpp:233-300; src/util/mpz.h:156-164.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

type ArbitraryFloatRationalSetReceipt struct {
	manager    *ArbitraryFloatManager
	frame      *arbitraryFloatRationalSetHistory
	invocation uint64
}
type arbitraryFloatRationalSetHistory struct {
	naturalOriginal                  [5]memory.Block
	issued                           ArbitraryFloatRationalSetReceipt
	gcd, division, extended          integerRawInvocationReceipt
	roundChild                       ArbitraryFloatRoundReceipt
	roundFrame                       *arbitraryFloatRoundScope
	roundInvocation                  uint64
	self                             *arbitraryFloatRationalSetHistory
	manager                          *ArbitraryFloatManager
	next                             *arbitraryFloatRationalSetHistory
	invocation, header               uint64
	numeric                          polynomialNumericRoots
	transferred                      memory.SmallAllocatorPendingAllocation
	cellRetired, owned               bool
	originalCell                     sourceCellBacking
	originalPending                  memory.Block
	prevSourceCause                  error
	bodyCause, sourceCause, ownCause error
}

func (s *arbitraryFloatRationalSetScope) allocationContext() integerAllocationInvocationContext {
	return integerAllocationInvocationContext{mpfRationalSet: s, mpfRationalSetInvocation: s.invocation}
}
func (c integerAllocationInvocationContext) rationalSetFrame(z *IntegerManager, executing bool) *arbitraryFloatRationalSetScope {
	s := c.mpfRationalSet
	if z == nil || z.self != z || s == nil || s.manager == nil || s.manager.self != s.manager || s.manager.integer != z || s != &s.manager.rationalSet || c.mpfRationalSetInvocation == 0 || c.mpfRationalSetInvocation != s.invocation {
		return nil
	}
	if executing && (!s.active || !s.executing || s.finishing || s.manager.closed || s.manager.sourceEnded || !s.manager.constructorComplete || !s.manager.rationalReady || z.closed || z.sourceFinalizing || z.constructionFailed || z.memory == nil || z.memory.FailureTransportActive()) {
		return nil
	}
	return s
}
func (c integerAllocationInvocationContext) prepareRationalSetHistory(z *IntegerManager) error {
	if c.mpfRationalSet == nil {
		return nil
	}
	s := c.rationalSetFrame(z, true)
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.history != nil {
		return nil
	}
	bytes := uint64(reflect.TypeOf(arbitraryFloatRationalSetHistory{}).Size())
	if e := s.manager.budget.Reserve(bytes); e != nil {
		return e
	}
	h := &arbitraryFloatRationalSetHistory{manager: s.manager, next: s.manager.rationalSetHistories, invocation: s.invocation, header: bytes}
	h.self = h
	h.issued = ArbitraryFloatRationalSetReceipt{s.manager, h, s.invocation}
	if s.receipt != nil {
		*s.receipt = h.issued
	}
	s.history = h
	s.manager.rationalSetHistories = h
	s.manager.rationalSetHistoryBytes += bytes
	return nil
}
func (z *IntegerManager) captureRationalSetAllocation(c integerAllocationInvocationContext, b sourceCellBacking, o *memory.SmallAllocatorAllocationOutcome) error {
	s := c.rationalSetFrame(z, true)
	if s == nil || c.raw != nil {
		return nil
	}
	h := s.history
	if b.object != (memory.ObjectRef{}) {
		h.originalCell = b
		if z.pendingCell == b {
			z.pendingCell = sourceCellBacking{}
		}
	}
	if o.Pending != (memory.Block{}) {
		h.originalPending = o.Pending
		if e := z.allocator.TransferAllocationPending(o, &h.transferred); e != nil {
			return e
		}
		if h.numeric.allocator[0] == o.Pending {
			h.numeric.allocator[0] = memory.Block{}
		}
	}
	return nil
}
func (s *arbitraryFloatRationalSetScope) recordHistory() {
	if h := s.history; h != nil {
		h.prevSourceCause = s.prevSourceCause
		h.gcd = s.gcd
		h.division = s.division
		h.extended = s.extended
		h.roundChild = s.roundChild
		h.roundFrame = s.roundFrame
		h.roundInvocation = s.roundInvocation
		h.bodyCause = s.bodyCause
		h.sourceCause = s.sourceCause
		h.ownCause = s.ownCause
	}
}
func (c integerAllocationInvocationContext) recordRationalSetRaw(z *IntegerManager, r *integerRawInvocationReceipt) {
	s := c.rationalSetFrame(z, true)
	if s == nil {
		return
	}
	r.frame.rationalSetContext = c
	switch r.frame {
	case &z.gcdRaw:
		s.gcd = *r
	case &z.divisionRaw:
		s.division = *r
	case &z.extendedRaw:
		s.extended = *r
	}
}
func (m *ArbitraryFloatManager) rationalSetChildrenSource(s *arbitraryFloatRationalSetScope) error {
	if e := m.rationalSetRoundSource(s); e != nil {
		return e
	}
	if s.prevSourceCause != nil {
		return s.prevSourceCause
	}
	if e := m.integer.natural.pnVerificationSourceCause(&s.verification); e != nil {
		return e
	}
	if e := m.integer.rawInvocationSourceCause(&s.extended); e != nil {
		return e
	}
	if e := m.integer.rawInvocationSourceCause(&s.division); e != nil {
		return e
	}
	return m.integer.rawInvocationSourceCause(&s.gcd)
}
func (m *ArbitraryFloatManager) cleanupRationalSetChildren(s *arbitraryFloatRationalSetScope, abort bool) error {
	if e := m.rationalSetRoundCleanup(s, abort); e != nil {
		return e
	}
	if e := m.integer.natural.cleanupPNVerification(&s.verification, abort); e != nil {
		return e
	}
	if abort {
		for i := len(s.buffers) - 1; i >= 0; i-- {
			if e := s.buffers[i].cleanup(); e != nil {
				return e
			}
		}
	}
	if e := m.integer.cleanupRawInvocation(&s.extended, abort); e != nil {
		return e
	}
	if e := m.integer.cleanupRawInvocation(&s.division, abort); e != nil {
		return e
	}
	return m.integer.cleanupRawInvocation(&s.gcd, abort)
}
func (m *ArbitraryFloatManager) releaseRationalSetHistories() {
	for h := m.rationalSetHistories; h != nil; h = h.next {
		m.budget.Release(h.header)
		h.header = 0
	}
	m.rationalSetHistories = nil
	m.rationalSetHistoryBytes = 0
	m.rationalSet.history = nil
}

func (m *ArbitraryFloatManager) rationalSetRoundSource(s *arbitraryFloatRationalSetScope) error {
	if s.roundChild.frame != nil {
		_, source, _ := m.RoundInvocationCauses(&s.roundChild)
		return source
	}
	if s.roundFrame == &m.roundScope && s.roundInvocation != 0 && s.roundInvocation == m.roundScope.invocation {
		return m.roundScope.sourceCause
	}
	return nil
}
func (m *ArbitraryFloatManager) rationalSetRoundCleanup(s *arbitraryFloatRationalSetScope, abort bool) error {
	if s.roundChild.frame != nil {
		return m.CleanupRoundWithReceipt(&s.roundChild, abort)
	}
	if s.roundFrame == &m.roundScope && s.roundInvocation != 0 && s.roundInvocation == m.roundScope.invocation {
		return m.CleanupRoundWithReceipt(nil, abort)
	}
	return nil
}

func (m *ArbitraryFloatManager) rationalSetHistory(r *ArbitraryFloatRationalSetReceipt) (*arbitraryFloatRationalSetHistory, error) {
	if m == nil || m.self != m || r == nil || r.manager != m || r.frame == nil || r.invocation == 0 {
		return nil, memory.ErrRuntimeOwner
	}
	for h := m.rationalSetHistories; h != nil; h = h.next {
		if h == r.frame {
			if h.self != h || h.manager != m || h.invocation != r.invocation || h.issued != *r {
				return nil, memory.ErrRuntimeOwner
			}
			return h, nil
		}
	}
	return nil, memory.ErrRuntimeOwner
}
func (m *ArbitraryFloatManager) ExponentRationalSetInvocationCauses(r *ArbitraryFloatRationalSetReceipt) (body, source, own error) {
	h, e := m.rationalSetHistory(r)
	if e != nil {
		return e, nil, nil
	}
	return h.bodyCause, h.sourceCause, h.ownCause
}

func (m *ArbitraryFloatManager) cleanupRationalSetHistory(h *arbitraryFloatRationalSetHistory, abort bool) error {
	if h == nil || h.self != h || h.manager != m || h.invocation == 0 {
		return memory.ErrRuntimeOwner
	}
	if h.owned {
		return nil
	}
	s := &m.rationalSet
	if s.executing || m.integer.memory.FailureTransportActive() {
		return ErrPolynomialPending
	}
	stop := func(e error) error {
		if e != nil && h.ownCause == nil {
			h.ownCause = e
		}
		return e
	}
	round := func(own bool) error {
		if h.roundChild.frame != nil {
			return m.CleanupRoundWithReceipt(&h.roundChild, own)
		}
		if h.roundFrame == &m.roundScope && h.roundInvocation != 0 && h.roundInvocation == m.roundScope.invocation {
			return m.CleanupRoundWithReceipt(nil, own)
		}
		return nil
	}
	if e := round(false); e != nil {
		return stop(e)
	}
	for _, r := range []*integerRawInvocationReceipt{&h.extended, &h.division, &h.gcd} {
		if e := m.integer.cleanupRawInvocation(r, false); e != nil {
			return stop(e)
		}
	}
	if !abort {
		return nil
	}
	if e := round(true); e != nil {
		return stop(e)
	}
	for _, r := range []*integerRawInvocationReceipt{&h.extended, &h.division, &h.gcd} {
		if e := m.integer.cleanupRawInvocation(r, true); e != nil {
			return stop(e)
		}
	}
	if e := m.integer.cleanupFormatAllocation(&h.numeric, &h.transferred, &h.cellRetired); e != nil {
		return stop(e)
	}
	h.owned = true
	return nil
}
