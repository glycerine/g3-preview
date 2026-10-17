package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPAFPComponentsDispatch(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	signSort, err := m.BitVectorSort(1)
	astTestMust(t, err)
	expSort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	sigSort, err := m.BitVectorSort(23)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpFP, nil,
		[]ast.ASTHandle{signSort, expSort, sigSort}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	sign, err := m.BitVectorNumeralUint64(1, 1)
	astTestMust(t, err)
	astTestPin(t, m, sign)
	exponent, err := m.BitVectorNumeralUint64(127, 8)
	astTestMust(t, err)
	astTestPin(t, m, exponent)
	significand, err := m.BitVectorNumeralUint64(0, 23)
	astTestMust(t, err)
	astTestPin(t, m, significand)
	out := astTestPin(t, m, sign)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{sign, exponent, significand}), out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	bits, found, _, err := m.RewriteFPAToIEEEBVNumeral(result, false)
	astTestMust(t, err)
	want, err := m.BitVectorNumeralUint64(0xbf800000, 32)
	astTestMust(t, err)
	if status != RewriteDone || !found || bits != want {
		t.Fatalf("FP components dispatch: status=%v found=%v bits=%v want=%v", status, found, bits, want)
	}
	symbolic, err := m.MkVar(0, sigSort)
	astTestMust(t, err)
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{sign, exponent, symbolic}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatalf("symbolic significand should refuse: %v", status)
	}
}
