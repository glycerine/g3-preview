// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
// Source mpz.cpp display and util/buffer.h lexical sbuffer<char,1024>.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

type integerCharacterScope struct {
	buffer                      sourceCharBuffer
	active, completed, pending  bool
	originalCause, cleanupCause error
}

func (m *NaturalManager) beginIntegerCharacters(n uint32) error {
	if e := m.check(); e != nil {
		return e
	}
	s := &m.charFormatting
	if s.active || s.pending {
		return arena.ErrInvalidSlot
	}
	*s = integerCharacterScope{active: true}
	if e := s.buffer.resize(m.runtime, n); e != nil {
		s.active = false
		s.pending = true
		s.originalCause = e
		return e // failed constructor's destructor must not run
	}
	s.completed = true
	return nil
}
func (m *NaturalManager) destroyIntegerCharacters() error {
	s := &m.charFormatting
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
	*s = integerCharacterScope{}
	return nil
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
		m.charFormatting = integerCharacterScope{}
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
