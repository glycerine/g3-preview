package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:1531-1553.

import "testing"

func TestASTBitVectorSignExtensionSymbolicDefaultLifetime(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(65)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	n, e := NewBitVectorSignExtensionNumerals(m)
	astTestMust(t, e)
	width := uint32(7)
	matched, e := n.Read(x, &width)
	astTestMust(t, e)
	if matched || width != 7 || !n.placed[0] || n.placed[1] || n.placed[2] {
		t.Fatal("symbolic source default lifetime", matched, width, n.placed)
	}
	astTestMust(t, n.CloseExpression())
	if !n.placed[0] {
		t.Fatal("expression cleanup consumed older r")
	}
	if e = n.SourceEnd(); e != ErrASTPending {
		t.Fatal("cleanup credited source return", e)
	}
	astTestMust(t, n.Close())
	astTestMust(t, n.Close())
}
