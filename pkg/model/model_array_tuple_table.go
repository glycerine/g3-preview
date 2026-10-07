// Source: src/model/model_evaluator.cpp:337-358;
// src/util/hash.h:72-122; src/util/hashtable.h:183-256,351-403,494-509,546-577.
package model

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	"reflect"
)

type arrayTupleCell struct {
	row   ast.ASTRefVector
	hash  uint32
	state uint8
}

// Source args_table borrows the extracted row storage; it owns only its cells.
type arrayTupleTable struct {
	cells                []arrayTupleCell
	arity, size, deleted uint32
}

func (t *arrayTupleTable) resize(m *ast.ASTManager, n int) error {
	width := uint64(reflect.TypeOf(arrayTupleCell{}).Size())
	if n < 8 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/width {
		return arena.ErrBounds
	}
	if e := m.Budget().Reserve(uint64(n) * width); e != nil {
		return e
	}
	next := make([]arrayTupleCell, n)
	for _, entry := range t.cells {
		if entry.state == modelMapUsed {
			i := int(entry.hash) & (n - 1)
			for next[i].state != modelMapFree {
				i = (i + 1) & (n - 1)
			}
			next[i] = entry
		}
	}
	m.Budget().Release(uint64(len(t.cells)) * width)
	t.cells = next
	t.deleted = 0
	return nil
}
func (t *arrayTupleTable) hash(m *ast.ASTManager, row ast.ASTRefVector) (uint32, error) {
	return arrayCompositeHash(t.arity, func(i uint32) (uint32, error) {
		h, e := m.RefVectorGet(row, i)
		if e != nil {
			return 0, e
		}
		return m.GetHash(h)
	})
}
func arrayCompositeHash(n uint32, child func(uint32) (uint32, error)) (uint32, error) {
	if n == 0 {
		return 0, ast.ErrASTDomain
	}
	a, b, c := uint32(0x9e3779b9), uint32(0x9e3779b9), uint32(11)
	switch n {
	case 1:
		a += 17
		h, e := child(0)
		if e != nil {
			return 0, e
		}
		b = h
		return util.Mix(a, b, c), nil
	case 2:
		a += 17
		h, e := child(0)
		if e != nil {
			return 0, e
		}
		b += h
		h, e = child(1)
		if e != nil {
			return 0, e
		}
		c += h
		return util.Mix(a, b, c), nil
	case 3:
		h, e := child(0)
		if e != nil {
			return 0, e
		}
		a += h
		h, e = child(1)
		if e != nil {
			return 0, e
		}
		b += h
		h, e = child(2)
		if e != nil {
			return 0, e
		}
		c += h
		a, b, c = util.MixState(a, b, c)
		a += 17
		return util.Mix(a, b, c), nil
	default:
		for n >= 3 {
			n--
			h, e := child(n)
			if e != nil {
				return 0, e
			}
			a += h
			n--
			h, e = child(n)
			if e != nil {
				return 0, e
			}
			b += h
			n--
			h, e = child(n)
			if e != nil {
				return 0, e
			}
			c += h
			a, b, c = util.MixState(a, b, c)
		}
		a += 17
		if n == 2 {
			h, e := child(1)
			if e != nil {
				return 0, e
			}
			b += h
		}
		if n >= 1 {
			h, e := child(0)
			if e != nil {
				return 0, e
			}
			c += h
		}
		return util.Mix(a, b, c), nil
	}
}
func (t *arrayTupleTable) equal(m *ast.ASTManager, a, b ast.ASTRefVector) (bool, error) {
	for i := uint32(0); i < t.arity; i++ {
		x, e := m.RefVectorGet(a, i)
		if e != nil {
			return false, e
		}
		y, e := m.RefVectorGet(b, i)
		if e != nil || x != y {
			return false, e
		}
	}
	return true, nil
}
func (t *arrayTupleTable) find(m *ast.ASTManager, row ast.ASTRefVector) (int, error) {
	hash, e := t.hash(m, row)
	if e != nil {
		return -1, e
	}
	if len(t.cells) == 0 {
		return -1, nil
	}
	mask := len(t.cells) - 1
	for i := int(hash) & mask; t.cells[i].state != modelMapFree; i = (i + 1) & mask {
		entry := t.cells[i]
		if entry.state == modelMapUsed && entry.hash == hash {
			same, e := t.equal(m, entry.row, row)
			if e != nil {
				return -1, e
			}
			if same {
				return i, nil
			}
		}
	}
	return -1, nil
}
func (t *arrayTupleTable) insert(m *ast.ASTManager, row ast.ASTRefVector) error {
	if uint64(t.size+t.deleted)*4 > uint64(len(t.cells))*3 {
		if len(t.cells) > int(^uint(0)>>1)/2 {
			return arena.ErrBounds
		}
		if e := t.resize(m, len(t.cells)*2); e != nil {
			return e
		}
	}
	hash, e := t.hash(m, row)
	if e != nil {
		return e
	}
	mask := len(t.cells) - 1
	deleted := -1
	for i := int(hash) & mask; ; i = (i + 1) & mask {
		entry := &t.cells[i]
		switch entry.state {
		case modelMapUsed:
			if entry.hash == hash {
				same, e := t.equal(m, entry.row, row)
				if e != nil {
					return e
				}
				if same {
					entry.row = row
					return nil
				}
			}
		case modelMapDeleted:
			deleted = i
		case modelMapFree:
			if deleted >= 0 {
				i = deleted
				t.deleted--
			}
			t.cells[i] = arrayTupleCell{row: row, hash: hash, state: modelMapUsed}
			t.size++
			return nil
		}
	}
}
func (t *arrayTupleTable) remove(m *ast.ASTManager, row ast.ASTRefVector) error {
	i, e := t.find(m, row)
	if e != nil || i < 0 {
		return e
	}
	t.size--
	if t.cells[(i+1)&(len(t.cells)-1)].state == modelMapFree {
		t.cells[i] = arrayTupleCell{}
	} else {
		t.cells[i] = arrayTupleCell{state: modelMapDeleted}
		t.deleted++
		if t.deleted > t.size && t.deleted > 64 && !m.IsOutOfMemory() {
			return t.resize(m, len(t.cells))
		}
	}
	return nil
}
func (t *arrayTupleTable) close(m *ast.ASTManager) {
	m.Budget().Release(uint64(len(t.cells)) * uint64(reflect.TypeOf(arrayTupleCell{}).Size()))
	*t = arrayTupleTable{}
}
