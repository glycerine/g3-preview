// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// src/ast/ast.h::ast_table specialization of src/util/chashtable.h.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type astChainCell struct {
	node astNodeRef
	next uint32
}
type astChainTable struct {
	cells                                     []astChainCell
	slots, used, size, next, free, collisions uint32
}

func astChainBytes(n uint32) uint64 { return uint64(n) * uint64(reflect.TypeOf(astChainCell{}).Size()) }
func (m *ASTManager) allocateChain(slots, cellar uint32) ([]astChainCell, error) {
	n := uint64(slots) + uint64(cellar)
	if slots == 0 || slots&(slots-1) != 0 || cellar == 0 || n >= uint64(chainEnd) || n > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(astChainCell{}).Size()) {
		return nil, arena.ErrBounds
	}
	if e := m.budget.Reserve(astChainBytes(uint32(n))); e != nil {
		return nil, e
	}
	a := make([]astChainCell, int(n))
	for i := range a {
		a[i].next = chainFree
	}
	return a, nil
}
func (m *ASTManager) initTable() error {
	a, e := m.allocateChain(8, 2)
	if e != nil {
		return e
	}
	m.table = astChainTable{cells: a, slots: 8, next: 8, free: chainEnd}
	return nil
}
func (m *ASTManager) astCopyTable(a []astChainCell, slots uint32) (uint32, bool) {
	t := &m.table
	t.used = 0
	next := slots
	for bucket := uint32(0); bucket < t.slots; bucket++ {
		if t.cells[bucket].next == chainFree {
			continue
		}
		for i := bucket; i != chainEnd; i = t.cells[i].next {
			index := m.nodeAt(t.cells[i].node.slot).hash & (slots - 1)
			if a[index].next == chainFree {
				a[index] = t.cells[i]
				a[index].next = chainEnd
				t.used++
			} else {
				if next == uint32(len(a)) {
					return next, false
				}
				a[next] = a[index]
				a[index] = t.cells[i]
				a[index].next = next
				next++
			}
		}
	}
	return next, true
}
func (m *ASTManager) expandTable() error {
	slots := uint64(m.table.slots) * 2
	cellar := uint64(len(m.table.cells)-int(m.table.slots)) * 2
	for {
		if slots >= uint64(chainEnd) || cellar >= uint64(chainEnd) {
			return arena.ErrBounds
		}
		a, e := m.allocateChain(uint32(slots), uint32(cellar))
		if e != nil {
			return e
		}
		next, ok := m.astCopyTable(a, uint32(slots))
		if ok {
			m.budget.Release(astChainBytes(uint32(len(m.table.cells))))
			m.table.cells = a
			m.table.slots = uint32(slots)
			m.table.next = next
			m.table.free = chainEnd
			return nil
		}
		m.budget.Release(astChainBytes(uint32(len(a))))
		cellar *= 2
	}
}
func (m *ASTManager) tableFind(key astNodeRef) (astNodeRef, bool, error) {
	t := &m.table
	hash := m.nodeAt(key.slot).hash
	i := hash & (t.slots - 1)
	if t.cells[i].next == chainFree {
		return astNodeRef{}, false, nil
	}
	for ; i != chainEnd; i = t.cells[i].next {
		other := t.cells[i].node
		if hash == m.nodeAt(other.slot).hash {
			eq, e := m.compareNodes(other, key)
			if e != nil {
				return astNodeRef{}, false, e
			}
			if eq {
				return other, true, nil
			}
		}
		t.collisions++
	}
	return astNodeRef{}, false, nil
}
func (m *ASTManager) tableInsert(key astNodeRef) (astNodeRef, bool, error) {
	t := &m.table
	if t.free == chainEnd && t.next == uint32(len(t.cells)) {
		if e := m.expandTable(); e != nil {
			return astNodeRef{}, false, e
		}
	}
	if old, found, e := m.tableFind(key); e != nil || found {
		return old, false, e
	}
	idx := m.nodeAt(key.slot).hash & (t.slots - 1)
	if t.cells[idx].next == chainFree {
		t.cells[idx] = astChainCell{key, chainEnd}
		t.used++
	} else {
		j := t.next
		if t.free != chainEnd {
			j = t.free
			t.free = t.cells[j].next
		} else {
			t.next++
		}
		t.cells[j] = t.cells[idx]
		t.cells[idx] = astChainCell{key, j}
	}
	t.size++
	return key, true, nil
}
func (m *ASTManager) tableEraseIdentity(key astNodeRef) error {
	t := &m.table
	i := m.nodeAt(key.slot).hash & (t.slots - 1)
	prev := uint32(math.MaxUint32)
	if t.cells[i].next == chainFree {
		return ErrASTHandle
	}
	for i != chainEnd {
		if t.cells[i].node == key {
			t.size--
			if prev == math.MaxUint32 {
				j := t.cells[i].next
				if j == chainEnd {
					t.used--
					t.cells[i].next = chainFree
				} else {
					t.cells[i] = t.cells[j]
					t.cells[j].next = t.free
					t.free = j
				}
			} else {
				t.cells[prev].next = t.cells[i].next
				t.cells[i].next = t.free
				t.free = i
			}
			return nil
		}
		t.collisions++
		prev = i
		i = t.cells[i].next
	}
	return ErrASTHandle
}

// Full three-register macro state is needed for successive source array
// triples; utilhash.Mix exports only its final c register.
func astMix(a, b, c uint32) (uint32, uint32, uint32) {
	a -= b
	a -= c
	a ^= c >> 13
	b -= c
	b -= a
	b ^= a << 8
	c -= a
	c -= b
	c ^= b >> 13
	a -= b
	a -= c
	a ^= c >> 12
	b -= c
	b -= a
	b ^= a << 16
	c -= a
	c -= b
	c ^= b >> 5
	a -= b
	a -= c
	a ^= c >> 3
	b -= c
	b -= a
	b ^= a << 10
	c -= a
	c -= b
	c ^= b >> 15
	return a, b, c
}
