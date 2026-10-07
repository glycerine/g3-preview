// Additional finite specialized identity-table tests; no registered sparse-mark test.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTSparseMarkActualIdentityAndNoRetains(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 9)
	mark, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer mark.Close()
	if mark.table.capacity != 8 {
		t.Fatal("source eager default table8")
	}
	for _, n := range h[:7] {
		astTestMust(t, mark.Mark(n, true))
		rc, e := m.GetRefCount(n)
		astTestMust(t, e)
		if rc != 0 {
			t.Fatal("sparse marker must not pin AST")
		}
	}
	// source >75%, not >=75%, and growth precedes duplicate lookup.
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	err := mark.Mark(h[0], true)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || mark.table.capacity != 8 || m.BudgetField.Used() != before {
		t.Fatal("grow-before-duplicate", err)
	}
	astTestMust(t, mark.Mark(h[0], true))
	if mark.table.capacity != 16 || mark.table.size != 7 {
		t.Fatal("source duplicate after growth")
	}
	astTestMust(t, mark.Mark(h[0], false))
	yes, e := mark.IsMarked(h[0])
	astTestMust(t, e)
	if yes {
		t.Fatal("identity erase")
	}
	// A deleted AST ID can be recycled, but the new object is not the same
	// pointer identity. Remove old key before destruction to keep table hashes
	// live, as required by source obj_hash_entry's borrowed lifetime.
	id, e := m.GetASTID(h[0])
	astTestMust(t, e)
	astTestMust(t, m.IncRef(h[0]))
	astTestMust(t, m.DecRef(h[0]))
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	fresh, e := m.MkVar(65000, b)
	astTestMust(t, e)
	got, e := m.GetASTID(fresh)
	astTestMust(t, e)
	if id != got {
		t.Fatal("recycle fixture")
	}
	yes, e = mark.IsMarked(fresh)
	astTestMust(t, e)
	if yes {
		t.Fatal("sparse identity cannot be ID bitset")
	}
	astTestMust(t, mark.Reset())
	if mark.table.size != 0 || mark.table.deleted != 0 {
		t.Fatal("reset")
	}
}

func TestASTSparseMarkResetAndFinalizeFreeBeforeReserve(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 1)[0]
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	astTestMust(t, s.table.rehash(m, 128))
	astTestMust(t, s.Mark(h, true))
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	e = s.Reset()
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.table.capacity != 64 || s.table.size != 1 || s.table.bytes != 0 || s.table.entries != nil || s.table.pending != 1 || m.BudgetField.Used() != before-1024 {
		t.Fatal("reset clear/free/publish-half prefix", e, s.table, m.BudgetField.Used(), before)
	}
	if _, e = s.IsMarked(h); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("nil backing must reject undefined entry access", e)
	}
	astTestMust(t, s.Reset())
	if s.table.size != 0 || s.table.bytes != 512 || s.table.pending != 0 {
		t.Fatal("reset allocation continuation")
	}
	astTestMust(t, s.table.rehash(m, 128))
	before = m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	e = s.table.finalize(m)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.table.capacity != 128 || s.table.bytes != 0 || s.table.pending != 2 || m.BudgetField.Used() != before-1024 {
		t.Fatal("finalize publishes64 only AFTER replacement", e, s.table)
	}
	astTestMust(t, s.table.finalize(m))
	if s.table.capacity != 64 || s.table.bytes != 512 || s.table.pending != 0 {
		t.Fatal("finalize retry")
	}
}

func TestASTSparseMarkBoundedIdentityOracle(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 100)
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	var expected [100]bool
	seed := uint32(17)
	for step := 0; step < 1500; step++ {
		seed = seed*1664525 + 1013904223
		idx := seed % 100
		flag := seed&256 != 0
		astTestMust(t, s.Mark(h[idx], flag))
		expected[idx] = flag
		if step%29 == 0 {
			count := uint32(0)
			for i, n := range h {
				yes, e := s.IsMarked(n)
				astTestMust(t, e)
				if yes != expected[i] {
					t.Fatal(step, i, yes, expected[i])
				}
				if expected[i] {
					count++
				}
			}
			if s.table.size != count {
				t.Fatal("count", step)
			}
		}
	}
}

func TestASTSparseMarkEraseFreeSuccessorSkipsCleanup(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 1)[0]
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	astTestMust(t, s.table.rehash(m, 128))
	r, e := m.internal(h)
	astTestMust(t, e)
	start := m.nodeAt(r.slot).hash & 127
	s.table.entries[start] = r
	s.table.size = 1
	// Source-valid tombstones, placed away from the following FREE slot.
	for off := uint32(2); off < 67; off++ {
		s.table.entries[(start+off)&127] = astNodeRef{slot: 1}
	}
	s.table.deleted = 65
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	e = s.Mark(h, false)
	m.BudgetField.FailAfter(-1)
	if e != nil || s.table.deleted != 65 || s.table.size != 0 || m.BudgetField.Used() != before {
		t.Fatal("source cleanup branch only after writing DELETED", e, s.table.deleted)
	}
}

func TestASTSparseMarkIteratorEpochExhaustionPreflight(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 9)
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	astTestMust(t, s.Mark(h[0], true))
	s.table.epoch = ^uint64(0)
	before := m.BudgetField.Used()
	astTestMust(t, s.Mark(h[1], true))
	astTestMust(t, s.Mark(h[0], false))
	astTestMust(t, s.Reset())
	if s.table.size != 0 || s.table.epoch != ^uint64(0) || m.BudgetField.Used() != before {
		t.Fatal("in-place operations do not invalidate backing")
	}
	for _, n := range h[:7] {
		astTestMust(t, s.Mark(n, true))
	}
	if e = s.Mark(h[0], true); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("grow-before-duplicate backing exhaustion", e)
	}
	if s.table.size != 7 || s.table.capacity != 8 || m.BudgetField.Used() != before {
		t.Fatal("refused relocation changes backing")
	}
	s.table.epoch = 0
	astTestMust(t, s.table.rehash(m, 128))
	astTestMust(t, s.Reset())
	astTestMust(t, s.Mark(h[0], true))
	s.table.epoch = ^uint64(0)
	before = m.BudgetField.Used()
	if e = s.Reset(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("shrinking reset exhaustion", e)
	}
	if s.table.size != 1 || s.table.capacity != 64 || s.table.entries == nil || m.BudgetField.Used() != before {
		t.Fatal("refused shrink must precede clear/free")
	}
}

func TestASTSparseMarkPendingResetCanFinishAtFinalEpoch(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 1)[0]
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	astTestMust(t, s.table.rehash(m, 128))
	astTestMust(t, s.Mark(h, true))
	s.table.epoch = ^uint64(0) - 1
	m.BudgetField.FailAfter(0)
	e = s.Reset()
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.table.epoch != ^uint64(0) || s.table.pending != 1 {
		t.Fatal("consume one epoch before pending reset", e)
	}
	astTestMust(t, s.Reset()) // old views were already invalidated before free
	if s.table.pending != 0 || s.table.size != 0 || s.table.epoch != ^uint64(0) {
		t.Fatal("retry must not require another invalidation")
	}
}

func TestASTSparseMarkLastTombstoneAndCurrentHashReads(t *testing.T) {
	m := astTestManager(t)
	h := markTestNodes(t, m, 100)
	var bins [8][]ASTHandle
	for _, n := range h {
		r, e := m.internal(n)
		astTestMust(t, e)
		b := m.nodeAt(r.slot).hash & 7
		bins[b] = append(bins[b], n)
	}
	var chosen []ASTHandle
	var start uint32
	for i, b := range bins {
		if len(b) >= 4 {
			chosen = b[:4]
			start = uint32(i)
			break
		}
	}
	if len(chosen) != 4 {
		t.Fatal("finite collision fixture")
	}
	s, e := NewASTExprSparseMark(m)
	astTestMust(t, e)
	defer s.Close()
	for _, n := range chosen[:3] {
		astTestMust(t, s.Mark(n, true))
	}
	astTestMust(t, s.Mark(chosen[0], false))
	astTestMust(t, s.Mark(chosen[1], false))
	astTestMust(t, s.Mark(chosen[3], true))
	r, e := m.internal(chosen[3])
	astTestMust(t, e)
	if s.table.entries[start] != (astNodeRef{slot: 1}) || s.table.entries[(start+1)&7] != r || s.table.deleted != 1 {
		t.Fatal("last encountered tombstone, not first")
	}
	// USED.get_hash() must read current node.hash(), not a cached hash. This
	// private mutation isolates the entry contract and is restored before the
	// actual AST interner can inspect or delete the node.
	original := m.nodeAt(r.slot).hash
	m.nodeAt(r.slot).hash = original + 8
	yes, e := s.IsMarked(chosen[3])
	m.nodeAt(r.slot).hash = original
	astTestMust(t, e)
	if !yes {
		t.Fatal("obj entry cached hash would miss changed current hash")
	}
}
