// Source: src/ast/rewriter/bv_trailing.cpp:338-379.
package rewriter

import (
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/insdict"
)

type bvTrailingEntry struct {
	key      ast.ASTHandle
	min, max uint32
	owned    bool
}

// Each entry owns one key reference. Dictionary order makes stopped cleanup
// deterministic; native storage growth has no numerical/object arena charge.
type bvTrailingTable struct {
	entries insdict.Dict[uint64, bvTrailingEntry]
	header  uint64
}

func bvTrailingKey(key ast.ASTHandle) uint64 {
	return uint64(key.Slot())<<32 | uint64(key.Generation())
}

func newBVTrailingTable(m *ast.ASTManager) (*bvTrailingTable, error) {
	header := uint64(reflect.TypeOf(bvTrailingTable{}).Size())
	if e := m.Budget().Reserve(header); e != nil {
		return nil, e
	}
	return &bvTrailingTable{header: header}, nil
}

func (t *bvTrailingTable) find(m *ast.ASTManager, key ast.ASTHandle) (bvTrailingEntry, bool, error) {
	if _, e := m.GetHash(key); e != nil {
		return bvTrailingEntry{}, false, e
	}
	entry, found := t.entries.Get2(bvTrailingKey(key))
	return entry, found, nil
}

func (t *bvTrailingTable) insert(key ast.ASTHandle, min, max uint32) {
	token := bvTrailingKey(key)
	entry, found := t.entries.Get2(token)
	if !found {
		entry = bvTrailingEntry{key: key, owned: true}
	}
	entry.min, entry.max = min, max
	t.entries.Put(token, entry)
}

func (t *bvTrailingTable) close(m *ast.ASTManager) error {
	for token, entry := range t.entries.All() {
		if !entry.owned {
			continue
		}
		consumed, e := m.DecRefConsumed(entry.key)
		if consumed {
			entry.owned = false
			t.entries.Set(token, entry)
		}
		if e != nil {
			return e
		}
		if !consumed {
			return ast.ErrASTPending
		}
	}
	m.Budget().Release(t.header)
	t.entries.Clear()
	t.header = 0
	return nil
}
