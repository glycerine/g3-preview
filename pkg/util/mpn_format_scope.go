// Source: src/util/mpn.cpp:371-399; src/util/buffer.h:84-95.
package util

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// Embedded cold ownership is charged by NaturalManager's actual header. Each
// invocation starts fresh scalar buffer lifetimes, without allocating a Go
// scope. Failed constructors do not run the source object's destructor.
// Their actual raw/pending blocks survive for explicitly named owning cleanup.
type naturalFormattingScope struct {
	invocation                  uint64
	Locals                      [3]SourceLimbBuffer
	CompletedField              uint8
	failedLocal                 uint8
	Active, PendingField        bool
	originalCause, cleanupCause error
}

func (m *NaturalManager) beginFormattingLocals(n uint32) error {
	return m.beginFormattingScope(&m.Formatting, n)
}
func (m *NaturalManager) beginFormattingScope(s *naturalFormattingScope, n uint32) error {
	return m.beginFormattingScopeTracked(s, n, nil)
}
func (m *NaturalManager) beginFormattingScopeTracked(s *naturalFormattingScope, n uint32, receipt *NumeralInsertionReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if s.Active || s.PendingField {
		return arena.ErrInvalidSlot
	}
	if s.invocation == math.MaxUint64 {
		return arena.ErrBounds
	}
	*s = naturalFormattingScope{Active: true, invocation: s.invocation + 1}
	if receipt != nil {
		receipt.formatting = s
		receipt.formattingOwned = true
		receipt.formattingInvocation = s.invocation
	}
	sizes := [3]uint32{n, n + 1, 1} // source size_t addition narrowed by mpn_sbuffer
	for i, size := range sizes {
		if e := s.Locals[i].resize(m.runtime, size); e != nil {
			s.failedLocal = uint8(i)
			s.originalCause = e
			s.Active = false
			s.PendingField = true
			// Only previously completed local objects have source destructors.
			if cleanup := m.unwindFormattingScope(s); cleanup != nil {
				s.cleanupCause = cleanup
				return &NumeralSourceFailure{BodyCause: e, DestructorCause: cleanup}
			}
			return e
		}
		s.CompletedField |= 1 << uint(i)
	}
	return nil
}

func (m *NaturalManager) unwindFormattingLocals() error {
	return m.unwindFormattingScope(&m.Formatting)
}
func (m *NaturalManager) unwindFormattingScope(s *naturalFormattingScope) error {
	for i := 2; i >= 0; i-- {
		bit := uint8(1) << uint(i)
		if s.CompletedField&bit == 0 {
			continue
		}
		// A completed destructor only frees the published source buffer. A
		// partially constructed allocation is reserved for owning cleanup below.
		b := &s.Locals[i]
		if b.Raw != (Block{}) {
			if e := m.runtime.Deallocate(&b.Raw); e != nil {
				return e
			}
		}
		b.cap = 0
		b.pos = 0
		b.initialized = 0
		s.CompletedField &^= bit
	}
	return nil
}
func (m *NaturalManager) finishFormattingLocals() error {
	return m.finishFormattingScope(&m.Formatting)
}
func (m *NaturalManager) finishFormattingScope(s *naturalFormattingScope) error {
	if e := m.unwindFormattingScope(s); e != nil {
		s.Active = false
		s.PendingField = true
		s.originalCause = e
		return &NumeralSourceFailure{DestructorCause: e}
	}
	for i := range s.Locals {
		if s.Locals[i].Pending != (Block{}) {
			// The source destructor frees its published buffer, not an
			// unpublished allocation retained by the Go continuation.
			s.Active = false
			s.PendingField = true
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
		return ErrRuntimeOwner
	}
	if m.ClosedField || m.runtime == nil {
		return arena.ErrClosed
	}
	return m.cleanupFormattingScope(&m.Formatting)
}
func (m *NaturalManager) cleanupFormattingScope(s *naturalFormattingScope) error {
	if s.Active {
		return arena.ErrInvalidSlot
	}
	if !s.PendingField {
		return nil
	}
	if e := m.unwindFormattingScope(s); e != nil {
		s.cleanupCause = e
		return e
	}
	for i := 2; i >= 0; i-- {
		if e := s.Locals[i].Finalize(m.runtime); e != nil {
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
