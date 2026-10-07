// Licensed under the MIT license.
// Source: src/ast/ast.h:920-924 (ast_table), src/util/chashtable.h.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
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
	if slots == 0 || slots&(slots-1) != 0 || cellar == 0 || n >= uint64(util.ChainEnd) || n > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(astChainCell{}).Size()) {
		return nil, arena.ErrBounds
	}
	if e := m.BudgetField.Reserve(astChainBytes(uint32(n))); e != nil {
		return nil, e
	}
	a := make([]astChainCell, int(n))
	for i := range a {
		a[i].next = util.ChainFree
	}
	return a, nil
}
func (m *ASTManager) initTable() error {
	a, e := m.allocateChain(8, 2)
	if e != nil {
		return e
	}
	m.table = astChainTable{cells: a, slots: 8, next: 8, free: util.ChainEnd}
	return nil
}
func (m *ASTManager) astCopyTable(a []astChainCell, slots uint32) (uint32, bool, error) {
	t := &m.table
	t.used = 0
	next := slots
	for bucket := uint32(0); bucket < t.slots; bucket++ {
		if t.cells[bucket].next == util.ChainFree {
			continue
		}
		for i := bucket; i != util.ChainEnd; i = t.cells[i].next {
			if e := m.validNode(t.cells[i].node); e != nil {
				return next, false, e
			}
			index := m.nodeAt(t.cells[i].node.slot).hash & (slots - 1)
			if a[index].next == util.ChainFree {
				a[index] = t.cells[i]
				a[index].next = util.ChainEnd
				t.used++
			} else {
				if next == uint32(len(a)) {
					return next, false, nil
				}
				a[next] = a[index]
				a[index] = t.cells[i]
				a[index].next = next
				next++
			}
		}
	}
	return next, true, nil
}
func (m *ASTManager) expandTable() error {
	slots := uint64(m.table.slots) * 2
	cellar := uint64(len(m.table.cells)-int(m.table.slots)) * 2
	for {
		if slots >= uint64(util.ChainEnd) || cellar >= uint64(util.ChainEnd) {
			return arena.ErrBounds
		}
		a, e := m.allocateChain(uint32(slots), uint32(cellar))
		if e != nil {
			return e
		}
		next, ok, copyError := m.astCopyTable(a, uint32(slots))
		if copyError != nil {
			m.BudgetField.Release(astChainBytes(uint32(len(a))))
			return copyError
		}
		if ok {
			m.BudgetField.Release(astChainBytes(uint32(len(m.table.cells))))
			m.table.cells = a
			m.table.slots = uint32(slots)
			m.table.next = next
			m.table.free = util.ChainEnd
			return nil
		}
		m.BudgetField.Release(astChainBytes(uint32(len(a))))
		cellar *= 2
	}
}
func (m *ASTManager) tableFind(key astNodeRef) (astNodeRef, bool, error) {
	t := &m.table
	hash := m.nodeAt(key.slot).hash
	i := hash & (t.slots - 1)
	if t.cells[i].next == util.ChainFree {
		return astNodeRef{}, false, nil
	}
	for ; i != util.ChainEnd; i = t.cells[i].next {
		other := t.cells[i].node
		if e := m.validNode(other); e != nil {
			return astNodeRef{}, false, e
		}
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
	if t.free == util.ChainEnd && t.next == uint32(len(t.cells)) {
		if e := m.expandTable(); e != nil {
			return astNodeRef{}, false, e
		}
	}
	if old, found, e := m.tableFind(key); e != nil || found {
		return old, false, e
	}
	idx := m.nodeAt(key.slot).hash & (t.slots - 1)
	if t.cells[idx].next == util.ChainFree {
		t.cells[idx] = astChainCell{key, util.ChainEnd}
		t.used++
	} else {
		j := t.next
		if t.free != util.ChainEnd {
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
	return m.tableEraseCaptured(key, m.nodeAt(key.slot).hash)
}

// Owning-only matching erasure uses the actual captured insertion token/hash,
// without dereferencing a source-freed placement or a replacement generation.
func (m *ASTManager) tableEraseCaptured(key astNodeRef, hash uint32) error {
	t := &m.table
	i := hash & (t.slots - 1)
	prev := uint32(math.MaxUint32)
	if t.cells[i].next == util.ChainFree {
		return ErrASTHandle
	}
	for i != util.ChainEnd {
		if t.cells[i].node == key {
			t.size--
			if prev == math.MaxUint32 {
				j := t.cells[i].next
				if j == util.ChainEnd {
					t.used--
					t.cells[i].next = util.ChainFree
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
