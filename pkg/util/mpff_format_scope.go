// Source: src/util/mpff.cpp:1229-1342.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Decimal scratch limbs use typed Go slices. The remaining small source buffer
// and character formatting retain their explicit Runtime cleanup.
type fastFormattingScope struct {
	large                       [3][]uint32
	small                       SourceLimbBuffer
	completed                   uint8
	active, pending             bool
	originalCause, cleanupCause error
}

func (m *FastFloatManager) beginFastFormatting() error {
	s := &m.formatting
	if s.active || s.pending {
		return arena.ErrInvalidSlot
	}
	*s = fastFormattingScope{large: s.large, active: true}
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
	return ffLimbView{runtime: m.natural.runtime, external: m.formatting.large[j], size: n}
}
func fastNaturalView(v ffLimbView) limbView {
	if v.source16 != nil {
		return limbView{runtime: v.runtime, source: v.source16, size: v.size}
	}
	if v.external != nil {
		return limbView{external: v.external[v.offset:], size: v.size}
	}
	return limbView{words: v.m.words, ref: v.v.ref, size: v.size}
}
func (m *FastFloatManager) beginFastCharacters(v ffLimbView, offset uint32) (uint32, error) {
	n := v.size
	if e := m.natural.beginIntegerCharacters(uint32(11 * n)); e != nil {
		return 0, e
	}
	e := m.natural.sourceToCString(fastNaturalView(v), sourceDecimalDestination{character: &m.natural.CharFormatting.Buffer, assembly: m.natural}, uint64(m.natural.CharFormatting.Buffer.pos))
	if e != nil {
		return 0, e
	}
	return m.natural.appendIntegerCString(offset)
}
func (m *FastFloatManager) finishFastFormatting() error {
	s := &m.formatting
	// Actual inner mpn destructors precede the enclosing completed character,
	// which precedes the outer decimal buffers.
	if m.natural.Formatting.CompletedField != 0 {
		if e := m.natural.unwindFormattingLocals(); e != nil {
			return e
		}
		m.natural.Formatting.Active = false
		m.natural.Formatting.PendingField = true
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
			if s.small.Raw != (Block{}) {
				if e := m.natural.runtime.Deallocate(&s.small.Raw); e != nil {
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
			s.large[j] = s.large[j][:0]
		}
		s.completed &^= bit
	}
	s.active = false
	s.pending = m.natural.Formatting.PendingField || m.natural.CharFormatting.pending || s.small.Pending != (Block{})
	if !s.pending {
		*s = fastFormattingScope{large: s.large}
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
	for i := range m.formatting.large {
		m.formatting.large[i] = nil
	}
	if e := m.formatting.small.Finalize(m.natural.runtime); e != nil {
		return e
	}
	m.formatting = fastFormattingScope{}
	return nil
}
