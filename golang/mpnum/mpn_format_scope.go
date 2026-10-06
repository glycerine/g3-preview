// Licensed under the MIT license.
// Source: src/util/mpn.cpp:371-399; src/util/buffer.h:84-95.
// Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
)

// Embedded cold ownership is charged by NaturalManager's actual header. Each
// invocation starts fresh scalar buffer lifetimes, without allocating a Go
// scope. Failed constructors do not run the source object's destructor.
// Their actual raw/pending blocks survive for explicitly named owning cleanup.
type naturalFormattingScope struct {
	invocation                  uint64
	locals                      [3]sourceLimbBuffer
	completed                   uint8
	failedLocal                 uint8
	active, pending             bool
	originalCause, cleanupCause error
}

func (m *NaturalManager) beginFormattingLocals(n uint32) error {
	return m.beginFormattingScope(&m.formatting, n)
}
func (m *NaturalManager) beginFormattingScope(s *naturalFormattingScope, n uint32) error {
	return m.beginFormattingScopeTracked(s, n, nil)
}
func (m *NaturalManager) beginFormattingScopeTracked(s *naturalFormattingScope, n uint32, receipt *NumeralInsertionReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if s.active || s.pending {
		return arena.ErrInvalidSlot
	}
	if s.invocation == math.MaxUint64 {
		return arena.ErrBounds
	}
	*s = naturalFormattingScope{active: true, invocation: s.invocation + 1}
	if receipt != nil {
		receipt.formatting = s
		receipt.formattingOwned = true
		receipt.formattingInvocation = s.invocation
	}
	sizes := [3]uint32{n, n + 1, 1} // source size_t addition narrowed by mpn_sbuffer
	for i, size := range sizes {
		if e := s.locals[i].resize(m.runtime, size); e != nil {
			s.failedLocal = uint8(i)
			s.originalCause = e
			s.active = false
			s.pending = true
			// Only previously completed local objects have source destructors.
			if cleanup := m.unwindFormattingScope(s); cleanup != nil {
				s.cleanupCause = cleanup
				return &NumeralSourceFailure{BodyCause: e, DestructorCause: cleanup}
			}
			return e
		}
		s.completed |= 1 << uint(i)
	}
	return nil
}

func (m *NaturalManager) unwindFormattingLocals() error {
	return m.unwindFormattingScope(&m.formatting)
}
func (m *NaturalManager) unwindFormattingScope(s *naturalFormattingScope) error {
	for i := 2; i >= 0; i-- {
		bit := uint8(1) << uint(i)
		if s.completed&bit == 0 {
			continue
		}
		// A completed destructor only frees the published source buffer. A
		// partially constructed allocation is reserved for owning cleanup below.
		b := &s.locals[i]
		if b.raw != (memory.Block{}) {
			if e := m.runtime.Deallocate(&b.raw); e != nil {
				return e
			}
		}
		b.cap = 0
		b.pos = 0
		b.initialized = 0
		s.completed &^= bit
	}
	return nil
}
func (m *NaturalManager) finishFormattingLocals() error {
	return m.finishFormattingScope(&m.formatting)
}
func (m *NaturalManager) finishFormattingScope(s *naturalFormattingScope) error {
	if e := m.unwindFormattingScope(s); e != nil {
		s.active = false
		s.pending = true
		s.originalCause = e
		return &NumeralSourceFailure{DestructorCause: e}
	}
	for i := range s.locals {
		if s.locals[i].pending != (memory.Block{}) {
			// The source destructor frees its published buffer, not an
			// unpublished allocation retained by the Go continuation.
			s.active = false
			s.pending = true
			return arena.ErrInvalidSlot
		}
	}
	*s = naturalFormattingScope{invocation: s.invocation}
	return nil
}

// CleanupFormatting is explicit Go recovery of actual retained physical roots;
// it neither reexecutes a failed constructor's destructor nor repairs input.
func (m *NaturalManager) CleanupFormatting() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.runtime == nil {
		return arena.ErrClosed
	}
	return m.cleanupFormattingScope(&m.formatting)
}
func (m *NaturalManager) cleanupFormattingScope(s *naturalFormattingScope) error {
	if s.active {
		return arena.ErrInvalidSlot
	}
	if !s.pending {
		return nil
	}
	if e := m.unwindFormattingScope(s); e != nil {
		s.cleanupCause = e
		return e
	}
	for i := 2; i >= 0; i-- {
		if e := s.locals[i].finalize(m.runtime); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	*s = naturalFormattingScope{invocation: s.invocation}
	return nil
}

// The source decrements size_t j before reversing. At zero the first
// dereference endangers the caller's buffer while all local objects live.
func formattingReversalGuard(digits uint32) error {
	if digits == 0 {
		return arena.ErrBounds
	}
	return nil
}
