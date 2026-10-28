package ast

// Source: src/ast/ast_trail.h.

import (
	"errors"
	"testing"
)

func TestASTSortAppTrailMapLIFOOwnershipAndUndo(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	s1, e := m.MkUninterpretedSort(astTestSymbol(t, m, "trail-sort-1"), nil)
	astTestMust(t, e)
	s2, e := m.MkUninterpretedSort(astTestSymbol(t, m, "trail-sort-2"), nil)
	astTestMust(t, e)
	a1, e := m.MkConst(astTestSymbol(t, m, "trail-app-1"), b)
	astTestMust(t, e)
	a2, e := m.MkConst(astTestSymbol(t, m, "trail-app-2"), b)
	astTestMust(t, e)
	for _, h := range []ASTHandle{s1, s2, a1, a2} {
		astTestPin(t, m, h)
	}
	refs := func(h ASTHandle) uint32 { n, err := m.GetRefCount(h); astTestMust(t, err); return n }
	base := []uint32{refs(s1), refs(s2), refs(a1), refs(a2)}
	table, e := NewASTSortAppTrailMap(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, table.Close()) })
	first, e := NewASTSortAppTrail(table, s1, a1)
	astTestMust(t, e)
	second, e := NewASTSortAppTrail(table, s2, a2)
	astTestMust(t, e)
	if got, found, err := table.Find(s2); err != nil || !found || got != a2 {
		t.Fatal("find newest", got, found, err)
	}
	if refs(s1) != base[0]+1 || refs(s2) != base[1]+1 || refs(a1) != base[2]+1 || refs(a2) != base[3]+1 {
		t.Fatal("insert pins")
	}
	astTestMust(t, second.Undo())
	if _, found, err := table.Find(s2); err != nil || found || refs(s2) != base[1] || refs(a2) != base[3] {
		t.Fatal("newest undo", found, err, refs(s2), refs(a2))
	}
	astTestMust(t, second.Undo())
	astTestMust(t, first.Undo())
	if refs(s1) != base[0] || refs(a1) != base[2] {
		t.Fatal("oldest undo pins")
	}
	if e := table.Pop(); !errors.Is(e, ErrASTDomain) {
		t.Fatal("empty pop", e)
	}
}

func TestASTSortAppTrailMapDuplicateAndTypeGuards(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkConst(astTestSymbol(t, m, "trail-guard-app"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	table, e := NewASTSortAppTrailMap(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, table.Close()) })
	astTestMust(t, table.Insert(b, a))
	if e = table.Insert(b, a); !errors.Is(e, ErrASTDomain) {
		t.Fatal("duplicate mapping", e)
	}
	if e = table.Insert(a, a); !errors.Is(e, ErrASTDomain) {
		t.Fatal("non-sort domain", e)
	}
	if e = table.Insert(b, b); !errors.Is(e, ErrASTDomain) {
		t.Fatal("non-app range", e)
	}
}
