// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestASTStructuralTableCollisionHeadCopyIdentityAndDuplicateGrowth(t *testing.T) {
	m := astTestManager(t)
	old := m.table
	astTestMust(t, m.initTable())
	var keys [4]astNodeRef
	defer func() {
		m.budget.Release(astChainBytes(uint32(len(m.table.cells))))
		m.table = old
		for _, key := range keys {
			if key.slot != 0 {
				m.freeNodeStorage(key)
			}
		}
	}()
	for i := range keys {
		r, e := m.allocateNode(ASTVar)
		astTestMust(t, e)
		keys[i] = r
		index := uint32(i)
		if i == 3 {
			index = 2
		} // structurally equal, distinct candidate identity
		*m.varAt(r.slot) = astVarRecord{index: index, sort: m.boolSort}
		m.nodeAt(r.slot).hash = 0 // bounded collision input, not an alternate production hash
	}
	for i := 0; i < 3; i++ {
		_, fresh, e := m.tableInsert(keys[i])
		astTestMust(t, e)
		if !fresh {
			t.Fatal("different var indices must differ")
		}
	}
	if m.table.cells[0].node != keys[2] || m.table.cells[8].node != keys[0] || m.table.cells[9].node != keys[1] || m.table.size != 3 || m.table.used != 1 || m.table.collisions != 3 {
		t.Fatal("source displaced-head cellar order/counters", m.table)
	}
	before := m.budget.Used()
	m.budget.FailAfter(0)
	_, _, e := m.tableInsert(keys[3])
	m.budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.table.slots != 8 || m.table.size != 3 || m.budget.Used() != before {
		t.Fatal("source grows before duplicate lookup", e, m.table)
	}
	found, fresh, e := m.tableInsert(keys[3])
	astTestMust(t, e)
	if fresh || found != keys[2] || m.table.slots != 16 || m.table.cells[0].node != keys[0] || m.table.cells[16].node != keys[2] || m.table.cells[17].node != keys[1] {
		t.Fatal("copy traversal reverses chain, then finds duplicate", found, fresh, m.table)
	}
	if e := m.tableEraseIdentity(keys[3]); !errors.Is(e, ErrASTHandle) {
		t.Fatal("erase uses identity, not structural equality", e)
	}
	astTestMust(t, m.tableEraseIdentity(keys[1]))
	if m.table.size != 2 || m.table.free != 17 {
		t.Fatal("cellar unlink/free list", m.table)
	}
	astTestMust(t, m.tableEraseIdentity(keys[0]))
	if m.table.cells[0].node != keys[2] || m.table.free != 16 {
		t.Fatal("head erase promotes next cellar", m.table)
	}
}
