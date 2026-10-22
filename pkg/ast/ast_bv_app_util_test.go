// Source: src/ast/bv_decl_plugin.h:397-434.
package ast

import "testing"

func TestASTBitVectorOriginalApplicationConstructors(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, sort)
	astTestMust(t, e)
	astTestPin(t, m, y)
	for _, test := range []struct {
		kind     int32
		reversed bool
		build    func(ASTHandle, ASTHandle) (ASTHandle, error)
	}{
		{22, false, m.MkBitVectorULE},
		{23, false, m.MkBitVectorSLE},
		{10, false, m.MkBitVectorURem},
		{9, false, m.MkBitVectorSRem},
		{8, false, m.MkBitVectorUDiv},
		{7, false, m.MkBitVectorSDiv},
		{11, false, m.MkBitVectorSMod},
		{17, false, m.MkBitVectorSDivInternal},
		{18, false, m.MkBitVectorUDivInternal},
		{19, false, m.MkBitVectorSRemInternal},
		{21, false, m.MkBitVectorSModInternal},
		{4, false, m.MkBitVectorAdd},
		{5, false, m.MkBitVectorSub},
		{6, false, m.MkBitVectorMul},
		{45, false, m.MkBitVectorSHL},
		{47, false, m.MkBitVectorASHR},
		{46, false, m.MkBitVectorLSHR},
		{50, false, m.MkBitVectorExtendedRotateLeft},
		{51, false, m.MkBitVectorExtendedRotateRight},
		{53, true, m.MkBitVectorSignedMulNoOverflow},
		{54, true, m.MkBitVectorSignedMulNoUnderflow},
		{52, true, m.MkBitVectorUnsignedMulNoOverflow},
	} {
		args := []ASTHandle{x, y}
		if test.reversed {
			args = []ASTHandle{y, x}
		}
		expected, e := m.MkFamilyApp(fid, test.kind, nil, args)
		astTestMust(t, e)
		astTestPin(t, m, expected)
		before := m.Stats()
		// The warm path must reuse the same cached declaration and hash-consed app.
		// Native allocation pools reuse the candidate block; g3debug uses
		// direct allocations even when hash-consing returns the existing node.
		if !ASTSourceDebug {
			m.Budget().FailAfter(0)
		}
		actual, e := test.build(x, y)
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if actual != expected || m.Stats() != before {
			t.Fatal("constructor identity/ownership", test.kind, actual, expected)
		}
		a, e := m.GetArg(actual, 0)
		astTestMust(t, e)
		b, e := m.GetArg(actual, 1)
		astTestMust(t, e)
		if a != args[0] || b != args[1] {
			t.Fatal("source argument order", test.kind)
		}
	}
	for _, test := range []struct {
		kind  int32
		build func(ASTHandle) (ASTHandle, error)
	}{{32, m.MkBitVectorNot}, {3, m.MkBitVectorNeg}, {14, m.MkBitVectorSRemZero}} {
		expected, e := m.MkFamilyApp(fid, test.kind, nil, []ASTHandle{x})
		astTestMust(t, e)
		astTestPin(t, m, expected)
		actual, e := test.build(x)
		astTestMust(t, e)
		if actual != expected {
			t.Fatal("unary constructor", test.kind)
		}
	}
}

func TestASTBitVectorOriginalVariadicConstructors(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, sort)
	astTestMust(t, e)
	astTestPin(t, m, y)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	falsity, e := m.MkFalse()
	astTestMust(t, e)
	for _, test := range []struct {
		kind  int32
		args  []ASTHandle
		width uint32
		build func([]ASTHandle) (ASTHandle, error)
	}{
		{37, []ASTHandle{x, y, x}, 24, m.MkBitVectorConcat},
		{31, []ASTHandle{x, y, x}, 8, m.MkBitVectorOr},
		{33, []ASTHandle{x, y, x}, 8, m.MkBitVectorXor},
		{56, []ASTHandle{truth, falsity, truth}, 3, m.MkBitVectorFromBools},
	} {
		expected, e := m.MkFamilyApp(fid, test.kind, nil, test.args)
		astTestMust(t, e)
		astTestPin(t, m, expected)
		actual, e := test.build(test.args)
		astTestMust(t, e)
		if actual != expected {
			t.Fatal("variadic constructor", test.kind)
		}
		width, e := m.BitVectorSize(actual)
		astTestMust(t, e)
		if width != test.width {
			t.Fatal("result width", test.kind, width)
		}
		for i, h := range test.args {
			a, e := m.GetArg(actual, uint32(i))
			astTestMust(t, e)
			if a != h {
				t.Fatal("argument order", test.kind, i)
			}
		}
	}
	expected, e := m.MkFamilyApp(fid, 37, nil, []ASTHandle{x, y})
	astTestMust(t, e)
	astTestPin(t, m, expected)
	actual, e := m.MkBitVectorConcatPair(x, y)
	astTestMust(t, e)
	if actual != expected {
		t.Fatal("binary concat overload")
	}
}
