// Source: src/util/obj_hashtable.h:58-105,144-159;
// src/util/hashtable.h:155-214,219-227,244-256,377-396,494-510;
// src/ast/rewriter/bv_trailing.cpp:338-379.
package rewriter

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

type bvTrailingEntry struct {
	key      ast.ASTHandle
	min, max uint32
	owned    bool
}
type bvTrailingTable struct {
	entries       []bvTrailingEntry
	size, cursor  uint32
	header, bytes uint64
}

func bvTrailingEntries(m *ast.ASTManager, capacity uint32) ([]bvTrailingEntry, uint64, error) {
	size := uint64(reflect.TypeOf(bvTrailingEntry{}).Size())
	bytes := uint64(capacity) * size
	if capacity == 0 || bytes > uint64(^uint(0)>>1) {
		return nil, 0, arena.ErrBounds
	}
	if e := m.Budget().Reserve(bytes); e != nil {
		return nil, 0, e
	}
	return make([]bvTrailingEntry, capacity), bytes, nil
}
func newBVTrailingTable(m *ast.ASTManager) (*bvTrailingTable, error) {
	header := uint64(reflect.TypeOf(bvTrailingTable{}).Size())
	if e := m.Budget().Reserve(header); e != nil {
		return nil, e
	}
	entries, bytes, e := bvTrailingEntries(m, 8)
	if e != nil {
		m.Budget().Release(header)
		return nil, e
	}
	return &bvTrailingTable{entries: entries, header: header, bytes: bytes}, nil
}
func (t *bvTrailingTable) find(m *ast.ASTManager, key ast.ASTHandle) (bvTrailingEntry, bool, error) {
	hash, e := m.GetHash(key)
	if e != nil {
		return bvTrailingEntry{}, false, e
	}
	mask := uint32(len(t.entries) - 1)
	for offset := uint32(0); offset < uint32(len(t.entries)); offset++ {
		entry := t.entries[(hash+offset)&mask]
		if entry.key.IsNull() {
			return bvTrailingEntry{}, false, nil
		}
		stored, e := m.GetHash(entry.key)
		if e != nil {
			return bvTrailingEntry{}, false, e
		}
		if stored == hash && entry.key == key {
			return entry, true, nil
		}
	}
	return bvTrailingEntry{}, false, nil
}
func (t *bvTrailingTable) insert(m *ast.ASTManager, key ast.ASTHandle, min, max uint32) error {
	if t.size<<2 > uint32(len(t.entries))*3 {
		capacity := uint32(len(t.entries)) << 1
		entries, bytes, e := bvTrailingEntries(m, capacity)
		if e != nil {
			return e
		}
		for _, entry := range t.entries {
			if entry.key.IsNull() {
				continue
			}
			hash, e := m.GetHash(entry.key)
			if e != nil {
				m.Budget().Release(bytes)
				return e
			}
			index := hash & (capacity - 1)
			for !entries[index].key.IsNull() {
				index = (index + 1) & (capacity - 1)
			}
			entries[index] = entry
		}
		m.Budget().Release(t.bytes)
		t.entries, t.bytes = entries, bytes
	}
	hash, e := m.GetHash(key)
	if e != nil {
		return e
	}
	mask := uint32(len(t.entries) - 1)
	for offset := uint32(0); offset < uint32(len(t.entries)); offset++ {
		index := (hash + offset) & mask
		entry := &t.entries[index]
		if entry.key.IsNull() {
			*entry = bvTrailingEntry{key: key, min: min, max: max, owned: true}
			t.size++
			return nil
		}
		stored, e := m.GetHash(entry.key)
		if e != nil {
			return e
		}
		if stored == hash && entry.key == key {
			entry.min, entry.max = min, max
			return nil
		}
	}
	return ast.ErrASTDomain
}
func (t *bvTrailingTable) close(m *ast.ASTManager) error {
	for t.cursor < uint32(len(t.entries)) {
		entry := &t.entries[t.cursor]
		if !entry.key.IsNull() && entry.owned {
			consumed, e := m.DecRefConsumed(entry.key)
			if consumed {
				entry.owned = false
				t.cursor++
			}
			if e != nil {
				return e
			}
			if !consumed {
				return ast.ErrASTPending
			}
		} else {
			t.cursor++
		}
	}
	m.Budget().Release(t.bytes)
	m.Budget().Release(t.header)
	t.entries = nil
	t.bytes, t.header = 0, 0
	return nil
}
