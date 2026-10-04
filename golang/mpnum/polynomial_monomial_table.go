// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
// Concrete chashtable<monomial*>. Links/keys live in actual Runtime bytes.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
)

const (
	monomialTemporaryKey uint32 = 1
	monomialObjectKey    uint32 = 2
)

type monomialKey struct{ kind, slot uint32 }

const monomialFreeLink uint64 = math.MaxUint64

// Two scalar pointer codec words per physical cell. This is an explicit 16-byte
// Go pointer adaptation, not a claim that source ILP32 pointers occupy 8 bytes.
type monomialTable struct {
	raw, pending                                                     memory.Block
	capacity, slots, size, usedSlots, nextCell, freeCell, collisions uint32
	pendingCapacity                                                  uint32
	completed, destroyConsumed                                       bool
}

func (t *monomialTable) construct(m *MonomialManager, slots, cellar uint32) error {
	if t.raw != (memory.Block{}) || t.pending != (memory.Block{}) {
		return ErrMonomialPending
	}
	*t = monomialTable{}
	t.slots = slots
	t.capacity = slots + cellar
	t.nextCell = slots
	n := uint64(t.capacity) * 16
	// Source alloc_vect byte arithmetic follows the selected size_t domain.
	if m.memory.SizeTBits() == 32 {
		n = uint64(uint32(n))
	}
	raw, e := m.memory.Allocate(n)
	if raw != (memory.Block{}) {
		t.pending = raw
		t.pendingCapacity = t.capacity
	}
	if e != nil {
		return e
	}
	for i := uint32(0); i < t.capacity; i++ {
		if e = t.writeNextAt(m, t.pending, i, monomialFreeLink); e != nil {
			return e
		}
	}
	t.raw = t.pending
	t.pending = memory.Block{}
	t.pendingCapacity = 0
	t.completed = true
	return nil
}
func (t *monomialTable) nextAt(m *MonomialManager, raw memory.Block, i uint32) (uint64, error) {
	lo, e := m.memory.ReadUint32(raw, uint64(i)*16)
	if e != nil {
		return 0, e
	}
	hi, e := m.memory.ReadUint32(raw, uint64(i)*16+4)
	return uint64(lo) | uint64(hi)<<32, e
}
func (t *monomialTable) next(m *MonomialManager, i uint32) (uint64, error) {
	return t.nextAt(m, t.raw, i)
}
func (t *monomialTable) writeNextAt(m *MonomialManager, raw memory.Block, i uint32, n uint64) error {
	if e := m.memory.WriteUint32(raw, uint64(i)*16, uint32(n)); e != nil {
		return e
	}
	return m.memory.WriteUint32(raw, uint64(i)*16+4, uint32(n>>32))
}
func (t *monomialTable) setNext(m *MonomialManager, i uint32, n uint64) error {
	return t.writeNextAt(m, t.raw, i, n)
}
func (t *monomialTable) key(m *MonomialManager, i uint32) (monomialKey, error) {
	var k monomialKey
	var e error
	k.kind, e = m.memory.ReadUint32(t.raw, uint64(i)*16+8)
	if e != nil {
		return k, e
	}
	k.slot, e = m.memory.ReadUint32(t.raw, uint64(i)*16+12)
	return k, e
}
func (t *monomialTable) setKey(m *MonomialManager, i uint32, k monomialKey) error {
	if e := m.memory.WriteUint32(t.raw, uint64(i)*16+8, k.kind); e != nil {
		return e
	}
	return m.memory.WriteUint32(t.raw, uint64(i)*16+12, k.slot)
}
func (t *monomialTable) copyCell(m *MonomialManager, dst, src uint32) error {
	for i := uint64(0); i < 4; i++ {
		v, e := m.memory.ReadUint32(t.raw, uint64(src)*16+i*4)
		if e != nil {
			return e
		}
		if e = m.memory.WriteUint32(t.raw, uint64(dst)*16+i*4, v); e != nil {
			return e
		}
	}
	return nil
}
func (t *monomialTable) freeAvailable() bool { return t.freeCell != 0 || t.nextCell < t.capacity }
func (t *monomialTable) takeFree(m *MonomialManager) (uint32, error) {
	if t.freeCell != 0 {
		i := t.freeCell - 1
		n, e := t.next(m, i)
		if e != nil {
			return 0, e
		}
		if n > uint64(math.MaxUint32) {
			return 0, arena.ErrBounds
		}
		t.freeCell = uint32(n)
		return i, nil
	}
	if t.nextCell >= t.capacity {
		return 0, arena.ErrBounds
	}
	i := t.nextCell
	t.nextCell++
	return i, nil
}
func (t *monomialTable) recycle(m *MonomialManager, i uint32) error {
	if e := t.setNext(m, i, uint64(t.freeCell)); e != nil {
		return e
	}
	t.freeCell = i + 1
	return nil
}
func (t *monomialTable) hash(m *MonomialManager, k monomialKey) (uint32, error) {
	return m.keyWord(k, 4)
}
func (t *monomialTable) equal(m *MonomialManager, a, b monomialKey) (bool, error) {
	na, e := m.keyWord(a, 3)
	if e != nil {
		return false, e
	}
	nb, e := m.keyWord(b, 3)
	if e != nil {
		return false, e
	}
	if na != nb {
		return false, nil
	}
	ha, e := t.hash(m, a)
	if e != nil {
		return false, e
	}
	hb, e := t.hash(m, b)
	if e != nil {
		return false, e
	}
	if ha != hb {
		return false, nil
	}
	// Source deliberately excludes total_degree: temporary values do not update it.
	for i := uint32(0); i < na; i++ {
		pa, e := m.keyPower(a, i)
		if e != nil {
			return false, e
		}
		pb, e := m.keyPower(b, i)
		if e != nil {
			return false, e
		}
		if pa != pb {
			return false, nil
		}
	}
	return true, nil
}
func (t *monomialTable) insert(m *MonomialManager, k monomialKey, replace bool) (uint32, bool, error) {
	if t.pending != (memory.Block{}) {
		return 0, false, ErrMonomialPending
	}
	// Grow before even checking for a duplicate, as the source does.
	if !t.freeAvailable() {
		if e := t.expand(m); e != nil {
			return 0, false, e
		}
	}
	hash, e := t.hash(m, k)
	if e != nil {
		return 0, false, e
	}
	if t.slots == 0 {
		return 0, false, arena.ErrBounds
	}
	i := hash & (t.slots - 1)
	next, e := t.next(m, i)
	if e != nil {
		return 0, false, e
	}
	if next == monomialFreeLink {
		t.size++
		t.usedSlots++
		if e = t.setKey(m, i, k); e != nil {
			return i, true, e
		}
		e = t.setNext(m, i, 0)
		return i, true, e
	}
	for cursor := i; ; {
		old, e := t.key(m, cursor)
		if e != nil {
			return 0, false, e
		}
		same, e := t.equal(m, old, k)
		if e != nil {
			return 0, false, e
		}
		if same {
			if replace {
				e = t.setKey(m, cursor, k)
			}
			return cursor, false, e
		}
		t.collisions++
		link, e := t.next(m, cursor)
		if e != nil {
			return 0, false, e
		}
		if link == 0 {
			break
		}
		if link > uint64(t.capacity) {
			return 0, false, arena.ErrBounds
		}
		cursor = uint32(link - 1)
	}
	t.size++
	fresh, e := t.takeFree(m)
	if e != nil {
		return 0, false, e
	}
	if e = t.copyCell(m, fresh, i); e != nil {
		return 0, false, e
	}
	if e = t.setKey(m, i, k); e != nil {
		return 0, false, e
	}
	e = t.setNext(m, i, uint64(fresh)+1)
	return i, true, e
}
func (t *monomialTable) erase(m *MonomialManager, k monomialKey) error {
	h, e := t.hash(m, k)
	if e != nil {
		return e
	}
	if t.slots == 0 {
		return arena.ErrBounds
	}
	head := h & (t.slots - 1)
	link, e := t.next(m, head)
	if e != nil {
		return e
	}
	if link == monomialFreeLink {
		return nil
	}
	prev := uint32(math.MaxUint32)
	for i := head; ; {
		old, e := t.key(m, i)
		if e != nil {
			return e
		}
		same, e := t.equal(m, old, k)
		if e != nil {
			return e
		}
		if same {
			return t.eraseCell(m, prev, i)
		}
		t.collisions++
		n, e := t.next(m, i)
		if e != nil {
			return e
		}
		if n == 0 {
			return nil
		}
		if n > uint64(t.capacity) {
			return arena.ErrBounds
		}
		prev = i
		i = uint32(n - 1)
	}
}
func (t *monomialTable) eraseCell(m *MonomialManager, prev, i uint32) error {
	t.size--
	next, e := t.next(m, i)
	if e != nil {
		return e
	}
	if prev == math.MaxUint32 {
		if next == 0 {
			t.usedSlots--
			return t.setNext(m, i, monomialFreeLink)
		}
		if next > uint64(t.capacity) {
			return arena.ErrBounds
		}
		child := uint32(next - 1)
		if e = t.copyCell(m, i, child); e != nil {
			return e
		}
		return t.recycle(m, child)
	}
	if e = t.setNext(m, prev, next); e != nil {
		return e
	}
	return t.recycle(m, i)
}

// Owning abort removes a caller-named reachable token even if a failed source
// rename has changed its hash bucket. This is not literal source erase/repair.
func (t *monomialTable) removeToken(m *MonomialManager, k monomialKey) error {
	for head := uint32(0); head < t.slots; head++ {
		n, e := t.next(m, head)
		if e != nil {
			return e
		}
		if n == monomialFreeLink {
			continue
		}
		prev := uint32(math.MaxUint32)
		for i := head; ; {
			old, e := t.key(m, i)
			if e != nil {
				return e
			}
			if old == k {
				return t.eraseCell(m, prev, i)
			}
			n, e = t.next(m, i)
			if e != nil {
				return e
			}
			if n == 0 {
				break
			}
			if n > uint64(t.capacity) {
				return arena.ErrBounds
			}
			prev = i
			i = uint32(n - 1)
		}
	}
	return nil
}
func (t *monomialTable) expand(m *MonomialManager) error {
	newSlots := t.slots * 2
	newCellar := (t.capacity - t.slots) * 2
	for {
		var candidate monomialTable
		if e := candidate.construct(m, newSlots, newCellar); e != nil {
			t.pending = candidate.pending
			t.pendingCapacity = candidate.pendingCapacity
			return e
		}
		nextCell, full, e := t.copyInto(m, &candidate)
		if e != nil {
			t.pending = candidate.raw
			t.pendingCapacity = candidate.capacity
			return e
		}
		if !full {
			if e = m.memory.Deallocate(&t.raw); e != nil {
				t.pending = candidate.raw
				t.pendingCapacity = candidate.capacity
				return e
			}
			t.raw = candidate.raw
			t.slots = candidate.slots
			t.capacity = candidate.capacity
			t.nextCell = nextCell
			t.freeCell = 0
			return nil
		}
		if e = m.memory.Deallocate(&candidate.raw); e != nil {
			t.pending = candidate.raw
			t.pendingCapacity = candidate.capacity
			return e
		}
		newCellar *= 2
	}
}
func (t *monomialTable) copyInto(m *MonomialManager, out *monomialTable) (uint32, bool, error) {
	t.usedSlots = 0 // Actual source copy_table writes through the old metadata ref.
	cellar := out.slots
	for head := uint32(0); head < t.slots; head++ {
		n, e := t.next(m, head)
		if e != nil {
			return cellar, false, e
		}
		if n == monomialFreeLink {
			continue
		}
		for i := head; ; {
			key, e := t.key(m, i)
			if e != nil {
				return cellar, false, e
			}
			h, e := t.hash(m, key)
			if e != nil {
				return cellar, false, e
			}
			if out.slots == 0 {
				return cellar, false, arena.ErrBounds
			}
			index := h & (out.slots - 1)
			next, e := out.next(m, index)
			if e != nil {
				return cellar, false, e
			}
			if next == monomialFreeLink {
				if e = out.setKey(m, index, key); e != nil {
					return cellar, false, e
				}
				if e = out.setNext(m, index, 0); e != nil {
					return cellar, false, e
				}
				t.usedSlots++
			} else {
				if cellar == out.capacity {
					return cellar, true, nil
				}
				if e = out.copyCell(m, cellar, index); e != nil {
					return cellar, false, e
				}
				if e = out.setKey(m, index, key); e != nil {
					return cellar, false, e
				}
				if e = out.setNext(m, index, uint64(cellar)+1); e != nil {
					return cellar, false, e
				}
				cellar++
			}
			next, e = t.next(m, i)
			if e != nil {
				return cellar, false, e
			}
			if next == 0 {
				break
			}
			if next > uint64(t.capacity) {
				return cellar, false, arena.ErrBounds
			}
			i = uint32(next - 1)
		}
	}
	return cellar, false, nil
}
func (t *monomialTable) cleanup(m *MonomialManager) error {
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

// find is the non-mutating source contains/find_core traversal. Growth belongs
// only to insertion, including the pre-duplicate growth performed there.
func (t *monomialTable) find(m *MonomialManager, k monomialKey) (uint32, bool, error) {
	h, e := t.hash(m, k)
	if e != nil {
		return 0, false, e
	}
	if t.slots == 0 {
		return 0, false, arena.ErrBounds
	}
	i := h & (t.slots - 1)
	next, e := t.next(m, i)
	if e != nil {
		return 0, false, e
	}
	if next == monomialFreeLink {
		return 0, false, nil
	}
	for {
		v, e := t.key(m, i)
		if e != nil {
			return 0, false, e
		}
		eq, e := t.equal(m, v, k)
		if e != nil {
			return 0, false, e
		}
		if eq {
			return i, true, nil
		}
		t.collisions++
		next, e = t.next(m, i)
		if e != nil {
			return 0, false, e
		}
		if next == 0 {
			return 0, false, nil
		}
		i = uint32(next - 1)
	}
}
func (t *monomialTable) sourceDestroy(m *MonomialManager) error {
	if t.raw != (memory.Block{}) {
		if e := m.memory.Deallocate(&t.raw); e != nil {
			return e
		}
	}
	return nil
}
