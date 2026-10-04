// Source: src/sat/sat_solver.h:483; src/util/map.h:24–73; src/util/hashtable.h:45–64,139–150,244–273; src/util/memory_manager.h:104–120.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Concrete constructor/destructor storage for u_map<index_set>. The selected
// LP64 source entry is40 bytes: hash/state8, key/padding8, core table24.
// Each inner default_hash_entry<unsigned> is12 bytes. All source entry fields
// use the actual Runtime bytes; only actual block provenance is cold metadata.
// The pointer word encodes that block's slot/generation under this fixed borrowed
// Runtime. It is an explicit scalar pointer adaptation, not a physical C++ address.
// Consequence/map operations remain a separate source algorithm dependency.
type satSolverIndexSet struct{ raw memory.Block }
type satSolverIndexMap struct {
	self                            *satSolverIndexMap
	runtime                         *memory.Runtime
	budget                          *arena.Budget
	raw, pending                    memory.Block
	entries                         [8]satSolverIndexSet
	ownerBytes                      uint64
	constructedEntries, closeCursor uint32
	constructed, closing, closed    bool
}

func newSATSolverIndexMap(r *memory.Runtime, b *arena.Budget) (*satSolverIndexMap, error) {
	if r == nil || r.Budget() != b {
		return nil, ErrSATOwner
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(satSolverIndexMap{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	m := &satSolverIndexMap{runtime: r, budget: b, ownerBytes: bytes}
	m.self = m
	// alloc_vect returns only after every ascending placement constructor. A
	// reached raw allocation is retained in pending until that return/publish.
	var e error
	m.pending, e = r.Allocate(8 * 40)
	if e != nil {
		return m, e
	}
	for i := uint32(0); i < 8; i++ {
		base := uint64(i) * 40
		// Entry declaration initializes state before default-constructing m_data.
		if e = r.WriteUint32(m.pending, base+4, 0); e != nil {
			return m, e
		}
		row := &m.entries[i]
		row.raw, e = r.Allocate(8 * 12)
		if e != nil {
			return m, e
		}
		for j := uint32(0); j < 8; j++ {
			if e = r.WriteUint32(row.raw, uint64(j)*12+4, 0); e != nil {
				return m, e
			}
		}
		// The inner core constructor publishes its pointer/capacity/counts only
		// after its alloc_vect has constructed every default unsigned entry.
		if e = r.WriteUint32(m.pending, base+16, row.raw.Slot()); e != nil {
			return m, e
		}
		if e = r.WriteUint32(m.pending, base+20, row.raw.Generation()); e != nil {
			return m, e
		}
		if e = r.WriteUint32(m.pending, base+24, 8); e != nil {
			return m, e
		}
		if e = r.WriteUint32(m.pending, base+28, 0); e != nil {
			return m, e
		}
		if e = r.WriteUint32(m.pending, base+32, 0); e != nil {
			return m, e
		}
		m.constructedEntries = i + 1
	}
	m.raw = m.pending
	m.pending = memory.Block{}
	m.constructed = true
	return m, nil
}

// Close of a completed member performs the source ascending dealloc_vect entry
// traversal, then frees the outer array. Failed alloc_vect has no completed
// C++ array destructor: draining its named prefix/pending raw allocation here is
// a separate Go owning-abort adapter, not claimed source exception cleanup.
func (m *satSolverIndexMap) Close() error {
	if m == nil || m.self != m {
		return ErrSATOwner
	}
	if m.closed {
		return nil
	}
	m.closing = true
	for m.closeCursor < 8 {
		if m.entries[m.closeCursor].raw != (memory.Block{}) {
			if e := m.runtime.Deallocate(&m.entries[m.closeCursor].raw); e != nil {
				return e
			}
		}
		m.closeCursor++
	}
	if m.raw != (memory.Block{}) {
		if e := m.runtime.Deallocate(&m.raw); e != nil {
			return e
		}
	}
	if m.pending != (memory.Block{}) {
		if e := m.runtime.Deallocate(&m.pending); e != nil {
			return e
		}
	}
	m.budget.Release(m.ownerBytes)
	m.ownerBytes = 0
	m.closed = true
	return nil
}
