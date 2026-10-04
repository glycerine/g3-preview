// Licensed under the MIT license.
// Source: mpz.cpp142–161 and reverse member destruction in mpz.h/mpn.h.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// SourceFinalize runs only the source destructor, preserving caller-owned
// direct allocations. CloseChecked is the separate owning Go cleanup adapter.
// The charged scalar cursor records completed actions across cleanup refusal;
// it does not claim C++ recovery from an interrupted destructor.
func (m *IntegerManager) SourceFinalize() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.sourceFinalized {
		return nil
	}
	if m.smt2Display.active {
		return ErrSMT2DisplayPending
	}
	// Constructor failure is not a completed C++ object destructor invocation.
	if m.originalCause != nil {
		return arena.ErrClosed
	}
	m.sourceFinalizing = true
	fail := func(e error) error { m.cleanupCause = e; return e }
	for m.sourceFinalizeStage < 8 {
		switch m.sourceFinalizeStage {
		case 0, 1:
			value := &m.two64
			if m.sourceFinalizeStage == 1 {
				value = &m.intMin
			}
			if value.cell != 0 {
				if !validNumericRef(m.cells, value.cell) {
					m.sourceGuard = true
					return fail(arena.ErrInvalidSlot)
				}
				if e := m.free(value.cell); e != nil {
					return fail(e)
				}
				value.cell = 0 // del preserves m_val.
			}
		case 2, 3, 4, 5:
			index := int(m.sourceFinalizeStage-2) / 2
			ref := &m.tmp[index]
			if m.sourceFinalizeStage%2 == 1 {
				ref = &m.arg[index]
			}
			// Unlike del(numeral), source deallocate(ptr) reads the capacity header.
			// A missing scratch pointer is guarded at this actual destructor access.
			if *ref == 0 || !validNumericRef(m.cells, *ref) {
				m.sourceGuard = true
				return fail(arena.ErrInvalidSlot)
			}
			if e := m.free(*ref); e != nil {
				return fail(e)
			}
			*ref = 0
		case 6:
			if m.natural != nil {
				if e := m.natural.SourceFinalize(); e != nil {
					return fail(e)
				}
			}
		case 7:
			e := m.allocator.Close()
			// Chunk frees may complete before a later TLS sync/diagnostic refusal.
			// Mark only physically consumed pooled roots, preserving surviving chunks.
			if mark := m.markSourceFreedPools(); mark != nil {
				return fail(mark)
			}
			if e != nil {
				return fail(e)
			}
		}
		m.sourceFinalizeStage++
	}
	m.sourceFinalized = true
	m.cleanupCause = nil
	return nil
}

func (m *IntegerManager) markSourceFreedPools() error {
	for n := uint64(1); n <= uint64(m.cells.Stats().Slots); n++ {
		slot := arena.Slot(n)
		c, e := m.cells.Read(slot)
		if e != nil || c.released || c.backing.offset == 0 {
			continue
		}
		_, e = m.memory.ReadUint32(c.backing.raw, c.backing.offset+4)
		if errors.Is(e, memory.ErrRuntimeOwner) {
			c.released = true
			if e := m.cells.Write(slot, c); e != nil {
				return e
			}
		} else if e != nil {
			return e
		}
	}
	return nil
}

// After source allocator destruction its locator directory no longer exists.
// Each charged descriptor retains the real raw allocation token; owning cleanup
// frees direct blocks through Runtime, without an invented allocator event.
func (m *IntegerManager) freeAfterSourceFinalize(ref numericCellRef) error {
	c := m.cell(ref)
	if !c.released {
		if c.backing.offset != 0 {
			return arena.ErrInvalidSlot
		}
		raw := c.backing.raw
		e := m.memory.Deallocate(&raw)
		if raw == (memory.Block{}) {
			c.released = true
			m.store(ref, c)
		}
		if e != nil {
			c.cleanupPending = true
			m.store(ref, c)
			return e
		}
	}
	return m.cells.Recycle(numericSlot(ref))
}
