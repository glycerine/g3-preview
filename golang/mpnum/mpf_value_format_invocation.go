// Source: src/util/mpf.cpp:1519-1569; src/util/mpq.cpp:156-185.
// Licensed under the MIT license. Source author: Christoph Wintersteiger;
// rational display source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// ArbitraryFloatFormatInvocation is real cold continuation storage embedded in
// the exact text context. NUM fields acquire constructor credit only when their
// source declarations are reached; the text owner charges this actual record.
type ArbitraryFloatFormatInvocation struct {
	originalPending                               memory.Block
	self                                          *ArbitraryFloatFormatInvocation
	manager                                       *ArbitraryFloatManager
	next                                          *ArbitraryFloatFormatInvocation
	invocation                                    uint64
	active, executing, finishing, returned, owned bool
	constructed                                   uint8
	sourceCursor, ownCursor                       uint8
	num, denom                                    Int
	q                                             Rational
	stream                                        ArbitraryFloatFormatStream
	text                                          ArbitraryFloatFormatText
	numeric                                       polynomialNumericRoots
	originalCell                                  sourceCellBacking
	cellRetired                                   bool
	transferred                                   memory.SmallAllocatorPendingAllocation
	gcd, division, extended                       integerRawInvocationReceipt
	buffers                                       [4]naturalPNBufferReceipt
	verification                                  naturalPNVerificationReceipt
	decimal                                       rationalDecimalScope
	bodyCause, sourceCause, ownCause              error
}

func (m *ArbitraryFloatManager) knownFormatFrame(s *ArbitraryFloatFormatInvocation) bool {
	if m == nil || m.self != m || s == nil {
		return false
	}
	for q := m.formatScopes; q != nil; q = q.next {
		if q == s {
			return true
		}
	}
	return false
}
func (m *ArbitraryFloatManager) matchesFormatFrame(s *ArbitraryFloatFormatInvocation) bool {
	return m.knownFormatFrame(s) && s.self == s && s.manager == m && s.invocation != 0
}
func (m *ArbitraryFloatManager) formatInvocationExecuting() bool {
	if m == nil || m.self != m {
		return false
	}
	for s := m.formatScopes; s != nil; s = s.next {
		if m.matchesFormatFrame(s) && (s.executing || s.finishing) {
			return true
		}
	}
	return false
}
func (m *ArbitraryFloatManager) beginFormatInvocation(s *ArbitraryFloatFormatInvocation, text ArbitraryFloatFormatText) error {
	if m == nil || m.self != m || m.integer == nil || m.integer.self != m.integer {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.sourceEnded || !m.constructorComplete {
		return arena.ErrClosed
	}

	if s == nil || text == nil {
		return memory.ErrRuntimeOwner
	}
	if m.formatInvocationExecuting() || m.valueExecuting {
		return ErrPolynomialPending
	}
	if s.self != nil || s.invocation != 0 || m.formatInvocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	m.formatInvocation++
	*s = ArbitraryFloatFormatInvocation{self: s, manager: m, next: m.formatScopes, invocation: m.formatInvocation, active: true, executing: true, text: text}
	m.formatScopes = s
	return nil
}

// FormatInvocationCauses is immutable invocation history, not acquisition.
func (m *ArbitraryFloatManager) FormatInvocationCauses(s *ArbitraryFloatFormatInvocation) (body, source, own error) {
	if !m.matchesFormatFrame(s) {
		return memory.ErrRuntimeOwner, nil, nil
	}
	return s.bodyCause, s.sourceCause, s.ownCause
}

// CleanupFormatInvocation acknowledges only the exact caller-owned invocation.
// False acknowledges completed children and the older local owning suffix.
// True aborts unjoined child roots only
// after the older scoped q/denom/num cleanup; source history remains untouched.
func (m *ArbitraryFloatManager) CleanupFormatInvocation(s *ArbitraryFloatFormatInvocation, abort bool) error {
	if !m.matchesFormatFrame(s) {
		return memory.ErrRuntimeOwner
	}
	if s.executing || s.finishing || m.valueExecuting || m.memory.FailureTransportActive() {
		return ErrPolynomialPending
	}
	if s.owned {
		return nil
	}
	s.finishing = true
	defer func() { s.finishing = false }()
	stop := func(e error) error {
		if e != nil && s.ownCause == nil {
			s.ownCause = e
		}
		return e
	}
	if e := m.cleanupFormatChildren(s, false); e != nil {
		return stop(e)
	}
	for s.ownCursor < 5 {
		var e error
		switch s.ownCursor {
		case 0:
			if s.stream != nil {
				e = s.stream.Close()
			}
		case 1:
			if s.constructed >= 3 {
				e = m.integer.Del(&s.q.num)
			}
		case 2:
			if s.constructed >= 3 {
				e = m.integer.Del(&s.q.den)
			}
		case 3:
			if s.constructed >= 2 {
				e = m.integer.Del(&s.denom)
			}
		case 4:
			if s.constructed >= 1 {
				e = m.integer.Del(&s.num)
			}
		}
		if e != nil {
			return stop(e)
		}
		s.ownCursor++
	}
	if !abort {
		return nil
	}
	if e := m.cleanupFormatChildren(s, true); e != nil {
		return stop(e)
	}
	s.owned = true
	s.active = false
	return nil
}

// EndFormatContext retires directory metadata only at the actual text context
// owner end, after all exact owning actions. It does not perform source DT.
func (m *ArbitraryFloatManager) EndFormatContext(s *ArbitraryFloatFormatInvocation) error {
	if !m.matchesFormatFrame(s) {
		return memory.ErrRuntimeOwner
	}
	if !s.owned || s.executing || s.finishing {
		return ErrPolynomialPending
	}
	link := &m.formatScopes
	for *link != nil {
		if *link == s {
			*link = s.next
			s.next = nil
			return nil
		}
		link = &(*link).next
	}
	return memory.ErrRuntimeOwner
}

// The text owner uses these methods at its actual source/owning end. An unused
// context has no reached numerical invocation and therefore no NUM authority.
func (s *ArbitraryFloatFormatInvocation) SourceCause() error {
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.manager == nil && s.self == nil && s.invocation == 0 {
		return nil
	}
	if s.manager == nil || !s.manager.matchesFormatFrame(s) {
		return memory.ErrRuntimeOwner
	}
	return s.sourceCause
}
func (s *ArbitraryFloatFormatInvocation) Cleanup(abort bool) error {
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.manager == nil && s.self == nil && s.invocation == 0 {
		return nil
	}
	if s.manager == nil {
		return memory.ErrRuntimeOwner
	}
	return s.manager.CleanupFormatInvocation(s, abort)
}
func (s *ArbitraryFloatFormatInvocation) EndContext() error {
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.manager == nil && s.self == nil && s.invocation == 0 {
		return nil
	}
	if s.manager == nil {
		return memory.ErrRuntimeOwner
	}
	return s.manager.EndFormatContext(s)
}

// SourceReady checks the literal younger callee before the text owner starts
// destroying res. It is distinct from the readonly source-history query.
func (s *ArbitraryFloatFormatInvocation) SourceReady() error {
	if s == nil {
		return memory.ErrRuntimeOwner
	}
	if s.manager == nil && s.self == nil && s.invocation == 0 {
		return nil
	}
	if s.manager == nil || !s.manager.matchesFormatFrame(s) {
		return memory.ErrRuntimeOwner
	}
	if s.executing || s.finishing || s.manager.memory.FailureTransportActive() {
		return ErrPolynomialPending
	}
	return s.sourceCause
}
