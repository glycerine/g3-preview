// Source: src/util/hashtable.h:183-230,244-256,378-408;
// src/util/obj_hashtable.h; src/ast/expr2var.cpp:25-54.
package tactic

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

// Concrete obj_map<expr,unsigned> specialization. AST hashes, linear probing,
// source bucket-order rehashing and iteration are preserved. No entries are
// erased during this translator's lifetime; atom entries own one AST reference.
type satGoalASTEntry struct {
	Key         ast.ASTHandle
	hash, value uint32
	Used        bool
}
type satGoalASTMap struct {
	Cells []satGoalASTEntry
	size  uint32
}

func (t *satGoalASTMap) resize(b *arena.Budget, n int) error {
	stride := uint64(reflect.TypeOf(satGoalASTEntry{}).Size())
	if n < 8 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := b.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	cells := make([]satGoalASTEntry, n)
	for _, entry := range t.Cells {
		if entry.Used {
			i := int(entry.hash) & (n - 1)
			for cells[i].Used {
				i = (i + 1) & (n - 1)
			}
			cells[i] = entry
		}
	}
	b.Release(uint64(len(t.Cells)) * stride)
	t.Cells = cells
	return nil
}
func (t *satGoalASTMap) prepare(b *arena.Budget) error {
	n := len(t.Cells)
	if n == 0 {
		return t.resize(b, 8)
	}
	if uint64(t.size)*4 > uint64(n)*3 {
		if n > int(^uint(0)>>1)/2 {
			return arena.ErrBounds
		}
		return t.resize(b, n*2)
	}
	return nil
}
func (t *satGoalASTMap) find(h ast.ASTHandle, hash uint32) (uint32, bool) {
	if len(t.Cells) == 0 {
		return 0, false
	}
	mask := len(t.Cells) - 1
	for i := int(hash) & mask; t.Cells[i].Used; i = (i + 1) & mask {
		if t.Cells[i].hash == hash && t.Cells[i].Key == h {
			return t.Cells[i].value, true
		}
	}
	return 0, false
}

// prepare must precede insertPrepared, allowing atom ownership to be acquired
// only after storage succeeds and before publishing the source map entry.
func (t *satGoalASTMap) insertPrepared(h ast.ASTHandle, hash, value uint32) {
	mask := len(t.Cells) - 1
	i := int(hash) & mask
	for t.Cells[i].Used {
		if t.Cells[i].hash == hash && t.Cells[i].Key == h {
			t.Cells[i].value = value
			return
		}
		i = (i + 1) & mask
	}
	t.Cells[i] = satGoalASTEntry{h, hash, value, true}
	t.size++
}
func (t *satGoalASTMap) clear() { clear(t.Cells); t.size = 0 }
func (t *satGoalASTMap) close(b *arena.Budget) {
	b.Release(uint64(len(t.Cells)) * uint64(reflect.TypeOf(satGoalASTEntry{}).Size()))
	t.Cells = nil
	t.size = 0
}
