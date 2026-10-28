package ast

// Source: src/ast/bv_decl_plugin.h:307-310.

import "testing"

func TestASTBitVectorExtractBoundsOriginalParameters(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(65)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	h, e := m.MkBitVectorExtract(64, 17, x)
	astTestMust(t, e)
	astTestPin(t, m, h)
	m.Budget().FailAfter(0)
	hi, e := m.BitVectorExtractHigh(h)
	astTestMust(t, e)
	lo, e := m.BitVectorExtractLow(h)
	astTestMust(t, e)
	m.Budget().FailAfter(-1)
	if hi != 64 || lo != 17 {
		t.Fatal(hi, lo)
	}
}
