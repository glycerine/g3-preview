// Source: src/util/obj_hashtable.h:61-154; src/util/hashtable.h:183-256,403-476,533-577.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

const (
	modelMapFree uint8 = iota
	modelMapDeleted
	modelMapUsed
)

// Concrete decl2expr cells contain handles, not interfaces or Go pointers.
// Probing uses the declaration's AST hash.
type modelConstantEntry struct {
	declaration, value           ASTHandle
	hash                         uint32
	state                        uint8
	declarationOwned, valueOwned bool
}
type modelConstantMap struct {
	cells         []modelConstantEntry
	size, deleted uint32
}

func (t *modelConstantMap) resize(m *ASTManager, n int) error {
	stride := uint64(reflect.TypeOf(modelConstantEntry{}).Size())
	if n < 8 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := m.budget.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]modelConstantEntry, n)
	for _, entry := range t.cells {
		if entry.state == modelMapUsed {
			i := int(entry.hash) & (n - 1)
			for next[i].state != modelMapFree {
				i = (i + 1) & (n - 1)
			}
			next[i] = entry
		}
	}
	m.budget.Release(uint64(len(t.cells)) * stride)
	t.cells = next
	t.deleted = 0
	return nil
}
func (t *modelConstantMap) find(d ASTHandle, hash uint32) int {
	if len(t.cells) == 0 {
		return -1
	}
	mask := len(t.cells) - 1
	for i := int(hash) & mask; t.cells[i].state != modelMapFree; i = (i + 1) & mask {
		entry := &t.cells[i]
		if entry.state == modelMapUsed && entry.hash == hash && entry.declaration == d {
			return i
		}
	}
	return -1
}
func (t *modelConstantMap) insertIfAbsent(m *ASTManager, d ASTHandle, hash uint32) (int, error) {
	n := len(t.cells)
	if n == 0 {
		if e := t.resize(m, 8); e != nil {
			return -1, e
		}
	} else if uint64(t.size+t.deleted)*4 > uint64(n)*3 {
		if n > int(^uint(0)>>1)/2 {
			return -1, arena.ErrBounds
		}
		if e := t.resize(m, n*2); e != nil {
			return -1, e
		}
	}
	mask := len(t.cells) - 1
	deleted := -1
	for i := int(hash) & mask; ; i = (i + 1) & mask {
		entry := &t.cells[i]
		switch entry.state {
		case modelMapUsed:
			if entry.hash == hash && entry.declaration == d {
				return i, nil
			}
		case modelMapDeleted:
			deleted = i
		case modelMapFree:
			if deleted != -1 {
				i = deleted
				t.deleted--
			}
			t.cells[i] = modelConstantEntry{declaration: d, hash: hash, state: modelMapUsed}
			t.size++
			return i, nil
		}
	}
}
func (t *modelConstantMap) remove(m *ASTManager, d ASTHandle, hash uint32) error {
	i := t.find(d, hash)
	if i < 0 {
		return nil
	}
	t.size--
	if t.cells[(i+1)&(len(t.cells)-1)].state == modelMapFree {
		t.cells[i] = modelConstantEntry{}
	} else {
		t.cells[i] = modelConstantEntry{state: modelMapDeleted}
		t.deleted++
		if t.deleted > t.size && t.deleted > 64 {
			if m.exprIDs.outOfMemory == nil || !m.exprIDs.outOfMemory() {
				return t.resize(m, len(t.cells))
			}
		}
	}
	return nil
}
func (t *modelConstantMap) close(m *ASTManager) {
	m.budget.Release(uint64(len(t.cells)) * uint64(reflect.TypeOf(modelConstantEntry{}).Size()))
	t.cells = nil
	t.size = 0
	t.deleted = 0
}

type modelDeclarationVector struct{ values []ASTHandle }

func (v *modelDeclarationVector) push(m *ASTManager, d ASTHandle) error {
	if len(v.values) == cap(v.values) {
		stride := uint64(reflect.TypeOf(d).Size())
		n, e := satGoalCapacity(cap(v.values), stride)
		if e != nil {
			return e
		}
		if e = m.budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]ASTHandle, len(v.values), n)
		copy(next, v.values)
		m.budget.Release(uint64(cap(v.values)) * stride)
		v.values = next
	}
	v.values = append(v.values, d)
	return nil
}
func (v *modelDeclarationVector) erase(d ASTHandle) {
	for i, h := range v.values {
		if h == d {
			copy(v.values[i:], v.values[i+1:])
			v.values[len(v.values)-1] = ASTHandle{}
			v.values = v.values[:len(v.values)-1]
			return
		}
	}
}
func (v *modelDeclarationVector) close(m *ASTManager) {
	m.budget.Release(uint64(cap(v.values)) * uint64(reflect.TypeOf(ASTHandle{}).Size()))
	v.values = nil
}
