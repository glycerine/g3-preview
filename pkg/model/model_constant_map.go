// Source: src/util/obj_hashtable.h:61-154; src/util/hashtable.h:183-256,403-476,533-577.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package model

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

const (
	modelMapFree uint8 = iota
	modelMapDeleted
	modelMapUsed
)

// Concrete decl2expr cells contain handles, not interfaces or Go pointers.
// Probing uses the declaration's AST hash.
type modelConstantEntry struct {
	declaration, value           ast.ASTHandle
	hash                         uint32
	state                        uint8
	declarationOwned, valueOwned bool
}
type modelConstantMap struct {
	Cells         []modelConstantEntry
	size, deleted uint32
}

func (t *modelConstantMap) resize(m *ast.ASTManager, n int) error {
	stride := uint64(reflect.TypeOf(modelConstantEntry{}).Size())
	if n < 8 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := m.Budget().Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]modelConstantEntry, n)
	for _, entry := range t.Cells {
		if entry.state == modelMapUsed {
			i := int(entry.hash) & (n - 1)
			for next[i].state != modelMapFree {
				i = (i + 1) & (n - 1)
			}
			next[i] = entry
		}
	}
	m.Budget().Release(uint64(len(t.Cells)) * stride)
	t.Cells = next
	t.deleted = 0
	return nil
}
func (t *modelConstantMap) find(d ast.ASTHandle, hash uint32) int {
	if len(t.Cells) == 0 {
		return -1
	}
	mask := len(t.Cells) - 1
	for i := int(hash) & mask; t.Cells[i].state != modelMapFree; i = (i + 1) & mask {
		entry := &t.Cells[i]
		if entry.state == modelMapUsed && entry.hash == hash && entry.declaration == d {
			return i
		}
	}
	return -1
}
func (t *modelConstantMap) insertIfAbsent(m *ast.ASTManager, d ast.ASTHandle, hash uint32) (int, error) {
	n := len(t.Cells)
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
	mask := len(t.Cells) - 1
	deleted := -1
	for i := int(hash) & mask; ; i = (i + 1) & mask {
		entry := &t.Cells[i]
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
			t.Cells[i] = modelConstantEntry{declaration: d, hash: hash, state: modelMapUsed}
			t.size++
			return i, nil
		}
	}
}
func (t *modelConstantMap) remove(m *ast.ASTManager, d ast.ASTHandle, hash uint32) error {
	i := t.find(d, hash)
	if i < 0 {
		return nil
	}
	t.size--
	if t.Cells[(i+1)&(len(t.Cells)-1)].state == modelMapFree {
		t.Cells[i] = modelConstantEntry{}
	} else {
		t.Cells[i] = modelConstantEntry{state: modelMapDeleted}
		t.deleted++
		if t.deleted > t.size && t.deleted > 64 {
			if !m.IsOutOfMemory() {
				return t.resize(m, len(t.Cells))
			}
		}
	}
	return nil
}
func (t *modelConstantMap) close(m *ast.ASTManager) {
	m.Budget().Release(uint64(len(t.Cells)) * uint64(reflect.TypeOf(modelConstantEntry{}).Size()))
	t.Cells = nil
	t.size = 0
	t.deleted = 0
}

type modelDeclarationVector struct {
	util.PointerVector[ast.ASTHandle]
}

func (v *modelDeclarationVector) push(m *ast.ASTManager, d ast.ASTHandle) error {
	return v.Push(m.Budget(), d)
}
func (v *modelDeclarationVector) erase(d ast.ASTHandle)   { v.Erase(d) }
func (v *modelDeclarationVector) close(m *ast.ASTManager) { v.Close(m.Budget()) }
