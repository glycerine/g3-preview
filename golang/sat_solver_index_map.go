// Source: src/util/map.h:24–163; src/util/hashtable.h:139–620; src/sat/sat_solver.h:481–483.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

func (m *satSolverIndexMap) entry(i uint32) *satSolverIndexSet {
	if m.extraEntries != nil {
		return &m.extraEntries[i]
	}
	return &m.entries[i]
}
func (m *satSolverIndexMap) open() error {
	if m == nil || m.self != m {
		return ErrSATOwner
	}
	if m.closed || m.closing || !m.constructed || m.pending != (memory.Block{}) || m.tempUsed != 0 {
		return arena.ErrClosed
	}
	return m.runtime.CheckOpen()
}
func (m *satSolverIndexMap) probe(k uint32) (uint32, bool, error) {
	if e := m.open(); e != nil {
		return 0, false, e
	}
	for j := uint32(0); j < m.capacity; j++ {
		i := ((k & (m.capacity - 1)) + j) & (m.capacity - 1)
		st, e := m.runtime.ReadUint32(m.raw, uint64(i)*40+4)
		if e != nil {
			return 0, false, e
		}
		if st == 0 {
			return i, false, nil
		}
		if st == 2 {
			hash, e := m.runtime.ReadUint32(m.raw, uint64(i)*40)
			if e != nil {
				return 0, false, e
			}
			if hash == k {
				key, e := m.runtime.ReadUint32(m.raw, uint64(i)*40+8)
				if e != nil {
					return 0, false, e
				}
				if key == k {
					return i, true, nil
				}
			}
		}
	}
	return 0, false, nil
}
func (m *satSolverIndexMap) contains(k uint32) (bool, error) { _, yes, e := m.probe(k); return yes, e }
func (m *satSolverIndexMap) find(k uint32) (*satSolverIndexSet, error) {
	i, yes, e := m.probe(k)
	if e != nil {
		return nil, e
	}
	if !yes {
		if satStorageSourceDebug {
			return nil, ErrSATAssertion
		}
		return nil, arena.ErrBounds
	}
	return m.entry(i), nil
}
func (m *satSolverIndexMap) innerOwnerCount() (uint32, error) {
	if e := m.open(); e != nil {
		return 0, e
	}
	n := uint32(0)
	for i := uint32(0); i < m.capacity; i++ {
		if m.entry(i).raw != (memory.Block{}) {
			n++
		}
	}
	return n, nil
}

// This cold provenance array owns actual concrete nested table states, including
// FREE values. Its full old+new physical backing is charged separately from the
// source40-byte array allocation and initial inline metadata in m's cold header.
func (m *satSolverIndexMap) allocateReplacement(n uint32) error {
	bytes := uint64(n) * uint64(reflect.TypeOf(satSolverIndexSet{}).Size())
	if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(satSolverIndexSet{}).Size()) {
		return arena.ErrBounds
	}
	if e := m.budget.Reserve(bytes); e != nil {
		return e
	}
	m.pendingMetadataBytes = bytes
	m.pendingEntries = make([]satSolverIndexSet, n)
	m.pendingCapacity = n
	h, e := m.runtime.Allocate(uint64(n) * 40)
	m.pending = h
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		base := uint64(i) * 40
		if e = m.runtime.WriteUint32(h, base+4, 0); e != nil {
			return e
		}
		row := &m.pendingEntries[i]
		row.parent, row.base = h, base+16
		if e = row.init(m.runtime); e != nil {
			return e
		}
		if e = row.sync(); e != nil {
			return e
		}
	}
	return nil
}
func (m *satSolverIndexMap) deleteOld() error {
	for m.closeCursor < m.capacity {
		if e := m.entry(m.closeCursor).close(); e != nil {
			return e
		}
		m.closeCursor++
	}
	if e := m.runtime.Deallocate(&m.raw); e != nil {
		return e
	}
	return nil
}
func (m *satSolverIndexMap) publishReplacement() {
	if m.metadataBytes != 0 {
		m.budget.Release(m.metadataBytes)
	}
	m.entries = [8]satSolverIndexSet{}
	m.extraEntries, m.pendingEntries = m.pendingEntries, nil
	m.metadataBytes, m.pendingMetadataBytes = m.pendingMetadataBytes, 0
	m.raw, m.pending = m.pending, memory.Block{}
	m.capacity, m.pendingCapacity = m.pendingCapacity, 0
	m.closeCursor = 0
}
func (m *satSolverIndexMap) expand() error {
	n := m.capacity << 1
	if e := m.allocateReplacement(n); e != nil {
		return e
	}
	for i := uint32(0); i < m.capacity; i++ {
		st, e := m.runtime.ReadUint32(m.raw, uint64(i)*40+4)
		if e != nil {
			return e
		}
		if st != 2 {
			continue
		}
		hash, e := m.runtime.ReadUint32(m.raw, uint64(i)*40)
		if e != nil {
			return e
		}
		found := false
		for j := uint32(0); j < n; j++ {
			k := ((hash & (n - 1)) + j) & (n - 1)
			dstst, e := m.runtime.ReadUint32(m.pending, uint64(k)*40+4)
			if e != nil {
				return e
			}
			if dstst != 0 {
				continue
			}
			// Implicit default_entry move assignment copies cached hash/state first,
			// then key, then inner hashtable copy-assignment (there is no move overload).
			for _, off := range [3]uint64{0, 4, 8} {
				v, e := m.runtime.ReadUint32(m.raw, uint64(i)*40+off)
				if e != nil {
					return e
				}
				if e = m.runtime.WriteUint32(m.pending, uint64(k)*40+off, v); e != nil {
					return e
				}
			}
			if e = m.pendingEntries[k].assign(m.entry(i)); e != nil {
				return e
			}
			found = true
			break
		}
		if !found {
			return ErrSATAssertion
		}
	}
	if e := m.deleteOld(); e != nil {
		return e
	}
	m.publishReplacement()
	m.deleted = 0
	return nil
}
func (m *satSolverIndexMap) insert(k uint32, v *satSolverIndexSet) (err error) {
	if e := m.open(); e != nil {
		return e
	}
	// table2map::insert creates key_data(k,v), then const insert creates its data
	// temporary. Both actual nested copy constructors are distinct source actions.
	m.tempUsed = 1
	if err = m.temps[0].copyCtor(v); err != nil {
		return err
	}
	m.tempUsed = 2
	if err = m.temps[1].copyCtor(&m.temps[0]); err != nil {
		// Failed data construction has no source destructor. The older completed
		// key_data temporary DOES unwind now; its source cleanup precedes any later
		// owning abort of the failed temporary's allocated prefix.
		if e := m.temps[0].close(); e == nil {
			m.temps[0] = satSolverIndexSet{}
		}
		return err
	}
	defer func() {
		for m.tempUsed > 0 {
			i := m.tempUsed - 1
			if e := m.temps[i].close(); e != nil {
				if err == nil {
					err = e
				}
				return
			}
			m.temps[i] = satSolverIndexSet{}
			m.tempUsed--
		}
	}()
	if (m.size+m.deleted)<<2 > m.capacity*3 {
		if err = m.expand(); err != nil {
			return err
		}
	}
	hasDeleted := false
	del := uint32(0)
	for j := uint32(0); j < m.capacity; j++ {
		i := ((k & (m.capacity - 1)) + j) & (m.capacity - 1)
		st, e := m.runtime.ReadUint32(m.raw, uint64(i)*40+4)
		if e != nil {
			return e
		}
		hit := false
		if st == 2 {
			hash, e := m.runtime.ReadUint32(m.raw, uint64(i)*40)
			if e != nil {
				return e
			}
			if hash == k {
				key, e := m.runtime.ReadUint32(m.raw, uint64(i)*40+8)
				if e != nil {
					return e
				}
				hit = key == k
			}
		}
		if hit || st == 0 {
			if !hit && hasDeleted {
				i = del
				m.deleted--
			}
			if e = m.runtime.WriteUint32(m.raw, uint64(i)*40+8, k); e != nil {
				return e
			}
			if e = m.entry(i).assign(&m.temps[1]); e != nil {
				return e
			}
			if e = m.runtime.WriteUint32(m.raw, uint64(i)*40+4, 2); e != nil {
				return e
			}
			if !hit {
				if e = m.runtime.WriteUint32(m.raw, uint64(i)*40, k); e != nil {
					return e
				}
				m.size++
			}
			return nil
		}
		if st == 1 {
			hasDeleted = true
			del = i
		}
	}
	return ErrSATAssertion
}
func (m *satSolverIndexMap) reset() error {
	if e := m.open(); e != nil {
		return e
	}
	if m.size == 0 && m.deleted == 0 {
		return nil
	}
	overhead := uint32(0)
	for i := uint32(0); i < m.capacity; i++ {
		st, e := m.runtime.ReadUint32(m.raw, uint64(i)*40+4)
		if e != nil {
			return e
		}
		if st == 0 {
			overhead++
		} else {
			if e = m.runtime.WriteUint32(m.raw, uint64(i)*40+4, 0); e != nil {
				return e
			}
		}
	}
	if m.capacity > 16 && overhead<<2 > m.capacity*3 {
		old := m.capacity
		if e := m.deleteOld(); e != nil {
			return e
		}
		// Source publishes capacity HALF before the replacement alloc_vect call.
		// Retain old metadata until the actual replacement cohort is established.
		m.capacity = old >> 1
		if e := m.allocateReplacement(m.capacity); e != nil {
			m.capacity = old >> 1
			return e
		}
		m.publishReplacement()
	}
	m.size, m.deleted = 0, 0
	return nil
}
