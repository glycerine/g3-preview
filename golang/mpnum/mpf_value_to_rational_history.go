// Source: src/util/mpf.cpp:1695-1711; src/util/scoped_numeral.h:27-51.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import "reflect"

import "github.com/glycerine/g3/golang/memory"

// Diagnostic records own no numeral payload. They are materialized only at a
// reached allocation leaf, and remain charged until actual MPF context end.
type arbitraryFloatRationalConversionHistory struct {
	self                             *arbitraryFloatRationalConversionHistory
	manager                          *ArbitraryFloatManager
	next                             *arbitraryFloatRationalConversionHistory
	invocation, header               uint64
	sourceCell, outputCell           sourceCellBacking
	sourcePending, outputPending     memory.Block
	copy                             ArbitraryFloatValueCopyReceipt
	gcd, division, extended          integerRawInvocationReceipt
	bodyCause, sourceCause, ownCause error
}

func (c integerAllocationInvocationContext) prepareConversionHistory(z *IntegerManager) error {
	if c.mpfConversion == nil {
		return nil
	}
	s := c.conversionFrame(z, true)
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.history != nil {
		return nil
	}
	m := s.manager
	n := uint64(reflect.TypeOf(arbitraryFloatRationalConversionHistory{}).Size())
	if e := m.budget.Reserve(n); e != nil {
		return e
	}
	h := &arbitraryFloatRationalConversionHistory{manager: m, next: m.conversionHistory, invocation: s.invocation, header: n}
	h.self = h
	m.conversionHistory = h
	m.conversionHistoryBytes += n
	s.history = h
	return nil
}
func (s *arbitraryFloatRationalConversionScope) recordHistory() {
	if s == nil || s.manager == nil || !s.manager.validConversionScope(s) {
		return
	}
	h := s.history
	if h == nil {
		return
	}
	if h.self != h || h.manager != s.manager || h.invocation != s.invocation {
		return
	}
	if h.sourceCell.object == (memory.ObjectRef{}) {
		h.sourceCell = s.originalCell
	}
	if h.outputCell.object == (memory.ObjectRef{}) {
		h.outputCell = s.outputAllocation.originalCell
	}
	if h.sourcePending == (memory.Block{}) {
		h.sourcePending = s.originalPending
	}
	if h.outputPending == (memory.Block{}) {
		h.outputPending = s.outputAllocation.originalPending
	}
	h.bodyCause = s.bodyCause
	h.sourceCause = s.sourceCause
	h.ownCause = s.ownCause
	h.copy = s.copy
	h.gcd = s.outputGCD
	h.division = s.outputDivision
	h.extended = s.outputExtended
}
func (m *ArbitraryFloatManager) RationalConversionCauses(invocation uint64) (body, source, own error, matched bool) {
	if m == nil || m.self != m || m.closed || invocation == 0 {
		return
	}
	for h := m.conversionHistory; h != nil; h = h.next {
		if h.self != h || h.manager != m || h.header == 0 {
			return
		}
		if h.invocation == invocation {
			return h.bodyCause, h.sourceCause, h.ownCause, true
		}
	}
	s := &m.rationalConversion
	if s.manager == m && s.invocation == invocation {
		return s.bodyCause, s.sourceCause, s.ownCause, true
	}
	return
}
func (m *ArbitraryFloatManager) RationalConversionHistoryBytes() uint64 {
	if m == nil || m.self != m {
		return 0
	}
	return m.conversionHistoryBytes
}
func (m *ArbitraryFloatManager) releaseRationalConversionHistory() {
	for m.conversionHistory != nil {
		h := m.conversionHistory
		m.conversionHistory = h.next
		m.budget.Release(h.header)
		m.conversionHistoryBytes -= h.header
		h.header = 0
	}
}
