// Source: src/util/hashtable.h:183-230,244-256,378-408;
// src/util/obj_hashtable.h; src/ast/expr2var.cpp:25-54.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Concrete obj_map<expr,unsigned> specialization. AST hashes, linear probing,
// source bucket-order rehashing and iteration are preserved. No entries are
// erased during this translator's lifetime; atom entries own one AST reference.
type satGoalASTEntry struct {
	key         ASTHandle
	hash, value uint32
	used        bool
}
type satGoalASTMap struct {
	cells []satGoalASTEntry
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
	for _, entry := range t.cells {
		if entry.used {
			i := int(entry.hash) & (n - 1)
			for cells[i].used {
				i = (i + 1) & (n - 1)
			}
			cells[i] = entry
		}
	}
	b.Release(uint64(len(t.cells)) * stride)
	t.cells = cells
	return nil
}
func (t *satGoalASTMap) prepare(b *arena.Budget) error {
	n := len(t.cells)
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
func (t *satGoalASTMap) find(h ASTHandle, hash uint32) (uint32, bool) {
	if len(t.cells) == 0 {
		return 0, false
	}
	mask := len(t.cells) - 1
	for i := int(hash) & mask; t.cells[i].used; i = (i + 1) & mask {
		if t.cells[i].hash == hash && t.cells[i].key == h {
			return t.cells[i].value, true
		}
	}
	return 0, false
}

// prepare must precede insertPrepared, allowing atom ownership to be acquired
// only after storage succeeds and before publishing the source map entry.
func (t *satGoalASTMap) insertPrepared(h ASTHandle, hash, value uint32) {
	mask := len(t.cells) - 1
	i := int(hash) & mask
	for t.cells[i].used {
		if t.cells[i].hash == hash && t.cells[i].key == h {
			t.cells[i].value = value
			return
		}
		i = (i + 1) & mask
	}
	t.cells[i] = satGoalASTEntry{h, hash, value, true}
	t.size++
}
func (t *satGoalASTMap) clear() { clear(t.cells); t.size = 0 }
func (t *satGoalASTMap) close(b *arena.Budget) {
	b.Release(uint64(len(t.cells)) * uint64(reflect.TypeOf(satGoalASTEntry{}).Size()))
	t.cells = nil
	t.size = 0
}
