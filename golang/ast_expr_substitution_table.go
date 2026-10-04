// Licensed under the MIT license.
// Selected obj_map/core_hashtable algorithms, with explicit callback metadata.
package g3

import "github.com/glycerine/g3/golang/arena"

func (m *ASTManager) substEntry(p arena.SmallBytesRef, i uint32) (astExprSubstEntry, error) {
	var a [6]uint32
	for j := range a {
		x, e := m.vectorWord(p, i*24+uint32(j)*4)
		if e != nil {
			return astExprSubstEntry{}, e
		}
		a[j] = x
	}
	return astExprSubstEntry{astNodeRef{a[0], a[1]}, astNodeRef{a[2], a[3]}, a[4], a[5]}, nil
}
func (m *ASTManager) substPutEntry(p arena.SmallBytesRef, i uint32, e astExprSubstEntry) error {
	for j, x := range [6]uint32{e.key.slot, e.key.generation, e.value.slot, e.value.generation, e.state, e.owned} {
		if err := m.vectorPutWord(p, i*24+uint32(j)*4, x); err != nil {
			return err
		}
	}
	return nil
}
func (m *ASTManager) substHash(key astNodeRef) (uint32, error) {
	if e := m.validNode(key); e != nil {
		return 0, e
	}
	n := m.nodeAt(key.slot)
	if n.kind != ASTApp && n.kind != ASTVar && n.kind != ASTQuantifier {
		return 0, ErrASTDomain
	}
	return n.hash, nil
}
func (m *ASTManager) substFind(t astExprSubstTable, key astNodeRef) (uint32, bool, error) {
	hash, e := m.substHash(key)
	if e != nil {
		return 0, false, e
	}
	if t.payload == (arena.SmallBytesRef{}) || t.capacity == 0 {
		return 0, false, ErrASTPending
	}
	for k := uint32(0); k < t.capacity; k++ {
		i := (hash + k) & (t.capacity - 1)
		a, e := m.substEntry(t.payload, i)
		if e != nil {
			return 0, false, e
		}
		if a.state == 0 {
			return i, false, nil
		}
		if a.state == 1 {
			h, e := m.substHash(a.key)
			if e != nil {
				return 0, false, e
			}
			if h == hash && a.key == key {
				return i, true, nil
			}
		}
	}
	return 0, false, nil
}
func (m *ASTManager) substRehash(s ASTExprSubstitution, which uint32, t *astExprSubstTable, capacity uint32) error {
	p, e := m.substAllocTable(capacity)
	if e != nil {
		return e
	}
	fail := func(err error) error { m.vectorAllocator.Deallocate(capacity*24, p); return err }
	for i := uint32(0); i < t.capacity; i++ {
		a, e := m.substEntry(t.payload, i)
		if e != nil {
			return fail(e)
		}
		if a.state != 1 {
			continue
		}
		h, e := m.substHash(a.key)
		if e != nil {
			return fail(e)
		}
		inserted := false
		for k := uint32(0); k < capacity; k++ {
			j := (h + k) & (capacity - 1)
			b, e := m.substEntry(p, j)
			if e != nil {
				return fail(e)
			}
			if b.state == 0 {
				if e = m.substPutEntry(p, j, a); e != nil {
					return fail(e)
				}
				inserted = true
				break
			}
		}
		if !inserted {
			return fail(ErrASTDomain)
		}
	}
	if e = m.vectorAllocator.Deallocate(t.capacity*24, t.payload); e != nil {
		return fail(e)
	}
	t.payload = p
	t.capacity = capacity
	t.deleted = 0
	return m.substStoreTable(s, which, *t)
}
func (m *ASTManager) substInsertSlot(s ASTExprSubstitution, which uint32, key astNodeRef, value astNodeRef, replace bool) (uint32, error) {
	t, e := m.substTable(s, which)
	if e != nil {
		return 0, e
	}
	if t.flags != 0 || t.payload == (arena.SmallBytesRef{}) {
		return 0, ErrASTPending
	}
	if (t.size+t.deleted)<<2 > t.capacity*3 {
		if t.capacity > ^uint32(0)/2 {
			return 0, arena.ErrBounds
		}
		if e = m.substRehash(s, which, &t, t.capacity<<1); e != nil {
			return 0, e
		}
	}
	h, e := m.substHash(key)
	if e != nil {
		return 0, e
	}
	del := uint32(0)
	haveDel := false
	for k := uint32(0); k < t.capacity; k++ {
		i := (h + k) & (t.capacity - 1)
		a, e := m.substEntry(t.payload, i)
		if e != nil {
			return 0, e
		}
		if a.state == 1 {
			hh, e := m.substHash(a.key)
			if e != nil {
				return 0, e
			}
			if hh == h && a.key == key {
				if replace {
					a.value = value
					a.owned &^= 2
					if e = m.substPutEntry(t.payload, i, a); e != nil {
						return 0, e
					}
				}
				return i, nil
			}
			continue
		}
		if a.state == 2 {
			del = i
			haveDel = true
			continue
		}
		if haveDel {
			i = del
			t.deleted--
		}
		if e = m.substPutEntry(t.payload, i, astExprSubstEntry{key: key, value: value, state: 1}); e != nil {
			return 0, e
		}
		t.size++
		if e = m.substStoreTable(s, which, t); e != nil {
			return 0, e
		}
		return i, nil
	}
	return 0, ErrASTDomain
}
func (m *ASTManager) substEraseSlot(s ASTExprSubstitution, which uint32, key astNodeRef) error {
	t, e := m.substTable(s, which)
	if e != nil {
		return e
	}
	i, found, e := m.substFind(t, key)
	if e != nil || !found {
		return e
	}
	a, e := m.substEntry(t.payload, i)
	if e != nil {
		return e
	}
	next, e := m.substEntry(t.payload, (i+1)&(t.capacity-1))
	if e != nil {
		return e
	}
	if next.state == 0 {
		a.state = 0
	} else {
		a.state = 2
		t.deleted++
	}
	a.owned = 0
	t.size--
	if e = m.substPutEntry(t.payload, i, a); e != nil {
		return e
	}
	if e = m.substStoreTable(s, which, t); e != nil {
		return e
	}
	if t.deleted > t.size && t.deleted > 64 {
		if m.exprIDs.outOfMemory != nil && m.exprIDs.outOfMemory() {
			return nil
		}
		return m.substRehash(s, which, &t, t.capacity)
	}
	return nil
}
func (m *ASTManager) substRawExpr(h ASTHandle) (astNodeRef, error) {
	if h.IsNull() {
		return astNodeRef{}, nil
	}
	if h.origin != m.origin {
		return astNodeRef{}, ErrASTHandle
	}
	return astNodeRef{h.slot, h.generation}, nil
}
func (m *ASTManager) substRawDependency(h ASTDependency) (astNodeRef, error) {
	if h.IsNull() {
		return astNodeRef{}, nil
	}
	if h.origin != m.origin {
		return astNodeRef{}, ErrASTHandle
	}
	return astNodeRef{uint32(h.record.Slot), uint32(h.record.Generation)}, nil
}
func (m *ASTManager) substDependency(r astNodeRef) ASTDependency {
	if r.slot == 0 {
		return ASTDependency{}
	}
	return ASTDependency{m.origin, dependencyRefPointer(r)}
}
