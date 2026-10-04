// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
// Concrete source polynomial.cpp tmp_monomial; actual Runtime byte storage.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
)

const monomialHeaderBytes uint64 = 20

func monomialObjectSize(n uint32) uint32         { return uint32(20) + n*8 }
func monomialRefIncrement(n uint32) uint32       { return n + 1 }
func monomialExponentProduct(a, b uint32) uint32 { return a * b }

type monomialTemp struct {
	raw, pending               memory.Block
	capacity                   uint32
	completed, destroyConsumed bool
}

func (m *MonomialManager) tempAllocate(t *monomialTemp, capacity uint32) error {
	return monomialTempAllocate(m.memory, t, capacity)
}
func monomialTempAllocate(r *memory.Runtime, t *monomialTemp, capacity uint32) error {
	if t.pending != (memory.Block{}) {
		return ErrMonomialPending
	}
	raw, e := r.Allocate(uint64(monomialObjectSize(capacity)))
	if raw != (memory.Block{}) {
		t.pending = raw
	}
	if e != nil {
		return e
	}
	// Placement monomial(UINT_MAX,0,nullptr,0) initializes exactly its header.
	for i, v := range [5]uint32{0, ^uint32(0), 0, 0, 0} {
		if e = r.WriteUint32(raw, uint64(i)*4, v); e != nil {
			return e
		}
	}
	return nil
}
func (m *MonomialManager) tempConstruct(t *monomialTemp) error {
	return monomialTempConstruct(m.memory, t)
}
func monomialTempConstruct(r *memory.Runtime, t *monomialTemp) error {
	if e := monomialTempAllocate(r, t, 128); e != nil {
		return e
	}
	t.raw = t.pending
	t.pending = memory.Block{}
	t.capacity = 128
	t.completed = true
	return nil
}
func (m *MonomialManager) tempSize(t *monomialTemp) (uint32, error) {
	return m.memory.ReadUint32(t.raw, 12)
}
func (m *MonomialManager) tempSetSize(t *monomialTemp, n uint32) error {
	if modularSourceDebug && n > t.capacity {
		return ErrMonomialAssertion
	}
	return m.memory.WriteUint32(t.raw, 12, n)
}
func (m *MonomialManager) tempPower(t *monomialTemp, i uint32) (MonomialPower, error) {
	var p MonomialPower
	v, e := m.memory.ReadUint32(t.raw, monomialHeaderBytes+uint64(i)*8)
	if e != nil {
		return p, e
	}
	p.Var = v
	p.Degree, e = m.memory.ReadUint32(t.raw, monomialHeaderBytes+uint64(i)*8+4)
	return p, e
}
func (m *MonomialManager) tempSetPower(t *monomialTemp, i uint32, p MonomialPower) error {
	if modularSourceDebug && i >= t.capacity {
		return ErrMonomialAssertion
	}
	if e := m.memory.WriteUint32(t.raw, monomialHeaderBytes+uint64(i)*8, p.Var); e != nil {
		return e
	}
	return m.memory.WriteUint32(t.raw, monomialHeaderBytes+uint64(i)*8+4, p.Degree)
}
func (m *MonomialManager) tempIncrease(t *monomialTemp, capacity uint32) error {
	if modularSourceDebug && capacity <= t.capacity {
		return ErrMonomialAssertion
	}
	// Source increase_capacity frees the old buffer before allocating its new
	// replacement. A missing old pointer remains a real stopped source hole.
	if e := m.memory.Deallocate(&t.raw); e != nil {
		return e
	}
	if e := m.tempAllocate(t, capacity); e != nil {
		return e
	}
	t.raw = t.pending
	t.pending = memory.Block{}
	t.capacity = capacity
	return nil
}
func (m *MonomialManager) tempExpand(t *monomialTemp, capacity uint32) error {
	if modularSourceDebug && capacity <= t.capacity {
		return ErrMonomialAssertion
	}
	if e := m.tempAllocate(t, capacity); e != nil {
		return e
	}
	n, e := m.tempSize(t)
	if e != nil {
		return e
	}
	if e = m.memory.WriteUint32(t.pending, 12, n); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		p, e := m.tempPower(t, i)
		if e != nil {
			return e
		}
		if e = m.memory.WriteUint32(t.pending, 20+uint64(i)*8, p.Var); e != nil {
			return e
		}
		if e = m.memory.WriteUint32(t.pending, 24+uint64(i)*8, p.Degree); e != nil {
			return e
		}
	}
	if e = m.memory.Deallocate(&t.raw); e != nil {
		return e
	}
	t.raw = t.pending
	t.pending = memory.Block{}
	t.capacity = capacity
	return nil
}
func (m *MonomialManager) tempInit(t *monomialTemp, p []MonomialPower) error {
	if uint64(len(p)) > uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	n := uint32(len(p))
	if n > t.capacity {
		if e := m.tempIncrease(t, n*2); e != nil {
			return e
		}
	}
	if modularSourceDebug && n >= t.capacity {
		return ErrMonomialAssertion
	}
	if e := m.memory.WriteUint32(t.raw, 12, n); e != nil {
		return e
	}
	for i, v := range p {
		if e := m.tempSetPower(t, uint32(i), v); e != nil {
			return e
		}
	}
	return nil
}
func (m *MonomialManager) tempReserve(t *monomialTemp, n uint32) error {
	if n > t.capacity {
		return m.tempIncrease(t, n*2)
	}
	return nil
}
func (m *MonomialManager) tempPush(t *monomialTemp, p MonomialPower) error {
	n, e := m.tempSize(t)
	if e != nil {
		return e
	}
	if n >= t.capacity {
		if e = m.tempExpand(t, n*2); e != nil {
			return e
		}
	}
	if e = m.tempSetPower(t, n, p); e != nil {
		return e
	}
	return m.memory.WriteUint32(t.raw, 12, n+1)
}
func (m *MonomialManager) tempHash(t *monomialTemp) (uint32, error) {
	n, e := m.tempSize(t)
	if e != nil {
		return 0, e
	}
	var failure error
	// The selected little-endian source hashes actual var/degree uint32 bytes;
	// no temporary spelling or shadow power slice is allocated.
	h := utilhash.WordReader(n*8, 11, func(i uint32) uint32 {
		if failure != nil {
			return 0
		}
		v, e := m.memory.ReadUint32(t.raw, 20+uint64(i)*4)
		if e != nil {
			failure = e
		}
		return v
	})
	if failure != nil {
		return 0, failure
	}
	if e = m.memory.WriteUint32(t.raw, 16, h); e != nil {
		return 0, e
	}
	return h, nil
}
func (m *MonomialManager) tempCleanup(t *monomialTemp) error {
	if t.pending != (memory.Block{}) {
		if e := m.memory.Deallocate(&t.pending); e != nil {
			return e
		}
	}
	if t.raw != (memory.Block{}) {
		if e := m.memory.Deallocate(&t.raw); e != nil {
			return e
		}
	}
	return nil
}
