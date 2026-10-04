// Licensed under the MIT license.
// Source mpn.cpp:371-399; constructor unwind from util/buffer.h:84-95.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Embedded cold ownership is charged by NaturalManager's actual header. Each
// invocation starts fresh scalar buffer lifetimes, without allocating a Go
// scope. Failed constructors do not run the source object's destructor.
// Their actual raw/pending blocks survive for explicitly named owning cleanup.
type naturalFormattingScope struct {
	locals                      [3]sourceLimbBuffer
	completed                   uint8
	failedLocal                 uint8
	active, pending             bool
	originalCause, cleanupCause error
}

func (m *NaturalManager) beginFormattingLocals(n uint32) error {
	if e := m.check(); e != nil {
		return e
	}
	s := &m.formatting
	if s.active || s.pending {
		return arena.ErrInvalidSlot
	}
	*s = naturalFormattingScope{active: true}
	sizes := [3]uint32{n, n + 1, 1} // source size_t addition narrowed by mpn_sbuffer
	for i, size := range sizes {
		if e := s.locals[i].resize(m.runtime, size); e != nil {
			s.failedLocal = uint8(i)
			s.originalCause = e
			s.active = false
			s.pending = true
			// Only previously completed local objects have source destructors.
			if cleanup := m.unwindFormattingLocals(); cleanup != nil {
				s.cleanupCause = cleanup
			}
			return e
		}
		s.completed |= 1 << uint(i)
	}
	return nil
}

func (m *NaturalManager) unwindFormattingLocals() error {
	s := &m.formatting
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
	if e := m.unwindFormattingLocals(); e != nil {
		m.formatting.active = false
		m.formatting.pending = true
		m.formatting.originalCause = e
		return e
	}
	for i := range m.formatting.locals {
		if m.formatting.locals[i].pending != (memory.Block{}) {
			// The source destructor frees its published buffer, not an
			// unpublished allocation retained by the Go continuation.
			m.formatting.active = false
			m.formatting.pending = true
			return arena.ErrInvalidSlot
		}
	}
	m.formatting = naturalFormattingScope{}
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
	s := &m.formatting
	if s.active {
		return arena.ErrInvalidSlot
	}
	if !s.pending {
		return nil
	}
	if e := m.unwindFormattingLocals(); e != nil {
		s.cleanupCause = e
		return e
	}
	for i := 2; i >= 0; i-- {
		if e := s.locals[i].finalize(m.runtime); e != nil {
			s.cleanupCause = e
			return e
		}
	}
	*s = naturalFormattingScope{}
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
