package ast

// Source: src/ast/bv_decl_plugin.cpp:26-178,214-248,437-468.

import (
	"errors"
	"testing"
)

func TestASTBitVectorNativeCacheGrowthPreservesValidationAndOwnership(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s8, e := m.BitVectorSort(8)
	astTestMust(t, e)
	s257, e := m.BitVectorSort(257)
	astTestMust(t, e)
	index, e := m.NewParameterInt(3)
	astTestMust(t, e)
	first, e := m.MkFamilyFuncDecl(fid, 55, []ASTParameter{index}, []ASTHandle{s8}, ASTHandle{})
	astTestMust(t, e)
	astTestMust(t, m.DelParameter(&index))
	negative, e := m.NewParameterInt(-1)
	astTestMust(t, e)
	width, e := m.NewParameterInt(4096)
	astTestMust(t, e)
	p := m.bitVectorPlugin
	refs, e := m.GetRefCount(first)
	astTestMust(t, e)
	used := m.Budget().Used()
	m.Budget().FailAfter(0)
	_, e = m.MkFamilyFuncDecl(fid, 55, []ASTParameter{negative}, []ASTHandle{s257}, ASTHandle{})
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal("negative index must follow native nested growth", e)
	}
	_, e = m.MkFamilyFuncDecl(fid, 57, []ASTParameter{width}, nil, ASTHandle{})
	if e == nil || e.Error() != "expecting one argument to int2bv" {
		t.Fatal("arity validation follows native conversion growth", e)
	}
	if len(p.bit2bool) != 258 || len(p.bit2bool[257]) != 257 || len(p.fixed[57]) != 4097 {
		t.Fatal("native resize prefixes")
	}
	if p.bit2bool[8][3] != (astNodeRef{first.slot, first.generation}) {
		t.Fatal("nested growth lost old owned row")
	}
	if m.Budget().Used() != used {
		t.Fatal("native cache backing charged an arena reservation")
	}
	h, e := m.MkFamilySort(fid, 0, []ASTParameter{width})
	astTestMust(t, e)
	if len(p.sorts) != 4097 || p.sorts[4096].slot == 0 || p.sortAcquired[4096] != 1 {
		t.Fatal("native sort growth and actual cache INC")
	}
	if m.Budget().Used() != used {
		t.Fatal("native cache growth charged a reservation")
	}
	m.Budget().FailAfter(-1)
	got, e := m.GetRefCount(first)
	astTestMust(t, e)
	if got != refs {
		t.Fatal("growth changed earlier cache reference count", got, refs)
	}
	got, e = m.GetRefCount(h)
	astTestMust(t, e)
	if got != 1 {
		t.Fatal("actual successful sort cache INC", got)
	}
	astTestMust(t, m.DelParameter(&negative))
	astTestMust(t, m.DelParameter(&width))
	astTestMust(t, m.SourceFinalize())
	if m.bitVectorPlugin != nil {
		t.Fatal("native owning cache cleanup incomplete")
	}
}
