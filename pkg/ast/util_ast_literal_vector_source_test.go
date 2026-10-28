package ast

// Source: src/util/sorting_network.h:120-1046; src/test/sorting_network.cpp:150-181.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestASTLiteralVectorNativeBorrowGrowthAndNonowningReset(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkFreshConst("y", sort)
	astTestMust(t, e)
	astTestPin(t, m, y)
	v, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	refs, e := m.GetRefCount(x)
	astTestMust(t, e)
	astTestMust(t, v.Push(x))
	astTestMust(t, v.Push(y))
	span, e := v.Span()
	astTestMust(t, e)
	astTestMust(t, v.Set(0, y))
	got, e := span.At(0)
	astTestMust(t, e)
	if got != y {
		t.Fatal("borrow lost same-backing mutation", got)
	}
	used := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	for i := 0; i < 100; i++ {
		astTestMust(t, v.Push(x))
	}
	m.BudgetField.FailAfter(-1)
	if m.BudgetField.Used() != used {
		t.Fatal("native backing charged to arena")
	}
	got, e = span.At(1)
	astTestMust(t, e)
	if got != y {
		t.Fatal("native growth invalidated captured slice", got)
	}
	astTestMust(t, v.Reset())
	n, e := v.Size()
	astTestMust(t, e)
	now, e := m.GetRefCount(x)
	astTestMust(t, e)
	if n != 0 || now != refs {
		t.Fatal("borrowed handles changed AST references", n, refs, now)
	}
	astTestMust(t, v.Close())
	if _, e = span.At(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner left capture usable", e)
	}
}
func astLiteralNamedGraph(t *testing.T, b *arena.Budget, r *umemory.Runtime, external uint64, owners []*ASTLiteralVector, anchors ...umemory.Block) {
	t.Helper()
	cold := external
	roots := map[umemory.Block]bool{}
	for _, h := range anchors {
		if h == (umemory.Block{}) {
			continue
		}
		if roots[h] {
			t.Fatal("duplicate actual Runtime root", h)
		}
		roots[h] = true
		_, e := r.Size(h)
		astTestMust(t, e)
	}
	for _, v := range owners {
		cold += v.header
	}
	stats := r.Stats()
	if stats.LiveBlocks != uint32(len(roots)) || b.Used() != stats.TotalBytes+cold {
		t.Fatal("native literal owner and actual Runtime graph", stats, len(roots), b.Used(), cold)
	}
}
