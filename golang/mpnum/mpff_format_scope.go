// Licensed under the MIT license.
// Source mpff.cpp display locals.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Three actual unsigned1024 and one unsigned16 inline owners, charged in the
// FastFloatManager header. Completed defaults are distinct from failed char
// constructors. Pending blocks are explicit Go owning continuations.
type fastFormattingScope struct {
	large                       [3]sourceLimbBuffer1024
	small                       sourceLimbBuffer
	completed                   uint8
	active, pending             bool
	originalCause, cleanupCause error
}

func (m *FastFloatManager) beginFastFormatting() error {
	s := &m.formatting
	if s.active || s.pending {
		return arena.ErrInvalidSlot
	}
	*s = fastFormattingScope{active: true}
	return nil
}
func (m *FastFloatManager) formatView(i int, n uint32) ffLimbView {
	if i == 2 {
		return ffLimbView{runtime: m.natural.runtime, source16: &m.formatting.small, size: n}
	}
	j := i
	if i == 3 {
		j = 2
	}
	return ffLimbView{runtime: m.natural.runtime, source1024: &m.formatting.large[j], size: n}
}
func fastNaturalView(v ffLimbView) limbView {
	if v.source16 != nil {
		return limbView{runtime: v.runtime, source: v.source16, size: v.size}
	}
	if v.source1024 != nil {
		return limbView{runtime: v.runtime, source1024: v.source1024, size: v.size}
	}
	return limbView{words: v.m.words, ref: v.v.ref, size: v.size}
}
func (m *FastFloatManager) beginFastCharacters(v ffLimbView, offset uint32) (uint32, error) {
	n := v.size
	if e := m.natural.beginIntegerCharacters(uint32(11 * n)); e != nil {
		return 0, e
	}
	e := m.natural.sourceToCString(fastNaturalView(v), sourceDecimalDestination{character: &m.natural.charFormatting.buffer, assembly: m.natural}, uint64(m.natural.charFormatting.buffer.pos))
	if e != nil {
		return 0, e
	}
	return m.natural.appendIntegerCString(offset)
}
func (m *FastFloatManager) finishFastFormatting() error {
	s := &m.formatting
	// Actual inner mpn destructors precede the enclosing completed character,
	// which precedes the outer decimal buffers.
	if m.natural.formatting.completed != 0 {
		if e := m.natural.unwindFormattingLocals(); e != nil {
			return e
		}
		m.natural.formatting.active = false
		m.natural.formatting.pending = true
	}
	if e := m.natural.destroyIntegerCharacters(); e != nil {
		return e
	}
	for i := 3; i >= 0; i-- {
		bit := uint8(1) << uint(i)
		if s.completed&bit == 0 {
			continue
		}
		if i == 2 {
			if s.small.raw != (memory.Block{}) {
				if e := m.natural.runtime.Deallocate(&s.small.raw); e != nil {
					return e
				}
			}
			s.small.pos = 0
			s.small.cap = 0
			s.small.initialized = 0
		} else {
			j := i
			if i == 3 {
				j = 2
			}
			b := &s.large[j]
			if b.raw != (memory.Block{}) {
				if e := m.natural.runtime.Deallocate(&b.raw); e != nil {
					return e
				}
			}
			b.pos = 0
			b.cap = 0
			b.initialized = [16]uint64{}
		}
		s.completed &^= bit
	}
	s.active = false
	s.pending = m.natural.formatting.pending || m.natural.charFormatting.pending || s.small.pending != (memory.Block{})
	for i := range s.large {
		s.pending = s.pending || s.large[i].pending != (memory.Block{})
	}
	if !s.pending {
		*s = fastFormattingScope{}
	}
	return nil
}
func (m *FastFloatManager) endFastFormatting(cause error) error {
	if cause != nil {
		m.formatting.originalCause = cause
	}
	cleanup := m.finishFastFormatting()
	if cleanup != nil {
		m.formatting.active = false
		m.formatting.pending = true
		m.formatting.cleanupCause = cleanup
	}
	if cause != nil {
		return cause
	}
	return cleanup
}

// CleanupFormatting is owning cleanup after the source completed-local cursor.
// It does not fabricate a failed constructor's destructor or repair inputs.
func (m *FastFloatManager) CleanupFormatting() error {
	if m == nil || m.self != m {
		return ErrFastFloatOwner
	}
	if m.closed || m.natural == nil {
		return arena.ErrClosed
	}
	if e := m.finishFastFormatting(); e != nil {
		return e
	}
	if e := m.natural.cleanupOutputFormatting(); e != nil {
		return e
	}
	for i := 2; i >= 0; i-- {
		if e := m.formatting.large[i].finalize(m.natural.runtime); e != nil {
			return e
		}
	}
	if e := m.formatting.small.finalize(m.natural.runtime); e != nil {
		return e
	}
	m.formatting = fastFormattingScope{}
	return nil
}
