// Source: src/ast/bv_decl_plugin.h:353-368; src/ast/ast.h:530-534.
package ast

import "testing"

func TestASTBitVectorOriginalBinaryOperandMatchers(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "bv-match-x"), s)
	astTestMust(t, e)
	y, e := m.MkConst(astTestSymbol(t, m, "bv-match-y"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	astTestPin(t, m, y)
	for _, test := range []struct {
		kind  int32
		match func(ASTHandle, *ASTHandle, *ASTHandle) (bool, error)
	}{
		{4, m.BitVectorAddOperands}, {6, m.BitVectorMulOperands},
		{23, m.BitVectorSLEOperands}, {22, m.BitVectorULEOperands}, {45, m.BitVectorSHLOperands},
		{10, m.BitVectorURemOperands}, {9, m.BitVectorSRemOperands}, {7, m.BitVectorSDivOperands}, {8, m.BitVectorUDivOperands}, {11, m.BitVectorSModOperands},
		{20, m.BitVectorURemIOperands}, {19, m.BitVectorSRemIOperands}, {17, m.BitVectorSDivIOperands}, {18, m.BitVectorUDivIOperands}, {21, m.BitVectorSModIOperands},
	} {
		h, e := m.MkFamilyApp(fid, test.kind, nil, []ASTHandle{x, y})
		astTestMust(t, e)
		astTestPin(t, m, h)
		left, right := ASTHandle{}, ASTHandle{}
		xr, _ := m.internal(x)
		yr, _ := m.internal(y)
		beforeX, beforeY := m.nodeAt(xr.slot).rc, m.nodeAt(yr.slot).rc
		before := m.Stats()
		m.Budget().FailAfter(0)
		yes, e := test.match(h, &left, &right)
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if !yes || left != x || right != y || m.Stats() != before || m.nodeAt(xr.slot).rc != beforeX || m.nodeAt(yr.slot).rc != beforeY {
			t.Fatal("borrowed binary operands", test.kind, yes, left, right)
		}
		// Aliased outputs receive argument zero then argument one.
		yes, e = test.match(h, &left, &left)
		astTestMust(t, e)
		if !yes || left != y {
			t.Fatal("output publication order", test.kind, left)
		}
	}
}

func TestASTBitVectorBinaryMatcherUnmatchedOutputsUnread(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	mul, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{x, x})
	astTestMust(t, e)
	astTestPin(t, m, mul)
	real, e := m.MkFamilySort(arith, 0, nil)
	astTestMust(t, e)
	r, e := m.MkVar(0, real)
	astTestMust(t, e)
	astTestPin(t, m, r)
	// Arithmetic LT has the same numerical kind (4), in another family.
	otherFamily, e := m.MkFamilyApp(arith, 4, nil, []ASTHandle{r, r})
	astTestMust(t, e)
	astTestPin(t, m, otherFamily)
	for _, args := range [][]ASTHandle{{x}, {x, x, x}} {
		h, e := m.MkFamilyApp(fid, 4, nil, args)
		astTestMust(t, e)
		astTestPin(t, m, h)
		left, right := truth, truth
		m.Budget().FailAfter(0)
		yes, e := m.BitVectorAddOperands(h, &left, &right)
		astTestMust(t, e)
		nilOutputs, e := m.BitVectorAddOperands(h, nil, nil)
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if yes || nilOutputs || left != truth || right != truth {
			t.Fatal("arity guard read/published outputs")
		}
	}
	for _, h := range []ASTHandle{x, truth, mul, otherFamily} {
		m.Budget().FailAfter(0)
		yes, e := m.BitVectorAddOperands(h, nil, nil)
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if yes {
			t.Fatal("matched non-BV/add expression")
		}
	}
}
