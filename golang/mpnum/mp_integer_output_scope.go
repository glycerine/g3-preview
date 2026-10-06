// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpz.cpp:1385-1410; src/util/buffer.h:84-95.
// Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
)

type integerCharacterScope struct {
	caller                      *RationalServiceInvocation
	self                        *integerCharacterScope
	owner                       *NaturalManager
	display                     *upolyDisplayScope
	displayInvocation           uint64
	formatting                  *naturalFormattingScope
	previous                    *integerCharacterScope
	callExecuting               bool
	buffer                      sourceCharBuffer
	invocation                  uint64
	active, completed, pending  bool
	originalCause, cleanupCause error
}

func (m *NaturalManager) beginIntegerCharacters(n uint32) error {
	return m.beginIntegerCharactersTracked(n, nil, nil)
}
func (m *NaturalManager) beginIntegerCharactersTracked(n uint32, service *RationalService, receipt *NumeralInsertionReceipt) error {
	return m.beginIntegerCharactersWithDisplay(n, service, receipt, nil)
}
func (m *NaturalManager) beginIntegerCharactersWithDisplay(n uint32, service *RationalService, receipt *NumeralInsertionReceipt, display *upolyDisplayScope) error {
	return m.beginIntegerCharacterScope(n, service, receipt, display, false)
}
func (m *NaturalManager) beginIntegerCharactersForInsertion(n uint32, service *RationalService, receipt *NumeralInsertionReceipt, display *upolyDisplayScope) error {
	return m.beginIntegerCharacterScope(n, service, receipt, display, true)
}
func (m *NaturalManager) beginIntegerCharacterScope(n uint32, service *RationalService, receipt *NumeralInsertionReceipt, display *upolyDisplayScope, executing bool) error {
	return m.beginIntegerCharacterScopeWithCaller(n, service, receipt, display, executing, nil)
}
func (m *NaturalManager) beginIntegerCharactersWithCaller(n uint32, service *RationalService, receipt *NumeralInsertionReceipt, scope *integerCharacterScope, caller *RationalServiceInvocation) error {
	if caller == nil || scope != &caller.characters {
		return arena.ErrInvalidSlot
	}
	return m.beginIntegerCharacterScopeWithCaller(n, service, receipt, nil, true, caller)
}
func (m *NaturalManager) beginIntegerCharacterScopeWithCaller(n uint32, service *RationalService, receipt *NumeralInsertionReceipt, display *upolyDisplayScope, executing bool, caller *RationalServiceInvocation) error {
	if e := m.check(); e != nil {
		return e
	}
	s := &m.charFormatting
	formatting := &m.formatting
	if display != nil {
		if !display.actual() || !display.executing || display.owner.config.Integer.natural != m || display.owner.currentDisplay != display {
			return ErrUPolynomialOwner
		}
		s = &display.characters
		formatting = &display.characterFormatting
	}
	if caller != nil {
		if !caller.valid() || !caller.executing || caller.integer.natural != m || display != nil {
			return arena.ErrInvalidSlot
		}
		s = &caller.characters
		formatting = &caller.characterFormatting
	}
	if s.active || s.pending {
		return arena.ErrInvalidSlot
	}
	if s.invocation == math.MaxUint64 {
		return arena.ErrBounds
	}
	if display != nil {
		if e := display.prepareStorage(); e != nil {
			return e
		}
	}
	*s = integerCharacterScope{caller: caller, owner: m, display: display, formatting: formatting, active: true, invocation: s.invocation + 1}
	s.self = s
	if display != nil {
		s.displayInvocation = display.invocation
	}
	if receipt != nil {
		*receipt = NumeralInsertionReceipt{service: service, natural: m, character: s, invocation: s.invocation, displayInvocation: s.displayInvocation, active: true, characterOwned: true}
	}
	if executing {
		m.beginIntegerCharacterCall(s)
	}
	if e := s.buffer.resize(m.runtime, n); e != nil {
		s.active = false
		s.pending = true
		s.originalCause = e
		return e // failed constructor's destructor must not run
	}
	s.completed = true
	if receipt != nil {
		receipt.characterCompleted = true
	}
	return nil
}
func (m *NaturalManager) destroyIntegerCharacters() error {
	return m.destroyIntegerCharacterScope(&m.charFormatting)
}
func (m *NaturalManager) destroyIntegerCharacterScope(s *integerCharacterScope) error {
	if !s.completed {
		return nil
	}
	if s.buffer.raw != (memory.Block{}) {
		if e := m.runtime.Deallocate(&s.buffer.raw); e != nil {
			s.active = false
			s.pending = true
			s.cleanupCause = e
			if s.originalCause == nil {
				s.originalCause = e
			}
			return e
		}
	}
	s.completed = false
	s.active = false
	if s.buffer.pending != (memory.Block{}) {
		s.pending = true
		return arena.ErrInvalidSlot
	}
	s.reset()
	return nil
}

func (s *integerCharacterScope) reset() {
	*s = integerCharacterScope{caller: s.caller, self: s.self, owner: s.owner, display: s.display, displayInvocation: s.displayInvocation, formatting: s.formatting, previous: s.previous, callExecuting: s.callExecuting, invocation: s.invocation}
}

func (m *NaturalManager) matchesIntegerCharacterScope(s *integerCharacterScope) bool {
	if m == nil || m.self != m || s == nil || s.self != s || s.owner != m {
		return false
	}
	if s == &m.charFormatting {
		return s.display == nil && s.formatting == &m.formatting
	}
	if c := s.caller; c != nil {
		return c.valid() && c.integer.natural == m && s == &c.characters && s.formatting == &c.characterFormatting
	}
	q := s.display
	return q != nil && q.actual() && q.owner.config.Integer.natural == m && s == &q.characters && s.displayInvocation != 0 && s.displayInvocation == q.invocation && s.formatting == &q.characterFormatting
}

func (m *NaturalManager) beginIntegerCharacterCall(s *integerCharacterScope) {
	s.previous = m.characterCalls
	s.callExecuting = true
	m.characterCalls = s
}
func (m *NaturalManager) endIntegerCharacterCall(s *integerCharacterScope) {
	if m.characterCalls == s && s.callExecuting {
		m.characterCalls = s.previous
		s.previous = nil
		s.callExecuting = false
	}
}

// cleanupOutputFormatting first resumes completed source local destructors,
// then the completed outer char destructor. Only afterwards does owning
// cleanup reclaim actual failed-constructor/unpublished allocations.
func (m *NaturalManager) cleanupOutputFormatting() error {
	if m.formatting.active || m.formatting.completed != 0 {
		if e := m.unwindFormattingLocals(); e != nil {
			return e
		}
		m.formatting.active = false
		m.formatting.pending = true
	}
	if e := m.destroyIntegerCharacters(); e != nil {
		return e
	}
	if e := m.CleanupFormatting(); e != nil {
		return e
	}
	if m.charFormatting.pending {
		if e := m.charFormatting.buffer.finalize(m.runtime); e != nil {
			m.charFormatting.cleanupCause = e
			return e
		}
		m.charFormatting = integerCharacterScope{invocation: m.charFormatting.invocation}
	}
	return nil
}

func (m *NaturalManager) appendIntegerCString(offset uint32) (uint32, error) {
	s := &m.charFormatting
	var n uint64
	for {
		v, e := s.buffer.readPhysical(m.runtime, n)
		if e != nil {
			return 0, e
		}
		if v == 0 {
			break
		}
		n++
	}
	end, e := checkedTextEnd(offset, n)
	if e != nil {
		return 0, e
	}
	// This retained byte assembly is an explicit Go output adapter, not the
	// source character buffer or a fictitious fresh source allocation.
	if e = m.reserveText(end); e != nil {
		return 0, e
	}
	for i := uint64(0); i < n; i++ {
		v, e := s.buffer.readPhysical(m.runtime, i)
		if e != nil {
			return 0, e
		}
		m.textWrite(offset+uint32(i), v)
	}
	return end, nil
}
