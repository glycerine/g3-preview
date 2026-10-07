// Source: src/model/func_interp.cpp:234-255; src/util/obj_hashtable.h.
package model

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"reflect"
)

// Concrete obj_map<expr,unsigned> cells retain borrowed scalar handles.
type functionOccurrenceCell struct {
	key         ast.ASTHandle
	hash, count uint32
}
type functionOccurrenceMap struct {
	cells []functionOccurrenceCell
	size  uint32
	bytes uint64
}

func (t *functionOccurrenceMap) grow(m *ast.ASTManager) error {
	n := 8
	if len(t.cells) > 0 {
		if len(t.cells) > int(^uint(0)>>1)/2 {
			return arena.ErrBounds
		}
		n = 2 * len(t.cells)
	}
	width := uint64(reflect.TypeOf(functionOccurrenceCell{}).Size())
	if uint64(n) > uint64(^uint(0)>>1)/width {
		return arena.ErrBounds
	}
	bytes := uint64(n) * width
	if e := m.Budget().Reserve(bytes); e != nil {
		return e
	}
	next := make([]functionOccurrenceCell, n)
	for _, cell := range t.cells {
		if !cell.key.IsNull() {
			i := int(cell.hash) & (n - 1)
			for !next[i].key.IsNull() {
				i = (i + 1) & (n - 1)
			}
			next[i] = cell
		}
	}
	m.Budget().Release(t.bytes)
	t.cells = next
	t.bytes = bytes
	return nil
}
func (t *functionOccurrenceMap) increment(m *ast.ASTManager, key ast.ASTHandle) (uint32, error) {
	hash, e := m.GetHash(key)
	if e != nil {
		return 0, e
	}
	if len(t.cells) == 0 || uint64(t.size)*4 > uint64(len(t.cells))*3 {
		if e = t.grow(m); e != nil {
			return 0, e
		}
	}
	mask := len(t.cells) - 1
	for i := int(hash) & mask; ; i = (i + 1) & mask {
		cell := &t.cells[i]
		if cell.key.IsNull() {
			cell.key = key
			cell.hash = hash
			t.size++
		}
		if cell.key == key && cell.hash == hash {
			cell.count++
			return cell.count, nil
		}
	}
}

// MaxOccurrenceResult uses the source running count and strict greater-than
// comparison. Ties retain whichever result first reached that running maximum.
func (f *FunctionInterpretation) MaxOccurrenceResult() (ast.ASTHandle, error) {
	if e := f.open(); e != nil {
		return ast.ASTHandle{}, e
	}
	if len(f.entries) == 0 {
		return ast.ASTHandle{}, nil
	}
	var counts functionOccurrenceMap
	defer func() { f.manager.Budget().Release(counts.bytes) }()
	var result ast.ASTHandle
	max := uint32(0)
	for _, entry := range f.entries {
		count, e := counts.increment(f.manager, entry.result)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if count > max {
			max = count
			result = entry.result
		}
	}
	return result, nil
}
